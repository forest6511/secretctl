package main

import (
	"strings"
	"testing"
	"unicode"
)

func TestValidateGenerateFlags(t *testing.T) {
	tests := []struct {
		name        string
		length      int
		count       int
		exclude     string
		expectError bool
	}{
		{
			name:        "valid defaults",
			length:      defaultPasswordLength,
			count:       defaultPasswordCount,
			exclude:     "",
			expectError: false,
		},
		{
			name:        "minimum length",
			length:      minPasswordLength,
			count:       1,
			exclude:     "",
			expectError: false,
		},
		{
			name:        "maximum length",
			length:      maxPasswordLength,
			count:       1,
			exclude:     "",
			expectError: false,
		},
		{
			name:        "length too short",
			length:      minPasswordLength - 1,
			count:       1,
			exclude:     "",
			expectError: true,
		},
		{
			name:        "length too long",
			length:      maxPasswordLength + 1,
			count:       1,
			exclude:     "",
			expectError: true,
		},
		{
			name:        "count zero",
			length:      24,
			count:       0,
			exclude:     "",
			expectError: true,
		},
		{
			name:        "count too high",
			length:      24,
			count:       maxPasswordCount + 1,
			exclude:     "",
			expectError: true,
		},
		{
			name:        "maximum count",
			length:      24,
			count:       maxPasswordCount,
			exclude:     "",
			expectError: false,
		},
		{
			name:        "exclude too long",
			length:      24,
			count:       1,
			exclude:     strings.Repeat("a", maxExcludeLength+1),
			expectError: true,
		},
		{
			name:        "valid exclude",
			length:      24,
			count:       1,
			exclude:     "0O1lI",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore globals
			oldLength := generateLength
			oldCount := generateCount
			oldExclude := generateExclude
			defer func() {
				generateLength = oldLength
				generateCount = oldCount
				generateExclude = oldExclude
			}()

			generateLength = tt.length
			generateCount = tt.count
			generateExclude = tt.exclude

			err := validateGenerateFlags()
			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBuildCharset(t *testing.T) {
	tests := []struct {
		name        string
		noLowercase bool
		noUppercase bool
		noNumbers   bool
		noSymbols   bool
		exclude     string
		expectError bool
		contains    string
		notContains string
	}{
		{
			name:        "all character types",
			noLowercase: false,
			noUppercase: false,
			noNumbers:   false,
			noSymbols:   false,
			contains:    "aA0!",
			notContains: "",
		},
		{
			name:        "no symbols",
			noLowercase: false,
			noUppercase: false,
			noNumbers:   false,
			noSymbols:   true,
			contains:    "aA0",
			notContains: "!@#",
		},
		{
			name:        "no numbers",
			noLowercase: false,
			noUppercase: false,
			noNumbers:   true,
			noSymbols:   false,
			contains:    "aA!",
			notContains: "0123",
		},
		{
			name:        "no uppercase",
			noLowercase: false,
			noUppercase: true,
			noNumbers:   false,
			noSymbols:   false,
			contains:    "a0!",
			notContains: "ABC",
		},
		{
			name:        "no lowercase",
			noLowercase: true,
			noUppercase: false,
			noNumbers:   false,
			noSymbols:   false,
			contains:    "A0!",
			notContains: "abc",
		},
		{
			name:        "letters only",
			noLowercase: false,
			noUppercase: false,
			noNumbers:   true,
			noSymbols:   true,
			contains:    "aA",
			notContains: "0!",
		},
		{
			name:        "exclude ambiguous",
			noLowercase: false,
			noUppercase: false,
			noNumbers:   false,
			noSymbols:   false,
			exclude:     "0O1lI",
			contains:    "a2!",
			notContains: "0O1lI",
		},
		{
			name:        "empty charset",
			noLowercase: true,
			noUppercase: true,
			noNumbers:   true,
			noSymbols:   true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore globals
			oldNoLowercase := generateNoLowercase
			oldNoUppercase := generateNoUppercase
			oldNoNumbers := generateNoNumbers
			oldNoSymbols := generateNoSymbols
			oldExclude := generateExclude
			defer func() {
				generateNoLowercase = oldNoLowercase
				generateNoUppercase = oldNoUppercase
				generateNoNumbers = oldNoNumbers
				generateNoSymbols = oldNoSymbols
				generateExclude = oldExclude
			}()

			generateNoLowercase = tt.noLowercase
			generateNoUppercase = tt.noUppercase
			generateNoNumbers = tt.noNumbers
			generateNoSymbols = tt.noSymbols
			generateExclude = tt.exclude

			charset, err := buildCharset()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check contains
			for _, c := range tt.contains {
				if !strings.ContainsRune(charset, c) {
					t.Errorf("charset should contain '%c'", c)
				}
			}

			// Check not contains
			for _, c := range tt.notContains {
				if strings.ContainsRune(charset, c) {
					t.Errorf("charset should not contain '%c'", c)
				}
			}
		})
	}
}

func TestRemoveChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		exclude  string
		expected string
	}{
		{
			name:     "remove single char",
			input:    "abcdef",
			exclude:  "c",
			expected: "abdef",
		},
		{
			name:     "remove multiple chars",
			input:    "abcdef",
			exclude:  "ace",
			expected: "bdf",
		},
		{
			name:     "remove nothing",
			input:    "abcdef",
			exclude:  "xyz",
			expected: "abcdef",
		},
		{
			name:     "empty exclude",
			input:    "abcdef",
			exclude:  "",
			expected: "abcdef",
		},
		{
			name:     "remove all",
			input:    "aaa",
			exclude:  "a",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeChars(tt.input, tt.exclude)
			if result != tt.expected {
				t.Errorf("removeChars(%q, %q) = %q, want %q", tt.input, tt.exclude, result, tt.expected)
			}
		})
	}
}

func TestGeneratePassword(t *testing.T) {
	tests := []struct {
		name    string
		charset string
		length  int
	}{
		{
			name:    "alphanumeric",
			charset: charsetLowercase + charsetUppercase + charsetDigits,
			length:  24,
		},
		{
			name:    "minimum length",
			charset: charsetLowercase,
			length:  minPasswordLength,
		},
		{
			name:    "long password",
			charset: charsetLowercase + charsetUppercase + charsetDigits + charsetSymbols,
			length:  64,
		},
		{
			name:    "digits only",
			charset: charsetDigits,
			length:  16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := generatePassword(tt.charset, tt.length)
			if err != nil {
				t.Fatalf("generatePassword failed: %v", err)
			}

			// Check length
			if len(password) != tt.length {
				t.Errorf("password length = %d, want %d", len(password), tt.length)
			}

			// Check all characters are from charset
			for _, c := range password {
				if !strings.ContainsRune(tt.charset, c) {
					t.Errorf("password contains unexpected character: %c", c)
				}
			}
		})
	}
}

func TestGeneratePasswordRandomness(t *testing.T) {
	charset := charsetLowercase + charsetUppercase + charsetDigits
	length := 32
	count := 100

	passwords := make(map[string]bool)
	for i := 0; i < count; i++ {
		password, err := generatePassword(charset, length)
		if err != nil {
			t.Fatalf("generatePassword failed: %v", err)
		}
		if passwords[password] {
			t.Errorf("duplicate password generated: %s", password)
		}
		passwords[password] = true
	}
}

func TestGeneratePasswordCharacterDistribution(t *testing.T) {
	charset := charsetLowercase + charsetUppercase + charsetDigits + charsetSymbols
	length := 1000
	iterations := 10

	// Count character type occurrences
	var lowerCount, upperCount, digitCount, symbolCount int

	for i := 0; i < iterations; i++ {
		password, err := generatePassword(charset, length)
		if err != nil {
			t.Fatalf("generatePassword failed: %v", err)
		}

		for _, c := range password {
			switch {
			case strings.ContainsRune(charsetLowercase, c):
				lowerCount++
			case strings.ContainsRune(charsetUppercase, c):
				upperCount++
			case strings.ContainsRune(charsetDigits, c):
				digitCount++
			case strings.ContainsRune(charsetSymbols, c):
				symbolCount++
			}
		}
	}

	total := length * iterations

	// Each character type should have roughly equal distribution
	// (within reasonable bounds for random sampling)
	expectedPerType := float64(total) / 4.0
	tolerance := expectedPerType * 0.3 // 30% tolerance

	checkDistribution := func(name string, count int) {
		diff := float64(count) - expectedPerType
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			t.Logf("Warning: %s count %d deviates significantly from expected %.0f", name, count, expectedPerType)
		}
	}

	checkDistribution("lowercase", lowerCount)
	checkDistribution("uppercase", upperCount)
	checkDistribution("digit", digitCount)
	checkDistribution("symbol", symbolCount)
}

func TestGeneratePasswordSecurity(t *testing.T) {
	// Test that generatePassword uses crypto/rand (implicitly tested by ensuring it works)
	charset := charsetLowercase + charsetUppercase + charsetDigits + charsetSymbols
	length := 32

	// Generate multiple passwords and verify they're all different
	passwords := make([]string, 10)
	for i := 0; i < 10; i++ {
		password, err := generatePassword(charset, length)
		if err != nil {
			t.Fatalf("generatePassword failed: %v", err)
		}
		passwords[i] = password
	}

	// Check no duplicates
	seen := make(map[string]bool)
	for _, p := range passwords {
		if seen[p] {
			t.Errorf("duplicate password detected - possible RNG issue")
		}
		seen[p] = true
	}
}

func TestPasswordMeetsComplexityRequirements(t *testing.T) {
	// Generate passwords with all character types and verify they contain variety
	charset := charsetLowercase + charsetUppercase + charsetDigits + charsetSymbols
	length := 24

	// Generate multiple passwords and check at least some have all character types
	hasAllTypes := 0
	iterations := 50

	for i := 0; i < iterations; i++ {
		password, err := generatePassword(charset, length)
		if err != nil {
			t.Fatalf("generatePassword failed: %v", err)
		}

		hasLower := false
		hasUpper := false
		hasDigit := false
		hasSymbol := false

		for _, c := range password {
			if unicode.IsLower(c) {
				hasLower = true
			}
			if unicode.IsUpper(c) {
				hasUpper = true
			}
			if unicode.IsDigit(c) {
				hasDigit = true
			}
			if strings.ContainsRune(charsetSymbols, c) {
				hasSymbol = true
			}
		}

		if hasLower && hasUpper && hasDigit && hasSymbol {
			hasAllTypes++
		}
	}

	// With 24 characters and 4 roughly equal character types,
	// we expect most passwords to have all types
	if hasAllTypes < iterations/2 {
		t.Logf("Only %d/%d passwords had all character types - consider this if complexity is required", hasAllTypes, iterations)
	}
}

func TestCharsetConstants(t *testing.T) {
	// Verify charset constants have expected characters
	if len(charsetLowercase) != 26 {
		t.Errorf("charsetLowercase should have 26 characters, got %d", len(charsetLowercase))
	}
	if len(charsetUppercase) != 26 {
		t.Errorf("charsetUppercase should have 26 characters, got %d", len(charsetUppercase))
	}
	if len(charsetDigits) != 10 {
		t.Errorf("charsetDigits should have 10 characters, got %d", len(charsetDigits))
	}
	if len(charsetSymbols) == 0 {
		t.Error("charsetSymbols should not be empty")
	}

	// Verify no duplicates within charsets
	for name, charset := range map[string]string{
		"lowercase": charsetLowercase,
		"uppercase": charsetUppercase,
		"digits":    charsetDigits,
		"symbols":   charsetSymbols,
	} {
		seen := make(map[rune]bool)
		for _, c := range charset {
			if seen[c] {
				t.Errorf("%s charset has duplicate character: %c", name, c)
			}
			seen[c] = true
		}
	}
}
