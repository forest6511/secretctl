// Package vault provides secure secret storage with AES-256-GCM encryption.
//
// The vault stores secrets in an encrypted SQLite database, using a master
// password to derive the encryption key via Argon2id. All secrets are
// encrypted individually with unique nonces.
//
// # Security Features
//
//   - AES-256-GCM authenticated encryption for all secrets
//   - Argon2id key derivation with OWASP-recommended parameters
//   - HMAC-chained audit logging for tamper detection
//   - Rate limiting for unlock attempts (5/10/20 failures = 30s/5m/30m cooldown)
//   - Secure file permissions (0600 for files, 0700 for directories)
//
// # Example Usage
//
//	// Create and initialize a new vault
//	v := vault.New("/path/to/vault")
//	err := v.Init("masterpassword")
//
//	// Open an existing vault
//	v := vault.New("/path/to/vault")
//	err := v.Unlock("masterpassword")
//
//	// Store and retrieve secrets
//	err = v.SetSecret("API_KEY", &vault.SecretEntry{Value: []byte("secret")})
//	entry, err := v.GetSecret("API_KEY")
//
//	// Lock when done
//	err = v.Lock()
package vault

import (
	"crypto/hmac"
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
	"time"

	"github.com/forest6511/secretctl/pkg/audit"
	"github.com/forest6511/secretctl/pkg/crypto"

	_ "modernc.org/sqlite"
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
	ErrSamePassword         = errors.New("vault: new password must be different from current password")
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
//
// Phase 2.5 Multi-Field Support:
// - Fields: map of field name to Field struct (replaces single Value)
// - Bindings: environment variable name to field name mapping
// - Schema: reserved for Phase 3 schema validation
//
// Phase 2c-X2 Folder Support (ADR-007):
// - FolderID: reference to folder for organization (NULL = unfiled)
//
// Backward Compatibility:
// - Value field is deprecated but still supported for reading legacy secrets
// - Legacy secrets are auto-converted to Fields["value"] on read
// - SetSecret uses Fields; Value is ignored if Fields is set
type SecretEntry struct {
	Key        string            // Secret key name
	Value      []byte            // Deprecated: use Fields instead. Kept for backward compatibility.
	Fields     map[string]Field  // Multi-field values (Phase 2.5+)
	Bindings   map[string]string // Environment variable bindings: env_var_name -> field_name
	Schema     string            // Reserved for Phase 3 schema validation
	FolderID   *string           // Reference to folder (Phase 2c-X2, NULL = unfiled)
	Metadata   *SecretMetadata   // Encrypted metadata (notes, url)
	Tags       []string          // Plaintext: searchable tags
	ExpiresAt  *time.Time        // Plaintext: expiration date
	FieldCount int               // Number of fields (plaintext for MCP secret_list)
	CreatedAt  time.Time         // Creation timestamp
	UpdatedAt  time.Time         // Last update timestamp
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
	// Convert password to []byte and wipe after use to minimize memory exposure
	passwordBytes := []byte(masterPassword)
	defer crypto.SecureWipe(passwordBytes)
	kek := crypto.DeriveKey(passwordBytes, salt)
	defer crypto.SecureWipe(kek) // Wipe KEK when done

	// 3. Generate DEK (32 bytes)
	dek := make([]byte, DEKLength)
	if _, err := rand.Read(dek); err != nil {
		return fmt.Errorf("vault: failed to generate DEK: %w", err)
	}
	defer crypto.SecureWipe(dek) // Wipe DEK when Init completes (vault is not unlocked)

	// 4. Encrypt DEK using crypto.Encrypt
	encryptedDEK, nonce, err := crypto.Encrypt(kek, dek)
	if err != nil {
		return fmt.Errorf("vault: failed to encrypt DEK: %w", err)
	}

	// 5. Initialize SQLite database
	dbPath := filepath.Join(v.path, DBFileName)

	// Pre-create the file with secure permissions (0600) to prevent race condition.
	// Without this, sql.Open creates the file with default umask permissions,
	// then we chmod after, leaving a window where the file could be world-readable.
	// This follows CWE-377 mitigation: create file atomically with correct permissions.
	f, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, FileMode)
	if err != nil {
		return fmt.Errorf("vault: failed to create database file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("vault: failed to close database file: %w", err)
	}

	// Enforce correct permissions in case file already existed (pre-creation attack defense)
	// or was created with different umask. This provides defense-in-depth.
	if err := os.Chmod(dbPath, FileMode); err != nil {
		return fmt.Errorf("vault: failed to set database permissions: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("vault: failed to open database: %w", err)
	}
	defer db.Close()

	// Create tables
	if err := v.createTables(db); err != nil {
		return fmt.Errorf("vault: failed to create tables: %w", err)
	}

	// 6. Save salt, encrypted DEK and nonce to database (ADR-003: salt in DB for atomic password change)
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("vault: failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO vault_keys(salt, encrypted_dek, dek_nonce) VALUES(?, ?, ?)")
	if err != nil {
		return fmt.Errorf("vault: failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	if _, err := stmt.Exec(salt, encryptedDEK, nonce); err != nil {
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

	// 1. Open database to read salt (ADR-003: salt stored in DB for atomic password change)
	dbPath := filepath.Join(v.path, DBFileName)
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return fmt.Errorf("vault: failed to open database: %w", err)
	}

	// Configure SQLite for single-connection mode
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Try to read salt from database first (v4+ schema)
	var salt []byte
	err = db.QueryRow("SELECT salt FROM vault_keys WHERE id = 1").Scan(&salt)
	if err != nil || len(salt) == 0 {
		// Fallback to file for pre-v4 vaults (migration will happen after unlock)
		db.Close()
		saltPath := filepath.Join(v.path, SaltFileName)
		salt, err = os.ReadFile(saltPath)
		if err != nil {
			if os.IsNotExist(err) {
				return ErrSaltNotFound
			}
			return fmt.Errorf("vault: failed to read salt file: %w", err)
		}
		// Reopen database for subsequent operations
		db, err = sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)")
		if err != nil {
			return fmt.Errorf("vault: failed to reopen database: %w", err)
		}
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	}

	// Validate salt length to detect corruption/tampering
	if len(salt) != SaltLength {
		db.Close()
		return ErrVaultCorrupted
	}

	// 2. Derive KEK
	// Convert password to []byte and wipe after use to minimize memory exposure
	passwordBytes := []byte(masterPassword)
	defer crypto.SecureWipe(passwordBytes)
	kek := crypto.DeriveKey(passwordBytes, salt)
	defer crypto.SecureWipe(kek) // Wipe KEK after decrypting DEK

	// 3. Read encrypted DEK and nonce from database (db already opened in step 1)
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

	// 6. Run schema migrations if needed
	if err := migrateSchema(db, v.path); err != nil {
		v.dek = nil
		v.db = nil
		db.Close()
		return fmt.Errorf("vault: schema migration failed: %w", err)
	}

	// 7. Enable foreign keys (required for ON DELETE RESTRICT per ADR-007)
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		v.dek = nil
		v.db = nil
		db.Close()
		return fmt.Errorf("vault: failed to enable foreign keys: %w", err)
	}

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

	// Clear audit logger HMAC key to minimize sensitive material lifetime
	v.audit.ClearHMACKey()

	// Close database connection
	if v.db != nil {
		v.db.Close()
		v.db = nil
	}
}

// ChangePassword changes the master password by re-wrapping the DEK.
// The DEK itself remains unchanged, so all secrets remain accessible.
//
// Process (ADR-003):
//  1. Validate inputs, reject same password
//  2. Create backup (VACUUM INTO, 0600)
//  3. Begin transaction (mutex protects in-process, SQLite file-lock for cross-process)
//  4. Verify current password, unwrap DEK
//  5. Generate new salt/nonce, derive new KEK
//  6. Re-wrap DEK with new KEK
//  7. Verify in-memory before commit
//  8. UPDATE vault_keys, COMMIT
//  9. Record audit log, SecureWipe key material
//
// Concurrency: v.mu.Lock() serializes in-process calls; SQLite's file locking
// prevents concurrent writes from multiple processes.
//
// Crash safety: SQLite atomic commit ensures all-or-nothing.
// Before COMMIT → auto-rollback → old password works.
// After COMMIT → change complete → new password works.
func (v *Vault) ChangePassword(currentPassword, newPassword string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Step 1: Validate inputs
	// Check vault is unlocked
	if v.dek == nil {
		return ErrVaultLocked
	}

	// Convert passwords to []byte early (avoid string copies)
	currentPasswordBytes := []byte(currentPassword)
	defer crypto.SecureWipe(currentPasswordBytes)
	newPasswordBytes := []byte(newPassword)
	defer crypto.SecureWipe(newPasswordBytes)

	// Reject if same password
	if currentPassword == newPassword {
		return ErrSamePassword
	}

	// Validate new password strength
	validation := ValidateMasterPassword(newPassword)
	if !validation.Valid {
		// Password too short or too long
		if len(newPassword) < MinPasswordLength {
			return ErrPasswordTooShort
		}
		if len(newPassword) > MaxPasswordLength {
			return ErrPasswordTooLong
		}
	}

	// Step 2: Create backup (optional, for user safety)
	backupPath := filepath.Join(v.path, fmt.Sprintf("%s.backup-%d", DBFileName, time.Now().Unix()))
	// Escape single quotes in path for SQL safety
	escapedPath := strings.ReplaceAll(backupPath, "'", "''")
	_, err := v.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", escapedPath))
	if err != nil {
		return fmt.Errorf("vault: failed to create backup: %w", err)
	}
	// Set secure permissions on backup
	if err := os.Chmod(backupPath, FileMode); err != nil {
		// Non-fatal, but warn
		fmt.Fprintf(os.Stderr, "warning: failed to set backup permissions: %v\n", err)
	}

	// Step 3: Begin transaction
	// SQLite's default transaction with deferred locking is sufficient.
	// The first write operation (UPDATE vault_keys) acquires a write lock,
	// and COMMIT is atomic. If crash occurs before COMMIT, changes are rolled back.
	tx, err := v.db.Begin()
	if err != nil {
		return fmt.Errorf("vault: failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Step 4: Verify current password
	// Read current salt and encrypted DEK from database
	var currentSalt, encryptedDEK, dekNonce []byte
	err = tx.QueryRow("SELECT salt, encrypted_dek, dek_nonce FROM vault_keys WHERE id = 1").
		Scan(&currentSalt, &encryptedDEK, &dekNonce)
	if err != nil {
		return fmt.Errorf("vault: failed to read vault keys: %w", err)
	}

	// Derive old KEK and verify by unwrapping DEK
	kekOld := crypto.DeriveKey(currentPasswordBytes, currentSalt)
	defer crypto.SecureWipe(kekOld)

	dekCopy, err := crypto.Decrypt(kekOld, encryptedDEK, dekNonce)
	if err != nil {
		return ErrInvalidPassword
	}
	defer crypto.SecureWipe(dekCopy)

	// Step 5: Generate new cryptographic material
	newSalt := make([]byte, SaltLength)
	if _, err := rand.Read(newSalt); err != nil {
		return fmt.Errorf("vault: failed to generate new salt: %w", err)
	}

	// Derive new KEK
	kekNew := crypto.DeriveKey(newPasswordBytes, newSalt)
	defer crypto.SecureWipe(kekNew)

	// Step 6: Re-wrap DEK with new KEK
	encryptedDEKNew, newNonce, err := crypto.Encrypt(kekNew, dekCopy)
	if err != nil {
		return fmt.Errorf("vault: failed to re-wrap DEK: %w", err)
	}

	// Step 7: Verify before commit (in-memory)
	testDEK, err := crypto.Decrypt(kekNew, encryptedDEKNew, newNonce)
	if err != nil {
		return fmt.Errorf("vault: verification failed, DEK re-wrap corrupted: %w", err)
	}
	crypto.SecureWipe(testDEK)

	// Step 8: Atomic update + Commit
	_, err = tx.Exec("UPDATE vault_keys SET salt = ?, encrypted_dek = ?, dek_nonce = ? WHERE id = 1",
		newSalt, encryptedDEKNew, newNonce)
	if err != nil {
		return fmt.Errorf("vault: failed to update vault keys: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("vault: failed to commit password change: %w", err)
	}

	// Step 9: Post-commit actions
	// Record audit log (file-based, best-effort)
	_ = v.audit.LogSuccess(audit.OpPasswordChanged, audit.SourceCLI, "")

	// Delete backup file (optional - keep for safety during initial rollout)
	// TODO: Consider making this configurable in future versions
	// os.Remove(backupPath)

	return nil
}

// Audit returns the vault's audit logger for MCP and other external use.
func (v *Vault) Audit() *audit.Logger {
	return v.audit
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
// Per requirements-ja.md §4.1: "Warn if permissions are not 0600"
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
	// Enable foreign keys (required for ON DELETE RESTRICT per ADR-007)
	_, err := db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// vault_keys table (encrypted DEK + salt per ADR-003)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS vault_keys (
			id INTEGER PRIMARY KEY,
			salt BLOB NOT NULL,
			encrypted_dek BLOB NOT NULL,
			dek_nonce BLOB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// folders table (per ADR-007: Folder Feature)
	// - id: UUID primary key
	// - name: display name (no "/" allowed)
	// - parent_id: NULL for root, UUID for nested
	// - icon/color: optional visual customization
	// - sort_order: for manual ordering
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS folders (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			parent_id TEXT,
			icon TEXT,
			color TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (parent_id) REFERENCES folders(id) ON DELETE RESTRICT,
			CHECK (name NOT LIKE '%/%')
		)
	`)
	if err != nil {
		return err
	}

	// Folder indexes
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders(parent_id)")
	if err != nil {
		return err
	}

	// Unique index for nested folder names (case-insensitive)
	// Per ADR-007: Unique (name, parent_id) constraint with case-insensitive matching
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_folders_name_parent ON folders(name COLLATE NOCASE, parent_id) WHERE parent_id IS NOT NULL`)
	if err != nil {
		return err
	}

	// Unique index for root folder names (NULL parent)
	// SQLite requires separate index for NULL values in partial index
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_folders_root_name ON folders(name COLLATE NOCASE) WHERE parent_id IS NULL`)
	if err != nil {
		return err
	}

	// secrets table (hybrid approach: encrypted metadata JSON + plaintext search fields)
	// Per project-proposal-ja.md Phase 0: Metadata support (notes/url/tags/expires_at)
	// Per ADR-002 Phase 2.5: Multi-field secrets support
	// Per ADR-007 Phase 2c-X2: Folder support
	// - encrypted_key: encrypted key name (nonce prepended)
	// - encrypted_value: legacy single value (nonce prepended) - kept for backward compatibility
	// - encrypted_fields: encrypted JSON map of Field structs (Phase 2.5+)
	// - encrypted_bindings: encrypted JSON map of env var bindings (Phase 2.5+)
	// - encrypted_metadata: encrypted notes/url JSON
	// - schema: plaintext schema name (reserved for Phase 3)
	// - field_count: plaintext field count for MCP secret_list (Phase 2.5+)
	// - folder_id: reference to folder (Phase 2c-X2, NULL = unfiled)
	// - tags, expires_at: plaintext for searchability
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS secrets (
			id INTEGER PRIMARY KEY,
			key_hash TEXT UNIQUE NOT NULL,
			encrypted_key BLOB NOT NULL,
			encrypted_value BLOB,
			encrypted_fields BLOB,
			encrypted_bindings BLOB,
			encrypted_metadata BLOB,
			schema TEXT,
			field_count INTEGER DEFAULT 1,
			folder_id TEXT REFERENCES folders(id) ON DELETE RESTRICT,
			tags TEXT,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Secret folder index
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_secrets_folder ON secrets(folder_id)")
	if err != nil {
		return err
	}

	// schema_version table for migration tracking
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY,
			migrated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Record current schema version for new vaults
	_, err = db.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", CurrentSchemaVersion)
	if err != nil {
		return err
	}

	return nil
}

// hashKey computes HMAC-SHA256 of key name for secure lookup.
// Uses DEK as the HMAC key to prevent offline brute-force attacks on key names.
// An attacker with database access cannot dictionary-attack key names without the DEK.
func (v *Vault) hashKey(key string) string {
	mac := hmac.New(sha256.New, v.dek)
	mac.Write([]byte(key))
	return hex.EncodeToString(mac.Sum(nil))
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
//
// Multi-field support (Phase 2.5):
//   - If entry.Fields is set, it will be used (preferred)
//   - If entry.Fields is nil but entry.Value is set, Value is converted to Fields["value"]
//   - Both legacy format (encrypted_value) and new format (encrypted_fields) are stored
//     for backward compatibility during transition period
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

	// Normalize Fields: if Fields is nil but Value is set, convert to Fields["value"]
	fields := entry.Fields
	if fields == nil && len(entry.Value) > 0 {
		fields = ConvertSingleValueToFields(entry.Value)
	}

	// Validate fields (multi-field format)
	if fields != nil {
		if err := ValidateFields(fields); err != nil {
			_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "INVALID_FIELDS", err.Error())
			return err
		}
	}

	// Validate bindings if present
	if entry.Bindings != nil {
		if len(fields) == 0 {
			_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "INVALID_BINDINGS", "bindings require fields")
			return fmt.Errorf("%w: bindings require fields", ErrBindingFieldNotFound)
		}
		if err := ValidateBindings(entry.Bindings, fields); err != nil {
			_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "INVALID_BINDINGS", err.Error())
			return err
		}
	}

	// Calculate total data size for disk space check
	dataSize := len(key)
	for name, field := range fields {
		dataSize += len(name) + len(field.Value) + len(field.Kind) + len(field.InputType) + len(field.Hint)
		for _, alias := range field.Aliases {
			dataSize += len(alias)
		}
	}
	for envVar, fieldName := range entry.Bindings {
		dataSize += len(envVar) + len(fieldName)
	}
	if entry.Metadata != nil {
		dataSize += len(entry.Metadata.Notes) + len(entry.Metadata.URL)
	}

	// Validate metadata per requirements-ja.md §2.5
	if err := validateMetadata(entry.Metadata, entry.Tags, entry.ExpiresAt); err != nil {
		_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "INVALID_METADATA", err.Error())
		return err
	}

	// Check disk space before write
	if err := v.checkDiskSpaceForWrite(dataSize); err != nil {
		_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "DISK_FULL", err.Error())
		return err
	}

	// Compute key hash for lookup (HMAC-SHA256 with DEK)
	keyHash := v.hashKey(key)

	// Encrypt key name (nonce prepended)
	encryptedKey, err := v.encryptWithNonce([]byte(key))
	if err != nil {
		_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "ENCRYPT_FAILED", err.Error())
		return fmt.Errorf("vault: failed to encrypt key: %w", err)
	}

	// Encrypt legacy value for backward compatibility
	// Use the "value" field if it exists, otherwise nil
	var encryptedValue []byte
	if defaultValue := GetDefaultFieldValue(fields); defaultValue != "" {
		encryptedValue, err = v.encryptWithNonce([]byte(defaultValue))
		if err != nil {
			_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "ENCRYPT_FAILED", err.Error())
			return fmt.Errorf("vault: failed to encrypt value: %w", err)
		}
	}

	// Encrypt fields as JSON blob (nonce prepended)
	var encryptedFields []byte
	if len(fields) > 0 {
		fieldsJSON, err := json.Marshal(fields)
		if err != nil {
			_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "MARSHAL_FAILED", err.Error())
			return fmt.Errorf("vault: failed to marshal fields: %w", err)
		}
		encryptedFields, err = v.encryptWithNonce(fieldsJSON)
		if err != nil {
			_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "ENCRYPT_FAILED", "fields: "+err.Error())
			return fmt.Errorf("vault: failed to encrypt fields: %w", err)
		}
	}

	// Encrypt bindings as JSON blob (nonce prepended)
	var encryptedBindings []byte
	if len(entry.Bindings) > 0 {
		bindingsJSON, err := json.Marshal(entry.Bindings)
		if err != nil {
			_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "MARSHAL_FAILED", err.Error())
			return fmt.Errorf("vault: failed to marshal bindings: %w", err)
		}
		encryptedBindings, err = v.encryptWithNonce(bindingsJSON)
		if err != nil {
			_ = v.audit.LogError(audit.OpSecretSet, audit.SourceCLI, key, "ENCRYPT_FAILED", "bindings: "+err.Error())
			return fmt.Errorf("vault: failed to encrypt bindings: %w", err)
		}
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

	// Calculate field count for MCP secret_list (plaintext, not sensitive)
	fieldCount := 1 // Default for legacy single-value secrets
	if len(entry.Fields) > 0 {
		fieldCount = len(entry.Fields)
	} else if len(entry.Value) == 0 {
		fieldCount = 0
	}

	// Begin transaction
	tx, err := v.db.Begin()
	if err != nil {
		return fmt.Errorf("vault: failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// UPSERT: update if key exists, insert otherwise
	// Store both legacy format (encrypted_value) and new format (encrypted_fields)
	// Per ADR-007: folder_id is stored as plaintext reference to folders table
	_, err = tx.Exec(`
		INSERT INTO secrets (key_hash, encrypted_key, encrypted_value, encrypted_fields, encrypted_bindings, encrypted_metadata, schema, field_count, folder_id, tags, expires_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key_hash) DO UPDATE SET
			encrypted_key = excluded.encrypted_key,
			encrypted_value = excluded.encrypted_value,
			encrypted_fields = excluded.encrypted_fields,
			encrypted_bindings = excluded.encrypted_bindings,
			encrypted_metadata = excluded.encrypted_metadata,
			schema = excluded.schema,
			field_count = excluded.field_count,
			folder_id = excluded.folder_id,
			tags = excluded.tags,
			expires_at = excluded.expires_at,
			updated_at = CURRENT_TIMESTAMP
	`, keyHash, encryptedKey, encryptedValue, encryptedFields, encryptedBindings, encryptedMetadata, entry.Schema, fieldCount, entry.FolderID, tagsStr, expiresAt)
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
//
// Multi-field support (Phase 2.5):
// - Prefers encrypted_fields if present (new format)
// - Falls back to encrypted_value if encrypted_fields is NULL (legacy format)
// - Legacy data is auto-converted to Fields["value"]
// - Value field is populated for backward compatibility
func (v *Vault) GetSecret(key string) (*SecretEntry, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Check if vault is locked
	if v.dek == nil {
		return nil, ErrVaultLocked
	}

	// Compute key hash (HMAC-SHA256 with DEK)
	keyHash := v.hashKey(key)

	// Get all fields from database (including new multi-field columns)
	var encryptedValue, encryptedFields, encryptedBindings, encryptedMetadata []byte
	var schema sql.NullString
	var folderID sql.NullString
	var tagsStr sql.NullString
	var expiresAt sql.NullTime
	var createdAt, updatedAt time.Time

	err := v.db.QueryRow(`
		SELECT encrypted_value, encrypted_fields, encrypted_bindings, encrypted_metadata, schema, folder_id, tags, expires_at, created_at, updated_at
		FROM secrets WHERE key_hash = ?`,
		keyHash,
	).Scan(&encryptedValue, &encryptedFields, &encryptedBindings, &encryptedMetadata, &schema, &folderID, &tagsStr, &expiresAt, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = v.audit.LogError(audit.OpSecretGet, audit.SourceCLI, key, "NOT_FOUND", "secret not found")
			return nil, ErrSecretNotFound
		}
		return nil, fmt.Errorf("vault: failed to read secret: %w", err)
	}

	entry := &SecretEntry{
		Key:       key,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	// Set folder ID if present (Phase 2c-X2)
	if folderID.Valid {
		entry.FolderID = &folderID.String
	}

	// Decrypt fields (new format) or value (legacy format)
	if len(encryptedFields) > 0 {
		// New multi-field format
		fieldsJSON, err := v.decryptWithNonce(encryptedFields)
		if err != nil {
			_ = v.audit.LogError(audit.OpSecretGet, audit.SourceCLI, key, "DECRYPT_FAILED", err.Error())
			return nil, fmt.Errorf("vault: failed to decrypt fields: %w", err)
		}
		var fields map[string]Field
		if err := json.Unmarshal(fieldsJSON, &fields); err != nil {
			return nil, fmt.Errorf("vault: failed to unmarshal fields: %w", err)
		}
		entry.Fields = fields
		// Populate legacy Value field for backward compatibility
		entry.Value = []byte(GetDefaultFieldValue(fields))
	} else if len(encryptedValue) > 0 {
		// Legacy single-value format - auto-convert to Fields["value"]
		plainValue, err := v.decryptWithNonce(encryptedValue)
		if err != nil {
			_ = v.audit.LogError(audit.OpSecretGet, audit.SourceCLI, key, "DECRYPT_FAILED", err.Error())
			return nil, fmt.Errorf("vault: failed to decrypt secret: %w", err)
		}
		entry.Value = plainValue
		entry.Fields = ConvertSingleValueToFields(plainValue)
	}

	// Decrypt bindings if present
	if len(encryptedBindings) > 0 {
		bindingsJSON, err := v.decryptWithNonce(encryptedBindings)
		if err != nil {
			return nil, fmt.Errorf("vault: failed to decrypt bindings: %w", err)
		}
		var bindings map[string]string
		if err := json.Unmarshal(bindingsJSON, &bindings); err != nil {
			return nil, fmt.Errorf("vault: failed to unmarshal bindings: %w", err)
		}
		entry.Bindings = bindings
	}

	// Set schema if present
	if schema.Valid {
		entry.Schema = schema.String
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

	// Compute key hash (HMAC-SHA256 with DEK)
	keyHash := v.hashKey(key)

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

// scanSecretEntryRowWithMetadata scans a row including metadata (but NOT secret value).
// This decrypts key name and metadata, but never touches the secret value.
// Use this for list operations that need metadata presence (HasNotes, HasURL).
//
// Multi-field support (Phase 2.5):
// - Also reads schema column for Phase 3 compatibility
// - Does NOT decrypt fields/bindings (not needed for list operations)
//
// Folder support (Phase 2c-X2):
// - Reads folder_id for folder-based filtering and display
func (v *Vault) scanSecretEntryRowWithMetadata(rows *sql.Rows) (*SecretEntry, error) {
	var encryptedKey []byte
	var encryptedMetadata []byte
	var schema sql.NullString
	var fieldCount sql.NullInt64
	var folderID sql.NullString
	var tagsStr sql.NullString
	var expiresAt sql.NullTime
	var createdAt, updatedAt time.Time

	if err := rows.Scan(&encryptedKey, &encryptedMetadata, &schema, &fieldCount, &folderID, &tagsStr, &expiresAt,
		&createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("vault: failed to scan row: %w", err)
	}

	// Decrypt key name (nonce is prepended)
	keyBytes, err := v.decryptWithNonce(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to decrypt key name: %w", err)
	}

	entry := &SecretEntry{
		Key:        string(keyBytes),
		FieldCount: 1, // Default for legacy secrets
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}

	// Set field count if present (Phase 2.5+)
	if fieldCount.Valid {
		entry.FieldCount = int(fieldCount.Int64)
	}

	// Set schema if present
	if schema.Valid {
		entry.Schema = schema.String
	}

	// Set folder ID if present (Phase 2c-X2)
	if folderID.Valid {
		entry.FolderID = &folderID.String
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
		SELECT encrypted_key, encrypted_metadata, schema, field_count, folder_id, tags, expires_at, created_at, updated_at
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
	// Include encrypted_metadata and schema for HasNotes/HasURL support and Phase 3
	rows, err := v.db.Query(`
		SELECT encrypted_key, encrypted_metadata, schema, field_count, folder_id, tags, expires_at, created_at, updated_at
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

	// Include encrypted_metadata and schema for HasNotes/HasURL support and Phase 3
	rows, err := v.db.Query(`
		SELECT encrypted_key, encrypted_metadata, schema, field_count, folder_id, tags, expires_at, created_at, updated_at
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
	db, err := sql.Open("sqlite", dbPath)
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

// CheckDiskSpace is defined in platform-specific files:
// - vault_unix.go: Unix/Linux/macOS implementation using syscall.Statfs
// - vault_windows.go: Windows implementation using GetDiskFreeSpaceEx

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
	db, err := sql.Open("sqlite", dbPath)
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
