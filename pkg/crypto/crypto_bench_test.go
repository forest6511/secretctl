package crypto_test

import (
	"crypto/rand"
	"testing"

	"github.com/forest6511/secretctl/pkg/crypto"
)

// BenchmarkDeriveKey measures Argon2id key derivation performance.
// Expected: ~35ms on modern hardware with 64MB memory cost (OWASP recommended parameters).
func BenchmarkDeriveKey(b *testing.B) {
	password := []byte("testpassword123!")
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		crypto.DeriveKey(password, salt)
	}
}

// BenchmarkEncrypt measures AES-256-GCM encryption performance with 1KB payload.
func BenchmarkEncrypt(b *testing.B) {
	key := make([]byte, crypto.KeyLength)
	if _, err := rand.Read(key); err != nil {
		b.Fatal(err)
	}
	data := make([]byte, 1024) // 1KB
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := crypto.Encrypt(key, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecrypt measures AES-256-GCM decryption performance with 1KB payload.
func BenchmarkDecrypt(b *testing.B) {
	key := make([]byte, crypto.KeyLength)
	if _, err := rand.Read(key); err != nil {
		b.Fatal(err)
	}
	data := make([]byte, 1024) // 1KB
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}
	ciphertext, nonce, err := crypto.Encrypt(key, data)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := crypto.Decrypt(key, ciphertext, nonce)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSecureWipe measures secure memory wiping performance.
func BenchmarkSecureWipe(b *testing.B) {
	data := make([]byte, 1024) // 1KB

	b.ReportAllocs()
	b.SetBytes(1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		crypto.SecureWipe(data)
	}
}

// Benchmark encryption with various payload sizes to measure throughput.

func BenchmarkEncrypt1KB(b *testing.B) {
	benchmarkEncrypt(b, 1024)
}

func BenchmarkEncrypt10KB(b *testing.B) {
	benchmarkEncrypt(b, 10*1024)
}

func BenchmarkEncrypt100KB(b *testing.B) {
	benchmarkEncrypt(b, 100*1024)
}

func BenchmarkEncrypt1MB(b *testing.B) {
	benchmarkEncrypt(b, 1024*1024)
}

func benchmarkEncrypt(b *testing.B, size int) {
	b.Helper()
	key := make([]byte, crypto.KeyLength)
	if _, err := rand.Read(key); err != nil {
		b.Fatal(err)
	}
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := crypto.Encrypt(key, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark decryption with various payload sizes to measure throughput.

func BenchmarkDecrypt1KB(b *testing.B) {
	benchmarkDecrypt(b, 1024)
}

func BenchmarkDecrypt10KB(b *testing.B) {
	benchmarkDecrypt(b, 10*1024)
}

func BenchmarkDecrypt100KB(b *testing.B) {
	benchmarkDecrypt(b, 100*1024)
}

func BenchmarkDecrypt1MB(b *testing.B) {
	benchmarkDecrypt(b, 1024*1024)
}

func benchmarkDecrypt(b *testing.B, size int) {
	b.Helper()
	key := make([]byte, crypto.KeyLength)
	if _, err := rand.Read(key); err != nil {
		b.Fatal(err)
	}
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}
	ciphertext, nonce, err := crypto.Encrypt(key, data)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := crypto.Decrypt(key, ciphertext, nonce)
		if err != nil {
			b.Fatal(err)
		}
	}
}
