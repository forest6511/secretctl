// Package main provides the secretctl CLI commands.
package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// Character set constants
const (
	charsetLowercase = "abcdefghijklmnopqrstuvwxyz"
	charsetUppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	charsetDigits    = "0123456789"
	charsetSymbols   = "!@#$%^&*()_+-=[]{}|;:,.<>?"

	minPasswordLength     = 8
	maxPasswordLength     = 256
	defaultPasswordLength = 24
	defaultPasswordCount  = 1
	maxPasswordCount      = 100
	maxExcludeLength      = 256
)

// Generate command flags
var (
	generateLength      int
	generateCount       int
	generateNoSymbols   bool
	generateNoNumbers   bool
	generateNoUppercase bool
	generateNoLowercase bool
	generateExclude     string
	generateCopy        bool
)

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().IntVarP(&generateLength, "length", "l", defaultPasswordLength, "Password length (8-256)")
	generateCmd.Flags().IntVarP(&generateCount, "count", "n", defaultPasswordCount, "Number of passwords to generate (1-100)")
	generateCmd.Flags().BoolVar(&generateNoSymbols, "no-symbols", false, "Exclude symbols")
	generateCmd.Flags().BoolVar(&generateNoNumbers, "no-numbers", false, "Exclude numbers")
	generateCmd.Flags().BoolVar(&generateNoUppercase, "no-uppercase", false, "Exclude uppercase letters")
	generateCmd.Flags().BoolVar(&generateNoLowercase, "no-lowercase", false, "Exclude lowercase letters")
	generateCmd.Flags().StringVar(&generateExclude, "exclude", "", "Characters to exclude")
	generateCmd.Flags().BoolVarP(&generateCopy, "copy", "c", false, "Copy first password to clipboard (accessible to all processes)")
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate secure random passwords",
	Long: `Generate cryptographically secure random passwords.

Examples:
  # Generate a 24-character password (default)
  secretctl generate

  # Generate a 32-character password without symbols
  secretctl generate -l 32 --no-symbols

  # Generate 5 passwords
  secretctl generate -n 5

  # Generate and copy to clipboard
  secretctl generate -c

  # Generate password excluding ambiguous characters
  secretctl generate --exclude "0O1lI"`,
	RunE: executeGenerate,
}

func executeGenerate(cmd *cobra.Command, args []string) error {
	// Validate flags
	if err := validateGenerateFlags(); err != nil {
		return err
	}

	// Build character set
	charset, err := buildCharset()
	if err != nil {
		return err
	}

	// Generate passwords
	passwords := make([]string, generateCount)
	for i := 0; i < generateCount; i++ {
		password, err := generatePassword(charset, generateLength)
		if err != nil {
			return fmt.Errorf("failed to generate password: %w", err)
		}
		passwords[i] = password
	}

	// Output passwords
	for _, password := range passwords {
		fmt.Println(password)
	}

	// Copy to clipboard if requested
	if generateCopy && len(passwords) > 0 {
		if err := copyToClipboard(passwords[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to copy to clipboard: %v\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "Password copied to clipboard")
		}
	}

	return nil
}

// validateGenerateFlags validates the generate command flags
func validateGenerateFlags() error {
	if generateLength < minPasswordLength {
		return fmt.Errorf("password length must be at least %d characters", minPasswordLength)
	}
	if generateLength > maxPasswordLength {
		return fmt.Errorf("password length must be at most %d characters", maxPasswordLength)
	}
	if generateCount < 1 {
		return fmt.Errorf("count must be at least 1")
	}
	if generateCount > maxPasswordCount {
		return fmt.Errorf("count must be at most %d", maxPasswordCount)
	}
	if len(generateExclude) > maxExcludeLength {
		return fmt.Errorf("exclude string must be at most %d characters", maxExcludeLength)
	}
	return nil
}

// buildCharset builds the character set based on flags
func buildCharset() (string, error) {
	var charset strings.Builder

	if !generateNoLowercase {
		charset.WriteString(charsetLowercase)
	}
	if !generateNoUppercase {
		charset.WriteString(charsetUppercase)
	}
	if !generateNoNumbers {
		charset.WriteString(charsetDigits)
	}
	if !generateNoSymbols {
		charset.WriteString(charsetSymbols)
	}

	result := charset.String()

	// Remove excluded characters
	if generateExclude != "" {
		result = removeChars(result, generateExclude)
	}

	if result == "" {
		return "", fmt.Errorf("character set is empty: adjust flags to include at least one character type")
	}

	return result, nil
}

// removeChars removes specified characters from a string
func removeChars(s, chars string) string {
	excludeSet := make(map[rune]bool)
	for _, c := range chars {
		excludeSet[c] = true
	}

	var result strings.Builder
	for _, c := range s {
		if !excludeSet[c] {
			result.WriteRune(c)
		}
	}
	return result.String()
}

// generatePassword generates a cryptographically secure random password
func generatePassword(charset string, length int) (string, error) {
	charsetLen := big.NewInt(int64(len(charset)))
	password := make([]byte, length)

	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		password[i] = charset[idx.Int64()]
	}

	return string(password), nil
}

// copyToClipboard copies text to the system clipboard
func copyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("clipboard tool not found: install xclip or xsel")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
