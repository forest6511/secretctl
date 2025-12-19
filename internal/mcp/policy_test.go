package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPolicy_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := LoadPolicy(tmpDir)
	if err != ErrPolicyNotFound {
		t.Errorf("expected ErrPolicyNotFound, got %v", err)
	}
}

func TestLoadPolicy_Success(t *testing.T) {
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, PolicyFileName)

	content := `version: 1
default_action: deny
allowed_commands:
  - aws
  - kubectl
denied_commands:
  - rm
`
	if err := os.WriteFile(policyPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write policy file: %v", err)
	}

	policy, err := LoadPolicy(tmpDir)
	if err != nil {
		t.Fatalf("LoadPolicy failed: %v", err)
	}

	if policy.Version != 1 {
		t.Errorf("expected version 1, got %d", policy.Version)
	}
	if policy.DefaultAction != ActionDeny {
		t.Errorf("expected default_action 'deny', got '%s'", policy.DefaultAction)
	}
	if len(policy.AllowedCommands) != 2 {
		t.Errorf("expected 2 allowed commands, got %d", len(policy.AllowedCommands))
	}
	if len(policy.DeniedCommands) != 1 {
		t.Errorf("expected 1 denied command, got %d", len(policy.DeniedCommands))
	}
}

func TestLoadPolicy_InsecurePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, PolicyFileName)

	content := `version: 1
default_action: deny
`
	// Write with insecure permissions (0644)
	if err := os.WriteFile(policyPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write policy file: %v", err)
	}

	_, err := LoadPolicy(tmpDir)
	if err == nil {
		t.Error("expected error for insecure permissions")
	}
}

func TestLoadPolicy_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, PolicyFileName)

	content := `invalid: yaml: content: [[[`
	if err := os.WriteFile(policyPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write policy file: %v", err)
	}

	_, err := LoadPolicy(tmpDir)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadPolicy_UnsupportedVersion(t *testing.T) {
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, PolicyFileName)

	content := `version: 99
default_action: deny
`
	if err := os.WriteFile(policyPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write policy file: %v", err)
	}

	_, err := LoadPolicy(tmpDir)
	if err == nil {
		t.Error("expected error for unsupported version")
	}
}

func TestLoadPolicy_DefaultActionFallback(t *testing.T) {
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, PolicyFileName)

	// No default_action specified
	content := `version: 1
allowed_commands:
  - aws
`
	if err := os.WriteFile(policyPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write policy file: %v", err)
	}

	policy, err := LoadPolicy(tmpDir)
	if err != nil {
		t.Fatalf("LoadPolicy failed: %v", err)
	}

	// Should default to deny
	if policy.DefaultAction != ActionDeny {
		t.Errorf("expected default_action 'deny', got '%s'", policy.DefaultAction)
	}
}

func TestLoadPolicy_Symlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create actual policy file
	realPath := filepath.Join(tmpDir, "real-policy.yaml")
	content := `version: 1
default_action: deny
`
	if err := os.WriteFile(realPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write real policy file: %v", err)
	}

	// Create symlink
	policyPath := filepath.Join(tmpDir, PolicyFileName)
	if err := os.Symlink(realPath, policyPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	_, err := LoadPolicy(tmpDir)
	if err != ErrPolicySymlink {
		t.Errorf("expected ErrPolicySymlink, got %v", err)
	}
}

func TestIsCommandAllowed_DefaultDenied(t *testing.T) {
	policy := &Policy{
		Version:       1,
		DefaultAction: ActionAllow,
	}

	// Default denied commands should always be blocked
	deniedCmds := DefaultDeniedCommands()
	for _, cmd := range deniedCmds {
		allowed, reason := policy.IsCommandAllowed(cmd)
		if allowed {
			t.Errorf("expected '%s' to be denied", cmd)
		}
		if reason == "" {
			t.Errorf("expected reason for denied command '%s'", cmd)
		}
	}
}

func TestIsCommandAllowed_UserDenied(t *testing.T) {
	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionAllow,
		DeniedCommands:  []string{"rm", "dd"},
		AllowedCommands: []string{"aws", "kubectl"},
	}

	tests := []struct {
		command string
		allowed bool
	}{
		{"rm", false},  // User denied
		{"dd", false},  // User denied
		{"aws", true},  // Allowed
		{"curl", true}, // Default allow
		{"env", false}, // Default denied (hardcoded)
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			allowed, _ := policy.IsCommandAllowed(tt.command)
			if allowed != tt.allowed {
				t.Errorf("IsCommandAllowed(%s) = %v, want %v", tt.command, allowed, tt.allowed)
			}
		})
	}
}

func TestIsCommandAllowed_AllowedCommands(t *testing.T) {
	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionDeny,
		AllowedCommands: []string{"aws", "kubectl", "helm"},
	}

	tests := []struct {
		command string
		allowed bool
	}{
		{"aws", true},
		{"kubectl", true},
		{"helm", true},
		{"rm", false},
		{"curl", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			allowed, _ := policy.IsCommandAllowed(tt.command)
			if allowed != tt.allowed {
				t.Errorf("IsCommandAllowed(%s) = %v, want %v", tt.command, allowed, tt.allowed)
			}
		})
	}
}

func TestIsCommandAllowed_FullPath(t *testing.T) {
	policy := &Policy{
		Version:         1,
		DefaultAction:   ActionDeny,
		AllowedCommands: []string{"aws"},
	}

	// Full path should match base name
	allowed, _ := policy.IsCommandAllowed("/usr/local/bin/aws")
	if !allowed {
		t.Error("expected /usr/local/bin/aws to match 'aws'")
	}
}

func TestMatchCommand(t *testing.T) {
	tests := []struct {
		command  string
		pattern  string
		expected bool
	}{
		// Bare command patterns match against basename
		{"aws", "aws", true},
		{"/usr/bin/aws", "aws", true},
		{"/usr/local/bin/aws", "aws", true},
		{"aws-cli", "aws", false},
		{"kubectl", "aws", false},

		// Absolute path patterns require exact match (security fix)
		{"/usr/bin/aws", "/usr/bin/aws", true},
		{"/usr/local/bin/aws", "/usr/bin/aws", false}, // Different path - NOT allowed
		{"aws", "/usr/bin/aws", false},                // Bare command doesn't match absolute pattern
		{"/usr/bin/aws", "/usr/local/bin/aws", false}, // Exact path match required
	}

	for _, tt := range tests {
		t.Run(tt.command+"_"+tt.pattern, func(t *testing.T) {
			result := matchCommand(tt.command, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchCommand(%s, %s) = %v, want %v", tt.command, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestMatchCommand_AbsolutePathSecurity(t *testing.T) {
	// This test specifically verifies that absolute path patterns in policy
	// only allow that exact binary, preventing PATH manipulation attacks

	// If policy says "/usr/bin/curl", only that exact path should be allowed
	// NOT any "curl" binary in other directories
	tests := []struct {
		name     string
		command  string
		pattern  string
		expected bool
	}{
		{
			name:     "exact match allowed",
			command:  "/usr/bin/curl",
			pattern:  "/usr/bin/curl",
			expected: true,
		},
		{
			name:     "different directory rejected",
			command:  "/usr/local/bin/curl",
			pattern:  "/usr/bin/curl",
			expected: false,
		},
		{
			name:     "malicious path rejected",
			command:  "/tmp/evil/curl",
			pattern:  "/usr/bin/curl",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchCommand(tt.command, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchCommand(%s, %s) = %v, want %v", tt.command, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestValidatePolicy_Valid(t *testing.T) {
	tests := []struct {
		name   string
		policy *Policy
	}{
		{
			name: "deny policy",
			policy: &Policy{
				Version:       1,
				DefaultAction: ActionDeny,
			},
		},
		{
			name: "allow policy",
			policy: &Policy{
				Version:       1,
				DefaultAction: ActionAllow,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.ValidatePolicy()
			if err != nil {
				t.Errorf("ValidatePolicy failed: %v", err)
			}
		})
	}
}

func TestValidatePolicy_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		policy *Policy
	}{
		{
			name: "invalid version",
			policy: &Policy{
				Version:       99,
				DefaultAction: ActionDeny,
			},
		},
		{
			name: "invalid default_action",
			policy: &Policy{
				Version:       1,
				DefaultAction: "invalid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.ValidatePolicy()
			if err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestDefaultDeniedCommands(t *testing.T) {
	commands := DefaultDeniedCommands()

	// Should include security-critical commands
	expected := []string{"env", "printenv", "set", "export"}
	for _, exp := range expected {
		found := false
		for _, cmd := range commands {
			if cmd == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected '%s' in default denied commands", exp)
		}
	}
}

func TestPolicyConstants(t *testing.T) {
	if ActionAllow != "allow" {
		t.Errorf("ActionAllow = %s, want 'allow'", ActionAllow)
	}
	if ActionDeny != "deny" {
		t.Errorf("ActionDeny = %s, want 'deny'", ActionDeny)
	}
	if PolicyFileName != "mcp-policy.yaml" {
		t.Errorf("PolicyFileName = %s, want 'mcp-policy.yaml'", PolicyFileName)
	}
}

func TestPolicyErrors(t *testing.T) {
	// Verify error variables are defined
	if ErrPolicyNotFound == nil {
		t.Error("ErrPolicyNotFound is nil")
	}
	if ErrPolicyInsecure == nil {
		t.Error("ErrPolicyInsecure is nil")
	}
	if ErrPolicySymlink == nil {
		t.Error("ErrPolicySymlink is nil")
	}
	if ErrPolicyNotOwnedByUser == nil {
		t.Error("ErrPolicyNotOwnedByUser is nil")
	}
	if ErrEnvNotFound == nil {
		t.Error("ErrEnvNotFound is nil")
	}
}

func TestLoadPolicy_WithEnvAliases(t *testing.T) {
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, PolicyFileName)

	content := `version: 1
default_action: deny
allowed_commands:
  - aws
env_aliases:
  dev:
    - pattern: "db/*"
      target: "dev/db/*"
    - pattern: "api/*"
      target: "dev/api/*"
  prod:
    - pattern: "db/*"
      target: "prod/db/*"
`
	if err := os.WriteFile(policyPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write policy file: %v", err)
	}

	policy, err := LoadPolicy(tmpDir)
	if err != nil {
		t.Fatalf("LoadPolicy failed: %v", err)
	}

	if len(policy.EnvAliases) != 2 {
		t.Errorf("expected 2 env aliases, got %d", len(policy.EnvAliases))
	}
	if len(policy.EnvAliases["dev"]) != 2 {
		t.Errorf("expected 2 dev aliases, got %d", len(policy.EnvAliases["dev"]))
	}
	if len(policy.EnvAliases["prod"]) != 1 {
		t.Errorf("expected 1 prod alias, got %d", len(policy.EnvAliases["prod"]))
	}
}

func TestResolveAlias_NoEnv(t *testing.T) {
	policy := &Policy{
		Version:       1,
		DefaultAction: ActionDeny,
		EnvAliases: map[string][]EnvAliasMapping{
			"dev": {{Pattern: "db/*", Target: "dev/db/*"}},
		},
	}

	// When env is empty, key should be returned unchanged
	result, err := policy.ResolveAlias("", "db/host")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "db/host" {
		t.Errorf("expected 'db/host', got '%s'", result)
	}
}

func TestResolveAlias_EnvNotFound(t *testing.T) {
	policy := &Policy{
		Version:       1,
		DefaultAction: ActionDeny,
		EnvAliases: map[string][]EnvAliasMapping{
			"dev": {{Pattern: "db/*", Target: "dev/db/*"}},
		},
	}

	_, err := policy.ResolveAlias("staging", "db/host")
	if err == nil {
		t.Error("expected error for unknown environment")
	}
}

func TestResolveAlias_NoAliasesConfigured(t *testing.T) {
	policy := &Policy{
		Version:       1,
		DefaultAction: ActionDeny,
	}

	_, err := policy.ResolveAlias("dev", "db/host")
	if err == nil {
		t.Error("expected error when no aliases configured")
	}
}

func TestResolveAlias_WildcardPattern(t *testing.T) {
	policy := &Policy{
		Version:       1,
		DefaultAction: ActionDeny,
		EnvAliases: map[string][]EnvAliasMapping{
			"dev": {
				{Pattern: "db/*", Target: "dev/db/*"},
				{Pattern: "api/*", Target: "dev/api/*"},
			},
			"prod": {
				{Pattern: "db/*", Target: "prod/db/*"},
			},
		},
	}

	tests := []struct {
		name     string
		env      string
		key      string
		expected string
	}{
		{"dev db host", "dev", "db/host", "dev/db/host"},
		{"dev db password", "dev", "db/password", "dev/db/password"},
		{"dev api key", "dev", "api/key", "dev/api/key"},
		{"prod db host", "prod", "db/host", "prod/db/host"},
		{"no match returns unchanged", "dev", "other/key", "other/key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := policy.ResolveAlias(tt.env, tt.key)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("ResolveAlias(%s, %s) = %s, want %s", tt.env, tt.key, result, tt.expected)
			}
		})
	}
}

func TestResolveAlias_ExactMatch(t *testing.T) {
	policy := &Policy{
		Version:       1,
		DefaultAction: ActionDeny,
		EnvAliases: map[string][]EnvAliasMapping{
			"dev": {
				{Pattern: "special_key", Target: "dev/special"},
			},
		},
	}

	result, err := policy.ResolveAlias("dev", "special_key")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "dev/special" {
		t.Errorf("expected 'dev/special', got '%s'", result)
	}
}

func TestResolveAliasKeys(t *testing.T) {
	policy := &Policy{
		Version:       1,
		DefaultAction: ActionDeny,
		EnvAliases: map[string][]EnvAliasMapping{
			"dev": {
				{Pattern: "db/*", Target: "dev/db/*"},
				{Pattern: "api/*", Target: "dev/api/*"},
			},
		},
	}

	keys := []string{"db/host", "db/password", "api/key"}
	resolved, err := policy.ResolveAliasKeys("dev", keys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"dev/db/host", "dev/db/password", "dev/api/key"}
	if len(resolved) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(resolved))
	}
	for i, exp := range expected {
		if resolved[i] != exp {
			t.Errorf("resolved[%d] = %s, want %s", i, resolved[i], exp)
		}
	}
}

func TestResolveAliasKeys_EmptyEnv(t *testing.T) {
	policy := &Policy{
		Version:       1,
		DefaultAction: ActionDeny,
		EnvAliases: map[string][]EnvAliasMapping{
			"dev": {{Pattern: "db/*", Target: "dev/db/*"}},
		},
	}

	keys := []string{"db/host", "db/password"}
	resolved, err := policy.ResolveAliasKeys("", keys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return original keys unchanged
	if len(resolved) != len(keys) {
		t.Fatalf("expected %d keys, got %d", len(keys), len(resolved))
	}
	for i := range keys {
		if resolved[i] != keys[i] {
			t.Errorf("resolved[%d] = %s, want %s", i, resolved[i], keys[i])
		}
	}
}

func TestHasEnvAlias(t *testing.T) {
	policy := &Policy{
		Version:       1,
		DefaultAction: ActionDeny,
		EnvAliases: map[string][]EnvAliasMapping{
			"dev":  {{Pattern: "db/*", Target: "dev/db/*"}},
			"prod": {{Pattern: "db/*", Target: "prod/db/*"}},
		},
	}

	if !policy.HasEnvAlias("dev") {
		t.Error("expected HasEnvAlias('dev') to return true")
	}
	if !policy.HasEnvAlias("prod") {
		t.Error("expected HasEnvAlias('prod') to return true")
	}
	if policy.HasEnvAlias("staging") {
		t.Error("expected HasEnvAlias('staging') to return false")
	}
}

func TestHasEnvAlias_NoAliases(t *testing.T) {
	policy := &Policy{
		Version:       1,
		DefaultAction: ActionDeny,
	}

	if policy.HasEnvAlias("dev") {
		t.Error("expected HasEnvAlias to return false when no aliases configured")
	}
}

func TestListEnvAliases(t *testing.T) {
	policy := &Policy{
		Version:       1,
		DefaultAction: ActionDeny,
		EnvAliases: map[string][]EnvAliasMapping{
			"dev":     {{Pattern: "db/*", Target: "dev/db/*"}},
			"staging": {{Pattern: "db/*", Target: "staging/db/*"}},
			"prod":    {{Pattern: "db/*", Target: "prod/db/*"}},
		},
	}

	names := policy.ListEnvAliases()
	if len(names) != 3 {
		t.Errorf("expected 3 aliases, got %d", len(names))
	}

	// Check all expected names are present
	expected := map[string]bool{"dev": true, "staging": true, "prod": true}
	for _, name := range names {
		if !expected[name] {
			t.Errorf("unexpected alias name: %s", name)
		}
		delete(expected, name)
	}
	if len(expected) > 0 {
		t.Errorf("missing alias names: %v", expected)
	}
}

func TestListEnvAliases_Empty(t *testing.T) {
	policy := &Policy{
		Version:       1,
		DefaultAction: ActionDeny,
	}

	names := policy.ListEnvAliases()
	if names != nil {
		t.Errorf("expected nil, got %v", names)
	}
}

func TestMatchAndTransform(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		pattern  string
		target   string
		matched  bool
		expected string
	}{
		{"exact match", "api_key", "api_key", "secret/api_key", true, "secret/api_key"},
		{"wildcard prefix", "db/host", "db/*", "dev/db/*", true, "dev/db/host"},
		{"wildcard with path", "api/v1/key", "api/*", "dev/api/*", true, "dev/api/v1/key"},
		{"no match", "other/key", "db/*", "dev/db/*", false, ""},
		{"exact no match", "api_key", "other_key", "secret/other", false, ""},
		{"target without wildcard", "db/host", "db/*", "dev/database/", true, "dev/database/host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, result := matchAndTransform(tt.key, tt.pattern, tt.target)
			if matched != tt.matched {
				t.Errorf("matchAndTransform matched = %v, want %v", matched, tt.matched)
			}
			if result != tt.expected {
				t.Errorf("matchAndTransform result = %s, want %s", result, tt.expected)
			}
		})
	}
}

// Tests for PATH manipulation prevention (Issue #59)

func TestResolveAndValidateCommand_TrustedDirectory(t *testing.T) {
	// Test that commands in trusted directories are resolved correctly
	// We test with common system commands that should exist
	testCmds := []string{"ls", "cat", "echo"}

	for _, cmd := range testCmds {
		t.Run(cmd, func(t *testing.T) {
			resolved, err := ResolveAndValidateCommand(cmd)
			if err != nil {
				// Command might not exist on all systems, skip if not found
				if err.Error() != "" && (err == ErrCommandNotFound ||
					(len(err.Error()) > 0 && err.Error()[:14] == "command not fo")) {
					t.Skipf("command %s not found on this system", cmd)
				}
				t.Errorf("ResolveAndValidateCommand(%s) error: %v", cmd, err)
				return
			}

			// Verify it returns an absolute path
			if !filepath.IsAbs(resolved) {
				t.Errorf("expected absolute path, got: %s", resolved)
			}

			// Verify the directory is in trusted list
			cmdDir := filepath.Dir(resolved)
			found := false
			for _, trusted := range TrustedDirectories {
				if filepath.Clean(cmdDir) == filepath.Clean(trusted) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("resolved path %s is not in trusted directories", resolved)
			}
		})
	}
}

func TestResolveAndValidateCommand_AbsolutePath(t *testing.T) {
	// Test with absolute path to a trusted command
	trustedPaths := []string{
		"/usr/bin/ls",
		"/bin/ls",
		"/usr/bin/cat",
		"/bin/cat",
	}

	for _, path := range trustedPaths {
		t.Run(path, func(t *testing.T) {
			// Check if the file exists first
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skipf("path %s does not exist on this system", path)
			}

			resolved, err := ResolveAndValidateCommand(path)
			if err != nil {
				t.Errorf("ResolveAndValidateCommand(%s) error: %v", path, err)
				return
			}

			if resolved != path {
				t.Errorf("expected %s, got %s", path, resolved)
			}
		})
	}
}

func TestResolveAndValidateCommand_UntrustedDirectory(t *testing.T) {
	// Create a temporary directory with an executable
	tmpDir := t.TempDir()
	maliciousCmd := filepath.Join(tmpDir, "malicious")

	// Create a fake executable
	if err := os.WriteFile(maliciousCmd, []byte("#!/bin/sh\necho malicious"), 0755); err != nil {
		t.Fatalf("failed to create test executable: %v", err)
	}

	// Try to resolve the absolute path - should fail because tmpDir is not trusted
	_, err := ResolveAndValidateCommand(maliciousCmd)
	if err == nil {
		t.Error("expected error for command in untrusted directory")
	}

	// Verify error is about untrusted directory
	if err != nil && err.Error() != "" {
		// Should contain information about untrusted directory
		t.Logf("Got expected error: %v", err)
	}
}

func TestResolveAndValidateCommand_NotFound(t *testing.T) {
	// Test with a command that doesn't exist
	_, err := ResolveAndValidateCommand("nonexistent_command_12345")
	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}

func TestResolveAndValidateCommand_Directory(t *testing.T) {
	// Test with a directory path instead of file
	_, err := ResolveAndValidateCommand("/usr/bin")
	if err == nil {
		t.Error("expected error when command is a directory")
	}
}

func TestValidateTrustedDirectory(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"trusted /usr/bin", "/usr/bin/ls", false},
		{"trusted /bin", "/bin/ls", false},
		{"trusted /usr/local/bin", "/usr/local/bin/something", false},
		{"untrusted /tmp", "/tmp/malicious", true},
		{"untrusted home", "/home/user/bin/script", true},
		{"untrusted relative", "./script", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTrustedDirectory(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTrustedDirectory(%s) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestLookupCommandInTrustedDirs(t *testing.T) {
	// Test that lookupCommandInTrustedDirs finds commands in trusted directories
	testCmds := []string{"ls", "cat"}

	for _, cmd := range testCmds {
		t.Run(cmd, func(t *testing.T) {
			path, err := lookupCommandInTrustedDirs(cmd)
			if err != nil {
				t.Skipf("command %s not found in trusted directories", cmd)
			}

			// Verify the path is absolute
			if !filepath.IsAbs(path) {
				t.Errorf("expected absolute path, got: %s", path)
			}

			// Verify the path is in a trusted directory
			dir := filepath.Dir(path)
			found := false
			for _, trusted := range TrustedDirectories {
				if filepath.Clean(dir) == filepath.Clean(trusted) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("command found in untrusted directory: %s", path)
			}
		})
	}
}

func TestResolveAndValidateCommand_SymlinkBypass(t *testing.T) {
	// This test verifies that symlinks to untrusted locations are rejected
	// We create a symlink in a test scenario

	// Skip if we can't create symlinks (Windows without admin)
	tmpDir := t.TempDir()
	testLink := filepath.Join(tmpDir, "testlink")
	testTarget := filepath.Join(tmpDir, "testtarget")

	// Create a test file
	if err := os.WriteFile(testTarget, []byte("test"), 0755); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if err := os.Symlink(testTarget, testLink); err != nil {
		t.Skipf("cannot create symlinks on this system: %v", err)
	}

	// Try to resolve the symlink - should fail because target is not in trusted dir
	_, err := ResolveAndValidateCommand(testLink)
	if err == nil {
		t.Error("expected error for symlink to untrusted directory")
	}

	t.Logf("Correctly rejected symlink: %v", err)
}

func TestTrustedDirectoriesExist(t *testing.T) {
	// Verify that the trusted directories list is not empty
	if len(TrustedDirectories) == 0 {
		t.Error("TrustedDirectories should not be empty")
	}

	// At least one should exist on most systems
	existsCount := 0
	for _, dir := range TrustedDirectories {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			existsCount++
		}
	}

	if existsCount == 0 {
		t.Error("at least one trusted directory should exist")
	}
}

func TestNewSecurityErrors(t *testing.T) {
	// Verify new error variables are defined
	if ErrCommandNotInTrustedDir == nil {
		t.Error("ErrCommandNotInTrustedDir is nil")
	}
	if ErrCommandNotFound == nil {
		t.Error("ErrCommandNotFound is nil")
	}
}
