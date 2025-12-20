// Package vault provides secure secret storage with AES-256-GCM encryption.
package vault

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)

	if v == nil {
		t.Fatal("New returned nil")
	}
	if v.path != tmpDir {
		t.Errorf("expected path %s, got %s", tmpDir, v.path)
	}
	if v.audit == nil {
		t.Error("expected audit logger to be initialized")
	}
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)

	err := v.Init("testpassword123")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify files were created
	saltPath := filepath.Join(tmpDir, SaltFileName)
	if _, err := os.Stat(saltPath); err != nil {
		t.Errorf("salt file not created: %v", err)
	}

	metaPath := filepath.Join(tmpDir, MetaFileName)
	if _, err := os.Stat(metaPath); err != nil {
		t.Errorf("meta file not created: %v", err)
	}

	dbPath := filepath.Join(tmpDir, DBFileName)
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("database file not created: %v", err)
	}

	// Try to init again - should fail
	err = v.Init("anotherpassword")
	if err != ErrVaultAlreadyExists {
		t.Errorf("expected ErrVaultAlreadyExists, got %v", err)
	}
}

func TestUnlockLock(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Lock (should be no-op since not unlocked)
	v.Lock()

	// Unlock with correct password
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	if v.IsLocked() {
		t.Error("expected vault to be unlocked")
	}

	// Try to unlock again - should fail
	err := v.Unlock(password)
	if err != ErrVaultAlreadyUnlocked {
		t.Errorf("expected ErrVaultAlreadyUnlocked, got %v", err)
	}

	// Lock
	v.Lock()
	if !v.IsLocked() {
		t.Error("expected vault to be locked")
	}

	// Unlock with wrong password
	err = v.Unlock("wrongpassword")
	if err != ErrInvalidPassword {
		t.Errorf("expected ErrInvalidPassword, got %v", err)
	}
}

func TestSecretOperations(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	defer v.Lock()

	// Set secret
	key := "test-key"
	value := []byte("test-value")
	entry := &SecretEntry{Value: value}
	if err := v.SetSecret(key, entry); err != nil {
		t.Fatalf("SetSecret failed: %v", err)
	}

	// Get secret
	retrieved, err := v.GetSecret(key)
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if string(retrieved.Value) != string(value) {
		t.Errorf("expected %s, got %s", string(value), string(retrieved.Value))
	}

	// List secrets
	keys, err := v.ListSecrets()
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}
	if len(keys) != 1 || keys[0] != key {
		t.Errorf("expected [%s], got %v", key, keys)
	}

	// Update secret
	newValue := []byte("updated-value")
	newEntry := &SecretEntry{Value: newValue}
	if err := v.SetSecret(key, newEntry); err != nil {
		t.Fatalf("SetSecret (update) failed: %v", err)
	}

	retrieved, err = v.GetSecret(key)
	if err != nil {
		t.Fatalf("GetSecret after update failed: %v", err)
	}
	if string(retrieved.Value) != string(newValue) {
		t.Errorf("expected %s, got %s", string(newValue), string(retrieved.Value))
	}

	// Delete secret
	if err := v.DeleteSecret(key); err != nil {
		t.Fatalf("DeleteSecret failed: %v", err)
	}

	// Verify deleted
	_, err = v.GetSecret(key)
	if err != ErrSecretNotFound {
		t.Errorf("expected ErrSecretNotFound, got %v", err)
	}
}

func TestSecretOperationsWhileLocked(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Try operations while locked
	err := v.SetSecret("key", &SecretEntry{Value: []byte("value")})
	if err != ErrVaultLocked {
		t.Errorf("SetSecret expected ErrVaultLocked, got %v", err)
	}

	_, err = v.GetSecret("key")
	if err != ErrVaultLocked {
		t.Errorf("GetSecret expected ErrVaultLocked, got %v", err)
	}

	_, err = v.ListSecrets()
	if err != ErrVaultLocked {
		t.Errorf("ListSecrets expected ErrVaultLocked, got %v", err)
	}

	err = v.DeleteSecret("key")
	if err != ErrVaultLocked {
		t.Errorf("DeleteSecret expected ErrVaultLocked, got %v", err)
	}
}

func TestCheckIntegrity(t *testing.T) {
	tmpDir := t.TempDir()
	// Set secure permissions on the temp directory
	if err := os.Chmod(tmpDir, 0700); err != nil {
		t.Fatalf("failed to set directory permissions: %v", err)
	}
	v := New(tmpDir)
	password := "testpassword123"

	// Check integrity on non-existent vault
	result, err := v.CheckIntegrity()
	if err != nil {
		t.Fatalf("CheckIntegrity failed: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result for non-existent vault")
	}

	// Initialize vault
	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Check integrity on valid vault
	result, err = v.CheckIntegrity()
	if err != nil {
		t.Fatalf("CheckIntegrity failed: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid result, got errors: %v", result.Errors)
	}
	if !result.SaltExists {
		t.Error("expected SaltExists to be true")
	}
	if !result.MetaValid {
		t.Error("expected MetaValid to be true")
	}
	if !result.DBExists {
		t.Error("expected DBExists to be true")
	}
	if !result.DBIntegrity {
		t.Error("expected DBIntegrity to be true")
	}
}

func TestCheckIntegrityWithCorruption(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Corrupt salt file
	saltPath := filepath.Join(tmpDir, SaltFileName)
	if err := os.WriteFile(saltPath, []byte("short"), FileMode); err != nil {
		t.Fatalf("failed to corrupt salt: %v", err)
	}

	result, err := v.CheckIntegrity()
	if err != nil {
		t.Fatalf("CheckIntegrity failed: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result for corrupted salt")
	}
	if !result.SaltExists {
		t.Error("expected SaltExists to be true (file exists but wrong size)")
	}
}

func TestCheckIntegrityMissingMeta(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Remove metadata file
	metaPath := filepath.Join(tmpDir, MetaFileName)
	if err := os.Remove(metaPath); err != nil {
		t.Fatalf("failed to remove meta: %v", err)
	}

	result, err := v.CheckIntegrity()
	if err != nil {
		t.Fatalf("CheckIntegrity failed: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result for missing metadata")
	}
	if result.MetaValid {
		t.Error("expected MetaValid to be false")
	}
}

func TestRepair(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Remove metadata file
	metaPath := filepath.Join(tmpDir, MetaFileName)
	if err := os.Remove(metaPath); err != nil {
		t.Fatalf("failed to remove meta: %v", err)
	}

	// Verify it's missing
	result, _ := v.CheckIntegrity()
	if result.MetaValid {
		t.Error("expected MetaValid to be false before repair")
	}

	// Repair
	if err := v.Repair(); err != nil {
		t.Fatalf("Repair failed: %v", err)
	}

	// Verify it's fixed
	result, _ = v.CheckIntegrity()
	if !result.MetaValid {
		t.Error("expected MetaValid to be true after repair")
	}
}

func TestMultipleSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	defer v.Lock()

	// Add multiple secrets
	secrets := map[string]string{
		"api-key":      "sk-1234567890",
		"db-password":  "postgres123",
		"jwt-secret":   "supersecretkey",
		"webhook-url":  "https://example.com/webhook",
		"config-value": "some-config-data",
	}

	for key, value := range secrets {
		if err := v.SetSecret(key, &SecretEntry{Value: []byte(value)}); err != nil {
			t.Fatalf("SetSecret(%s) failed: %v", key, err)
		}
	}

	// Retrieve and verify all secrets
	for key, expected := range secrets {
		entry, err := v.GetSecret(key)
		if err != nil {
			t.Fatalf("GetSecret(%s) failed: %v", key, err)
		}
		if string(entry.Value) != expected {
			t.Errorf("GetSecret(%s) expected %s, got %s", key, expected, string(entry.Value))
		}
	}

	// List all keys
	keys, err := v.ListSecrets()
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}
	if len(keys) != len(secrets) {
		t.Errorf("expected %d keys, got %d", len(secrets), len(keys))
	}
}

func TestPath(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)

	if v.Path() != tmpDir {
		t.Errorf("expected path %s, got %s", tmpDir, v.Path())
	}
}

func TestFailedAttemptTracking(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// First few failed attempts should just return ErrInvalidPassword
	for i := 0; i < CooldownThreshold1-1; i++ {
		err := v.Unlock("wrongpassword")
		if err != ErrInvalidPassword {
			t.Errorf("attempt %d: expected ErrInvalidPassword, got %v", i+1, err)
		}
	}

	// Check lock state
	state, err := v.GetLockState()
	if err != nil {
		t.Fatalf("GetLockState failed: %v", err)
	}
	if state.FailedAttempts != CooldownThreshold1-1 {
		t.Errorf("expected %d failed attempts, got %d", CooldownThreshold1-1, state.FailedAttempts)
	}

	// The 5th attempt should trigger cooldown (30 seconds per spec)
	err = v.Unlock("wrongpassword")
	if err == nil {
		t.Error("expected error on 5th failed attempt")
	}
	// Check error contains ErrTooManyAttempts
	if err != nil && err.Error() == ErrInvalidPassword.Error() {
		t.Error("5th attempt should trigger cooldown, not just invalid password")
	}

	// Verify cooldown is active (should be ~30 seconds per spec)
	remaining := v.RemainingCooldown()
	if remaining <= 0 {
		t.Error("expected positive remaining cooldown")
	}
	if remaining > 31*1000*1000*1000 { // 31 seconds in nanoseconds
		t.Errorf("expected cooldown <= 30s, got %v", remaining)
	}
}

func TestCooldownBlocksUnlock(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Trigger cooldown by failing 5 times (CooldownThreshold1)
	for i := 0; i < CooldownThreshold1; i++ {
		_ = v.Unlock("wrongpassword")
	}

	// Now even correct password should be blocked
	err := v.Unlock(password)
	if err == nil {
		t.Error("expected error during cooldown")
	}
	// Error should mention cooldown
	if err != nil && !containsError(err, ErrCooldownActive) {
		t.Errorf("expected ErrCooldownActive, got %v", err)
	}
}

func TestSuccessfulUnlockClearsLockState(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Make a few failed attempts (but not enough to trigger cooldown)
	for i := 0; i < CooldownThreshold1-2; i++ {
		_ = v.Unlock("wrongpassword")
	}

	// Verify failed attempts recorded
	state, _ := v.GetLockState()
	if state.FailedAttempts == 0 {
		t.Error("expected some failed attempts recorded")
	}

	// Successful unlock
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Unlock with correct password failed: %v", err)
	}
	v.Lock()

	// Lock state should be cleared
	state, _ = v.GetLockState()
	if state.FailedAttempts != 0 {
		t.Errorf("expected failed attempts to be cleared, got %d", state.FailedAttempts)
	}
}

// containsError checks if an error wraps a specific error
func containsError(err, target error) bool {
	if err == nil {
		return target == nil
	}
	return err.Error() == target.Error() ||
		(len(err.Error()) > len(target.Error()) &&
			err.Error()[:len(target.Error())] == target.Error())
}

func TestCheckDiskSpace(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)

	info, err := v.CheckDiskSpace()
	if err != nil {
		t.Fatalf("CheckDiskSpace failed: %v", err)
	}

	if info.Total == 0 {
		t.Error("expected non-zero total disk space")
	}
	if info.Available == 0 {
		t.Error("expected non-zero available disk space")
	}
	if info.UsedPct < 0 || info.UsedPct > 100 {
		t.Errorf("expected UsedPct between 0-100, got %d", info.UsedPct)
	}

	t.Logf("Disk space: Total=%d MB, Free=%d MB, Available=%d MB, Used=%d%%",
		info.Total/(1024*1024),
		info.Free/(1024*1024),
		info.Available/(1024*1024),
		info.UsedPct)
}

func TestHasSufficientDiskSpace(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)

	sufficient, err := v.HasSufficientDiskSpace()
	if err != nil {
		t.Fatalf("HasSufficientDiskSpace failed: %v", err)
	}

	// In a normal test environment, we should have sufficient space
	if !sufficient {
		t.Log("warning: test system has low disk space")
	}
}

func TestIsDiskSpaceLow(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)

	low, err := v.IsDiskSpaceLow()
	if err != nil {
		t.Fatalf("IsDiskSpaceLow failed: %v", err)
	}

	// Just verify the function works - result depends on actual disk state
	t.Logf("Disk space low: %v", low)
}

func TestValidKeyNames(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	defer v.Lock()

	validKeys := []string{
		"api-key",
		"API_KEY",
		"db.password",
		"config/prod/api-key",
		"secret_123",
		"a",
		"test.key.with.dots",
	}

	for _, key := range validKeys {
		if err := v.SetSecret(key, &SecretEntry{Value: []byte("test")}); err != nil {
			t.Errorf("SetSecret(%q) should succeed, got: %v", key, err)
		}
	}
}

func TestInvalidKeyNames(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	defer v.Lock()

	tests := []struct {
		key       string
		expectErr error
	}{
		{"", ErrKeyTooShort},
		{".hidden", ErrKeyInvalid},
		{"-invalid", ErrKeyInvalid},
		{"key with spaces", ErrKeyInvalid},
		{"key@email", ErrKeyInvalid},
		{"key:colon", ErrKeyInvalid},
		// Per requirements-ja.md §2.1: forbid ".." pattern
		{"path/../traversal", ErrKeyInvalid},
		{"..start", ErrKeyInvalid},
		{"end..", ErrKeyInvalid},
		// Per requirements-ja.md §2.1: forbid leading/trailing "/"
		{"/leading", ErrKeyInvalid},
		{"trailing/", ErrKeyInvalid},
		{"/both/", ErrKeyInvalid},
		// Per requirements-ja.md §2.1: reserved prefixes
		{"_internal/secret", ErrKeyInvalid},
		{"_system/config", ErrKeyInvalid},
	}

	for _, tc := range tests {
		err := v.SetSecret(tc.key, &SecretEntry{Value: []byte("test")})
		if err == nil {
			t.Errorf("SetSecret(%q) should fail", tc.key)
			continue
		}
		if !containsError(err, tc.expectErr) {
			t.Errorf("SetSecret(%q) expected %v, got %v", tc.key, tc.expectErr, err)
		}
	}
}

func TestKeyTooLong(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	defer v.Lock()

	// Create a key that's exactly at the limit
	maxKey := make([]byte, MaxKeyLength)
	for i := range maxKey {
		maxKey[i] = 'a'
	}
	if err := v.SetSecret(string(maxKey), &SecretEntry{Value: []byte("test")}); err != nil {
		t.Errorf("SetSecret with max length key should succeed, got: %v", err)
	}

	// Create a key that's too long
	tooLongKey := make([]byte, MaxKeyLength+1)
	for i := range tooLongKey {
		tooLongKey[i] = 'a'
	}
	err := v.SetSecret(string(tooLongKey), &SecretEntry{Value: []byte("test")})
	if err == nil {
		t.Error("SetSecret with too long key should fail")
	}
	if !containsError(err, ErrKeyTooLong) {
		t.Errorf("expected ErrKeyTooLong, got %v", err)
	}
}

func TestValueTooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	defer v.Lock()

	// Create a value that's exactly at the limit
	maxValue := make([]byte, MaxValueSize)
	if err := v.SetSecret("max-value", &SecretEntry{Value: maxValue}); err != nil {
		t.Errorf("SetSecret with max size value should succeed, got: %v", err)
	}

	// Create a value that's too large
	tooLargeValue := make([]byte, MaxValueSize+1)
	err := v.SetSecret("too-large", &SecretEntry{Value: tooLargeValue})
	if err == nil {
		t.Error("SetSecret with too large value should fail")
	}
	if !containsError(err, ErrValueTooLarge) {
		t.Errorf("expected ErrValueTooLarge, got %v", err)
	}
}

// TestMetadataEncryptionRoundTrip verifies that encrypted metadata (notes, url)
// is correctly encrypted, stored, and decrypted during round-trip operations.
// This addresses requirements-ja.md §5 regarding encryption round-trip testing.
func TestMetadataEncryptionRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	defer v.Lock()

	testCases := []struct {
		name     string
		key      string
		value    []byte
		metadata *SecretMetadata
		tags     []string
	}{
		{
			name:  "notes only",
			key:   "test-notes",
			value: []byte("secret-value"),
			metadata: &SecretMetadata{
				Notes: "This is a test note with special chars: 日本語 émoji 🔐",
			},
		},
		{
			name:  "url only",
			key:   "test-url",
			value: []byte("another-secret"),
			metadata: &SecretMetadata{
				URL: "https://example.com/api?key=test&param=value#anchor",
			},
		},
		{
			name:  "notes and url",
			key:   "test-both",
			value: []byte("multi-metadata-secret"),
			metadata: &SecretMetadata{
				Notes: "Multi-line notes\nLine 2\nLine 3",
				URL:   "https://secure.example.com/endpoint",
			},
		},
		{
			name:  "with tags and expiration",
			key:   "test-full",
			value: []byte("full-metadata-secret"),
			metadata: &SecretMetadata{
				Notes: "Production API key",
				URL:   "https://api.service.com",
			},
			tags: []string{"prod", "api", "critical"},
		},
		{
			name:     "no metadata (nil)",
			key:      "test-nil-metadata",
			value:    []byte("simple-secret"),
			metadata: nil,
		},
		{
			name:  "empty metadata fields",
			key:   "test-empty-metadata",
			value: []byte("empty-fields-secret"),
			metadata: &SecretMetadata{
				Notes: "",
				URL:   "",
			},
		},
		{
			name:  "large notes",
			key:   "test-large-notes",
			value: []byte("large-notes-secret"),
			metadata: &SecretMetadata{
				Notes: string(make([]byte, 5000)), // 5KB of notes
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := &SecretEntry{
				Value:    tc.value,
				Metadata: tc.metadata,
				Tags:     tc.tags,
			}

			// Set secret with metadata
			if err := v.SetSecret(tc.key, entry); err != nil {
				t.Fatalf("SetSecret failed: %v", err)
			}

			// Retrieve secret
			retrieved, err := v.GetSecret(tc.key)
			if err != nil {
				t.Fatalf("GetSecret failed: %v", err)
			}

			// Verify value
			if string(retrieved.Value) != string(tc.value) {
				t.Errorf("value mismatch: expected %q, got %q", string(tc.value), string(retrieved.Value))
			}

			// Verify metadata
			if tc.metadata != nil && (tc.metadata.Notes != "" || tc.metadata.URL != "") {
				if retrieved.Metadata == nil {
					t.Fatal("expected metadata to be present")
				}
				if retrieved.Metadata.Notes != tc.metadata.Notes {
					t.Errorf("notes mismatch: expected %q, got %q", tc.metadata.Notes, retrieved.Metadata.Notes)
				}
				if retrieved.Metadata.URL != tc.metadata.URL {
					t.Errorf("url mismatch: expected %q, got %q", tc.metadata.URL, retrieved.Metadata.URL)
				}
			} else if tc.metadata == nil || (tc.metadata.Notes == "" && tc.metadata.URL == "") {
				// For nil or empty metadata, retrieved.Metadata may be nil
				if retrieved.Metadata != nil && (retrieved.Metadata.Notes != "" || retrieved.Metadata.URL != "") {
					t.Errorf("expected nil or empty metadata, got %+v", retrieved.Metadata)
				}
			}

			// Verify tags
			if len(tc.tags) > 0 {
				if len(retrieved.Tags) != len(tc.tags) {
					t.Errorf("tags count mismatch: expected %d, got %d", len(tc.tags), len(retrieved.Tags))
				}
				for i, tag := range tc.tags {
					if i < len(retrieved.Tags) && retrieved.Tags[i] != tag {
						t.Errorf("tag mismatch at index %d: expected %q, got %q", i, tag, retrieved.Tags[i])
					}
				}
			}
		})
	}
}

// TestCheckIntegrityPermissions verifies that CheckIntegrity detects insecure file permissions.
// This addresses security-design-ja.md §10 regarding file permission checks (0600).
func TestCheckIntegrityPermissions(t *testing.T) {
	// Skip on Windows where file permissions work differently
	if filepath.Separator == '\\' {
		t.Skip("Skipping permission tests on Windows")
	}

	t.Run("valid permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		// t.TempDir() creates with 0755, we need to set 0700 for valid permissions
		if err := os.Chmod(tmpDir, 0700); err != nil {
			t.Fatalf("failed to set directory permissions: %v", err)
		}

		v := New(tmpDir)
		password := "testpassword123"

		if err := v.Init(password); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		result, err := v.CheckIntegrity()
		if err != nil {
			t.Fatalf("CheckIntegrity failed: %v", err)
		}

		if !result.PermissionsValid {
			t.Errorf("expected PermissionsValid to be true, errors: %v", result.Errors)
		}
	})

	t.Run("insecure vault directory permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		v := New(tmpDir)
		password := "testpassword123"

		if err := v.Init(password); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Make directory world-readable
		if err := os.Chmod(tmpDir, 0755); err != nil {
			t.Fatalf("failed to change permissions: %v", err)
		}

		result, err := v.CheckIntegrity()
		if err != nil {
			t.Fatalf("CheckIntegrity failed: %v", err)
		}

		if result.PermissionsValid {
			t.Error("expected PermissionsValid to be false for world-readable directory")
		}

		// Verify error message mentions permissions
		hasPermError := false
		for _, e := range result.Errors {
			if len(e) > 10 && (e[:10] == "vault dire" || contains(e, "permissions")) {
				hasPermError = true
				break
			}
		}
		if !hasPermError {
			t.Errorf("expected permission error message, got: %v", result.Errors)
		}
	})

	t.Run("insecure salt file permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		v := New(tmpDir)
		password := "testpassword123"

		if err := v.Init(password); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Make salt file world-readable
		saltPath := filepath.Join(tmpDir, SaltFileName)
		if err := os.Chmod(saltPath, 0644); err != nil {
			t.Fatalf("failed to change permissions: %v", err)
		}

		result, err := v.CheckIntegrity()
		if err != nil {
			t.Fatalf("CheckIntegrity failed: %v", err)
		}

		if result.PermissionsValid {
			t.Error("expected PermissionsValid to be false for world-readable salt file")
		}
	})

	t.Run("insecure database file permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		v := New(tmpDir)
		password := "testpassword123"

		if err := v.Init(password); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Make database file world-readable
		dbPath := filepath.Join(tmpDir, DBFileName)
		if err := os.Chmod(dbPath, 0644); err != nil {
			t.Fatalf("failed to change permissions: %v", err)
		}

		result, err := v.CheckIntegrity()
		if err != nil {
			t.Fatalf("CheckIntegrity failed: %v", err)
		}

		if result.PermissionsValid {
			t.Error("expected PermissionsValid to be false for world-readable database file")
		}
		// Permission failures should also mark Valid=false
		if result.Valid {
			t.Error("expected Valid to be false for insecure permissions")
		}
	})

	t.Run("insecure metadata file permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		v := New(tmpDir)
		password := "testpassword123"

		if err := v.Init(password); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Make metadata file world-readable
		metaPath := filepath.Join(tmpDir, MetaFileName)
		if err := os.Chmod(metaPath, 0644); err != nil {
			t.Fatalf("failed to change permissions: %v", err)
		}

		result, err := v.CheckIntegrity()
		if err != nil {
			t.Fatalf("CheckIntegrity failed: %v", err)
		}

		if result.PermissionsValid {
			t.Error("expected PermissionsValid to be false for world-readable metadata file")
		}
		// Permission failures should also mark Valid=false
		if result.Valid {
			t.Error("expected Valid to be false for insecure permissions")
		}
	})
}

// TestUnlockPermissionWarning verifies that Unlock warns about insecure permissions
// per requirements-ja.md §4.1
func TestUnlockPermissionWarning(t *testing.T) {
	// Skip on Windows where file permissions work differently
	if filepath.Separator == '\\' {
		t.Skip("Skipping permission tests on Windows")
	}

	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Make salt file world-readable to trigger warning
	saltPath := filepath.Join(tmpDir, SaltFileName)
	if err := os.Chmod(saltPath, 0644); err != nil {
		t.Fatalf("failed to change permissions: %v", err)
	}

	// Unlock should succeed but print warning to stderr
	// We can't easily capture stderr in this test, but we verify unlock works
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	// Verify vault is unlocked despite insecure permissions (warning only, not blocking)
	if v.IsLocked() {
		t.Error("expected vault to be unlocked despite insecure permissions")
	}

	v.Lock()
}

// contains is a helper function for string containment check
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestMetadataUpdateRoundTrip verifies that updating metadata preserves data integrity
func TestMetadataUpdateRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	defer v.Lock()

	key := "update-test"

	// Initial set without metadata
	if err := v.SetSecret(key, &SecretEntry{Value: []byte("initial")}); err != nil {
		t.Fatalf("SetSecret failed: %v", err)
	}

	// Update with metadata
	if err := v.SetSecret(key, &SecretEntry{
		Value: []byte("updated"),
		Metadata: &SecretMetadata{
			Notes: "Added notes",
			URL:   "https://example.com",
		},
		Tags: []string{"updated", "test"},
	}); err != nil {
		t.Fatalf("SetSecret update failed: %v", err)
	}

	// Verify update
	retrieved, err := v.GetSecret(key)
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}

	if string(retrieved.Value) != "updated" {
		t.Errorf("value not updated: got %q", string(retrieved.Value))
	}
	if retrieved.Metadata == nil {
		t.Fatal("expected metadata after update")
	}
	if retrieved.Metadata.Notes != "Added notes" {
		t.Errorf("notes not updated: got %q", retrieved.Metadata.Notes)
	}
	if retrieved.Metadata.URL != "https://example.com" {
		t.Errorf("url not updated: got %q", retrieved.Metadata.URL)
	}

	// Update again to remove notes
	if err := v.SetSecret(key, &SecretEntry{
		Value: []byte("final"),
		Metadata: &SecretMetadata{
			URL: "https://final.example.com",
		},
	}); err != nil {
		t.Fatalf("SetSecret second update failed: %v", err)
	}

	// Verify second update
	retrieved, err = v.GetSecret(key)
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}

	if string(retrieved.Value) != "final" {
		t.Errorf("value not updated: got %q", string(retrieved.Value))
	}
	if retrieved.Metadata == nil {
		t.Fatal("expected metadata after update")
	}
	if retrieved.Metadata.Notes != "" {
		t.Errorf("notes should be empty: got %q", retrieved.Metadata.Notes)
	}
	if retrieved.Metadata.URL != "https://final.example.com" {
		t.Errorf("url not updated: got %q", retrieved.Metadata.URL)
	}
}

// TestMetadataValidation tests metadata validation per requirements-ja.md §2.5 and §4.2.2
func TestMetadataValidation(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	defer v.Lock()

	tests := []struct {
		name      string
		entry     *SecretEntry
		wantErr   error
		errSubstr string
	}{
		{
			name: "notes exceeds maximum size",
			entry: &SecretEntry{
				Value: []byte("test"),
				Metadata: &SecretMetadata{
					Notes: string(make([]byte, MaxNotesSize+1)),
				},
			},
			wantErr:   ErrNotesTooLarge,
			errSubstr: "notes too large",
		},
		{
			name: "url exceeds maximum length",
			entry: &SecretEntry{
				Value: []byte("test"),
				Metadata: &SecretMetadata{
					URL: "https://example.com/" + string(make([]byte, MaxURLLength)),
				},
			},
			wantErr:   ErrURLTooLong,
			errSubstr: "url too long",
		},
		{
			name: "too many tags",
			entry: &SecretEntry{
				Value: []byte("test"),
				Tags:  []string{"t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8", "t9", "t10", "t11"},
			},
			wantErr:   ErrTooManyTags,
			errSubstr: "too many tags",
		},
		{
			name: "tag too long",
			entry: &SecretEntry{
				Value: []byte("test"),
				Tags:  []string{string(make([]byte, MaxTagLength+1))},
			},
			wantErr:   ErrTagInvalid,
			errSubstr: "invalid tag format",
		},
		{
			name: "tag with invalid characters",
			entry: &SecretEntry{
				Value: []byte("test"),
				Tags:  []string{"invalid tag!"},
			},
			wantErr:   ErrTagInvalid,
			errSubstr: "must match [a-zA-Z0-9_-]",
		},
		{
			name: "empty tag",
			entry: &SecretEntry{
				Value: []byte("test"),
				Tags:  []string{""},
			},
			wantErr:   ErrTagInvalid,
			errSubstr: "must be 1-64 characters",
		},
		{
			name: "valid metadata at limits",
			entry: &SecretEntry{
				Value: []byte("test"),
				Metadata: &SecretMetadata{
					Notes: string(make([]byte, MaxNotesSize)),
					URL:   "https://example.com",
				},
				Tags: []string{"t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8", "t9", "t10"},
			},
			wantErr: nil,
		},
		{
			name: "valid tags with underscore and hyphen",
			entry: &SecretEntry{
				Value: []byte("test"),
				Tags:  []string{"my-tag", "my_tag", "tag123", "TAG"},
			},
			wantErr: nil,
		},
		// URL scheme validation tests
		{
			name: "javascript url scheme rejected",
			entry: &SecretEntry{
				Value: []byte("test"),
				Metadata: &SecretMetadata{
					URL: "javascript:alert('xss')",
				},
			},
			wantErr:   ErrURLInvalid,
			errSubstr: "only http and https schemes are allowed",
		},
		{
			name: "file url scheme rejected",
			entry: &SecretEntry{
				Value: []byte("test"),
				Metadata: &SecretMetadata{
					URL: "file:///etc/passwd",
				},
			},
			wantErr:   ErrURLInvalid,
			errSubstr: "only http and https schemes are allowed",
		},
		{
			name: "ftp url scheme rejected",
			entry: &SecretEntry{
				Value: []byte("test"),
				Metadata: &SecretMetadata{
					URL: "ftp://example.com/file",
				},
			},
			wantErr:   ErrURLInvalid,
			errSubstr: "only http and https schemes are allowed",
		},
		{
			name: "url without host rejected",
			entry: &SecretEntry{
				Value: []byte("test"),
				Metadata: &SecretMetadata{
					URL: "http:///path/only",
				},
			},
			wantErr:   ErrURLInvalid,
			errSubstr: "URL must have a host",
		},
		{
			name: "valid https url",
			entry: &SecretEntry{
				Value: []byte("test"),
				Metadata: &SecretMetadata{
					URL: "https://example.com/api/v1",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid http url",
			entry: &SecretEntry{
				Value: []byte("test"),
				Metadata: &SecretMetadata{
					URL: "http://localhost:8080/health",
				},
			},
			wantErr: nil,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use unique key for each test
			key := "test-key-" + string(rune('a'+i))
			err := v.SetSecret(key, tt.entry)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error containing %v, got nil", tt.wantErr)
					return
				}
				if !contains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

// TestValidateMasterPassword tests password validation per requirements-ja.md §2.3
func TestValidateMasterPassword(t *testing.T) {
	tests := []struct {
		name            string
		password        string
		expectValid     bool
		expectStrength  PasswordStrength
		expectWarnings  bool
		minWarningCount int
	}{
		// Hard requirement failures
		{
			name:            "too short",
			password:        "abc123",
			expectValid:     false,
			expectStrength:  PasswordWeak,
			expectWarnings:  true,
			minWarningCount: 1,
		},
		{
			name:            "empty password",
			password:        "",
			expectValid:     false,
			expectStrength:  PasswordWeak,
			expectWarnings:  true,
			minWarningCount: 1,
		},
		{
			name:            "just under minimum",
			password:        "1234567",
			expectValid:     false,
			expectStrength:  PasswordWeak,
			expectWarnings:  true,
			minWarningCount: 1,
		},
		{
			name:            "too long (129 chars)",
			password:        string(make([]byte, MaxPasswordLength+1)),
			expectValid:     false,
			expectStrength:  PasswordWeak,
			expectWarnings:  true,
			minWarningCount: 1,
		},
		// Valid passwords with varying strengths
		{
			name:            "exactly minimum length, simple",
			password:        "password",
			expectValid:     true,
			expectStrength:  PasswordWeak,
			expectWarnings:  true,
			minWarningCount: 2, // complexity + length warning
		},
		{
			name:            "8 chars with numbers",
			password:        "pass1234",
			expectValid:     true,
			expectStrength:  PasswordFair, // 2 character types (lower + digit)
			expectWarnings:  true,
			minWarningCount: 1, // length warning only
		},
		{
			name:            "12 chars mixed case",
			password:        "Password1234",
			expectValid:     true,
			expectStrength:  PasswordGood,
			expectWarnings:  false,
			minWarningCount: 0,
		},
		{
			name:            "16 chars with all types",
			password:        "Password1234!@#$",
			expectValid:     true,
			expectStrength:  PasswordStrong,
			expectWarnings:  false,
			minWarningCount: 0,
		},
		{
			name:            "maximum length valid",
			password:        string(make([]byte, MaxPasswordLength)),
			expectValid:     true,
			expectStrength:  PasswordFair,
			expectWarnings:  true,
			minWarningCount: 1, // complexity warning (all nulls)
		},
		{
			name:            "only lowercase",
			password:        "verylongpassword",
			expectValid:     true,
			expectStrength:  PasswordFair, // long but low complexity
			expectWarnings:  true,
			minWarningCount: 1, // complexity warning
		},
		{
			name:            "short but complex",
			password:        "Pa1!sswd",
			expectValid:     true,
			expectStrength:  PasswordFair,
			expectWarnings:  true,
			minWarningCount: 1, // length warning
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateMasterPassword(tt.password)

			if result.Valid != tt.expectValid {
				t.Errorf("Valid: expected %v, got %v", tt.expectValid, result.Valid)
			}

			if result.Strength != tt.expectStrength {
				t.Errorf("Strength: expected %v, got %v", tt.expectStrength, result.Strength)
			}

			if tt.expectWarnings && len(result.Warnings) < tt.minWarningCount {
				t.Errorf("Warnings: expected at least %d, got %d: %v",
					tt.minWarningCount, len(result.Warnings), result.Warnings)
			}

			if !tt.expectWarnings && len(result.Warnings) > 0 {
				t.Errorf("Warnings: expected none, got %v", result.Warnings)
			}
		})
	}
}

// TestPasswordStrengthString tests the String method of PasswordStrength
func TestPasswordStrengthString(t *testing.T) {
	tests := []struct {
		strength PasswordStrength
		expected string
	}{
		{PasswordWeak, "weak"},
		{PasswordFair, "fair"},
		{PasswordGood, "good"},
		{PasswordStrong, "strong"},
		{PasswordStrength(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.strength.String(); got != tt.expected {
				t.Errorf("String(): expected %q, got %q", tt.expected, got)
			}
		})
	}
}

// TestListSecretsWithMetadata verifies that ListSecretsWithMetadata:
// 1. Returns entries with metadata (Notes/URL) populated
// 2. Does NOT return secret values (they should be nil/empty)
func TestListSecretsWithMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)

	// Initialize and unlock
	if err := v.Init("testpassword123"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := v.Unlock("testpassword123"); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	// Create test secrets with and without metadata
	secretValue := []byte("supersecretvalue123")

	// Secret with metadata
	if err := v.SetSecret("key-with-meta", &SecretEntry{
		Value: secretValue,
		Metadata: &SecretMetadata{
			Notes: "This is a test note",
			URL:   "https://example.com",
		},
		Tags: []string{"test"},
	}); err != nil {
		t.Fatalf("SetSecret with metadata failed: %v", err)
	}

	// Secret without metadata
	if err := v.SetSecret("key-without-meta", &SecretEntry{
		Value: secretValue,
	}); err != nil {
		t.Fatalf("SetSecret without metadata failed: %v", err)
	}

	// List with metadata
	entries, err := v.ListSecretsWithMetadata()
	if err != nil {
		t.Fatalf("ListSecretsWithMetadata failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Check that values are NOT populated (security requirement)
	for _, entry := range entries {
		if len(entry.Value) != 0 {
			t.Errorf("entry %q has Value populated (security violation!)", entry.Key)
		}
	}

	// Find the entry with metadata
	var withMeta, withoutMeta *SecretEntry
	for _, entry := range entries {
		if entry.Key == "key-with-meta" {
			withMeta = entry
		} else if entry.Key == "key-without-meta" {
			withoutMeta = entry
		}
	}

	if withMeta == nil || withoutMeta == nil {
		t.Fatal("couldn't find expected entries")
	}

	// Verify metadata IS populated for the entry that has it
	if withMeta.Metadata == nil {
		t.Error("expected metadata to be populated for key-with-meta")
	} else {
		if withMeta.Metadata.Notes != "This is a test note" {
			t.Errorf("expected notes 'This is a test note', got %q", withMeta.Metadata.Notes)
		}
		if withMeta.Metadata.URL != "https://example.com" {
			t.Errorf("expected URL 'https://example.com', got %q", withMeta.Metadata.URL)
		}
	}

	// Verify metadata is nil for entry without it
	if withoutMeta.Metadata != nil {
		t.Error("expected metadata to be nil for key-without-meta")
	}

	// Verify tags are populated
	if len(withMeta.Tags) != 1 || withMeta.Tags[0] != "test" {
		t.Errorf("expected tags ['test'], got %v", withMeta.Tags)
	}
}

// TestListSecretsByTagNoValueDecryption verifies that ListSecretsByTag
// does NOT return secret values (they should be nil/empty).
func TestListSecretsByTagNoValueDecryption(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)

	// Initialize and unlock
	if err := v.Init("testpassword123"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := v.Unlock("testpassword123"); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	secretValue := []byte("supersecretvalue123")

	// Create secrets with tags
	if err := v.SetSecret("tagged-secret-1", &SecretEntry{
		Value: secretValue,
		Tags:  []string{"production", "database"},
	}); err != nil {
		t.Fatalf("SetSecret 1 failed: %v", err)
	}

	if err := v.SetSecret("tagged-secret-2", &SecretEntry{
		Value: secretValue,
		Tags:  []string{"production"},
	}); err != nil {
		t.Fatalf("SetSecret 2 failed: %v", err)
	}

	// List by tag
	entries, err := v.ListSecretsByTag("production")
	if err != nil {
		t.Fatalf("ListSecretsByTag failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// CRITICAL: Verify values are NOT populated (security requirement)
	for _, entry := range entries {
		if len(entry.Value) != 0 {
			t.Errorf("entry %q has Value populated (security violation!)", entry.Key)
		}
	}

	// Verify tags are populated
	for _, entry := range entries {
		if len(entry.Tags) == 0 {
			t.Errorf("entry %q has no tags", entry.Key)
		}
	}
}

// TestListExpiringSecretsNoValueDecryption verifies that ListExpiringSecrets
// does NOT return secret values (they should be nil/empty).
func TestListExpiringSecretsNoValueDecryption(t *testing.T) {
	tmpDir := t.TempDir()
	v := New(tmpDir)

	// Initialize and unlock
	if err := v.Init("testpassword123"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := v.Unlock("testpassword123"); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	secretValue := []byte("supersecretvalue123")

	// Create a secret that expires soon (within 30 days)
	expiresSoon := time.Now().Add(7 * 24 * time.Hour) // 7 days from now
	if err := v.SetSecret("expiring-secret", &SecretEntry{
		Value:     secretValue,
		ExpiresAt: &expiresSoon,
		Metadata: &SecretMetadata{
			Notes: "This secret expires soon",
		},
	}); err != nil {
		t.Fatalf("SetSecret failed: %v", err)
	}

	// List expiring secrets (within 30 days)
	entries, err := v.ListExpiringSecrets(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("ListExpiringSecrets failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	// CRITICAL: Verify value is NOT populated (security requirement)
	if len(entries[0].Value) != 0 {
		t.Error("expiring entry has Value populated (security violation!)")
	}

	// Verify metadata IS populated
	if entries[0].Metadata == nil {
		t.Error("expected metadata to be populated")
	} else if entries[0].Metadata.Notes != "This secret expires soon" {
		t.Errorf("expected notes, got %q", entries[0].Metadata.Notes)
	}

	// Verify expiration is populated
	if entries[0].ExpiresAt == nil {
		t.Error("expected ExpiresAt to be populated")
	}
}
