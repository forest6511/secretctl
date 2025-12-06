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
		{"aws", "aws", true},
		{"/usr/bin/aws", "aws", true},
		{"aws", "/usr/bin/aws", true},
		{"/usr/bin/aws", "/usr/local/bin/aws", true}, // Both resolve to "aws"
		{"aws-cli", "aws", false},
		{"kubectl", "aws", false},
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
}
