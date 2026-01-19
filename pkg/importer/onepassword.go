package importer

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/forest6511/secretctl/pkg/vault"
)

// OnePasswordParser parses 1Password CSV export files.
// Per ADR-006: 1Password CSV format (9 columns):
// Title,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes
type OnePasswordParser struct{}

// 1Password CSV column names (header-based parsing).
const (
	op1ColTitle    = "Title"
	op1ColWebsite  = "Website"
	op1ColUsername = "Username"
	op1ColPassword = "Password"
	op1ColOTPAuth  = "OTPAuth"
	op1ColFavorite = "Favorite"
	op1ColArchived = "Archived"
	op1ColTags     = "Tags"
	op1ColNotes    = "Notes"
)

// Source returns the source type for this parser.
func (p *OnePasswordParser) Source() Source {
	return Source1Password
}

// Parse parses 1Password CSV data.
func (p *OnePasswordParser) Parse(data []byte, opts ParseOptions) (*ImportResult, error) {
	result := &ImportResult{
		Secrets:  make([]*ImportedSecret, 0),
		Warnings: make([]string, 0),
		Skipped:  make([]SkippedItem, 0),
	}

	// Strip UTF-8 BOM if present
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	reader := csv.NewReader(bytes.NewReader(data))
	reader.LazyQuotes = true // Handle malformed exports

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Build column index map (header-based parsing per ADR-006)
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[col] = i
	}

	// Verify we have at least the essential columns
	if _, ok := colIndex[op1ColTitle]; !ok {
		return nil, fmt.Errorf("missing required column: %s", op1ColTitle)
	}

	// Track for key generation fallback
	itemCounter := 1

	// Process rows
	rowNum := 1 // 1-indexed (header is row 0)
	for {
		rowNum++
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("row %d: failed to parse: %v", rowNum, err))
			continue
		}

		// Validate column count
		if len(row) != len(header) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("row %d: column count mismatch (expected %d, got %d)",
					rowNum, len(header), len(row)))
			continue
		}

		secret, warning := p.parseRow(row, colIndex, opts, &itemCounter)
		if warning != "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("row %d: %s", rowNum, warning))
		}
		if secret != nil {
			result.Secrets = append(result.Secrets, secret)
		}
	}

	// Deduplicate keys
	DeduplicateKeys(result.Secrets)

	return result, nil
}

// parseRow parses a single CSV row into an ImportedSecret.
func (p *OnePasswordParser) parseRow(row []string, colIndex map[string]int, opts ParseOptions, itemCounter *int) (*ImportedSecret, string) {
	getValue := func(col string) string {
		if idx, ok := colIndex[col]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	title := getValue(op1ColTitle)
	website := getValue(op1ColWebsite)
	username := getValue(op1ColUsername)
	password := getValue(op1ColPassword)
	otpAuth := getValue(op1ColOTPAuth)
	tagsStr := getValue(op1ColTags)
	notes := getValue(op1ColNotes)

	// Generate key name
	keyName := SanitizeKeyName(title, opts.PreserveCase)
	if keyName == "" {
		// Fallback: use URL hostname or counter
		keyName = SanitizeKeyName(GenerateFallbackKey(website, *itemCounter), opts.PreserveCase)
		*itemCounter++
	}

	// Skip if no useful data
	if username == "" && password == "" && otpAuth == "" && notes == "" {
		return nil, "skipped: no useful data"
	}

	// Build fields
	fields := make(map[string]vault.Field)

	if username != "" {
		fields["username"] = vault.Field{
			Value:     username,
			Sensitive: false,
		}
	}

	if password != "" {
		fields["password"] = vault.Field{
			Value:     password,
			Sensitive: true,
			Kind:      "password",
		}
	}

	if otpAuth != "" {
		fields["totp"] = vault.Field{
			Value:     otpAuth,
			Sensitive: true,
			Hint:      "TOTP seed",
		}
	}

	if notes != "" {
		fields["notes"] = vault.Field{
			Value:     notes,
			Sensitive: true, // Per ADR-006: notes may contain sensitive info
			InputType: "textarea",
		}
	}

	// Build metadata
	var metadata *vault.SecretMetadata
	if website != "" {
		metadata = &vault.SecretMetadata{
			URL: website,
		}
	}

	// Parse tags (comma-separated)
	var tags []string
	if tagsStr != "" {
		rawTags := strings.Split(tagsStr, ",")
		for _, t := range rawTags {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	return &ImportedSecret{
		Key:          keyName,
		OriginalName: title,
		Fields:       fields,
		Tags:         tags,
		Metadata:     metadata,
	}, ""
}
