// Package vault provides secure secret storage with AES-256-GCM encryption.
package vault

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// Schema version constants
const (
	// SchemaVersion1 is the original single-value schema
	SchemaVersion1 = 1
	// SchemaVersion2 adds multi-field support (Phase 2.5)
	SchemaVersion2 = 2
	// SchemaVersion3 adds field_count column for MCP secret_list
	SchemaVersion3 = 3
	// SchemaVersion4 adds salt column to vault_keys (Phase 2c-P: Password Change)
	SchemaVersion4 = 4
	// SchemaVersion5 adds folders table and folder_id column (Phase 2c-X2: Folder Feature)
	SchemaVersion5 = 5
	// CurrentSchemaVersion is the current schema version
	CurrentSchemaVersion = SchemaVersion5
)

// getSchemaVersion returns the current schema version from the database.
// Returns 1 if no version is stored (legacy database).
func getSchemaVersion(db *sql.DB) (int, error) {
	// Check if schema_version table exists
	var tableName string
	err := db.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name='schema_version'
	`).Scan(&tableName)

	if err == sql.ErrNoRows {
		// No schema_version table = version 1 (legacy)
		return SchemaVersion1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("vault: failed to check schema_version table: %w", err)
	}

	// Get the version
	var version int
	err = db.QueryRow("SELECT version FROM schema_version ORDER BY version DESC LIMIT 1").Scan(&version)
	if err == sql.ErrNoRows {
		return SchemaVersion1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("vault: failed to get schema version: %w", err)
	}

	return version, nil
}

// setSchemaVersion sets the schema version in the database.
func setSchemaVersion(db *sql.DB, version int) error {
	// Create schema_version table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY,
			migrated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("vault: failed to create schema_version table: %w", err)
	}

	// Insert the version
	_, err = db.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", version)
	if err != nil {
		return fmt.Errorf("vault: failed to set schema version: %w", err)
	}

	return nil
}

// migrateSchema migrates the database schema to the current version.
// vaultPath is needed for v4 migration to read salt file.
func migrateSchema(db *sql.DB, vaultPath string) error {
	version, err := getSchemaVersion(db)
	if err != nil {
		return err
	}

	// Apply migrations in order
	if version < SchemaVersion2 {
		if err := migrateToV2(db); err != nil {
			return fmt.Errorf("vault: migration to v2 failed: %w", err)
		}
	}

	if version < SchemaVersion3 {
		if err := migrateToV3(db); err != nil {
			return fmt.Errorf("vault: migration to v3 failed: %w", err)
		}
	}

	if version < SchemaVersion4 {
		if err := migrateToV4(db, vaultPath); err != nil {
			return fmt.Errorf("vault: migration to v4 failed: %w", err)
		}
	}

	if version < SchemaVersion5 {
		if err := migrateToV5(db); err != nil {
			return fmt.Errorf("vault: migration to v5 failed: %w", err)
		}
	}

	return nil
}

// migrateToV2 adds multi-field support columns.
// This migration:
// 1. Adds encrypted_fields column (BLOB, nullable for backward compat)
// 2. Adds encrypted_bindings column (BLOB, nullable)
// 3. Adds schema column (TEXT, nullable, for Phase 3)
// 4. Updates schema version
//
// Note: Existing data is NOT migrated during schema migration.
// Conversion happens on-the-fly during read operations:
// - If encrypted_fields is NULL and encrypted_value is not, auto-convert to Fields["value"]
// - This lazy migration approach minimizes downtime and risk.
func migrateToV2(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if columns already exist (idempotent migration)
	columns, err := getTableColumns(tx, "secrets")
	if err != nil {
		return fmt.Errorf("failed to get table columns: %w", err)
	}

	// Add encrypted_fields column if it doesn't exist
	if !columns["encrypted_fields"] {
		_, err = tx.Exec("ALTER TABLE secrets ADD COLUMN encrypted_fields BLOB")
		if err != nil {
			return fmt.Errorf("failed to add encrypted_fields column: %w", err)
		}
	}

	// Add encrypted_bindings column if it doesn't exist
	if !columns["encrypted_bindings"] {
		_, err = tx.Exec("ALTER TABLE secrets ADD COLUMN encrypted_bindings BLOB")
		if err != nil {
			return fmt.Errorf("failed to add encrypted_bindings column: %w", err)
		}
	}

	// Add schema column if it doesn't exist
	if !columns["schema"] {
		_, err = tx.Exec("ALTER TABLE secrets ADD COLUMN schema TEXT")
		if err != nil {
			return fmt.Errorf("failed to add schema column: %w", err)
		}
	}

	// Update schema version
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY,
			migrated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_version table: %w", err)
	}

	_, err = tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", SchemaVersion2)
	if err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

// getTableColumns returns a map of column names for a table.
func getTableColumns(tx *sql.Tx, tableName string) (map[string]bool, error) {
	rows, err := tx.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		columns[name] = true
	}

	return columns, rows.Err()
}

// migrateToV3 adds field_count column for MCP secret_list.
// This migration:
// 1. Adds field_count column (INTEGER, default 1 for legacy single-value secrets)
// 2. Updates schema version
//
// The field_count is stored in plaintext (not encrypted) as it's not sensitive
// and allows efficient querying without decryption (AI-Safe Access compliant).
//
// Note: Multi-field secrets created on v2 schema will have incorrect field_count=1
// until re-saved. This is acceptable because:
// - Phase 2.5 (multi-field) ships together with v3 schema in v0.7.0
// - No production data with multi-field secrets exists before this release
// - Re-saving any secret will recalculate the correct field_count
func migrateToV3(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if columns already exist (idempotent migration)
	columns, err := getTableColumns(tx, "secrets")
	if err != nil {
		return fmt.Errorf("failed to get table columns: %w", err)
	}

	// Add field_count column if it doesn't exist
	// Default to 1 for existing secrets (legacy single-value format)
	if !columns["field_count"] {
		_, err = tx.Exec("ALTER TABLE secrets ADD COLUMN field_count INTEGER DEFAULT 1")
		if err != nil {
			return fmt.Errorf("failed to add field_count column: %w", err)
		}
	}

	// Update schema version
	_, err = tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", SchemaVersion3)
	if err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

// migrateToV4 adds salt column to vault_keys for atomic password change.
// This migration (ADR-003):
// 1. Adds salt column to vault_keys table (BLOB, NOT NULL with default empty for migration)
// 2. Reads salt from vault.salt file
// 3. Stores salt in vault_keys table
// 4. Updates schema version
//
// The salt is moved from file to database to enable atomic password change.
// Old vault.salt file is NOT deleted for backward compatibility during transition.
// Future versions may remove the file after successful migration verification.
func migrateToV4(db *sql.DB, vaultPath string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if salt column already exists (idempotent migration)
	columns, err := getTableColumnsFromDB(db, "vault_keys")
	if err != nil {
		return fmt.Errorf("failed to get vault_keys columns: %w", err)
	}

	// Add salt column if it doesn't exist
	if !columns["salt"] {
		// Add column with empty default first (SQLite requires default for NOT NULL on existing rows)
		_, err = tx.Exec("ALTER TABLE vault_keys ADD COLUMN salt BLOB NOT NULL DEFAULT ''")
		if err != nil {
			return fmt.Errorf("failed to add salt column: %w", err)
		}

		// Read salt from file
		saltPath := filepath.Join(vaultPath, SaltFileName)
		salt, err := os.ReadFile(saltPath)
		if err != nil {
			return fmt.Errorf("failed to read salt file: %w", err)
		}

		// Validate salt length
		if len(salt) != SaltLength {
			return fmt.Errorf("invalid salt length: expected %d, got %d", SaltLength, len(salt))
		}

		// Update vault_keys with the salt
		_, err = tx.Exec("UPDATE vault_keys SET salt = ? WHERE id = 1", salt)
		if err != nil {
			return fmt.Errorf("failed to store salt in database: %w", err)
		}
	}

	// Update schema version
	_, err = tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", SchemaVersion4)
	if err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

// migrateToV5 adds folders table and folder_id column for folder feature (ADR-007).
// This migration:
// 1. Creates folders table with hierarchical structure
// 2. Adds folder_id column to secrets table
// 3. Creates indexes for efficient queries
// 4. Updates schema version
//
// Note: Existing secrets remain "unfiled" (folder_id = NULL) after migration.
func migrateToV5(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Enable foreign keys for this connection (required for ON DELETE RESTRICT)
	_, err = tx.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Check if folders table already exists (idempotent migration)
	var tableName string
	err = tx.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name='folders'
	`).Scan(&tableName)

	if err == sql.ErrNoRows {
		// Create folders table per ADR-007
		_, err = tx.Exec(`
			CREATE TABLE folders (
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
			return fmt.Errorf("failed to create folders table: %w", err)
		}

		// Create index for parent lookups
		_, err = tx.Exec("CREATE INDEX idx_folders_parent ON folders(parent_id)")
		if err != nil {
			return fmt.Errorf("failed to create idx_folders_parent: %w", err)
		}

		// Create unique index for name within same parent (non-NULL parent)
		_, err = tx.Exec(`
			CREATE UNIQUE INDEX idx_folders_name_parent ON folders(name COLLATE NOCASE, parent_id)
			WHERE parent_id IS NOT NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to create idx_folders_name_parent: %w", err)
		}

		// Create unique index for root folder names (NULL parent)
		_, err = tx.Exec(`
			CREATE UNIQUE INDEX idx_folders_root_name ON folders(name COLLATE NOCASE)
			WHERE parent_id IS NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to create idx_folders_root_name: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check folders table: %w", err)
	}

	// Check if folder_id column already exists in secrets
	columns, err := getTableColumns(tx, "secrets")
	if err != nil {
		return fmt.Errorf("failed to get secrets columns: %w", err)
	}

	// Add folder_id column if it doesn't exist
	if !columns["folder_id"] {
		// Note: SQLite doesn't support adding foreign key constraints via ALTER TABLE
		// The constraint is enforced at application level for existing tables
		_, err = tx.Exec("ALTER TABLE secrets ADD COLUMN folder_id TEXT")
		if err != nil {
			return fmt.Errorf("failed to add folder_id column: %w", err)
		}

		// Create index for folder lookups
		_, err = tx.Exec("CREATE INDEX idx_secrets_folder ON secrets(folder_id)")
		if err != nil {
			return fmt.Errorf("failed to create idx_secrets_folder: %w", err)
		}
	}

	// Update schema version
	_, err = tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", SchemaVersion5)
	if err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

// getTableColumnsFromDB returns a map of column names for a table using db connection.
// Unlike getTableColumns, this uses *sql.DB instead of *sql.Tx.
func getTableColumnsFromDB(db *sql.DB, tableName string) (map[string]bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		columns[name] = true
	}

	return columns, rows.Err()
}
