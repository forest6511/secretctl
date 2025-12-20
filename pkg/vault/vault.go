// Package vault provides secure secret storage with AES-256-GCM encryption.
// Implements the vault management specified in security-design-ja.md §2-3.
package vault

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/forest6511/secretctl/pkg/audit"
	"github.com/forest6511/secretctl/pkg/crypto"

	_ "github.com/mattn/go-sqlite3"
)

// Constants
const (
	SaltLength   = 16 // 128-bit salt
	DEKLength    = 32 // 256-bit DEK
	SaltFileName = "vault.salt"
	MetaFileName = "vault.meta"
	DBFileName   = "vault.db"
	LockFileName = "vault.lock"
	FileMode     = 0600 // Owner read/write only
	DirMode      = 0700 // Owner read/write/execute only

	// Unlock attempt limits per requirements-ja.md §1.1
	// 5 attempts -> 30s, 10 attempts -> 5min, 20 attempts -> 30min
	CooldownThreshold1 = 5    // First cooldown threshold
	CooldownThreshold2 = 10   // Second cooldown threshold
	CooldownThreshold3 = 20   // Third cooldown threshold
	CooldownDuration1  = 30   // 30 seconds for 5 failures
	CooldownDuration2  = 300  // 5 minutes for 10 failures
	CooldownDuration3  = 1800 // 30 minutes for 20 failures

	// Disk capacity thresholds
	MinDiskSpaceBytes  = 10 * 1024 * 1024 // 10 MB minimum free space
	DiskWarningPercent = 90               // Warn when disk is 90% full

	// Input validation limits
	MaxKeyLength = 256         // Maximum key name length
	MaxValueSize = 1024 * 1024 // 1 MB maximum value size
	MinKeyLength = 1           // Minimum key name length

	// Metadata validation limits per requirements-ja.md §2.5
	MaxNotesSize = 10 * 1024 // 10 KB maximum notes size
	MaxURLLength = 2048      // Maximum URL length (RFC 3986)
	MaxTagCount  = 10        // Maximum number of tags
	MaxTagLength = 64        // Maximum length of each tag
	MinTagLength = 1         // Minimum length of each tag
)

// Errors
var (
	ErrVaultAlreadyExists   = errors.New("vault: vault already exists at this path")
	ErrVaultNotFound        = errors.New("vault: vault not found at this path")
	ErrVaultLocked          = errors.New("vault: vault is locked")
	ErrVaultAlreadyUnlocked = errors.New("vault: vault is already unlocked")
	ErrInvalidPassword      = errors.New("vault: invalid master password")
	ErrSaltNotFound         = errors.New("vault: salt file not found")
	ErrDEKNotFound          = errors.New("vault: encrypted DEK not found in database")
	ErrSecretNotFound       = errors.New("vault: secret not found")
	ErrVaultCorrupted       = errors.New("vault: vault is corrupted")
	ErrMetadataCorrupted    = errors.New("vault: metadata file is corrupted")
	ErrDatabaseCorrupted    = errors.New("vault: database is corrupted")
	ErrTooManyAttempts      = errors.New("vault: too many failed unlock attempts")
	ErrCooldownActive       = errors.New("vault: cooldown period active")
	ErrInsufficientDisk     = errors.New("vault: insufficient disk space")
	ErrKeyTooLong           = errors.New("vault: key name too long")
	ErrKeyTooShort          = errors.New("vault: key name too short")
	ErrKeyInvalid           = errors.New("vault: key name contains invalid characters")
	ErrValueTooLarge        = errors.New("vault: value too large")
	ErrNotesTooLarge        = errors.New("vault: notes too large")
	ErrURLTooLong           = errors.New("vault: url too long")
	ErrURLInvalid           = errors.New("vault: invalid url format")
	ErrTooManyTags          = errors.New("vault: too many tags")
	ErrTagInvalid           = errors.New("vault: invalid tag format")
	ErrExpiresInPast        = errors.New("vault: expires_at must be in the future")
	ErrPasswordTooShort     = errors.New("vault: password must be at least 8 characters")
	ErrPasswordTooLong      = errors.New("vault: password must be at most 128 characters")
)

// Password validation constants per requirements-ja.md §2.3
const (
	MinPasswordLength = 8
	MaxPasswordLength = 128
)

// PasswordStrength represents the strength level of a password
type PasswordStrength int

const (
	PasswordWeak PasswordStrength = iota
	PasswordFair
	PasswordGood
	PasswordStrong
)

// String returns a human-readable representation of password strength
func (s PasswordStrength) String() string {
	switch s {
	case PasswordWeak:
		return "weak"
	case PasswordFair:
		return "fair"
	case PasswordGood:
		return "good"
	case PasswordStrong:
		return "strong"
	default:
		return "unknown"
	}
}

// PasswordValidationResult contains the result of password validation
type PasswordValidationResult struct {
	Valid    bool             // Whether password meets minimum requirements
	Strength PasswordStrength // Estimated strength
	Warnings []string         // Suggestions for improvement (not errors)
}

// ValidateMasterPassword validates a master password per requirements-ja.md §2.3
// Returns validation result with strength assessment and warnings (not errors for complexity)
func ValidateMasterPassword(password string) *PasswordValidationResult {
	result := &PasswordValidationResult{
		Valid:    true,
		Strength: PasswordFair,
	}

	// Hard requirements (errors)
	if len(password) < MinPasswordLength {
		result.Valid = false
		result.Strength = PasswordWeak
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Password must be at least %d characters", MinPasswordLength))
		return result
	}
	if len(password) > MaxPasswordLength {
		result.Valid = false
		result.Strength = PasswordWeak
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Password must be at most %d characters", MaxPasswordLength))
		return result
	}

	// Complexity checks (warnings only, per requirements)
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`\d`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>\-_=+\[\]\\;'~/\x60]`).MatchString(password)

	complexity := 0
	if hasUpper {
		complexity++
	}
	if hasLower {
		complexity++
	}
	if hasDigit {
		complexity++
	}
	if hasSpecial {
		complexity++
	}

	// Generate warnings for weak complexity
	if complexity < 2 {
		result.Warnings = append(result.Warnings,
			"Consider using a mix of uppercase, lowercase, numbers, and symbols")
	}
	if len(password) < 12 {
		result.Warnings = append(result.Warnings,
			"Longer passwords (12+ characters) are more secure")
	}

	// Determine strength based on complexity and length
	switch {
	case complexity >= 3 && len(password) >= 16:
		result.Strength = PasswordStrong
	case complexity >= 2 && len(password) >= 12:
		result.Strength = PasswordGood
	case complexity >= 2 || len(password) >= 12:
		result.Strength = PasswordFair
	default:
		result.Strength = PasswordWeak
	}

	return result
}

// VaultMeta holds vault metadata
type VaultMeta struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

// LockState tracks failed unlock attempts for cooldown enforcement
type LockState struct {
	FailedAttempts int       `json:"failed_attempts"`
	LastAttempt    time.Time `json:"last_attempt"`
	CooldownUntil  time.Time `json:"cooldown_until"`
	LockoutCount   int       `json:"lockout_count"` // Number of times cooldown was triggered
}

// SecretMetadata contains encrypted auxiliary data (stored as single JSON blob)
// Per project-proposal-ja.md: notes/url are encrypted together
type SecretMetadata struct {
	Notes string `json:"notes,omitempty"` // Encrypted: additional notes
	URL   string `json:"url,omitempty"`   // Encrypted: associated URL
}

// SecretEntry represents a complete secret with all its data
// This is the primary structure for secret operations
type SecretEntry struct {
	Key       string          // Secret key name
	Value     []byte          // Secret value
	Metadata  *SecretMetadata // Encrypted metadata (notes, url)
	Tags      []string        // Plaintext: searchable tags
	ExpiresAt *time.Time      // Plaintext: expiration date
	CreatedAt time.Time       // Creation timestamp
	UpdatedAt time.Time       // Last update timestamp
}

// Vault manages the entire secret storage
type Vault struct {
	path  string        // Path to vault directory (e.g., ~/.secretctl)
	dek   []byte        // Decrypted Data Encryption Key (held in memory when unlocked)
	db    *sql.DB       // SQLite database connection
	mu    sync.RWMutex  // Concurrency control
	audit *audit.Logger // Audit logger
}

// New creates a new Vault management object for the specified path
func New(path string) *Vault {
	auditPath := filepath.Join(path, "audit")
	return &Vault{
		path:  path,
		audit: audit.NewLogger(auditPath),
	}
}

// Init initializes a new vault:
// 1. Generate salt and save to vault.salt
// 2. Derive KEK from master password and salt
// 3. Generate DEK
// 4. Encrypt DEK with KEK
// 5. Create vault.db and define tables
// 6. Save encrypted DEK to database
// 7. Create vault.meta file
func (v *Vault) Init(masterPassword string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Check if vault already exists
	if v.exists() {
		return ErrVaultAlreadyExists
	}

	// Check disk space before initialization (per Codex review)
	if err := v.checkDiskSpaceForWrite(1024 * 1024); err != nil { // Require at least 1MB for init
		return err
	}

	// Create vault directory
	if err := os.MkdirAll(v.path, DirMode); err != nil {
		return fmt.Errorf("vault: failed to create vault directory: %w", err)
	}

	// 1. Generate and save salt (16 bytes)
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("vault: failed to generate salt: %w", err)
	}
	saltPath := filepath.Join(v.path, SaltFileName)
	if err := os.WriteFile(saltPath, salt, FileMode); err != nil {
		return fmt.Errorf("vault: failed to write salt file: %w", err)
	}

	// 2. Derive KEK using crypto.DeriveKey
	kek := crypto.DeriveKey([]byte(masterPassword), salt)

	// 3. Generate DEK (32 bytes)
	dek := make([]byte, DEKLength)
	if _, err := rand.Read(dek); err != nil {
		return fmt.Errorf("vault: failed to generate DEK: %w", err)
	}

	// 4. Encrypt DEK using crypto.Encrypt
	encryptedDEK, nonce, err := crypto.Encrypt(kek, dek)
	if err != nil {
		return fmt.Errorf("vault: failed to encrypt DEK: %w", err)
	}

	// 5. Initialize SQLite database
	dbPath := filepath.Join(v.path, DBFileName)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("vault: failed to open database: %w", err)
	}
	defer db.Close()

	// Create tables
	if err := v.createTables(db); err != nil {
		return fmt.Errorf("vault: failed to create tables: %w", err)
	}

	// 6. Save encrypted DEK and nonce to database
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("vault: failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO vault_keys(encrypted_dek, dek_nonce) VALUES(?, ?)")
	if err != nil {
		return fmt.Errorf("vault: failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	if _, err := stmt.Exec(encryptedDEK, nonce); err != nil {
		return fmt.Errorf("vault: failed to save encrypted DEK: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("vault: failed to commit transaction: %w", err)
	}

	// 7. Create metadata file
	meta := VaultMeta{
		Version:   "1.0.0",
		CreatedAt: time.Now().UTC(),
	}
	metaJSON, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("vault: failed to marshal metadata: %w", err)
	}
	metaPath := filepath.Join(v.path, MetaFileName)
	if err := os.WriteFile(metaPath, metaJSON, FileMode); err != nil {
		return fmt.Errorf("vault: failed to write metadata file: %w", err)
	}

	// Set file permissions (0600)
	if err := os.Chmod(dbPath, FileMode); err != nil {
		return fmt.Errorf("vault: failed to set database permissions: %w", err)
	}

	// Initialize audit logger with derived key and log vault init
	if err := v.audit.SetHMACKey(dek); err != nil {
		// Non-fatal: audit logging is best-effort in Phase 0
		fmt.Fprintf(os.Stderr, "warning: failed to initialize audit logger: %v\n", err)
	} else {
		_ = v.audit.LogSuccess(audit.OpVaultInit, audit.SourceCLI, "")
	}

	return nil
}

// Unlock unlocks the vault using the master password:
// 1. Check cooldown status
// 2. Read salt file
// 3. Derive KEK from master password and salt
// 4. Read encrypted DEK and nonce from database
// 5. Decrypt DEK using KEK
// 6. Store decrypted DEK in Vault struct
func (v *Vault) Unlock(masterPassword string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Check if vault exists
	if !v.exists() {
		return ErrVaultNotFound
	}

	// Check if already unlocked
	if v.dek != nil {
		return ErrVaultAlreadyUnlocked
	}

	// Check cooldown status
	if remaining, err := v.checkCooldown(); err != nil {
		if errors.Is(err, ErrCooldownActive) {
			return fmt.Errorf("%w: please wait %v", ErrCooldownActive, remaining.Round(time.Second))
		}
		return err
	}

	// 1. Read salt and validate length
	saltPath := filepath.Join(v.path, SaltFileName)
	salt, err := os.ReadFile(saltPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrSaltNotFound
		}
		return fmt.Errorf("vault: failed to read salt file: %w", err)
	}

	// Validate salt length to detect corruption/tampering
	if len(salt) != SaltLength {
		return ErrVaultCorrupted
	}

	// 2. Derive KEK
	kek := crypto.DeriveKey([]byte(masterPassword), salt)

	// 3. Read encrypted DEK and nonce from database
	dbPath := filepath.Join(v.path, DBFileName)
	db, err := sql.Open("sqlite3", dbPath+"?_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("vault: failed to open database: %w", err)
	}

	// Configure SQLite for single-connection mode to avoid "database is locked" errors
	// This is appropriate for CLI usage where concurrent access is limited
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	var encryptedDEK, nonce []byte
	err = db.QueryRow("SELECT encrypted_dek, dek_nonce FROM vault_keys WHERE id = 1").
		Scan(&encryptedDEK, &nonce)
	if err != nil {
		db.Close()
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDEKNotFound
		}
		return fmt.Errorf("vault: failed to read encrypted DEK: %w", err)
	}

	// 4. Decrypt DEK
	dek, err := crypto.Decrypt(kek, encryptedDEK, nonce)
	if err != nil {
		db.Close()
		if errors.Is(err, crypto.ErrDecryptionFailed) {
			// Record failed attempt and check if cooldown triggered
			cooldown, recordErr := v.recordFailedAttempt()
			if recordErr != nil {
				// Log but don't fail - security is more important than audit
				fmt.Fprintf(os.Stderr, "warning: failed to record unlock attempt: %v\n", recordErr)
			}
			// Log failed unlock attempt
			_ = v.audit.LogError(audit.OpVaultUnlockFailed, audit.SourceCLI, "", "AUTH_FAILED", "invalid master password")
			if cooldown > 0 {
				return fmt.Errorf("%w: cooldown activated for %v", ErrTooManyAttempts, cooldown.Round(time.Second))
			}
			return ErrInvalidPassword
		}
		return fmt.Errorf("vault: failed to decrypt DEK: %w", err)
	}

	// 5. Store DEK in memory on success
	v.dek = dek
	v.db = db

	// Clear lock state on successful unlock
	if err := v.clearLockState(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to clear lock state: %v\n", err)
	}

	// Initialize audit logger with DEK and log successful unlock
	if err := v.audit.SetHMACKey(dek); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to initialize audit logger: %v\n", err)
	} else {
		_ = v.audit.LogSuccess(audit.OpVaultUnlock, audit.SourceCLI, "")
	}

	// Check file permissions and warn if insecure (per requirements-ja.md §4.1)
	// This is a warning only, not blocking - user may have intentional reasons
	v.checkAndWarnPermissions()

	return nil
}

// Lock locks the vault, securely destroying the DEK in memory
func (v *Vault) Lock() {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Log lock operation before clearing DEK
	if v.dek != nil {
		_ = v.audit.LogSuccess(audit.OpVaultLock, audit.SourceCLI, "")
	}

	// Overwrite DEK with zeros for secure destruction
	// Use SecureWipe to prevent compiler optimization from removing the operation
	if v.dek != nil {
		crypto.SecureWipe(v.dek)
		v.dek = nil
	}

	// Close database connection
	if v.db != nil {
		v.db.Close()
		v.db = nil
	}
}

// IsLocked returns whether the vault is locked
func (v *Vault) IsLocked() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.dek == nil
}

// Path returns the vault path
func (v *Vault) Path() string {
	return v.path
}

// exists checks if the vault exists
func (v *Vault) exists() bool {
	saltPath := filepath.Join(v.path, SaltFileName)
	_, err := os.Stat(saltPath)
	return err == nil
}

// checkAndWarnPermissions checks file permissions and prints warnings if insecure.
// Per requirements-ja.md §4.1: "0600以外のパーミッションで警告表示"
// This is advisory only and does not block operations.
func (v *Vault) checkAndWarnPermissions() {
	// Check vault directory (should be 0700)
	if info, err := os.Stat(v.path); err == nil {
		if perm := info.Mode().Perm(); perm&0077 != 0 {
			fmt.Fprintf(os.Stderr, "warning: vault directory has insecure permissions %04o (expected 0700)\n", perm)
		}
	}

	// Check critical files (should be 0600)
	files := []string{SaltFileName, MetaFileName, DBFileName}
	for _, fname := range files {
		fpath := filepath.Join(v.path, fname)
		if info, err := os.Stat(fpath); err == nil {
			if perm := info.Mode().Perm(); perm&0077 != 0 {
				fmt.Fprintf(os.Stderr, "warning: %s has insecure permissions %04o (expected 0600)\n", fname, perm)
			}
		}
	}
}

// createTables creates the required SQLite tables
func (v *Vault) createTables(db *sql.DB) error {
	// vault_keys table (encrypted DEK)
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS vault_keys (
			id INTEGER PRIMARY KEY,
			encrypted_dek BLOB NOT NULL,
			dek_nonce BLOB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// secrets table (hybrid approach: encrypted metadata JSON + plaintext search fields)
	// Per project-proposal-ja.md Phase 0: メタデータ対応 (notes/url/tags/expires_at)
	// - encrypted_key, encrypted_value, encrypted_metadata: nonce prepended
	// - tags, expires_at: plaintext for searchability
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS secrets (
			id INTEGER PRIMARY KEY,
			key_hash TEXT UNIQUE NOT NULL,
			encrypted_key BLOB NOT NULL,
			encrypted_value BLOB NOT NULL,
			encrypted_metadata BLOB,
			tags TEXT,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	return nil
}

// hashKey computes SHA-256 hash of key name for lookup
func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// encryptWithNonce encrypts data and prepends the nonce to the ciphertext.
// This simplifies storage by combining nonce and ciphertext into a single blob.
func (v *Vault) encryptWithNonce(plaintext []byte) ([]byte, error) {
	ciphertext, nonce, err := crypto.Encrypt(v.dek, plaintext)
	if err != nil {
		return nil, err
	}
	return append(nonce, ciphertext...), nil
}

// decryptWithNonce decrypts data where the nonce is prepended to the ciphertext.
func (v *Vault) decryptWithNonce(blob []byte) ([]byte, error) {
	if len(blob) < crypto.NonceLength {
		return nil, fmt.Errorf("vault: invalid encrypted data: too short")
	}
	nonce := blob[:crypto.NonceLength]
	ciphertext := blob[crypto.NonceLength:]
	return crypto.Decrypt(v.dek, ciphertext, nonce)
}

// validateKeyName validates a secret key name per requirements-ja.md §2.1
func validateKeyName(key string) error {
	if len(key) < MinKeyLength {
		return ErrKeyTooShort
	}
	if len(key) > MaxKeyLength {
		return ErrKeyTooLong
	}

	// Check for valid characters: alphanumeric, dash, underscore, dot, slash
	for _, r := range key {
		if !isValidKeyChar(r) {
			return fmt.Errorf("%w: '%c' is not allowed", ErrKeyInvalid, r)
		}
	}

	// Check for dangerous patterns
	if key[0] == '.' || key[0] == '-' {
		return fmt.Errorf("%w: cannot start with '.' or '-'", ErrKeyInvalid)
	}

	// Forbid consecutive dots (path traversal prevention)
	if strings.Contains(key, "..") {
		return fmt.Errorf("%w: cannot contain '..'", ErrKeyInvalid)
	}

	// Forbid leading/trailing slash
	if strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") {
		return fmt.Errorf("%w: cannot start or end with '/'", ErrKeyInvalid)
	}

	// Reserved prefixes for system use
	if strings.HasPrefix(key, "_internal/") || strings.HasPrefix(key, "_system/") {
		return fmt.Errorf("%w: prefix is reserved for system use", ErrKeyInvalid)
	}

	return nil
}

// isValidKeyChar checks if a rune is valid for a key name
func isValidKeyChar(r rune) bool {
	// Allow: a-z, A-Z, 0-9, -, _, ., /
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_' || r == '.' || r == '/'
}

// validateValueSize validates the size of a secret value
func validateValueSize(value []byte) error {
	if len(value) > MaxValueSize {
		return fmt.Errorf("%w: %d bytes exceeds maximum of %d bytes",
			ErrValueTooLarge, len(value), MaxValueSize)
	}
	return nil
}

// tagRegex validates tag format: alphanumeric, underscore, hyphen only
var tagRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateMetadata validates metadata fields per requirements-ja.md §2.5
func validateMetadata(metadata *SecretMetadata, tags []string, expiresAt *time.Time) error {
	// Validate metadata fields if present
	if metadata != nil {
		// notes: maximum 10KB
		if len(metadata.Notes) > MaxNotesSize {
			return fmt.Errorf("%w: %d bytes exceeds maximum of %d bytes",
				ErrNotesTooLarge, len(metadata.Notes), MaxNotesSize)
		}

		// url: maximum 2048 characters, http/https only with host required
		if metadata.URL != "" {
			if len(metadata.URL) > MaxURLLength {
				return fmt.Errorf("%w: %d characters exceeds maximum of %d",
					ErrURLTooLong, len(metadata.URL), MaxURLLength)
			}
			parsedURL, err := url.Parse(metadata.URL)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrURLInvalid, err)
			}
			// Only allow http and https schemes to prevent javascript: and other dangerous schemes
			if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
				return fmt.Errorf("%w: only http and https schemes are allowed", ErrURLInvalid)
			}
			// Require a host to prevent malformed URLs
			if parsedURL.Host == "" {
				return fmt.Errorf("%w: URL must have a host", ErrURLInvalid)
			}
		}
	}

	// Validate tags: maximum 10 tags, each 1-64 characters, pattern [a-zA-Z0-9_-]
	if len(tags) > MaxTagCount {
		return fmt.Errorf("%w: %d tags exceeds maximum of %d",
			ErrTooManyTags, len(tags), MaxTagCount)
	}
	for _, tag := range tags {
		if len(tag) < MinTagLength || len(tag) > MaxTagLength {
			return fmt.Errorf("%w: tag '%s' must be %d-%d characters",
				ErrTagInvalid, tag, MinTagLength, MaxTagLength)
		}
		if !tagRegex.MatchString(tag) {
			return fmt.Errorf("%w: tag '%s' must match [a-zA-Z0-9_-]",
				ErrTagInvalid, tag)
		}
	}

	// Validate expires_at: must be in the future
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return ErrExpiresInPast
	}

	return nil
}

// SetSecret saves or updates a secret with all its data.
// Uses hybrid approach: encrypted metadata JSON blob + plaintext search fields.
func (v *Vault) SetSecret(key string, entry *SecretEntry) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Check if vault is locked
	if v.dek == nil {
		return ErrVaultLocked
	}

	// Validate key name
	if err := validateKeyName(key); err != nil {
		_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "INVALID_KEY", err.Error())
		return err
	}

	// Validate value size
	if err := validateValueSize(entry.Value); err != nil {
		_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "VALUE_TOO_LARGE", err.Error())
		return err
	}

	// Validate metadata per requirements-ja.md §2.5
	if err := validateMetadata(entry.Metadata, entry.Tags, entry.ExpiresAt); err != nil {
		_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "INVALID_METADATA", err.Error())
		return err
	}

	// Check disk space before write
	dataSize := len(key) + len(entry.Value)
	if entry.Metadata != nil {
		dataSize += len(entry.Metadata.Notes) + len(entry.Metadata.URL)
	}
	if err := v.checkDiskSpaceForWrite(dataSize); err != nil {
		_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "DISK_FULL", err.Error())
		return err
	}

	// Compute key hash for lookup
	keyHash := hashKey(key)

	// Encrypt key name (nonce prepended)
	encryptedKey, err := v.encryptWithNonce([]byte(key))
	if err != nil {
		_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "ENCRYPT_FAILED", err.Error())
		return fmt.Errorf("vault: failed to encrypt key: %w", err)
	}

	// Encrypt value (nonce prepended)
	encryptedValue, err := v.encryptWithNonce(entry.Value)
	if err != nil {
		_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "ENCRYPT_FAILED", err.Error())
		return fmt.Errorf("vault: failed to encrypt value: %w", err)
	}

	// Encrypt metadata as JSON blob (nonce prepended)
	var encryptedMetadata []byte
	if entry.Metadata != nil && (entry.Metadata.Notes != "" || entry.Metadata.URL != "") {
		metadataJSON, err := json.Marshal(entry.Metadata)
		if err != nil {
			_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "MARSHAL_FAILED", err.Error())
			return fmt.Errorf("vault: failed to marshal metadata: %w", err)
		}
		encryptedMetadata, err = v.encryptWithNonce(metadataJSON)
		if err != nil {
			_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "ENCRYPT_FAILED", "metadata: "+err.Error())
			return fmt.Errorf("vault: failed to encrypt metadata: %w", err)
		}
	}

	// Prepare plaintext fields for search
	var tagsStr sql.NullString
	if len(entry.Tags) > 0 {
		tagsJSON, err := json.Marshal(entry.Tags)
		if err == nil {
			tagsStr = sql.NullString{String: string(tagsJSON), Valid: true}
		}
	}

	var expiresAt sql.NullTime
	if entry.ExpiresAt != nil {
		expiresAt = sql.NullTime{Time: *entry.ExpiresAt, Valid: true}
	}

	// Begin transaction
	tx, err := v.db.Begin()
	if err != nil {
		return fmt.Errorf("vault: failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// UPSERT: update if key exists, insert otherwise
	_, err = tx.Exec(`
		INSERT INTO secrets (key_hash, encrypted_key, encrypted_value, encrypted_metadata, tags, expires_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key_hash) DO UPDATE SET
			encrypted_key = excluded.encrypted_key,
			encrypted_value = excluded.encrypted_value,
			encrypted_metadata = excluded.encrypted_metadata,
			tags = excluded.tags,
			expires_at = excluded.expires_at,
			updated_at = CURRENT_TIMESTAMP
	`, keyHash, encryptedKey, encryptedValue, encryptedMetadata, tagsStr, expiresAt)
	if err != nil {
		_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "DB_ERROR", err.Error())
		return fmt.Errorf("vault: failed to save secret: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("vault: failed to commit transaction: %w", err)
	}

	// Log successful operation
	_ = v.audit.LogSuccess(audit.OpSecretSet, audit.SourceCLI, key)

	return nil
}

// GetSecret retrieves a complete secret entry by key name
func (v *Vault) GetSecret(key string) (*SecretEntry, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Check if vault is locked
	if v.dek == nil {
		return nil, ErrVaultLocked
	}

	// Compute key hash
	keyHash := hashKey(key)

	// Get all fields from database
	var encryptedValue, encryptedMetadata []byte
	var tagsStr sql.NullString
	var expiresAt sql.NullTime
	var createdAt, updatedAt time.Time

	err := v.db.QueryRow(`
		SELECT encrypted_value, encrypted_metadata, tags, expires_at, created_at, updated_at
		FROM secrets WHERE key_hash = ?`,
		keyHash,
	).Scan(&encryptedValue, &encryptedMetadata, &tagsStr, &expiresAt, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = v.audit.LogError(audit.OpSecretGet, audit.SourceCLI, key, "NOT_FOUND", "secret not found")
			return nil, ErrSecretNotFound
		}
		return nil, fmt.Errorf("vault: failed to read secret: %w", err)
	}

	// Decrypt value (nonce is prepended)
	plainValue, err := v.decryptWithNonce(encryptedValue)
	if err != nil {
		_ = v.audit.LogError(audit.OpSecretGet, audit.SourceCLI, key, "DECRYPT_FAILED", err.Error())
		return nil, fmt.Errorf("vault: failed to decrypt secret: %w", err)
	}

	entry := &SecretEntry{
		Key:       key,
		Value:     plainValue,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	// Decrypt metadata if present
	if len(encryptedMetadata) > 0 {
		metadataJSON, err := v.decryptWithNonce(encryptedMetadata)
		if err != nil {
			return nil, fmt.Errorf("vault: failed to decrypt metadata: %w", err)
		}
		var meta SecretMetadata
		if err := json.Unmarshal(metadataJSON, &meta); err != nil {
			return nil, fmt.Errorf("vault: failed to unmarshal metadata: %w", err)
		}
		entry.Metadata = &meta
	}

	// Parse tags from JSON
	if tagsStr.Valid && tagsStr.String != "" {
		var tags []string
		if err := json.Unmarshal([]byte(tagsStr.String), &tags); err == nil {
			entry.Tags = tags
		}
	}

	// Set expiration
	if expiresAt.Valid {
		entry.ExpiresAt = &expiresAt.Time
	}

	// Log successful operation
	_ = v.audit.LogSuccess(audit.OpSecretGet, audit.SourceCLI, key)

	return entry, nil
}

// ListSecrets retrieves all secret key names
func (v *Vault) ListSecrets() ([]string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Check if vault is locked
	if v.dek == nil {
		return nil, ErrVaultLocked
	}

	// Get all encrypted_key records
	rows, err := v.db.Query("SELECT encrypted_key FROM secrets ORDER BY created_at")
	if err != nil {
		return nil, fmt.Errorf("vault: failed to query secrets: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var encryptedKey []byte
		if err := rows.Scan(&encryptedKey); err != nil {
			return nil, fmt.Errorf("vault: failed to scan row: %w", err)
		}

		// Decrypt key name (nonce is prepended)
		keyBytes, err := v.decryptWithNonce(encryptedKey)
		if err != nil {
			return nil, fmt.Errorf("vault: failed to decrypt key name: %w", err)
		}

		keys = append(keys, string(keyBytes))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vault: error iterating rows: %w", err)
	}

	// Log successful operation
	_ = v.audit.LogSuccess(audit.OpSecretList, audit.SourceCLI, "")

	return keys, nil
}

// DeleteSecret deletes a secret by key name
func (v *Vault) DeleteSecret(key string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Check if vault is locked
	if v.dek == nil {
		return ErrVaultLocked
	}

	// Compute key hash
	keyHash := hashKey(key)

	// Begin transaction
	tx, err := v.db.Begin()
	if err != nil {
		return fmt.Errorf("vault: failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete record
	result, err := tx.Exec("DELETE FROM secrets WHERE key_hash = ?", keyHash)
	if err != nil {
		_ = v.audit.LogError(audit.OpSecretDelete, audit.SourceCLI, key, "DB_ERROR", err.Error())
		return fmt.Errorf("vault: failed to delete secret: %w", err)
	}

	// Check rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("vault: failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		_ = v.audit.LogError(audit.OpSecretDelete, audit.SourceCLI, key, "NOT_FOUND", "secret not found")
		return ErrSecretNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("vault: failed to commit transaction: %w", err)
	}

	// Log successful operation
	_ = v.audit.LogSuccess(audit.OpSecretDelete, audit.SourceCLI, key)

	return nil
}

// scanSecretEntryRow scans a row and returns a SecretEntry (without value).
// This is a helper to avoid code duplication for list operations.
func (v *Vault) scanSecretEntryRow(rows *sql.Rows) (*SecretEntry, error) {
	var encryptedKey []byte
	var tagsStr sql.NullString
	var expiresAt sql.NullTime
	var createdAt, updatedAt time.Time

	if err := rows.Scan(&encryptedKey, &tagsStr, &expiresAt,
		&createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("vault: failed to scan row: %w", err)
	}

	// Decrypt key name (nonce is prepended)
	keyBytes, err := v.decryptWithNonce(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to decrypt key name: %w", err)
	}

	entry := &SecretEntry{
		Key:       string(keyBytes),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	// Parse tags from JSON
	if tagsStr.Valid && tagsStr.String != "" {
		var tags []string
		if err := json.Unmarshal([]byte(tagsStr.String), &tags); err == nil {
			entry.Tags = tags
		}
	}

	if expiresAt.Valid {
		entry.ExpiresAt = &expiresAt.Time
	}

	return entry, nil
}

// scanSecretEntryRowWithMetadata scans a row including metadata (but NOT secret value).
// This decrypts key name and metadata, but never touches the secret value.
// Use this for list operations that need metadata presence (HasNotes, HasURL).
func (v *Vault) scanSecretEntryRowWithMetadata(rows *sql.Rows) (*SecretEntry, error) {
	var encryptedKey []byte
	var encryptedMetadata []byte
	var tagsStr sql.NullString
	var expiresAt sql.NullTime
	var createdAt, updatedAt time.Time

	if err := rows.Scan(&encryptedKey, &encryptedMetadata, &tagsStr, &expiresAt,
		&createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("vault: failed to scan row: %w", err)
	}

	// Decrypt key name (nonce is prepended)
	keyBytes, err := v.decryptWithNonce(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to decrypt key name: %w", err)
	}

	entry := &SecretEntry{
		Key:       string(keyBytes),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	// Decrypt metadata if present (small data - notes/URL, not credentials)
	if len(encryptedMetadata) > 0 {
		metadataJSON, err := v.decryptWithNonce(encryptedMetadata)
		if err != nil {
			// Log but continue - metadata decryption failure is non-fatal for listing
			// We still need to parse tags and expiration below
			log.Printf("vault: metadata decryption failed for key %q: %v", entry.Key, err)
		} else {
			var meta SecretMetadata
			if err := json.Unmarshal(metadataJSON, &meta); err == nil {
				entry.Metadata = &meta
			}
		}
	}

	// Parse tags from JSON
	if tagsStr.Valid && tagsStr.String != "" {
		var tags []string
		if err := json.Unmarshal([]byte(tagsStr.String), &tags); err == nil {
			entry.Tags = tags
		}
	}

	if expiresAt.Valid {
		entry.ExpiresAt = &expiresAt.Time
	}

	return entry, nil
}

// ListSecretsWithMetadata lists all secrets with metadata but WITHOUT decrypting values.
// This is more secure than GetSecret for list operations as it never exposes secret values.
func (v *Vault) ListSecretsWithMetadata() ([]*SecretEntry, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Check if vault is locked
	if v.dek == nil {
		return nil, ErrVaultLocked
	}

	rows, err := v.db.Query(`
		SELECT encrypted_key, encrypted_metadata, tags, expires_at, created_at, updated_at
		FROM secrets
		ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to query secrets: %w", err)
	}
	defer rows.Close()

	var secrets []*SecretEntry
	for rows.Next() {
		entry, err := v.scanSecretEntryRowWithMetadata(rows)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vault: error iterating rows: %w", err)
	}

	return secrets, nil
}

// ListSecretsByTag retrieves secrets filtered by tag (includes metadata, NOT values)
func (v *Vault) ListSecretsByTag(tag string) ([]*SecretEntry, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Check if vault is locked
	if v.dek == nil {
		return nil, ErrVaultLocked
	}

	// Query with tag filter (tags stored as JSON array, search for the tag string)
	// This searches for the tag within the JSON array string
	// Include encrypted_metadata for HasNotes/HasURL support
	rows, err := v.db.Query(`
		SELECT encrypted_key, encrypted_metadata, tags, expires_at, created_at, updated_at
		FROM secrets
		WHERE tags LIKE ?
		ORDER BY created_at`,
		`%"`+tag+`"%`)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to query secrets: %w", err)
	}
	defer rows.Close()

	var secrets []*SecretEntry
	for rows.Next() {
		entry, err := v.scanSecretEntryRowWithMetadata(rows)
		if err != nil {
			return nil, err
		}
		// Double-check the tag is actually in the parsed tags
		for _, t := range entry.Tags {
			if t == tag {
				secrets = append(secrets, entry)
				break
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vault: error iterating rows: %w", err)
	}

	return secrets, nil
}

// ListExpiringSecrets retrieves secrets expiring within the specified duration (includes metadata, NOT values)
func (v *Vault) ListExpiringSecrets(within time.Duration) ([]*SecretEntry, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Check if vault is locked
	if v.dek == nil {
		return nil, ErrVaultLocked
	}

	deadline := time.Now().Add(within)

	// Include encrypted_metadata for HasNotes/HasURL support
	rows, err := v.db.Query(`
		SELECT encrypted_key, encrypted_metadata, tags, expires_at, created_at, updated_at
		FROM secrets
		WHERE expires_at IS NOT NULL AND expires_at <= ?
		ORDER BY expires_at`,
		deadline)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to query secrets: %w", err)
	}
	defer rows.Close()

	var secrets []*SecretEntry
	for rows.Next() {
		entry, err := v.scanSecretEntryRowWithMetadata(rows)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vault: error iterating rows: %w", err)
	}

	return secrets, nil
}

// AuditLogger returns the audit logger for external use
func (v *Vault) AuditLogger() *audit.Logger {
	return v.audit
}

// AuditVerify verifies the integrity of the audit log chain
func (v *Vault) AuditVerify() (*audit.VerifyResult, error) {
	return v.audit.Verify()
}

// IntegrityCheckResult contains the results of vault integrity verification
type IntegrityCheckResult struct {
	Valid            bool     `json:"valid"`
	SaltExists       bool     `json:"salt_exists"`
	MetaValid        bool     `json:"meta_valid"`
	DBExists         bool     `json:"db_exists"`
	DBIntegrity      bool     `json:"db_integrity"`
	PermissionsValid bool     `json:"permissions_valid"`
	Errors           []string `json:"errors,omitempty"`
}

// CheckIntegrity performs a comprehensive integrity check on the vault.
// This checks:
// 1. Salt file exists and has correct size
// 2. Metadata file is valid JSON with required fields
// 3. Database file exists and passes SQLite integrity check
// 4. Database schema contains expected tables
// 5. File permissions are secure (0600 for files, 0700 for directories)
func (v *Vault) CheckIntegrity() (*IntegrityCheckResult, error) {
	result := &IntegrityCheckResult{
		Valid:            true,
		PermissionsValid: true, // Assume valid until proven otherwise
	}

	// Check vault directory permissions (should be 0700)
	// Permission failures are security issues and mark the vault as invalid
	dirInfo, err := os.Stat(v.path)
	if err == nil {
		dirPerm := dirInfo.Mode().Perm()
		if dirPerm&0077 != 0 { // Check if group/other have any permissions
			result.Valid = false
			result.PermissionsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("vault directory has insecure permissions: %04o (expected 0700)", dirPerm))
		}
	}

	// Check salt file
	saltPath := filepath.Join(v.path, SaltFileName)
	saltInfo, err := os.Stat(saltPath)
	if err != nil {
		result.Valid = false
		result.SaltExists = false
		result.Errors = append(result.Errors, "salt file not found: "+saltPath)
	} else {
		result.SaltExists = true
		if saltInfo.Size() != SaltLength {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("salt file has incorrect size: expected %d, got %d", SaltLength, saltInfo.Size()))
		}
		// Check salt file permissions (should be 0600)
		saltPerm := saltInfo.Mode().Perm()
		if saltPerm&0077 != 0 { // Check if group/other have any permissions
			result.Valid = false
			result.PermissionsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("salt file has insecure permissions: %04o (expected 0600)", saltPerm))
		}
	}

	// Check metadata file
	metaPath := filepath.Join(v.path, MetaFileName)
	metaInfo, err := os.Stat(metaPath)
	if err != nil {
		result.Valid = false
		result.MetaValid = false
		result.Errors = append(result.Errors, "metadata file not found: "+metaPath)
	} else {
		// Check metadata file permissions (should be 0600)
		metaPerm := metaInfo.Mode().Perm()
		if metaPerm&0077 != 0 { // Check if group/other have any permissions
			result.Valid = false
			result.PermissionsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("metadata file has insecure permissions: %04o (expected 0600)", metaPerm))
		}

		metaData, err := os.ReadFile(metaPath)
		if err != nil {
			result.Valid = false
			result.MetaValid = false
			result.Errors = append(result.Errors, "failed to read metadata file: "+err.Error())
		} else {
			var meta VaultMeta
			if err := json.Unmarshal(metaData, &meta); err != nil {
				result.Valid = false
				result.MetaValid = false
				result.Errors = append(result.Errors, "metadata file is not valid JSON: "+err.Error())
			} else if meta.Version == "" {
				result.Valid = false
				result.MetaValid = false
				result.Errors = append(result.Errors, "metadata file missing version field")
			} else {
				result.MetaValid = true
			}
		}
	}

	// Check database file
	dbPath := filepath.Join(v.path, DBFileName)
	dbInfo, err := os.Stat(dbPath)
	if err != nil {
		result.Valid = false
		result.DBExists = false
		result.Errors = append(result.Errors, "database file not found: "+dbPath)
		return result, nil
	}
	result.DBExists = true

	// Check database file permissions (should be 0600)
	dbPerm := dbInfo.Mode().Perm()
	if dbPerm&0077 != 0 { // Check if group/other have any permissions
		result.Valid = false
		result.PermissionsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("database file has insecure permissions: %04o (expected 0600)", dbPerm))
	}

	// Open database and check integrity
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		result.Valid = false
		result.DBIntegrity = false
		result.Errors = append(result.Errors, "failed to open database: "+err.Error())
		return result, nil
	}
	defer db.Close()

	// Run SQLite integrity check
	var integrityResult string
	err = db.QueryRow("PRAGMA integrity_check").Scan(&integrityResult)
	if err != nil {
		result.Valid = false
		result.DBIntegrity = false
		result.Errors = append(result.Errors, "database integrity check failed: "+err.Error())
		return result, nil
	}
	if integrityResult != "ok" {
		result.Valid = false
		result.DBIntegrity = false
		result.Errors = append(result.Errors, "database integrity check returned: "+integrityResult)
		return result, nil
	}

	// Check for required tables
	tables := []string{"vault_keys", "secrets"}
	for _, table := range tables {
		var name string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			result.Valid = false
			result.DBIntegrity = false
			result.Errors = append(result.Errors, "required table not found: "+table)
		}
	}

	// DBIntegrity is true only when no errors were found during DB checks
	// The condition uses AND to ensure all checks passed
	if len(result.Errors) == 0 && result.SaltExists && result.MetaValid && result.DBExists {
		result.DBIntegrity = true
	}

	return result, nil
}

// loadLockState reads the lock state from the lock file
func (v *Vault) loadLockState() (*LockState, error) {
	lockPath := filepath.Join(v.path, LockFileName)
	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &LockState{}, nil // No lock state yet
		}
		return nil, fmt.Errorf("vault: failed to read lock state: %w", err)
	}

	var state LockState
	if err := json.Unmarshal(data, &state); err != nil {
		// Corrupted lock file - reset state
		return &LockState{}, nil
	}
	return &state, nil
}

// saveLockState writes the lock state to the lock file
func (v *Vault) saveLockState(state *LockState) error {
	// Check disk space before write (per Codex review)
	if err := v.checkDiskSpaceForWrite(1024); err != nil { // Lock state is small (~1KB)
		return err
	}

	lockPath := filepath.Join(v.path, LockFileName)
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("vault: failed to marshal lock state: %w", err)
	}
	if err := os.WriteFile(lockPath, data, FileMode); err != nil {
		return fmt.Errorf("vault: failed to write lock state: %w", err)
	}
	return nil
}

// clearLockState removes the lock state file (called on successful unlock)
func (v *Vault) clearLockState() error {
	lockPath := filepath.Join(v.path, LockFileName)
	err := os.Remove(lockPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("vault: failed to clear lock state: %w", err)
	}
	return nil
}

// checkCooldown verifies if unlock is allowed or if cooldown is active
func (v *Vault) checkCooldown() (time.Duration, error) {
	state, err := v.loadLockState()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	if !state.CooldownUntil.IsZero() && now.Before(state.CooldownUntil) {
		remaining := state.CooldownUntil.Sub(now)
		return remaining, ErrCooldownActive
	}

	return 0, nil
}

// recordFailedAttempt records a failed unlock attempt and potentially triggers cooldown
// per requirements-ja.md §1.1: 5 attempts -> 30s, 10 attempts -> 5min, 20 attempts -> 30min
func (v *Vault) recordFailedAttempt() (time.Duration, error) {
	state, err := v.loadLockState()
	if err != nil {
		return 0, err
	}

	state.FailedAttempts++
	state.LastAttempt = time.Now()

	var cooldownDuration time.Duration

	// Determine cooldown based on cumulative failed attempts
	switch {
	case state.FailedAttempts >= CooldownThreshold3:
		// 20+ failures -> 30 minute cooldown
		cooldownDuration = time.Duration(CooldownDuration3) * time.Second
		state.CooldownUntil = time.Now().Add(cooldownDuration)
	case state.FailedAttempts >= CooldownThreshold2:
		// 10+ failures -> 5 minute cooldown
		cooldownDuration = time.Duration(CooldownDuration2) * time.Second
		state.CooldownUntil = time.Now().Add(cooldownDuration)
	case state.FailedAttempts >= CooldownThreshold1:
		// 5+ failures -> 30 second cooldown
		cooldownDuration = time.Duration(CooldownDuration1) * time.Second
		state.CooldownUntil = time.Now().Add(cooldownDuration)
	}

	if err := v.saveLockState(state); err != nil {
		return cooldownDuration, err
	}

	return cooldownDuration, nil
}

// GetLockState returns the current lock state for display purposes
func (v *Vault) GetLockState() (*LockState, error) {
	return v.loadLockState()
}

// RemainingCooldown returns the remaining cooldown time, or 0 if not in cooldown
func (v *Vault) RemainingCooldown() time.Duration {
	state, err := v.loadLockState()
	if err != nil {
		return 0
	}

	now := time.Now()
	if !state.CooldownUntil.IsZero() && now.Before(state.CooldownUntil) {
		return state.CooldownUntil.Sub(now)
	}
	return 0
}

// DiskSpaceInfo contains disk usage information
type DiskSpaceInfo struct {
	Total     uint64 `json:"total"`     // Total disk space in bytes
	Free      uint64 `json:"free"`      // Free disk space in bytes
	Available uint64 `json:"available"` // Available to non-root users
	UsedPct   int    `json:"used_pct"`  // Percentage of disk used
}

// CheckDiskSpace returns disk space information for the vault directory
func (v *Vault) CheckDiskSpace() (*DiskSpaceInfo, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(v.path, &stat); err != nil {
		// If vault directory doesn't exist yet, check parent
		parentDir := filepath.Dir(v.path)
		if err := syscall.Statfs(parentDir, &stat); err != nil {
			return nil, fmt.Errorf("vault: failed to get disk stats: %w", err)
		}
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)

	usedPct := 0
	if total > 0 {
		usedPct = int(100 * (total - free) / total)
	}

	return &DiskSpaceInfo{
		Total:     total,
		Free:      free,
		Available: available,
		UsedPct:   usedPct,
	}, nil
}

// HasSufficientDiskSpace checks if there's enough disk space for operations
func (v *Vault) HasSufficientDiskSpace() (bool, error) {
	info, err := v.CheckDiskSpace()
	if err != nil {
		return false, err
	}
	return info.Available >= MinDiskSpaceBytes, nil
}

// IsDiskSpaceLow returns true if disk usage is above warning threshold
func (v *Vault) IsDiskSpaceLow() (bool, error) {
	info, err := v.CheckDiskSpace()
	if err != nil {
		return false, err
	}
	return info.UsedPct >= DiskWarningPercent, nil
}

// checkDiskSpaceForWrite verifies sufficient disk space before write operations
func (v *Vault) checkDiskSpaceForWrite(dataSize int) error {
	info, err := v.CheckDiskSpace()
	if err != nil {
		// Log warning but don't block operation
		fmt.Fprintf(os.Stderr, "warning: failed to check disk space: %v\n", err)
		return nil
	}

	// Need at least MinDiskSpaceBytes or 2x the data size, whichever is larger
	required := uint64(MinDiskSpaceBytes)
	if uint64(dataSize*2) > required {
		required = uint64(dataSize * 2)
	}

	if info.Available < required {
		return fmt.Errorf("%w: only %d MB available, need at least %d MB",
			ErrInsufficientDisk,
			info.Available/(1024*1024),
			required/(1024*1024))
	}

	// Warn if disk space is low
	if info.UsedPct >= DiskWarningPercent {
		fmt.Fprintf(os.Stderr, "warning: disk is %d%% full, consider freeing space\n", info.UsedPct)
	}

	return nil
}

// Repair attempts to repair minor vault issues.
// Currently supports:
// - Recreating missing metadata file (if vault_keys table exists)
func (v *Vault) Repair() error {
	// Check if metadata file exists
	metaPath := filepath.Join(v.path, MetaFileName)
	if _, err := os.Stat(metaPath); err == nil {
		// Metadata exists, check if valid
		data, readErr := os.ReadFile(metaPath)
		if readErr == nil {
			var meta VaultMeta
			if json.Unmarshal(data, &meta) == nil && meta.Version != "" {
				return nil // Already valid
			}
		}
	}

	// Check if database exists with vault_keys
	dbPath := filepath.Join(v.path, DBFileName)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("vault: cannot repair without valid database: %w", err)
	}
	defer db.Close()

	// Get creation timestamp from vault_keys
	var createdAt time.Time
	err = db.QueryRow("SELECT created_at FROM vault_keys WHERE id = 1").Scan(&createdAt)
	if err != nil {
		return fmt.Errorf("vault: cannot determine vault creation time: %w", err)
	}

	// Recreate metadata file
	meta := VaultMeta{
		Version:   "1.0.0",
		CreatedAt: createdAt,
	}
	metaJSON, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("vault: failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(metaPath, metaJSON, FileMode); err != nil {
		return fmt.Errorf("vault: failed to write metadata file: %w", err)
	}

	return nil
}
