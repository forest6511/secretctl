// Package security provides security analysis and scoring for vault secrets.
package security

// PasswordStrength represents the strength level of a password or API key.
type PasswordStrength int

const (
	// PasswordWeak indicates an insecure password (less than 8 chars for passwords, 16 for API keys).
	PasswordWeak PasswordStrength = iota
	// PasswordFair indicates a minimally acceptable password.
	PasswordFair
	// PasswordGood indicates a good password.
	PasswordGood
	// PasswordStrong indicates a strong password.
	PasswordStrong
)

// String returns a human-readable representation of the password strength.
func (s PasswordStrength) String() string {
	switch s {
	case PasswordWeak:
		return "Weak"
	case PasswordFair:
		return "Fair"
	case PasswordGood:
		return "Good"
	case PasswordStrong:
		return "Strong"
	default:
		return "Unknown"
	}
}

// Points returns the score points for this strength level.
// Used in StrengthScore calculation: Weak=0, Fair=8, Good=17, Strong=25.
func (s PasswordStrength) Points() int {
	switch s {
	case PasswordWeak:
		return 0
	case PasswordFair:
		return 8
	case PasswordGood:
		return 17
	case PasswordStrong:
		return 25
	default:
		return 0
	}
}

// CalculateFieldStrength calculates strength based on field type.
// For "api_key" or "token" kinds, uses entropy-based calculation.
// For other kinds (passwords), uses NIST-recommended length-first approach.
func CalculateFieldStrength(value string, fieldKind string) PasswordStrength {
	if fieldKind == "api_key" || fieldKind == "token" {
		return calculateAPIKeyStrength(value)
	}
	return calculatePasswordStrength(value)
}

// calculatePasswordStrength evaluates human-created passwords.
// Length is the primary factor per NIST guidelines (composition rules discouraged).
// NIST SP 800-63B recommends:
// - Minimum 8 characters for user-chosen passwords
// - No complexity requirements (uppercase, numbers, symbols)
// - Focus on length and avoiding compromised passwords
func calculatePasswordStrength(value string) PasswordStrength {
	length := len(value)

	switch {
	case length >= 20:
		return PasswordStrong
	case length >= 14:
		return PasswordGood
	case length >= 10:
		return PasswordFair
	case length >= 8:
		return PasswordFair
	default:
		return PasswordWeak
	}
}

// calculateAPIKeyStrength evaluates machine-generated tokens.
// For random strings, length directly correlates with entropy:
// - 32+ chars (~128 bits for alphanumeric): Strong
// - 20+ chars (~80 bits): Good
// - 16+ chars (~64 bits): Fair
// - Less than 16: Weak
func calculateAPIKeyStrength(value string) PasswordStrength {
	length := len(value)

	switch {
	case length >= 32:
		return PasswordStrong
	case length >= 20:
		return PasswordGood
	case length >= 16:
		return PasswordFair
	default:
		return PasswordWeak
	}
}

// IsPasswordField determines if a field should be treated as a password
// based on its kind and name.
func IsPasswordField(fieldName string, fieldKind string) bool {
	// Explicit password kind
	if fieldKind == "password" {
		return true
	}

	// Common password field names
	passwordNames := []string{
		"password", "pwd", "pass", "passwd",
		"secret", "credential", "credentials",
	}

	lowerName := toLowerCase(fieldName)
	for _, name := range passwordNames {
		if lowerName == name || contains(lowerName, name) {
			return true
		}
	}

	return false
}

// IsAPIKeyField determines if a field should be treated as an API key or token.
func IsAPIKeyField(fieldKind string) bool {
	return fieldKind == "api_key" || fieldKind == "token"
}

// toLowerCase returns the lowercase version of a string.
func toLowerCase(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// contains checks if s contains substr (simple implementation).
func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
