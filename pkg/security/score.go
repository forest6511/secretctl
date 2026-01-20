package security

import (
	"strconv"
	"time"

	"github.com/forest6511/secretctl/pkg/vault"
)

// SecurityScore represents the overall security assessment of a vault.
type SecurityScore struct {
	// Overall is the total score (0-100).
	Overall int `json:"overall"`
	// Components breaks down the score into categories.
	Components ScoreComponents `json:"components"`
	// Issues contains the detected security issues.
	Issues []SecurityIssue `json:"issues"`
	// Suggestions provides actionable recommendations.
	Suggestions []string `json:"suggestions"`
	// Limited indicates if results were limited due to edition.
	Limited bool `json:"limited"`
}

// ScoreComponents breaks down the security score into categories.
// Each component contributes up to 25 points (total: 100).
type ScoreComponents struct {
	// StrengthScore is based on average password strength (0-25).
	StrengthScore int `json:"strength"`
	// UniquenessScore is based on percentage of unique passwords (0-25).
	UniquenessScore int `json:"uniqueness"`
	// ExpirationScore is based on percentage of non-expired secrets (0-25).
	ExpirationScore int `json:"expiration"`
	// CoverageScore is based on field coverage for templated secrets (0-25).
	CoverageScore int `json:"coverage"`
}

// IssueType identifies the type of security issue.
type IssueType string

const (
	// IssueWeakPassword indicates a password with insufficient strength.
	IssueWeakPassword IssueType = "weak"
	// IssueDuplicatePassword indicates passwords reused across secrets.
	IssueDuplicatePassword IssueType = "duplicate"
	// IssueExpiringSoon indicates a secret expiring within the warning period.
	IssueExpiringSoon IssueType = "expiring"
	// IssueExpired indicates a secret that has already expired.
	IssueExpired IssueType = "expired"
	// IssueMissingField indicates a required field is missing.
	IssueMissingField IssueType = "missing_field"
)

// Severity indicates the urgency of a security issue.
type Severity string

const (
	// SeverityCritical requires immediate attention.
	SeverityCritical Severity = "critical"
	// SeverityWarning should be addressed soon.
	SeverityWarning Severity = "warning"
	// SeverityInfo is informational only.
	SeverityInfo Severity = "info"
)

// SecurityIssue represents a detected security problem.
type SecurityIssue struct {
	// Type identifies the category of issue.
	Type IssueType `json:"type"`
	// Severity indicates urgency.
	Severity Severity `json:"severity"`
	// SecretKey is the affected secret (may be empty for privacy).
	SecretKey string `json:"secret_key,omitempty"`
	// SecretKeys is used for duplicate issues (multiple secrets).
	SecretKeys []string `json:"secret_keys,omitempty"`
	// FieldName is the specific field with the issue.
	FieldName string `json:"field_name,omitempty"`
	// Description explains the issue.
	Description string `json:"description"`
	// Suggestion provides remediation guidance.
	Suggestion string `json:"suggestion,omitempty"`
}

// Calculator computes security scores for a vault.
type Calculator struct {
	vault      *vault.Vault
	edition    Edition
	limits     Limits
	hmacKey    []byte // Session-local key for duplicate detection
	expiryDays int    // Days until expiration to warn (default 30)
}

// NewCalculator creates a new security calculator for the given vault.
func NewCalculator(v *vault.Vault, edition Edition) *Calculator {
	return &Calculator{
		vault:      v,
		edition:    edition,
		limits:     GetLimits(edition),
		expiryDays: 30,
	}
}

// WithExpiryDays sets the number of days to consider as "expiring soon".
func (c *Calculator) WithExpiryDays(days int) *Calculator {
	c.expiryDays = days
	return c
}

// CalculateScore computes the full security score for the vault.
func (c *Calculator) CalculateScore(includeKeys bool) (*SecurityScore, error) {
	secrets, err := c.vault.ListSecrets()
	if err != nil {
		return nil, err
	}

	// Empty vault: perfect score
	if len(secrets) == 0 {
		return &SecurityScore{
			Overall: 100,
			Components: ScoreComponents{
				StrengthScore:   25,
				UniquenessScore: 25,
				ExpirationScore: 25,
				CoverageScore:   25,
			},
			Issues:      []SecurityIssue{},
			Suggestions: []string{},
			Limited:     false,
		}, nil
	}

	// Load all secrets with full details
	var secretEntries []*vault.SecretEntry
	for _, key := range secrets {
		entry, err := c.vault.GetSecret(key)
		if err != nil {
			continue // Skip inaccessible secrets
		}
		secretEntries = append(secretEntries, entry)
	}

	// Calculate each component
	strengthScore, weakIssues := c.calculateStrengthScore(secretEntries, includeKeys)
	uniquenessScore, dupIssues := c.calculateUniquenessScore(secretEntries, includeKeys)
	expirationScore, expIssues := c.calculateExpirationScore(secretEntries, includeKeys)
	coverageScore := c.calculateCoverageScore(secretEntries)

	// Combine all issues
	allIssues := make([]SecurityIssue, 0)
	allIssues = append(allIssues, weakIssues...)
	allIssues = append(allIssues, dupIssues...)
	allIssues = append(allIssues, expIssues...)

	// Apply limits
	limited := false
	if c.limits.IsLimited() {
		allIssues, limited = c.applyLimits(allIssues)
	}

	// Generate suggestions based on issues
	suggestions := c.generateSuggestions(allIssues)

	return &SecurityScore{
		Overall: strengthScore + uniquenessScore + expirationScore + coverageScore,
		Components: ScoreComponents{
			StrengthScore:   strengthScore,
			UniquenessScore: uniquenessScore,
			ExpirationScore: expirationScore,
			CoverageScore:   coverageScore,
		},
		Issues:      allIssues,
		Suggestions: suggestions,
		Limited:     limited,
	}, nil
}

// calculateStrengthScore evaluates password strength across all secrets.
// Returns score (0-25) and weak password issues.
func (c *Calculator) calculateStrengthScore(secrets []*vault.SecretEntry, includeKeys bool) (int, []SecurityIssue) {
	var issues []SecurityIssue
	totalPoints := 0
	passwordCount := 0

	for _, entry := range secrets {
		for fieldName, field := range entry.Fields {
			// Only evaluate password-type fields
			if !IsPasswordField(fieldName, field.Kind) && !IsAPIKeyField(field.Kind) {
				continue
			}

			if field.Value == "" {
				continue
			}

			passwordCount++
			strength := CalculateFieldStrength(field.Value, field.Kind)
			totalPoints += strength.Points()

			if strength == PasswordWeak {
				issue := SecurityIssue{
					Type:        IssueWeakPassword,
					Severity:    SeverityWarning,
					FieldName:   fieldName,
					Description: "Password has insufficient strength",
					Suggestion:  "Use a longer password (14+ characters recommended)",
				}
				if includeKeys {
					issue.SecretKey = entry.Key
				}
				issues = append(issues, issue)
			}
		}
	}

	// No password fields: full score (N/A)
	if passwordCount == 0 {
		return 25, issues
	}

	// Calculate average and scale to 0-25
	avgPoints := float64(totalPoints) / float64(passwordCount)
	score := int(avgPoints)
	if score > 25 {
		score = 25
	}

	return score, issues
}

// calculateUniquenessScore evaluates password reuse across secrets.
// Returns score (0-25) and duplicate issues.
func (c *Calculator) calculateUniquenessScore(secrets []*vault.SecretEntry, includeKeys bool) (int, []SecurityIssue) {
	duplicates, err := c.FindDuplicates(secrets, includeKeys, 0) // No limit for calculation
	if err != nil {
		return 25, nil // On error, assume no duplicates
	}

	// Count total passwords and unique passwords
	passwordHashes := make(map[string]bool)
	totalPasswords := 0

	for _, entry := range secrets {
		for fieldName, field := range entry.Fields {
			if !IsPasswordField(fieldName, field.Kind) && !IsAPIKeyField(field.Kind) {
				continue
			}
			if field.Value == "" {
				continue
			}
			totalPasswords++
			hash := computeValueHash(field.Value, c.hmacKey)
			passwordHashes[hash] = true
		}
	}

	// No passwords: full score (N/A)
	if totalPasswords == 0 {
		return 25, nil
	}

	// Convert duplicates to issues
	var issues []SecurityIssue
	for _, dup := range duplicates {
		issue := SecurityIssue{
			Type:        IssueDuplicatePassword,
			Severity:    SeverityWarning,
			Description: "Multiple secrets share the same password",
			Suggestion:  "Use unique passwords for each secret",
		}
		if includeKeys {
			issue.SecretKeys = dup.SecretKeys
		}
		issues = append(issues, issue)
	}

	// Calculate uniqueness ratio
	uniqueCount := len(passwordHashes)
	uniquenessRatio := float64(uniqueCount) / float64(totalPasswords)
	score := int(uniquenessRatio * 25)

	return score, issues
}

// calculateExpirationScore evaluates expiration status of secrets.
// Returns score (0-25) and expiration issues.
func (c *Calculator) calculateExpirationScore(secrets []*vault.SecretEntry, includeKeys bool) (int, []SecurityIssue) {
	var issues []SecurityIssue
	now := time.Now()
	warningThreshold := now.AddDate(0, 0, c.expiryDays)

	secretsWithExpiration := 0
	nonExpiredCount := 0

	for _, entry := range secrets {
		if entry.ExpiresAt == nil {
			continue // No expiration set
		}

		secretsWithExpiration++
		expiresAt := *entry.ExpiresAt

		//nolint:gocritic // if-else chain is clearer for time comparisons
		if expiresAt.Before(now) {
			// Already expired
			issue := SecurityIssue{
				Type:        IssueExpired,
				Severity:    SeverityCritical,
				Description: "Secret has expired",
				Suggestion:  "Renew or remove expired credentials",
			}
			if includeKeys {
				issue.SecretKey = entry.Key
			}
			issues = append(issues, issue)
		} else if expiresAt.Before(warningThreshold) {
			// Expiring soon (but not expired)
			nonExpiredCount++
			daysLeft := int(expiresAt.Sub(now).Hours() / 24)
			issue := SecurityIssue{
				Type:        IssueExpiringSoon,
				Severity:    SeverityWarning,
				Description: "Secret expires in " + formatDays(daysLeft),
				Suggestion:  "Plan to renew before expiration",
			}
			if includeKeys {
				issue.SecretKey = entry.Key
			}
			issues = append(issues, issue)
		} else {
			nonExpiredCount++
		}
	}

	// No secrets with expiration: full score (N/A)
	if secretsWithExpiration == 0 {
		return 25, issues
	}

	// Calculate score based on non-expired ratio
	nonExpiredRatio := float64(nonExpiredCount) / float64(secretsWithExpiration)
	score := int(nonExpiredRatio * 25)

	return score, issues
}

// calculateCoverageScore evaluates field coverage for templated secrets.
// Currently returns full score (N/A) as templates are Phase 3.
func (c *Calculator) calculateCoverageScore(_ []*vault.SecretEntry) int {
	// Phase 3: Schema/template validation
	// For now, all secrets get full coverage score
	return 25
}

// applyLimits applies edition-based limits to issues.
func (c *Calculator) applyLimits(issues []SecurityIssue) ([]SecurityIssue, bool) {
	limited := false
	weakCount := 0
	dupCount := 0
	var result []SecurityIssue

	for _, issue := range issues {
		switch issue.Type {
		case IssueWeakPassword:
			if c.limits.WeakLimit > 0 && weakCount >= c.limits.WeakLimit {
				limited = true
				continue
			}
			weakCount++
		case IssueDuplicatePassword:
			if c.limits.DuplicateLimit > 0 && dupCount >= c.limits.DuplicateLimit {
				limited = true
				continue
			}
			dupCount++
		}
		result = append(result, issue)
	}

	return result, limited
}

// generateSuggestions creates actionable recommendations based on issues.
func (c *Calculator) generateSuggestions(issues []SecurityIssue) []string {
	var suggestions []string
	hasWeak := false
	hasDuplicate := false
	hasExpiring := false
	hasExpired := false

	for _, issue := range issues {
		switch issue.Type {
		case IssueWeakPassword:
			hasWeak = true
		case IssueDuplicatePassword:
			hasDuplicate = true
		case IssueExpiringSoon:
			hasExpiring = true
		case IssueExpired:
			hasExpired = true
		}
	}

	if hasWeak {
		suggestions = append(suggestions, "Update weak passwords with stronger alternatives (14+ characters)")
	}
	if hasDuplicate {
		suggestions = append(suggestions, "Replace duplicate passwords with unique values")
	}
	if hasExpired {
		suggestions = append(suggestions, "Remove or renew expired credentials immediately")
	}
	if hasExpiring {
		suggestions = append(suggestions, "Plan to renew expiring credentials before they expire")
	}

	return suggestions
}

// formatDays returns a human-readable day count.
func formatDays(days int) string {
	if days == 0 {
		return "today"
	}
	if days == 1 {
		return "1 day"
	}
	return strconv.Itoa(days) + " days"
}
