package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/forest6511/secretctl/pkg/importer"
	"github.com/forest6511/secretctl/pkg/vault"
)

// Competitor import flags.
var (
	importFrom         string
	importPreserveCase bool
	importTag          string
)

// initCompetitorImportFlags adds competitor import flags to the import command.
// Called from init() in import.go
func initCompetitorImportFlags() {
	importCmd.Flags().StringVar(&importFrom, "from", "", "Import source: 1password, bitwarden, lastpass (required for competitor imports)")
	importCmd.Flags().BoolVar(&importPreserveCase, "preserve-case", false, "Preserve original case in key names (reduces collisions)")
	importCmd.Flags().StringVar(&importTag, "tag", "", "Add tag to all imported items")
}

// isCompetitorImport checks if this is a competitor import based on flags.
func isCompetitorImport() bool {
	return importFrom != ""
}

// executeCompetitorImport handles import from competitor password managers.
func executeCompetitorImport(filePath string) error {
	// Validate --from flag
	source := importer.Source(strings.ToLower(importFrom))
	_, err := importer.GetParser(source)
	if err != nil {
		return fmt.Errorf("invalid --from value '%s': must be one of %v", importFrom, importer.ValidSources())
	}

	// Read and validate file
	data, err := readCompetitorFile(filePath)
	if err != nil {
		return err
	}

	// Parse the file
	parser, _ := importer.GetParser(source)
	result, err := parser.Parse(data, importer.ParseOptions{
		PreserveCase: importPreserveCase,
	})
	if err != nil {
		return fmt.Errorf("failed to parse %s file: %w", importFrom, err)
	}

	// Print warnings
	for _, warning := range result.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
	}

	// Print skipped items
	for _, skipped := range result.Skipped {
		fmt.Fprintf(os.Stderr, "Skipped: %s (%s)\n", skipped.OriginalName, skipped.Reason)
	}

	if len(result.Secrets) == 0 {
		fmt.Println("No secrets found in file")
		return nil
	}

	fmt.Printf("Found %d secrets to import\n", len(result.Secrets))

	// Add import tag if specified
	if importTag != "" {
		for _, s := range result.Secrets {
			s.Tags = append(s.Tags, importTag)
		}
	}

	// Filter by key patterns if specified
	if len(importKeys) > 0 {
		result.Secrets, err = filterCompetitorKeys(result.Secrets, importKeys)
		if err != nil {
			return err
		}
		if len(result.Secrets) == 0 {
			return fmt.Errorf("no secrets match the specified key patterns")
		}
	}

	// Unlock vault (skip if dry-run)
	if !importDryRun {
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()
	}

	// Process import
	return processCompetitorImport(result.Secrets)
}

// readCompetitorFile reads and validates a competitor export file.
func readCompetitorFile(filePath string) ([]byte, error) {
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

	return data, nil
}

// filterCompetitorKeys filters secrets by key patterns.
func filterCompetitorKeys(secrets []*importer.ImportedSecret, patterns []string) ([]*importer.ImportedSecret, error) {
	// Get all keys
	allKeys := make([]string, len(secrets))
	keyToSecret := make(map[string]*importer.ImportedSecret)
	for i, s := range secrets {
		allKeys[i] = s.Key
		keyToSecret[s.Key] = s
	}

	// Match patterns
	matchedKeys := make(map[string]bool)
	for _, pattern := range patterns {
		// Validate pattern syntax
		if _, err := filepath.Match(pattern, ""); err != nil {
			return nil, fmt.Errorf("invalid pattern '%s': %w", pattern, err)
		}

		hasGlob := strings.ContainsAny(pattern, "*?[")

		if !hasGlob {
			// Exact match
			if _, exists := keyToSecret[pattern]; exists {
				matchedKeys[pattern] = true
			} else {
				return nil, fmt.Errorf("key '%s' not found in import file", pattern)
			}
		} else {
			// Glob matching
			for _, key := range allKeys {
				matched, err := filepath.Match(pattern, key)
				if err != nil {
					return nil, err
				}
				if matched {
					matchedKeys[key] = true
				}
			}
		}
	}

	if len(matchedKeys) == 0 {
		return nil, fmt.Errorf("no keys match the specified patterns")
	}

	// Filter secrets
	var filtered []*importer.ImportedSecret
	for _, s := range secrets {
		if matchedKeys[s.Key] {
			filtered = append(filtered, s)
		}
	}

	return filtered, nil
}

// processCompetitorImport processes the import of multi-field secrets.
func processCompetitorImport(secrets []*importer.ImportedSecret) error {
	var imported, skipped, conflicts, failed int
	var errs []string

	// Get existing keys for conflict detection
	existingKeys, err := getExistingKeysMap()
	if err != nil {
		return err
	}

	// Sort by key for consistent output
	sort.Slice(secrets, func(i, j int) bool {
		return secrets[i].Key < secrets[j].Key
	})

	for _, secret := range secrets {
		if importDryRun {
			fmt.Printf("[dry-run] Would import: %s (%d fields)\n", secret.Key, len(secret.Fields))
			imported++
			continue
		}

		// Check for conflicts
		exists := existingKeys[secret.Key]
		action, errMsg := handleCompetitorConflict(secret.Key, exists)
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
		if err := saveCompetitorSecret(secret, exists); err != nil {
			errs = append(errs, fmt.Sprintf("failed to import '%s': %v", secret.Key, err))
			failed++
			continue
		}
		imported++
	}

	// Print summary
	printCompetitorImportSummary(imported, skipped, conflicts, failed)

	// Return error if any issues
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// getExistingKeysMap returns a map of existing secret keys.
func getExistingKeysMap() (map[string]bool, error) {
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

// handleCompetitorConflict handles conflict detection for competitor imports.
func handleCompetitorConflict(key string, exists bool) (action string, errMsg string) {
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

// saveCompetitorSecret saves a multi-field secret to the vault.
func saveCompetitorSecret(secret *importer.ImportedSecret, exists bool) error {
	entry := &vault.SecretEntry{
		Key:      secret.Key,
		Fields:   secret.Fields,
		Metadata: secret.Metadata,
		Tags:     secret.Tags,
	}
	if err := v.SetSecret(secret.Key, entry); err != nil {
		return err
	}
	fieldCount := len(secret.Fields)
	if exists {
		fmt.Printf("Overwritten: %s (%d fields)\n", secret.Key, fieldCount)
	} else {
		fmt.Printf("Imported: %s (%d fields)\n", secret.Key, fieldCount)
	}
	return nil
}

// printCompetitorImportSummary prints the import summary.
func printCompetitorImportSummary(imported, skipped, conflicts, failed int) {
	fmt.Printf("\nImport summary:\n")
	fmt.Printf("  Imported:  %d\n", imported)
	if skipped > 0 {
		fmt.Printf("  Skipped:   %d\n", skipped)
	}
	if conflicts > 0 {
		fmt.Printf("  Conflicts: %d\n", conflicts)
	}
	if failed > 0 {
		fmt.Printf("  Failed:    %d\n", failed)
	}
}
