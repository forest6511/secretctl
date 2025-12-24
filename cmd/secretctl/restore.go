package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/forest6511/secretctl/pkg/backup"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	restoreDryRun     bool
	restoreVerifyOnly bool
	restoreOnConflict string
	restoreKeyFile    string
	restoreForce      bool
	restoreWithAudit  bool
)

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().BoolVar(&restoreDryRun, "dry-run", false, "Show what would be restored without making changes")
	restoreCmd.Flags().BoolVar(&restoreVerifyOnly, "verify-only", false, "Only verify backup integrity")
	restoreCmd.Flags().StringVar(&restoreOnConflict, "on-conflict", "error", "Conflict resolution: skip, overwrite, error")
	restoreCmd.Flags().StringVar(&restoreKeyFile, "key-file", "", "Decryption key file")
	restoreCmd.Flags().BoolVarP(&restoreForce, "force", "f", false, "Skip confirmation prompt")
	restoreCmd.Flags().BoolVar(&restoreWithAudit, "with-audit", false, "Restore audit log (overwrites existing)")
}

var restoreCmd = &cobra.Command{
	Use:   "restore <backup-file>",
	Short: "Restore vault from encrypted backup",
	Long: `Restore the vault from an encrypted backup file.

Examples:
  # Dry run (preview only)
  secretctl restore backup.enc --dry-run

  # Verify backup integrity without restoring
  secretctl restore backup.enc --verify-only

  # Restore, skip conflicts
  secretctl restore backup.enc --on-conflict=skip

  # Restore, overwrite conflicts
  secretctl restore backup.enc --on-conflict=overwrite

  # Restore, error on conflicts (default)
  secretctl restore backup.enc --on-conflict=error

  # Restore with audit log
  secretctl restore backup.enc --with-audit

  # Use key file for decryption
  secretctl restore backup.enc --key-file=backup.key`,
	Args: cobra.ExactArgs(1),
	RunE: executeRestore,
}

func executeRestore(cmd *cobra.Command, args []string) error {
	backupPath := args[0]

	// Validate flags
	if err := validateRestoreFlags(); err != nil {
		return err
	}

	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	// Parse conflict mode
	conflictMode, err := parseConflictMode(restoreOnConflict)
	if err != nil {
		return err
	}

	// Get password or key file
	var password []byte
	if restoreKeyFile == "" {
		fmt.Print("Enter backup password (or master password): ")
		pwd, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()
		password = pwd
	}

	// Verify only mode
	if restoreVerifyOnly {
		result, err := backup.Verify(backupPath, password, restoreKeyFile)
		if err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}
		if !result.Valid {
			return fmt.Errorf("verification failed: %s", result.Error)
		}
		fmt.Printf("Backup verification successful!\n")
		fmt.Printf("  Version: %d\n", result.Version)
		fmt.Printf("  Created: %s\n", result.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Secrets: %d\n", result.SecretCount)
		fmt.Printf("  Includes Audit: %v\n", result.IncludesAudit)
		return nil
	}

	// Confirmation prompt
	if !restoreForce && !restoreDryRun {
		fmt.Print("This will restore the vault from backup. Continue? [y/N]: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Restore cancelled.")
			return nil
		}
	}

	// Create restore options
	opts := backup.RestoreOptions{
		VaultPath:  vaultPath,
		OnConflict: conflictMode,
		DryRun:     restoreDryRun,
		VerifyOnly: false,
		WithAudit:  restoreWithAudit,
		Password:   password,
		KeyFile:    restoreKeyFile,
	}

	// Perform restore
	result, err := backup.Restore(backupPath, opts)
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	// Print results
	if restoreDryRun {
		fmt.Printf("Dry run complete. Would restore:\n")
	} else {
		fmt.Printf("Restore complete!\n")
	}
	fmt.Printf("  Secrets restored: %d\n", result.SecretsRestored)
	fmt.Printf("  Secrets skipped: %d\n", result.SecretsSkipped)
	if result.AuditRestored {
		fmt.Printf("  Audit log: restored\n")
	}

	return nil
}

func validateRestoreFlags() error {
	validModes := map[string]bool{"skip": true, "overwrite": true, "error": true}
	if !validModes[restoreOnConflict] {
		return fmt.Errorf("invalid --on-conflict value: %s (valid: skip, overwrite, error)", restoreOnConflict)
	}
	if restoreDryRun && restoreVerifyOnly {
		return fmt.Errorf("--dry-run and --verify-only are mutually exclusive")
	}
	return nil
}

func parseConflictMode(mode string) (backup.ConflictMode, error) {
	switch mode {
	case "skip":
		return backup.ConflictSkip, nil
	case "overwrite":
		return backup.ConflictOverwrite, nil
	case "error":
		return backup.ConflictError, nil
	default:
		return backup.ConflictError, fmt.Errorf("unknown conflict mode: %s", mode)
	}
}
