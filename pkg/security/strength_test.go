package security

import "testing"

func TestPasswordStrength_String(t *testing.T) {
	tests := []struct {
		strength PasswordStrength
		want     string
	}{
		{PasswordWeak, "Weak"},
		{PasswordFair, "Fair"},
		{PasswordGood, "Good"},
		{PasswordStrong, "Strong"},
		{PasswordStrength(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.strength.String(); got != tt.want {
				t.Errorf("PasswordStrength.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPasswordStrength_Points(t *testing.T) {
	tests := []struct {
		strength PasswordStrength
		want     int
	}{
		{PasswordWeak, 0},
		{PasswordFair, 8},
		{PasswordGood, 17},
		{PasswordStrong, 25},
		{PasswordStrength(99), 0},
	}

	for _, tt := range tests {
		t.Run(tt.strength.String(), func(t *testing.T) {
			if got := tt.strength.Points(); got != tt.want {
				t.Errorf("PasswordStrength.Points() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateFieldStrength_Password(t *testing.T) {
	tests := []struct {
		name  string
		value string
		kind  string
		want  PasswordStrength
	}{
		{"empty", "", "", PasswordWeak},
		{"very_short", "abc", "", PasswordWeak},
		{"7_chars", "1234567", "", PasswordWeak},
		{"8_chars", "12345678", "", PasswordFair},
		{"10_chars", "1234567890", "", PasswordFair},
		{"13_chars", "1234567890abc", "", PasswordFair},
		{"14_chars", "1234567890abcd", "", PasswordGood},
		{"19_chars", "1234567890abcdefghi", "", PasswordGood},
		{"20_chars", "1234567890abcdefghij", "", PasswordStrong},
		{"30_chars", "123456789012345678901234567890", "", PasswordStrong},
		// With kind="password"
		{"password_kind_short", "abc", "password", PasswordWeak},
		{"password_kind_long", "1234567890abcdefghij", "password", PasswordStrong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateFieldStrength(tt.value, tt.kind)
			if got != tt.want {
				t.Errorf("CalculateFieldStrength(%q, %q) = %v, want %v", tt.value, tt.kind, got, tt.want)
			}
		})
	}
}

func TestCalculateFieldStrength_APIKey(t *testing.T) {
	tests := []struct {
		name  string
		value string
		kind  string
		want  PasswordStrength
	}{
		{"api_key_empty", "", "api_key", PasswordWeak},
		{"api_key_short", "abc123", "api_key", PasswordWeak},
		{"api_key_15", "123456789012345", "api_key", PasswordWeak},
		{"api_key_16", "1234567890123456", "api_key", PasswordFair},
		{"api_key_19", "1234567890123456789", "api_key", PasswordFair},
		{"api_key_20", "12345678901234567890", "api_key", PasswordGood},
		{"api_key_31", "1234567890123456789012345678901", "api_key", PasswordGood},
		{"api_key_32", "12345678901234567890123456789012", "api_key", PasswordStrong},
		{"api_key_64", "1234567890123456789012345678901234567890123456789012345678901234", "api_key", PasswordStrong},
		// Token kind
		{"token_short", "abc", "token", PasswordWeak},
		{"token_32", "12345678901234567890123456789012", "token", PasswordStrong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateFieldStrength(tt.value, tt.kind)
			if got != tt.want {
				t.Errorf("CalculateFieldStrength(%q, %q) = %v, want %v", tt.value, tt.kind, got, tt.want)
			}
		})
	}
}

func TestIsPasswordField(t *testing.T) {
	tests := []struct {
		fieldName string
		fieldKind string
		want      bool
	}{
		{"password", "", true},
		{"Password", "", true},
		{"PASSWORD", "", true},
		{"pwd", "", true},
		{"pass", "", true},
		{"passwd", "", true},
		{"secret", "", true},
		{"credential", "", true},
		{"credentials", "", true},
		{"my_password", "", true},
		{"db_pass", "", true},
		{"user_passwd", "", true},
		{"any_field", "password", true},
		{"username", "", false},
		{"email", "", false},
		{"api_key", "", false},
		{"token", "", false},
		{"url", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName+"_"+tt.fieldKind, func(t *testing.T) {
			got := IsPasswordField(tt.fieldName, tt.fieldKind)
			if got != tt.want {
				t.Errorf("IsPasswordField(%q, %q) = %v, want %v", tt.fieldName, tt.fieldKind, got, tt.want)
			}
		})
	}
}

func TestIsAPIKeyField(t *testing.T) {
	tests := []struct {
		kind string
		want bool
	}{
		{"api_key", true},
		{"token", true},
		{"password", false},
		{"", false},
		{"secret", false},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			got := IsAPIKeyField(tt.kind)
			if got != tt.want {
				t.Errorf("IsAPIKeyField(%q) = %v, want %v", tt.kind, got, tt.want)
			}
		})
	}
}
