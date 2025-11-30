// Package audit provides audit logging with HMAC chain for tamper detection.
package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(tmpDir)

	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
	if logger.path != tmpDir {
		t.Errorf("expected path %s, got %s", tmpDir, logger.path)
	}
	if logger.prevHash != "genesis" {
		t.Errorf("expected prevHash 'genesis', got %s", logger.prevHash)
	}
	if logger.sessionID == "" {
		t.Error("expected non-empty sessionID")
	}
}

func TestSetHMACKey(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(tmpDir)

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	err := logger.SetHMACKey(masterKey)
	if err != nil {
		t.Fatalf("SetHMACKey failed: %v", err)
	}

	if !logger.hmacKeySet {
		t.Error("expected hmacKeySet to be true")
	}
	if len(logger.hmacKey) != 32 {
		t.Errorf("expected hmacKey length 32, got %d", len(logger.hmacKey))
	}
}

func TestLogWithoutHMACKey(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(tmpDir)

	err := logger.Log(OpSecretGet, SourceCLI, ResultSuccess, "test-key", nil, nil)
	if err == nil {
		t.Error("expected error when logging without HMAC key")
	}
}

func TestLogSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(tmpDir)

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}
	if err := logger.SetHMACKey(masterKey); err != nil {
		t.Fatalf("SetHMACKey failed: %v", err)
	}

	err := logger.LogSuccess(OpSecretGet, SourceCLI, "test-key")
	if err != nil {
		t.Fatalf("LogSuccess failed: %v", err)
	}

	// Verify log file was created
	files, err := filepath.Glob(filepath.Join(tmpDir, "*.jsonl"))
	if err != nil {
		t.Fatalf("failed to list log files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 log file, got %d", len(files))
	}

	// Read and parse the log entry
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(data[:len(data)-1], &event); err != nil { // -1 to remove trailing newline
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if event.Version != 1 {
		t.Errorf("expected version 1, got %d", event.Version)
	}
	if event.Operation != OpSecretGet {
		t.Errorf("expected operation %s, got %s", OpSecretGet, event.Operation)
	}
	if event.Result != ResultSuccess {
		t.Errorf("expected result %s, got %s", ResultSuccess, event.Result)
	}
	if event.Actor.Source != SourceCLI {
		t.Errorf("expected source %s, got %s", SourceCLI, event.Actor.Source)
	}
	if event.Chain.Sequence != 1 {
		t.Errorf("expected sequence 1, got %d", event.Chain.Sequence)
	}
	if event.Chain.PrevHash != "genesis" {
		t.Errorf("expected prevHash 'genesis', got %s", event.Chain.PrevHash)
	}
	if event.Chain.HMAC == "" {
		t.Error("expected non-empty HMAC")
	}
}

func TestLogError(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(tmpDir)

	masterKey := make([]byte, 32)
	if err := logger.SetHMACKey(masterKey); err != nil {
		t.Fatalf("SetHMACKey failed: %v", err)
	}

	err := logger.LogError(OpVaultUnlockFailed, SourceCLI, "", "AUTH_FAILED", "invalid password")
	if err != nil {
		t.Fatalf("LogError failed: %v", err)
	}

	// Verify log file content
	files, _ := filepath.Glob(filepath.Join(tmpDir, "*.jsonl"))
	data, _ := os.ReadFile(files[0])

	var event AuditEvent
	json.Unmarshal(data[:len(data)-1], &event)

	if event.Result != ResultError {
		t.Errorf("expected result %s, got %s", ResultError, event.Result)
	}
	if event.Error == nil {
		t.Error("expected error info to be set")
	} else {
		if event.Error.Code != "AUTH_FAILED" {
			t.Errorf("expected error code AUTH_FAILED, got %s", event.Error.Code)
		}
		if event.Error.Message != "invalid password" {
			t.Errorf("expected error message 'invalid password', got %s", event.Error.Message)
		}
	}
}

func TestLogDenied(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(tmpDir)

	masterKey := make([]byte, 32)
	if err := logger.SetHMACKey(masterKey); err != nil {
		t.Fatalf("SetHMACKey failed: %v", err)
	}

	err := logger.LogDenied(OpSecretRunDenied, SourceMCP, "API_KEY", "policy violation")
	if err != nil {
		t.Fatalf("LogDenied failed: %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(tmpDir, "*.jsonl"))
	data, _ := os.ReadFile(files[0])

	var event AuditEvent
	json.Unmarshal(data[:len(data)-1], &event)

	if event.Result != ResultDenied {
		t.Errorf("expected result %s, got %s", ResultDenied, event.Result)
	}
	if event.Context == nil {
		t.Error("expected context to be set")
	} else if event.Context["reason"] != "policy violation" {
		t.Errorf("expected reason 'policy violation', got %v", event.Context["reason"])
	}
}

func TestChainIntegrity(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(tmpDir)

	masterKey := make([]byte, 32)
	if err := logger.SetHMACKey(masterKey); err != nil {
		t.Fatalf("SetHMACKey failed: %v", err)
	}

	// Log multiple events
	for i := 0; i < 5; i++ {
		if err := logger.LogSuccess(OpSecretGet, SourceCLI, "test-key"); err != nil {
			t.Fatalf("LogSuccess failed on iteration %d: %v", i, err)
		}
	}

	// Verify chain
	result, err := logger.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected valid chain, got errors: %v", result.Errors)
	}
	if result.RecordsTotal != 5 {
		t.Errorf("expected 5 records, got %d", result.RecordsTotal)
	}
	if result.RecordsVerified != 5 {
		t.Errorf("expected 5 verified records, got %d", result.RecordsVerified)
	}
}

func TestChainPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	masterKey := make([]byte, 32)

	// First session: log some events
	logger1 := NewLogger(tmpDir)
	if err := logger1.SetHMACKey(masterKey); err != nil {
		t.Fatalf("SetHMACKey failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := logger1.LogSuccess(OpSecretSet, SourceCLI, "key1"); err != nil {
			t.Fatalf("LogSuccess failed: %v", err)
		}
	}

	// Second session: continue the chain
	logger2 := NewLogger(tmpDir)
	if err := logger2.SetHMACKey(masterKey); err != nil {
		t.Fatalf("SetHMACKey failed: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := logger2.LogSuccess(OpSecretGet, SourceCLI, "key2"); err != nil {
			t.Fatalf("LogSuccess failed: %v", err)
		}
	}

	// Verify entire chain
	result, err := logger2.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected valid chain after session resume, got errors: %v", result.Errors)
	}
	if result.RecordsTotal != 5 {
		t.Errorf("expected 5 total records, got %d", result.RecordsTotal)
	}
}

func TestGenerateULID(t *testing.T) {
	id1 := generateULID()
	id2 := generateULID()

	if id1 == "" {
		t.Error("expected non-empty ULID")
	}
	if len(id1) != 32 { // 16 bytes * 2 (hex encoding)
		t.Errorf("expected ULID length 32, got %d", len(id1))
	}
	if id1 == id2 {
		t.Error("expected unique ULIDs")
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()

	if id1 == "" {
		t.Error("expected non-empty session ID")
	}
	if id1 == id2 {
		t.Error("expected unique session IDs")
	}
}

// TestTamperingDetection tests that the HMAC chain detects various forms of tampering.
// This addresses security-design-ja.md §8 regarding audit log integrity.
func TestTamperingDetection(t *testing.T) {
	t.Run("detect modified record", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger := NewLogger(tmpDir)

		masterKey := make([]byte, 32)
		if err := logger.SetHMACKey(masterKey); err != nil {
			t.Fatalf("SetHMACKey failed: %v", err)
		}

		// Log some events
		for i := 0; i < 3; i++ {
			if err := logger.LogSuccess(OpSecretGet, SourceCLI, "test-key"); err != nil {
				t.Fatalf("LogSuccess failed: %v", err)
			}
		}

		// Verify chain is initially valid
		result, err := logger.Verify()
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		if !result.Valid {
			t.Fatalf("expected valid chain before tampering: %v", result.Errors)
		}

		// Find and tamper with the log file
		files, _ := filepath.Glob(filepath.Join(tmpDir, "*.jsonl"))
		if len(files) == 0 {
			t.Fatal("no log files found")
		}

		data, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		// Modify one of the records (change operation)
		tampered := []byte(string(data))
		// Replace "secret.get" with "secret.set" in one record
		for i := 0; i < len(tampered)-10; i++ {
			if string(tampered[i:i+10]) == "secret.get" {
				copy(tampered[i:i+10], "secret.set")
				break
			}
		}

		if err := os.WriteFile(files[0], tampered, 0600); err != nil {
			t.Fatalf("failed to write tampered file: %v", err)
		}

		// Verify should detect tampering
		logger2 := NewLogger(tmpDir)
		if err := logger2.SetHMACKey(masterKey); err != nil {
			t.Fatalf("SetHMACKey failed: %v", err)
		}

		result, err = logger2.Verify()
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid chain after tampering, but verification passed")
		}
		if len(result.Errors) == 0 {
			t.Error("expected errors to be reported")
		}
	})

	t.Run("detect deleted record (chain break)", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger := NewLogger(tmpDir)

		masterKey := make([]byte, 32)
		if err := logger.SetHMACKey(masterKey); err != nil {
			t.Fatalf("SetHMACKey failed: %v", err)
		}

		// Log multiple events
		for i := 0; i < 5; i++ {
			if err := logger.LogSuccess(OpSecretGet, SourceCLI, "test-key"); err != nil {
				t.Fatalf("LogSuccess failed: %v", err)
			}
		}

		// Find and modify the log file to remove a record
		files, _ := filepath.Glob(filepath.Join(tmpDir, "*.jsonl"))
		data, _ := os.ReadFile(files[0])

		// Split into lines and remove the middle one
		lines := []byte{}
		lineCount := 0
		start := 0
		for i := 0; i < len(data); i++ {
			if data[i] == '\n' {
				lineCount++
				if lineCount != 3 { // Skip line 3
					lines = append(lines, data[start:i+1]...)
				}
				start = i + 1
			}
		}

		if err := os.WriteFile(files[0], lines, 0600); err != nil {
			t.Fatalf("failed to write modified file: %v", err)
		}

		// Verify should detect the broken chain
		logger2 := NewLogger(tmpDir)
		if err := logger2.SetHMACKey(masterKey); err != nil {
			t.Fatalf("SetHMACKey failed: %v", err)
		}

		result, err := logger2.Verify()
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid chain after record deletion")
		}
	})

	t.Run("detect wrong HMAC key", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger := NewLogger(tmpDir)

		masterKey := make([]byte, 32)
		for i := range masterKey {
			masterKey[i] = byte(i)
		}
		if err := logger.SetHMACKey(masterKey); err != nil {
			t.Fatalf("SetHMACKey failed: %v", err)
		}

		// Log events
		for i := 0; i < 3; i++ {
			if err := logger.LogSuccess(OpSecretGet, SourceCLI, "test-key"); err != nil {
				t.Fatalf("LogSuccess failed: %v", err)
			}
		}

		// Try to verify with a different key
		wrongKey := make([]byte, 32)
		for i := range wrongKey {
			wrongKey[i] = byte(255 - i)
		}

		logger2 := NewLogger(tmpDir)
		if err := logger2.SetHMACKey(wrongKey); err != nil {
			t.Fatalf("SetHMACKey failed: %v", err)
		}

		result, err := logger2.Verify()
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid chain with wrong HMAC key")
		}
	})

	t.Run("detect inserted record", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger := NewLogger(tmpDir)

		masterKey := make([]byte, 32)
		if err := logger.SetHMACKey(masterKey); err != nil {
			t.Fatalf("SetHMACKey failed: %v", err)
		}

		// Log events
		for i := 0; i < 3; i++ {
			if err := logger.LogSuccess(OpSecretGet, SourceCLI, "test-key"); err != nil {
				t.Fatalf("LogSuccess failed: %v", err)
			}
		}

		// Insert a fake record
		files, _ := filepath.Glob(filepath.Join(tmpDir, "*.jsonl"))
		data, _ := os.ReadFile(files[0])

		// Create a fake event JSON (with fake HMAC)
		fakeEvent := `{"v":1,"id":"fake123","ts":"2025-01-01T00:00:00Z","op":"secret.get","actor":{"type":"user","source":"cli","session_id":"fake"},"result":"success","chain":{"seq":999,"prev":"fake_prev","hmac":"fake_hmac"}}` + "\n"

		// Insert after first line
		lines := []byte{}
		firstNewline := 0
		for i := 0; i < len(data); i++ {
			if data[i] == '\n' {
				firstNewline = i + 1
				break
			}
		}
		lines = append(lines, data[:firstNewline]...)
		lines = append(lines, []byte(fakeEvent)...)
		lines = append(lines, data[firstNewline:]...)

		if err := os.WriteFile(files[0], lines, 0600); err != nil {
			t.Fatalf("failed to write modified file: %v", err)
		}

		// Verify should detect the invalid chain
		logger2 := NewLogger(tmpDir)
		if err := logger2.SetHMACKey(masterKey); err != nil {
			t.Fatalf("SetHMACKey failed: %v", err)
		}

		result, err := logger2.Verify()
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid chain after record insertion")
		}
	})
}

// TestVerifyEmptyLog tests verification behavior with no records
func TestVerifyEmptyLog(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(tmpDir)

	masterKey := make([]byte, 32)
	if err := logger.SetHMACKey(masterKey); err != nil {
		t.Fatalf("SetHMACKey failed: %v", err)
	}

	// Verify with no records should pass
	result, err := logger.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid result for empty log: %v", result.Errors)
	}
	if result.RecordsTotal != 0 {
		t.Errorf("expected 0 records, got %d", result.RecordsTotal)
	}
}

// TestListEvents tests the audit log list functionality
func TestListEvents(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger(tmpDir)

	masterKey := make([]byte, 32)
	if err := logger.SetHMACKey(masterKey); err != nil {
		t.Fatalf("SetHMACKey failed: %v", err)
	}

	// Log various events
	_ = logger.LogSuccess(OpSecretSet, SourceCLI, "key1")
	_ = logger.LogSuccess(OpSecretGet, SourceMCP, "key2")
	_ = logger.LogError(OpVaultUnlockFailed, SourceCLI, "", "AUTH_FAILED", "bad password")
	_ = logger.LogDenied(OpSecretRunDenied, SourceMCP, "key3", "policy violation")
	_ = logger.LogSuccess(OpSecretDelete, SourceUI, "key4")

	// List all events (use zero time to get all events)
	var zeroTime time.Time
	events, err := logger.ListEvents(100, zeroTime)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}

	if len(events) != 5 {
		t.Errorf("expected 5 events, got %d", len(events))
	}

	// Verify event types
	operations := make(map[string]int)
	for _, e := range events {
		operations[e.Operation]++
	}

	if operations[OpSecretSet] != 1 {
		t.Errorf("expected 1 secret.set, got %d", operations[OpSecretSet])
	}
	if operations[OpSecretGet] != 1 {
		t.Errorf("expected 1 secret.get, got %d", operations[OpSecretGet])
	}
}
