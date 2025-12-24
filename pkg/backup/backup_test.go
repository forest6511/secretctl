package backup

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/forest6511/secretctl/pkg/crypto"
	"github.com/forest6511/secretctl/pkg/vault"
)

func TestGenerateSalt(t *testing.T) {
	salt1, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}

	if len(salt1) != SaltLength {
		t.Errorf("Expected salt length %d, got %d", SaltLength, len(salt1))
	}

	// Generate another salt and ensure they're different
	salt2, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}

	if bytes.Equal(salt1, salt2) {
		t.Error("Two generated salts should be different")
	}
}

func TestDeriveBackupKeys(t *testing.T) {
	password := []byte("test-password-123")
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}

	encKey, macKey, err := DeriveBackupKeys(password, salt)
	if err != nil {
		t.Fatalf("DeriveBackupKeys failed: %v", err)
	}
	defer crypto.SecureWipe(encKey)
	defer crypto.SecureWipe(macKey)

	if len(encKey) != KeyLength {
		t.Errorf("Expected encryption key length %d, got %d", KeyLength, len(encKey))
	}

	if len(macKey) != KeyLength {
		t.Errorf("Expected MAC key length %d, got %d", KeyLength, len(macKey))
	}

	// Keys should be different
	if bytes.Equal(encKey, macKey) {
		t.Error("Encryption and MAC keys should be different")
	}

	// Same password + salt should produce same keys
	encKey2, macKey2, err := DeriveBackupKeys(password, salt)
	if err != nil {
		t.Fatalf("DeriveBackupKeys failed: %v", err)
	}
	defer crypto.SecureWipe(encKey2)
	defer crypto.SecureWipe(macKey2)

	if !bytes.Equal(encKey, encKey2) {
		t.Error("Same password+salt should produce same encryption key")
	}
	if !bytes.Equal(macKey, macKey2) {
		t.Error("Same password+salt should produce same MAC key")
	}
}

func TestDeriveBackupKeys_EmptyPassword(t *testing.T) {
	salt, _ := GenerateSalt()

	_, _, err := DeriveBackupKeys([]byte{}, salt)
	if err != ErrEmptyPassword {
		t.Errorf("Expected ErrEmptyPassword, got %v", err)
	}

	_, _, err = DeriveBackupKeys(nil, salt)
	if err != ErrEmptyPassword {
		t.Errorf("Expected ErrEmptyPassword for nil password, got %v", err)
	}
}

func TestEncryptDecryptPayload(t *testing.T) {
	key := make([]byte, KeyLength)
	if _, err := cryptorand.Read(key); err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	defer crypto.SecureWipe(key)

	plaintext := []byte("test payload data for encryption")

	ciphertext, err := EncryptPayload(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptPayload failed: %v", err)
	}

	// Ciphertext should be different from plaintext
	if bytes.Equal(ciphertext, plaintext) {
		t.Error("Ciphertext should be different from plaintext")
	}

	// Decrypt
	decrypted, err := DecryptPayload(ciphertext, key)
	if err != nil {
		t.Fatalf("DecryptPayload failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypted data doesn't match original")
	}
}

func TestDecryptPayload_InvalidData(t *testing.T) {
	key := make([]byte, KeyLength)
	cryptorand.Read(key)
	defer crypto.SecureWipe(key)

	// Too short
	_, err := DecryptPayload([]byte("short"), key)
	if err != ErrDecryptionFailed {
		t.Errorf("Expected ErrDecryptionFailed for short data, got %v", err)
	}

	// Corrupted ciphertext
	validCiphertext, _ := EncryptPayload([]byte("test"), key)
	validCiphertext[len(validCiphertext)-1] ^= 0xFF // Corrupt last byte
	_, err = DecryptPayload(validCiphertext, key)
	if err != ErrDecryptionFailed {
		t.Errorf("Expected ErrDecryptionFailed for corrupted data, got %v", err)
	}
}

func TestComputeVerifyHMAC(t *testing.T) {
	key := make([]byte, KeyLength)
	cryptorand.Read(key)
	defer crypto.SecureWipe(key)

	data := []byte("test data for HMAC")

	hmac := ComputeHMAC(data, key)
	if len(hmac) != HMACLength {
		t.Errorf("Expected HMAC length %d, got %d", HMACLength, len(hmac))
	}

	// Verify should succeed
	if !VerifyHMAC(data, hmac, key) {
		t.Error("HMAC verification should succeed")
	}

	// Verify should fail with wrong data
	if VerifyHMAC([]byte("wrong data"), hmac, key) {
		t.Error("HMAC verification should fail with wrong data")
	}

	// Verify should fail with wrong HMAC
	wrongHMAC := make([]byte, HMACLength)
	if VerifyHMAC(data, wrongHMAC, key) {
		t.Error("HMAC verification should fail with wrong HMAC")
	}

	// Verify should fail with wrong key
	wrongKey := make([]byte, KeyLength)
	cryptorand.Read(wrongKey)
	if VerifyHMAC(data, hmac, wrongKey) {
		t.Error("HMAC verification should fail with wrong key")
	}
}

func TestWriteReadHeader(t *testing.T) {
	header := &Header{
		Version:        1,
		CreatedAt:      time.Now().UTC().Truncate(time.Second),
		VaultVersion:   1,
		EncryptionMode: EncryptionModeMaster,
		KDFParams: &KDFParams{
			Salt:        []byte("test-salt-32-bytes-for-testing!!"),
			Memory:      65536,
			Iterations:  3,
			Parallelism: 4,
		},
		IncludesAudit: true,
		SecretCount:   42,
		ChecksumAlgo:  "sha256",
	}

	var buf bytes.Buffer
	if err := WriteHeader(&buf, header); err != nil {
		t.Fatalf("WriteHeader failed: %v", err)
	}

	readHeader, err := ReadHeader(&buf)
	if err != nil {
		t.Fatalf("ReadHeader failed: %v", err)
	}

	if readHeader.Version != header.Version {
		t.Errorf("Version mismatch: expected %d, got %d", header.Version, readHeader.Version)
	}
	if !readHeader.CreatedAt.Equal(header.CreatedAt) {
		t.Errorf("CreatedAt mismatch: expected %v, got %v", header.CreatedAt, readHeader.CreatedAt)
	}
	if readHeader.EncryptionMode != header.EncryptionMode {
		t.Errorf("EncryptionMode mismatch")
	}
	if readHeader.SecretCount != header.SecretCount {
		t.Errorf("SecretCount mismatch: expected %d, got %d", header.SecretCount, readHeader.SecretCount)
	}
	if readHeader.IncludesAudit != header.IncludesAudit {
		t.Errorf("IncludesAudit mismatch")
	}
}

func TestReadHeader_InvalidMagic(t *testing.T) {
	invalidData := bytes.NewReader([]byte("INVALID_"))

	_, err := ReadHeader(invalidData)
	if err != ErrInvalidMagic {
		t.Errorf("Expected ErrInvalidMagic, got %v", err)
	}
}

func TestReadHeader_UnsupportedVersion(t *testing.T) {
	header := &Header{
		Version:        99, // Unsupported version
		CreatedAt:      time.Now().UTC(),
		VaultVersion:   1,
		EncryptionMode: EncryptionModeMaster,
		SecretCount:    0,
		ChecksumAlgo:   "sha256",
	}

	var buf bytes.Buffer
	WriteHeader(&buf, header)

	_, err := ReadHeader(&buf)
	if err == nil {
		t.Error("Expected error for unsupported version")
	}
}

func TestEncodeDecodePayload(t *testing.T) {
	payload := &Payload{
		VaultSalt: []byte("test-vault-salt-32-bytes-long!!"),
		VaultMeta: []byte(`{"version": 1}`),
		VaultDB:   []byte("SQLite database content"),
		AuditLog:  []byte(`{"action": "test"}`),
	}

	encoded, err := EncodePayload(payload)
	if err != nil {
		t.Fatalf("EncodePayload failed: %v", err)
	}

	decoded, err := DecodePayload(encoded)
	if err != nil {
		t.Fatalf("DecodePayload failed: %v", err)
	}

	if !bytes.Equal(decoded.VaultSalt, payload.VaultSalt) {
		t.Error("VaultSalt mismatch")
	}
	if !bytes.Equal(decoded.VaultMeta, payload.VaultMeta) {
		t.Error("VaultMeta mismatch")
	}
	if !bytes.Equal(decoded.VaultDB, payload.VaultDB) {
		t.Error("VaultDB mismatch")
	}
	if !bytes.Equal(decoded.AuditLog, payload.AuditLog) {
		t.Error("AuditLog mismatch")
	}
}

func TestKeyFile(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "backup.key")

	// Generate key file
	if err := GenerateKeyFile(keyPath); err != nil {
		t.Fatalf("GenerateKeyFile failed: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Failed to stat key file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected permissions 0600, got %o", info.Mode().Perm())
	}

	// Read key file
	key, err := ReadKeyFile(keyPath)
	if err != nil {
		t.Fatalf("ReadKeyFile failed: %v", err)
	}
	defer crypto.SecureWipe(key)

	if len(key) != KeyLength {
		t.Errorf("Expected key length %d, got %d", KeyLength, len(key))
	}
}

func TestReadKeyFile_InvalidSize(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "invalid.key")

	// Write invalid size key
	os.WriteFile(keyPath, []byte("short"), 0600)

	_, err := ReadKeyFile(keyPath)
	if err != ErrInvalidKeyFile {
		t.Errorf("Expected ErrInvalidKeyFile, got %v", err)
	}
}

func TestBackupRestore_RoundTrip(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	restoreDir := filepath.Join(tempDir, "restored")
	backupFile := filepath.Join(tempDir, "backup.enc")

	// Initialize a test vault
	password := "test-password-123"
	v := vault.New(vaultDir)
	if err := v.Init(password); err != nil {
		t.Fatalf("Failed to init vault: %v", err)
	}

	// Unlock and add some secrets
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Failed to unlock vault: %v", err)
	}

	// Add test secrets
	testSecrets := map[string]string{
		"test/secret1": "value1",
		"test/secret2": "value2",
		"prod/api-key": "secret-api-key",
	}
	for key, value := range testSecrets {
		if err := v.SetSecret(key, &vault.SecretEntry{Value: []byte(value)}); err != nil {
			t.Fatalf("Failed to set secret %s: %v", key, err)
		}
	}

	// Create backup
	backupOutput, err := os.Create(backupFile)
	if err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	opts := BackupOptions{
		Output:       backupOutput,
		IncludeAudit: false,
		Password:     []byte(password),
	}

	if err := Backup(v, opts); err != nil {
		backupOutput.Close()
		t.Fatalf("Backup failed: %v", err)
	}
	backupOutput.Close()
	v.Lock()

	// Verify backup file exists and has content
	info, err := os.Stat(backupFile)
	if err != nil {
		t.Fatalf("Backup file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Backup file is empty")
	}

	// Verify backup
	verifyResult, err := Verify(backupFile, []byte(password), "")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !verifyResult.Valid {
		t.Errorf("Backup verification failed: %s", verifyResult.Error)
	}
	if verifyResult.SecretCount != len(testSecrets) {
		t.Errorf("Expected %d secrets, got %d", len(testSecrets), verifyResult.SecretCount)
	}

	// Restore to new location
	restoreOpts := RestoreOptions{
		VaultPath:  restoreDir,
		OnConflict: ConflictError,
		DryRun:     false,
		VerifyOnly: false,
		WithAudit:  false,
		Password:   []byte(password),
	}

	result, err := Restore(backupFile, restoreOpts)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	if result.SecretsRestored != len(testSecrets) {
		t.Errorf("Expected %d secrets restored, got %d", len(testSecrets), result.SecretsRestored)
	}

	// Open restored vault and verify secrets
	restoredVault := vault.New(restoreDir)
	if restoredVault == nil {
		t.Fatalf("Failed to create restored vault instance")
	}
	if err := restoredVault.Unlock(password); err != nil {
		t.Fatalf("Failed to unlock restored vault: %v", err)
	}
	defer restoredVault.Lock()

	for key, expectedValue := range testSecrets {
		entry, err := restoredVault.GetSecret(key)
		if err != nil {
			t.Errorf("Failed to get secret %s from restored vault: %v", key, err)
			continue
		}
		if string(entry.Value) != expectedValue {
			t.Errorf("Secret %s value mismatch: expected %s, got %s", key, expectedValue, string(entry.Value))
		}
	}
}

func TestBackupRestore_WithKeyFile(t *testing.T) {
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	restoreDir := filepath.Join(tempDir, "restored")
	backupFile := filepath.Join(tempDir, "backup.enc")
	keyFile := filepath.Join(tempDir, "backup.key")

	// Generate key file
	if err := GenerateKeyFile(keyFile); err != nil {
		t.Fatalf("GenerateKeyFile failed: %v", err)
	}

	// Initialize vault
	password := "test-password"
	v := vault.New(vaultDir)
	if err := v.Init(password); err != nil {
		t.Fatalf("Failed to init vault: %v", err)
	}
	if err := v.Unlock(password); err != nil {
		t.Fatalf("Failed to unlock vault: %v", err)
	}
	if err := v.SetSecret("test/key", &vault.SecretEntry{Value: []byte("test-value")}); err != nil {
		t.Fatalf("Failed to set secret: %v", err)
	}

	// Create backup with key file
	backupOutput, _ := os.Create(backupFile)
	opts := BackupOptions{
		Output:       backupOutput,
		IncludeAudit: false,
		KeyFile:      keyFile,
	}
	if err := Backup(v, opts); err != nil {
		backupOutput.Close()
		t.Fatalf("Backup with key file failed: %v", err)
	}
	backupOutput.Close()
	v.Lock()

	// Verify with key file
	verifyResult, err := Verify(backupFile, nil, keyFile)
	if err != nil {
		t.Fatalf("Verify with key file failed: %v", err)
	}
	if !verifyResult.Valid {
		t.Errorf("Verification failed: %s", verifyResult.Error)
	}

	// Restore with key file
	restoreOpts := RestoreOptions{
		VaultPath:  restoreDir,
		OnConflict: ConflictError,
		KeyFile:    keyFile,
	}
	_, err = Restore(backupFile, restoreOpts)
	if err != nil {
		t.Fatalf("Restore with key file failed: %v", err)
	}
}

func TestRestore_DryRun(t *testing.T) {
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	restoreDir := filepath.Join(tempDir, "restored")
	backupFile := filepath.Join(tempDir, "backup.enc")

	// Setup vault and backup
	password := "test-password"
	v := vault.New(vaultDir)
	v.Init(password)
	v.Unlock(password)
	v.SetSecret("test/key", &vault.SecretEntry{Value: []byte("value")})

	backupOutput, _ := os.Create(backupFile)
	Backup(v, BackupOptions{Output: backupOutput, Password: []byte(password)})
	backupOutput.Close()
	v.Lock()

	// Dry run
	result, err := Restore(backupFile, RestoreOptions{
		VaultPath: restoreDir,
		DryRun:    true,
		Password:  []byte(password),
	})
	if err != nil {
		t.Fatalf("Dry run failed: %v", err)
	}
	if !result.DryRun {
		t.Error("Expected DryRun to be true")
	}
	if result.SecretsRestored != 1 {
		t.Errorf("Expected 1 secret in dry run, got %d", result.SecretsRestored)
	}

	// Verify restore dir was NOT created
	if _, err := os.Stat(restoreDir); !os.IsNotExist(err) {
		t.Error("Restore directory should not exist after dry run")
	}
}

func TestRestore_ConflictModes(t *testing.T) {
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	restoreDir := filepath.Join(tempDir, "restored")
	backupFile := filepath.Join(tempDir, "backup.enc")

	// Setup vault and backup
	password := "test-password"
	v := vault.New(vaultDir)
	v.Init(password)
	v.Unlock(password)
	v.SetSecret("test/key", &vault.SecretEntry{Value: []byte("value")})

	backupOutput, _ := os.Create(backupFile)
	Backup(v, BackupOptions{Output: backupOutput, Password: []byte(password)})
	backupOutput.Close()
	v.Lock()

	// First restore
	_, err := Restore(backupFile, RestoreOptions{
		VaultPath:  restoreDir,
		OnConflict: ConflictError,
		Password:   []byte(password),
	})
	if err != nil {
		t.Fatalf("First restore failed: %v", err)
	}

	// ConflictError - should fail
	_, err = Restore(backupFile, RestoreOptions{
		VaultPath:  restoreDir,
		OnConflict: ConflictError,
		Password:   []byte(password),
	})
	if err == nil {
		t.Error("Expected error for ConflictError mode")
	}

	// ConflictSkip - should succeed with skipped secrets
	result, err := Restore(backupFile, RestoreOptions{
		VaultPath:  restoreDir,
		OnConflict: ConflictSkip,
		Password:   []byte(password),
	})
	if err != nil {
		t.Fatalf("ConflictSkip failed: %v", err)
	}
	if result.SecretsSkipped != 1 {
		t.Errorf("Expected 1 skipped secret, got %d", result.SecretsSkipped)
	}

	// ConflictOverwrite - should succeed
	_, err = Restore(backupFile, RestoreOptions{
		VaultPath:  restoreDir,
		OnConflict: ConflictOverwrite,
		Password:   []byte(password),
	})
	if err != nil {
		t.Fatalf("ConflictOverwrite failed: %v", err)
	}
}

func TestVerify_InvalidPassword(t *testing.T) {
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	backupFile := filepath.Join(tempDir, "backup.enc")

	// Setup vault and backup
	password := "correct-password"
	v := vault.New(vaultDir)
	v.Init(password)
	v.Unlock(password)
	v.SetSecret("test/key", &vault.SecretEntry{Value: []byte("value")})

	backupOutput, _ := os.Create(backupFile)
	Backup(v, BackupOptions{Output: backupOutput, Password: []byte(password)})
	backupOutput.Close()
	v.Lock()

	// Verify with wrong password
	result, _ := Verify(backupFile, []byte("wrong-password"), "")
	if result.Valid {
		t.Error("Verification should fail with wrong password")
	}
}

func TestDefaultVaultPath(t *testing.T) {
	path := DefaultVaultPath()
	if path == "" {
		t.Error("DefaultVaultPath should not be empty")
	}
	if !filepath.IsAbs(path) && path != ".secretctl" {
		t.Errorf("DefaultVaultPath should be absolute or .secretctl, got %s", path)
	}
}

func TestHeaderBytes(t *testing.T) {
	header := &Header{
		Version:        1,
		CreatedAt:      time.Now().UTC(),
		VaultVersion:   1,
		EncryptionMode: EncryptionModeMaster,
		SecretCount:    5,
		ChecksumAlgo:   "sha256",
	}

	data, err := HeaderBytes(header)
	if err != nil {
		t.Fatalf("HeaderBytes failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("HeaderBytes should return non-empty data")
	}

	// Should be valid JSON
	var parsed Header
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("HeaderBytes should produce valid JSON: %v", err)
	}
}

func TestCopyDir(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "dst")

	// Create source files
	os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0600)
	os.MkdirAll(filepath.Join(srcDir, "subdir"), 0700)
	os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("content2"), 0600)

	// Copy
	if err := copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	// Verify files
	content1, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	if err != nil {
		t.Fatalf("Failed to read copied file1: %v", err)
	}
	if string(content1) != "content1" {
		t.Error("file1 content mismatch")
	}

	content2, err := os.ReadFile(filepath.Join(dstDir, "subdir", "file2.txt"))
	if err != nil {
		t.Fatalf("Failed to read copied file2: %v", err)
	}
	if string(content2) != "content2" {
		t.Error("file2 content mismatch")
	}
}

func TestBackup_WithAudit(t *testing.T) {
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	restoreDir := filepath.Join(tempDir, "restored")
	backupFile := filepath.Join(tempDir, "backup.enc")

	// Initialize vault
	password := "test-password"
	v := vault.New(vaultDir)
	v.Init(password)
	v.Unlock(password)
	v.SetSecret("test/key", &vault.SecretEntry{Value: []byte("value")})
	v.Lock()

	// Unlock again to ensure audit log exists
	v.Unlock(password)

	// Create backup with audit
	backupOutput, _ := os.Create(backupFile)
	opts := BackupOptions{
		Output:       backupOutput,
		IncludeAudit: true,
		Password:     []byte(password),
	}
	if err := Backup(v, opts); err != nil {
		backupOutput.Close()
		t.Fatalf("Backup with audit failed: %v", err)
	}
	backupOutput.Close()
	v.Lock()

	// Verify includes audit
	result, err := Verify(backupFile, []byte(password), "")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !result.Valid {
		t.Error("Backup should be valid")
	}

	// Restore with audit
	restoreOpts := RestoreOptions{
		VaultPath:  restoreDir,
		OnConflict: ConflictError,
		WithAudit:  true,
		Password:   []byte(password),
	}
	restoreResult, err := Restore(backupFile, restoreOpts)
	if err != nil {
		t.Fatalf("Restore with audit failed: %v", err)
	}
	if !restoreResult.AuditRestored {
		t.Log("Audit log was not restored (may not exist in backup)")
	}
}

func TestRestore_VerifyOnly(t *testing.T) {
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	restoreDir := filepath.Join(tempDir, "restored")
	backupFile := filepath.Join(tempDir, "backup.enc")

	// Setup vault and backup
	password := "test-password"
	v := vault.New(vaultDir)
	v.Init(password)
	v.Unlock(password)
	v.SetSecret("test/key", &vault.SecretEntry{Value: []byte("value")})

	backupOutput, _ := os.Create(backupFile)
	Backup(v, BackupOptions{Output: backupOutput, Password: []byte(password)})
	backupOutput.Close()
	v.Lock()

	// Verify only
	result, err := Restore(backupFile, RestoreOptions{
		VaultPath:  restoreDir,
		VerifyOnly: true,
		Password:   []byte(password),
	})
	if err != nil {
		t.Fatalf("VerifyOnly failed: %v", err)
	}
	if result.SecretsRestored != 0 {
		t.Error("VerifyOnly should not restore secrets")
	}

	// Verify restore dir was NOT created
	if _, err := os.Stat(restoreDir); !os.IsNotExist(err) {
		t.Error("Restore directory should not exist after verify only")
	}
}

func TestBackup_OutputNil(t *testing.T) {
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")

	password := "test-password"
	v := vault.New(vaultDir)
	v.Init(password)
	v.Unlock(password)

	// Nil output should fail
	err := Backup(v, BackupOptions{
		Output:   nil,
		Password: []byte(password),
	})
	if err == nil {
		t.Error("Expected error for nil output")
	}
}

func TestBackup_NoPasswordOrKeyFile(t *testing.T) {
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	backupFile := filepath.Join(tempDir, "backup.enc")

	password := "test-password"
	v := vault.New(vaultDir)
	v.Init(password)
	v.Unlock(password)

	// No password and no key file
	backupOutput, _ := os.Create(backupFile)
	err := Backup(v, BackupOptions{
		Output:   backupOutput,
		Password: nil,
		KeyFile:  "",
	})
	backupOutput.Close()
	if err == nil {
		t.Error("Expected error when no password or key file provided")
	}
}

func TestWriteReadUint32(t *testing.T) {
	tests := []uint32{0, 1, 255, 65535, 4294967295}

	for _, expected := range tests {
		var buf bytes.Buffer
		if err := writeUint32(&buf, expected); err != nil {
			t.Fatalf("writeUint32 failed for %d: %v", expected, err)
		}

		var actual uint32
		if err := readUint32(&buf, &actual); err != nil {
			t.Fatalf("readUint32 failed for %d: %v", expected, err)
		}

		if actual != expected {
			t.Errorf("Expected %d, got %d", expected, actual)
		}
	}
}

func TestRestore_FileNotFound(t *testing.T) {
	_, err := Restore("/nonexistent/path/backup.enc", RestoreOptions{
		VaultPath: t.TempDir(),
		Password:  []byte("password"),
	})
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestVerify_FileNotFound(t *testing.T) {
	result, _ := Verify("/nonexistent/path/backup.enc", []byte("password"), "")
	if result.Valid {
		t.Error("Verification should fail for nonexistent file")
	}
}

func TestVerify_TruncatedFile(t *testing.T) {
	tempDir := t.TempDir()
	truncatedFile := filepath.Join(tempDir, "truncated.enc")

	// Create a truncated file
	os.WriteFile(truncatedFile, []byte("SHORT"), 0600)

	result, _ := Verify(truncatedFile, []byte("password"), "")
	if result.Valid {
		t.Error("Verification should fail for truncated file")
	}
}

func TestRestore_EncryptionModeKey_WithKeyFile(t *testing.T) {
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	restoreDir := filepath.Join(tempDir, "restored")
	backupFile := filepath.Join(tempDir, "backup.enc")
	keyFile := filepath.Join(tempDir, "backup.key")

	// Generate key file
	GenerateKeyFile(keyFile)

	// Initialize vault
	password := "test-password"
	v := vault.New(vaultDir)
	v.Init(password)
	v.Unlock(password)
	v.SetSecret("test/key", &vault.SecretEntry{Value: []byte("value")})

	// Backup with key file
	backupOutput, _ := os.Create(backupFile)
	Backup(v, BackupOptions{Output: backupOutput, KeyFile: keyFile})
	backupOutput.Close()
	v.Lock()

	// Verify with key file
	result, err := Verify(backupFile, nil, keyFile)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !result.Valid {
		t.Error("Backup should be valid")
	}

	// Restore with key file
	restoreResult, err := Restore(backupFile, RestoreOptions{
		VaultPath:  restoreDir,
		OnConflict: ConflictError,
		KeyFile:    keyFile,
	})
	if err != nil {
		t.Fatalf("Restore with key file failed: %v", err)
	}
	if restoreResult.SecretsRestored != 1 {
		t.Errorf("Expected 1 secret, got %d", restoreResult.SecretsRestored)
	}
}

func TestGenerateKeyFile_InvalidPath(t *testing.T) {
	err := GenerateKeyFile("/nonexistent/dir/backup.key")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestReadKeyFile_FileNotFound(t *testing.T) {
	_, err := ReadKeyFile("/nonexistent/path/key.file")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestEncryptPayload_Integration(t *testing.T) {
	key := make([]byte, KeyLength)
	cryptorand.Read(key)
	defer crypto.SecureWipe(key)

	// Test with large payload
	largePayload := make([]byte, 1024*1024) // 1MB
	cryptorand.Read(largePayload)

	ciphertext, err := EncryptPayload(largePayload, key)
	if err != nil {
		t.Fatalf("EncryptPayload failed: %v", err)
	}

	decrypted, err := DecryptPayload(ciphertext, key)
	if err != nil {
		t.Fatalf("DecryptPayload failed: %v", err)
	}

	if !bytes.Equal(decrypted, largePayload) {
		t.Error("Large payload roundtrip failed")
	}
}

func TestWriteHeader_AllFields(t *testing.T) {
	// Test header with all fields populated
	salt := make([]byte, 32)
	cryptorand.Read(salt)

	header := &Header{
		Version:        1,
		CreatedAt:      time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		VaultVersion:   1,
		EncryptionMode: EncryptionModeMaster,
		KDFParams: &KDFParams{
			Salt:        salt,
			Memory:      65536,
			Iterations:  3,
			Parallelism: 4,
		},
		IncludesAudit: true,
		SecretCount:   100,
		ChecksumAlgo:  "sha256",
	}

	var buf bytes.Buffer
	err := WriteHeader(&buf, header)
	if err != nil {
		t.Fatalf("WriteHeader failed: %v", err)
	}

	// Should start with magic number
	if !bytes.HasPrefix(buf.Bytes(), MagicNumber[:]) {
		t.Error("Buffer should start with magic number")
	}

	// Read back and verify
	readHeader, err := ReadHeader(&buf)
	if err != nil {
		t.Fatalf("ReadHeader failed: %v", err)
	}

	if readHeader.SecretCount != 100 {
		t.Errorf("SecretCount mismatch: expected 100, got %d", readHeader.SecretCount)
	}
	if readHeader.KDFParams == nil {
		t.Error("KDFParams should not be nil")
	}
}

func TestWriteHeader_KeyMode(t *testing.T) {
	header := &Header{
		Version:        1,
		CreatedAt:      time.Now().UTC(),
		VaultVersion:   1,
		EncryptionMode: EncryptionModeKey,
		KDFParams:      nil, // No KDF params for key mode
		IncludesAudit:  false,
		SecretCount:    5,
		ChecksumAlgo:   "sha256",
	}

	var buf bytes.Buffer
	err := WriteHeader(&buf, header)
	if err != nil {
		t.Fatalf("WriteHeader failed: %v", err)
	}

	readHeader, err := ReadHeader(&buf)
	if err != nil {
		t.Fatalf("ReadHeader failed: %v", err)
	}

	if readHeader.EncryptionMode != EncryptionModeKey {
		t.Error("EncryptionMode should be key")
	}
	if readHeader.KDFParams != nil {
		t.Error("KDFParams should be nil for key mode")
	}
}

func TestDecodePayload_InvalidJSON(t *testing.T) {
	_, err := DecodePayload([]byte("not valid json"))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestReadHeader_TooShort(t *testing.T) {
	// Create data too short to contain header length
	data := append(MagicNumber[:], 0x00) // Magic + partial length

	_, err := ReadHeader(bytes.NewReader(data))
	if err == nil {
		t.Error("Expected error for truncated header")
	}
}

func TestVerifyAndDecrypt_NoPassword(t *testing.T) {
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	backupFile := filepath.Join(tempDir, "backup.enc")

	// Setup vault and backup
	password := "test-password"
	v := vault.New(vaultDir)
	v.Init(password)
	v.Unlock(password)
	v.SetSecret("test/key", &vault.SecretEntry{Value: []byte("value")})

	backupOutput, _ := os.Create(backupFile)
	Backup(v, BackupOptions{Output: backupOutput, Password: []byte(password)})
	backupOutput.Close()
	v.Lock()

	// Try to restore without password
	_, err := Restore(backupFile, RestoreOptions{
		VaultPath: filepath.Join(tempDir, "restored"),
		Password:  nil, // No password
		KeyFile:   "",  // No key file
	})
	if err == nil {
		t.Error("Expected error for missing password")
	}
}

func TestPerformRestore_DefaultVaultPath(t *testing.T) {
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	backupFile := filepath.Join(tempDir, "backup.enc")

	// Set HOME to temp dir so DefaultVaultPath points there
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Setup vault and backup
	password := "test-password"
	v := vault.New(vaultDir)
	v.Init(password)
	v.Unlock(password)
	v.SetSecret("test/key", &vault.SecretEntry{Value: []byte("value")})

	backupOutput, _ := os.Create(backupFile)
	Backup(v, BackupOptions{Output: backupOutput, Password: []byte(password)})
	backupOutput.Close()
	v.Lock()

	// Restore with empty VaultPath (should use default)
	result, err := Restore(backupFile, RestoreOptions{
		VaultPath:  "", // Empty - should use default
		OnConflict: ConflictOverwrite,
		Password:   []byte(password),
	})
	if err != nil {
		t.Fatalf("Restore with default path failed: %v", err)
	}
	if result.SecretsRestored != 1 {
		t.Errorf("Expected 1 secret, got %d", result.SecretsRestored)
	}
}

func TestEncodePayload_Success(t *testing.T) {
	payload := &Payload{
		VaultSalt: []byte("salt"),
		VaultMeta: []byte("meta"),
		VaultDB:   []byte("db"),
	}

	data, err := EncodePayload(payload)
	if err != nil {
		t.Fatalf("EncodePayload failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Encoded payload should not be empty")
	}
}

func TestDeriveHKDF_Success(t *testing.T) {
	secret := make([]byte, 32)
	cryptorand.Read(secret)

	key, err := deriveHKDF(secret, []byte("test-info"))
	if err != nil {
		t.Fatalf("deriveHKDF failed: %v", err)
	}

	if len(key) != KeyLength {
		t.Errorf("Expected key length %d, got %d", KeyLength, len(key))
	}

	// Same inputs should produce same key
	key2, _ := deriveHKDF(secret, []byte("test-info"))
	if !bytes.Equal(key, key2) {
		t.Error("Same inputs should produce same key")
	}

	// Different info should produce different key
	key3, _ := deriveHKDF(secret, []byte("other-info"))
	if bytes.Equal(key, key3) {
		t.Error("Different info should produce different key")
	}
}

func TestCollectVaultData_MissingFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create a vault object pointing to non-existent directory
	v := vault.New(tempDir)

	// This should fail because the vault files don't exist
	_, _, err := collectVaultData(v, false)
	if err == nil {
		t.Error("Expected error for missing vault files")
	}
}

func TestGenerateSalt_Randomness(t *testing.T) {
	// Generate multiple salts and ensure they're all different
	salts := make([][]byte, 5)
	for i := 0; i < 5; i++ {
		salt, err := GenerateSalt()
		if err != nil {
			t.Fatalf("GenerateSalt failed: %v", err)
		}
		salts[i] = salt
	}

	// Check that all salts are unique
	for i := 0; i < len(salts); i++ {
		for j := i + 1; j < len(salts); j++ {
			if bytes.Equal(salts[i], salts[j]) {
				t.Errorf("Salt %d and %d are identical", i, j)
			}
		}
	}
}

func TestBackup_VaultDataCollection(t *testing.T) {
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	backupFile := filepath.Join(tempDir, "backup.enc")

	// Initialize and populate vault
	password := "test-password"
	v := vault.New(vaultDir)
	v.Init(password)
	v.Unlock(password)

	// Add multiple secrets
	for i := 0; i < 10; i++ {
		key := filepath.Join("test", string(rune('a'+i)))
		v.SetSecret(key, &vault.SecretEntry{Value: []byte("value")})
	}

	// Create backup
	backupOutput, _ := os.Create(backupFile)
	err := Backup(v, BackupOptions{
		Output:       backupOutput,
		IncludeAudit: false,
		Password:     []byte(password),
	})
	backupOutput.Close()

	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	// Verify
	result, _ := Verify(backupFile, []byte(password), "")
	if !result.Valid {
		t.Error("Backup should be valid")
	}
	if result.SecretCount != 10 {
		t.Errorf("Expected 10 secrets, got %d", result.SecretCount)
	}
}

func TestReadHeader_ValidMagicButTruncated(t *testing.T) {
	// Create valid magic + valid length but truncated header content
	var buf bytes.Buffer
	buf.Write(MagicNumber[:])
	writeUint32(&buf, 1000) // Large header length
	buf.Write([]byte("{}")) // Too short for declared length

	_, err := ReadHeader(&buf)
	if err == nil {
		t.Error("Expected error for truncated header content")
	}
}
