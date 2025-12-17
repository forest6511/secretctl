package mcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/forest6511/secretctl/pkg/vault"
)

// Tool input types per mcp-design-ja.md §3

// SecretListInput represents input for secret_list tool.
type SecretListInput struct {
	Tag            string `json:"tag,omitempty"`
	ExpiringWithin string `json:"expiring_within,omitempty"`
}

// SecretListOutput represents output for secret_list tool.
type SecretListOutput struct {
	Secrets []SecretInfo `json:"secrets"`
}

// SecretInfo represents metadata for a secret (no value).
type SecretInfo struct {
	Key       string   `json:"key"`
	Tags      []string `json:"tags,omitempty"`
	ExpiresAt string   `json:"expires_at,omitempty"`
	HasNotes  bool     `json:"has_notes"`
	HasURL    bool     `json:"has_url"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// SecretExistsInput represents input for secret_exists tool.
type SecretExistsInput struct {
	Key string `json:"key"`
}

// SecretExistsOutput represents output for secret_exists tool.
type SecretExistsOutput struct {
	Exists    bool     `json:"exists"`
	Key       string   `json:"key"`
	Tags      []string `json:"tags,omitempty"`
	ExpiresAt string   `json:"expires_at,omitempty"`
	HasNotes  bool     `json:"has_notes"`
	HasURL    bool     `json:"has_url"`
	CreatedAt string   `json:"created_at,omitempty"`
	UpdatedAt string   `json:"updated_at,omitempty"`
}

// SecretGetMaskedInput represents input for secret_get_masked tool.
type SecretGetMaskedInput struct {
	Key string `json:"key"`
}

// SecretGetMaskedOutput represents output for secret_get_masked tool.
type SecretGetMaskedOutput struct {
	Key         string `json:"key"`
	MaskedValue string `json:"masked_value"`
	ValueLength int    `json:"value_length"`
}

// SecretRunInput represents input for secret_run tool.
type SecretRunInput struct {
	Keys      []string `json:"keys"`
	Command   string   `json:"command"`
	Args      []string `json:"args,omitempty"`
	Timeout   string   `json:"timeout,omitempty"`
	EnvPrefix string   `json:"env_prefix,omitempty"`
	Env       string   `json:"env,omitempty"` // Environment alias (e.g., "dev", "staging", "prod")
}

// SecretRunOutput represents output for secret_run tool.
type SecretRunOutput struct {
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"duration_ms"`
	Sanitized  bool   `json:"sanitized"`
}

// handleSecretList handles the secret_list tool call.
func (s *Server) handleSecretList(_ context.Context, _ *mcp.CallToolRequest, input SecretListInput) (*mcp.CallToolResult, SecretListOutput, error) {
	var entries []*vault.SecretEntry
	var err error

	switch {
	case input.Tag != "":
		// Filter by tag
		entries, err = s.vault.ListSecretsByTag(input.Tag)
		if err != nil {
			return nil, SecretListOutput{}, fmt.Errorf("failed to list secrets by tag: %w", err)
		}
	case input.ExpiringWithin != "":
		// Filter by expiration
		duration, parseErr := parseDuration(input.ExpiringWithin)
		if parseErr != nil {
			return nil, SecretListOutput{}, fmt.Errorf("invalid expiring_within format: %w", parseErr)
		}
		entries, err = s.vault.ListExpiringSecrets(duration)
		if err != nil {
			return nil, SecretListOutput{}, fmt.Errorf("failed to list expiring secrets: %w", err)
		}
	default:
		// List all secrets - need to get full entries for metadata
		keys, listErr := s.vault.ListSecrets()
		if listErr != nil {
			return nil, SecretListOutput{}, fmt.Errorf("failed to list secrets: %w", listErr)
		}
		// Get metadata for each key
		for _, key := range keys {
			entry, getErr := s.vault.GetSecret(key)
			if getErr != nil {
				continue // Skip entries we can't read
			}
			entry.Key = key
			entries = append(entries, entry)
		}
	}

	// Convert to output format (no values!)
	output := SecretListOutput{
		Secrets: make([]SecretInfo, 0, len(entries)),
	}

	for _, entry := range entries {
		info := SecretInfo{
			Key:       entry.Key,
			Tags:      entry.Tags,
			HasNotes:  entry.Metadata != nil && entry.Metadata.Notes != "",
			HasURL:    entry.Metadata != nil && entry.Metadata.URL != "",
			CreatedAt: entry.CreatedAt.Format(time.RFC3339),
			UpdatedAt: entry.UpdatedAt.Format(time.RFC3339),
		}
		if entry.ExpiresAt != nil {
			info.ExpiresAt = entry.ExpiresAt.Format(time.RFC3339)
		}
		output.Secrets = append(output.Secrets, info)
	}

	return nil, output, nil
}

// handleSecretExists handles the secret_exists tool call.
func (s *Server) handleSecretExists(_ context.Context, _ *mcp.CallToolRequest, input SecretExistsInput) (*mcp.CallToolResult, SecretExistsOutput, error) {
	if input.Key == "" {
		return nil, SecretExistsOutput{}, errors.New("key is required")
	}

	entry, err := s.vault.GetSecret(input.Key)
	if err != nil {
		if errors.Is(err, vault.ErrSecretNotFound) {
			return nil, SecretExistsOutput{
				Exists: false,
				Key:    input.Key,
			}, nil
		}
		return nil, SecretExistsOutput{}, fmt.Errorf("failed to get secret: %w", err)
	}

	output := SecretExistsOutput{
		Exists:    true,
		Key:       input.Key,
		Tags:      entry.Tags,
		HasNotes:  entry.Metadata != nil && entry.Metadata.Notes != "",
		HasURL:    entry.Metadata != nil && entry.Metadata.URL != "",
		CreatedAt: entry.CreatedAt.Format(time.RFC3339),
		UpdatedAt: entry.UpdatedAt.Format(time.RFC3339),
	}
	if entry.ExpiresAt != nil {
		output.ExpiresAt = entry.ExpiresAt.Format(time.RFC3339)
	}

	return nil, output, nil
}

// handleSecretGetMasked handles the secret_get_masked tool call.
func (s *Server) handleSecretGetMasked(_ context.Context, _ *mcp.CallToolRequest, input SecretGetMaskedInput) (*mcp.CallToolResult, SecretGetMaskedOutput, error) {
	if input.Key == "" {
		return nil, SecretGetMaskedOutput{}, errors.New("key is required")
	}

	entry, err := s.vault.GetSecret(input.Key)
	if err != nil {
		return nil, SecretGetMaskedOutput{}, fmt.Errorf("failed to get secret: %w", err)
	}

	// Mask the value per mcp-design-ja.md §3.3
	masked := maskValue(entry.Value)

	return nil, SecretGetMaskedOutput{
		Key:         input.Key,
		MaskedValue: masked,
		ValueLength: len(entry.Value),
	}, nil
}

// maskValue masks a secret value per mcp-design-ja.md §3.3
// | Length  | Format          | Example   |
// |---------|-----------------|-----------|
// | 1-4     | All *           | ****      |
// | 5-8     | Show last 2     | ******XY  |
// | 9+      | Show last 4     | ****WXYZ  |
func maskValue(value []byte) string {
	length := len(value)
	if length == 0 {
		return ""
	}

	switch {
	case length <= 4:
		return strings.Repeat("*", length)
	case length <= 8:
		suffix := string(value[length-2:])
		return strings.Repeat("*", length-2) + suffix
	default:
		suffix := string(value[length-4:])
		return strings.Repeat("*", length-4) + suffix
	}
}

// handleSecretRun handles the secret_run tool call.
func (s *Server) handleSecretRun(ctx context.Context, _ *mcp.CallToolRequest, input *SecretRunInput) (*mcp.CallToolResult, SecretRunOutput, error) {
	// Acquire semaphore for concurrency limiting per §6.4 (max 5 concurrent secret_run)
	select {
	case s.runSem <- struct{}{}:
		defer func() { <-s.runSem }()
	default:
		return nil, SecretRunOutput{}, errors.New("too many concurrent secret_run operations (max 5)")
	}

	// Validate required fields
	if len(input.Keys) == 0 {
		return nil, SecretRunOutput{}, errors.New("keys is required")
	}
	if input.Command == "" {
		return nil, SecretRunOutput{}, errors.New("command is required")
	}

	// Validate limits per mcp-design-ja.md §6.4
	if len(input.Keys) > 10 {
		return nil, SecretRunOutput{}, errors.New("too many keys (max 10)")
	}
	if len(input.Command) > 4096 {
		return nil, SecretRunOutput{}, errors.New("command too long (max 4096)")
	}
	if len(input.Args) > 100 {
		return nil, SecretRunOutput{}, errors.New("too many args (max 100)")
	}

	// Check policy
	if s.policy == nil {
		return nil, SecretRunOutput{}, errors.New("MCP policy not configured. Create ~/.secretctl/mcp-policy.yaml to enable secret_run")
	}

	allowed, reason := s.policy.IsCommandAllowed(input.Command)
	if !allowed {
		return nil, SecretRunOutput{}, fmt.Errorf("command not allowed by policy: %s", reason)
	}

	// Resolve environment aliases if env is specified
	keys := input.Keys
	if input.Env != "" {
		resolvedKeys, err := s.policy.ResolveAliasKeys(input.Env, input.Keys)
		if err != nil {
			return nil, SecretRunOutput{}, fmt.Errorf("failed to resolve environment alias '%s': %w", input.Env, err)
		}
		keys = resolvedKeys
	}

	// Parse timeout
	timeout := 5 * time.Minute // default per design
	if input.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(input.Timeout)
		if err != nil {
			// Try our custom duration parser
			timeout, err = parseDuration(input.Timeout)
			if err != nil {
				return nil, SecretRunOutput{}, fmt.Errorf("invalid timeout format: %w", err)
			}
		}
	}
	// Cap at 1 hour per design
	if timeout > time.Hour {
		timeout = time.Hour
	}

	// Collect secrets (using resolved keys if env alias was applied)
	secrets, err := s.collectSecrets(keys)
	if err != nil {
		return nil, SecretRunOutput{}, err
	}
	defer wipeSecrets(secrets)

	// Build environment
	env, err := s.buildEnvironment(secrets, input.EnvPrefix)
	if err != nil {
		return nil, SecretRunOutput{}, err
	}

	// Execute command
	startTime := time.Now()
	result, err := s.executeCommand(ctx, input.Command, input.Args, env, secrets, timeout)
	if err != nil {
		return nil, SecretRunOutput{}, err
	}

	result.DurationMs = time.Since(startTime).Milliseconds()
	result.Sanitized = true

	return nil, *result, nil
}

// secretData holds a secret key and its decrypted value
type secretData struct {
	key   string
	value []byte
}

// wipeSecrets zeroes out all secret values in memory.
// Uses runtime.KeepAlive to prevent compiler optimization from removing the zeroing.
func wipeSecrets(secrets []secretData) {
	for i := range secrets {
		for j := range secrets[i].value {
			secrets[i].value[j] = 0
		}
		// Prevent compiler from optimizing away the zeroing
		runtime.KeepAlive(secrets[i].value)
	}
}

// collectSecrets expands patterns and fetches secret values
func (s *Server) collectSecrets(patterns []string) ([]secretData, error) {
	allKeys, err := s.vault.ListSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	seen := make(map[string]bool)
	var matchedKeys []string

	for _, pattern := range patterns {
		matches, err := expandPattern(pattern, allKeys)
		if err != nil {
			return nil, err
		}
		for _, key := range matches {
			if !seen[key] {
				seen[key] = true
				matchedKeys = append(matchedKeys, key)
			}
		}
	}

	if len(matchedKeys) == 0 {
		return nil, errors.New("no secrets match the specified patterns")
	}

	now := time.Now()
	var secrets []secretData

	for _, key := range matchedKeys {
		entry, err := s.vault.GetSecret(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret '%s': %w", key, err)
		}

		// Check if secret is expired per design
		if entry.ExpiresAt != nil && entry.ExpiresAt.Before(now) {
			return nil, fmt.Errorf("secret '%s' has expired", key)
		}

		secrets = append(secrets, secretData{
			key:   key,
			value: entry.Value,
		})
	}

	return secrets, nil
}

// expandPattern expands a glob pattern against available keys
func expandPattern(pattern string, availableKeys []string) ([]string, error) {
	if _, err := filepath.Match(pattern, ""); err != nil {
		return nil, fmt.Errorf("invalid pattern '%s': %w", pattern, err)
	}

	hasGlob := strings.ContainsAny(pattern, "*?[")
	if !hasGlob {
		for _, key := range availableKeys {
			if key == pattern {
				return []string{pattern}, nil
			}
		}
		return nil, fmt.Errorf("secret '%s' not found", pattern)
	}

	var matches []string
	for _, key := range availableKeys {
		matched, err := filepath.Match(pattern, key)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, key)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no secrets match pattern '%s'", pattern)
	}

	return matches, nil
}

// blockedEnvVars per mcp-design-ja.md §6.3.3
var blockedEnvVars = map[string]bool{
	// secretctl related
	"SECRETCTL_PASSWORD": true,

	// Dynamic linker attacks
	"LD_PRELOAD":            true,
	"LD_LIBRARY_PATH":       true,
	"DYLD_INSERT_LIBRARIES": true,
	"DYLD_LIBRARY_PATH":     true,

	// Shell startup script execution
	"BASH_ENV":  true,
	"ENV":       true,
	"SHELLOPTS": true,
	"BASHOPTS":  true,

	// Script language auto-execution
	"PERL5OPT":      true,
	"PYTHONSTARTUP": true,
	"PYTHONPATH":    true,
	"RUBYOPT":       true,
	"NODE_OPTIONS":  true,

	// Other dangerous variables
	"IFS":        true,
	"CDPATH":     true,
	"GLOBIGNORE": true,
}

// safeEnvVars are the only environment variables we inherit
var safeEnvVars = map[string]bool{
	"PATH":    true,
	"HOME":    true,
	"USER":    true,
	"LOGNAME": true,
	"LANG":    true,
	"LC_ALL":  true,
	"TERM":    true,
	"TZ":      true,
}

// buildEnvironment creates environment variables from secrets
func (s *Server) buildEnvironment(secrets []secretData, prefix string) ([]string, error) {
	// Start with minimal safe environment per mcp-design-ja.md §6.3.2
	var env []string
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 && safeEnvVars[parts[0]] && !blockedEnvVars[parts[0]] {
			env = append(env, e)
		}
	}

	// Add secrets as environment variables
	for _, secret := range secrets {
		envName := keyToEnvName(secret.key)
		if prefix != "" {
			envName = prefix + envName
		}

		// Validate per §6.3.5
		if err := validateEnvName(envName); err != nil {
			return nil, fmt.Errorf("invalid environment variable name for key '%s': %w", secret.key, err)
		}

		// Check for blocked env vars
		if blockedEnvVars[envName] {
			return nil, fmt.Errorf("cannot use blocked environment variable name: %s", envName)
		}

		// Check for NUL bytes per §6.3.5
		if strings.ContainsRune(envName, '\x00') || bytes.ContainsRune(secret.value, '\x00') {
			return nil, fmt.Errorf("NUL byte detected in key '%s'", secret.key)
		}

		env = append(env, fmt.Sprintf("%s=%s", envName, string(secret.value)))
	}

	return env, nil
}

// keyToEnvName converts a secret key to an environment variable name
func keyToEnvName(key string) string {
	name := strings.ReplaceAll(key, "/", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return strings.ToUpper(name)
}

// validateEnvName validates environment variable name
func validateEnvName(name string) error {
	if name == "" {
		return errors.New("environment variable name cannot be empty")
	}
	first := name[0]
	if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_') {
		return errors.New("must start with a letter or underscore")
	}
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return fmt.Errorf("contains invalid character '%c'", c)
		}
	}
	return nil
}

// executeCommand runs the command with secrets in environment
func (s *Server) executeCommand(ctx context.Context, command string, args []string, env []string, secrets []secretData, timeout time.Duration) (*SecretRunOutput, error) {
	// Validate command path per §6.3.4
	if err := validateCommand(command); err != nil {
		return nil, err
	}

	// Validate args per §6.3.5
	if err := validateArgs(args); err != nil {
		return nil, err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Look up command
	cmdPath, err := exec.LookPath(command)
	if err != nil {
		return nil, fmt.Errorf("command not found: %s", command)
	}

	// Create command - NO shell (shell=false per §6.3.1)
	cmd := exec.CommandContext(ctx, cmdPath, args...)
	cmd.Env = env

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	err = cmd.Run()

	// Sanitize output
	sanitizer := newOutputSanitizer(secrets)
	sanitizedStdout := sanitizer.sanitize(stdout.Bytes())
	sanitizedStderr := sanitizer.sanitize(stderr.Bytes())

	// Limit output size per §6.4 (10MB)
	const maxOutputSize = 10 * 1024 * 1024
	if len(sanitizedStdout) > maxOutputSize {
		sanitizedStdout = sanitizedStdout[:maxOutputSize]
	}
	if len(sanitizedStderr) > maxOutputSize {
		sanitizedStderr = sanitizedStderr[:maxOutputSize]
	}

	result := &SecretRunOutput{
		ExitCode: 0,
		Stdout:   string(sanitizedStdout),
		Stderr:   string(sanitizedStderr),
	}

	if err != nil {
		var exitErr *exec.ExitError
		switch {
		case errors.As(err, &exitErr):
			result.ExitCode = exitErr.ExitCode()
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			return nil, fmt.Errorf("command timed out after %v", timeout)
		default:
			return nil, fmt.Errorf("command execution failed: %w", err)
		}
	}

	return result, nil
}

// validateCommand validates command path per §6.3.4
func validateCommand(cmd string) error {
	// Check for path traversal
	if strings.Contains(cmd, "..") || strings.Contains(cmd, "/./") {
		return errors.New("path traversal detected in command")
	}

	// Check for NUL byte
	if strings.ContainsRune(cmd, '\x00') {
		return errors.New("null byte detected in command")
	}

	return nil
}

// validateArgs validates command arguments per §6.3.5
func validateArgs(args []string) error {
	for i, arg := range args {
		if strings.ContainsRune(arg, '\x00') {
			return fmt.Errorf("null byte in argument %d", i)
		}
		if len(arg) > 32768 {
			return fmt.Errorf("argument %d too long", i)
		}
	}
	return nil
}

// outputSanitizer sanitizes output by replacing secret values
type outputSanitizer struct {
	replacements []secretReplacement
}

type secretReplacement struct {
	secret      []byte
	placeholder []byte
}

func newOutputSanitizer(secrets []secretData) *outputSanitizer {
	var replacements []secretReplacement
	for _, secret := range secrets {
		// Only sanitize values >= 4 bytes per design
		if len(secret.value) >= 4 {
			replacements = append(replacements, secretReplacement{
				secret:      secret.value,
				placeholder: []byte(fmt.Sprintf("[REDACTED:%s]", keyToEnvName(secret.key))),
			})
		}
	}
	return &outputSanitizer{replacements: replacements}
}

func (s *outputSanitizer) sanitize(data []byte) []byte {
	if len(s.replacements) == 0 {
		return data
	}
	result := data
	for _, r := range s.replacements {
		result = bytes.ReplaceAll(result, r.secret, r.placeholder)
	}
	return result
}

// parseDuration parses a duration string like "30d", "1y", "24h"
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("duration too short: %s", s)
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]

	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return 0, fmt.Errorf("invalid duration value: %s", valueStr)
	}

	switch unit {
	case 'h':
		return time.Duration(value) * time.Hour, nil
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case 'm':
		return time.Duration(value) * 30 * 24 * time.Hour, nil
	case 'y':
		return time.Duration(value) * 365 * 24 * time.Hour, nil
	default:
		return time.ParseDuration(s)
	}
}
