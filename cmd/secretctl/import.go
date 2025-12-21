package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/forest6511/secretctl/pkg/vault"
)

// Import conflict handling modes
const (
	conflictSkip      = "skip"
	conflictOverwrite = "overwrite"
	conflictError     = "error"
)

var (
	importFormat   string
	importConflict string
	importDryRun   bool
	importKeys     []string
)

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importFormat, "format", "f", "", "Input format: env, json (auto-detected if not specified)")
	importCmd.Flags().StringVar(&importConflict, "conflict", conflictSkip, "Conflict handling: skip, overwrite, error")
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Show what would be imported without making changes")
	importCmd.Flags().StringSliceVarP(&importKeys, "key", "k", nil, "Keys to import (glob pattern supported)")

	// Convenience aliases
	importCmd.Flags().Bool("skip", false, "Skip existing keys (same as --conflict=skip)")
	importCmd.Flags().Bool("overwrite", false, "Overwrite existing keys (same as --conflict=overwrite)")
	importCmd.Flags().Bool("error", false, "Error on conflict (same as --conflict=error)")
}

var importCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import secrets from .env or JSON file",
	Long: `Import secrets from .env or JSON file into the vault.

Examples:
  # Import from .env file (auto-detected format)
  secretctl import .env

  # Import from JSON file
  secretctl import config.json --format=json

  # Import with overwrite mode
  secretctl import .env --overwrite

  # Preview import without making changes
  secretctl import .env --dry-run

  # Import specific keys only
  secretctl import .env -k "AWS_*" -k "DB_*"

Conflict handling:
  --skip       Skip keys that already exist (default)
  --overwrite  Overwrite existing keys with new values
  --error      Exit with error if any key already exists`,
	Args: cobra.ExactArgs(1),
	RunE: executeImport,
}

func executeImport(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Handle convenience flags
	if skip, _ := cmd.Flags().GetBool("skip"); skip {
		importConflict = conflictSkip
	}
	if overwrite, _ := cmd.Flags().GetBool("overwrite"); overwrite {
		importConflict = conflictOverwrite
	}
	if errFlag, _ := cmd.Flags().GetBool("error"); errFlag {
		importConflict = conflictError
	}

	// Validate conflict flag
	if err := validateImportFlags(); err != nil {
		return err
	}

	// Detect format if not specified
	format, err := detectImportFormat(filePath)
	if err != nil {
		return err
	}

	// Parse the input file
	secrets, err := parseImportFile(filePath, format)
	if err != nil {
		return err
	}

	if len(secrets) == 0 {
		fmt.Println("No secrets found in file")
		return nil
	}

	// Filter by key patterns if specified
	if len(importKeys) > 0 {
		secrets, err = filterImportKeys(secrets, importKeys)
		if err != nil {
			return err
		}
		if len(secrets) == 0 {
			return fmt.Errorf("no secrets match the specified key patterns")
		}
	}

	// Unlock vault (skip if dry-run and just show what would be imported)
	if !importDryRun {
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()
	}

	// Process import
	return processImport(secrets)
}

func validateImportFlags() error {
	importConflict = strings.ToLower(importConflict)
	if importConflict != conflictSkip && importConflict != conflictOverwrite && importConflict != conflictError {
		return fmt.Errorf("invalid conflict mode '%s': must be 'skip', 'overwrite', or 'error'", importConflict)
	}
	return nil
}

func detectImportFormat(filePath string) (string, error) {
	if importFormat != "" {
		format := strings.ToLower(importFormat)
		if format != formatEnv && format != formatJSON {
			return "", fmt.Errorf("invalid format '%s': must be '%s' or '%s'", importFormat, formatEnv, formatJSON)
		}
		return format, nil
	}

	// Auto-detect from file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		return formatJSON, nil
	case ".env":
		return formatEnv, nil
	default:
		// Check if filename contains "env" (e.g., ".env.local", "env.production")
		base := strings.ToLower(filepath.Base(filePath))
		if strings.Contains(base, "env") {
			return formatEnv, nil
		}
		// Default to env format for extensionless files
		return formatEnv, nil
	}
}

func parseImportFile(filePath string, format string) (map[string]string, error) {
	// Validate file path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Check file exists
	info, err := os.Lstat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		return nil, fmt.Errorf("failed to access file: %w", err)
	}

	// Security check: reject symlinks
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("security: refusing to read symlink: %s", absPath)
	}

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	switch format {
	case formatEnv:
		return parseEnvFile(data)
	case formatJSON:
		return parseJSONFile(data)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// envLineRegex matches KEY=VALUE patterns, supporting quoted values
var envLineRegex = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_/\-.]*)\s*=\s*(.*)$`)

func parseEnvFile(data []byte) (map[string]string, error) {
	secrets := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	// Increase buffer size to handle large secrets (1MB max line)
	const maxLineSize = 1024 * 1024
	buf := make([]byte, maxLineSize)
	scanner.Buffer(buf, maxLineSize)

	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Parse KEY=VALUE
		matches := envLineRegex.FindStringSubmatch(trimmed)
		if matches == nil {
			// Skip invalid lines (be lenient)
			continue
		}

		key := matches[1]
		value := matches[2]

		// Handle quoted values
		value = unquoteEnvValue(value)

		secrets[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return secrets, nil
}

func unquoteEnvValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return value
	}

	// Check for quoted strings
	if (value[0] == '"' && value[len(value)-1] == '"') ||
		(value[0] == '\'' && value[len(value)-1] == '\'') {
		quote := value[0]
		value = value[1 : len(value)-1]

		// Handle escape sequences for double-quoted strings
		// Process in single pass to avoid ordering issues
		if quote == '"' {
			var result strings.Builder
			result.Grow(len(value))
			for i := 0; i < len(value); i++ {
				if value[i] == '\\' && i+1 < len(value) {
					switch value[i+1] {
					case '"':
						result.WriteByte('"')
						i++
					case '\\':
						result.WriteByte('\\')
						i++
					case 'n':
						result.WriteByte('\n')
						i++
					case 'r':
						result.WriteByte('\r')
						i++
					case 't':
						result.WriteByte('\t')
						i++
					case '$':
						result.WriteByte('$')
						i++
					default:
						// Unknown escape - keep as-is
						result.WriteByte(value[i])
					}
				} else {
					result.WriteByte(value[i])
				}
			}
			value = result.String()
		}
	}

	return value
}

func parseJSONFile(data []byte) (map[string]string, error) {
	// Try parsing as flat key-value object first
	var flatMap map[string]interface{}
	if err := json.Unmarshal(data, &flatMap); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	secrets := make(map[string]string)
	for key, val := range flatMap {
		switch v := val.(type) {
		case string:
			secrets[key] = v
		case float64:
			// Handle numbers
			secrets[key] = fmt.Sprintf("%v", v)
		case bool:
			// Handle booleans
			secrets[key] = fmt.Sprintf("%v", v)
		case nil:
			// Skip null values
			continue
		default:
			// Skip complex types (arrays, nested objects)
			continue
		}
	}

	return secrets, nil
}

func filterImportKeys(secrets map[string]string, patterns []string) (map[string]string, error) {
	allKeys := make([]string, 0, len(secrets))
	for key := range secrets {
		allKeys = append(allKeys, key)
	}

	// Get matching keys
	matchedKeys := make(map[string]bool)
	for _, pattern := range patterns {
		matches, err := expandImportPattern(pattern, allKeys)
		if err != nil {
			return nil, err
		}
		for _, key := range matches {
			matchedKeys[key] = true
		}
	}

	// Filter secrets to only matched keys
	filtered := make(map[string]string)
	for key, value := range secrets {
		if matchedKeys[key] {
			filtered[key] = value
		}
	}

	return filtered, nil
}

func expandImportPattern(pattern string, availableKeys []string) ([]string, error) {
	// Validate pattern syntax
	if _, err := filepath.Match(pattern, ""); err != nil {
		return nil, fmt.Errorf("invalid pattern '%s': %w", pattern, err)
	}

	// Check if pattern contains glob characters
	hasGlob := strings.ContainsAny(pattern, "*?[")

	if !hasGlob {
		// Exact match - verify key exists
		for _, key := range availableKeys {
			if key == pattern {
				return []string{pattern}, nil
			}
		}
		return nil, fmt.Errorf("key '%s' not found in import file", pattern)
	}

	// Glob matching
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
		return nil, fmt.Errorf("no keys match pattern '%s' in import file", pattern)
	}

	return matches, nil
}

func processImport(secrets map[string]string) error {
	var imported, skipped, conflicts, failed int
	var errs []string

	// Get existing keys for conflict detection
	existingKeys, err := getExistingKeys()
	if err != nil {
		return err
	}

	// Sort keys for consistent output
	sortedKeys := sortKeys(secrets)

	for _, key := range sortedKeys {
		value := secrets[key]

		if importDryRun {
			fmt.Printf("[dry-run] Would import: %s\n", key)
			imported++
			continue
		}

		// Check for conflicts
		exists := existingKeys[key]
		action, errMsg := handleConflict(key, exists)
		switch action {
		case "skip":
			skipped++
			continue
		case "error":
			errs = append(errs, errMsg)
			conflicts++
			continue
		}

		// Save secret
		if err := saveSecret(key, value, exists); err != nil {
			errs = append(errs, fmt.Sprintf("failed to import '%s': %v", key, err))
			failed++
			continue
		}
		imported++
	}

	// Print summary
	printImportSummary(imported, skipped, conflicts, failed)

	// Return error if any issues
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

func getExistingKeys() (map[string]bool, error) {
	if importDryRun {
		return nil, nil
	}
	keys, err := v.ListSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to list existing secrets: %w", err)
	}
	existingKeys := make(map[string]bool)
	for _, key := range keys {
		existingKeys[key] = true
	}
	return existingKeys, nil
}

func sortKeys(secrets map[string]string) []string {
	sortedKeys := make([]string, 0, len(secrets))
	for key := range secrets {
		sortedKeys = append(sortedKeys, key)
	}
	for i := 0; i < len(sortedKeys); i++ {
		for j := i + 1; j < len(sortedKeys); j++ {
			if sortedKeys[i] > sortedKeys[j] {
				sortedKeys[i], sortedKeys[j] = sortedKeys[j], sortedKeys[i]
			}
		}
	}
	return sortedKeys
}

func handleConflict(key string, exists bool) (action string, errMsg string) {
	if !exists {
		return "import", ""
	}
	switch importConflict {
	case conflictSkip:
		fmt.Printf("Skipped (exists): %s\n", key)
		return "skip", ""
	case conflictError:
		return "error", fmt.Sprintf("key already exists: %s", key)
	default:
		return "import", ""
	}
}

func saveSecret(key, value string, exists bool) error {
	entry := &vault.SecretEntry{
		Value: []byte(value),
	}
	if err := v.SetSecret(key, entry); err != nil {
		return err
	}
	if exists {
		fmt.Printf("Overwritten: %s\n", key)
	} else {
		fmt.Printf("Imported: %s\n", key)
	}
	return nil
}

func printImportSummary(imported, skipped, conflicts, failed int) {
	fmt.Println()
	if importDryRun {
		fmt.Printf("Dry-run complete: %d secret(s) would be imported\n", imported)
		return
	}
	fmt.Printf("Import summary: %d imported", imported)
	if skipped > 0 {
		fmt.Printf(", %d skipped", skipped)
	}
	if conflicts > 0 {
		fmt.Printf(", %d conflicts", conflicts)
	}
	if failed > 0 {
		fmt.Printf(", %d failed", failed)
	}
	fmt.Println()
}
