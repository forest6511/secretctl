// Package crypto provides cryptographic primitives for secretctl.
//
// This package implements AES-256-GCM authenticated encryption and Argon2id
// key derivation following OWASP recommendations.
//
// # Security Features
//
//   - AES-256-GCM authenticated encryption
//   - Argon2id key derivation (64MB memory, 3 iterations, 4 threads)
//   - Cryptographically secure random nonce generation
//   - Secure memory wiping for sensitive data
//
// # Example Usage
//
//	// Derive a key from password
//	salt := make([]byte, 16)
//	rand.Read(salt)
//	key := crypto.DeriveKey([]byte("password"), salt)
//
//	// Encrypt data
//	ciphertext, nonce, err := crypto.Encrypt(key, plaintext)
//
//	// Decrypt data
//	plaintext, err := crypto.Decrypt(key, ciphertext, nonce)
//
//	// Securely wipe sensitive data
//	crypto.SecureWipe(key)
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"runtime"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters following OWASP recommendations.
const (
	// Argon2Memory is the memory cost in KiB (64MB).
	Argon2Memory = 64 * 1024

	// Argon2Time is the number of iterations.
	Argon2Time = 3

	// Argon2Threads is the degree of parallelism.
	Argon2Threads = 4

	// KeyLength is the length of encryption keys in bytes (256 bits).
	KeyLength = 32

	// NonceLength is the length of GCM nonces in bytes (96 bits).
	NonceLength = 12
)

// Sentinel errors returned by crypto functions.
var (
	// ErrInvalidKeyLength indicates the key is not 32 bytes.
	ErrInvalidKeyLength = errors.New("crypto: invalid key length, must be 32 bytes")

	// ErrInvalidNonceLength indicates the nonce is not 12 bytes.
	ErrInvalidNonceLength = errors.New("crypto: invalid nonce length, must be 12 bytes")

	// ErrDecryptionFailed indicates decryption or authentication tag verification failed.
	ErrDecryptionFailed = errors.New("crypto: decryption failed, authentication tag verification failed")

	// ErrCiphertextTooShort indicates the ciphertext is shorter than the GCM tag.
	ErrCiphertextTooShort = errors.New("crypto: ciphertext too short")
)

// DeriveKey derives a 256-bit encryption key from a password using Argon2id.
//
// The function uses OWASP-recommended parameters:
//   - Memory: 64 MB
//   - Iterations: 3
//   - Parallelism: 4 threads
//
// The salt should be at least 16 bytes of cryptographically secure random data.
// Returns a 32-byte key suitable for AES-256 encryption.
func DeriveKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, Argon2Time, Argon2Memory, Argon2Threads, KeyLength)
}

// Encrypt encrypts plaintext using AES-256-GCM authenticated encryption.
//
// The function generates a cryptographically secure random 12-byte nonce
// using crypto/rand. The authentication tag is appended to the ciphertext.
//
// Parameters:
//   - key: 32-byte encryption key (use DeriveKey to generate)
//   - plaintext: data to encrypt (can be any length)
//
// Returns:
//   - ciphertext: encrypted data with authentication tag
//   - nonce: 12-byte nonce (must be stored with ciphertext for decryption)
//   - err: ErrInvalidKeyLength if key is not 32 bytes
func Encrypt(key, plaintext []byte) (ciphertext []byte, nonce []byte, err error) {
	if len(key) != KeyLength {
		return nil, nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("crypto: failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("crypto: failed to create GCM: %w", err)
	}

	// Generate cryptographically secure random nonce
	nonce = make([]byte, NonceLength)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("crypto: failed to generate nonce: %w", err)
	}

	// Encrypt with GCM (authentication tag is appended to ciphertext)
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)

	return ciphertext, nonce, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM authenticated encryption.
//
// The function verifies the authentication tag before returning the plaintext.
// If the tag verification fails (indicating tampering or corruption),
// ErrDecryptionFailed is returned.
//
// Parameters:
//   - key: 32-byte encryption key (same key used for encryption)
//   - ciphertext: encrypted data with authentication tag
//   - nonce: 12-byte nonce used during encryption
//
// Returns:
//   - plaintext: decrypted data
//   - err: ErrInvalidKeyLength, ErrInvalidNonceLength, ErrCiphertextTooShort,
//     or ErrDecryptionFailed
func Decrypt(key, ciphertext, nonce []byte) (plaintext []byte, err error) {
	if len(key) != KeyLength {
		return nil, ErrInvalidKeyLength
	}

	if len(nonce) != NonceLength {
		return nil, ErrInvalidNonceLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: failed to create GCM: %w", err)
	}

	// Verify ciphertext has minimum length (GCM tag is 16 bytes)
	if len(ciphertext) < gcm.Overhead() {
		return nil, ErrCiphertextTooShort
	}

	// Decrypt with GCM (includes authentication tag verification)
	plaintext, err = gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// SecureWipe overwrites a byte slice with zeros in a way that prevents
// compiler optimization from removing the operation.
// This is critical for securely destroying sensitive data like DEK.
func SecureWipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
	// runtime.KeepAlive ensures the write operations are not optimized away
	// by the compiler since b is still "in use" after the loop.
	runtime.KeepAlive(b)
}
