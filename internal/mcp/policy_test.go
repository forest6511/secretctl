package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPolicyIsCommandAllowed(t *testing.T) {
	tests := []struct {
		name     string
		policy   Policy
		command  string
		allowed  bool
	}{
		{
			name: "denied command",
			policy: Policy{
				Version:        1,
				DefaultAction:  "allow",
				DeniedCommands: []string{"env", "printenv"},
			},
			command: "env",
			allowed: false,
		},
		{
			name: "allowed command",
			policy: Policy{
				Version:         1,
				DefaultAction:   "deny",
				AllowedCommands: []string{"aws", "gcloud"},
			},
			command: "aws",
			allowed: true,
		},
		{
			name: "default deny",
			policy: Policy{
				Version:         1,
				DefaultAction:   "deny",
				AllowedCommands: []string{"aws"},
			},
			command: "kubectl",
			allowed: false,
		},
		{
			name: "default allow",
			policy: Policy{
				Version:         1,
				DefaultAction:   "allow",
				DeniedCommands:  []string{"env"},
				AllowedCommands: []string{},
			},
			command: "kubectl",
			allowed: true,
		},
		{
			name: "denied takes priority over allowed",
			policy: Policy{
				Version:         1,
				DefaultAction:   "allow",
				DeniedCommands:  []string{"env"},
				AllowedCommands: []string{"env"}, // contradicting
			},
			command: "env",
			allowed: false, // denied takes priority
		},
		{
			name: "full path match",
			policy: Policy{
				Version:         1,
				DefaultAction:   "deny",
				AllowedCommands: []string{"aws"},
			},
			command: "/usr/bin/aws",
			allowed: true,
		},
		{
			name: "default denied env always blocked",
			policy: Policy{
				Version:         1,
				DefaultAction:   "allow",
				AllowedCommands: []string{"env"}, // even if explicitly allowed
			},
			command: "env",
			allowed: false, // always denied per §4.2
		},
		{
			name: "default denied printenv always blocked",
			policy: Policy{
				Version:       1,
				DefaultAction: "allow",
			},
			command: "printenv",
			allowed: false, // always denied per §4.2
		},
		{
			name: "default denied set always blocked",
			policy: Policy{
				Version:       1,
				DefaultAction: "allow",
			},
			command: "set",
			allowed: false, // always denied per §4.2
		},
		{
			name: "default denied export always blocked",
			policy: Policy{
				Version:       1,
				DefaultAction: "allow",
			},
			command: "/bin/export",
			allowed: false, // always denied per §4.2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, _ := tt.policy.IsCommandAllowed(tt.command)
			if allowed != tt.allowed {
				t.Errorf("IsCommandAllowed(%q) = %v, want %v", tt.command, allowed, tt.allowed)
			}
		})
	}
}

func TestLoadPolicy_NotFound(t *testing.T) {
	tempDir := t.TempDir()

	_, err := LoadPolicy(tempDir)
	if err != ErrPolicyNotFound {
		t.Errorf("LoadPolicy() error = %v, want ErrPolicyNotFound", err)
	}
}

func TestLoadPolicy_InsecurePermissions(t *testing.T) {
	tempDir := t.TempDir()
	policyPath := filepath.Join(tempDir, PolicyFileName)

	// Create policy file with insecure permissions
	content := []byte("version: 1\ndefault_action: deny\n")
	if err := os.WriteFile(policyPath, content, 0644); err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	_, err := LoadPolicy(tempDir)
	if err == nil {
		t.Error("LoadPolicy() expected error for insecure permissions")
	}
}

func TestLoadPolicy_ValidPolicy(t *testing.T) {
	tempDir := t.TempDir()
	policyPath := filepath.Join(tempDir, PolicyFileName)

	content := `version: 1
default_action: deny
denied_commands:
  - env
  - printenv
allowed_commands:
  - aws
  - gcloud
`
	if err := os.WriteFile(policyPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	policy, err := LoadPolicy(tempDir)
	if err != nil {
		t.Fatalf("LoadPolicy() error = %v", err)
	}

	if policy.Version != 1 {
		t.Errorf("policy.Version = %v, want 1", policy.Version)
	}
	if policy.DefaultAction != "deny" {
		t.Errorf("policy.DefaultAction = %v, want deny", policy.DefaultAction)
	}
	if len(policy.DeniedCommands) != 2 {
		t.Errorf("len(policy.DeniedCommands) = %v, want 2", len(policy.DeniedCommands))
	}
	if len(policy.AllowedCommands) != 2 {
		t.Errorf("len(policy.AllowedCommands) = %v, want 2", len(policy.AllowedCommands))
	}
}

func TestValidatePolicy(t *testing.T) {
	tests := []struct {
		name    string
		policy  Policy
		wantErr bool
	}{
		{
			name:    "valid policy",
			policy:  Policy{Version: 1, DefaultAction: "deny"},
			wantErr: false,
		},
		{
			name:    "valid policy allow",
			policy:  Policy{Version: 1, DefaultAction: "allow"},
			wantErr: false,
		},
		{
			name:    "invalid version",
			policy:  Policy{Version: 2, DefaultAction: "deny"},
			wantErr: true,
		},
		{
			name:    "invalid default action",
			policy:  Policy{Version: 1, DefaultAction: "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.ValidatePolicy()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultDeniedCommands(t *testing.T) {
	defaults := DefaultDeniedCommands()

	// Should contain common dangerous commands
	expected := map[string]bool{
		"env":      true,
		"printenv": true,
		"set":      true,
		"export":   true,
	}

	for _, cmd := range defaults {
		if expected[cmd] {
			delete(expected, cmd)
		}
	}

	if len(expected) > 0 {
		t.Errorf("DefaultDeniedCommands() missing expected commands: %v", expected)
	}
}
