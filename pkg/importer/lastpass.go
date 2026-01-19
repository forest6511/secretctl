package importer

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/forest6511/secretctl/pkg/vault"
)

// LastPassParser parses LastPass CSV export files.
// Per ADR-006: LastPass CSV format:
// url,username,password,totp,extra,name,grouping,fav
type LastPassParser struct{}

// LastPass CSV column names (header-based parsing).
const (
	lpColURL      = "url"
	lpColUsername = "username"
	lpColPassword = "password"
	lpColTOTP     = "totp"
	lpColExtra    = "extra"
	lpColName     = "name"
	lpColGrouping = "grouping"
	lpColFav      = "fav"
)

// Source returns the source type for this parser.
func (p *LastPassParser) Source() Source {
	return SourceLastPass
}

// Parse parses LastPass CSV data.
func (p *LastPassParser) Parse(data []byte, opts ParseOptions) (*ImportResult, error) {
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
		// LastPass uses lowercase column names
		colIndex[strings.ToLower(col)] = i
	}

	// Verify we have at least the essential columns
	hasNameCol := false
	if _, ok := colIndex[lpColName]; ok {
		hasNameCol = true
	}
	if !hasNameCol {
		return nil, fmt.Errorf("missing required column: %s", lpColName)
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
func (p *LastPassParser) parseRow(row []string, colIndex map[string]int, opts ParseOptions, itemCounter *int) (*ImportedSecret, string) {
	getValue := func(col string) string {
		if idx, ok := colIndex[col]; ok && idx < len(row) {
			// Per ADR-006: Decode HTML entities
			return DecodeHTMLEntities(strings.TrimSpace(row[idx]))
		}
		return ""
	}

	name := getValue(lpColName)
	url := getValue(lpColURL)
	username := getValue(lpColUsername)
	password := getValue(lpColPassword)
	totp := getValue(lpColTOTP)
	extra := getValue(lpColExtra)
	grouping := getValue(lpColGrouping)

	// Generate key name
	keyName := SanitizeKeyName(name, opts.PreserveCase)
	if keyName == "" {
		// Fallback: use URL hostname or counter
		keyName = SanitizeKeyName(GenerateFallbackKey(url, *itemCounter), opts.PreserveCase)
		*itemCounter++
	}

	// Skip if no useful data
	if username == "" && password == "" && totp == "" && extra == "" {
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

	// Per ADR-006: LastPass may not export TOTP in some versions
	if totp != "" {
		fields["totp"] = vault.Field{
			Value:     totp,
			Sensitive: true,
			Hint:      "TOTP seed",
		}
	}

	// Per ADR-006: extra field often contains sensitive info
	if extra != "" {
		fields["notes"] = vault.Field{
			Value:     extra,
			Sensitive: true,
			InputType: "textarea",
		}
	}

	if url != "" && url != "http://sn" { // LastPass uses "http://sn" for Secure Notes
		fields["url"] = vault.Field{
			Value:     url,
			Sensitive: false,
		}
	}

	// Build metadata
	var metadata *vault.SecretMetadata
	if url != "" && url != "http://sn" {
		metadata = &vault.SecretMetadata{
			URL: url,
		}
	}

	// Parse tags (grouping - nested groups preserved as-is per ADR-006)
	var tags []string
	if grouping != "" {
		tags = append(tags, grouping)
	}

	return &ImportedSecret{
		Key:          keyName,
		OriginalName: name,
		Fields:       fields,
		Tags:         tags,
		Metadata:     metadata,
	}, ""
}
