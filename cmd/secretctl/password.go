// Package main provides the secretctl CLI application.
package main

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/forest6511/secretctl/pkg/crypto"
	"github.com/forest6511/secretctl/pkg/vault"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// passwordCmd is the parent command for password operations.
var passwordCmd = &cobra.Command{
	Use:   "password",
	Short: "Master password operations",
}

// passwordChangeCmd changes the master password.
var passwordChangeCmd = &cobra.Command{
	Use:   "change",
	Short: "Change the master password",
	Long: `Change the master password by re-wrapping the data encryption key (DEK).

This operation:
  1. Verifies the current password
  2. Creates a backup before making changes
  3. Re-wraps the DEK with the new password
  4. All secrets remain accessible with the new password

The change is atomic: either fully succeeds or has no effect.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ensure vault is unlocked (prompts for password if needed)
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		fmt.Println("Changing master password...")
		fmt.Println()

		// 1. Prompt for current password (for verification)
		fmt.Print("Enter current password: ")
		currentPassword, err := term.ReadPassword(int(syscall.Stdin))
		defer crypto.SecureWipe(currentPassword)
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		// 2. Prompt for new password
		fmt.Print("Enter new password: ")
		newPassword1, err := term.ReadPassword(int(syscall.Stdin))
		defer crypto.SecureWipe(newPassword1)
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		// 3. Confirm new password
		fmt.Print("Confirm new password: ")
		newPassword2, err := term.ReadPassword(int(syscall.Stdin))
		defer crypto.SecureWipe(newPassword2)
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		// 4. Check new passwords match
		if string(newPassword1) != string(newPassword2) {
			return errors.New("new passwords do not match")
		}

		// 5. Validate new password strength
		validation := vault.ValidateMasterPassword(string(newPassword1))
		if !validation.Valid {
			return fmt.Errorf("password validation failed: %s", validation.Warnings[0])
		}

		// Display password strength
		fmt.Printf("New password strength: %s\n", validation.Strength)
		for _, warning := range validation.Warnings {
			fmt.Printf("Warning: %s\n", warning)
		}
		fmt.Println()

		// 6. Execute password change
		fmt.Println("Changing password...")
		if err := v.ChangePassword(string(currentPassword), string(newPassword1)); err != nil {
			if errors.Is(err, vault.ErrInvalidPassword) {
				return errors.New("current password is incorrect")
			}
			if errors.Is(err, vault.ErrSamePassword) {
				return errors.New("new password must be different from current password")
			}
			return fmt.Errorf("failed to change password: %w", err)
		}

		fmt.Println()
		fmt.Println("Password changed successfully!")
		fmt.Println("A backup of your vault was created before the change.")

		return nil
	},
}
