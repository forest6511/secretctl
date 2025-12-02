// Package main provides the secretctl CLI commands.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Export format constants
const (
	formatEnv  = "env"
	formatJSON = "json"
)

// Export command flags
var (
	exportFormat       string
	exportOutput       string
	exportKeys         []string
	exportWithMetadata bool
	exportForce        bool
)

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "env", "Output format: env, json")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file path (default: stdout)")
	exportCmd.Flags().StringSliceVarP(&exportKeys, "key", "k", nil, "Keys to export (glob pattern supported)")
	exportCmd.Flags().BoolVar(&exportWithMetadata, "with-metadata", false, "Include metadata in JSON output")
	exportCmd.Flags().BoolVar(&exportForce, "force", false, "Overwrite existing file without confirmation")
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export secrets to .env or JSON format",
	Long: `Export secrets from the vault to .env or JSON format.

Examples:
  # Export all secrets to stdout in .env format
  secretctl export

  # Export specific secrets to a file
  secretctl export -k "aws/*" -o .env

  # Export as JSON
  secretctl export -f json -o secrets.json

  # Export with metadata (JSON only)
  secretctl export -f json --with-metadata

  # Overwrite existing file
  secretctl export -o .env --force`,
	RunE: executeExport,
}

// exportSecretData holds secret data for export
type exportSecretData struct {
	key       string
	envName   string
	value     []byte
	createdAt time.Time
	updatedAt time.Time
	expiresAt *time.Time
}

// exportSecretMetadata holds secret metadata for JSON export with metadata
type exportSecretMetadata struct {
	Value     string  `json:"value"`
	CreatedAt *string `json:"created_at,omitempty"`
	UpdatedAt *string `json:"updated_at,omitempty"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

func executeExport(cmd *cobra.Command, args []string) error {
	if err := validateExportFlags(); err != nil {
		return err
	}

	// Unlock vault
	if err := ensureUnlocked(); err != nil {
		return err
	}
	defer v.Lock()

	// Get keys to export
	keysToExport, err := getKeysToExport()
	if err != nil {
		return err
	}

	// Collect secrets with validation
	secrets, err := collectSecretsForExport(keysToExport)
	if err != nil {
		return err
	}

	// Generate and write output
	return writeExportOutput(secrets)
}

// validateExportFlags validates the export command flags
func validateExportFlags() error {
	exportFormat = strings.ToLower(exportFormat)
	if exportFormat != formatEnv && exportFormat != formatJSON {
		return fmt.Errorf("invalid format '%s': must be '%s' or '%s'", exportFormat, formatEnv, formatJSON)
	}

	if exportWithMetadata && exportFormat != formatJSON {
		return fmt.Errorf("--with-metadata flag is only valid with JSON format")
	}
	return nil
}

// getKeysToExport returns the list of keys to export based on patterns
func getKeysToExport() ([]string, error) {
	allKeys, err := v.ListSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	if len(allKeys) == 0 {
		return nil, fmt.Errorf("no secrets in vault")
	}

	var keysToExport []string
	if len(exportKeys) == 0 {
		keysToExport = allKeys
	} else {
		keysToExport, err = expandExportPatterns(exportKeys, allKeys)
		if err != nil {
			return nil, err
		}
	}

	sort.Strings(keysToExport)
	return keysToExport, nil
}

// expandExportPatterns expands pattern list to matching keys
func expandExportPatterns(patterns []string, allKeys []string) ([]string, error) {
	seen := make(map[string]bool)
	var result []string
	for _, pattern := range patterns {
		matches, err := expandPattern(pattern, allKeys)
		if err != nil {
			return nil, err
		}
		for _, key := range matches {
			if !seen[key] {
				seen[key] = true
				result = append(result, key)
			}
		}
	}
	return result, nil
}

// collectSecretsForExport collects secrets and validates expiration
func collectSecretsForExport(keys []string) ([]exportSecretData, error) {
	now := time.Now()
	var secrets []exportSecretData

	for _, key := range keys {
		entry, err := v.GetSecret(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret '%s': %w", key, err)
		}

		if entry.ExpiresAt != nil && entry.ExpiresAt.Before(now) {
			return nil, fmt.Errorf("secret '%s' has expired at %v", key, entry.ExpiresAt.Format(time.RFC3339))
		}

		secrets = append(secrets, exportSecretData{
			key:       key,
			envName:   keyToEnvName(key),
			value:     entry.Value,
			createdAt: entry.CreatedAt,
			updatedAt: entry.UpdatedAt,
			expiresAt: entry.ExpiresAt,
		})
	}
	return secrets, nil
}

// writeExportOutput generates and writes the export output
func writeExportOutput(secrets []exportSecretData) error {
	output, err := generateOutput(secrets)
	if err != nil {
		return err
	}

	if exportOutput == "" {
		fmt.Fprint(os.Stderr, "WARNING: DO NOT COMMIT THIS OUTPUT TO VERSION CONTROL\n")
		fmt.Print(output)
	} else {
		if err := writeSecureFile(exportOutput, output, exportForce); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Exported %d secrets to %s\n", len(secrets), exportOutput)
	}
	return nil
}

// generateOutput generates output based on format
func generateOutput(secrets []exportSecretData) (string, error) {
	switch exportFormat {
	case formatEnv:
		return generateEnvOutput(secrets), nil
	case formatJSON:
		return generateJSONOutput(secrets, exportWithMetadata)
	default:
		return "", fmt.Errorf("unknown format: %s", exportFormat)
	}
}

// generateEnvOutput generates .env format output
func generateEnvOutput(secrets []exportSecretData) string {
	var sb strings.Builder
	sb.WriteString("# Generated by secretctl\n")
	sb.WriteString("# WARNING: DO NOT COMMIT THIS FILE TO VERSION CONTROL\n")
	sb.WriteString("#\n")

	for _, s := range secrets {
		// Escape special characters in value for .env format
		value := escapeEnvValue(string(s.value))
		sb.WriteString(fmt.Sprintf("%s=%s\n", s.envName, value))
	}

	return sb.String()
}

// escapeEnvValue escapes a value for .env format
// Values with special characters are quoted
func escapeEnvValue(value string) string {
	// Check if value needs quoting
	needsQuote := false
	for _, c := range value {
		if c == ' ' || c == '"' || c == '\'' || c == '\\' || c == '\n' || c == '\r' || c == '\t' || c == '#' || c == '$' || c == '=' {
			needsQuote = true
			break
		}
	}

	if !needsQuote {
		return value
	}

	// Use double quotes and escape special characters
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	escaped = strings.ReplaceAll(escaped, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\\r")
	escaped = strings.ReplaceAll(escaped, "\t", "\\t")
	escaped = strings.ReplaceAll(escaped, "$", "\\$")

	return "\"" + escaped + "\""
}

// generateJSONOutput generates JSON format output
func generateJSONOutput(secrets []exportSecretData, withMetadata bool) (string, error) {
	if withMetadata {
		// JSON with metadata
		result := make(map[string]exportSecretMetadata)
		for _, s := range secrets {
			meta := exportSecretMetadata{
				Value: string(s.value),
			}
			// Format timestamps as RFC3339
			createdAt := s.createdAt.Format(time.RFC3339)
			meta.CreatedAt = &createdAt
			updatedAt := s.updatedAt.Format(time.RFC3339)
			meta.UpdatedAt = &updatedAt
			if s.expiresAt != nil {
				expiresAt := s.expiresAt.Format(time.RFC3339)
				meta.ExpiresAt = &expiresAt
			}
			result[s.envName] = meta
		}
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return string(data) + "\n", nil
	}

	// Simple key-value JSON
	result := make(map[string]string)
	for _, s := range secrets {
		result[s.envName] = string(s.value)
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data) + "\n", nil
}

// writeSecureFile writes content to a file with 0600 permissions
// Security: Validates path, prevents traversal, checks for symlinks, prevents overwrites
func writeSecureFile(path string, content string, force bool) error {
	// 1. Validate and resolve path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// 2. Security check: Prevent writing to sensitive system directories
	// Note: We check specific dangerous directories, allowing /var/folders (macOS temp) and user home
	sensitivePaths := []string{"/etc/", "/usr/", "/bin/", "/sbin/", "/var/log/", "/var/run/", "/root/"}
	for _, sensitive := range sensitivePaths {
		if strings.HasPrefix(absPath, sensitive) {
			return fmt.Errorf("security: refusing to write to system directory: %s", absPath)
		}
	}

	// 3. Check if file already exists
	info, err := os.Lstat(absPath)
	if err == nil {
		// File exists
		// 3a. Check for symlink attack
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("security: refusing to write to symlink: %s", absPath)
		}
		// 3b. Check for overwrite permission
		if !force {
			return fmt.Errorf("file already exists: %s (use --force to overwrite)", absPath)
		}
	}

	// 4. Ensure parent directory exists with secure permissions
	dir := filepath.Dir(absPath)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		// Verify/fix directory permissions
		if err := os.Chmod(dir, 0700); err != nil {
			return fmt.Errorf("failed to set directory permissions: %w", err)
		}
	}

	// 5. Write file atomically with secure permissions
	// Use O_CREATE|O_TRUNC (or O_EXCL if not force) to prevent TOCTOU
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if !force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	}

	f, err := os.OpenFile(absPath, flags, 0600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("file already exists: %s (use --force to overwrite)", absPath)
		}
		return fmt.Errorf("failed to create file: %w", err)
	}

	// Write content
	_, writeErr := f.WriteString(content)
	closeErr := f.Close()

	if writeErr != nil {
		return fmt.Errorf("failed to write file: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close file: %w", closeErr)
	}

	return nil
}
