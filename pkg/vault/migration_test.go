// Package vault provides secure secret storage with AES-256-GCM encryption.
package vault

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestGetSchemaVersion(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create a bare database without schema_version table
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Test: No schema_version table should return version 1
	version, err := getSchemaVersion(db)
	if err != nil {
		t.Fatalf("getSchemaVersion failed: %v", err)
	}
	if version != SchemaVersion1 {
		t.Errorf("expected version %d, got %d", SchemaVersion1, version)
	}
}

func TestSetSchemaVersion(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Set version 2
	if err := setSchemaVersion(db, SchemaVersion2); err != nil {
		t.Fatalf("setSchemaVersion failed: %v", err)
	}

	// Verify version was set
	version, err := getSchemaVersion(db)
	if err != nil {
		t.Fatalf("getSchemaVersion failed: %v", err)
	}
	if version != SchemaVersion2 {
		t.Errorf("expected version %d, got %d", SchemaVersion2, version)
	}

	// Update version
	if err := setSchemaVersion(db, SchemaVersion2+1); err != nil {
		t.Fatalf("setSchemaVersion update failed: %v", err)
	}

	version, err = getSchemaVersion(db)
	if err != nil {
		t.Fatalf("getSchemaVersion after update failed: %v", err)
	}
	if version != SchemaVersion2+1 {
		t.Errorf("expected version %d, got %d", SchemaVersion2+1, version)
	}
}

func TestMigrateToV2(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create v1 schema (without multi-field columns)
	_, err = db.Exec(`
		CREATE TABLE secrets (
			key_hash TEXT PRIMARY KEY,
			encrypted_value BLOB NOT NULL,
			encrypted_key BLOB NOT NULL,
			encrypted_metadata BLOB,
			encrypted_tags BLOB,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create v1 schema: %v", err)
	}

	// Add some test data
	_, err = db.Exec(`
		INSERT INTO secrets (key_hash, encrypted_value, encrypted_key)
		VALUES ('hash1', X'0102030405', X'0102030405')
	`)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Run migration
	if err := migrateToV2(db); err != nil {
		t.Fatalf("migrateToV2 failed: %v", err)
	}

	// Verify new columns exist
	columns, err := getTableColumnsFromDB(db, "secrets")
	if err != nil {
		t.Fatalf("failed to get columns: %v", err)
	}

	expectedColumns := []string{"encrypted_fields", "encrypted_bindings", "schema"}
	for _, col := range expectedColumns {
		if !columns[col] {
			t.Errorf("missing column after migration: %s", col)
		}
	}

	// Verify schema version
	version, err := getSchemaVersion(db)
	if err != nil {
		t.Fatalf("getSchemaVersion failed: %v", err)
	}
	if version != SchemaVersion2 {
		t.Errorf("expected schema version %d, got %d", SchemaVersion2, version)
	}

	// Run migration again (should be idempotent)
	if err := migrateToV2(db); err != nil {
		t.Fatalf("migrateToV2 (idempotent) failed: %v", err)
	}
}

func TestMigrateSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create v1 schema
	_, err = db.Exec(`
		CREATE TABLE secrets (
			key_hash TEXT PRIMARY KEY,
			encrypted_value BLOB NOT NULL,
			encrypted_key BLOB NOT NULL,
			encrypted_metadata BLOB,
			encrypted_tags BLOB,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create v1 schema: %v", err)
	}

	// Run full migration
	if err := migrateSchema(db); err != nil {
		t.Fatalf("migrateSchema failed: %v", err)
	}

	// Verify we're at current version
	version, err := getSchemaVersion(db)
	if err != nil {
		t.Fatalf("getSchemaVersion failed: %v", err)
	}
	if version != CurrentSchemaVersion {
		t.Errorf("expected current schema version %d, got %d", CurrentSchemaVersion, version)
	}
}

func TestGetTableColumns(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY,
			name TEXT,
			value BLOB
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	// Get columns using transaction
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	columns, err := getTableColumns(tx, "test_table")
	if err != nil {
		t.Fatalf("getTableColumns failed: %v", err)
	}

	expected := []string{"id", "name", "value"}
	for _, col := range expected {
		if !columns[col] {
			t.Errorf("missing column: %s", col)
		}
	}
	if len(columns) != len(expected) {
		t.Errorf("expected %d columns, got %d", len(expected), len(columns))
	}
}

// Helper function for testing - uses db.Query directly instead of transaction
func getTableColumnsFromDB(db *sql.DB, tableName string) (map[string]bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return getTableColumns(tx, tableName)
}

func TestVaultSchemaMigrationOnUnlock(t *testing.T) {
	tmpDir := t.TempDir()

	// Set secure permissions for the vault directory
	if err := os.Chmod(tmpDir, 0700); err != nil {
		t.Fatalf("failed to set directory permissions: %v", err)
	}

	v := New(tmpDir)
	password := "testpassword123"

	// Initialize vault (creates v2 schema directly)
	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Unlock - should run migration check
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	defer v.Lock()

	// Verify vault is functional
	entry := &SecretEntry{
		Fields: map[string]Field{
			"test_field": {Value: "test_value", Sensitive: true},
		},
	}
	if err := v.SetSecret("test/migration", entry); err != nil {
		t.Fatalf("SetSecret failed: %v", err)
	}

	retrieved, err := v.GetSecret("test/migration")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}

	if retrieved.Fields["test_field"].Value != "test_value" {
		t.Errorf("value mismatch after migration check")
	}
}
