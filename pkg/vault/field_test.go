// Package vault provides secure secret storage with AES-256-GCM encryption.
package vault

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateFieldName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// Valid names
		{"valid simple", "password", nil},
		{"valid with underscore", "api_key", nil},
		{"valid with number", "key1", nil},
		{"valid complex", "aws_access_key_id", nil},
		{"valid single char", "a", nil},
		{"valid max length", strings.Repeat("a", MaxFieldNameLength), nil},

		// Invalid: too short
		{"empty string", "", ErrFieldNameTooShort},

		// Invalid: too long
		{"too long", strings.Repeat("a", MaxFieldNameLength+1), ErrFieldNameTooLong},

		// Invalid: not snake_case
		{"uppercase", "Password", ErrFieldNameInvalid},
		{"camelCase", "apiKey", ErrFieldNameInvalid},
		{"starts with number", "1password", ErrFieldNameInvalid},
		{"starts with underscore", "_key", ErrFieldNameInvalid},
		{"contains hyphen", "api-key", ErrFieldNameInvalid},
		{"contains space", "api key", ErrFieldNameInvalid},
		{"contains special", "api@key", ErrFieldNameInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFieldName(tt.input)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateFieldName(%q) = %v, want nil", tt.input, err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateFieldName(%q) = %v, want %v", tt.input, err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidateField(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		field     *Field
		wantErr   error
	}{
		// Valid fields
		{
			name:      "valid simple field",
			fieldName: "password",
			field:     &Field{Value: "secret123", Sensitive: true},
			wantErr:   nil,
		},
		{
			name:      "valid field with aliases",
			fieldName: "password",
			field:     &Field{Value: "secret", Sensitive: true, Aliases: []string{"pwd", "pass"}},
			wantErr:   nil,
		},
		{
			name:      "valid field with hint",
			fieldName: "api_key",
			field:     &Field{Value: "key", Hint: "AWS API key for production"},
			wantErr:   nil,
		},
		{
			name:      "valid field with kind",
			fieldName: "port",
			field:     &Field{Value: "8080", Kind: "port"},
			wantErr:   nil,
		},
		{
			name:      "valid non-sensitive field",
			fieldName: "username",
			field:     &Field{Value: "admin", Sensitive: false},
			wantErr:   nil,
		},

		// Invalid field name
		{
			name:      "invalid field name",
			fieldName: "InvalidName",
			field:     &Field{Value: "test"},
			wantErr:   ErrFieldNameInvalid,
		},

		// Value too large
		{
			name:      "value too large",
			fieldName: "data",
			field:     &Field{Value: strings.Repeat("x", MaxFieldValueSize+1)},
			wantErr:   ErrFieldValueTooLarge,
		},

		// Too many aliases
		{
			name:      "too many aliases",
			fieldName: "password",
			field: &Field{
				Value:   "test",
				Aliases: make([]string, MaxAliasCount+1),
			},
			wantErr: ErrTooManyAliases,
		},

		// Alias too long
		{
			name:      "alias too long",
			fieldName: "password",
			field: &Field{
				Value:   "test",
				Aliases: []string{strings.Repeat("a", MaxAliasLength+1)},
			},
			wantErr: ErrAliasTooLong,
		},

		// Invalid alias format
		{
			name:      "invalid alias format",
			fieldName: "password",
			field: &Field{
				Value:   "test",
				Aliases: []string{"Invalid-Alias"},
			},
			wantErr: ErrAliasInvalid,
		},

		// Hint too long
		{
			name:      "hint too long",
			fieldName: "api_key",
			field: &Field{
				Value: "test",
				Hint:  strings.Repeat("x", MaxHintLength+1),
			},
			wantErr: ErrHintTooLong,
		},

		// Kind too long
		{
			name:      "kind too long",
			fieldName: "data",
			field: &Field{
				Value: "test",
				Kind:  strings.Repeat("x", MaxKindLength+1),
			},
			wantErr: ErrKindTooLong,
		},

		// InputType validation (ADR-005)
		{
			name:      "valid inputType empty",
			fieldName: "private_key",
			field:     &Field{Value: "key", InputType: ""},
			wantErr:   nil,
		},
		{
			name:      "valid inputType text",
			fieldName: "password",
			field:     &Field{Value: "secret", InputType: "text"},
			wantErr:   nil,
		},
		{
			name:      "valid inputType textarea",
			fieldName: "private_key",
			field:     &Field{Value: "-----BEGIN RSA PRIVATE KEY-----", InputType: "textarea"},
			wantErr:   nil,
		},
		{
			name:      "invalid inputType",
			fieldName: "field",
			field:     &Field{Value: "test", InputType: "invalid"},
			wantErr:   ErrInputTypeInvalid,
		},
		{
			name:      "invalid inputType multiline",
			fieldName: "field",
			field:     &Field{Value: "test", InputType: "multiline"},
			wantErr:   ErrInputTypeInvalid,
		},
	}

	// Initialize aliases for "too many aliases" test
	for i := range tests {
		if tests[i].name == "too many aliases" {
			for j := range tests[i].field.Aliases {
				tests[i].field.Aliases[j] = "alias" + string(rune('a'+j%26))
			}
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateField(tt.fieldName, tt.field)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateField(%q) = %v, want nil", tt.fieldName, err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateField(%q) = %v, want %v", tt.fieldName, err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidateFields(t *testing.T) {
	tests := []struct {
		name    string
		fields  map[string]Field
		wantErr error
	}{
		// Valid cases
		{
			name: "single field",
			fields: map[string]Field{
				"password": {Value: "secret", Sensitive: true},
			},
			wantErr: nil,
		},
		{
			name: "multiple fields",
			fields: map[string]Field{
				"username": {Value: "admin", Sensitive: false},
				"password": {Value: "secret", Sensitive: true},
				"api_key":  {Value: "key123", Sensitive: true},
			},
			wantErr: nil,
		},
		{
			name: "fields with non-conflicting aliases",
			fields: map[string]Field{
				"password": {Value: "secret", Aliases: []string{"pwd"}},
				"username": {Value: "admin", Aliases: []string{"user"}},
			},
			wantErr: nil,
		},

		// Too many fields
		{
			name:    "too many fields",
			fields:  makeManyFields(MaxFieldCount + 1),
			wantErr: ErrTooManyFields,
		},

		// Alias conflicts with field name
		{
			name: "alias conflicts with field name",
			fields: map[string]Field{
				"password": {Value: "secret", Aliases: []string{"username"}},
				"username": {Value: "admin"},
			},
			wantErr: ErrAliasConflict,
		},

		// Alias conflicts with another alias
		{
			name: "alias conflicts with another alias",
			fields: map[string]Field{
				"password": {Value: "secret", Aliases: []string{"cred"}},
				"api_key":  {Value: "key", Aliases: []string{"cred"}},
			},
			wantErr: ErrAliasConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFields(tt.fields)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateFields() = %v, want nil", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateFields() = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidateBindings(t *testing.T) {
	fields := map[string]Field{
		"username": {Value: "admin", Sensitive: false},
		"password": {Value: "secret", Sensitive: true, Aliases: []string{"pwd"}},
		"api_key":  {Value: "key123", Sensitive: true},
	}

	tests := []struct {
		name     string
		bindings map[string]string
		fields   map[string]Field
		wantErr  error
	}{
		// Valid cases
		{
			name:     "empty bindings",
			bindings: map[string]string{},
			fields:   fields,
			wantErr:  nil,
		},
		{
			name: "valid bindings",
			bindings: map[string]string{
				"DB_USER": "username",
				"DB_PASS": "password",
			},
			fields:  fields,
			wantErr: nil,
		},
		{
			name: "binding to alias",
			bindings: map[string]string{
				"PASSWORD": "pwd",
			},
			fields:  fields,
			wantErr: nil,
		},

		// Invalid: field not found
		{
			name: "field not found",
			bindings: map[string]string{
				"DB_HOST": "hostname",
			},
			fields:  fields,
			wantErr: ErrBindingFieldNotFound,
		},

		// Invalid: duplicate env var (case-insensitive)
		{
			name: "duplicate env var",
			bindings: map[string]string{
				"DB_PASS": "password",
				"db_pass": "api_key",
			},
			fields:  fields,
			wantErr: ErrBindingConflict,
		},

		// Too many bindings
		{
			name:     "too many bindings",
			bindings: makeManyBindings(MaxBindingCount + 1),
			fields:   makeManyFields(MaxBindingCount + 1),
			wantErr:  ErrTooManyBindings,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBindings(tt.bindings, tt.fields)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateBindings() = %v, want nil", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateBindings() = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestResolveFieldName(t *testing.T) {
	fields := map[string]Field{
		"password": {Value: "secret", Sensitive: true, Aliases: []string{"pwd", "pass"}},
		"username": {Value: "admin", Sensitive: false},
		"api_key":  {Value: "key123", Sensitive: true},
	}

	tests := []struct {
		name          string
		lookupName    string
		wantFieldName string
		wantValue     string
		wantErr       error
	}{
		// Exact match
		{
			name:          "exact match",
			lookupName:    "password",
			wantFieldName: "password",
			wantValue:     "secret",
			wantErr:       nil,
		},
		// Alias match
		{
			name:          "alias match pwd",
			lookupName:    "pwd",
			wantFieldName: "password",
			wantValue:     "secret",
			wantErr:       nil,
		},
		{
			name:          "alias match pass",
			lookupName:    "pass",
			wantFieldName: "password",
			wantValue:     "secret",
			wantErr:       nil,
		},
		// Case-insensitive alias match
		{
			name:          "case-insensitive alias",
			lookupName:    "PWD",
			wantFieldName: "password",
			wantValue:     "secret",
			wantErr:       nil,
		},
		// Case-insensitive field name match
		{
			name:          "case-insensitive field name",
			lookupName:    "PASSWORD",
			wantFieldName: "password",
			wantValue:     "secret",
			wantErr:       nil,
		},
		// Not found
		{
			name:       "not found",
			lookupName: "nonexistent",
			wantErr:    ErrFieldNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotField, err := ResolveFieldName(fields, tt.lookupName)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ResolveFieldName(%q) error = %v, want %v", tt.lookupName, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("ResolveFieldName(%q) unexpected error: %v", tt.lookupName, err)
				return
			}
			if gotName != tt.wantFieldName {
				t.Errorf("ResolveFieldName(%q) name = %q, want %q", tt.lookupName, gotName, tt.wantFieldName)
			}
			if gotField.Value != tt.wantValue {
				t.Errorf("ResolveFieldName(%q) value = %q, want %q", tt.lookupName, gotField.Value, tt.wantValue)
			}
		})
	}
}

func TestNewDefaultField(t *testing.T) {
	value := "test-secret-value"
	field := NewDefaultField(value)

	if field.Value != value {
		t.Errorf("NewDefaultField().Value = %q, want %q", field.Value, value)
	}
	if !field.Sensitive {
		t.Error("NewDefaultField().Sensitive = false, want true")
	}
	if len(field.Aliases) != 0 {
		t.Errorf("NewDefaultField().Aliases = %v, want empty", field.Aliases)
	}
	if field.Kind != "" {
		t.Errorf("NewDefaultField().Kind = %q, want empty", field.Kind)
	}
	if field.InputType != "" {
		t.Errorf("NewDefaultField().InputType = %q, want empty", field.InputType)
	}
	if field.Hint != "" {
		t.Errorf("NewDefaultField().Hint = %q, want empty", field.Hint)
	}
}

func TestConvertSingleValueToFields(t *testing.T) {
	value := []byte("my-secret-value")
	fields := ConvertSingleValueToFields(value)

	if len(fields) != 1 {
		t.Fatalf("ConvertSingleValueToFields() returned %d fields, want 1", len(fields))
	}

	field, ok := fields[DefaultFieldName]
	if !ok {
		t.Fatalf("ConvertSingleValueToFields() missing %q field", DefaultFieldName)
	}

	if field.Value != string(value) {
		t.Errorf("ConvertSingleValueToFields() value = %q, want %q", field.Value, string(value))
	}
	if !field.Sensitive {
		t.Error("ConvertSingleValueToFields() sensitive = false, want true")
	}
}

func TestGetDefaultFieldValue(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]Field
		want   string
	}{
		{
			name: "has default field",
			fields: map[string]Field{
				DefaultFieldName: {Value: "secret"},
			},
			want: "secret",
		},
		{
			name: "no default field",
			fields: map[string]Field{
				"password": {Value: "secret"},
			},
			want: "",
		},
		{
			name:   "empty fields",
			fields: map[string]Field{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDefaultFieldValue(tt.fields)
			if got != tt.want {
				t.Errorf("GetDefaultFieldValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsSingleFieldSecret(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]Field
		want   bool
	}{
		{
			name: "single value field",
			fields: map[string]Field{
				DefaultFieldName: {Value: "secret"},
			},
			want: true,
		},
		{
			name: "multi-field",
			fields: map[string]Field{
				"username": {Value: "admin"},
				"password": {Value: "secret"},
			},
			want: false,
		},
		{
			name: "single non-default field",
			fields: map[string]Field{
				"password": {Value: "secret"},
			},
			want: false,
		},
		{
			name:   "empty fields",
			fields: map[string]Field{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSingleFieldSecret(tt.fields)
			if got != tt.want {
				t.Errorf("IsSingleFieldSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions

func makeManyFields(count int) map[string]Field {
	fields := make(map[string]Field, count)
	for i := 0; i < count; i++ {
		name := "field" + string(rune('a'+i%26)) + string(rune('0'+i/26%10))
		fields[name] = Field{Value: "value"}
	}
	return fields
}

func makeManyBindings(count int) map[string]string {
	bindings := make(map[string]string, count)
	for i := 0; i < count; i++ {
		envVar := "ENV_VAR_" + string(rune('A'+i%26)) + string(rune('0'+i/26%10))
		fieldName := "field" + string(rune('a'+i%26)) + string(rune('0'+i/26%10))
		bindings[envVar] = fieldName
	}
	return bindings
}
