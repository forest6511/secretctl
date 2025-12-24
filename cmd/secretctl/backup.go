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
	backupOutput         string
	backupStdout         bool
	backupWithAudit      bool
	backupBackupPassword bool
	backupKeyFile        string
	backupForce          bool
)

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVarP(&backupOutput, "output", "o", "", "Output file path")
	backupCmd.Flags().BoolVar(&backupStdout, "stdout", false, "Output to stdout (for piping)")
	backupCmd.Flags().BoolVar(&backupWithAudit, "with-audit", false, "Include audit log in backup")
	backupCmd.Flags().BoolVar(&backupBackupPassword, "backup-password", false, "Use separate backup password")
	backupCmd.Flags().StringVar(&backupKeyFile, "key-file", "", "Encryption key file (32 bytes)")
	backupCmd.Flags().BoolVarP(&backupForce, "force", "f", false, "Overwrite existing file")
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create encrypted backup of the vault",
	Long: `Create an encrypted backup of the vault data.

Examples:
  # Backup to a file
  secretctl backup -o vault-backup.enc

  # Backup with audit log
  secretctl backup -o full-backup.enc --with-audit

  # Backup to stdout (for piping)
  secretctl backup --stdout | gpg --encrypt > backup.gpg

  # Use separate backup password
  secretctl backup -o backup.enc --backup-password

  # Use key file for encryption
  secretctl backup -o backup.enc --key-file=backup.key

  # Overwrite existing file
  secretctl backup -o backup.enc --force`,
	RunE: executeBackup,
}

func executeBackup(cmd *cobra.Command, args []string) error {
	// Validate flags
	if err := validateBackupFlags(); err != nil {
		return err
	}

	// Unlock vault with master password
	if err := ensureUnlocked(); err != nil {
		return err
	}
	defer v.Lock()

	// Determine output
	var output *os.File
	if backupStdout {
		output = os.Stdout
	} else {
		// Check if file exists
		if !backupForce {
			if _, err := os.Stat(backupOutput); err == nil {
				return fmt.Errorf("output file already exists: %s (use --force to overwrite)", backupOutput)
			}
		}

		var err error
		output, err = os.OpenFile(backupOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer output.Close()
	}

	// Get encryption password/key
	var password []byte
	var keyFilePath string

	if backupKeyFile != "" {
		keyFilePath = backupKeyFile
	} else if backupBackupPassword {
		// Prompt for separate backup password
		pwd, err := promptBackupPassword()
		if err != nil {
			return err
		}
		password = pwd
	}
	// If neither key-file nor backup-password, use master password (already unlocked)

	// Create backup options
	opts := backup.BackupOptions{
		Output:       output,
		IncludeAudit: backupWithAudit,
		Password:     password,
		KeyFile:      keyFilePath,
	}

	// Perform backup
	if err := backup.Backup(v, opts); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	if !backupStdout {
		fmt.Printf("Backup created successfully: %s\n", backupOutput)
	}

	return nil
}

func validateBackupFlags() error {
	if !backupStdout && backupOutput == "" {
		return fmt.Errorf("either --output or --stdout is required")
	}
	if backupStdout && backupOutput != "" {
		return fmt.Errorf("--output and --stdout are mutually exclusive")
	}
	if backupKeyFile != "" && backupBackupPassword {
		return fmt.Errorf("--key-file and --backup-password are mutually exclusive")
	}
	return nil
}

func promptBackupPassword() ([]byte, error) {
	fmt.Print("Enter backup password: ")
	password1, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println()

	fmt.Print("Confirm backup password: ")
	password2, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println()

	if string(password1) != string(password2) {
		return nil, fmt.Errorf("passwords do not match")
	}

	if len(password1) == 0 {
		return nil, fmt.Errorf("password cannot be empty")
	}

	return password1, nil
}
