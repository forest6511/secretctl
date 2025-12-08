package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// TestDeriveKey tests the Argon2id key derivation function
func TestDeriveKey(t *testing.T) {
	password := []byte("test-password-123")
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		t.Fatalf("failed to generate salt: %v", err)
	}

	// Test key derivation produces correct length
	key := DeriveKey(password, salt)
	if len(key) != KeyLength {
		t.Errorf("DeriveKey() returned key of length %d, want %d", len(key), KeyLength)
	}

	// Test same password + salt produces same key (deterministic)
	key2 := DeriveKey(password, salt)
	if !bytes.Equal(key, key2) {
		t.Error("DeriveKey() with same inputs should produce identical keys")
	}

	// Test different password produces different key
	differentKey := DeriveKey([]byte("different-password"), salt)
	if bytes.Equal(key, differentKey) {
		t.Error("DeriveKey() with different password should produce different key")
	}

	// Test different salt produces different key
	differentSalt := make([]byte, 16)
	if _, err := rand.Read(differentSalt); err != nil {
		t.Fatalf("failed to generate salt: %v", err)
	}
	differentKey = DeriveKey(password, differentSalt)
	if bytes.Equal(key, differentKey) {
		t.Error("DeriveKey() with different salt should produce different key")
	}
}

// TestDeriveKeyParameters verifies Argon2id parameters match OWASP recommendations
func TestDeriveKeyParameters(t *testing.T) {
	// Verify constants match expected OWASP values
	if Argon2Memory != 64*1024 {
		t.Errorf("Argon2Memory = %d, want %d (64MB)", Argon2Memory, 64*1024)
	}
	if Argon2Time != 3 {
		t.Errorf("Argon2Time = %d, want 3", Argon2Time)
	}
	if Argon2Threads != 4 {
		t.Errorf("Argon2Threads = %d, want 4", Argon2Threads)
	}
	if KeyLength != 32 {
		t.Errorf("KeyLength = %d, want 32 (256-bit)", KeyLength)
	}
}

// TestEncrypt tests the AES-256-GCM encryption function
func TestEncrypt(t *testing.T) {
	key := make([]byte, KeyLength)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	plaintext := []byte("secret data to encrypt")

	// Test successful encryption
	ciphertext, nonce, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Verify nonce length
	if len(nonce) != NonceLength {
		t.Errorf("Encrypt() nonce length = %d, want %d", len(nonce), NonceLength)
	}

	// Verify ciphertext is different from plaintext
	if bytes.Equal(ciphertext, plaintext) {
		t.Error("Encrypt() ciphertext should not equal plaintext")
	}

	// Verify ciphertext includes authentication tag (16 bytes overhead)
	expectedMinLen := len(plaintext) + 16 // GCM tag is 16 bytes
	if len(ciphertext) < expectedMinLen {
		t.Errorf("Encrypt() ciphertext length = %d, want >= %d", len(ciphertext), expectedMinLen)
	}
}

// TestEncryptInvalidKeyLength tests that Encrypt rejects invalid key lengths
func TestEncryptInvalidKeyLength(t *testing.T) {
	tests := []struct {
		name    string
		keyLen  int
		wantErr error
	}{
		{"too short (16 bytes)", 16, ErrInvalidKeyLength},
		{"too short (24 bytes)", 24, ErrInvalidKeyLength},
		{"too long (48 bytes)", 48, ErrInvalidKeyLength},
		{"empty key", 0, ErrInvalidKeyLength},
	}

	plaintext := []byte("test data")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLen)
			_, _, err := Encrypt(key, plaintext)
			if err != tt.wantErr {
				t.Errorf("Encrypt() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestEncryptEmptyPlaintext tests encryption of empty data
func TestEncryptEmptyPlaintext(t *testing.T) {
	key := make([]byte, KeyLength)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	ciphertext, nonce, err := Encrypt(key, []byte{})
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Empty plaintext should still produce ciphertext (just the tag)
	if len(ciphertext) != 16 { // GCM tag only
		t.Errorf("Encrypt() empty plaintext ciphertext length = %d, want 16", len(ciphertext))
	}
	if len(nonce) != NonceLength {
		t.Errorf("Encrypt() nonce length = %d, want %d", len(nonce), NonceLength)
	}
}

// TestDecrypt tests the AES-256-GCM decryption function
func TestDecrypt(t *testing.T) {
	key := make([]byte, KeyLength)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	plaintext := []byte("secret data to encrypt and decrypt")

	// Encrypt first
	ciphertext, nonce, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Test successful decryption
	decrypted, err := Decrypt(key, ciphertext, nonce)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	// Verify decrypted data matches original
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypt() = %q, want %q", decrypted, plaintext)
	}
}

// TestDecryptInvalidKey tests that decryption fails with wrong key
func TestDecryptInvalidKey(t *testing.T) {
	key := make([]byte, KeyLength)
	wrongKey := make([]byte, KeyLength)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	if _, err := rand.Read(wrongKey); err != nil {
		t.Fatalf("failed to generate wrong key: %v", err)
	}

	plaintext := []byte("secret data")

	// Encrypt with correct key
	ciphertext, nonce, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Attempt decryption with wrong key
	_, err = Decrypt(wrongKey, ciphertext, nonce)
	if err != ErrDecryptionFailed {
		t.Errorf("Decrypt() with wrong key error = %v, want %v", err, ErrDecryptionFailed)
	}
}

// TestDecryptInvalidNonce tests that decryption fails with wrong nonce
func TestDecryptInvalidNonce(t *testing.T) {
	key := make([]byte, KeyLength)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	plaintext := []byte("secret data")

	// Encrypt
	ciphertext, _, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Create wrong nonce
	wrongNonce := make([]byte, NonceLength)
	if _, err := rand.Read(wrongNonce); err != nil {
		t.Fatalf("failed to generate wrong nonce: %v", err)
	}

	// Attempt decryption with wrong nonce
	_, err = Decrypt(key, ciphertext, wrongNonce)
	if err != ErrDecryptionFailed {
		t.Errorf("Decrypt() with wrong nonce error = %v, want %v", err, ErrDecryptionFailed)
	}
}

// TestDecryptInvalidKeyLength tests that Decrypt rejects invalid key lengths
func TestDecryptInvalidKeyLength(t *testing.T) {
	tests := []struct {
		name   string
		keyLen int
	}{
		{"too short (16 bytes)", 16},
		{"too short (24 bytes)", 24},
		{"too long (48 bytes)", 48},
		{"empty key", 0},
	}

	ciphertext := make([]byte, 32)
	nonce := make([]byte, NonceLength)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLen)
			_, err := Decrypt(key, ciphertext, nonce)
			if err != ErrInvalidKeyLength {
				t.Errorf("Decrypt() error = %v, want %v", err, ErrInvalidKeyLength)
			}
		})
	}
}

// TestDecryptInvalidNonceLength tests that Decrypt rejects invalid nonce lengths
func TestDecryptInvalidNonceLength(t *testing.T) {
	key := make([]byte, KeyLength)
	ciphertext := make([]byte, 32)

	tests := []struct {
		name     string
		nonceLen int
	}{
		{"too short (8 bytes)", 8},
		{"too long (16 bytes)", 16},
		{"empty nonce", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nonce := make([]byte, tt.nonceLen)
			_, err := Decrypt(key, ciphertext, nonce)
			if err != ErrInvalidNonceLength {
				t.Errorf("Decrypt() error = %v, want %v", err, ErrInvalidNonceLength)
			}
		})
	}
}

// TestDecryptCiphertextTooShort tests that Decrypt handles short ciphertext
func TestDecryptCiphertextTooShort(t *testing.T) {
	key := make([]byte, KeyLength)
	nonce := make([]byte, NonceLength)

	// Ciphertext shorter than GCM tag (16 bytes)
	shortCiphertext := make([]byte, 10)

	_, err := Decrypt(key, shortCiphertext, nonce)
	if err != ErrCiphertextTooShort {
		t.Errorf("Decrypt() error = %v, want %v", err, ErrCiphertextTooShort)
	}
}

// TestDecryptTamperedCiphertext tests that tampering is detected
func TestDecryptTamperedCiphertext(t *testing.T) {
	key := make([]byte, KeyLength)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	plaintext := []byte("secret data that should be protected")

	// Encrypt
	ciphertext, nonce, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Tamper with ciphertext (flip a bit)
	tamperedCiphertext := make([]byte, len(ciphertext))
	copy(tamperedCiphertext, ciphertext)
	tamperedCiphertext[0] ^= 0x01

	// Attempt decryption of tampered data
	_, err = Decrypt(key, tamperedCiphertext, nonce)
	if err != ErrDecryptionFailed {
		t.Errorf("Decrypt() with tampered ciphertext error = %v, want %v", err, ErrDecryptionFailed)
	}
}

// TestEncryptDecryptRoundTrip tests multiple encrypt/decrypt cycles
func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, KeyLength)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	testCases := []struct {
		name      string
		plaintext []byte
	}{
		{"empty", []byte{}},
		{"small", []byte("x")},
		{"medium", []byte("This is a medium-length test string for encryption.")},
		{"large", make([]byte, 10000)}, // 10KB
		{"binary", []byte{0x00, 0xFF, 0x01, 0xFE, 0x02, 0xFD}},
	}

	// Fill large test case with random data
	if _, err := rand.Read(testCases[3].plaintext); err != nil {
		t.Fatalf("failed to generate random data: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ciphertext, nonce, err := Encrypt(key, tc.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			decrypted, err := Decrypt(key, ciphertext, nonce)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if !bytes.Equal(decrypted, tc.plaintext) {
				t.Errorf("Round trip failed: got length %d, want length %d", len(decrypted), len(tc.plaintext))
			}
		})
	}
}

// TestEncryptProducesUniqueNonce tests that each encryption produces a unique nonce
func TestEncryptProducesUniqueNonce(t *testing.T) {
	key := make([]byte, KeyLength)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	plaintext := []byte("test data")
	nonces := make(map[string]bool)

	// Generate 100 nonces and verify they're all unique
	for i := 0; i < 100; i++ {
		_, nonce, err := Encrypt(key, plaintext)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}

		nonceStr := string(nonce)
		if nonces[nonceStr] {
			t.Errorf("Encrypt() produced duplicate nonce on iteration %d", i)
		}
		nonces[nonceStr] = true
	}
}

// TestSecureWipe tests that SecureWipe zeros out memory
func TestSecureWipe(t *testing.T) {
	// Create a slice with non-zero data
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	original := make([]byte, len(data))
	copy(original, data)

	// Wipe the data
	SecureWipe(data)

	// Verify all bytes are zero
	for i, b := range data {
		if b != 0 {
			t.Errorf("SecureWipe() byte[%d] = %d, want 0", i, b)
		}
	}

	// Verify original data was actually non-zero
	hasNonZero := false
	for _, b := range original {
		if b != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("Test setup error: original data should have non-zero bytes")
	}
}

// TestSecureWipeEmptySlice tests SecureWipe with empty slice
func TestSecureWipeEmptySlice(t *testing.T) {
	t.Log("Testing SecureWipe with empty and nil slices")

	// Should not panic on empty slice
	data := []byte{}
	SecureWipe(data)

	// Should not panic on nil slice
	var nilData []byte
	SecureWipe(nilData)
}

// TestSecureWipeLargeSlice tests SecureWipe with large data
func TestSecureWipeLargeSlice(t *testing.T) {
	data := make([]byte, 1024*1024) // 1MB
	if _, err := rand.Read(data); err != nil {
		t.Fatalf("failed to generate random data: %v", err)
	}

	SecureWipe(data)

	// Check that all bytes are zero
	for i, b := range data {
		if b != 0 {
			t.Errorf("SecureWipe() byte[%d] = %d, want 0", i, b)
			break // Don't flood output
		}
	}
}

// TestConstants verifies crypto constants are correct
func TestConstants(t *testing.T) {
	if NonceLength != 12 {
		t.Errorf("NonceLength = %d, want 12 (96-bit GCM standard)", NonceLength)
	}
	if KeyLength != 32 {
		t.Errorf("KeyLength = %d, want 32 (256-bit AES)", KeyLength)
	}
}

// BenchmarkDeriveKey benchmarks the key derivation function
func BenchmarkDeriveKey(b *testing.B) {
	password := []byte("benchmark-password-123")
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		b.Fatalf("failed to generate salt: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DeriveKey(password, salt)
	}
}

// BenchmarkEncrypt benchmarks the encryption function
func BenchmarkEncrypt(b *testing.B) {
	key := make([]byte, KeyLength)
	if _, err := rand.Read(key); err != nil {
		b.Fatalf("failed to generate key: %v", err)
	}
	plaintext := make([]byte, 1024) // 1KB data
	if _, err := rand.Read(plaintext); err != nil {
		b.Fatalf("failed to generate plaintext: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = Encrypt(key, plaintext)
	}
}

// BenchmarkDecrypt benchmarks the decryption function
func BenchmarkDecrypt(b *testing.B) {
	key := make([]byte, KeyLength)
	if _, err := rand.Read(key); err != nil {
		b.Fatalf("failed to generate key: %v", err)
	}
	plaintext := make([]byte, 1024) // 1KB data
	if _, err := rand.Read(plaintext); err != nil {
		b.Fatalf("failed to generate plaintext: %v", err)
	}

	ciphertext, nonce, err := Encrypt(key, plaintext)
	if err != nil {
		b.Fatalf("Encrypt() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Decrypt(key, ciphertext, nonce)
	}
}
