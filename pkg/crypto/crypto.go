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

// Argon2idパラメータ（security-design-ja.md準拠、OWASP推奨値）
const (
	Argon2Memory  = 64 * 1024 // 64MB
	Argon2Time    = 3         // 反復回数
	Argon2Threads = 4         // 並列度
	KeyLength     = 32        // AES-256用（256-bit）
	NonceLength   = 12        // GCM標準（96-bit）
)

var (
	ErrInvalidKeyLength   = errors.New("crypto: invalid key length, must be 32 bytes")
	ErrInvalidNonceLength = errors.New("crypto: invalid nonce length, must be 12 bytes")
	ErrDecryptionFailed   = errors.New("crypto: decryption failed, authentication tag verification failed")
	ErrCiphertextTooShort = errors.New("crypto: ciphertext too short")
)

// DeriveKey はマスターパスワードとソルトからArgon2idを用いてKEKを導出する。
// security-design-ja.mdに記載されたパラメータを使用:
// memory=64MB, time=3, threads=4
func DeriveKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, Argon2Time, Argon2Memory, Argon2Threads, KeyLength)
}

// Encrypt は平文データをAES-256-GCMで暗号化する。
// 戻り値: 暗号文、ノンス（96-bit）、エラー
// ノンスはcrypto/randで安全に生成される。
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

	// crypto/randで安全にノンスを生成
	nonce = make([]byte, NonceLength)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("crypto: failed to generate nonce: %w", err)
	}

	// GCMで暗号化（認証タグは暗号文に付加される）
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)

	return ciphertext, nonce, nil
}

// Decrypt はAES-256-GCMで暗号化されたデータを復号する。
// GCMによる認証タグの検証が失敗した場合はエラーを返す。
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

	// 暗号文が最低限の長さを持つか確認（GCMタグ16バイト）
	if len(ciphertext) < gcm.Overhead() {
		return nil, ErrCiphertextTooShort
	}

	// GCMで復号（認証タグ検証を含む）
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
