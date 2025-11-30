package main

import (
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

func init() {
	// Add subcommands to rootCmd
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(auditCmd)

	// Add metadata flags to set command
	setCmd.Flags().StringVar(&setNotes, "notes", "", "Add notes to the secret")
	setCmd.Flags().StringVar(&setURL, "url", "", "Add URL to the secret")
	setCmd.Flags().StringVar(&setTags, "tags", "", "Comma-separated tags (e.g., dev,api)")
	setCmd.Flags().StringVar(&setExpires, "expires", "", "Expiration duration (e.g., 30d, 1y)")

	// Add metadata flags to list command
	listCmd.Flags().StringVar(&listTag, "tag", "", "Filter by tag")
	listCmd.Flags().StringVar(&listExpiring, "expiring", "", "Show secrets expiring within duration (e.g., 7d)")

	// Add metadata flags to get command
	getCmd.Flags().BoolVar(&getShowMetadata, "show-metadata", false, "Show metadata with the secret")

	// Add audit subcommands
	auditCmd.AddCommand(auditListCmd)
	auditCmd.AddCommand(auditVerifyCmd)

	// Add flags to audit list
	auditListCmd.Flags().IntVar(&auditLimit, "limit", 100, "Maximum number of events to show")
	auditListCmd.Flags().StringVar(&auditSince, "since", "", "Show events since duration (e.g., 24h)")
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

		// Minimum password length check
		if len(password1) < 8 {
			return fmt.Errorf("password must be at least 8 characters")
		}

		// 4. Initialize vault
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
	Short: "Sets a secret value from standard input",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		// 1. Unlock vault
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		// 2. Read secret value from stdin
		// Use io.ReadAll to support multi-line and binary secrets
		// For interactive input, user can enter value and press Ctrl+D (Unix) or Ctrl+Z (Windows)
		fmt.Print("Enter secret value (Ctrl+D to finish): ")
		valueBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read secret value: %w", err)
		}

		// Trim trailing newline for interactive single-line input convenience
		// but preserve content for multi-line/binary secrets
		value := valueBytes
		if len(value) > 0 && value[len(value)-1] == '\n' {
			value = value[:len(value)-1]
		}
		if len(value) > 0 && value[len(value)-1] == '\r' {
			value = value[:len(value)-1]
		}

		// 3. Build SecretEntry
		entry := &vault.SecretEntry{
			Value: value,
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

		// 4. Save secret
		if err := v.SetSecret(key, entry); err != nil {
			return fmt.Errorf("failed to set secret: %w", err)
		}

		fmt.Printf("Secret '%s' saved successfully\n", key)
		return nil
	},
}

// getCmd retrieves a secret value
var getCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Gets a secret value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		// 1. Unlock vault
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		// 2. Get secret (always returns SecretEntry now)
		entry, err := v.GetSecret(key)
		if err != nil {
			return fmt.Errorf("failed to get secret: %w", err)
		}

		if getShowMetadata {
			// Print value
			fmt.Printf("Value: %s\n", string(entry.Value))

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
			// Write value to stdout (no newline, pipe-friendly)
			os.Stdout.Write(entry.Value)
			fmt.Println()
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
