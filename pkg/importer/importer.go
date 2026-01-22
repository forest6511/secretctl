// Package importer provides parsers for importing secrets from competitor password managers.
// Supports 1Password CSV, Bitwarden JSON, and LastPass CSV formats.
//
// Per ADR-006: Competitor Import Design
package importer

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/forest6511/secretctl/pkg/vault"
)

// Source represents the source password manager format.
type Source string

const (
	Source1Password Source = "1password"
	SourceBitwarden Source = "bitwarden"
	SourceLastPass  Source = "lastpass"
)

// MaxKeyLength is the maximum allowed key length (from vault package).
const MaxKeyLength = 128

// ImportedSecret represents a secret imported from a competitor format.
// It maps to vault.SecretEntry but with additional metadata for import processing.
type ImportedSecret struct {
	// Key is the sanitized secret key name.
	Key string

	// OriginalName is the original name before sanitization.
	OriginalName string

	// Fields contains the multi-field values for this secret.
	Fields map[string]vault.Field

	// Tags are the tags/folders from the source.
	Tags []string

	// Metadata contains notes and URL.
	Metadata *vault.SecretMetadata
}

// ImportResult contains the results of an import operation.
type ImportResult struct {
	// Secrets are the successfully parsed secrets.
	Secrets []*ImportedSecret

	// Warnings are non-fatal issues encountered during parsing.
	Warnings []string

	// Skipped are items that were skipped with reasons.
	Skipped []SkippedItem
}

// SkippedItem represents an item that was skipped during import.
type SkippedItem struct {
	OriginalName string
	Reason       string
}

// Parser is the interface for competitor format parsers.
type Parser interface {
	// Parse parses the input data and returns imported secrets.
	Parse(data []byte, opts ParseOptions) (*ImportResult, error)

	// Source returns the source type for this parser.
	Source() Source
}

// ParseOptions contains options for parsing.
type ParseOptions struct {
	// PreserveCase prevents lowercasing of key names.
	PreserveCase bool
}

// keyNameRegex matches valid key characters (alphanumeric, underscore, hyphen).
var keyNameRegex = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// SanitizeKeyName sanitizes a name to be a valid secret key.
// Per ADR-006:
// 1. Replace spaces with underscores
// 2. Remove invalid characters (keep alphanumeric, _, -)
// 3. Truncate to MaxKeyLength (128)
// 4. Convert to lowercase (unless preserveCase is true)
func SanitizeKeyName(name string, preserveCase bool) string {
	if name == "" {
		return ""
	}

	// Normalize Unicode (NFC)
	name = norm.NFC.String(name)

	// Replace spaces with underscores
	name = strings.ReplaceAll(name, " ", "_")

	// Remove invalid characters
	name = keyNameRegex.ReplaceAllString(name, "")

	// Truncate to max length
	if len(name) > MaxKeyLength {
		name = name[:MaxKeyLength]
	}

	// Convert to lowercase unless preserveCase is true
	if !preserveCase {
		name = strings.ToLower(name)
	}

	return name
}

// DeduplicateKeys ensures all keys are unique by appending suffixes (_1, _2, etc.).
func DeduplicateKeys(secrets []*ImportedSecret) {
	seen := make(map[string]int)

	for _, s := range secrets {
		baseKey := s.Key
		count := seen[strings.ToLower(baseKey)]

		if count > 0 {
			// Append suffix
			s.Key = fmt.Sprintf("%s_%d", baseKey, count)
		}

		seen[strings.ToLower(baseKey)] = count + 1
	}
}

// GenerateFallbackKey generates a fallback key when the original name is empty.
// Per ADR-006:
// 1. Use first non-empty URL hostname as fallback
// 2. If no URL, use imported_item_N
func GenerateFallbackKey(url string, counter int) string {
	if url != "" {
		// Extract hostname from URL
		hostname := extractHostname(url)
		if hostname != "" {
			return hostname
		}
	}
	return fmt.Sprintf("imported_item_%d", counter)
}

// extractHostname extracts the hostname from a URL.
func extractHostname(urlStr string) string {
	// Simple hostname extraction without full URL parsing
	// Remove protocol
	urlStr = strings.TrimPrefix(urlStr, "https://")
	urlStr = strings.TrimPrefix(urlStr, "http://")

	// Remove path
	if idx := strings.Index(urlStr, "/"); idx != -1 {
		urlStr = urlStr[:idx]
	}

	// Remove port
	if idx := strings.Index(urlStr, ":"); idx != -1 {
		urlStr = urlStr[:idx]
	}

	// Remove www. prefix
	urlStr = strings.TrimPrefix(urlStr, "www.")

	return urlStr
}

// DecodeHTMLEntities decodes common HTML entities found in LastPass exports.
// Per ADR-006: LastPass may HTML-encode special characters.
func DecodeHTMLEntities(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&apos;", "'")
	return s
}

// NormalizeValue normalizes a value for comparison (e.g., in duplicate detection).
// Trims whitespace and normalizes Unicode.
func NormalizeValue(s string) string {
	s = strings.TrimSpace(s)
	s = norm.NFC.String(s)
	return s
}

// IsEmptyOrWhitespace checks if a string is empty or contains only whitespace.
func IsEmptyOrWhitespace(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// ToSecretEntry converts an ImportedSecret to a vault.SecretEntry.
func (s *ImportedSecret) ToSecretEntry() *vault.SecretEntry {
	return &vault.SecretEntry{
		Key:      s.Key,
		Fields:   s.Fields,
		Metadata: s.Metadata,
		Tags:     s.Tags,
	}
}

// GetParser returns a parser for the given source.
func GetParser(source Source) (Parser, error) {
	switch source {
	case Source1Password:
		return &OnePasswordParser{}, nil
	case SourceBitwarden:
		return &BitwardenParser{}, nil
	case SourceLastPass:
		return &LastPassParser{}, nil
	default:
		return nil, fmt.Errorf("unsupported import source: %s", source)
	}
}

// ValidSources returns a list of valid source names.
func ValidSources() []string {
	return []string{
		string(Source1Password),
		string(SourceBitwarden),
		string(SourceLastPass),
	}
}
