// Package vault provides secure secret storage with AES-256-GCM encryption.
package vault

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Field constants per ADR-002
const (
	MaxFieldNameLength = 64          // Maximum field name length
	MinFieldNameLength = 1           // Minimum field name length
	MaxFieldValueSize  = 1024 * 1024 // 1 MB maximum field value size
	MaxAliasCount      = 10          // Maximum number of aliases per field
	MaxAliasLength     = 64          // Maximum alias length
	MaxHintLength      = 256         // Maximum hint length
	MaxKindLength      = 64          // Maximum kind length
	MaxFieldCount      = 100         // Maximum number of fields per secret
	MaxBindingCount    = 100         // Maximum number of bindings per secret

	// DefaultFieldName is the field name used for backward compatibility
	// with single-value secrets
	DefaultFieldName = "value"
)

// Field validation errors
var (
	ErrFieldNameTooLong     = errors.New("vault: field name too long")
	ErrFieldNameTooShort    = errors.New("vault: field name too short")
	ErrFieldNameInvalid     = errors.New("vault: field name must be snake_case (lowercase letters, numbers, underscores)")
	ErrFieldValueTooLarge   = errors.New("vault: field value too large")
	ErrTooManyFields        = errors.New("vault: too many fields")
	ErrTooManyAliases       = errors.New("vault: too many aliases")
	ErrAliasInvalid         = errors.New("vault: alias must be snake_case")
	ErrAliasTooLong         = errors.New("vault: alias too long")
	ErrHintTooLong          = errors.New("vault: hint too long")
	ErrKindTooLong          = errors.New("vault: kind too long")
	ErrTooManyBindings      = errors.New("vault: too many bindings")
	ErrBindingFieldNotFound = errors.New("vault: binding references non-existent field")
	ErrBindingConflict      = errors.New("vault: multiple bindings map to same environment variable")
	ErrAliasConflict        = errors.New("vault: alias conflicts with another field name or alias")
	ErrFieldNotFound        = errors.New("vault: field not found")
	ErrFieldSensitive       = errors.New("vault: field is marked as sensitive")
	ErrInputTypeInvalid     = errors.New("vault: inputType must be empty, \"text\", or \"textarea\"")
)

// Field represents a single field within a multi-field secret.
// Per ADR-002: Schema-less design with well-known field names.
type Field struct {
	// Value is the actual secret value for this field.
	Value string `json:"value"`

	// Sensitive indicates whether this field contains sensitive data.
	// When true, the field cannot be retrieved via MCP secret_get_field.
	// Default is true for security.
	Sensitive bool `json:"sensitive"`

	// Aliases are alternative names for this field.
	// Used for compatibility (e.g., "pwd" -> "password").
	// Alias resolution is case-insensitive.
	Aliases []string `json:"aliases,omitempty"`

	// Kind is reserved for Phase 3 schema validation.
	// Examples: "password", "url", "port", "hostname"
	Kind string `json:"kind,omitempty"`

	// InputType specifies UI rendering preference for this field.
	// Per ADR-005: Separate from Kind to avoid conflict with Phase 3 schema validation.
	// Valid values: "" (default, treated as "text"), "text", "textarea"
	InputType string `json:"inputType,omitempty"`

	// Hint provides UI/AI description for this field.
	// Not encrypted, visible to AI agents.
	Hint string `json:"hint,omitempty"`
}

// fieldNameRegex validates field names: lowercase letters, numbers, underscores only (snake_case)
var fieldNameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// ValidateFieldName validates a field name per ADR-002 naming rules.
// Field names must be:
// - 1-64 characters
// - snake_case format (lowercase letters, numbers, underscores)
// - Start with a lowercase letter
func ValidateFieldName(name string) error {
	if len(name) < MinFieldNameLength {
		return ErrFieldNameTooShort
	}
	if len(name) > MaxFieldNameLength {
		return ErrFieldNameTooLong
	}
	if !fieldNameRegex.MatchString(name) {
		return fmt.Errorf("%w: got %q", ErrFieldNameInvalid, name)
	}
	return nil
}

// ValidateField validates a single field's properties.
func ValidateField(name string, field *Field) error {
	// Validate field name
	if err := ValidateFieldName(name); err != nil {
		return err
	}

	// Validate value size
	if len(field.Value) > MaxFieldValueSize {
		return fmt.Errorf("%w: field %q exceeds %d bytes", ErrFieldValueTooLarge, name, MaxFieldValueSize)
	}

	// Validate aliases
	if len(field.Aliases) > MaxAliasCount {
		return fmt.Errorf("%w: field %q has %d aliases (max %d)", ErrTooManyAliases, name, len(field.Aliases), MaxAliasCount)
	}
	for _, alias := range field.Aliases {
		if len(alias) > MaxAliasLength {
			return fmt.Errorf("%w: %q", ErrAliasTooLong, alias)
		}
		// Aliases follow the same naming rules as field names
		if !fieldNameRegex.MatchString(strings.ToLower(alias)) {
			return fmt.Errorf("%w: %q", ErrAliasInvalid, alias)
		}
	}

	// Validate hint
	if len(field.Hint) > MaxHintLength {
		return fmt.Errorf("%w: field %q hint exceeds %d characters", ErrHintTooLong, name, MaxHintLength)
	}

	// Validate kind (Phase 3 reserved, but validate length)
	if len(field.Kind) > MaxKindLength {
		return fmt.Errorf("%w: field %q kind exceeds %d characters", ErrKindTooLong, name, MaxKindLength)
	}

	// Validate inputType per ADR-005: must be empty, "text", or "textarea"
	if field.InputType != "" && field.InputType != "text" && field.InputType != "textarea" {
		return fmt.Errorf("%w: field %q has invalid inputType %q", ErrInputTypeInvalid, name, field.InputType)
	}

	return nil
}

// ValidateFields validates all fields in a multi-field secret.
// Checks for:
// - Maximum field count
// - Individual field validation
// - Alias conflicts (no alias should match another field name or alias)
func ValidateFields(fields map[string]Field) error {
	if len(fields) > MaxFieldCount {
		return fmt.Errorf("%w: has %d fields (max %d)", ErrTooManyFields, len(fields), MaxFieldCount)
	}

	// Build a set of all names and aliases to detect conflicts
	allNames := make(map[string]string) // lowercase name/alias -> original field name

	for name, field := range fields {
		// Validate individual field
		if err := ValidateField(name, &field); err != nil {
			return err
		}

		// Check field name doesn't conflict
		lowerName := strings.ToLower(name)
		if existing, ok := allNames[lowerName]; ok {
			return fmt.Errorf("%w: %q conflicts with field %q", ErrAliasConflict, name, existing)
		}
		allNames[lowerName] = name

		// Check aliases don't conflict
		for _, alias := range field.Aliases {
			lowerAlias := strings.ToLower(alias)
			if existing, ok := allNames[lowerAlias]; ok {
				return fmt.Errorf("%w: alias %q in field %q conflicts with %q", ErrAliasConflict, alias, name, existing)
			}
			allNames[lowerAlias] = name
		}
	}

	return nil
}

// ValidateBindings validates environment variable bindings.
// Checks for:
// - Maximum binding count
// - All referenced fields exist
// - No duplicate environment variable names
func ValidateBindings(bindings map[string]string, fields map[string]Field) error {
	if len(bindings) > MaxBindingCount {
		return fmt.Errorf("%w: has %d bindings (max %d)", ErrTooManyBindings, len(bindings), MaxBindingCount)
	}

	// Build lowercase field name set for lookup (including aliases)
	fieldLookup := make(map[string]bool)
	for name, field := range fields {
		fieldLookup[strings.ToLower(name)] = true
		for _, alias := range field.Aliases {
			fieldLookup[strings.ToLower(alias)] = true
		}
	}

	// Track environment variable names to detect conflicts
	envVars := make(map[string]string) // uppercase env var -> field name

	for envVar, fieldName := range bindings {
		// Check field exists
		if !fieldLookup[strings.ToLower(fieldName)] {
			return fmt.Errorf("%w: %q references field %q", ErrBindingFieldNotFound, envVar, fieldName)
		}

		// Check for environment variable conflicts (case-insensitive)
		upperEnv := strings.ToUpper(envVar)
		if existing, ok := envVars[upperEnv]; ok {
			return fmt.Errorf("%w: %q and %q both map to %q", ErrBindingConflict, existing, fieldName, upperEnv)
		}
		envVars[upperEnv] = fieldName
	}

	return nil
}

// ResolveFieldName resolves a field name, checking aliases if the exact name is not found.
// Resolution is case-insensitive for aliases.
// Returns the canonical field name and a copy of the Field, or an error if not found.
// NOTE: The returned Field is a copy. Modifications to it will NOT affect the original map.
func ResolveFieldName(fields map[string]Field, name string) (string, *Field, error) {
	// Try exact match first
	if field, ok := fields[name]; ok {
		return name, &field, nil
	}

	// Try case-insensitive match on field names
	for fieldName, field := range fields {
		if strings.EqualFold(fieldName, name) {
			return fieldName, &field, nil
		}

		// Check aliases (case-insensitive)
		for _, alias := range field.Aliases {
			if strings.EqualFold(alias, name) {
				return fieldName, &field, nil
			}
		}
	}

	return "", nil, ErrFieldNotFound
}

// NewDefaultField creates a new Field with default settings (sensitive=true).
func NewDefaultField(value string) Field {
	return Field{
		Value:     value,
		Sensitive: true,
	}
}

// ConvertSingleValueToFields converts a single value to the multi-field format.
// This is used for backward compatibility with legacy single-value secrets.
func ConvertSingleValueToFields(value []byte) map[string]Field {
	return map[string]Field{
		DefaultFieldName: {
			Value:     string(value),
			Sensitive: true,
		},
	}
}

// GetDefaultFieldValue extracts the default "value" field from a Fields map.
// Returns empty string if not found. Used for backward compatibility.
func GetDefaultFieldValue(fields map[string]Field) string {
	if field, ok := fields[DefaultFieldName]; ok {
		return field.Value
	}
	return ""
}

// IsSingleFieldSecret returns true if the secret has only the default "value" field.
// Used to detect legacy-format secrets for backward-compatible display.
func IsSingleFieldSecret(fields map[string]Field) bool {
	if len(fields) != 1 {
		return false
	}
	_, ok := fields[DefaultFieldName]
	return ok
}
