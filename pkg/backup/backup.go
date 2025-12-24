// Package backup provides vault backup and restore functionality.
//
// Features:
//   - Encrypted backup with AES-256-GCM
//   - Argon2id key derivation with separate backup salt
//   - HMAC-SHA256 integrity verification
//   - Atomic restore with temp directory swap
//   - Optional audit log inclusion
//
// Security:
//   - Backup salt is generated fresh for each backup (never reuses vault.salt)
//   - Outer HMAC covers header + ciphertext for tamper detection
//   - File permissions: 0600 for files, 0700 for directories
//   - Sensitive data cleared from memory with SecureWipe
package backup

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/forest6511/secretctl/pkg/crypto"
	"github.com/forest6511/secretctl/pkg/vault"
)

// ConflictMode specifies how to handle key conflicts during restore.
type ConflictMode int

const (
	// ConflictError returns an error if a key already exists.
	ConflictError ConflictMode = iota
	// ConflictSkip skips existing keys and only adds new ones.
	ConflictSkip
	// ConflictOverwrite overwrites existing keys.
	ConflictOverwrite
)

// BackupOptions configures the backup operation.
type BackupOptions struct {
	// Output is the destination writer for the backup.
	Output io.Writer
	// IncludeAudit includes audit logs in the backup.
	IncludeAudit bool
	// Password for encryption (if nil, uses master password).
	Password []byte
	// KeyFile path for encryption key (overrides Password).
	KeyFile string
}

// RestoreOptions configures the restore operation.
type RestoreOptions struct {
	// VaultPath is the target vault directory.
	VaultPath string
	// OnConflict specifies how to handle existing keys.
	OnConflict ConflictMode
	// DryRun previews restore without making changes.
	DryRun bool
	// VerifyOnly only verifies backup integrity.
	VerifyOnly bool
	// WithAudit restores audit logs (overwrites existing).
	WithAudit bool
	// Password for decryption.
	Password []byte
	// KeyFile path for decryption key (overrides Password).
	KeyFile string
}

// RestoreResult contains the result of a restore operation.
type RestoreResult struct {
	// SecretsRestored is the number of secrets restored.
	SecretsRestored int
	// SecretsSkipped is the number of secrets skipped (conflicts).
	SecretsSkipped int
	// AuditRestored indicates if audit logs were restored.
	AuditRestored bool
	// DryRun indicates this was a dry run.
	DryRun bool
}

// VerifyResult contains the result of a verify operation.
type VerifyResult struct {
	// Valid indicates the backup passed all integrity checks.
	Valid bool
	// Version is the backup format version.
	Version int
	// CreatedAt is when the backup was created.
	CreatedAt time.Time
	// SecretCount is the number of secrets in the backup.
	SecretCount int
	// IncludesAudit indicates if audit logs are included.
	IncludesAudit bool
	// Error is set if verification failed.
	Error string
}

// Backup creates an encrypted backup of the vault.
func Backup(v *vault.Vault, opts BackupOptions) error {
	if opts.Output == nil {
		return fmt.Errorf("output writer is required")
	}

	// Determine encryption key
	var encKey, macKey []byte
	var kdfParams *KDFParams
	var encMode EncryptionMode
	var err error

	if opts.KeyFile != "" {
		// Use key file
		encKey, err = ReadKeyFile(opts.KeyFile)
		if err != nil {
			return err
		}
		defer crypto.SecureWipe(encKey)

		// Derive MAC key from encryption key
		macKey, err = deriveHKDF(encKey, []byte(hkdfInfoMAC))
		if err != nil {
			return fmt.Errorf("failed to derive MAC key: %w", err)
		}
		defer crypto.SecureWipe(macKey)

		encMode = EncryptionModeKey
	} else {
		// Use password (master or custom)
		password := opts.Password
		if password == nil {
			return fmt.Errorf("password or key file is required")
		}

		// Generate fresh salt for backup
		salt, err := GenerateSalt()
		if err != nil {
			return err
		}

		encKey, macKey, err = DeriveBackupKeys(password, salt)
		if err != nil {
			return err
		}
		defer crypto.SecureWipe(encKey)
		defer crypto.SecureWipe(macKey)

		kdfParams = &KDFParams{
			Salt:        salt,
			Memory:      crypto.Argon2Memory,
			Iterations:  crypto.Argon2Time,
			Parallelism: crypto.Argon2Threads,
		}
		encMode = EncryptionModeMaster
	}

	// Collect vault data
	payload, secretCount, err := collectVaultData(v, opts.IncludeAudit)
	if err != nil {
		return fmt.Errorf("failed to collect vault data: %w", err)
	}

	// Encode payload
	payloadBytes, err := EncodePayload(payload)
	if err != nil {
		return err
	}
	defer crypto.SecureWipe(payloadBytes)

	// Encrypt payload
	ciphertext, err := EncryptPayload(payloadBytes, encKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt payload: %w", err)
	}

	// Create header
	header := &Header{
		Version:        FormatVersion,
		CreatedAt:      time.Now().UTC(),
		VaultVersion:   1, // TODO: get from vault metadata
		EncryptionMode: encMode,
		KDFParams:      kdfParams,
		IncludesAudit:  opts.IncludeAudit,
		SecretCount:    secretCount,
		ChecksumAlgo:   "sha256",
	}

	// Write to buffer first (for HMAC calculation)
	var buf bytes.Buffer

	// Write header
	if err := WriteHeader(&buf, header); err != nil {
		return err
	}

	// Write ciphertext length and data
	ciphertextLen := uint32(len(ciphertext))
	if err := writeUint32(&buf, ciphertextLen); err != nil {
		return err
	}
	if _, err := buf.Write(ciphertext); err != nil {
		return fmt.Errorf("failed to write ciphertext: %w", err)
	}

	// Compute HMAC over header + ciphertext
	hmacValue := ComputeHMAC(buf.Bytes(), macKey)

	// Write everything to output
	if _, err := opts.Output.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}
	if _, err := opts.Output.Write(hmacValue); err != nil {
		return fmt.Errorf("failed to write HMAC: %w", err)
	}

	return nil
}

// Restore restores a vault from an encrypted backup.
func Restore(backupPath string, opts RestoreOptions) (*RestoreResult, error) {
	// Read backup file
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	// Verify and decrypt
	header, payload, err := verifyAndDecrypt(data, opts.Password, opts.KeyFile)
	if err != nil {
		return nil, err
	}

	if opts.VerifyOnly {
		return &RestoreResult{
			SecretsRestored: 0,
			SecretsSkipped:  0,
			AuditRestored:   false,
			DryRun:          true,
		}, nil
	}

	if opts.DryRun {
		return &RestoreResult{
			SecretsRestored: header.SecretCount,
			SecretsSkipped:  0,
			AuditRestored:   header.IncludesAudit && opts.WithAudit,
			DryRun:          true,
		}, nil
	}

	// Perform actual restore
	result, err := performRestore(opts, header, payload)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Verify checks backup integrity without restoring.
func Verify(backupPath string, password []byte, keyFile string) (*VerifyResult, error) {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return &VerifyResult{Valid: false, Error: err.Error()}, nil
	}

	header, _, err := verifyAndDecrypt(data, password, keyFile)
	if err != nil {
		return &VerifyResult{Valid: false, Error: err.Error()}, nil
	}

	return &VerifyResult{
		Valid:         true,
		Version:       header.Version,
		CreatedAt:     header.CreatedAt,
		SecretCount:   header.SecretCount,
		IncludesAudit: header.IncludesAudit,
	}, nil
}

// collectVaultData collects all vault data for backup.
func collectVaultData(v *vault.Vault, includeAudit bool) (*Payload, int, error) {
	vaultPath := v.Path()

	// Read vault.salt
	saltPath := filepath.Join(vaultPath, "vault.salt")
	vaultSalt, err := os.ReadFile(saltPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read vault.salt: %w", err)
	}

	// Read vault.meta
	metaPath := filepath.Join(vaultPath, "vault.meta")
	vaultMeta, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read vault.meta: %w", err)
	}

	// Read vault.db
	dbPath := filepath.Join(vaultPath, "vault.db")
	vaultDB, err := os.ReadFile(dbPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read vault.db: %w", err)
	}

	// Get secret count
	secrets, err := v.ListSecrets()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list secrets: %w", err)
	}

	payload := &Payload{
		VaultSalt: vaultSalt,
		VaultMeta: vaultMeta,
		VaultDB:   vaultDB,
	}

	// Read audit log if requested
	if includeAudit {
		auditPath := filepath.Join(vaultPath, "audit.jsonl")
		auditData, err := os.ReadFile(auditPath)
		if err == nil {
			payload.AuditLog = auditData
		}
		// Ignore error if audit file doesn't exist
	}

	return payload, len(secrets), nil
}

// verifyAndDecrypt verifies the backup integrity and decrypts the payload.
func verifyAndDecrypt(data []byte, password []byte, keyFile string) (*Header, *Payload, error) {
	if len(data) < 8+4+HMACLength {
		return nil, nil, ErrInvalidMagic
	}

	// Read header
	reader := bytes.NewReader(data)
	header, err := ReadHeader(reader)
	if err != nil {
		return nil, nil, err
	}

	// Get current position (after header)
	headerEnd := len(data) - int(reader.Len())

	// Read ciphertext length
	var ciphertextLen uint32
	if err := readUint32(reader, &ciphertextLen); err != nil {
		return nil, nil, fmt.Errorf("failed to read ciphertext length: %w", err)
	}

	// Verify we have enough data
	remaining := reader.Len()
	if remaining < int(ciphertextLen)+HMACLength {
		return nil, nil, fmt.Errorf("backup file truncated")
	}

	// Read ciphertext
	ciphertext := make([]byte, ciphertextLen)
	if _, err := io.ReadFull(reader, ciphertext); err != nil {
		return nil, nil, fmt.Errorf("failed to read ciphertext: %w", err)
	}

	// Read HMAC
	storedHMAC := make([]byte, HMACLength)
	if _, err := io.ReadFull(reader, storedHMAC); err != nil {
		return nil, nil, fmt.Errorf("failed to read HMAC: %w", err)
	}

	// Derive keys
	var encKey, macKey []byte

	if keyFile != "" {
		encKey, err = ReadKeyFile(keyFile)
		if err != nil {
			return nil, nil, err
		}
		defer crypto.SecureWipe(encKey)

		macKey, err = deriveHKDF(encKey, []byte(hkdfInfoMAC))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to derive MAC key: %w", err)
		}
		defer crypto.SecureWipe(macKey)
	} else if header.EncryptionMode == EncryptionModeMaster && header.KDFParams != nil {
		if password == nil {
			return nil, nil, ErrEmptyPassword
		}
		encKey, macKey, err = DeriveBackupKeys(password, header.KDFParams.Salt)
		if err != nil {
			return nil, nil, err
		}
		defer crypto.SecureWipe(encKey)
		defer crypto.SecureWipe(macKey)
	} else {
		return nil, nil, fmt.Errorf("cannot determine decryption key")
	}

	// Verify HMAC (header + ciphertext length + ciphertext)
	dataToVerify := data[:headerEnd+4+int(ciphertextLen)]
	if !VerifyHMAC(dataToVerify, storedHMAC, macKey) {
		return nil, nil, ErrIntegrityFailed
	}

	// Decrypt payload
	plaintext, err := DecryptPayload(ciphertext, encKey)
	if err != nil {
		return nil, nil, err
	}
	defer crypto.SecureWipe(plaintext)

	// Decode payload
	payload, err := DecodePayload(plaintext)
	if err != nil {
		return nil, nil, err
	}

	return header, payload, nil
}

// DefaultVaultPath returns the default vault path (~/.secretctl).
func DefaultVaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".secretctl"
	}
	return filepath.Join(home, ".secretctl")
}

// performRestore performs the actual restore operation.
func performRestore(opts RestoreOptions, header *Header, payload *Payload) (*RestoreResult, error) {
	vaultPath := opts.VaultPath
	if vaultPath == "" {
		vaultPath = DefaultVaultPath()
	}

	// Create temp directory for atomic restore
	tempDir, err := os.MkdirTemp("", "secretctl-restore-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Set secure permissions on temp dir
	if err := os.Chmod(tempDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to set temp directory permissions: %w", err)
	}

	// Write vault files to temp directory
	if err := os.WriteFile(filepath.Join(tempDir, "vault.salt"), payload.VaultSalt, 0600); err != nil {
		return nil, fmt.Errorf("failed to write vault.salt: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "vault.meta"), payload.VaultMeta, 0600); err != nil {
		return nil, fmt.Errorf("failed to write vault.meta: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "vault.db"), payload.VaultDB, 0600); err != nil {
		return nil, fmt.Errorf("failed to write vault.db: %w", err)
	}

	// Write audit log if included and requested
	auditRestored := false
	if opts.WithAudit && len(payload.AuditLog) > 0 {
		if err := os.WriteFile(filepath.Join(tempDir, "audit.jsonl"), payload.AuditLog, 0600); err != nil {
			return nil, fmt.Errorf("failed to write audit.jsonl: %w", err)
		}
		auditRestored = true
	}

	// Check if vault already exists
	if _, err := os.Stat(vaultPath); err == nil {
		// Vault exists - handle based on conflict mode
		switch opts.OnConflict {
		case ConflictError:
			return nil, fmt.Errorf("vault already exists at %s (use --on-conflict to override)", vaultPath)
		case ConflictSkip:
			return &RestoreResult{
				SecretsRestored: 0,
				SecretsSkipped:  header.SecretCount,
				AuditRestored:   false,
				DryRun:          false,
			}, nil
		case ConflictOverwrite:
			// Remove existing vault
			if err := os.RemoveAll(vaultPath); err != nil {
				return nil, fmt.Errorf("failed to remove existing vault: %w", err)
			}
		}
	}

	// Create parent directory if needed
	parentDir := filepath.Dir(vaultPath)
	if err := os.MkdirAll(parentDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Move temp directory to vault path (atomic on same filesystem)
	if err := os.Rename(tempDir, vaultPath); err != nil {
		// If rename fails (cross-device), fall back to copy
		if err := copyDir(tempDir, vaultPath); err != nil {
			return nil, fmt.Errorf("failed to restore vault: %w", err)
		}
	}

	return &RestoreResult{
		SecretsRestored: header.SecretCount,
		SecretsSkipped:  0,
		AuditRestored:   auditRestored,
		DryRun:          false,
	}, nil
}

// copyDir copies a directory recursively.
func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0700); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0600); err != nil {
				return err
			}
		}
	}

	return nil
}

// writeUint32 writes a uint32 in big-endian format.
func writeUint32(w io.Writer, v uint32) error {
	buf := make([]byte, 4)
	buf[0] = byte(v >> 24)
	buf[1] = byte(v >> 16)
	buf[2] = byte(v >> 8)
	buf[3] = byte(v)
	_, err := w.Write(buf)
	return err
}

// readUint32 reads a uint32 in big-endian format.
func readUint32(r io.Reader, v *uint32) error {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	*v = uint32(buf[0])<<24 | uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])
	return nil
}
