package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/forest6511/secretctl/pkg/audit"
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
	Key        string   `json:"key"`
	FieldCount int      `json:"field_count"`
	Tags       []string `json:"tags,omitempty"`
	ExpiresAt  string   `json:"expires_at,omitempty"`
	HasNotes   bool     `json:"has_notes"`
	HasURL     bool     `json:"has_url"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
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
	Key         string                 `json:"key"`
	MaskedValue string                 `json:"masked_value"`
	ValueLength int                    `json:"value_length"`
	FieldCount  int                    `json:"field_count"`
	Fields      map[string]MaskedField `json:"fields,omitempty"`
}

// MaskedField represents a field with its value (masked if sensitive).
type MaskedField struct {
	Value       string `json:"value"`
	Sensitive   bool   `json:"sensitive"`
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

// SecretListFieldsInput represents input for secret_list_fields tool.
type SecretListFieldsInput struct {
	Key string `json:"key"`
}

// SecretListFieldsOutput represents output for secret_list_fields tool.
type SecretListFieldsOutput struct {
	Key    string      `json:"key"`
	Fields []FieldInfo `json:"fields"`
}

// FieldInfo represents metadata for a single field (no value).
type FieldInfo struct {
	Name      string   `json:"name"`
	Sensitive bool     `json:"sensitive"`
	Hint      string   `json:"hint,omitempty"`
	Kind      string   `json:"kind,omitempty"`
	Aliases   []string `json:"aliases,omitempty"`
}

// SecretGetFieldInput represents input for secret_get_field tool.
type SecretGetFieldInput struct {
	Key   string `json:"key"`
	Field string `json:"field"`
}

// SecretGetFieldOutput represents output for secret_get_field tool.
type SecretGetFieldOutput struct {
	Key       string `json:"key"`
	Field     string `json:"field"`
	Value     string `json:"value"`
	Sensitive bool   `json:"sensitive"`
}

// SecretRunWithBindingsInput represents input for secret_run_with_bindings tool.
type SecretRunWithBindingsInput struct {
	Key     string   `json:"key"`
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Timeout string   `json:"timeout,omitempty"`
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
			_ = s.vault.Audit().LogError(audit.OpSecretList, audit.SourceMCP, "", "LIST_FAILED", err.Error())
			return nil, SecretListOutput{}, fmt.Errorf("failed to list secrets by tag: %w", err)
		}
	case input.ExpiringWithin != "":
		// Filter by expiration
		duration, parseErr := parseDuration(input.ExpiringWithin)
		if parseErr != nil {
			_ = s.vault.Audit().LogError(audit.OpSecretList, audit.SourceMCP, "", "INVALID_DURATION", parseErr.Error())
			return nil, SecretListOutput{}, fmt.Errorf("invalid expiring_within format: %w", parseErr)
		}
		entries, err = s.vault.ListExpiringSecrets(duration)
		if err != nil {
			_ = s.vault.Audit().LogError(audit.OpSecretList, audit.SourceMCP, "", "LIST_FAILED", err.Error())
			return nil, SecretListOutput{}, fmt.Errorf("failed to list expiring secrets: %w", err)
		}
	default:
		// List all secrets with metadata but WITHOUT decrypting values
		// This follows Option D+ principle: minimize plaintext exposure
		entries, err = s.vault.ListSecretsWithMetadata()
		if err != nil {
			_ = s.vault.Audit().LogError(audit.OpSecretList, audit.SourceMCP, "", "LIST_FAILED", err.Error())
			return nil, SecretListOutput{}, fmt.Errorf("failed to list secrets: %w", err)
		}
	}

	// Convert to output format (no values!)
	output := SecretListOutput{
		Secrets: make([]SecretInfo, 0, len(entries)),
	}

	for _, entry := range entries {
		info := SecretInfo{
			Key:        entry.Key,
			FieldCount: entry.FieldCount,
			Tags:       entry.Tags,
			HasNotes:   entry.Metadata != nil && entry.Metadata.Notes != "",
			HasURL:     entry.Metadata != nil && entry.Metadata.URL != "",
			CreatedAt:  entry.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  entry.UpdatedAt.Format(time.RFC3339),
		}
		if entry.ExpiresAt != nil {
			info.ExpiresAt = entry.ExpiresAt.Format(time.RFC3339)
		}
		output.Secrets = append(output.Secrets, info)
	}

	// Log successful list operation
	_ = s.vault.Audit().LogSuccess(audit.OpSecretList, audit.SourceMCP, "")

	return nil, output, nil
}

// handleSecretExists handles the secret_exists tool call.
func (s *Server) handleSecretExists(_ context.Context, _ *mcp.CallToolRequest, input SecretExistsInput) (*mcp.CallToolResult, SecretExistsOutput, error) {
	if input.Key == "" {
		_ = s.vault.Audit().LogError(audit.OpSecretExists, audit.SourceMCP, "", "INVALID_INPUT", "key is required")
		return nil, SecretExistsOutput{}, errors.New("key is required")
	}

	entry, err := s.vault.GetSecret(input.Key)
	if err != nil {
		if errors.Is(err, vault.ErrSecretNotFound) {
			// Log successful check (key doesn't exist is a valid result)
			_ = s.vault.Audit().LogSuccess(audit.OpSecretExists, audit.SourceMCP, input.Key)
			return nil, SecretExistsOutput{
				Exists: false,
				Key:    input.Key,
			}, nil
		}
		_ = s.vault.Audit().LogError(audit.OpSecretExists, audit.SourceMCP, input.Key, "GET_FAILED", err.Error())
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

	// Log successful exists check
	_ = s.vault.Audit().LogSuccess(audit.OpSecretExists, audit.SourceMCP, input.Key)

	return nil, output, nil
}

// handleSecretGetMasked handles the secret_get_masked tool call.
// For multi-field secrets, returns all fields with masking for sensitive ones.
// For legacy single-value secrets, maintains backward compatibility.
func (s *Server) handleSecretGetMasked(_ context.Context, _ *mcp.CallToolRequest, input SecretGetMaskedInput) (*mcp.CallToolResult, SecretGetMaskedOutput, error) {
	if input.Key == "" {
		_ = s.vault.Audit().LogError(audit.OpSecretGetMasked, audit.SourceMCP, "", "INVALID_INPUT", "key is required")
		return nil, SecretGetMaskedOutput{}, errors.New("key is required")
	}

	entry, err := s.vault.GetSecret(input.Key)
	if err != nil {
		_ = s.vault.Audit().LogError(audit.OpSecretGetMasked, audit.SourceMCP, input.Key, "GET_FAILED", err.Error())
		return nil, SecretGetMaskedOutput{}, fmt.Errorf("failed to get secret: %w", err)
	}

	// Build output with backward compatibility
	output := SecretGetMaskedOutput{
		Key:         input.Key,
		MaskedValue: maskValue(entry.Value),
		ValueLength: len(entry.Value),
		FieldCount:  1, // Default for legacy secrets
	}

	// For multi-field secrets, include all fields with appropriate masking
	if len(entry.Fields) > 0 {
		output.FieldCount = len(entry.Fields)
		output.Fields = make(map[string]MaskedField, len(entry.Fields))

		for name, field := range entry.Fields {
			mf := MaskedField{
				Sensitive:   field.Sensitive,
				ValueLength: len(field.Value),
			}

			if field.Sensitive {
				// Mask sensitive field values
				mf.Value = maskValue([]byte(field.Value))
			} else {
				// Show non-sensitive field values in full
				mf.Value = field.Value
			}

			output.Fields[name] = mf
		}
	}

	// Log successful masked get
	_ = s.vault.Audit().LogSuccess(audit.OpSecretGetMasked, audit.SourceMCP, input.Key)

	return nil, output, nil
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
		_ = s.vault.Audit().LogError(audit.OpSecretRun, audit.SourceMCP, "", "RATE_LIMITED", "too many concurrent operations")
		return nil, SecretRunOutput{}, errors.New("too many concurrent secret_run operations (max 5)")
	}

	// Validate required fields
	if len(input.Keys) == 0 {
		_ = s.vault.Audit().LogError(audit.OpSecretRun, audit.SourceMCP, "", "INVALID_INPUT", "keys is required")
		return nil, SecretRunOutput{}, errors.New("keys is required")
	}
	if input.Command == "" {
		_ = s.vault.Audit().LogError(audit.OpSecretRun, audit.SourceMCP, "", "INVALID_INPUT", "command is required")
		return nil, SecretRunOutput{}, errors.New("command is required")
	}

	// Validate limits per mcp-design-ja.md §6.4
	if len(input.Keys) > 10 {
		_ = s.vault.Audit().LogError(audit.OpSecretRun, audit.SourceMCP, "", "INVALID_INPUT", "too many keys")
		return nil, SecretRunOutput{}, errors.New("too many keys (max 10)")
	}
	if len(input.Command) > 4096 {
		_ = s.vault.Audit().LogError(audit.OpSecretRun, audit.SourceMCP, "", "INVALID_INPUT", "command too long")
		return nil, SecretRunOutput{}, errors.New("command too long (max 4096)")
	}
	if len(input.Args) > 100 {
		_ = s.vault.Audit().LogError(audit.OpSecretRun, audit.SourceMCP, "", "INVALID_INPUT", "too many args")
		return nil, SecretRunOutput{}, errors.New("too many args (max 100)")
	}

	// Check policy
	if s.policy == nil {
		_ = s.vault.Audit().LogDenied(audit.OpSecretRunDenied, audit.SourceMCP, "", "NO_POLICY")
		return nil, SecretRunOutput{}, errors.New("MCP policy not configured. Create ~/.secretctl/mcp-policy.yaml to enable secret_run")
	}

	// SECURITY: Resolve command path BEFORE policy check to prevent PATH manipulation attacks.
	// This ensures we check the policy against the actual binary that will be executed,
	// not just the command name which could be spoofed via PATH.
	resolvedCmd, err := ResolveAndValidateCommand(input.Command)
	if err != nil {
		return nil, SecretRunOutput{}, fmt.Errorf("command validation failed: %w", err)
	}

	// Check policy against BOTH the original command name and the resolved path
	// This allows policies to specify either "curl" or "/usr/bin/curl"
	allowed, reason := s.policy.IsCommandAllowed(input.Command)
	if !allowed {
		// Also try with the resolved path in case policy uses absolute paths
		allowed, reason = s.policy.IsCommandAllowed(resolvedCmd)
	}
	if !allowed {
		_ = s.vault.Audit().LogDenied(audit.OpSecretRunDenied, audit.SourceMCP, input.Command, reason)
		return nil, SecretRunOutput{}, fmt.Errorf("command not allowed by policy: %s", reason)
	}

	// Resolve environment aliases if env is specified
	keys := input.Keys
	if input.Env != "" {
		resolvedKeys, err := s.policy.ResolveAliasKeys(input.Env, input.Keys)
		if err != nil {
			_ = s.vault.Audit().LogError(audit.OpSecretRun, audit.SourceMCP, "", "ALIAS_FAILED", err.Error())
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
				_ = s.vault.Audit().LogError(audit.OpSecretRun, audit.SourceMCP, "", "INVALID_TIMEOUT", err.Error())
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
		_ = s.vault.Audit().LogError(audit.OpSecretRun, audit.SourceMCP, "", "SECRET_COLLECT_FAILED", err.Error())
		return nil, SecretRunOutput{}, err
	}
	defer wipeSecrets(secrets)

	// Build environment
	env, err := s.buildEnvironment(secrets, input.EnvPrefix)
	if err != nil {
		_ = s.vault.Audit().LogError(audit.OpSecretRun, audit.SourceMCP, "", "ENV_BUILD_FAILED", err.Error())
		return nil, SecretRunOutput{}, err
	}

	// Execute command using the pre-resolved and validated path
	startTime := time.Now()
	result, err := s.executeCommand(ctx, resolvedCmd, input.Args, env, secrets, timeout)
	if err != nil {
		_ = s.vault.Audit().LogError(audit.OpSecretRun, audit.SourceMCP, input.Command, "EXEC_FAILED", err.Error())
		return nil, SecretRunOutput{}, err
	}

	result.DurationMs = time.Since(startTime).Milliseconds()
	result.Sanitized = true

	// Log successful command execution
	_ = s.vault.Audit().LogSuccess(audit.OpSecretRun, audit.SourceMCP, input.Command)

	return nil, *result, nil
}

// handleSecretListFields handles the secret_list_fields tool call.
func (s *Server) handleSecretListFields(_ context.Context, _ *mcp.CallToolRequest, input SecretListFieldsInput) (*mcp.CallToolResult, SecretListFieldsOutput, error) {
	if input.Key == "" {
		_ = s.vault.Audit().LogError(audit.OpSecretListFields, audit.SourceMCP, "", "INVALID_INPUT", "key is required")
		return nil, SecretListFieldsOutput{}, errors.New("key is required")
	}

	entry, err := s.vault.GetSecret(input.Key)
	if err != nil {
		_ = s.vault.Audit().LogError(audit.OpSecretListFields, audit.SourceMCP, input.Key, "GET_FAILED", err.Error())
		return nil, SecretListFieldsOutput{}, fmt.Errorf("failed to get secret: %w", err)
	}

	// Build field info list (no values!)
	output := SecretListFieldsOutput{
		Key:    input.Key,
		Fields: make([]FieldInfo, 0, len(entry.Fields)),
	}

	for name, field := range entry.Fields {
		info := FieldInfo{
			Name:      name,
			Sensitive: field.Sensitive,
			Hint:      field.Hint,
			Kind:      field.Kind,
			Aliases:   field.Aliases,
		}
		output.Fields = append(output.Fields, info)
	}

	// Sort fields by name for consistent output
	sort.Slice(output.Fields, func(i, j int) bool {
		return output.Fields[i].Name < output.Fields[j].Name
	})

	// Log successful list fields operation
	_ = s.vault.Audit().LogSuccess(audit.OpSecretListFields, audit.SourceMCP, input.Key)

	return nil, output, nil
}

// handleSecretGetField handles the secret_get_field tool call.
// Per Option D+: Only non-sensitive fields can be retrieved via MCP.
func (s *Server) handleSecretGetField(_ context.Context, _ *mcp.CallToolRequest, input SecretGetFieldInput) (*mcp.CallToolResult, SecretGetFieldOutput, error) {
	if input.Key == "" {
		_ = s.vault.Audit().LogError(audit.OpSecretGetField, audit.SourceMCP, "", "INVALID_INPUT", "key is required")
		return nil, SecretGetFieldOutput{}, errors.New("key is required")
	}
	if input.Field == "" {
		_ = s.vault.Audit().LogError(audit.OpSecretGetField, audit.SourceMCP, input.Key, "INVALID_INPUT", "field is required")
		return nil, SecretGetFieldOutput{}, errors.New("field is required")
	}

	entry, err := s.vault.GetSecret(input.Key)
	if err != nil {
		_ = s.vault.Audit().LogError(audit.OpSecretGetField, audit.SourceMCP, input.Key, "GET_FAILED", err.Error())
		return nil, SecretGetFieldOutput{}, fmt.Errorf("failed to get secret: %w", err)
	}

	// Resolve field name (supports aliases)
	canonicalName, field, err := vault.ResolveFieldName(entry.Fields, input.Field)
	if err != nil {
		_ = s.vault.Audit().LogError(audit.OpSecretGetField, audit.SourceMCP, input.Key, "FIELD_NOT_FOUND", input.Field)
		return nil, SecretGetFieldOutput{}, fmt.Errorf("field '%s' not found in secret '%s'", input.Field, input.Key)
	}

	// Option D+ enforcement: Reject sensitive fields
	if field.Sensitive {
		_ = s.vault.Audit().LogDenied(audit.OpSecretGetFieldDenied, audit.SourceMCP, input.Key, fmt.Sprintf("sensitive field: %s", canonicalName))
		return nil, SecretGetFieldOutput{}, fmt.Errorf("field '%s' is marked as sensitive and cannot be retrieved via MCP (Option D+ policy)", canonicalName)
	}

	// Log successful get field operation
	_ = s.vault.Audit().LogSuccess(audit.OpSecretGetField, audit.SourceMCP, fmt.Sprintf("%s.%s", input.Key, canonicalName))

	return nil, SecretGetFieldOutput{
		Key:       input.Key,
		Field:     canonicalName,
		Value:     field.Value,
		Sensitive: field.Sensitive,
	}, nil
}

// handleSecretRunWithBindings handles the secret_run_with_bindings tool call.
// Uses the secret's Bindings map to inject environment variables.
func (s *Server) handleSecretRunWithBindings(ctx context.Context, _ *mcp.CallToolRequest, input *SecretRunWithBindingsInput) (*mcp.CallToolResult, SecretRunOutput, error) {
	// Acquire semaphore for concurrency limiting
	select {
	case s.runSem <- struct{}{}:
		defer func() { <-s.runSem }()
	default:
		_ = s.vault.Audit().LogError(audit.OpSecretRunWithBindings, audit.SourceMCP, "", "RATE_LIMITED", "too many concurrent operations")
		return nil, SecretRunOutput{}, errors.New("too many concurrent secret_run operations (max 5)")
	}

	// Validate required fields
	if input.Key == "" {
		_ = s.vault.Audit().LogError(audit.OpSecretRunWithBindings, audit.SourceMCP, "", "INVALID_INPUT", "key is required")
		return nil, SecretRunOutput{}, errors.New("key is required")
	}
	if input.Command == "" {
		_ = s.vault.Audit().LogError(audit.OpSecretRunWithBindings, audit.SourceMCP, "", "INVALID_INPUT", "command is required")
		return nil, SecretRunOutput{}, errors.New("command is required")
	}
	if len(input.Command) > 4096 {
		_ = s.vault.Audit().LogError(audit.OpSecretRunWithBindings, audit.SourceMCP, "", "INVALID_INPUT", "command too long")
		return nil, SecretRunOutput{}, errors.New("command too long (max 4096)")
	}
	if len(input.Args) > 100 {
		_ = s.vault.Audit().LogError(audit.OpSecretRunWithBindings, audit.SourceMCP, "", "INVALID_INPUT", "too many args")
		return nil, SecretRunOutput{}, errors.New("too many args (max 100)")
	}

	// Check policy
	if s.policy == nil {
		_ = s.vault.Audit().LogDenied(audit.OpSecretRunWithBindings, audit.SourceMCP, "", "NO_POLICY")
		return nil, SecretRunOutput{}, errors.New("MCP policy not configured. Create ~/.secretctl/mcp-policy.yaml to enable secret_run")
	}

	// Resolve and validate command
	resolvedCmd, err := ResolveAndValidateCommand(input.Command)
	if err != nil {
		_ = s.vault.Audit().LogError(audit.OpSecretRunWithBindings, audit.SourceMCP, input.Command, "CMD_VALIDATION_FAILED", err.Error())
		return nil, SecretRunOutput{}, fmt.Errorf("command validation failed: %w", err)
	}

	// Check policy
	allowed, reason := s.policy.IsCommandAllowed(input.Command)
	if !allowed {
		allowed, reason = s.policy.IsCommandAllowed(resolvedCmd)
	}
	if !allowed {
		_ = s.vault.Audit().LogDenied(audit.OpSecretRunWithBindings, audit.SourceMCP, input.Command, reason)
		return nil, SecretRunOutput{}, fmt.Errorf("command not allowed by policy: %s", reason)
	}

	// Get the secret
	entry, err := s.vault.GetSecret(input.Key)
	if err != nil {
		_ = s.vault.Audit().LogError(audit.OpSecretRunWithBindings, audit.SourceMCP, input.Key, "GET_FAILED", err.Error())
		return nil, SecretRunOutput{}, fmt.Errorf("failed to get secret: %w", err)
	}

	// Check if secret has bindings
	if len(entry.Bindings) == 0 {
		_ = s.vault.Audit().LogError(audit.OpSecretRunWithBindings, audit.SourceMCP, input.Key, "NO_BINDINGS", "secret has no bindings defined")
		return nil, SecretRunOutput{}, fmt.Errorf("secret '%s' has no bindings defined. Use 'secretctl set %s --binding ENV=field' to add bindings", input.Key, input.Key)
	}

	// Check expiration
	if entry.ExpiresAt != nil && entry.ExpiresAt.Before(time.Now()) {
		_ = s.vault.Audit().LogError(audit.OpSecretRunWithBindings, audit.SourceMCP, input.Key, "EXPIRED", "secret has expired")
		return nil, SecretRunOutput{}, fmt.Errorf("secret '%s' has expired", input.Key)
	}

	// Parse timeout
	timeout := 5 * time.Minute
	if input.Timeout != "" {
		timeout, err = time.ParseDuration(input.Timeout)
		if err != nil {
			timeout, err = parseDuration(input.Timeout)
			if err != nil {
				_ = s.vault.Audit().LogError(audit.OpSecretRunWithBindings, audit.SourceMCP, "", "INVALID_TIMEOUT", err.Error())
				return nil, SecretRunOutput{}, fmt.Errorf("invalid timeout format: %w", err)
			}
		}
	}
	if timeout > time.Hour {
		timeout = time.Hour
	}

	// Build environment from bindings
	env, secrets, err := s.buildEnvironmentFromBindings(entry)
	if err != nil {
		_ = s.vault.Audit().LogError(audit.OpSecretRunWithBindings, audit.SourceMCP, input.Key, "ENV_BUILD_FAILED", err.Error())
		return nil, SecretRunOutput{}, err
	}
	defer wipeBindingSecrets(secrets)
	defer wipeEnvSlice(env)

	// Execute command
	startTime := time.Now()
	result, err := s.executeCommandWithBindings(ctx, resolvedCmd, input.Args, env, secrets, timeout)
	if err != nil {
		_ = s.vault.Audit().LogError(audit.OpSecretRunWithBindings, audit.SourceMCP, input.Command, "EXEC_FAILED", err.Error())
		return nil, SecretRunOutput{}, err
	}

	result.DurationMs = time.Since(startTime).Milliseconds()
	result.Sanitized = true

	// Log successful command execution
	_ = s.vault.Audit().LogSuccess(audit.OpSecretRunWithBindings, audit.SourceMCP, fmt.Sprintf("%s:%s", input.Key, input.Command))

	return nil, *result, nil
}

// bindingSecretData holds an environment variable name and its value for binding-based injection.
type bindingSecretData struct {
	envVar string
	value  []byte
}

// wipeBindingSecrets zeroes out all secret values in memory.
func wipeBindingSecrets(secrets []bindingSecretData) {
	for i := range secrets {
		for j := range secrets[i].value {
			secrets[i].value[j] = 0
		}
		runtime.KeepAlive(secrets[i].value)
	}
}

// wipeEnvSlice clears environment variable string references.
// NOTE: Due to Go's immutable strings, this cannot truly wipe the underlying
// memory - it only clears references to allow GC collection. For defense-in-depth,
// we also use bindingSecretData with []byte for the actual secret values which
// ARE properly wiped in wipeBindingSecrets. The env slice is cleared as a
// best-effort secondary measure.
func wipeEnvSlice(env []string) {
	for i := range env {
		env[i] = ""
	}
	runtime.KeepAlive(env)
}

// wipeBuffer zeroes out the contents of a bytes.Buffer.
// This is used to clear buffers that may contain sensitive data from command output.
func wipeBuffer(buf *bytes.Buffer) {
	b := buf.Bytes()
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
	buf.Reset()
}

// buildEnvironmentFromBindings creates environment variables from secret bindings.
func (s *Server) buildEnvironmentFromBindings(entry *vault.SecretEntry) ([]string, []bindingSecretData, error) {
	// Start with minimal safe environment
	var env []string
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 && safeEnvVars[parts[0]] && !blockedEnvVars[parts[0]] {
			env = append(env, e)
		}
	}

	var secrets []bindingSecretData

	// Add secrets as environment variables based on bindings
	for envVar, fieldName := range entry.Bindings {
		// Resolve field name (supports aliases)
		_, field, err := vault.ResolveFieldName(entry.Fields, fieldName)
		if err != nil {
			return nil, nil, fmt.Errorf("binding '%s' references non-existent field '%s'", envVar, fieldName)
		}

		// Validate environment variable name
		if err := validateEnvName(envVar); err != nil {
			return nil, nil, fmt.Errorf("invalid environment variable name '%s': %w", envVar, err)
		}

		// Check for blocked env vars
		if blockedEnvVars[envVar] {
			return nil, nil, fmt.Errorf("cannot use blocked environment variable name: %s", envVar)
		}

		// Check for NUL bytes
		if strings.ContainsRune(envVar, '\x00') || strings.ContainsRune(field.Value, '\x00') {
			return nil, nil, fmt.Errorf("NUL byte detected in binding '%s'", envVar)
		}

		env = append(env, fmt.Sprintf("%s=%s", envVar, field.Value))
		secrets = append(secrets, bindingSecretData{
			envVar: envVar,
			value:  []byte(field.Value),
		})
	}

	return env, secrets, nil
}

// executeCommandWithBindings runs the command with binding-based environment variables.
func (s *Server) executeCommandWithBindings(ctx context.Context, command string, args []string, env []string, secrets []bindingSecretData, timeout time.Duration) (*SecretRunOutput, error) {
	// Validate command path
	if err := validateCommand(command); err != nil {
		return nil, err
	}

	// Validate args
	if err := validateArgs(args); err != nil {
		return nil, err
	}

	// Verify command is an absolute path
	if !filepath.IsAbs(command) {
		return nil, fmt.Errorf("security error: command must be an absolute path, got: %s", command)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = env

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	err := cmd.Run()

	// Convert bindingSecretData to secretData for sanitization
	secretDataList := make([]secretData, len(secrets))
	for i, s := range secrets {
		secretDataList[i] = secretData{
			key:   s.envVar,
			value: s.value,
		}
	}

	// Sanitize output
	sanitizer := newOutputSanitizer(secretDataList)
	sanitizedStdout := sanitizer.sanitize(stdout.Bytes())
	sanitizedStderr := sanitizer.sanitize(stderr.Bytes())

	// Wipe original buffers that may contain secrets in raw output
	wipeBuffer(&stdout)
	wipeBuffer(&stderr)

	// Limit output size
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

// executeCommand runs the command with secrets in environment.
// IMPORTANT: The command parameter MUST be an absolute path that has already been
// resolved and validated by ResolveAndValidateCommand. This function does NOT
// perform path lookup to prevent PATH manipulation attacks.
func (s *Server) executeCommand(ctx context.Context, command string, args []string, env []string, secrets []secretData, timeout time.Duration) (*SecretRunOutput, error) {
	// Validate command path per §6.3.4
	if err := validateCommand(command); err != nil {
		return nil, err
	}

	// Validate args per §6.3.5
	if err := validateArgs(args); err != nil {
		return nil, err
	}

	// Verify command is an absolute path (security check - should always be true
	// since we require pre-resolution via ResolveAndValidateCommand)
	if !filepath.IsAbs(command) {
		return nil, fmt.Errorf("security error: command must be an absolute path, got: %s", command)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create command directly with the pre-resolved absolute path
	// NO LookPath here - the path was already resolved and validated
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = env

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	err := cmd.Run()

	// Sanitize output
	sanitizer := newOutputSanitizer(secrets)
	sanitizedStdout := sanitizer.sanitize(stdout.Bytes())
	sanitizedStderr := sanitizer.sanitize(stderr.Bytes())

	// Wipe original buffers that may contain secrets in raw output
	wipeBuffer(&stdout)
	wipeBuffer(&stderr)

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

// outputSanitizer sanitizes output by replacing secret values and their encoded forms.
// This prevents secrets from leaking through MCP output in any form.
type outputSanitizer struct {
	replacements []secretReplacement
}

type secretReplacement struct {
	secret      []byte
	placeholder []byte
}

// newOutputSanitizer creates a sanitizer that will redact:
// - The raw secret value (all lengths, not just >= 4 bytes)
// - Base64-encoded forms (padded and raw/unpadded, standard and URL-safe)
// - URL-encoded forms (QueryEscape and PathEscape styles)
// - Hex-encoded forms (uppercase and lowercase, with and without 0x prefix)
func newOutputSanitizer(secrets []secretData) *outputSanitizer {
	// Use a map to deduplicate encoded forms
	seen := make(map[string]bool)
	var replacements []secretReplacement

	addReplacement := func(secret []byte, placeholder []byte) {
		key := string(secret)
		if seen[key] || len(secret) == 0 {
			return
		}
		seen[key] = true
		replacements = append(replacements, secretReplacement{
			secret:      secret,
			placeholder: placeholder,
		})
	}

	for _, secret := range secrets {
		placeholder := []byte(fmt.Sprintf("[REDACTED:%s]", keyToEnvName(secret.key)))

		// Always add the raw value replacement, regardless of length
		// Short secrets are security-critical too (e.g., PIN codes, short API keys)
		addReplacement(secret.value, placeholder)

		// Only generate encoded forms for secrets with content
		if len(secret.value) == 0 {
			continue
		}

		// Base64-encoded forms (padded)
		addReplacement([]byte(base64.StdEncoding.EncodeToString(secret.value)), placeholder)
		addReplacement([]byte(base64.URLEncoding.EncodeToString(secret.value)), placeholder)

		// Base64-encoded forms (raw/unpadded) - catches JWT segments, etc.
		addReplacement([]byte(base64.RawStdEncoding.EncodeToString(secret.value)), placeholder)
		addReplacement([]byte(base64.RawURLEncoding.EncodeToString(secret.value)), placeholder)

		// URL-encoded forms
		addReplacement([]byte(url.QueryEscape(string(secret.value))), placeholder)    // space as +
		addReplacement([]byte(url.PathEscape(string(secret.value))), placeholder)     // space as %20
		addReplacement([]byte(percentEncodeLower(string(secret.value))), placeholder) // lowercase %xx

		// Hex-encoded forms (plain)
		hexLower := hex.EncodeToString(secret.value)
		hexUpper := strings.ToUpper(hexLower)
		addReplacement([]byte(hexLower), placeholder)
		addReplacement([]byte(hexUpper), placeholder)

		// Hex-encoded forms with 0x prefix (common in debug output)
		addReplacement([]byte("0x"+hexLower), placeholder)
		addReplacement([]byte("0X"+hexUpper), placeholder)
	}

	// Sort replacements by length (longest first) to avoid partial replacements
	// e.g., if we have "secret" and "secretkey", replace "secretkey" first
	sortReplacementsByLength(replacements)

	return &outputSanitizer{replacements: replacements}
}

// percentEncodeLower generates URL encoding with lowercase hex digits
// (some systems output %2f instead of %2F)
func percentEncodeLower(s string) string {
	return strings.ToLower(url.QueryEscape(s))
}

// sortReplacementsByLength sorts replacements by secret length in descending order.
// This ensures longer matches are replaced first, preventing partial replacement issues.
func sortReplacementsByLength(replacements []secretReplacement) {
	sort.Slice(replacements, func(i, j int) bool {
		return len(replacements[i].secret) > len(replacements[j].secret)
	})
}

func (s *outputSanitizer) sanitize(data []byte) []byte {
	// Always make a copy to avoid aliasing issues when the original buffer is wiped.
	// This ensures wipeBuffer on the source doesn't affect the sanitized result.
	result := make([]byte, len(data))
	copy(result, data)

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
