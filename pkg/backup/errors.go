// Package backup provides vault backup and restore functionality.
package backup

import "errors"

// Backup/Restore errors
var (
	// ErrInvalidMagic indicates the backup file has an invalid magic number.
	ErrInvalidMagic = errors.New("invalid backup file: magic number mismatch")

	// ErrUnsupportedVersion indicates the backup format version is not supported.
	ErrUnsupportedVersion = errors.New("unsupported backup format version")

	// ErrIntegrityFailed indicates the HMAC verification failed.
	ErrIntegrityFailed = errors.New("backup integrity check failed: HMAC mismatch")

	// ErrDecryptionFailed indicates decryption failed due to invalid password or corruption.
	ErrDecryptionFailed = errors.New("backup decryption failed: invalid password or corrupted data")

	// ErrVaultLocked indicates another process holds the vault lock.
	ErrVaultLocked = errors.New("vault is locked by another process")

	// ErrVaultNotFound indicates no vault exists at the expected path.
	ErrVaultNotFound = errors.New("vault not found")

	// ErrConflict indicates a key already exists during restore.
	ErrConflict = errors.New("restore conflict: key already exists")

	// ErrPartialRestore indicates restore failed partway through.
	ErrPartialRestore = errors.New("partial restore failed: rolling back")

	// ErrInvalidKeyFile indicates the key file is invalid or wrong size.
	ErrInvalidKeyFile = errors.New("invalid key file: must be exactly 32 bytes")

	// ErrAuditChainInvalid indicates the audit log chain verification failed.
	ErrAuditChainInvalid = errors.New("audit log chain verification failed")

	// ErrEmptyPassword indicates an empty password was provided.
	ErrEmptyPassword = errors.New("password cannot be empty")
)
