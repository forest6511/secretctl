package mcp

import (
	"testing"
)

func TestMaskValue(t *testing.T) {
	tests := []struct {
		name     string
		value    []byte
		expected string
	}{
		// Length 0
		{
			name:     "empty value",
			value:    []byte{},
			expected: "",
		},
		// Length 1-4: all asterisks
		{
			name:     "1 character",
			value:    []byte("a"),
			expected: "*",
		},
		{
			name:     "2 characters",
			value:    []byte("ab"),
			expected: "**",
		},
		{
			name:     "3 characters",
			value:    []byte("abc"),
			expected: "***",
		},
		{
			name:     "4 characters",
			value:    []byte("abcd"),
			expected: "****",
		},
		// Length 5-8: show last 2
		{
			name:     "5 characters",
			value:    []byte("abcde"),
			expected: "***de",
		},
		{
			name:     "6 characters",
			value:    []byte("abcdef"),
			expected: "****ef",
		},
		{
			name:     "7 characters",
			value:    []byte("abcdefg"),
			expected: "*****fg",
		},
		{
			name:     "8 characters",
			value:    []byte("abcdefgh"),
			expected: "******gh",
		},
		// Length 9+: show last 4
		{
			name:     "9 characters",
			value:    []byte("abcdefghi"),
			expected: "*****fghi",
		},
		{
			name:     "10 characters",
			value:    []byte("abcdefghij"),
			expected: "******ghij",
		},
		{
			name:     "long value",
			value:    []byte("sk-proj-1234567890abcdef"),
			expected: "********************cdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskValue(tt.value)
			if result != tt.expected {
				t.Errorf("maskValue(%q) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestKeyToEnvName(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"API_KEY", "API_KEY"},
		{"api_key", "API_KEY"},
		{"aws/access_key", "AWS_ACCESS_KEY"},
		{"db-password", "DB_PASSWORD"},
		{"aws/prod/secret-key", "AWS_PROD_SECRET_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := keyToEnvName(tt.key)
			if result != tt.expected {
				t.Errorf("keyToEnvName(%q) = %q, want %q", tt.key, result, tt.expected)
			}
		})
	}
}

func TestValidateEnvName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"API_KEY", false},
		{"_PRIVATE", false},
		{"ABC123", false},
		{"", true},        // empty
		{"123ABC", true},  // starts with number
		{"API-KEY", true}, // contains hyphen
		{"API KEY", true}, // contains space
		{"API.KEY", true}, // contains dot
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnvName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEnvName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		command string
		wantErr bool
	}{
		{"aws", false},
		{"/usr/bin/aws", false},
		{"../evil", true},       // path traversal
		{"./evil/../bad", true}, // path traversal
		{"cmd\x00arg", true},    // NUL byte
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			err := validateCommand(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCommand(%q) error = %v, wantErr %v", tt.command, err, tt.wantErr)
			}
		})
	}
}

func TestValidateArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"normal args", []string{"s3", "ls"}, false},
		{"empty args", []string{}, false},
		{"NUL byte in arg", []string{"arg\x00"}, true},
		{"arg too long", []string{string(make([]byte, 32769))}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateArgs(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
		})
	}
}

func TestOutputSanitizer(t *testing.T) {
	secrets := []secretData{
		{key: "API_KEY", value: []byte("secret123")},
		{key: "DB_PASS", value: []byte("password456")},
		{key: "SHORT", value: []byte("abc")}, // short secrets ARE redacted (security fix)
	}

	sanitizer := newOutputSanitizer(secrets)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no secrets",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "single secret",
			input:    "the key is secret123",
			expected: "the key is [REDACTED:API_KEY]",
		},
		{
			name:     "multiple secrets",
			input:    "key=secret123 pass=password456",
			expected: "key=[REDACTED:API_KEY] pass=[REDACTED:DB_PASS]",
		},
		{
			name:     "short secret IS redacted",
			input:    "short=abc",
			expected: "short=[REDACTED:SHORT]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(sanitizer.sanitize([]byte(tt.input)))
			if result != tt.expected {
				t.Errorf("sanitize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected int // seconds
		wantErr  bool
	}{
		{"30s", 30, false},
		{"5h", 18000, false}, // 5 hours (use 'h' not 'm' which is month)
		{"1h", 3600, false},
		{"7d", 604800, false},
		{"1m", 2592000, false},  // 1 month = 30 days
		{"2w", 1209600, false},  // 2 weeks
		{"1y", 31536000, false}, // 1 year = 365 days
		{"invalid", 0, true},
		{"", 0, true},
		{"x", 0, true}, // too short
		{"abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && int(result.Seconds()) != tt.expected {
				t.Errorf("parseDuration(%q) = %v seconds, want %v", tt.input, result.Seconds(), tt.expected)
			}
		})
	}
}

func TestExpandPattern(t *testing.T) {
	availableKeys := []string{"aws/access_key", "aws/secret_key", "db/password", "api_key"}

	tests := []struct {
		name      string
		pattern   string
		expected  []string
		wantErr   bool
		errString string
	}{
		{
			name:     "exact match",
			pattern:  "api_key",
			expected: []string{"api_key"},
			wantErr:  false,
		},
		{
			name:     "wildcard match",
			pattern:  "aws/*",
			expected: []string{"aws/access_key", "aws/secret_key"},
			wantErr:  false,
		},
		{
			name:     "single char wildcard",
			pattern:  "db/?assword",
			expected: []string{"db/password"},
			wantErr:  false,
		},
		{
			name:      "no match exact",
			pattern:   "nonexistent",
			expected:  nil,
			wantErr:   true,
			errString: "not found",
		},
		{
			name:      "no match glob",
			pattern:   "other/*",
			expected:  nil,
			wantErr:   true,
			errString: "no secrets match",
		},
		{
			name:      "invalid pattern",
			pattern:   "[invalid",
			expected:  nil,
			wantErr:   true,
			errString: "invalid pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandPattern(tt.pattern, availableKeys)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expandPattern(%q) expected error, got nil", tt.pattern)
				} else if tt.errString != "" && !contains(err.Error(), tt.errString) {
					t.Errorf("expandPattern(%q) error = %v, want error containing %q", tt.pattern, err, tt.errString)
				}
				return
			}
			if err != nil {
				t.Errorf("expandPattern(%q) unexpected error: %v", tt.pattern, err)
				return
			}
			if !slicesEqual(result, tt.expected) {
				t.Errorf("expandPattern(%q) = %v, want %v", tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestWipeSecrets(t *testing.T) {
	secrets := []secretData{
		{key: "key1", value: []byte("secret1")},
		{key: "key2", value: []byte("password123")},
	}

	// Verify values exist before wipe
	if string(secrets[0].value) != "secret1" {
		t.Fatal("setup failed")
	}

	wipeSecrets(secrets)

	// Verify all bytes are zeroed
	for i, s := range secrets {
		for j, b := range s.value {
			if b != 0 {
				t.Errorf("secrets[%d].value[%d] = %d, want 0", i, j, b)
			}
		}
	}
}

func TestOutputSanitizerMultipleOccurrences(t *testing.T) {
	secrets := []secretData{
		{key: "API_KEY", value: []byte("secret123")},
	}
	sanitizer := newOutputSanitizer(secrets)

	// Multiple occurrences should all be redacted
	input := []byte("first: secret123, second: secret123")
	result := sanitizer.sanitize(input)
	expected := "first: [REDACTED:API_KEY], second: [REDACTED:API_KEY]"

	if string(result) != expected {
		t.Errorf("sanitize = %q, want %q", result, expected)
	}
}

func TestOutputSanitizerEmpty(t *testing.T) {
	var secrets []secretData
	sanitizer := newOutputSanitizer(secrets)

	// Empty secrets should return unchanged data
	input := []byte("some data")
	result := sanitizer.sanitize(input)

	if string(result) != string(input) {
		t.Errorf("sanitize with no secrets = %q, want %q", result, input)
	}
}

func TestOutputSanitizerEncodedForms(t *testing.T) {
	// Test secret: "password" (known values for encoding)
	secrets := []secretData{
		{key: "SECRET", value: []byte("password")},
	}
	sanitizer := newOutputSanitizer(secrets)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "raw value",
			input:    "value=password",
			expected: "value=[REDACTED:SECRET]",
		},
		{
			name:     "base64 encoded (padded)",
			input:    "encoded=cGFzc3dvcmQ=", // "password" in base64
			expected: "encoded=[REDACTED:SECRET]",
		},
		{
			name:     "base64 encoded (raw/unpadded)",
			input:    "jwt=cGFzc3dvcmQ", // raw base64 without padding
			expected: "jwt=[REDACTED:SECRET]",
		},
		{
			name:     "hex lowercase",
			input:    "hex=70617373776f7264", // "password" in hex
			expected: "hex=[REDACTED:SECRET]",
		},
		{
			name:     "hex uppercase",
			input:    "hex=70617373776F7264", // HEX("password") - uppercase
			expected: "hex=[REDACTED:SECRET]",
		},
		{
			name:     "hex with 0x prefix lowercase",
			input:    "debug=0x70617373776f7264",
			expected: "debug=[REDACTED:SECRET]",
		},
		{
			name:     "hex with 0X prefix uppercase",
			input:    "debug=0X70617373776F7264",
			expected: "debug=[REDACTED:SECRET]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(sanitizer.sanitize([]byte(tt.input)))
			if result != tt.expected {
				t.Errorf("sanitize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestOutputSanitizerURLEncoded(t *testing.T) {
	// Test secret with special chars that will be URL-encoded differently
	secrets := []secretData{
		{key: "PASS", value: []byte("pass word!")}, // contains space and !
	}
	sanitizer := newOutputSanitizer(secrets)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "raw value",
			input:    "p=pass word!",
			expected: "p=[REDACTED:PASS]",
		},
		{
			name:     "QueryEscape (space as +)",
			input:    "p=pass+word%21", // space becomes + in query encoding
			expected: "p=[REDACTED:PASS]",
		},
		{
			name:     "PathEscape (space as %20)",
			input:    "p=pass%20word%21", // space becomes %20 in path encoding
			expected: "p=[REDACTED:PASS]",
		},
		{
			name:     "lowercase percent codes",
			input:    "p=pass+word%21", // lowercase should also match
			expected: "p=[REDACTED:PASS]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(sanitizer.sanitize([]byte(tt.input)))
			if result != tt.expected {
				t.Errorf("sanitize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestOutputSanitizerLongestFirst(t *testing.T) {
	// Test that longer matches are replaced first to avoid partial replacements
	secrets := []secretData{
		{key: "SHORT", value: []byte("secret")},
		{key: "LONG", value: []byte("secretkey")}, // contains "secret"
	}
	sanitizer := newOutputSanitizer(secrets)

	// "secretkey" should be replaced as a whole, not "secret" + "key"
	input := "value=secretkey"
	result := string(sanitizer.sanitize([]byte(input)))
	expected := "value=[REDACTED:LONG]"

	if result != expected {
		t.Errorf("sanitize(%q) = %q, want %q", input, result, expected)
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || substr == "" ||
		(s != "" && substr != "" && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
