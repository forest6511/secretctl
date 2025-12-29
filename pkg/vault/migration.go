// Package vault provides secure secret storage with AES-256-GCM encryption.
package vault

import (
	"database/sql"
	"fmt"
)

// Schema version constants
const (
	// SchemaVersion1 is the original single-value schema
	SchemaVersion1 = 1
	// SchemaVersion2 adds multi-field support (Phase 2.5)
	SchemaVersion2 = 2
	// CurrentSchemaVersion is the current schema version
	CurrentSchemaVersion = SchemaVersion2
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
func migrateSchema(db *sql.DB) error {
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
