package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
		wantErr  bool
	}{
		{
			name: "simple key-value pairs",
			input: `API_KEY=secret123
DB_HOST=localhost
DB_PORT=5432`,
			expected: map[string]string{
				"API_KEY": "secret123",
				"DB_HOST": "localhost",
				"DB_PORT": "5432",
			},
		},
		{
			name: "with comments and empty lines",
			input: `# This is a comment
API_KEY=secret123

# Another comment
DB_HOST=localhost`,
			expected: map[string]string{
				"API_KEY": "secret123",
				"DB_HOST": "localhost",
			},
		},
		{
			name: "double quoted values",
			input: `PASSWORD="my secret password"
MESSAGE="hello \"world\""`,
			expected: map[string]string{
				"PASSWORD": "my secret password",
				"MESSAGE":  `hello "world"`,
			},
		},
		{
			name: "single quoted values",
			input: `PASSWORD='my secret password'
COMMAND='echo "hello"'`,
			expected: map[string]string{
				"PASSWORD": "my secret password",
				"COMMAND":  `echo "hello"`,
			},
		},
		{
			name:  "values with equals sign",
			input: `CONNECTION_STRING=host=localhost;port=5432;user=admin`,
			expected: map[string]string{
				"CONNECTION_STRING": "host=localhost;port=5432;user=admin",
			},
		},
		{
			name:  "escape sequences in double quotes",
			input: `MULTILINE="line1\nline2\ttab"`,
			expected: map[string]string{
				"MULTILINE": "line1\nline2\ttab",
			},
		},
		{
			name: "keys with special characters",
			input: `AWS/ACCESS_KEY=AKIAEXAMPLE
DB.CONNECTION.HOST=localhost
SERVICE-API-KEY=secret`,
			expected: map[string]string{
				"AWS/ACCESS_KEY":     "AKIAEXAMPLE",
				"DB.CONNECTION.HOST": "localhost",
				"SERVICE-API-KEY":    "secret",
			},
		},
		{
			name:     "empty file",
			input:    "",
			expected: map[string]string{},
		},
		{
			name: "only comments",
			input: `# comment 1
# comment 2`,
			expected: map[string]string{},
		},
		{
			name: "whitespace around equals",
			input: `KEY1 = value1
KEY2= value2
KEY3 =value3`,
			expected: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
				"KEY3": "value3",
			},
		},
		{
			name: "dollar sign in double quotes",
			input: `PRICE="$100"
ESCAPED="\$HOME"`,
			expected: map[string]string{
				"PRICE":   "$100",
				"ESCAPED": "$HOME",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseEnvFile([]byte(tc.input))

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tc.expected) {
				t.Errorf("got %d keys, want %d keys", len(result), len(tc.expected))
			}

			for key, expectedValue := range tc.expected {
				actualValue, ok := result[key]
				if !ok {
					t.Errorf("missing key: %s", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("key %s: got %q, want %q", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestParseJSONFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
		wantErr  bool
	}{
		{
			name:  "flat key-value object",
			input: `{"API_KEY": "secret123", "DB_HOST": "localhost"}`,
			expected: map[string]string{
				"API_KEY": "secret123",
				"DB_HOST": "localhost",
			},
		},
		{
			name:  "with numbers",
			input: `{"PORT": 5432, "TIMEOUT": 30.5}`,
			expected: map[string]string{
				"PORT":    "5432",
				"TIMEOUT": "30.5",
			},
		},
		{
			name:  "with booleans",
			input: `{"DEBUG": true, "PRODUCTION": false}`,
			expected: map[string]string{
				"DEBUG":      "true",
				"PRODUCTION": "false",
			},
		},
		{
			name:  "with null values (skipped)",
			input: `{"API_KEY": "secret", "OPTIONAL": null}`,
			expected: map[string]string{
				"API_KEY": "secret",
			},
		},
		{
			name:  "with nested objects (skipped)",
			input: `{"API_KEY": "secret", "CONFIG": {"nested": "value"}}`,
			expected: map[string]string{
				"API_KEY": "secret",
			},
		},
		{
			name:  "with arrays (skipped)",
			input: `{"API_KEY": "secret", "HOSTS": ["host1", "host2"]}`,
			expected: map[string]string{
				"API_KEY": "secret",
			},
		},
		{
			name:     "invalid JSON",
			input:    `{invalid json}`,
			wantErr:  true,
			expected: nil,
		},
		{
			name:     "empty object",
			input:    `{}`,
			expected: map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseJSONFile([]byte(tc.input))

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tc.expected) {
				t.Errorf("got %d keys, want %d keys", len(result), len(tc.expected))
			}

			for key, expectedValue := range tc.expected {
				actualValue, ok := result[key]
				if !ok {
					t.Errorf("missing key: %s", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("key %s: got %q, want %q", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestUnquoteEnvValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unquoted value",
			input:    "simple",
			expected: "simple",
		},
		{
			name:     "double quoted",
			input:    `"hello world"`,
			expected: "hello world",
		},
		{
			name:     "single quoted",
			input:    `'hello world'`,
			expected: "hello world",
		},
		{
			name:     "escaped double quote",
			input:    `"say \"hello\""`,
			expected: `say "hello"`,
		},
		{
			name:     "newline escape",
			input:    `"line1\nline2"`,
			expected: "line1\nline2",
		},
		{
			name:     "tab escape",
			input:    `"col1\tcol2"`,
			expected: "col1\tcol2",
		},
		{
			name:     "backslash escape",
			input:    `"C:\\Users\\name"`,
			expected: `C:\Users\name`,
		},
		{
			name:     "dollar escape",
			input:    `"\$HOME"`,
			expected: "$HOME",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single character",
			input:    "a",
			expected: "a",
		},
		{
			name:     "whitespace trimming",
			input:    "  value  ",
			expected: "value",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := unquoteEnvValue(tc.input)
			if result != tc.expected {
				t.Errorf("got %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestDetectImportFormat(t *testing.T) {
	tests := []struct {
		name           string
		filePath       string
		explicitFormat string
		expected       string
		wantErr        bool
	}{
		{
			name:     "json extension",
			filePath: "config.json",
			expected: "json",
		},
		{
			name:     "env extension",
			filePath: ".env",
			expected: "env",
		},
		{
			name:     "env.local",
			filePath: ".env.local",
			expected: "env",
		},
		{
			name:     "env.production",
			filePath: "env.production",
			expected: "env",
		},
		{
			name:     "no extension defaults to env",
			filePath: "secrets",
			expected: "env",
		},
		{
			name:           "explicit format overrides",
			filePath:       "config.json",
			explicitFormat: "env",
			expected:       "env",
		},
		{
			name:           "explicit json format",
			filePath:       ".env",
			explicitFormat: "json",
			expected:       "json",
		},
		{
			name:           "invalid explicit format",
			filePath:       ".env",
			explicitFormat: "yaml",
			wantErr:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			importFormat = tc.explicitFormat
			defer func() { importFormat = "" }()

			result, err := detectImportFormat(tc.filePath)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tc.expected {
				t.Errorf("got %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestValidateImportFlags(t *testing.T) {
	tests := []struct {
		name     string
		conflict string
		wantErr  bool
	}{
		{
			name:     "valid skip",
			conflict: "skip",
			wantErr:  false,
		},
		{
			name:     "valid overwrite",
			conflict: "overwrite",
			wantErr:  false,
		},
		{
			name:     "valid error",
			conflict: "error",
			wantErr:  false,
		},
		{
			name:     "case insensitive",
			conflict: "SKIP",
			wantErr:  false,
		},
		{
			name:     "invalid mode",
			conflict: "ignore",
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			importConflict = tc.conflict
			defer func() { importConflict = conflictSkip }()

			err := validateImportFlags()

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestExpandImportPattern(t *testing.T) {
	availableKeys := []string{
		"AWS_ACCESS_KEY",
		"AWS_SECRET_KEY",
		"DB_HOST",
		"DB_PORT",
		"DB_PASSWORD",
		"API_KEY",
	}

	tests := []struct {
		name     string
		pattern  string
		expected []string
		wantErr  bool
	}{
		{
			name:     "exact match",
			pattern:  "API_KEY",
			expected: []string{"API_KEY"},
		},
		{
			name:     "wildcard prefix",
			pattern:  "AWS_*",
			expected: []string{"AWS_ACCESS_KEY", "AWS_SECRET_KEY"},
		},
		{
			name:     "wildcard suffix",
			pattern:  "*_KEY",
			expected: []string{"AWS_ACCESS_KEY", "AWS_SECRET_KEY", "API_KEY"},
		},
		{
			name:     "question mark wildcard",
			pattern:  "DB_????",
			expected: []string{"DB_HOST", "DB_PORT"},
		},
		{
			name:     "all keys",
			pattern:  "*",
			expected: []string{"AWS_ACCESS_KEY", "AWS_SECRET_KEY", "DB_HOST", "DB_PORT", "DB_PASSWORD", "API_KEY"},
		},
		{
			name:    "no match",
			pattern: "NONEXISTENT_*",
			wantErr: true,
		},
		{
			name:    "exact match not found",
			pattern: "NONEXISTENT",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := expandImportPattern(tc.pattern, availableKeys)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tc.expected) {
				t.Errorf("got %d results, want %d", len(result), len(tc.expected))
			}

			for _, exp := range tc.expected {
				found := false
				for _, r := range result {
					if r == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("missing expected key: %s", exp)
				}
			}
		})
	}
}

func TestFilterImportKeys(t *testing.T) {
	secrets := map[string]string{
		"AWS_ACCESS_KEY": "AKIA...",
		"AWS_SECRET_KEY": "secret",
		"DB_HOST":        "localhost",
		"DB_PORT":        "5432",
		"API_KEY":        "apikey123",
	}

	tests := []struct {
		name     string
		patterns []string
		expected []string
		wantErr  bool
	}{
		{
			name:     "single pattern",
			patterns: []string{"AWS_*"},
			expected: []string{"AWS_ACCESS_KEY", "AWS_SECRET_KEY"},
		},
		{
			name:     "multiple patterns",
			patterns: []string{"AWS_*", "API_KEY"},
			expected: []string{"AWS_ACCESS_KEY", "AWS_SECRET_KEY", "API_KEY"},
		},
		{
			name:     "overlapping patterns",
			patterns: []string{"*_KEY", "API_*"},
			expected: []string{"AWS_ACCESS_KEY", "AWS_SECRET_KEY", "API_KEY"},
		},
		{
			name:     "no match error",
			patterns: []string{"NONEXISTENT"},
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := filterImportKeys(secrets, tc.patterns)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tc.expected) {
				t.Errorf("got %d keys, want %d", len(result), len(tc.expected))
			}

			for _, exp := range tc.expected {
				if _, ok := result[exp]; !ok {
					t.Errorf("missing expected key: %s", exp)
				}
			}
		})
	}
}

func TestParseImportFile(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	t.Run("env file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, ".env")
		content := "API_KEY=secret123\nDB_HOST=localhost"
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		result, err := parseImportFile(filePath, "env")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if len(result) != 2 {
			t.Errorf("got %d keys, want 2", len(result))
		}
	})

	t.Run("json file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "config.json")
		content := `{"API_KEY": "secret123", "DB_HOST": "localhost"}`
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		result, err := parseImportFile(filePath, "json")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if len(result) != 2 {
			t.Errorf("got %d keys, want 2", len(result))
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := parseImportFile(filepath.Join(tempDir, "nonexistent"), "env")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("symlink rejection", func(t *testing.T) {
		realFile := filepath.Join(tempDir, "real.env")
		symlink := filepath.Join(tempDir, "link.env")

		if err := os.WriteFile(realFile, []byte("KEY=value"), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		if err := os.Symlink(realFile, symlink); err != nil {
			t.Skip("cannot create symlinks on this system")
		}

		_, err := parseImportFile(symlink, "env")
		if err == nil {
			t.Error("expected error for symlink")
		}
	})
}

func TestImportCmdFlags(t *testing.T) {
	// Test that all flags are properly registered
	flags := importCmd.Flags()

	tests := []struct {
		name      string
		flagType  string
		shorthand string
	}{
		{"format", "string", "f"},
		{"conflict", "string", ""},
		{"dry-run", "bool", ""},
		{"key", "stringSlice", "k"},
		{"skip", "bool", ""},
		{"overwrite", "bool", ""},
		{"error", "bool", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flag := flags.Lookup(tc.name)
			if flag == nil {
				t.Errorf("flag --%s not found", tc.name)
				return
			}

			if tc.shorthand != "" && flag.Shorthand != tc.shorthand {
				t.Errorf("flag --%s: got shorthand %q, want %q", tc.name, flag.Shorthand, tc.shorthand)
			}
		})
	}
}

func TestEnvLineRegex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantKey string
		wantVal string
		matches bool
	}{
		{
			name:    "simple",
			input:   "KEY=value",
			wantKey: "KEY",
			wantVal: "value",
			matches: true,
		},
		{
			name:    "with underscores",
			input:   "MY_KEY=value",
			wantKey: "MY_KEY",
			wantVal: "value",
			matches: true,
		},
		{
			name:    "with numbers",
			input:   "KEY123=value",
			wantKey: "KEY123",
			wantVal: "value",
			matches: true,
		},
		{
			name:    "with slashes",
			input:   "aws/secret/key=value",
			wantKey: "aws/secret/key",
			wantVal: "value",
			matches: true,
		},
		{
			name:    "with dots",
			input:   "db.connection.host=localhost",
			wantKey: "db.connection.host",
			wantVal: "localhost",
			matches: true,
		},
		{
			name:    "with dashes",
			input:   "api-key=value",
			wantKey: "api-key",
			wantVal: "value",
			matches: true,
		},
		{
			name:    "empty value",
			input:   "KEY=",
			wantKey: "KEY",
			wantVal: "",
			matches: true,
		},
		{
			name:    "starts with number (invalid)",
			input:   "123KEY=value",
			matches: false,
		},
		{
			name:    "no equals sign",
			input:   "INVALID",
			matches: false,
		},
		{
			name:    "comment line",
			input:   "#KEY=value",
			matches: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matches := envLineRegex.FindStringSubmatch(tc.input)

			if tc.matches {
				if matches == nil {
					t.Error("expected match, got nil")
					return
				}
				if matches[1] != tc.wantKey {
					t.Errorf("key: got %q, want %q", matches[1], tc.wantKey)
				}
				if matches[2] != tc.wantVal {
					t.Errorf("value: got %q, want %q", matches[2], tc.wantVal)
				}
			} else if matches != nil {
				t.Errorf("expected no match, got %v", matches)
			}
		})
	}
}
