package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/forest6511/secretctl/pkg/vault"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	vaultPath string
	v         *vault.Vault
)

var rootCmd = &cobra.Command{
	Use:   "secretctl",
	Short: "secretctl is a simple, AI-ready secrets manager",
	Long:  `A fast and modern secrets manager built with Go.`,
	// PersistentPreRunE runs before the root command and all subcommands.
	// This initializes the Vault object.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip for init command since the vault doesn't exist yet
		if cmd.Use == "init" {
			return nil
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		vaultPath = filepath.Join(home, ".secretctl")
		v = vault.New(vaultPath)
		return nil
	},
}

// Metadata flags for set command
var (
	setNotes   string
	setURL     string
	setTags    string
	setExpires string

	// Multi-field support (Phase 2.5b)
	setFields   []string // --field name=value (can be repeated)
	setBindings []string // --binding ENV=field (can be repeated)
	setTemplate string   // --template name

	// Get field support
	getField      string // --field name (get specific field)
	getShowFields bool   // --fields (list all fields)
)

// Metadata flags for list command
var (
	listTag      string
	listExpiring string
)

// Metadata flags for get command
var (
	getShowMetadata bool
)

// Audit flags
var (
	auditLimit int
	auditSince string
)

// Audit export flags
var (
	auditExportFormat string
	auditExportSince  string
	auditExportUntil  string
	auditExportOutput string
)

// Audit prune flags
var (
	auditPruneOlderThan string
	auditPruneDryRun    bool
	auditPruneForce     bool
)

func init() {
	// Add subcommands to rootCmd
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(passwordCmd)

	// Add metadata flags to set command
	setCmd.Flags().StringVar(&setNotes, "notes", "", "Add notes to the secret")
	setCmd.Flags().StringVar(&setURL, "url", "", "Add URL to the secret")
	setCmd.Flags().StringVar(&setTags, "tags", "", "Comma-separated tags (e.g., dev,api)")
	setCmd.Flags().StringVar(&setExpires, "expires", "", "Expiration duration (e.g., 30d, 1y)")

	// Multi-field flags for set command (Phase 2.5b)
	setCmd.Flags().StringArrayVar(&setFields, "field", nil, "Set field value (name=value, can be repeated)")
	setCmd.Flags().StringArrayVar(&setBindings, "binding", nil, "Set env binding (ENV_VAR=field, can be repeated)")
	setCmd.Flags().StringVar(&setTemplate, "template", "", "Use template (login, database, api, ssh)")

	// Add metadata flags to list command
	listCmd.Flags().StringVar(&listTag, "tag", "", "Filter by tag")
	listCmd.Flags().StringVar(&listExpiring, "expiring", "", "Show secrets expiring within duration (e.g., 7d)")

	// Add metadata flags to get command
	getCmd.Flags().BoolVar(&getShowMetadata, "show-metadata", false, "Show metadata with the secret")
	getCmd.Flags().StringVar(&getField, "field", "", "Get specific field value")
	getCmd.Flags().BoolVar(&getShowFields, "fields", false, "List all field names")

	// Add audit subcommands
	auditCmd.AddCommand(auditListCmd)
	auditCmd.AddCommand(auditVerifyCmd)
	auditCmd.AddCommand(auditExportCmd)
	auditCmd.AddCommand(auditPruneCmd)

	// Add password subcommands
	passwordCmd.AddCommand(passwordChangeCmd)

	// Add flags to audit list
	auditListCmd.Flags().IntVar(&auditLimit, "limit", 100, "Maximum number of events to show")
	auditListCmd.Flags().StringVar(&auditSince, "since", "", "Show events since duration (e.g., 24h)")

	// Add flags to audit export
	auditExportCmd.Flags().StringVar(&auditExportFormat, "format", "json", "Output format: json, csv")
	auditExportCmd.Flags().StringVar(&auditExportSince, "since", "", "Export events since duration (e.g., 30d)")
	auditExportCmd.Flags().StringVar(&auditExportUntil, "until", "", "Export events until date (RFC 3339)")
	auditExportCmd.Flags().StringVarP(&auditExportOutput, "output", "o", "", "Output file path (default: stdout)")

	// Add flags to audit prune
	auditPruneCmd.Flags().StringVar(&auditPruneOlderThan, "older-than", "", "Delete logs older than duration (e.g., 12m for 12 months)")
	auditPruneCmd.Flags().BoolVar(&auditPruneDryRun, "dry-run", false, "Show what would be deleted without deleting")
	auditPruneCmd.Flags().BoolVarP(&auditPruneForce, "force", "f", false, "Skip confirmation prompt")
}

// initCmd initializes a new vault
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes a new secret vault",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Set vault path
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		vaultPath = filepath.Join(home, ".secretctl")

		fmt.Println("Initializing new vault...")

		// 1. Prompt for master password
		fmt.Print("Enter master password: ")
		password1, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		// 2. Confirm password
		fmt.Print("Confirm master password: ")
		password2, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		// 3. Check passwords match
		if string(password1) != string(password2) {
			return fmt.Errorf("passwords do not match")
		}

		// 4. Validate password strength per requirements-ja.md §2.3
		passwordResult := vault.ValidateMasterPassword(string(password1))
		if !passwordResult.Valid {
			// Hard errors (length requirements)
			return fmt.Errorf("password validation failed: %s", passwordResult.Warnings[0])
		}

		// Display strength and warnings (warnings are advisory, not blocking)
		fmt.Printf("Password strength: %s\n", passwordResult.Strength)
		for _, warning := range passwordResult.Warnings {
			fmt.Printf("Warning: %s\n", warning)
		}

		// 5. Initialize vault
		v = vault.New(vaultPath)
		if err := v.Init(string(password1)); err != nil {
			return fmt.Errorf("failed to initialize vault: %w", err)
		}

		fmt.Printf("Vault initialized successfully at %s\n", vaultPath)
		return nil
	},
}

// setCmd sets a secret value
var setCmd = &cobra.Command{
	Use:   "set [key]",
	Short: "Sets a secret value from standard input or fields",
	Long: `Sets a secret value. Supports two modes:

1. Single value mode (default):
   secretctl set mykey
   # Enter value interactively

2. Multi-field mode (--field or --template):
   secretctl set mykey --field username=admin --field password=secret
   secretctl set mykey --template database
   
Available templates: login, database, api, ssh`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		// 1. Unlock vault
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		// 2. Build SecretEntry
		entry := &vault.SecretEntry{}

		// Check if using multi-field mode
		if setTemplate != "" || len(setFields) > 0 {
			// Multi-field mode
			fields, bindings, err := buildFieldsFromFlags()
			if err != nil {
				return err
			}
			entry.Fields = fields
			entry.Bindings = bindings
		} else {
			// Legacy single-value mode
			fmt.Print("Enter secret value (Ctrl+D to finish): ")
			valueBytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read secret value: %w", err)
			}

			// Trim trailing newline for interactive single-line input convenience
			value := valueBytes
			if len(value) > 0 && value[len(value)-1] == '\n' {
				value = value[:len(value)-1]
			}
			if len(value) > 0 && value[len(value)-1] == '\r' {
				value = value[:len(value)-1]
			}
			entry.Value = value
		}

		// Add metadata if any flags are set
		if setNotes != "" || setURL != "" {
			entry.Metadata = &vault.SecretMetadata{
				Notes: setNotes,
				URL:   setURL,
			}
		}

		// Add tags (plaintext)
		if setTags != "" {
			entry.Tags = strings.Split(setTags, ",")
		}

		// Add expiration (plaintext)
		if setExpires != "" {
			duration, err := parseDuration(setExpires)
			if err != nil {
				return fmt.Errorf("invalid expiration format: %w", err)
			}
			expiresAt := time.Now().Add(duration)
			entry.ExpiresAt = &expiresAt
		}

		// 3. Save secret
		if err := v.SetSecret(key, entry); err != nil {
			return fmt.Errorf("failed to set secret: %w", err)
		}

		if entry.Fields != nil {
			fmt.Printf("Secret '%s' saved with %d fields\n", key, len(entry.Fields))
		} else {
			fmt.Printf("Secret '%s' saved successfully\n", key)
		}
		return nil
	},
}

// buildFieldsFromFlags builds Fields and Bindings from CLI flags.
func buildFieldsFromFlags() (fields map[string]vault.Field, bindings map[string]string, err error) {
	fields = make(map[string]vault.Field)
	bindings = make(map[string]string)

	// If template is specified, prompt for template fields
	if setTemplate != "" {
		if err := promptTemplateFields(fields); err != nil {
			return nil, nil, err
		}
	}

	// Add/override fields from --field flags
	if err := parseFieldFlags(fields); err != nil {
		return nil, nil, err
	}

	// Add bindings from --binding flags
	if err := parseBindingFlags(fields, bindings); err != nil {
		return nil, nil, err
	}

	if len(fields) == 0 {
		return nil, nil, fmt.Errorf("no fields specified (use --field or --template)")
	}

	return fields, bindings, nil
}

// promptTemplateFields prompts user for template fields interactively
func promptTemplateFields(fields map[string]vault.Field) error {
	template, ok := BuiltinTemplates[setTemplate]
	if !ok {
		return fmt.Errorf("unknown template: %s (available: %v)", setTemplate, ListTemplates())
	}

	fmt.Printf("Using template: %s (%s)\n", template.Name, template.Description)

	for _, tf := range template.Fields {
		value, err := readTemplateField(tf)
		if err != nil {
			return err
		}

		if value == "" && tf.Required {
			return fmt.Errorf("field %q is required", tf.Name)
		}

		if value != "" {
			fields[tf.Name] = vault.Field{
				Value:     value,
				Sensitive: tf.Sensitive,
				Kind:      tf.Kind,
				InputType: tf.InputType,
			}
		}
	}
	return nil
}

// readTemplateField reads a single template field from user input
func readTemplateField(tf TemplateField) (string, error) {
	prompt := tf.Prompt
	if !tf.Required {
		prompt += " (optional)"
	}

	switch {
	case tf.InputType == "textarea":
		// Multi-line input for textarea fields (read until EOF) per ADR-005
		// Note: InputType check comes BEFORE Sensitive check
		fmt.Printf("%s (paste, then Ctrl+D):\n", prompt)
		valueBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
		value := strings.TrimSuffix(string(valueBytes), "\n")
		return strings.TrimSuffix(value, "\r"), nil

	case tf.Sensitive:
		// Secure password input (no echo) for sensitive fields
		fmt.Printf("%s: ", prompt)
		if isTerminal(int(os.Stdin.Fd())) {
			passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println() // Add newline after hidden input
			if err != nil {
				return "", fmt.Errorf("failed to read password: %w", err)
			}
			return string(passwordBytes), nil
		}
		// Fallback for piped input
		return readLine()

	default:
		// Single-line input for non-sensitive fields
		fmt.Printf("%s: ", prompt)
		return readLine()
	}
}

// readLine reads a single line from stdin, trimming trailing newline
func readLine() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	value := strings.TrimSuffix(line, "\n")
	return strings.TrimSuffix(value, "\r"), nil
}

// parseFieldFlags parses --field flags into fields map
func parseFieldFlags(fields map[string]vault.Field) error {
	for _, f := range setFields {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid field format %q (expected name=value)", f)
		}
		name, value := parts[0], parts[1]
		fields[name] = vault.Field{
			Value:     value,
			Sensitive: true, // Default to sensitive
		}
	}
	return nil
}

// parseBindingFlags parses --binding flags into bindings map
func parseBindingFlags(fields map[string]vault.Field, bindings map[string]string) error {
	for _, b := range setBindings {
		parts := strings.SplitN(b, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid binding format %q (expected ENV_VAR=field)", b)
		}
		envVar, fieldName := parts[0], parts[1]

		// Validate that the referenced field exists
		if _, exists := fields[fieldName]; !exists {
			return fmt.Errorf("binding %q references non-existent field %q", envVar, fieldName)
		}
		bindings[envVar] = fieldName
	}
	return nil
}

// isTerminal returns true if the file descriptor is a terminal
func isTerminal(fd int) bool {
	return term.IsTerminal(fd)
}

// getCmd retrieves a secret value
var getCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Gets a secret value or field",
	Long: `Gets a secret value. Supports multiple modes:

1. Single value mode (default):
   secretctl get mykey
   # Outputs the value (or Fields["value"] for multi-field secrets)

2. Specific field mode:
   secretctl get mykey --field password
   # Outputs the password field value

3. List fields mode:
   secretctl get mykey --fields
   # Lists all field names`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		// 1. Unlock vault
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		// 2. Get secret
		entry, err := v.GetSecret(key)
		if err != nil {
			return fmt.Errorf("failed to get secret: %w", err)
		}

		// 3. Handle different output modes
		if getShowFields {
			// List all field names
			if len(entry.Fields) == 0 {
				fmt.Println("(no fields - legacy single-value secret)")
				return nil
			}
			for name, field := range entry.Fields {
				sensitive := ""
				if field.Sensitive {
					sensitive = " [sensitive]"
				}
				fmt.Printf("%s%s\n", name, sensitive)
			}
			return nil
		}

		if getField != "" {
			// Get specific field
			if len(entry.Fields) == 0 {
				return fmt.Errorf("secret has no fields (legacy single-value secret)")
			}
			fieldName, field, err := vault.ResolveFieldName(entry.Fields, getField)
			if err != nil {
				return fmt.Errorf("field %q not found", getField)
			}
			_ = fieldName // resolved name (for alias case)
			os.Stdout.WriteString(field.Value)
			fmt.Println()
			return nil
		}

		if getShowMetadata {
			// Show full metadata
			if len(entry.Fields) > 0 && !vault.IsSingleFieldSecret(entry.Fields) {
				// Multi-field secret
				fmt.Println("Fields:")
				for name, field := range entry.Fields {
					if field.Sensitive {
						fmt.Printf("  %s: [sensitive]\n", name)
					} else {
						fmt.Printf("  %s: %s\n", name, field.Value)
					}
				}
				if len(entry.Bindings) > 0 {
					fmt.Println("Bindings:")
					for env, field := range entry.Bindings {
						fmt.Printf("  %s -> %s\n", env, field)
					}
				}
			} else {
				// Single value
				fmt.Printf("Value: %s\n", string(entry.Value))
			}

			// Print encrypted metadata if present
			if entry.Metadata != nil {
				if entry.Metadata.Notes != "" {
					fmt.Printf("Notes: %s\n", entry.Metadata.Notes)
				}
				if entry.Metadata.URL != "" {
					fmt.Printf("URL: %s\n", entry.Metadata.URL)
				}
			}
			// Print plaintext metadata
			if len(entry.Tags) > 0 {
				fmt.Printf("Tags: %s\n", strings.Join(entry.Tags, ", "))
			}
			if entry.ExpiresAt != nil {
				fmt.Printf("Expires: %s\n", entry.ExpiresAt.Format(time.RFC3339))
			}
			fmt.Printf("Created: %s\n", entry.CreatedAt.Format(time.RFC3339))
			fmt.Printf("Updated: %s\n", entry.UpdatedAt.Format(time.RFC3339))
		} else {
			// Default: output value (or default field for multi-field)
			if len(entry.Fields) > 0 && vault.IsSingleFieldSecret(entry.Fields) {
				// Single-field secret, output the value
				os.Stdout.WriteString(vault.GetDefaultFieldValue(entry.Fields))
				fmt.Println()
			} else if len(entry.Fields) > 0 {
				// Multi-field secret without --field flag - return error for script safety
				return fmt.Errorf("multi-field secret with %d fields; use --field <name> to get specific field or --fields to list", len(entry.Fields))
			} else {
				// Legacy format
				os.Stdout.Write(entry.Value)
				fmt.Println()
			}
		}
		return nil
	},
}

// listCmd lists all secret keys
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists all secret keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Unlock vault
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		// 2. Get key list based on filters
		var entries []*vault.SecretEntry
		var err error

		if listTag != "" {
			// Filter by tag
			entries, err = v.ListSecretsByTag(listTag)
			if err != nil {
				return fmt.Errorf("failed to list secrets: %w", err)
			}
		} else if listExpiring != "" {
			// Filter by expiration
			duration, parseErr := parseDuration(listExpiring)
			if parseErr != nil {
				return fmt.Errorf("invalid expiring format: %w", parseErr)
			}
			entries, err = v.ListExpiringSecrets(duration)
			if err != nil {
				return fmt.Errorf("failed to list secrets: %w", err)
			}
		} else {
			// No filter - use simple list
			keys, listErr := v.ListSecrets()
			if listErr != nil {
				return fmt.Errorf("failed to list secrets: %w", listErr)
			}

			// 3. Display key list
			if len(keys) == 0 {
				fmt.Println("No secrets stored")
				return nil
			}

			for _, key := range keys {
				fmt.Println(key)
			}
			return nil
		}

		// 3. Display filtered secrets with metadata
		if len(entries) == 0 {
			fmt.Println("No secrets found")
			return nil
		}

		for _, entry := range entries {
			line := entry.Key
			if len(entry.Tags) > 0 {
				line += fmt.Sprintf(" [%s]", strings.Join(entry.Tags, ","))
			}
			if entry.ExpiresAt != nil {
				line += fmt.Sprintf(" (expires: %s)", entry.ExpiresAt.Format("2006-01-02"))
			}
			fmt.Println(line)
		}
		return nil
	},
}

// deleteCmd deletes a secret
var deleteCmd = &cobra.Command{
	Use:   "delete [key]",
	Short: "Deletes a secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		// 1. Unlock vault
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		// 2. Delete secret
		if err := v.DeleteSecret(key); err != nil {
			return fmt.Errorf("failed to delete secret: %w", err)
		}

		fmt.Printf("Secret '%s' deleted successfully\n", key)
		return nil
	},
}

// ensureUnlocked ensures the vault is unlocked.
// If locked, prompts for password and attempts to unlock.
func ensureUnlocked() error {
	if v.IsLocked() {
		fmt.Print("Enter master password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		if err := v.Unlock(string(passwordBytes)); err != nil {
			return fmt.Errorf("failed to unlock vault: %w", err)
		}
	}
	return nil
}

// auditCmd is the parent command for audit operations
var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit log operations",
}

// auditListCmd lists audit log entries
var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audit log entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Unlock vault to access audit logs
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		// 2. Parse since duration
		var since time.Time
		if auditSince != "" {
			duration, err := parseDuration(auditSince)
			if err != nil {
				return fmt.Errorf("invalid since format: %w", err)
			}
			since = time.Now().Add(-duration)
		}

		// 3. Get audit events
		events, err := v.AuditLogger().ListEvents(auditLimit, since)
		if err != nil {
			return fmt.Errorf("failed to list audit events: %w", err)
		}

		if len(events) == 0 {
			fmt.Println("No audit events found")
			return nil
		}

		// 4. Display events
		for _, event := range events {
			// Format: TIMESTAMP OPERATION RESULT [KEY]
			line := fmt.Sprintf("%s %s %s", event.Timestamp, event.Operation, event.Result)
			if event.Key != "" {
				// Show truncated key hash
				keyDisplay := event.Key
				if len(keyDisplay) > 16 {
					keyDisplay = keyDisplay[:16] + "..."
				}
				line += fmt.Sprintf(" key:%s", keyDisplay)
			}
			if event.Error != nil {
				line += fmt.Sprintf(" error:%s", event.Error.Code)
			}
			fmt.Println(line)
		}

		fmt.Printf("\nTotal: %d events\n", len(events))
		return nil
	},
}

// auditVerifyCmd verifies audit log integrity
var auditVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify audit log HMAC chain integrity",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Unlock vault to access HMAC key
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		fmt.Println("Verifying audit log integrity...")

		// 2. Run verification
		result, err := v.AuditVerify()
		if err != nil {
			return fmt.Errorf("failed to verify audit log: %w", err)
		}

		// 3. Display result
		if result.Valid {
			fmt.Printf("✓ Audit log verified: %d records, chain intact\n", result.RecordsTotal)
		} else {
			fmt.Printf("✗ Audit log verification FAILED\n")
			fmt.Printf("  Records total: %d\n", result.RecordsTotal)
			fmt.Printf("  Records verified: %d\n", result.RecordsVerified)
			fmt.Println("  Errors:")
			for _, e := range result.Errors {
				fmt.Printf("    - %s\n", e)
			}
			return fmt.Errorf("audit log integrity check failed")
		}

		// Also output as JSON for machine parsing
		jsonResult, _ := json.Marshal(result)
		fmt.Printf("\nJSON: %s\n", string(jsonResult))

		return nil
	},
}

// auditExportCmd exports audit logs
var auditExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export audit logs to JSON or CSV format",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Unlock vault to access audit logs
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		// 2. Validate format
		if auditExportFormat != "json" && auditExportFormat != "csv" {
			return fmt.Errorf("invalid format: %s (use 'json' or 'csv')", auditExportFormat)
		}

		// 3. Parse time filters
		var since, until time.Time
		if auditExportSince != "" {
			duration, err := parseDuration(auditExportSince)
			if err != nil {
				return fmt.Errorf("invalid since format: %w", err)
			}
			since = time.Now().Add(-duration)
		}
		if auditExportUntil != "" {
			var err error
			until, err = time.Parse(time.RFC3339, auditExportUntil)
			if err != nil {
				return fmt.Errorf("invalid until format (use RFC 3339): %w", err)
			}
		}

		// 4. Export events
		data, err := v.AuditLogger().Export(auditExportFormat, since, until)
		if err != nil {
			return fmt.Errorf("failed to export audit logs: %w", err)
		}

		// 5. Output to file or stdout
		if auditExportOutput != "" {
			// Validate output path to prevent path traversal
			absPath, err := filepath.Abs(auditExportOutput)
			if err != nil {
				return fmt.Errorf("invalid output path: %w", err)
			}

			// Get current working directory for validation
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			// Allow paths within: current directory, home directory, or /tmp
			homeDir, _ := os.UserHomeDir()
			validPrefixes := []string{cwd, homeDir, "/tmp"}
			isValid := false
			for _, prefix := range validPrefixes {
				if strings.HasPrefix(absPath, prefix) {
					isValid = true
					break
				}
			}
			if !isValid {
				return fmt.Errorf("output path must be within current directory, home directory, or /tmp")
			}

			// Write to file with secure permissions
			if err := os.WriteFile(absPath, data, 0600); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Warning: Exported audit logs contain key hashes and operation metadata.\n")
			fmt.Fprintf(os.Stderr, "Audit logs exported to %s\n", absPath)
		} else {
			// Write to stdout
			os.Stdout.Write(data)
		}

		return nil
	},
}

// auditPruneCmd deletes old audit logs
var auditPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Delete old audit log entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Validate older-than flag
		if auditPruneOlderThan == "" {
			return fmt.Errorf("--older-than flag is required")
		}

		duration, err := parseDuration(auditPruneOlderThan)
		if err != nil {
			return fmt.Errorf("invalid older-than format: %w", err)
		}

		// 2. Unlock vault to access audit logs
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		// 3. Dry-run mode
		if auditPruneDryRun {
			count, err := v.AuditLogger().PrunePreview(duration)
			if err != nil {
				return fmt.Errorf("failed to preview prune: %w", err)
			}
			fmt.Printf("Would delete %d audit log entries older than %s\n", count, auditPruneOlderThan)
			return nil
		}

		// 4. Get preview count for confirmation
		count, err := v.AuditLogger().PrunePreview(duration)
		if err != nil {
			return fmt.Errorf("failed to preview prune: %w", err)
		}

		if count == 0 {
			fmt.Println("No audit log entries to delete")
			return nil
		}

		// 5. Confirmation prompt (unless --force)
		if !auditPruneForce {
			fmt.Printf("This will delete %d audit log entries older than %s.\n", count, auditPruneOlderThan)
			fmt.Print("Are you sure? [y/N]: ")
			var response string
			if _, err := fmt.Scanln(&response); err != nil {
				// Treat read error as "no"
				fmt.Println("Aborted")
				return nil
			}
			if response != "y" && response != "Y" {
				fmt.Println("Aborted")
				return nil
			}
		}

		// 6. Perform prune
		deleted, err := v.AuditLogger().Prune(duration)
		if err != nil {
			return fmt.Errorf("failed to prune audit logs: %w", err)
		}

		fmt.Printf("Deleted %d audit log entries\n", deleted)
		return nil
	},
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
		// Try standard time.ParseDuration
		return time.ParseDuration(s)
	}
}
