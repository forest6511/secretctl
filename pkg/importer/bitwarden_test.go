package importer

import (
	"encoding/json"
	"testing"
)

func TestBitwardenParser_Source(t *testing.T) {
	p := &BitwardenParser{}
	if p.Source() != SourceBitwarden {
		t.Errorf("Source() = %q, want %q", p.Source(), SourceBitwarden)
	}
}

func TestBitwardenParser_ParseLogin(t *testing.T) {
	jsonData := `{
		"items": [{
			"type": 1,
			"name": "GitHub Login",
			"notes": "My GitHub account",
			"folderId": "folder1",
			"login": {
				"uris": [{"uri": "https://github.com"}],
				"username": "johndoe",
				"password": "mysecretpass123",
				"totp": "JBSWY3DPEHPK3PXP"
			}
		}],
		"folders": [{"id": "folder1", "name": "Work"}]
	}`

	p := &BitwardenParser{}
	result, err := p.Parse([]byte(jsonData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 1 {
		t.Fatalf("Secrets count = %d, want 1", len(result.Secrets))
	}

	s := result.Secrets[0]

	if s.Key != "github_login" {
		t.Errorf("Key = %q, want %q", s.Key, "github_login")
	}

	if s.Fields["username"].Value != "johndoe" {
		t.Errorf("username = %q, want %q", s.Fields["username"].Value, "johndoe")
	}

	if s.Fields["password"].Value != "mysecretpass123" {
		t.Errorf("password = %q, want %q", s.Fields["password"].Value, "mysecretpass123")
	}

	if !s.Fields["password"].Sensitive {
		t.Error("password should be sensitive")
	}

	if s.Fields["totp"].Value != "JBSWY3DPEHPK3PXP" {
		t.Errorf("totp = %q, want %q", s.Fields["totp"].Value, "JBSWY3DPEHPK3PXP")
	}

	if !s.Fields["notes"].Sensitive {
		t.Error("notes should be sensitive")
	}

	if len(s.Tags) != 1 || s.Tags[0] != "Work" {
		t.Errorf("Tags = %v, want [Work]", s.Tags)
	}

	if s.Metadata == nil || s.Metadata.URL != "https://github.com" {
		t.Errorf("Metadata.URL = %v, want https://github.com", s.Metadata)
	}
}

func TestBitwardenParser_ParseSecureNote(t *testing.T) {
	jsonData := `{
		"items": [{
			"type": 2,
			"name": "Secret Note",
			"notes": "This is a very secret note"
		}],
		"folders": []
	}`

	p := &BitwardenParser{}
	result, err := p.Parse([]byte(jsonData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 1 {
		t.Fatalf("Secrets count = %d, want 1", len(result.Secrets))
	}

	s := result.Secrets[0]

	if s.Key != "secret_note" {
		t.Errorf("Key = %q, want %q", s.Key, "secret_note")
	}

	if s.Fields["notes"].Value != "This is a very secret note" {
		t.Errorf("notes = %q, want %q", s.Fields["notes"].Value, "This is a very secret note")
	}

	if !s.Fields["notes"].Sensitive {
		t.Error("notes should be sensitive")
	}
}

func TestBitwardenParser_ParseCard(t *testing.T) {
	jsonData := `{
		"items": [{
			"type": 3,
			"name": "My Credit Card",
			"notes": "Primary card",
			"card": {
				"cardholderName": "John Doe",
				"number": "4111111111111111",
				"expMonth": "12",
				"expYear": "2025",
				"code": "123",
				"brand": "Visa"
			}
		}],
		"folders": []
	}`

	p := &BitwardenParser{}
	result, err := p.Parse([]byte(jsonData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 1 {
		t.Fatalf("Secrets count = %d, want 1", len(result.Secrets))
	}

	s := result.Secrets[0]

	if s.Key != "my_credit_card" {
		t.Errorf("Key = %q, want %q", s.Key, "my_credit_card")
	}

	// Check sensitive fields per ADR-006
	if s.Fields["number"].Value != "4111111111111111" {
		t.Errorf("number = %q, want %q", s.Fields["number"].Value, "4111111111111111")
	}
	if !s.Fields["number"].Sensitive {
		t.Error("number should be sensitive")
	}

	if s.Fields["cvv"].Value != "123" {
		t.Errorf("cvv = %q, want %q", s.Fields["cvv"].Value, "123")
	}
	if !s.Fields["cvv"].Sensitive {
		t.Error("cvv should be sensitive")
	}

	// Check non-sensitive fields
	if s.Fields["cardholder_name"].Value != "John Doe" {
		t.Errorf("cardholder_name = %q, want %q", s.Fields["cardholder_name"].Value, "John Doe")
	}
	if s.Fields["cardholder_name"].Sensitive {
		t.Error("cardholder_name should not be sensitive")
	}

	if s.Fields["brand"].Value != "Visa" {
		t.Errorf("brand = %q, want %q", s.Fields["brand"].Value, "Visa")
	}
}

func TestBitwardenParser_ParseIdentity(t *testing.T) {
	jsonData := `{
		"items": [{
			"type": 4,
			"name": "My Identity",
			"notes": "Personal identity",
			"identity": {
				"title": "Mr",
				"firstName": "John",
				"middleName": "William",
				"lastName": "Doe",
				"username": "johndoe",
				"company": "Acme Corp",
				"email": "john@example.com",
				"phone": "555-1234",
				"address1": "123 Main St",
				"address2": "Apt 4B",
				"address3": "",
				"city": "New York",
				"state": "NY",
				"postalCode": "10001",
				"country": "US",
				"ssn": "123-45-6789",
				"passportNumber": "AB1234567",
				"licenseNumber": "D123456789"
			}
		}],
		"folders": []
	}`

	p := &BitwardenParser{}
	result, err := p.Parse([]byte(jsonData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 1 {
		t.Fatalf("Secrets count = %d, want 1", len(result.Secrets))
	}

	s := result.Secrets[0]

	// Check non-sensitive fields per ADR-006
	nonSensitiveFields := []string{"title", "username", "company", "state", "country"}
	for _, f := range nonSensitiveFields {
		if _, exists := s.Fields[f]; !exists {
			continue
		}
		if s.Fields[f].Sensitive {
			t.Errorf("field %q should not be sensitive", f)
		}
	}

	// Check sensitive PII fields per ADR-006
	sensitiveFields := []string{
		"first_name", "middle_name", "last_name", "email", "phone",
		"address1", "address2", "city", "postal_code", "ssn", "passport", "license",
	}
	for _, f := range sensitiveFields {
		if _, exists := s.Fields[f]; !exists {
			continue
		}
		if !s.Fields[f].Sensitive {
			t.Errorf("field %q should be sensitive", f)
		}
	}

	// Verify specific values
	if s.Fields["ssn"].Value != "123-45-6789" {
		t.Errorf("ssn = %q, want %q", s.Fields["ssn"].Value, "123-45-6789")
	}
	if s.Fields["passport"].Value != "AB1234567" {
		t.Errorf("passport = %q, want %q", s.Fields["passport"].Value, "AB1234567")
	}
}

func TestBitwardenParser_CustomFields(t *testing.T) {
	jsonData := `{
		"items": [{
			"type": 1,
			"name": "Test Login",
			"login": {
				"uris": [{"uri": "https://example.com"}],
				"username": "user",
				"password": "pass"
			},
			"fields": [
				{"name": "API Key", "value": "secret-api-key", "type": 1},
				{"name": "Note", "value": "visible note", "type": 0},
				{"name": "Active", "value": "true", "type": 2}
			]
		}],
		"folders": []
	}`

	p := &BitwardenParser{}
	result, err := p.Parse([]byte(jsonData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 1 {
		t.Fatalf("Secrets count = %d, want 1", len(result.Secrets))
	}

	s := result.Secrets[0]

	// Hidden field (type 1) should be sensitive
	apiKey, exists := s.Fields["api_key"]
	if !exists {
		t.Fatal("api_key field not found")
	}
	if !apiKey.Sensitive {
		t.Error("api_key (hidden field) should be sensitive")
	}

	// Text field (type 0) should not be sensitive
	note, exists := s.Fields["note"]
	if !exists {
		t.Fatal("note field not found")
	}
	if note.Sensitive {
		t.Error("note (text field) should not be sensitive")
	}

	// Boolean field (type 2) should not be sensitive
	active, exists := s.Fields["active"]
	if !exists {
		t.Fatal("active field not found")
	}
	if active.Sensitive {
		t.Error("active (boolean field) should not be sensitive")
	}
}

func TestBitwardenParser_MultipleURIs(t *testing.T) {
	jsonData := `{
		"items": [{
			"type": 1,
			"name": "Multi-URL Login",
			"login": {
				"uris": [
					{"uri": "https://example.com"},
					{"uri": "https://app.example.com"},
					{"uri": "https://api.example.com"}
				],
				"username": "user",
				"password": "pass"
			}
		}],
		"folders": []
	}`

	p := &BitwardenParser{}
	result, err := p.Parse([]byte(jsonData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 1 {
		t.Fatalf("Secrets count = %d, want 1", len(result.Secrets))
	}

	s := result.Secrets[0]

	// Primary URL
	if s.Fields["url"].Value != "https://example.com" {
		t.Errorf("url = %q, want %q", s.Fields["url"].Value, "https://example.com")
	}

	// Additional URLs
	if s.Fields["url_2"].Value != "https://app.example.com" {
		t.Errorf("url_2 = %q, want %q", s.Fields["url_2"].Value, "https://app.example.com")
	}
	if s.Fields["url_3"].Value != "https://api.example.com" {
		t.Errorf("url_3 = %q, want %q", s.Fields["url_3"].Value, "https://api.example.com")
	}

	// Metadata should have primary URL
	if s.Metadata == nil || s.Metadata.URL != "https://example.com" {
		t.Errorf("Metadata.URL = %v, want https://example.com", s.Metadata)
	}
}

func TestBitwardenParser_UnsupportedType(t *testing.T) {
	jsonData := `{
		"items": [{
			"type": 99,
			"name": "Unknown Type"
		}],
		"folders": []
	}`

	p := &BitwardenParser{}
	result, err := p.Parse([]byte(jsonData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Warnings) != 1 {
		t.Errorf("Warnings count = %d, want 1", len(result.Warnings))
	}

	if len(result.Secrets) != 0 {
		t.Errorf("Secrets count = %d, want 0", len(result.Secrets))
	}
}

func TestBitwardenParser_InvalidJSON(t *testing.T) {
	jsonData := `{invalid json`

	p := &BitwardenParser{}
	_, err := p.Parse([]byte(jsonData), ParseOptions{PreserveCase: false})
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestBitwardenParser_EmptyItems(t *testing.T) {
	jsonData := `{
		"items": [],
		"folders": []
	}`

	p := &BitwardenParser{}
	result, err := p.Parse([]byte(jsonData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 0 {
		t.Errorf("Secrets count = %d, want 0", len(result.Secrets))
	}
}

func TestBitwardenParser_FolderMapping(t *testing.T) {
	jsonData := `{
		"items": [
			{"type": 1, "name": "Login1", "folderId": "f1", "login": {"username": "u", "password": "p"}},
			{"type": 1, "name": "Login2", "folderId": "f2", "login": {"username": "u", "password": "p"}},
			{"type": 1, "name": "Login3", "folderId": null, "login": {"username": "u", "password": "p"}}
		],
		"folders": [
			{"id": "f1", "name": "Personal"},
			{"id": "f2", "name": "Work/Projects"}
		]
	}`

	p := &BitwardenParser{}
	result, err := p.Parse([]byte(jsonData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 3 {
		t.Fatalf("Secrets count = %d, want 3", len(result.Secrets))
	}

	// Check folder tags
	if len(result.Secrets[0].Tags) != 1 || result.Secrets[0].Tags[0] != "Personal" {
		t.Errorf("Login1 Tags = %v, want [Personal]", result.Secrets[0].Tags)
	}

	if len(result.Secrets[1].Tags) != 1 || result.Secrets[1].Tags[0] != "Work/Projects" {
		t.Errorf("Login2 Tags = %v, want [Work/Projects]", result.Secrets[1].Tags)
	}

	if len(result.Secrets[2].Tags) != 0 {
		t.Errorf("Login3 Tags = %v, want []", result.Secrets[2].Tags)
	}
}

func TestBitwardenParser_PreserveCase(t *testing.T) {
	jsonData := `{
		"items": [{
			"type": 1,
			"name": "GitHub_API_Key",
			"login": {"username": "user", "password": "pass"}
		}],
		"folders": []
	}`

	tests := []struct {
		name         string
		preserveCase bool
		wantKey      string
	}{
		{
			name:         "lowercase",
			preserveCase: false,
			wantKey:      "github_api_key",
		},
		{
			name:         "preserve case",
			preserveCase: true,
			wantKey:      "GitHub_API_Key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &BitwardenParser{}
			result, err := p.Parse([]byte(jsonData), ParseOptions{PreserveCase: tt.preserveCase})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Secrets[0].Key != tt.wantKey {
				t.Errorf("Key = %q, want %q", result.Secrets[0].Key, tt.wantKey)
			}
		})
	}
}

func TestBitwardenParser_Deduplication(t *testing.T) {
	jsonData := `{
		"items": [
			{"type": 1, "name": "Login", "login": {"username": "u1", "password": "p1"}},
			{"type": 1, "name": "Login", "login": {"username": "u2", "password": "p2"}},
			{"type": 1, "name": "Login", "login": {"username": "u3", "password": "p3"}}
		],
		"folders": []
	}`

	p := &BitwardenParser{}
	result, err := p.Parse([]byte(jsonData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 3 {
		t.Fatalf("Secrets count = %d, want 3", len(result.Secrets))
	}

	keys := make(map[string]bool)
	for _, s := range result.Secrets {
		if keys[s.Key] {
			t.Errorf("duplicate key found: %q", s.Key)
		}
		keys[s.Key] = true
	}

	expectedKeys := []string{"login", "login_1", "login_2"}
	for _, k := range expectedKeys {
		if !keys[k] {
			t.Errorf("expected key %q not found", k)
		}
	}
}

func TestBitwardenParser_LargeExport(t *testing.T) {
	// Generate large export with 500 items
	export := bitwardenExport{
		Items:   make([]bitwardenItem, 500),
		Folders: []bitwardenFolder{{ID: "f1", Name: "Test"}},
	}

	for i := 0; i < 500; i++ {
		folderID := "f1"
		export.Items[i] = bitwardenItem{
			Type:     1,
			Name:     "Login" + string(rune('A'+i%26)),
			FolderID: &folderID,
			Login: &bitwardenLogin{
				Username: "user",
				Password: "pass",
			},
		}
	}

	jsonData, _ := json.Marshal(export)

	p := &BitwardenParser{}
	result, err := p.Parse(jsonData, ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 500 {
		t.Errorf("Secrets count = %d, want 500", len(result.Secrets))
	}
}
