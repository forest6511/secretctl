package mcp

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/forest6511/secretctl/pkg/vault"
)

// testVault creates a temporary vault for testing
func testVault(t *testing.T) (v *vault.Vault, tmpDir string) {
	t.Helper()
	tmpDir = t.TempDir()
	v = vault.New(tmpDir)
	password := "testpassword123"

	if err := v.Init(password); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}
	if err := v.Unlock(password); err != nil {
		t.Fatalf("failed to unlock vault: %v", err)
	}

	return
}

// addTestSecret adds a secret to the vault for testing
func addTestSecret(t *testing.T, v *vault.Vault, key string, value []byte) {
	t.Helper()
	entry := &vault.SecretEntry{
		Value: value,
	}
	if err := v.SetSecret(key, entry); err != nil {
		t.Fatalf("failed to add secret '%s': %v", key, err)
	}
}

// addTestSecretWithTags adds a secret with tags to the vault for testing
func addTestSecretWithTags(t *testing.T, v *vault.Vault, key string, value []byte, tags []string) {
	t.Helper()
	entry := &vault.SecretEntry{
		Value: value,
		Tags:  tags,
	}
	if err := v.SetSecret(key, entry); err != nil {
		t.Fatalf("failed to add secret '%s': %v", key, err)
	}
}

// addTestSecretWithExpiration adds a secret with expiration to the vault for testing
func addTestSecretWithExpiration(t *testing.T, v *vault.Vault, key string, value []byte, expiresAt *time.Time) {
	t.Helper()
	entry := &vault.SecretEntry{
		Value:     value,
		ExpiresAt: expiresAt,
	}
	if err := v.SetSecret(key, entry); err != nil {
		t.Fatalf("failed to add secret '%s': %v", key, err)
	}
}

// addTestSecretMultiField adds a multi-field secret to the vault for testing
func addTestSecretMultiField(t *testing.T, v *vault.Vault, key string, fields map[string]vault.Field) {
	t.Helper()
	entry := &vault.SecretEntry{
		Fields: fields,
	}
	if err := v.SetSecret(key, entry); err != nil {
		t.Fatalf("failed to add multi-field secret '%s': %v", key, err)
	}
}

// createTestPolicy creates a test policy file
func createTestPolicy(t *testing.T, vaultPath string, content string) {
	t.Helper()
	policyPath := filepath.Join(vaultPath, PolicyFileName)
	if err := os.WriteFile(policyPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to create policy file: %v", err)
	}
}

func TestNewServer_NoPassword(t *testing.T) {
	tmpDir := t.TempDir()
	v := vault.New(tmpDir)
	if err := v.Init("password123"); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	// Clear environment
	os.Unsetenv("SECRETCTL_PASSWORD")

	_, err := NewServer(&ServerOptions{
		VaultPath: tmpDir,
		Password:  "",
	})
	if err == nil {
		t.Error("expected error when no password provided")
	}
	if err.Error() != "no password provided: set SECRETCTL_PASSWORD environment variable" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewServer_InvalidPassword(t *testing.T) {
	tmpDir := t.TempDir()
	v := vault.New(tmpDir)
	if err := v.Init("correctpassword"); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	_, err := NewServer(&ServerOptions{
		VaultPath: tmpDir,
		Password:  "wrongpassword",
	})
	if err == nil {
		t.Error("expected error with invalid password")
	}
}

func TestNewServer_Success(t *testing.T) {
	tmpDir := t.TempDir()
	v := vault.New(tmpDir)
	password := "testpassword123"
	if err := v.Init(password); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	server, err := NewServer(&ServerOptions{
		VaultPath: tmpDir,
		Password:  password,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if server == nil {
		t.Fatal("server is nil")
	}
	if server.vault == nil {
		t.Error("vault is nil")
	}
	if server.server == nil {
		t.Error("mcp server is nil")
	}

	// Close should not error
	if err := server.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestNewServer_FromEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	v := vault.New(tmpDir)
	password := "envpassword123"
	if err := v.Init(password); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	// Set password via environment
	os.Setenv("SECRETCTL_PASSWORD", password)

	server, err := NewServer(&ServerOptions{
		VaultPath: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Verify env was cleared
	if os.Getenv("SECRETCTL_PASSWORD") != "" {
		t.Error("SECRETCTL_PASSWORD should be cleared after reading")
	}

	server.Close()
}

func TestNewServer_WithPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	v := vault.New(tmpDir)
	password := "testpassword123"
	if err := v.Init(password); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	// Create policy file
	policyContent := `version: 1
default_action: deny
allowed_commands:
  - aws
  - kubectl
`
	createTestPolicy(t, tmpDir, policyContent)

	server, err := NewServer(&ServerOptions{
		VaultPath: tmpDir,
		Password:  password,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if server.policy == nil {
		t.Error("policy should be loaded")
	}

	// Check policy is correct
	allowed, _ := server.policy.IsCommandAllowed("aws")
	if !allowed {
		t.Error("aws should be allowed")
	}
	allowed, _ = server.policy.IsCommandAllowed("rm")
	if allowed {
		t.Error("rm should not be allowed")
	}

	server.Close()
}

func TestServer_Close(t *testing.T) {
	tmpDir := t.TempDir()
	v := vault.New(tmpDir)
	password := "testpassword123"
	if err := v.Init(password); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	server, err := NewServer(&ServerOptions{
		VaultPath: tmpDir,
		Password:  password,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Close should lock the vault
	err = server.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	// Vault should be locked after close
	if !server.vault.IsLocked() {
		t.Error("vault should be locked after Close")
	}
}

func TestHandleSecretList_Empty(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretList(ctx, nil, SecretListInput{})
	if err != nil {
		t.Fatalf("handleSecretList failed: %v", err)
	}
	if len(output.Secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(output.Secrets))
	}
}

func TestHandleSecretList_WithSecrets(t *testing.T) {
	v, tmpDir := testVault(t)

	// Add some secrets
	addTestSecret(t, v, "api_key", []byte("secret123"))
	addTestSecret(t, v, "db_password", []byte("dbpass456"))

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretList(ctx, nil, SecretListInput{})
	if err != nil {
		t.Fatalf("handleSecretList failed: %v", err)
	}
	if len(output.Secrets) != 2 {
		t.Errorf("expected 2 secrets, got %d", len(output.Secrets))
	}

	// Verify no values are exposed
	for _, secret := range output.Secrets {
		if secret.Key == "" {
			t.Error("secret key is empty")
		}
	}
}

func TestHandleSecretList_ByTag(t *testing.T) {
	v, tmpDir := testVault(t)

	// Add secrets with tags
	addTestSecretWithTags(t, v, "api_key", []byte("secret123"), []string{"prod", "api"})
	addTestSecretWithTags(t, v, "dev_key", []byte("devpass"), []string{"dev"})

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretList(ctx, nil, SecretListInput{Tag: "prod"})
	if err != nil {
		t.Fatalf("handleSecretList failed: %v", err)
	}
	if len(output.Secrets) != 1 {
		t.Errorf("expected 1 secret with tag 'prod', got %d", len(output.Secrets))
	}
	if len(output.Secrets) > 0 && output.Secrets[0].Key != "api_key" {
		t.Errorf("expected api_key, got %s", output.Secrets[0].Key)
	}
}

func TestHandleSecretList_ExpiringWithin(t *testing.T) {
	v, tmpDir := testVault(t)

	// Add secrets with expiration
	expireTime := time.Now().Add(12 * time.Hour)
	addTestSecretWithExpiration(t, v, "expiring_key", []byte("secret123"), &expireTime)
	addTestSecret(t, v, "long_lived_key", []byte("longlived"))

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretList(ctx, nil, SecretListInput{ExpiringWithin: "1d"})
	if err != nil {
		t.Fatalf("handleSecretList failed: %v", err)
	}
	if len(output.Secrets) != 1 {
		t.Errorf("expected 1 expiring secret, got %d", len(output.Secrets))
	}
}

func TestHandleSecretList_InvalidDuration(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretList(ctx, nil, SecretListInput{ExpiringWithin: "invalid"})
	if err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestHandleSecretExists_Found(t *testing.T) {
	v, tmpDir := testVault(t)

	addTestSecret(t, v, "test_key", []byte("testvalue"))

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretExists(ctx, nil, SecretExistsInput{Key: "test_key"})
	if err != nil {
		t.Fatalf("handleSecretExists failed: %v", err)
	}
	if !output.Exists {
		t.Error("expected Exists to be true")
	}
	if output.Key != "test_key" {
		t.Errorf("expected key 'test_key', got '%s'", output.Key)
	}
}

func TestHandleSecretExists_NotFound(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretExists(ctx, nil, SecretExistsInput{Key: "nonexistent"})
	if err != nil {
		t.Fatalf("handleSecretExists failed: %v", err)
	}
	if output.Exists {
		t.Error("expected Exists to be false")
	}
}

func TestHandleSecretExists_EmptyKey(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretExists(ctx, nil, SecretExistsInput{Key: ""})
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestHandleSecretGetMasked_Success(t *testing.T) {
	v, tmpDir := testVault(t)

	secretValue := "sk-1234567890abcd"
	addTestSecret(t, v, "api_key", []byte(secretValue))

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretGetMasked(ctx, nil, SecretGetMaskedInput{Key: "api_key"})
	if err != nil {
		t.Fatalf("handleSecretGetMasked failed: %v", err)
	}
	if output.Key != "api_key" {
		t.Errorf("expected key 'api_key', got '%s'", output.Key)
	}
	if output.ValueLength != len(secretValue) {
		t.Errorf("expected value length %d, got %d", len(secretValue), output.ValueLength)
	}
	// For 17 char value (length 9+), should show last 4 characters
	// Masked: 13 asterisks + "abcd"
	expectedMasked := "*************abcd"
	if output.MaskedValue != expectedMasked {
		t.Errorf("unexpected masked value: got '%s', want '%s'", output.MaskedValue, expectedMasked)
	}
}

func TestHandleSecretGetMasked_NotFound(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretGetMasked(ctx, nil, SecretGetMaskedInput{Key: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
}

func TestHandleSecretGetMasked_EmptyKey(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretGetMasked(ctx, nil, SecretGetMaskedInput{Key: ""})
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestHandleSecretGetMasked_MultiField(t *testing.T) {
	v, tmpDir := testVault(t)

	// Add multi-field secret with sensitive and non-sensitive fields
	fields := map[string]vault.Field{
		"username": {Value: "dbadmin", Sensitive: false},
		"password": {Value: "supersecretpassword123", Sensitive: true},
		"host":     {Value: "db.example.com", Sensitive: false},
	}
	addTestSecretMultiField(t, v, "db/postgres", fields)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretGetMasked(ctx, nil, SecretGetMaskedInput{Key: "db/postgres"})
	if err != nil {
		t.Fatalf("handleSecretGetMasked failed: %v", err)
	}

	// Verify basic output
	if output.Key != "db/postgres" {
		t.Errorf("expected key 'db/postgres', got '%s'", output.Key)
	}
	if output.FieldCount != 3 {
		t.Errorf("expected field_count 3, got %d", output.FieldCount)
	}

	// For multi-field secrets, MaskedValue and ValueLength should be empty/0
	// (they are only used for backward compatibility with single-value secrets)
	if output.MaskedValue != "" {
		t.Errorf("multi-field secrets should have empty MaskedValue, got '%s'", output.MaskedValue)
	}
	if output.ValueLength != 0 {
		t.Errorf("multi-field secrets should have ValueLength 0, got %d", output.ValueLength)
	}

	// Verify fields map is populated
	if output.Fields == nil {
		t.Fatal("expected Fields map to be populated for multi-field secret")
	}
	if len(output.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(output.Fields))
	}

	// Verify non-sensitive field shows full value
	usernameField, ok := output.Fields["username"]
	if !ok {
		t.Fatal("expected 'username' field in output")
	}
	if usernameField.Value != "dbadmin" {
		t.Errorf("non-sensitive field should show full value, got '%s'", usernameField.Value)
	}
	if usernameField.Sensitive != false {
		t.Error("expected username.sensitive to be false")
	}

	// Verify sensitive field is masked
	passwordField, ok := output.Fields["password"]
	if !ok {
		t.Fatal("expected 'password' field in output")
	}
	if passwordField.Value == "supersecretpassword123" {
		t.Error("sensitive field should be masked")
	}
	if passwordField.Sensitive != true {
		t.Error("expected password.sensitive to be true")
	}
	if passwordField.ValueLength != 22 {
		t.Errorf("expected password value_length 22, got %d", passwordField.ValueLength)
	}
	// Password is 22 chars, should show last 4
	expectedMasked := "******************d123"
	if passwordField.Value != expectedMasked {
		t.Errorf("expected masked password '%s', got '%s'", expectedMasked, passwordField.Value)
	}

	// Verify host field (non-sensitive)
	hostField, ok := output.Fields["host"]
	if !ok {
		t.Fatal("expected 'host' field in output")
	}
	if hostField.Value != "db.example.com" {
		t.Errorf("non-sensitive host field should show full value, got '%s'", hostField.Value)
	}
}

func TestHandleSecretGetMasked_MaskingBoundaries(t *testing.T) {
	v, tmpDir := testVault(t)

	// Test masking boundary cases with multi-field secret
	fields := map[string]vault.Field{
		"short":  {Value: "1234", Sensitive: true},      // 4 chars -> all asterisks
		"medium": {Value: "12345678", Sensitive: true},  // 8 chars -> last 2 visible
		"long":   {Value: "123456789", Sensitive: true}, // 9 chars -> last 4 visible
	}
	addTestSecretMultiField(t, v, "test/boundaries", fields)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretGetMasked(ctx, nil, SecretGetMaskedInput{Key: "test/boundaries"})
	if err != nil {
		t.Fatalf("handleSecretGetMasked failed: %v", err)
	}

	// Test 1-4 chars: all asterisks
	shortField := output.Fields["short"]
	if shortField.Value != "****" {
		t.Errorf("1-4 char masking: expected '****', got '%s'", shortField.Value)
	}

	// Test 5-8 chars: last 2 visible
	mediumField := output.Fields["medium"]
	if mediumField.Value != "******78" {
		t.Errorf("5-8 char masking: expected '******78', got '%s'", mediumField.Value)
	}

	// Test 9+ chars: last 4 visible
	longField := output.Fields["long"]
	if longField.Value != "*****6789" {
		t.Errorf("9+ char masking: expected '*****6789', got '%s'", longField.Value)
	}
}

func TestHandleSecretRun_NoPolicy(t *testing.T) {
	v, tmpDir := testVault(t)

	addTestSecret(t, v, "api_key", []byte("secret123"))

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    nil, // No policy
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretRun(ctx, nil, &SecretRunInput{
		Keys:    []string{"api_key"},
		Command: "echo",
		Args:    []string{"hello"},
	})
	if err == nil {
		t.Error("expected error when no policy configured")
	}
	if err.Error() != "MCP policy not configured. Create ~/.secretctl/mcp-policy.yaml to enable secret_run" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleSecretRun_CommandNotAllowed(t *testing.T) {
	v, tmpDir := testVault(t)

	addTestSecret(t, v, "api_key", []byte("secret123"))

	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionDeny,
		AllowedCommands: []string{"aws"},
	}

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretRun(ctx, nil, &SecretRunInput{
		Keys:    []string{"api_key"},
		Command: "rm", // Not allowed
		Args:    []string{"-rf", "/"},
	})
	if err == nil {
		t.Error("expected error for disallowed command")
	}
}

func TestHandleSecretRun_DefaultDeniedCommand(t *testing.T) {
	v, tmpDir := testVault(t)

	addTestSecret(t, v, "api_key", []byte("secret123"))

	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionAllow,     // Even with allow-all
		AllowedCommands: []string{"env"}, // And explicitly allowed
	}

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretRun(ctx, nil, &SecretRunInput{
		Keys:    []string{"api_key"},
		Command: "env", // Default denied
	})
	if err == nil {
		t.Error("expected error for default denied command")
	}
}

func TestHandleSecretRun_Validation(t *testing.T) {
	v, tmpDir := testVault(t)

	addTestSecret(t, v, "api_key", []byte("secret123"))

	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionAllow,
		AllowedCommands: []string{"echo"},
	}

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		input   *SecretRunInput
		wantErr string
	}{
		{
			name: "empty keys",
			input: &SecretRunInput{
				Keys:    []string{},
				Command: "echo",
			},
			wantErr: "keys is required",
		},
		{
			name: "empty command",
			input: &SecretRunInput{
				Keys:    []string{"api_key"},
				Command: "",
			},
			wantErr: "command is required",
		},
		{
			name: "too many keys",
			input: &SecretRunInput{
				Keys:    []string{"k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9", "k10", "k11"},
				Command: "echo",
			},
			wantErr: "too many keys (max 10)",
		},
		{
			name: "too many args",
			input: &SecretRunInput{
				Keys:    []string{"api_key"},
				Command: "echo",
				Args:    make([]string, 101),
			},
			wantErr: "too many args (max 100)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := server.handleSecretRun(ctx, nil, tt.input)
			if err == nil {
				t.Errorf("expected error: %s", tt.wantErr)
				return
			}
			if err.Error() != tt.wantErr {
				t.Errorf("expected error '%s', got '%s'", tt.wantErr, err.Error())
			}
		})
	}
}

func TestHandleSecretRun_Success(t *testing.T) {
	v, tmpDir := testVault(t)

	addTestSecret(t, v, "api_key", []byte("secret123"))

	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionDeny,
		AllowedCommands: []string{"echo"},
	}

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretRun(ctx, nil, &SecretRunInput{
		Keys:    []string{"api_key"},
		Command: "echo",
		Args:    []string{"hello"},
	})
	if err != nil {
		t.Fatalf("handleSecretRun failed: %v", err)
	}
	if output.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", output.ExitCode)
	}
	if output.Stdout != "hello\n" {
		t.Errorf("expected stdout 'hello\\n', got '%s'", output.Stdout)
	}
	if !output.Sanitized {
		t.Error("expected Sanitized to be true")
	}
}

func TestHandleSecretRun_OutputSanitization(t *testing.T) {
	v, tmpDir := testVault(t)

	secretValue := "supersecret123"
	addTestSecret(t, v, "api_key", []byte(secretValue))

	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionDeny,
		AllowedCommands: []string{"echo"},
	}

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretRun(ctx, nil, &SecretRunInput{
		Keys:    []string{"api_key"},
		Command: "echo",
		Args:    []string{secretValue}, // Echo the secret value
	})
	if err != nil {
		t.Fatalf("handleSecretRun failed: %v", err)
	}

	// Secret should be redacted in output
	if output.Stdout == secretValue+"\n" {
		t.Error("secret should be redacted in output")
	}
	if output.Stdout != "[REDACTED:API_KEY]\n" {
		t.Errorf("expected redacted output, got '%s'", output.Stdout)
	}
}

func TestHandleSecretRun_Timeout(t *testing.T) {
	v, tmpDir := testVault(t)

	addTestSecret(t, v, "api_key", []byte("secret123"))

	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionDeny,
		AllowedCommands: []string{"sleep"},
	}

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretRun(ctx, nil, &SecretRunInput{
		Keys:    []string{"api_key"},
		Command: "sleep",
		Args:    []string{"10"},
		Timeout: "100ms",
	})
	// The command may return an error or exit with non-zero code
	// Either way, it should be killed due to timeout
	if err != nil {
		// Timeout error is expected
		return
	}
	// If no error, the exit code should be non-zero (killed by signal)
	if output.ExitCode == 0 {
		t.Error("expected non-zero exit code or timeout error")
	}
}

func TestHandleSecretRun_ConcurrencyLimit(t *testing.T) {
	v, tmpDir := testVault(t)

	addTestSecret(t, v, "api_key", []byte("secret123"))

	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionDeny,
		AllowedCommands: []string{"sleep"},
	}

	// Create server with semaphore at capacity
	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	// Fill the semaphore
	for i := 0; i < maxConcurrentRuns; i++ {
		server.runSem <- struct{}{}
	}

	ctx := context.Background()
	_, _, err := server.handleSecretRun(ctx, nil, &SecretRunInput{
		Keys:    []string{"api_key"},
		Command: "sleep",
		Args:    []string{"1"},
	})
	if err == nil {
		t.Error("expected concurrency limit error")
	}
	if err.Error() != "too many concurrent secret_run operations (max 5)" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCollectSecrets(t *testing.T) {
	v, tmpDir := testVault(t)

	addTestSecret(t, v, "aws/access_key", []byte("AKIAIOSFODNN7EXAMPLE"))
	addTestSecret(t, v, "aws/secret_key", []byte("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"))
	addTestSecret(t, v, "db/password", []byte("dbpassword"))

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	tests := []struct {
		name        string
		patterns    []string
		wantCount   int
		wantErr     bool
		errContains string
	}{
		{
			name:      "exact match",
			patterns:  []string{"aws/access_key"},
			wantCount: 1,
		},
		{
			name:      "wildcard match",
			patterns:  []string{"aws/*"},
			wantCount: 2,
		},
		{
			name:      "multiple patterns",
			patterns:  []string{"aws/*", "db/password"},
			wantCount: 3,
		},
		{
			name:        "no match",
			patterns:    []string{"nonexistent"},
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets, err := server.collectSecrets(tt.patterns)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("collectSecrets failed: %v", err)
			}
			if len(secrets) != tt.wantCount {
				t.Errorf("expected %d secrets, got %d", tt.wantCount, len(secrets))
			}
		})
	}
}

func TestBuildEnvironment(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	secrets := []secretData{
		{key: "api_key", value: []byte("secret123")},
		{key: "db/password", value: []byte("dbpass")},
	}

	env, err := server.buildEnvironment(secrets, "")
	if err != nil {
		t.Fatalf("buildEnvironment failed: %v", err)
	}

	// Check that secrets are in environment
	found := make(map[string]bool)
	for _, e := range env {
		if e == "API_KEY=secret123" {
			found["API_KEY"] = true
		}
		if e == "DB_PASSWORD=dbpass" {
			found["DB_PASSWORD"] = true
		}
	}

	if !found["API_KEY"] {
		t.Error("API_KEY not found in environment")
	}
	if !found["DB_PASSWORD"] {
		t.Error("DB_PASSWORD not found in environment")
	}
}

func TestBuildEnvironment_WithPrefix(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	secrets := []secretData{
		{key: "api_key", value: []byte("secret123")},
	}

	env, err := server.buildEnvironment(secrets, "MY_")
	if err != nil {
		t.Fatalf("buildEnvironment failed: %v", err)
	}

	found := false
	for _, e := range env {
		if e == "MY_API_KEY=secret123" {
			found = true
			break
		}
	}

	if !found {
		t.Error("MY_API_KEY not found in environment")
	}
}

func TestBuildEnvironment_BlockedEnvVar(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	secrets := []secretData{
		{key: "LD_PRELOAD", value: []byte("/evil/lib.so")},
	}

	_, err := server.buildEnvironment(secrets, "")
	if err == nil {
		t.Error("expected error for blocked env var")
	}
}

func TestBuildEnvironment_NullByte(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	secrets := []secretData{
		{key: "api_key", value: []byte("secret\x00value")},
	}

	_, err := server.buildEnvironment(secrets, "")
	if err == nil {
		t.Error("expected error for NUL byte in value")
	}
}

func TestExecuteCommand_PathTraversal(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, err := server.executeCommand(ctx, "../../../bin/sh", nil, nil, nil, time.Second)
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestExecuteCommand_NotFound(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, err := server.executeCommand(ctx, "nonexistentcommand12345", nil, nil, nil, time.Second)
	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}

// addTestMultiFieldSecret adds a multi-field secret to the vault for testing
func addTestMultiFieldSecret(t *testing.T, v *vault.Vault, key string, fields map[string]vault.Field, bindings map[string]string) {
	t.Helper()
	entry := &vault.SecretEntry{
		Fields:   fields,
		Bindings: bindings,
	}
	if err := v.SetSecret(key, entry); err != nil {
		t.Fatalf("failed to add multi-field secret '%s': %v", key, err)
	}
}

// Tests for secret_list_fields

func TestHandleSecretListFields_Success(t *testing.T) {
	v, tmpDir := testVault(t)

	// Add a multi-field secret
	fields := map[string]vault.Field{
		"username": {Value: "admin", Sensitive: false, Hint: "Database username"},
		"password": {Value: "secret123", Sensitive: true, Hint: "Database password"},
		"host":     {Value: "db.example.com", Sensitive: false, Aliases: []string{"hostname"}},
	}
	addTestMultiFieldSecret(t, v, "db_creds", fields, nil)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretListFields(ctx, nil, SecretListFieldsInput{Key: "db_creds"})
	if err != nil {
		t.Fatalf("handleSecretListFields failed: %v", err)
	}
	if output.Key != "db_creds" {
		t.Errorf("expected key 'db_creds', got '%s'", output.Key)
	}
	if len(output.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(output.Fields))
	}

	// Verify fields are sorted by name
	expectedOrder := []string{"host", "password", "username"}
	for i, expected := range expectedOrder {
		if output.Fields[i].Name != expected {
			t.Errorf("expected field[%d] to be '%s', got '%s'", i, expected, output.Fields[i].Name)
		}
	}

	// Verify no values are exposed
	for _, field := range output.Fields {
		if field.Name == "password" && !field.Sensitive {
			t.Error("password field should be marked as sensitive")
		}
		if field.Name == "host" && len(field.Aliases) != 1 {
			t.Errorf("host field should have 1 alias, got %d", len(field.Aliases))
		}
	}
}

func TestHandleSecretListFields_NotFound(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretListFields(ctx, nil, SecretListFieldsInput{Key: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
}

func TestHandleSecretListFields_EmptyKey(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretListFields(ctx, nil, SecretListFieldsInput{Key: ""})
	if err == nil {
		t.Error("expected error for empty key")
	}
}

// Tests for secret_get_field

func TestHandleSecretGetField_NonSensitive(t *testing.T) {
	v, tmpDir := testVault(t)

	fields := map[string]vault.Field{
		"host":     {Value: "db.example.com", Sensitive: false},
		"password": {Value: "secret123", Sensitive: true},
	}
	addTestMultiFieldSecret(t, v, "db_creds", fields, nil)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretGetField(ctx, nil, SecretGetFieldInput{Key: "db_creds", Field: "host"})
	if err != nil {
		t.Fatalf("handleSecretGetField failed: %v", err)
	}
	if output.Key != "db_creds" {
		t.Errorf("expected key 'db_creds', got '%s'", output.Key)
	}
	if output.Field != "host" {
		t.Errorf("expected field 'host', got '%s'", output.Field)
	}
	if output.Value != "db.example.com" {
		t.Errorf("expected value 'db.example.com', got '%s'", output.Value)
	}
	if output.Sensitive {
		t.Error("expected Sensitive to be false")
	}
}

func TestHandleSecretGetField_SensitiveRejected(t *testing.T) {
	v, tmpDir := testVault(t)

	fields := map[string]vault.Field{
		"host":     {Value: "db.example.com", Sensitive: false},
		"password": {Value: "secret123", Sensitive: true},
	}
	addTestMultiFieldSecret(t, v, "db_creds", fields, nil)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretGetField(ctx, nil, SecretGetFieldInput{Key: "db_creds", Field: "password"})
	if err == nil {
		t.Error("expected error for sensitive field")
	}
	if err.Error() != "field 'password' is marked as sensitive and cannot be retrieved via MCP (AI-Safe Access policy)" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleSecretGetField_AliasResolution(t *testing.T) {
	v, tmpDir := testVault(t)

	fields := map[string]vault.Field{
		"hostname": {Value: "db.example.com", Sensitive: false, Aliases: []string{"host", "server"}},
	}
	addTestMultiFieldSecret(t, v, "db_creds", fields, nil)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	// Request using alias "host"
	_, output, err := server.handleSecretGetField(ctx, nil, SecretGetFieldInput{Key: "db_creds", Field: "host"})
	if err != nil {
		t.Fatalf("handleSecretGetField with alias failed: %v", err)
	}
	if output.Field != "hostname" {
		t.Errorf("expected canonical field name 'hostname', got '%s'", output.Field)
	}
	if output.Value != "db.example.com" {
		t.Errorf("expected value 'db.example.com', got '%s'", output.Value)
	}
}

func TestHandleSecretGetField_FieldNotFound(t *testing.T) {
	v, tmpDir := testVault(t)

	fields := map[string]vault.Field{
		"host": {Value: "db.example.com", Sensitive: false},
	}
	addTestMultiFieldSecret(t, v, "db_creds", fields, nil)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretGetField(ctx, nil, SecretGetFieldInput{Key: "db_creds", Field: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent field")
	}
}

func TestHandleSecretGetField_EmptyInputs(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()

	// Empty key
	_, _, err := server.handleSecretGetField(ctx, nil, SecretGetFieldInput{Key: "", Field: "host"})
	if err == nil {
		t.Error("expected error for empty key")
	}

	// Empty field
	_, _, err = server.handleSecretGetField(ctx, nil, SecretGetFieldInput{Key: "db_creds", Field: ""})
	if err == nil {
		t.Error("expected error for empty field")
	}
}

// Tests for secret_run_with_bindings

func TestHandleSecretRunWithBindings_Success(t *testing.T) {
	v, tmpDir := testVault(t)

	fields := map[string]vault.Field{
		"host":     {Value: "db.example.com", Sensitive: false},
		"password": {Value: "secret123", Sensitive: true},
	}
	bindings := map[string]string{
		"DB_HOST":     "host",
		"DB_PASSWORD": "password",
	}
	addTestMultiFieldSecret(t, v, "db_creds", fields, bindings)

	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionDeny,
		AllowedCommands: []string{"echo"},
	}

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretRunWithBindings(ctx, nil, &SecretRunWithBindingsInput{
		Key:     "db_creds",
		Command: "echo",
		Args:    []string{"hello"},
	})
	if err != nil {
		t.Fatalf("handleSecretRunWithBindings failed: %v", err)
	}
	if output.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", output.ExitCode)
	}
	if output.Stdout != "hello\n" {
		t.Errorf("expected stdout 'hello\\n', got '%s'", output.Stdout)
	}
	if !output.Sanitized {
		t.Error("expected Sanitized to be true")
	}
}

func TestHandleSecretRunWithBindings_OutputSanitization(t *testing.T) {
	v, tmpDir := testVault(t)

	secretValue := "supersecret123"
	fields := map[string]vault.Field{
		"password": {Value: secretValue, Sensitive: true},
	}
	bindings := map[string]string{
		"SECRET_VALUE": "password",
	}
	addTestMultiFieldSecret(t, v, "test_secret", fields, bindings)

	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionDeny,
		AllowedCommands: []string{"echo"},
	}

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, output, err := server.handleSecretRunWithBindings(ctx, nil, &SecretRunWithBindingsInput{
		Key:     "test_secret",
		Command: "echo",
		Args:    []string{secretValue}, // Echo the secret value
	})
	if err != nil {
		t.Fatalf("handleSecretRunWithBindings failed: %v", err)
	}

	// Secret should be redacted in output
	if output.Stdout == secretValue+"\n" {
		t.Error("secret should be redacted in output")
	}
	if output.Stdout != "[REDACTED:SECRET_VALUE]\n" {
		t.Errorf("expected redacted output, got '%s'", output.Stdout)
	}
}

func TestHandleSecretRunWithBindings_NoBindings(t *testing.T) {
	v, tmpDir := testVault(t)

	fields := map[string]vault.Field{
		"host": {Value: "db.example.com", Sensitive: false},
	}
	// No bindings
	addTestMultiFieldSecret(t, v, "db_creds", fields, nil)

	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionDeny,
		AllowedCommands: []string{"echo"},
	}

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretRunWithBindings(ctx, nil, &SecretRunWithBindingsInput{
		Key:     "db_creds",
		Command: "echo",
		Args:    []string{"hello"},
	})
	if err == nil {
		t.Error("expected error when no bindings defined")
	}
	if err.Error() != "secret 'db_creds' has no bindings defined. Use 'secretctl set db_creds --binding ENV=field' to add bindings" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleSecretRunWithBindings_NoPolicy(t *testing.T) {
	v, tmpDir := testVault(t)

	fields := map[string]vault.Field{
		"host": {Value: "db.example.com", Sensitive: false},
	}
	bindings := map[string]string{
		"DB_HOST": "host",
	}
	addTestMultiFieldSecret(t, v, "db_creds", fields, bindings)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    nil, // No policy
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretRunWithBindings(ctx, nil, &SecretRunWithBindingsInput{
		Key:     "db_creds",
		Command: "echo",
		Args:    []string{"hello"},
	})
	if err == nil {
		t.Error("expected error when no policy configured")
	}
}

func TestHandleSecretRunWithBindings_ValidationErrors(t *testing.T) {
	v, tmpDir := testVault(t)

	fields := map[string]vault.Field{
		"host": {Value: "db.example.com", Sensitive: false},
	}
	bindings := map[string]string{
		"DB_HOST": "host",
	}
	addTestMultiFieldSecret(t, v, "db_creds", fields, bindings)

	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionDeny,
		AllowedCommands: []string{"echo"},
	}

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		input   *SecretRunWithBindingsInput
		wantErr string
	}{
		{
			name: "empty key",
			input: &SecretRunWithBindingsInput{
				Key:     "",
				Command: "echo",
			},
			wantErr: "key is required",
		},
		{
			name: "empty command",
			input: &SecretRunWithBindingsInput{
				Key:     "db_creds",
				Command: "",
			},
			wantErr: "command is required",
		},
		{
			name: "too many args",
			input: &SecretRunWithBindingsInput{
				Key:     "db_creds",
				Command: "echo",
				Args:    make([]string, 101),
			},
			wantErr: "too many args (max 100)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := server.handleSecretRunWithBindings(ctx, nil, tt.input)
			if err == nil {
				t.Errorf("expected error: %s", tt.wantErr)
				return
			}
			if err.Error() != tt.wantErr {
				t.Errorf("expected error '%s', got '%s'", tt.wantErr, err.Error())
			}
		})
	}
}

func TestHandleSecretRunWithBindings_ConcurrencyLimit(t *testing.T) {
	v, tmpDir := testVault(t)

	fields := map[string]vault.Field{
		"host": {Value: "db.example.com", Sensitive: false},
	}
	bindings := map[string]string{
		"DB_HOST": "host",
	}
	addTestMultiFieldSecret(t, v, "db_creds", fields, bindings)

	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionDeny,
		AllowedCommands: []string{"sleep"},
	}

	// Create server with semaphore at capacity
	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	// Fill the semaphore
	for i := 0; i < maxConcurrentRuns; i++ {
		server.runSem <- struct{}{}
	}

	ctx := context.Background()
	_, _, err := server.handleSecretRunWithBindings(ctx, nil, &SecretRunWithBindingsInput{
		Key:     "db_creds",
		Command: "sleep",
		Args:    []string{"1"},
	})
	if err == nil {
		t.Error("expected concurrency limit error")
	}
	if err.Error() != "too many concurrent secret_run operations (max 5)" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBuildEnvironmentFromBindings_Success(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	entry := &vault.SecretEntry{
		Fields: map[string]vault.Field{
			"host":     {Value: "db.example.com", Sensitive: false},
			"password": {Value: "secret123", Sensitive: true},
		},
		Bindings: map[string]string{
			"DB_HOST":     "host",
			"DB_PASSWORD": "password",
		},
	}

	env, secrets, err := server.buildEnvironmentFromBindings(entry)
	if err != nil {
		t.Fatalf("buildEnvironmentFromBindings failed: %v", err)
	}

	// Check that bindings are in environment
	found := make(map[string]bool)
	for _, e := range env {
		if e == "DB_HOST=db.example.com" {
			found["DB_HOST"] = true
		}
		if e == "DB_PASSWORD=secret123" {
			found["DB_PASSWORD"] = true
		}
	}

	if !found["DB_HOST"] {
		t.Error("DB_HOST not found in environment")
	}
	if !found["DB_PASSWORD"] {
		t.Error("DB_PASSWORD not found in environment")
	}

	// Check secrets for sanitization
	if len(secrets) != 2 {
		t.Errorf("expected 2 secrets, got %d", len(secrets))
	}
}

func TestBuildEnvironmentFromBindings_FieldNotFound(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	entry := &vault.SecretEntry{
		Fields: map[string]vault.Field{
			"host": {Value: "db.example.com", Sensitive: false},
		},
		Bindings: map[string]string{
			"DB_HOST":     "host",
			"DB_PASSWORD": "nonexistent", // Field doesn't exist
		},
	}

	_, _, err := server.buildEnvironmentFromBindings(entry)
	if err == nil {
		t.Error("expected error for nonexistent field in binding")
	}
}

func TestBuildEnvironmentFromBindings_BlockedEnvVar(t *testing.T) {
	v, tmpDir := testVault(t)

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	entry := &vault.SecretEntry{
		Fields: map[string]vault.Field{
			"evil": {Value: "/evil/lib.so", Sensitive: false},
		},
		Bindings: map[string]string{
			"LD_PRELOAD": "evil", // Blocked env var
		},
	}

	_, _, err := server.buildEnvironmentFromBindings(entry)
	if err == nil {
		t.Error("expected error for blocked env var")
	}
}

func TestHandleSecretRunWithBindings_CommandNotAllowed(t *testing.T) {
	v, tmpDir := testVault(t)

	fields := map[string]vault.Field{
		"host": {Value: "db.example.com", Sensitive: false},
	}
	bindings := map[string]string{
		"DB_HOST": "host",
	}
	addTestMultiFieldSecret(t, v, "db_creds", fields, bindings)

	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionDeny,
		AllowedCommands: []string{"echo"}, // Only echo allowed
	}

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretRunWithBindings(ctx, nil, &SecretRunWithBindingsInput{
		Key:     "db_creds",
		Command: "rm", // Not allowed
		Args:    []string{"-rf", "/"},
	})
	if err == nil {
		t.Error("expected error for disallowed command")
	}
}

// Note: TestHandleSecretRunWithBindings_ExpiredSecret is not implemented because
// the vault correctly prevents creating expired secrets at set time.
// The expiration check in handleSecretRunWithBindings is a defense-in-depth measure
// for the edge case where a secret expires between check and use.

func TestHandleSecretRunWithBindings_SecretNotFound(t *testing.T) {
	v, tmpDir := testVault(t)

	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionDeny,
		AllowedCommands: []string{"echo"},
	}

	server := &Server{
		vault:     v,
		vaultPath: tmpDir,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	ctx := context.Background()
	_, _, err := server.handleSecretRunWithBindings(ctx, nil, &SecretRunWithBindingsInput{
		Key:     "nonexistent",
		Command: "echo",
		Args:    []string{"hello"},
	})
	if err == nil {
		t.Error("expected error for nonexistent secret")
	}
}

func TestWipeEnvSlice(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"SECRET=mysecretvalue",
		"HOME=/home/user",
	}

	// Wipe the slice
	wipeEnvSlice(env)

	// All entries should be empty strings
	for i, e := range env {
		if e != "" {
			t.Errorf("env[%d] should be empty, got '%s'", i, e)
		}
	}
}

func TestWipeBuffer(t *testing.T) {
	buf := bytes.NewBufferString("sensitive-secret-data-12345")
	originalLen := buf.Len()

	// Get reference to underlying bytes before wipe
	underlying := buf.Bytes()

	// Wipe the buffer
	wipeBuffer(buf)

	// Buffer should be reset (empty)
	if buf.Len() != 0 {
		t.Errorf("buffer length should be 0, got %d", buf.Len())
	}

	// The underlying byte slice should be zeroed
	for i := 0; i < originalLen; i++ {
		if underlying[i] != 0 {
			t.Errorf("underlying[%d] should be 0, got %d", i, underlying[i])
		}
	}
}
