// Package main provides the secretctl CLI commands.
package main

import "github.com/forest6511/secretctl/pkg/vault"

// SecretTemplate defines a template for multi-field secrets.
type SecretTemplate struct {
	Name        string
	Description string
	Fields      []TemplateField
}

// TemplateField defines a field in a secret template.
type TemplateField struct {
	Name      string
	Prompt    string
	Sensitive bool
	Required  bool
	Kind      string
	InputType string // "text" (default) | "textarea" per ADR-005
}

// BuiltinTemplates contains the predefined secret templates.
var BuiltinTemplates = map[string]SecretTemplate{
	"login": {
		Name:        "login",
		Description: "Login credentials (username, password)",
		Fields: []TemplateField{
			{Name: "username", Prompt: "Username", Sensitive: false, Required: true},
			{Name: "password", Prompt: "Password", Sensitive: true, Required: true},
		},
	},
	"database": {
		Name:        "database",
		Description: "Database connection (host, port, username, password, database)",
		Fields: []TemplateField{
			{Name: "host", Prompt: "Host", Sensitive: false, Required: true, Kind: "hostname"},
			{Name: "port", Prompt: "Port", Sensitive: false, Required: false, Kind: "port"},
			{Name: "username", Prompt: "Username", Sensitive: false, Required: true},
			{Name: "password", Prompt: "Password", Sensitive: true, Required: true},
			{Name: "database", Prompt: "Database name", Sensitive: false, Required: false},
		},
	},
	"api": {
		Name:        "api",
		Description: "API credentials (api_key, api_secret, endpoint)",
		Fields: []TemplateField{
			{Name: "api_key", Prompt: "API Key", Sensitive: true, Required: true},
			{Name: "api_secret", Prompt: "API Secret", Sensitive: true, Required: false},
			{Name: "endpoint", Prompt: "Endpoint URL", Sensitive: false, Required: false, Kind: "url"},
		},
	},
	"ssh": {
		Name:        "ssh",
		Description: "SSH connection (host, port, username, private_key)",
		Fields: []TemplateField{
			{Name: "host", Prompt: "Host", Sensitive: false, Required: true, Kind: "hostname"},
			{Name: "port", Prompt: "Port (default: 22)", Sensitive: false, Required: false, Kind: "port"},
			{Name: "username", Prompt: "Username", Sensitive: false, Required: true},
			{Name: "private_key", Prompt: "Private Key (paste, then Ctrl+D)", Sensitive: true, Required: true, InputType: "textarea"},
		},
	},
}

// TemplateToFields converts template fields input to vault.Field map.
func TemplateToFields(template SecretTemplate, values map[string]string) map[string]vault.Field {
	fields := make(map[string]vault.Field)
	for _, tf := range template.Fields {
		value, ok := values[tf.Name]
		if !ok || value == "" {
			if !tf.Required {
				continue
			}
		}
		fields[tf.Name] = vault.Field{
			Value:     value,
			Sensitive: tf.Sensitive,
			Kind:      tf.Kind,
			InputType: tf.InputType,
		}
	}
	return fields
}

// ListTemplates returns the names of all available templates.
func ListTemplates() []string {
	names := make([]string, 0, len(BuiltinTemplates))
	for name := range BuiltinTemplates {
		names = append(names, name)
	}
	return names
}
