package mcp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"gopkg.in/yaml.v3"
)

// Policy represents the MCP policy configuration per mcp-design-ja.md §4
type Policy struct {
	Version         int      `yaml:"version"`
	DefaultAction   string   `yaml:"default_action"`
	DeniedCommands  []string `yaml:"denied_commands"`
	AllowedCommands []string `yaml:"allowed_commands"`
}

// PolicyFileName is the name of the policy file
const PolicyFileName = "mcp-policy.yaml"

// ErrPolicyNotFound is returned when no policy file exists
var ErrPolicyNotFound = errors.New("MCP policy file not found")

// ErrPolicyInsecure is returned when policy file has insecure permissions
var ErrPolicyInsecure = errors.New("MCP policy file has insecure permissions")

// ErrPolicySymlink is returned when policy file is a symlink
var ErrPolicySymlink = errors.New("MCP policy file is a symlink")

// ErrPolicyNotOwnedByUser is returned when policy file is not owned by current user
var ErrPolicyNotOwnedByUser = errors.New("MCP policy file not owned by current user")

// LoadPolicy loads the MCP policy from the vault directory.
// Implements TOCTOU-safe loading per mcp-design-ja.md §4.5.2
func LoadPolicy(vaultPath string) (*Policy, error) {
	policyPath := filepath.Join(vaultPath, PolicyFileName)

	// 1. Open with O_NOFOLLOW to reject symlinks per §4.5.2
	f, err := os.OpenFile(policyPath, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrPolicyNotFound
		}
		if os.IsPermission(err) || errors.Is(err, syscall.ELOOP) {
			return nil, ErrPolicySymlink
		}
		return nil, fmt.Errorf("failed to open policy file: %w", err)
	}
	defer f.Close()

	// 2. Use fstat on the opened file descriptor to avoid TOCTOU
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat policy file: %w", err)
	}

	// 3. Check permissions (must be 0600)
	perm := info.Mode().Perm()
	if perm != 0600 {
		return nil, fmt.Errorf("%w: %o (expected 0600)", ErrPolicyInsecure, perm)
	}

	// 4. Check ownership (must be current user)
	stat, ok := info.Sys().(*syscall.Stat_t)
	if ok {
		if stat.Uid != uint32(os.Getuid()) {
			return nil, ErrPolicyNotOwnedByUser
		}
	}

	// 5. Read and parse the policy file
	content, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var policy Policy
	if err := yaml.Unmarshal(content, &policy); err != nil {
		return nil, fmt.Errorf("failed to parse policy file: %w", err)
	}

	// Validate policy version
	if policy.Version != 1 {
		return nil, fmt.Errorf("unsupported policy version: %d", policy.Version)
	}

	// Default to deny if not specified
	if policy.DefaultAction == "" {
		policy.DefaultAction = "deny"
	}

	return &policy, nil
}

// IsCommandAllowed checks if a command is allowed by the policy.
// Evaluation order per mcp-design-ja.md §4.3:
// 0. default_denied_commands → always deny (hardcoded security)
// 1. denied_commands → deny
// 2. allowed_commands → allow
// 3. default_action
func (p *Policy) IsCommandAllowed(command string) (bool, string) {
	// 0. Check default denied commands first (always blocked per §4.2)
	for _, denied := range DefaultDeniedCommands() {
		if matchCommand(command, denied) {
			return false, fmt.Sprintf("command '%s' is always denied for security", command)
		}
	}

	// 1. Check user-defined denied commands (highest priority)
	for _, denied := range p.DeniedCommands {
		if matchCommand(command, denied) {
			return false, fmt.Sprintf("command '%s' matches denied pattern '%s'", command, denied)
		}
	}

	// 2. Check allowed commands
	for _, allowed := range p.AllowedCommands {
		if matchCommand(command, allowed) {
			return true, ""
		}
	}

	// 3. Use default action
	if p.DefaultAction == "allow" {
		return true, ""
	}

	return false, fmt.Sprintf("command '%s' not in allowed_commands list", command)
}

// matchCommand checks if a command matches a pattern.
// For simplicity, this uses exact match on the command name (base path).
// Future: could support glob patterns.
func matchCommand(command, pattern string) bool {
	// Extract base command name
	cmdBase := filepath.Base(command)
	patternBase := filepath.Base(pattern)

	// Exact match
	return cmdBase == patternBase || command == pattern
}

// ValidatePolicy validates the policy configuration
func (p *Policy) ValidatePolicy() error {
	if p.Version != 1 {
		return fmt.Errorf("unsupported policy version: %d", p.Version)
	}

	if p.DefaultAction != "deny" && p.DefaultAction != "allow" {
		return fmt.Errorf("invalid default_action: %s (must be 'deny' or 'allow')", p.DefaultAction)
	}

	return nil
}

// DefaultDeniedCommands returns the default list of denied commands
// that should always be blocked per mcp-design-ja.md §4.2
func DefaultDeniedCommands() []string {
	return []string{
		"env",
		"printenv",
		"set",
		"export",
		"cat /proc/*/environ",
	}
}
