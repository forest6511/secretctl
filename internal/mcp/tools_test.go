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
		{"", true},           // empty
		{"123ABC", true},     // starts with number
		{"API-KEY", true},    // contains hyphen
		{"API KEY", true},    // contains space
		{"API.KEY", true},    // contains dot
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
		{"../evil", true},         // path traversal
		{"./evil/../bad", true},   // path traversal
		{"cmd\x00arg", true},      // NUL byte
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
		{key: "SHORT", value: []byte("abc")}, // too short, should be skipped
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
			name:     "short secret not redacted",
			input:    "short=abc",
			expected: "short=abc",
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
		{"5h", 18000, false},       // 5 hours (use 'h' not 'm' which is month)
		{"1h", 3600, false},
		{"7d", 604800, false},
		{"1m", 2592000, false},     // 1 month = 30 days
		{"invalid", 0, true},
		{"", 0, true},
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
