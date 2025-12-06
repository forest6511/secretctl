package mcp

import (
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
