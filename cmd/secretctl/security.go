package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/forest6511/secretctl/pkg/security"
	"github.com/forest6511/secretctl/pkg/vault"

	"github.com/spf13/cobra"
)

// Security command flags
var (
	securityVerbose bool
	securityJSON    bool
	securityDays    int
)

// securityCmd is the root security command.
var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Analyze vault security health",
	Long: `Analyze the security health of your vault and get recommendations.

The security score is calculated from:
  - Password Strength (0-25): Average strength of password fields
  - Uniqueness (0-25): Percentage of unique passwords
  - Expiration (0-25): Percentage of non-expired secrets
  - Coverage (0-25): Field coverage for templated secrets

Example:
  secretctl security              # Show security score and top issues
  secretctl security --verbose    # Show all components and suggestions
  secretctl security --json       # Output in JSON format`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		calc := security.NewCalculator(v, security.EditionFree).
			WithExpiryDays(securityDays)

		score, err := calc.CalculateScore(true)
		if err != nil {
			return fmt.Errorf("failed to calculate security score: %w", err)
		}

		if securityJSON {
			return outputSecurityJSON(score)
		}

		return outputSecurityText(score, securityVerbose)
	},
}

// securityDuplicatesCmd lists duplicate passwords.
var securityDuplicatesCmd = &cobra.Command{
	Use:   "duplicates",
	Short: "List duplicate passwords",
	Long: `Show secrets that share the same password.

In Free edition, only the top 3 duplicate groups are shown.
Upgrade to Team edition for the full list.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		secrets, err := v.ListSecrets()
		if err != nil {
			return err
		}

		// Load full secret entries
		var entries []*vault.SecretEntry
		for _, key := range secrets {
			e, err := v.GetSecret(key)
			if err != nil {
				continue
			}
			entries = append(entries, e)
		}

		calc := security.NewCalculator(v, security.EditionFree)
		limits := security.GetLimits(security.EditionFree)

		// Find duplicates
		groups, err := calc.FindDuplicates(entries, true, limits.DuplicateLimit)
		if err != nil {
			return fmt.Errorf("failed to find duplicates: %w", err)
		}

		if len(groups) == 0 {
			fmt.Println("No duplicate passwords found!")
			return nil
		}

		fmt.Printf("Duplicate Passwords (%d groups found)\n\n", len(groups))
		for i, group := range groups {
			fmt.Printf("%d. %d secrets share the same password:\n", i+1, group.Count)
			for _, key := range group.SecretKeys {
				fmt.Printf("   - %s\n", key)
			}
			fmt.Println()
		}

		if limits.DuplicateLimit > 0 && len(groups) >= limits.DuplicateLimit {
			fmt.Println("Upgrade to Team for the full duplicate list.")
		}

		return nil
	},
}

// securityWeakCmd lists weak passwords.
var securityWeakCmd = &cobra.Command{
	Use:   "weak",
	Short: "List weak passwords",
	Long: `Show secrets with weak password fields.

Passwords are considered weak if they are too short:
  - Passwords: Less than 8 characters
  - API keys: Less than 16 characters

In Free edition, only the top 3 weak passwords are shown.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		secrets, err := v.ListSecrets()
		if err != nil {
			return err
		}

		limits := security.GetLimits(security.EditionFree)
		var issues []security.SecurityIssue

		count := 0
		for _, key := range secrets {
			entry, err := v.GetSecret(key)
			if err != nil {
				continue
			}

			for fieldName, f := range entry.Fields {
				if !security.IsPasswordField(fieldName, f.Kind) && !security.IsAPIKeyField(f.Kind) {
					continue
				}
				if f.Value == "" {
					continue
				}

				strength := security.CalculateFieldStrength(f.Value, f.Kind)
				if strength == security.PasswordWeak {
					issues = append(issues, security.SecurityIssue{
						Type:        security.IssueWeakPassword,
						Severity:    security.SeverityWarning,
						SecretKey:   key,
						FieldName:   fieldName,
						Description: fmt.Sprintf("Password is weak (%d characters)", len(f.Value)),
					})
					count++

					if limits.WeakLimit > 0 && count >= limits.WeakLimit {
						break
					}
				}
			}

			if limits.WeakLimit > 0 && count >= limits.WeakLimit {
				break
			}
		}

		if len(issues) == 0 {
			fmt.Println("✅ No weak passwords found!")
			return nil
		}

		fmt.Printf("💪 Weak Passwords (%d found)\n\n", len(issues))
		for i, issue := range issues {
			fmt.Printf("%d. %s / %s\n", i+1, issue.SecretKey, issue.FieldName)
			fmt.Printf("   %s\n\n", issue.Description)
		}

		if limits.WeakLimit > 0 && len(issues) >= limits.WeakLimit {
			fmt.Println("🔓 Upgrade to Team for the full weak password list.")
		}

		return nil
	},
}

// securityExpiringCmd lists expiring secrets.
var securityExpiringCmd = &cobra.Command{
	Use:   "expiring",
	Short: "List secrets expiring soon",
	Long: `Show secrets that will expire within the specified number of days.

By default, shows secrets expiring within 30 days.
Use --days to change the expiration window.

Example:
  secretctl security expiring          # Show secrets expiring within 30 days
  secretctl security expiring --days=7 # Show secrets expiring within 7 days`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		// Use vault's built-in expiring secrets list
		duration := time.Duration(securityDays) * 24 * time.Hour
		entries, err := v.ListExpiringSecrets(duration)
		if err != nil {
			return fmt.Errorf("failed to list expiring secrets: %w", err)
		}

		if len(entries) == 0 {
			fmt.Printf("✅ No secrets expiring within %d days!\n", securityDays)
			return nil
		}

		fmt.Printf("⏰ Secrets Expiring Within %d Days (%d found)\n\n", securityDays, len(entries))
		for i, entry := range entries {
			if entry.ExpiresAt == nil {
				continue
			}
			daysLeft := int(time.Until(*entry.ExpiresAt).Hours() / 24)
			status := "expiring"
			if daysLeft < 0 {
				status = "EXPIRED"
				daysLeft = -daysLeft
			}
			fmt.Printf("%d. %s - %s in %d days\n", i+1, entry.Key, status, daysLeft)
		}

		return nil
	},
}

// outputSecurityJSON outputs the security score as JSON.
func outputSecurityJSON(score *security.SecurityScore) error {
	data, err := json.MarshalIndent(score, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// outputSecurityText outputs the security score as formatted text.
func outputSecurityText(score *security.SecurityScore, verbose bool) error { //nolint:unparam // error return for future use
	// Score header
	emoji := "🔒"
	var rating string
	switch {
	case score.Overall >= 90:
		rating = "Excellent"
	case score.Overall >= 70:
		rating = "Good"
	case score.Overall >= 50:
		emoji = "⚠️"
		rating = "Fair"
	default:
		emoji = "🚨"
		rating = "Needs Attention"
	}

	fmt.Printf("%s Security Score: %d/100 (%s)\n\n", emoji, score.Overall, rating)

	// Components
	fmt.Println("Components:")
	fmt.Printf("  Password Strength: %d/25 %s\n", score.Components.StrengthScore, progressBar(score.Components.StrengthScore, 25))
	fmt.Printf("  Uniqueness:        %d/25 %s\n", score.Components.UniquenessScore, progressBar(score.Components.UniquenessScore, 25))
	fmt.Printf("  Expiration:        %d/25 %s\n", score.Components.ExpirationScore, progressBar(score.Components.ExpirationScore, 25))
	fmt.Printf("  Coverage:          %d/25 %s\n", score.Components.CoverageScore, progressBar(score.Components.CoverageScore, 25))
	fmt.Println()

	// Issues
	if len(score.Issues) > 0 {
		fmt.Printf("⚠️  Top Issues (%d):\n", len(score.Issues))
		for i, issue := range score.Issues {
			typeLabel := strings.ToUpper(string(issue.Type))
			keyInfo := ""
			if issue.SecretKey != "" {
				keyInfo = fmt.Sprintf(" %q", issue.SecretKey)
			} else if len(issue.SecretKeys) > 0 {
				keyInfo = fmt.Sprintf(" %s", strings.Join(issue.SecretKeys, ", "))
			}
			fmt.Printf("  %d. [%s]%s: %s\n", i+1, typeLabel, keyInfo, issue.Description)
		}
		fmt.Println()
	}

	// Suggestions
	if len(score.Suggestions) > 0 && verbose {
		fmt.Println("💡 Suggestions:")
		for _, suggestion := range score.Suggestions {
			fmt.Printf("  - %s\n", suggestion)
		}
		fmt.Println()
	}

	// Freemium notice
	if score.Limited {
		fmt.Println("🔓 Upgrade to Team for full duplicate and weak password lists.")
	}

	return nil
}

// progressBar creates a simple ASCII progress bar.
func progressBar(value, maxVal int) string { //nolint:unparam // maxVal kept for flexibility
	width := 20
	filled := value * width / maxVal
	empty := width - filled
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", empty) + "]"
}

func init() {
	// Add security command to root
	rootCmd.AddCommand(securityCmd)

	// Add subcommands
	securityCmd.AddCommand(securityDuplicatesCmd)
	securityCmd.AddCommand(securityWeakCmd)
	securityCmd.AddCommand(securityExpiringCmd)

	// Add flags
	securityCmd.Flags().BoolVarP(&securityVerbose, "verbose", "v", false, "Show all details including suggestions")
	securityCmd.Flags().BoolVar(&securityJSON, "json", false, "Output in JSON format")
	securityCmd.Flags().IntVar(&securityDays, "days", 30, "Expiration warning window in days")

	securityExpiringCmd.Flags().IntVar(&securityDays, "days", 30, "Expiration window in days")
}
