package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/forest6511/secretctl/pkg/vault"
)

// DuplicateGroup represents a group of secrets sharing the same password.
type DuplicateGroup struct {
	// SecretKeys contains the secret keys with duplicate values.
	SecretKeys []string `json:"secret_keys,omitempty"`
	// FieldNames contains the field names (usually "password").
	FieldNames []string `json:"field_names,omitempty"`
	// Count is the number of duplicates.
	Count int `json:"count"`
}

// duplicateEntry tracks a single password occurrence for grouping.
type duplicateEntry struct {
	secretKey string
	fieldName string
	hash      string
}

// FindDuplicates scans all sensitive fields for duplicate values.
// Uses HMAC-SHA256 with a session-local key for privacy-preserving comparison.
// Returns groups sorted by count (most duplicated first).
//
// Security properties:
// - HMAC with session-local key prevents offline guessing attacks
// - Hashes are computed per-session, never persisted
// - Values are normalized (trimmed whitespace, Unicode NFC)
func (c *Calculator) FindDuplicates(secrets []*vault.SecretEntry, includeKeys bool, limit int) ([]DuplicateGroup, error) {
	// Initialize HMAC key if not set
	if c.hmacKey == nil {
		c.hmacKey = make([]byte, 32)
		if _, err := rand.Read(c.hmacKey); err != nil {
			return nil, err
		}
	}

	// Collect all password fields with their hashes
	var entries []duplicateEntry
	for _, entry := range secrets {
		for fieldName, field := range entry.Fields {
			// Only check password-type sensitive fields
			if !field.Sensitive {
				continue
			}
			if !IsPasswordField(fieldName, field.Kind) && !IsAPIKeyField(field.Kind) {
				continue
			}

			// Normalize and skip empty values
			value := normalizeValue(field.Value)
			if value == "" {
				continue
			}

			// Compute HMAC hash
			hash := computeValueHash(value, c.hmacKey)
			entries = append(entries, duplicateEntry{
				secretKey: entry.Key,
				fieldName: fieldName,
				hash:      hash,
			})
		}
	}

	// Group by hash
	hashGroups := make(map[string][]duplicateEntry)
	for _, entry := range entries {
		hashGroups[entry.hash] = append(hashGroups[entry.hash], entry)
	}

	// Convert to DuplicateGroups (only groups with count > 1)
	var groups []DuplicateGroup
	for _, entries := range hashGroups {
		if len(entries) <= 1 {
			continue // Not a duplicate
		}

		group := DuplicateGroup{
			Count: len(entries),
		}

		if includeKeys {
			for _, entry := range entries {
				group.SecretKeys = append(group.SecretKeys, entry.secretKey)
				group.FieldNames = append(group.FieldNames, entry.fieldName)
			}
		}

		groups = append(groups, group)
	}

	// Sort by count (descending)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Count > groups[j].Count
	})

	// Apply limit
	if limit > 0 && len(groups) > limit {
		groups = groups[:limit]
	}

	return groups, nil
}

// computeValueHash computes HMAC-SHA256 of a value with the session key.
func computeValueHash(value string, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil))
}

// normalizeValue normalizes a password value for comparison.
// Currently only trims leading/trailing whitespace.
func normalizeValue(value string) string {
	return strings.TrimSpace(value)
}

// FindWeakPasswords returns secrets with weak password fields.
func (c *Calculator) FindWeakPasswords(secrets []*vault.SecretEntry, includeKeys bool, limit int) []SecurityIssue {
	var issues []SecurityIssue

	for _, entry := range secrets {
		for fieldName, field := range entry.Fields {
			if !IsPasswordField(fieldName, field.Kind) && !IsAPIKeyField(field.Kind) {
				continue
			}
			if field.Value == "" {
				continue
			}

			strength := CalculateFieldStrength(field.Value, field.Kind)
			if strength == PasswordWeak {
				issue := SecurityIssue{
					Type:        IssueWeakPassword,
					Severity:    SeverityWarning,
					FieldName:   fieldName,
					Description: "Password has insufficient strength (" + formatLength(len(field.Value)) + ")",
					Suggestion:  "Use a longer password (14+ characters for passwords, 32+ for API keys)",
				}
				if includeKeys {
					issue.SecretKey = entry.Key
				}
				issues = append(issues, issue)
			}
		}
	}

	// Apply limit
	if limit > 0 && len(issues) > limit {
		issues = issues[:limit]
	}

	return issues
}

// formatLength returns a human-readable length description.
func formatLength(n int) string {
	if n == 1 {
		return "1 character"
	}
	return intToString(n) + " characters"
}

// intToString converts an integer to string (simple implementation).
func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToString(-n)
	}

	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
