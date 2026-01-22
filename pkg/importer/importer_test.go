package importer

import (
	"testing"

	"github.com/forest6511/secretctl/pkg/vault"
)

func TestSanitizeKeyName(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		preserveCase bool
		want         string
	}{
		{
			name:         "simple name",
			input:        "MySecret",
			preserveCase: false,
			want:         "mysecret",
		},
		{
			name:         "simple name preserve case",
			input:        "MySecret",
			preserveCase: true,
			want:         "MySecret",
		},
		{
			name:         "spaces to underscores",
			input:        "My Secret Key",
			preserveCase: false,
			want:         "my_secret_key",
		},
		{
			name:         "special characters removed",
			input:        "My@Secret#Key$",
			preserveCase: false,
			want:         "mysecretkey",
		},
		{
			name:         "hyphens preserved",
			input:        "my-secret-key",
			preserveCase: false,
			want:         "my-secret-key",
		},
		{
			name:         "underscores preserved",
			input:        "my_secret_key",
			preserveCase: false,
			want:         "my_secret_key",
		},
		{
			name:         "empty string",
			input:        "",
			preserveCase: false,
			want:         "",
		},
		{
			name:         "only special characters",
			input:        "@#$%",
			preserveCase: false,
			want:         "",
		},
		{
			name:         "unicode normalization",
			input:        "caf\u00e9",
			preserveCase: false,
			want:         "caf",
		},
		{
			name:         "long name truncation",
			input:        "this_is_a_very_long_key_name_that_exceeds_the_maximum_allowed_length_of_128_characters_and_should_be_truncated_to_fit_the_limit_exactly",
			preserveCase: false,
			want:         "this_is_a_very_long_key_name_that_exceeds_the_maximum_allowed_length_of_128_characters_and_should_be_truncated_to_fit_the_limit_",
		},
		{
			name:         "mixed case with preserve",
			input:        "GitHub_API_Key",
			preserveCase: true,
			want:         "GitHub_API_Key",
		},
		{
			name:         "numbers preserved",
			input:        "secret123",
			preserveCase: false,
			want:         "secret123",
		},
		{
			name:         "leading numbers",
			input:        "123secret",
			preserveCase: false,
			want:         "123secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeKeyName(tt.input, tt.preserveCase)
			if got != tt.want {
				t.Errorf("SanitizeKeyName(%q, %v) = %q, want %q", tt.input, tt.preserveCase, got, tt.want)
			}
		})
	}
}

func TestDeduplicateKeys(t *testing.T) {
	tests := []struct {
		name   string
		input  []*ImportedSecret
		wanted []string // expected keys after deduplication
	}{
		{
			name:   "no duplicates",
			input:  []*ImportedSecret{{Key: "key1"}, {Key: "key2"}, {Key: "key3"}},
			wanted: []string{"key1", "key2", "key3"},
		},
		{
			name:   "two duplicates",
			input:  []*ImportedSecret{{Key: "key1"}, {Key: "key1"}, {Key: "key2"}},
			wanted: []string{"key1", "key1_1", "key2"},
		},
		{
			name:   "three duplicates",
			input:  []*ImportedSecret{{Key: "key1"}, {Key: "key1"}, {Key: "key1"}},
			wanted: []string{"key1", "key1_1", "key1_2"},
		},
		{
			name:   "case insensitive duplicates",
			input:  []*ImportedSecret{{Key: "Key1"}, {Key: "key1"}, {Key: "KEY1"}},
			wanted: []string{"Key1", "key1_1", "KEY1_2"},
		},
		{
			name:   "empty slice",
			input:  []*ImportedSecret{},
			wanted: []string{},
		},
		{
			name:   "single item",
			input:  []*ImportedSecret{{Key: "key1"}},
			wanted: []string{"key1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DeduplicateKeys(tt.input)
			for i, s := range tt.input {
				if s.Key != tt.wanted[i] {
					t.Errorf("DeduplicateKeys: index %d got %q, want %q", i, s.Key, tt.wanted[i])
				}
			}
		})
	}
}

func TestGenerateFallbackKey(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		counter int
		want    string
	}{
		{
			name:    "simple URL",
			url:     "https://github.com",
			counter: 1,
			want:    "github.com",
		},
		{
			name:    "URL with path",
			url:     "https://github.com/user/repo",
			counter: 1,
			want:    "github.com",
		},
		{
			name:    "URL with www",
			url:     "https://www.example.com",
			counter: 1,
			want:    "example.com",
		},
		{
			name:    "URL with port",
			url:     "https://example.com:8080/path",
			counter: 1,
			want:    "example.com",
		},
		{
			name:    "HTTP URL",
			url:     "http://example.com",
			counter: 1,
			want:    "example.com",
		},
		{
			name:    "empty URL",
			url:     "",
			counter: 5,
			want:    "imported_item_5",
		},
		{
			name:    "URL with only www",
			url:     "https://www.",
			counter: 3,
			want:    "imported_item_3",
		},
		{
			name:    "subdomain",
			url:     "https://api.github.com",
			counter: 1,
			want:    "api.github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateFallbackKey(tt.url, tt.counter)
			if got != tt.want {
				t.Errorf("GenerateFallbackKey(%q, %d) = %q, want %q", tt.url, tt.counter, got, tt.want)
			}
		})
	}
}

func TestDecodeHTMLEntities(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "ampersand",
			input: "&amp;",
			want:  "&",
		},
		{
			name:  "less than",
			input: "&lt;",
			want:  "<",
		},
		{
			name:  "greater than",
			input: "&gt;",
			want:  ">",
		},
		{
			name:  "double quote",
			input: "&quot;",
			want:  "\"",
		},
		{
			name:  "single quote (numeric)",
			input: "&#39;",
			want:  "'",
		},
		{
			name:  "single quote (named)",
			input: "&apos;",
			want:  "'",
		},
		{
			name:  "mixed content",
			input: "Hello &amp; goodbye &lt;world&gt;",
			want:  "Hello & goodbye <world>",
		},
		{
			name:  "no entities",
			input: "plain text",
			want:  "plain text",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "multiple same entities",
			input: "&amp;&amp;&amp;",
			want:  "&&&",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecodeHTMLEntities(tt.input)
			if got != tt.want {
				t.Errorf("DecodeHTMLEntities(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "leading whitespace",
			input: "  hello",
			want:  "hello",
		},
		{
			name:  "trailing whitespace",
			input: "hello  ",
			want:  "hello",
		},
		{
			name:  "both whitespace",
			input: "  hello  ",
			want:  "hello",
		},
		{
			name:  "no whitespace",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeValue(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsEmptyOrWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "empty string",
			input: "",
			want:  true,
		},
		{
			name:  "only spaces",
			input: "   ",
			want:  true,
		},
		{
			name:  "only tabs",
			input: "\t\t\t",
			want:  true,
		},
		{
			name:  "mixed whitespace",
			input: " \t \n ",
			want:  true,
		},
		{
			name:  "has content",
			input: "hello",
			want:  false,
		},
		{
			name:  "content with whitespace",
			input: "  hello  ",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEmptyOrWhitespace(tt.input)
			if got != tt.want {
				t.Errorf("IsEmptyOrWhitespace(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetParser(t *testing.T) {
	tests := []struct {
		name      string
		source    Source
		wantType  string
		wantError bool
	}{
		{
			name:      "1Password",
			source:    Source1Password,
			wantType:  "*importer.OnePasswordParser",
			wantError: false,
		},
		{
			name:      "Bitwarden",
			source:    SourceBitwarden,
			wantType:  "*importer.BitwardenParser",
			wantError: false,
		},
		{
			name:      "LastPass",
			source:    SourceLastPass,
			wantType:  "*importer.LastPassParser",
			wantError: false,
		},
		{
			name:      "unsupported source",
			source:    Source("unknown"),
			wantType:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := GetParser(tt.source)
			if tt.wantError {
				if err == nil {
					t.Errorf("GetParser(%q) expected error, got nil", tt.source)
				}
			} else {
				if err != nil {
					t.Errorf("GetParser(%q) unexpected error: %v", tt.source, err)
				}
				if parser == nil {
					t.Errorf("GetParser(%q) returned nil parser", tt.source)
				}
			}
		})
	}
}

func TestValidSources(t *testing.T) {
	sources := ValidSources()
	if len(sources) != 3 {
		t.Errorf("ValidSources() returned %d sources, want 3", len(sources))
	}

	expected := map[string]bool{
		"1password": true,
		"bitwarden": true,
		"lastpass":  true,
	}

	for _, s := range sources {
		if !expected[s] {
			t.Errorf("ValidSources() contains unexpected source: %q", s)
		}
	}
}

func TestImportedSecretToSecretEntry(t *testing.T) {
	secret := &ImportedSecret{
		Key:          "test_key",
		OriginalName: "Test Key",
		Fields: map[string]vault.Field{
			"username": {Value: "user", Sensitive: false},
			"password": {Value: "pass", Sensitive: true},
		},
		Tags: []string{"tag1", "tag2"},
	}

	entry := secret.ToSecretEntry()

	if entry.Key != secret.Key {
		t.Errorf("ToSecretEntry().Key = %q, want %q", entry.Key, secret.Key)
	}

	if len(entry.Fields) != len(secret.Fields) {
		t.Errorf("ToSecretEntry().Fields length = %d, want %d", len(entry.Fields), len(secret.Fields))
	}

	if len(entry.Tags) != len(secret.Tags) {
		t.Errorf("ToSecretEntry().Tags length = %d, want %d", len(entry.Tags), len(secret.Tags))
	}
}
