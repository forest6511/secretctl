package importer

import (
	"encoding/json"
	"fmt"

	"github.com/forest6511/secretctl/pkg/vault"
)

// BitwardenParser parses Bitwarden JSON export files.
// Per ADR-006: Bitwarden JSON format with type codes 1-4.
type BitwardenParser struct{}

// Bitwarden item types.
const (
	bitwardenTypeLogin      = 1
	bitwardenTypeSecureNote = 2
	bitwardenTypeCard       = 3
	bitwardenTypeIdentity   = 4
)

// Bitwarden custom field types.
const (
	bitwardenFieldText    = 0
	bitwardenFieldHidden  = 1
	bitwardenFieldBoolean = 2
)

// bitwardenExport represents the top-level Bitwarden export structure.
type bitwardenExport struct {
	Items   []bitwardenItem   `json:"items"`
	Folders []bitwardenFolder `json:"folders"`
}

// bitwardenFolder represents a Bitwarden folder.
type bitwardenFolder struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// bitwardenItem represents a Bitwarden vault item.
type bitwardenItem struct {
	Type          int                    `json:"type"`
	Name          string                 `json:"name"`
	Notes         string                 `json:"notes"`
	FolderID      *string                `json:"folderId"`
	CollectionIDs []string               `json:"collectionIds"`
	Login         *bitwardenLogin        `json:"login"`
	Card          *bitwardenCard         `json:"card"`
	Identity      *bitwardenIdentity     `json:"identity"`
	Fields        []bitwardenCustomField `json:"fields"`
}

// bitwardenLogin represents Bitwarden login data.
type bitwardenLogin struct {
	URIs     []bitwardenURI `json:"uris"`
	Username string         `json:"username"`
	Password string         `json:"password"`
	TOTP     string         `json:"totp"`
}

// bitwardenURI represents a Bitwarden URI entry.
type bitwardenURI struct {
	URI string `json:"uri"`
}

// bitwardenCard represents Bitwarden card data.
type bitwardenCard struct {
	CardholderName string `json:"cardholderName"`
	Number         string `json:"number"`
	ExpMonth       string `json:"expMonth"`
	ExpYear        string `json:"expYear"`
	Code           string `json:"code"`
	Brand          string `json:"brand"`
}

// bitwardenIdentity represents Bitwarden identity data.
type bitwardenIdentity struct {
	Title          string `json:"title"`
	FirstName      string `json:"firstName"`
	MiddleName     string `json:"middleName"`
	LastName       string `json:"lastName"`
	Username       string `json:"username"`
	Company        string `json:"company"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	Address1       string `json:"address1"`
	Address2       string `json:"address2"`
	Address3       string `json:"address3"`
	City           string `json:"city"`
	State          string `json:"state"`
	PostalCode     string `json:"postalCode"`
	Country        string `json:"country"`
	SSN            string `json:"ssn"`
	PassportNumber string `json:"passportNumber"`
	LicenseNumber  string `json:"licenseNumber"`
}

// bitwardenCustomField represents a Bitwarden custom field.
type bitwardenCustomField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  int    `json:"type"`
}

// Source returns the source type for this parser.
func (p *BitwardenParser) Source() Source {
	return SourceBitwarden
}

// Parse parses Bitwarden JSON data.
func (p *BitwardenParser) Parse(data []byte, opts ParseOptions) (*ImportResult, error) {
	result := &ImportResult{
		Secrets:  make([]*ImportedSecret, 0),
		Warnings: make([]string, 0),
		Skipped:  make([]SkippedItem, 0),
	}

	var export bitwardenExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("failed to parse Bitwarden JSON: %w", err)
	}

	// Build folder lookup map
	folderMap := make(map[string]string)
	for _, f := range export.Folders {
		folderMap[f.ID] = f.Name
	}

	// Track for key generation fallback
	itemCounter := 1

	// Process items
	for i := range export.Items {
		item := &export.Items[i]
		secret, warning := p.parseItem(item, folderMap, opts, &itemCounter)
		if warning != "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("item %d (%s): %s", i+1, item.Name, warning))
		}
		if secret != nil {
			result.Secrets = append(result.Secrets, secret)
		} else if warning == "" {
			result.Skipped = append(result.Skipped, SkippedItem{
				OriginalName: item.Name,
				Reason:       "no useful data",
			})
		}
	}

	// Deduplicate keys
	DeduplicateKeys(result.Secrets)

	return result, nil
}

// parseItem parses a single Bitwarden item.
func (p *BitwardenParser) parseItem(item *bitwardenItem, folderMap map[string]string, opts ParseOptions, itemCounter *int) (*ImportedSecret, string) {
	var fields map[string]vault.Field
	var metadata *vault.SecretMetadata
	var warning string

	switch item.Type {
	case bitwardenTypeLogin:
		fields, metadata, warning = p.parseLogin(item)
	case bitwardenTypeSecureNote:
		fields, metadata, warning = p.parseSecureNote(item)
	case bitwardenTypeCard:
		fields, metadata, warning = p.parseCard(item)
	case bitwardenTypeIdentity:
		fields, metadata, warning = p.parseIdentity(item)
	default:
		return nil, fmt.Sprintf("unsupported item type: %d", item.Type)
	}

	if len(fields) == 0 {
		return nil, warning
	}

	// Add custom fields
	for _, cf := range item.Fields {
		fieldName := SanitizeKeyName(cf.Name, opts.PreserveCase)
		if fieldName == "" {
			fieldName = "custom_field"
		}

		// Per ADR-006: Custom field type mapping
		sensitive := false
		switch cf.Type {
		case bitwardenFieldHidden:
			sensitive = true
		case bitwardenFieldText, bitwardenFieldBoolean:
			sensitive = false
		}

		fields[fieldName] = vault.Field{
			Value:     cf.Value,
			Sensitive: sensitive,
		}
	}

	// Generate key name
	keyName := SanitizeKeyName(item.Name, opts.PreserveCase)
	if keyName == "" {
		// Fallback: use URL or counter
		var url string
		if metadata != nil && metadata.URL != "" {
			url = metadata.URL
		}
		keyName = SanitizeKeyName(GenerateFallbackKey(url, *itemCounter), opts.PreserveCase)
		*itemCounter++
	}

	// Build tags from folder
	var tags []string
	if item.FolderID != nil {
		if folderName, ok := folderMap[*item.FolderID]; ok && folderName != "" {
			tags = append(tags, folderName)
		}
	}
	// Also include collection IDs as tags (for org exports)
	for _, collID := range item.CollectionIDs {
		if collName, ok := folderMap[collID]; ok && collName != "" {
			tags = append(tags, collName)
		}
	}

	return &ImportedSecret{
		Key:          keyName,
		OriginalName: item.Name,
		Fields:       fields,
		Tags:         tags,
		Metadata:     metadata,
	}, warning
}

// parseLogin parses a Login type item.
func (p *BitwardenParser) parseLogin(item *bitwardenItem) (map[string]vault.Field, *vault.SecretMetadata, string) {
	fields := make(map[string]vault.Field)
	var metadata *vault.SecretMetadata

	if item.Login == nil {
		return fields, metadata, ""
	}

	login := item.Login

	if login.Username != "" {
		fields["username"] = vault.Field{
			Value:     login.Username,
			Sensitive: false,
		}
	}

	if login.Password != "" {
		fields["password"] = vault.Field{
			Value:     login.Password,
			Sensitive: true,
			Kind:      "password",
		}
	}

	if login.TOTP != "" {
		fields["totp"] = vault.Field{
			Value:     login.TOTP,
			Sensitive: true,
			Hint:      "TOTP seed (use with authenticator app)",
		}
	}

	// Per ADR-006: notes are sensitive
	if item.Notes != "" {
		fields["notes"] = vault.Field{
			Value:     item.Notes,
			Sensitive: true,
			InputType: "textarea",
		}
	}

	// Handle multiple URIs
	if len(login.URIs) > 0 {
		// Primary URL goes to metadata and url field
		primaryURL := login.URIs[0].URI
		if primaryURL != "" {
			metadata = &vault.SecretMetadata{URL: primaryURL}
			fields["url"] = vault.Field{
				Value:     primaryURL,
				Sensitive: false,
			}
		}

		// Additional URIs
		for i := 1; i < len(login.URIs); i++ {
			if login.URIs[i].URI != "" {
				fields[fmt.Sprintf("url_%d", i+1)] = vault.Field{
					Value:     login.URIs[i].URI,
					Sensitive: false,
				}
			}
		}
	}

	return fields, metadata, ""
}

// parseSecureNote parses a Secure Note type item.
func (p *BitwardenParser) parseSecureNote(item *bitwardenItem) (map[string]vault.Field, *vault.SecretMetadata, string) {
	fields := make(map[string]vault.Field)

	// Per ADR-006: notes are sensitive
	if item.Notes != "" {
		fields["notes"] = vault.Field{
			Value:     item.Notes,
			Sensitive: true,
			InputType: "textarea",
		}
	}

	return fields, nil, ""
}

// parseCard parses a Card type item.
func (p *BitwardenParser) parseCard(item *bitwardenItem) (map[string]vault.Field, *vault.SecretMetadata, string) {
	fields := make(map[string]vault.Field)

	if item.Card == nil {
		return fields, nil, ""
	}

	card := item.Card

	if card.CardholderName != "" {
		fields["cardholder_name"] = vault.Field{
			Value:     card.CardholderName,
			Sensitive: false,
		}
	}

	if card.Number != "" {
		fields["number"] = vault.Field{
			Value:     card.Number,
			Sensitive: true,
		}
	}

	if card.ExpMonth != "" {
		fields["exp_month"] = vault.Field{
			Value:     card.ExpMonth,
			Sensitive: false,
		}
	}

	if card.ExpYear != "" {
		fields["exp_year"] = vault.Field{
			Value:     card.ExpYear,
			Sensitive: false,
		}
	}

	if card.Code != "" {
		fields["cvv"] = vault.Field{
			Value:     card.Code,
			Sensitive: true,
		}
	}

	if card.Brand != "" {
		fields["brand"] = vault.Field{
			Value:     card.Brand,
			Sensitive: false,
		}
	}

	// Per ADR-006: notes are sensitive
	if item.Notes != "" {
		fields["notes"] = vault.Field{
			Value:     item.Notes,
			Sensitive: true,
			InputType: "textarea",
		}
	}

	return fields, nil, ""
}

// addIdentityField adds a field to the map if the value is non-empty.
func addIdentityField(fields map[string]vault.Field, name, value string, sensitive bool) {
	if value != "" {
		fields[name] = vault.Field{
			Value:     value,
			Sensitive: sensitive,
		}
	}
}

// parseIdentity parses an Identity type item.
// Per ADR-006: PII fields are marked sensitive.
func (p *BitwardenParser) parseIdentity(item *bitwardenItem) (map[string]vault.Field, *vault.SecretMetadata, string) {
	fields := make(map[string]vault.Field)

	if item.Identity == nil {
		return fields, nil, ""
	}

	id := item.Identity

	// Non-sensitive fields
	addIdentityField(fields, "title", id.Title, false)
	addIdentityField(fields, "username", id.Username, false)
	addIdentityField(fields, "company", id.Company, false)
	addIdentityField(fields, "state", id.State, false)
	addIdentityField(fields, "country", id.Country, false)

	// Sensitive PII fields (per ADR-006)
	addIdentityField(fields, "first_name", id.FirstName, true)
	addIdentityField(fields, "middle_name", id.MiddleName, true)
	addIdentityField(fields, "last_name", id.LastName, true)
	addIdentityField(fields, "email", id.Email, true)
	addIdentityField(fields, "phone", id.Phone, true)
	addIdentityField(fields, "address1", id.Address1, true)
	addIdentityField(fields, "address2", id.Address2, true)
	addIdentityField(fields, "address3", id.Address3, true)
	addIdentityField(fields, "city", id.City, true)
	addIdentityField(fields, "postal_code", id.PostalCode, true)
	addIdentityField(fields, "ssn", id.SSN, true)
	addIdentityField(fields, "passport", id.PassportNumber, true)
	addIdentityField(fields, "license", id.LicenseNumber, true)

	// Per ADR-006: notes are sensitive
	if item.Notes != "" {
		fields["notes"] = vault.Field{
			Value:     item.Notes,
			Sensitive: true,
			InputType: "textarea",
		}
	}

	return fields, nil, ""
}
