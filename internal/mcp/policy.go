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

// EnvAliasMapping represents a single pattern-to-target mapping
type EnvAliasMapping struct {
	Pattern string `yaml:"pattern"`
	Target  string `yaml:"target"`
}

// Policy represents the MCP policy configuration per mcp-design-ja.md §4
type Policy struct {
	Version         int                          `yaml:"version"`
	DefaultAction   string                       `yaml:"default_action"`
	DeniedCommands  []string                     `yaml:"denied_commands"`
	AllowedCommands []string                     `yaml:"allowed_commands"`
	EnvAliases      map[string][]EnvAliasMapping `yaml:"env_aliases"`
}

// PolicyFileName is the name of the policy file
const PolicyFileName = "mcp-policy.yaml"

// Policy action constants
const (
	ActionAllow = "allow"
	ActionDeny  = "deny"
)

// ErrPolicyNotFound is returned when no policy file exists
var ErrPolicyNotFound = errors.New("MCP policy file not found")

// ErrPolicyInsecure is returned when policy file has insecure permissions
var ErrPolicyInsecure = errors.New("MCP policy file has insecure permissions")

// ErrPolicySymlink is returned when policy file is a symlink
var ErrPolicySymlink = errors.New("MCP policy file is a symlink")

// ErrPolicyNotOwnedByUser is returned when policy file is not owned by current user
var ErrPolicyNotOwnedByUser = errors.New("MCP policy file not owned by current user")

// ErrEnvNotFound is returned when the specified environment alias is not found
var ErrEnvNotFound = errors.New("environment alias not found")

// ErrCommandNotInTrustedDir is returned when command is not in a trusted directory
var ErrCommandNotInTrustedDir = errors.New("command not in trusted directory")

// ErrCommandNotFound is returned when command cannot be found
var ErrCommandNotFound = errors.New("command not found")

// TrustedDirectories are the only directories from which commands can be executed.
// This prevents PATH manipulation attacks where a malicious binary is placed
// earlier in PATH to bypass the allowlist.
var TrustedDirectories = []string{
	"/usr/bin",
	"/bin",
	"/usr/sbin",
	"/sbin",
	"/usr/local/bin",
	"/opt/homebrew/bin", // macOS Homebrew
}

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
		policy.DefaultAction = ActionDeny
	}

	return &policy, nil
}

// IsCommandAllowed checks if a command is allowed by the policy.
// Evaluation order per mcp-design-ja.md §4.3:
// 0. default_denied_commands → always deny (hardcoded security)
// 1. denied_commands → deny
// 2. allowed_commands → allow
// 3. default_action
func (p *Policy) IsCommandAllowed(command string) (allowed bool, reason string) {
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
	if p.DefaultAction == ActionAllow {
		return true, ""
	}

	return false, fmt.Sprintf("command '%s' not in allowed_commands list", command)
}

// matchCommand checks if a command matches a pattern.
// Security behavior:
// - If pattern is an absolute path (e.g., "/usr/bin/curl"), require exact match
// - If pattern contains spaces (e.g., "cat /proc/*/environ"), require exact match
// - If pattern is just a command name (e.g., "curl"), match against basename
// This prevents a policy entry like "/usr/bin/aws" from allowing any "aws" binary.
func matchCommand(command, pattern string) bool {
	// If pattern is an absolute path, require exact match
	// This ensures "/usr/bin/aws" in policy only allows that exact binary
	if filepath.IsAbs(pattern) {
		// Clean both paths for comparison
		return filepath.Clean(command) == filepath.Clean(pattern)
	}

	// If pattern contains spaces (command line with args), require exact match
	// This handles patterns like "cat /proc/*/environ" in DefaultDeniedCommands
	if containsSpace(pattern) {
		return command == pattern
	}

	// Pattern is a bare command name - match against basename
	cmdBase := filepath.Base(command)
	return cmdBase == pattern
}

// containsSpace checks if a string contains any whitespace
func containsSpace(s string) bool {
	for _, c := range s {
		if c == ' ' || c == '\t' {
			return true
		}
	}
	return false
}

// ResolveAndValidateCommand resolves a command name to its full path and validates
// that it is in a trusted directory. This prevents PATH manipulation attacks.
//
// Security: This function MUST be called BEFORE IsCommandAllowed to ensure
// that we check the policy against the actual binary that will be executed.
//
// Returns the resolved absolute path of the command, or an error if:
// - The command cannot be found
// - The command is not in a trusted directory
// - The command resolves through symlinks to an untrusted location
func ResolveAndValidateCommand(command string) (string, error) {
	var cmdPath string

	if filepath.IsAbs(command) {
		// Absolute path provided - use it directly
		cmdPath = command
	} else {
		// Search for command in trusted directories only
		// NOTE: We do NOT modify the global PATH to avoid race conditions
		var err error
		cmdPath, err = lookupCommandInTrustedDirs(command)
		if err != nil {
			return "", err
		}
	}

	// Verify the file exists, is not a directory, and is executable
	info, err := os.Stat(cmdPath)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrCommandNotFound, cmdPath)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%w: %s is a directory", ErrCommandNotFound, cmdPath)
	}
	// Check if executable (at least one execute bit set)
	if info.Mode()&0111 == 0 {
		return "", fmt.Errorf("%w: %s is not executable", ErrCommandNotFound, cmdPath)
	}

	// CRITICAL: Resolve symlinks to get the real path
	// This prevents symlink bypass attacks where a symlink in a trusted dir
	// points to a malicious binary in an untrusted location
	realPath, err := filepath.EvalSymlinks(cmdPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlinks for %s: %w", cmdPath, err)
	}

	// Validate that the REAL path (after symlink resolution) is in a trusted directory
	if err := validateTrustedDirectory(realPath); err != nil {
		return "", fmt.Errorf("symlink target not in trusted directory: %w", err)
	}

	// Return the real path to ensure we execute the actual validated binary
	return realPath, nil
}

// lookupCommandInTrustedDirs searches for a command in trusted directories only.
// This is a thread-safe alternative to modifying the global PATH.
func lookupCommandInTrustedDirs(command string) (string, error) {
	for _, dir := range TrustedDirectories {
		// Check if the directory exists
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		cmdPath := filepath.Join(dir, command)

		// Check if the file exists and is executable
		info, err := os.Stat(cmdPath)
		if err != nil {
			continue
		}

		// Skip directories
		if info.IsDir() {
			continue
		}

		// Check if executable (on Unix-like systems)
		if info.Mode()&0111 != 0 {
			return cmdPath, nil
		}
	}

	return "", fmt.Errorf("%w: %s", ErrCommandNotFound, command)
}

// validateTrustedDirectory checks if the command path is within a trusted directory
func validateTrustedDirectory(cmdPath string) error {
	// Get the directory containing the command
	cmdDir := filepath.Dir(cmdPath)

	// Clean the path to normalize it
	cmdDir = filepath.Clean(cmdDir)

	// Check if the directory is in our trusted list
	for _, trusted := range TrustedDirectories {
		trustedClean := filepath.Clean(trusted)
		if cmdDir == trustedClean {
			return nil
		}
	}

	return fmt.Errorf("%w: %s (allowed: %v)", ErrCommandNotInTrustedDir, cmdPath, TrustedDirectories)
}

// ValidatePolicy validates the policy configuration
func (p *Policy) ValidatePolicy() error {
	if p.Version != 1 {
		return fmt.Errorf("unsupported policy version: %d", p.Version)
	}

	if p.DefaultAction != ActionDeny && p.DefaultAction != ActionAllow {
		return fmt.Errorf("invalid default_action: %s (must be '%s' or '%s')", p.DefaultAction, ActionDeny, ActionAllow)
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

// ResolveAlias resolves a key pattern using environment aliases.
// If env is empty, returns the original key unchanged.
// If env is specified but not found in policy, returns ErrEnvNotFound.
// If env is specified and found, applies matching alias transformations.
func (p *Policy) ResolveAlias(env, key string) (string, error) {
	// No environment specified, return key unchanged
	if env == "" {
		return key, nil
	}

	// No aliases configured
	if p.EnvAliases == nil {
		return "", fmt.Errorf("%w: '%s'", ErrEnvNotFound, env)
	}

	// Look up environment aliases
	aliases, exists := p.EnvAliases[env]
	if !exists {
		return "", fmt.Errorf("%w: '%s'", ErrEnvNotFound, env)
	}

	// Try to match each alias pattern
	for _, alias := range aliases {
		if matched, result := matchAndTransform(key, alias.Pattern, alias.Target); matched {
			return result, nil
		}
	}

	// No matching alias, return key unchanged
	return key, nil
}

// ResolveAliasKeys resolves multiple keys using environment aliases.
// Returns the resolved keys and any error encountered.
func (p *Policy) ResolveAliasKeys(env string, keys []string) ([]string, error) {
	if env == "" {
		return keys, nil
	}

	resolved := make([]string, 0, len(keys))
	for _, key := range keys {
		r, err := p.ResolveAlias(env, key)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, r)
	}
	return resolved, nil
}

// HasEnvAlias checks if the specified environment alias exists
func (p *Policy) HasEnvAlias(env string) bool {
	if p.EnvAliases == nil {
		return false
	}
	_, exists := p.EnvAliases[env]
	return exists
}

// ListEnvAliases returns all available environment alias names
func (p *Policy) ListEnvAliases() []string {
	if p.EnvAliases == nil {
		return nil
	}
	names := make([]string, 0, len(p.EnvAliases))
	for name := range p.EnvAliases {
		names = append(names, name)
	}
	return names
}

// matchAndTransform checks if key matches pattern and applies transformation.
// Pattern supports glob-style wildcards:
//   - "db/*" matches "db/host", "db/password", etc.
//   - "*" at the end matches any suffix
//
// Returns (matched, transformedKey)
func matchAndTransform(key, pattern, target string) (matched bool, transformedKey string) {
	// Handle wildcard patterns
	if pattern != "" && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			// Extract the suffix that matched the wildcard
			suffix := key[len(prefix):]
			// Apply to target (replace * with the matched suffix)
			if target != "" && target[len(target)-1] == '*' {
				return true, target[:len(target)-1] + suffix
			}
			return true, target + suffix
		}
		return false, ""
	}

	// Exact match
	if key == pattern {
		return true, target
	}

	return false, ""
}
