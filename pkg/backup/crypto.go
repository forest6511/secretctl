package backup

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/hkdf"

	"github.com/forest6511/secretctl/pkg/crypto"
)

const (
	// SaltLength is the length of the backup salt in bytes.
	SaltLength = 32

	// HMACLength is the length of the HMAC-SHA256 in bytes.
	HMACLength = 32

	// KeyLength is the length of encryption keys in bytes (256 bits).
	KeyLength = 32
)

// HKDF info strings for key derivation.
const (
	hkdfInfoEncryption = "secretctl-backup-encryption"
	hkdfInfoMAC        = "secretctl-backup-mac"
)

// GenerateSalt generates a cryptographically secure random salt.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// DeriveBackupKeys derives encryption and MAC keys from a password and salt.
func DeriveBackupKeys(password, salt []byte) (encKey, macKey []byte, err error) {
	if len(password) == 0 {
		return nil, nil, ErrEmptyPassword
	}

	// Derive master key using Argon2id (same parameters as vault)
	masterKey := crypto.DeriveKey(password, salt)
	defer crypto.SecureWipe(masterKey)

	// Use HKDF to derive separate encryption and MAC keys
	encKey, err = deriveHKDF(masterKey, []byte(hkdfInfoEncryption))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	macKey, err = deriveHKDF(masterKey, []byte(hkdfInfoMAC))
	if err != nil {
		crypto.SecureWipe(encKey)
		return nil, nil, fmt.Errorf("failed to derive MAC key: %w", err)
	}

	return encKey, macKey, nil
}

// deriveHKDF derives a key using HKDF-SHA256.
func deriveHKDF(secret, info []byte) ([]byte, error) {
	hkdfReader := hkdf.New(sha256.New, secret, nil, info)
	key := make([]byte, KeyLength)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptPayload encrypts the payload using AES-256-GCM.
// Returns nonce prepended to ciphertext.
func EncryptPayload(plaintext, key []byte) ([]byte, error) {
	ciphertext, nonce, err := crypto.Encrypt(key, plaintext)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}
	// Prepend nonce to ciphertext for storage
	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result[:len(nonce)], nonce)
	copy(result[len(nonce):], ciphertext)
	return result, nil
}

// DecryptPayload decrypts the payload using AES-256-GCM.
// Expects nonce prepended to ciphertext.
func DecryptPayload(data, key []byte) ([]byte, error) {
	if len(data) < crypto.NonceLength {
		return nil, ErrDecryptionFailed
	}
	nonce := data[:crypto.NonceLength]
	ciphertext := data[crypto.NonceLength:]
	plaintext, err := crypto.Decrypt(key, ciphertext, nonce)
	if err != nil {
		return nil, ErrDecryptionFailed
	}
	return plaintext, nil
}

// ComputeHMAC computes HMAC-SHA256 over the given data.
func ComputeHMAC(data, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// VerifyHMAC verifies the HMAC-SHA256 of the given data.
func VerifyHMAC(data, expectedMAC, key []byte) bool {
	actualMAC := ComputeHMAC(data, key)
	return hmac.Equal(actualMAC, expectedMAC)
}

// ReadKeyFile reads a 32-byte encryption key from a file.
func ReadKeyFile(path string) ([]byte, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	if len(key) != KeyLength {
		crypto.SecureWipe(key)
		return nil, ErrInvalidKeyFile
	}

	return key, nil
}

// GenerateKeyFile generates a random 32-byte key and writes it to a file.
func GenerateKeyFile(path string) error {
	key := make([]byte, KeyLength)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}
	defer crypto.SecureWipe(key)

	// Write with secure permissions (0600)
	if err := os.WriteFile(path, key, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}
