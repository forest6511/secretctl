package sanitize

import (
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestSanitizeMultiLayer_ExactMatch(t *testing.T) {
	secrets := map[string]string{
		"API_KEY": "sk-secret-key-12345",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "exact match at start",
			input:    "sk-secret-key-12345 is my key",
			expected: "[REDACTED:API_KEY] is my key",
		},
		{
			name:     "exact match at end",
			input:    "my key is sk-secret-key-12345",
			expected: "my key is [REDACTED:API_KEY]",
		},
		{
			name:     "exact match in middle",
			input:    "the sk-secret-key-12345 works",
			expected: "the [REDACTED:API_KEY] works",
		},
		{
			name:     "no match",
			input:    "no secrets here",
			expected: "no secrets here",
		},
		{
			name:     "multiple matches",
			input:    "sk-secret-key-12345 and sk-secret-key-12345",
			expected: "[REDACTED:API_KEY] and [REDACTED:API_KEY]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeMultiLayer(tc.input, secrets, DefaultConfig)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestSanitizeMultiLayer_Base64(t *testing.T) {
	secrets := map[string]string{
		"API_KEY": "my-super-secret-key",
	}

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "base64 encoded secret",
			input:    "The encoded key is " + encodeBase64("my-super-secret-key"),
			contains: "[REDACTED:API_KEY]",
		},
		{
			name:     "base64 with prefix",
			input:    "prefix " + encodeBase64("my-super-secret-key") + " suffix",
			contains: "[REDACTED:API_KEY]",
		},
		{
			name:     "base64 containing secret",
			input:    "data: " + encodeBase64("user=my-super-secret-key&pass=test"),
			contains: "[REDACTED:API_KEY]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeMultiLayer(tc.input, secrets, DefaultConfig)
			if !contains(result, tc.contains) {
				t.Errorf("expected result to contain %q, got %q", tc.contains, result)
			}
		})
	}
}

func TestSanitizeMultiLayer_Hex(t *testing.T) {
	// Use a longer secret so hex encoding produces 40+ chars
	secrets := map[string]string{
		"SECRET": "password123-secret-key-abcdef", // 31 chars -> 62 hex chars
	}

	hexEncoded := encodeHex("password123-secret-key-abcdef")

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "hex encoded secret",
			input:    "The hex key is " + hexEncoded,
			contains: "[REDACTED:SECRET]",
		},
		{
			name:     "hex with prefix",
			input:    "prefix " + hexEncoded + " suffix",
			contains: "[REDACTED:SECRET]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeMultiLayer(tc.input, secrets, DefaultConfig)
			if !contains(result, tc.contains) {
				t.Errorf("expected result to contain %q, got %q", tc.contains, result)
			}
		})
	}
}

func TestSanitizeMultiLayer_Fragments(t *testing.T) {
	secrets := map[string]string{
		"API_KEY": "sk-test-123-abcdef-xyz", // Longer secret for fragment detection
	}

	tests := []struct {
		name         string
		input        string
		shouldRedact bool
	}{
		{
			name:         "PART pattern fragments (not enough matches)",
			input:        "PART1: sk- PART2: test- PART3: 123",
			shouldRedact: false, // Only 3 matches, need 10+
		},
		{
			name:         "many fragments could trigger",
			input:        "PART1:a PART2:b PART3:c PART4:d PART5:e PART6:f PART7:g PART8:h PART9:i PART10:j PART11:k",
			shouldRedact: false, // Fragments don't reconstruct the secret
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeMultiLayer(tc.input, secrets, DefaultConfig)
			redacted := contains(result, "[REDACTED:")
			if tc.shouldRedact && !redacted {
				t.Errorf("expected redaction, got %q", result)
			}
			if !tc.shouldRedact && redacted {
				t.Errorf("unexpected redaction in %q", result)
			}
		})
	}
}

func TestSanitizeMultiLayer_DisabledLayers(t *testing.T) {
	secrets := map[string]string{
		"API_KEY": "secret123",
	}

	encoded := encodeBase64("secret123")

	// With Base64 detection disabled
	config := Config{
		DetectBase64:    false,
		DetectHex:       true,
		DetectFragments: true,
	}

	result := SanitizeMultiLayer("encoded: "+encoded, secrets, config)
	// Should NOT redact since base64 detection is disabled
	if contains(result, "[REDACTED:") {
		t.Errorf("unexpected redaction with base64 disabled: %q", result)
	}
}

func TestContainsSecret(t *testing.T) {
	secrets := map[string]string{
		"API_KEY": "my-super-secret-api-key-12345678", // Long enough for hex encoding
	}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "direct match",
			input:    "key is my-super-secret-api-key-12345678",
			expected: true,
		},
		{
			name:     "no match",
			input:    "no secrets here",
			expected: false,
		},
		{
			name:     "base64 encoded",
			input:    "encoded: " + encodeBase64("my-super-secret-api-key-12345678"),
			expected: true,
		},
		{
			name:     "hex encoded (long enough)",
			input:    "hex: " + encodeHex("my-super-secret-api-key-12345678"),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ContainsSecret(tc.input, secrets)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestShortSecretsNotSanitized(t *testing.T) {
	// Secrets < 4 bytes should not be sanitized (avoid false positives)
	secrets := map[string]string{
		"SHORT": "abc",
	}

	input := "abc here"
	result := SanitizeMultiLayer(input, secrets, DefaultConfig)
	if result != input {
		t.Errorf("short secret should not be sanitized: got %q", result)
	}
}

// Helper functions
func encodeBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func encodeHex(s string) string {
	return hex.EncodeToString([]byte(s))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
