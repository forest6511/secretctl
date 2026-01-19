package importer

import (
	"strings"
	"testing"
)

func TestLastPassParser_Source(t *testing.T) {
	p := &LastPassParser{}
	if p.Source() != SourceLastPass {
		t.Errorf("Source() = %q, want %q", p.Source(), SourceLastPass)
	}
}

func TestLastPassParser_Parse(t *testing.T) {
	tests := []struct {
		name         string
		csvData      string
		opts         ParseOptions
		wantSecrets  int
		wantWarnings int
		wantError    bool
		checkFirst   func(t *testing.T, s *ImportedSecret)
	}{
		{
			name: "standard login entry",
			csvData: `url,username,password,totp,extra,name,grouping,fav
https://github.com,johndoe,mysecretpass123,JBSWY3DPEHPK3PXP,My GitHub notes,GitHub,Work,1`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  1,
			wantWarnings: 0,
			wantError:    false,
			checkFirst: func(t *testing.T, s *ImportedSecret) {
				if s.Key != "github" {
					t.Errorf("Key = %q, want %q", s.Key, "github")
				}
				if s.OriginalName != "GitHub" {
					t.Errorf("OriginalName = %q, want %q", s.OriginalName, "GitHub")
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
			},
		},
		{
			name: "preserve case",
			csvData: `url,username,password,totp,extra,name,grouping,fav
https://github.com,johndoe,pass,,,GitHub_API,,`,
			opts:         ParseOptions{PreserveCase: true},
			wantSecrets:  1,
			wantWarnings: 0,
			wantError:    false,
			checkFirst: func(t *testing.T, s *ImportedSecret) {
				if s.Key != "GitHub_API" {
					t.Errorf("Key = %q, want %q", s.Key, "GitHub_API")
				}
			},
		},
		{
			name: "secure note (http://sn URL)",
			csvData: `url,username,password,totp,extra,name,grouping,fav
http://sn,,,,"This is a secure note",My Secret Note,Notes,0`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  1,
			wantWarnings: 0,
			wantError:    false,
			checkFirst: func(t *testing.T, s *ImportedSecret) {
				if s.Key != "my_secret_note" {
					t.Errorf("Key = %q, want %q", s.Key, "my_secret_note")
				}
				// URL field should not be included for secure notes
				if _, exists := s.Fields["url"]; exists {
					t.Error("url field should not exist for secure notes")
				}
				if s.Fields["notes"].Value != "This is a secure note" {
					t.Errorf("notes = %q, want %q", s.Fields["notes"].Value, "This is a secure note")
				}
				// Metadata should be nil for secure notes
				if s.Metadata != nil {
					t.Errorf("Metadata = %v, want nil for secure notes", s.Metadata)
				}
			},
		},
		{
			name: "HTML entity decoding",
			csvData: `url,username,password,totp,extra,name,grouping,fav
https://test.com,user&amp;admin,pass&lt;123&gt;,,Notes with &quot;quotes&quot; and &#39;apostrophe&#39;,Test &amp; Entry,,`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  1,
			wantWarnings: 0,
			wantError:    false,
			checkFirst: func(t *testing.T, s *ImportedSecret) {
				if s.Fields["username"].Value != "user&admin" {
					t.Errorf("username = %q, want %q", s.Fields["username"].Value, "user&admin")
				}
				if s.Fields["password"].Value != "pass<123>" {
					t.Errorf("password = %q, want %q", s.Fields["password"].Value, "pass<123>")
				}
				expectedNotes := `Notes with "quotes" and 'apostrophe'`
				if s.Fields["notes"].Value != expectedNotes {
					t.Errorf("notes = %q, want %q", s.Fields["notes"].Value, expectedNotes)
				}
			},
		},
		{
			name: "nested grouping preserved",
			csvData: `url,username,password,totp,extra,name,grouping,fav
https://test.com,user,pass,,,Test,Work/Servers/Production,`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  1,
			wantWarnings: 0,
			wantError:    false,
			checkFirst: func(t *testing.T, s *ImportedSecret) {
				if len(s.Tags) != 1 || s.Tags[0] != "Work/Servers/Production" {
					t.Errorf("Tags = %v, want [Work/Servers/Production]", s.Tags)
				}
			},
		},
		{
			name: "multiple entries",
			csvData: `url,username,password,totp,extra,name,grouping,fav
https://github.com,user1,pass1,,,GitHub,,
https://gitlab.com,user2,pass2,,,GitLab,,
https://bitbucket.org,user3,pass3,,,Bitbucket,,`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  3,
			wantWarnings: 0,
			wantError:    false,
		},
		{
			name: "skip empty entries",
			csvData: `url,username,password,totp,extra,name,grouping,fav
https://github.com,,,,,GitHub,,
https://gitlab.com,user2,pass2,,,GitLab,,`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  1,
			wantWarnings: 1, // skipped warning
			wantError:    false,
			checkFirst: func(t *testing.T, s *ImportedSecret) {
				if s.Key != "gitlab" {
					t.Errorf("Key = %q, want %q", s.Key, "gitlab")
				}
			},
		},
		{
			name: "fallback key from URL",
			csvData: `url,username,password,totp,extra,name,grouping,fav
https://example.com,user,pass,,,,,`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  1,
			wantWarnings: 0,
			wantError:    false,
			checkFirst: func(t *testing.T, s *ImportedSecret) {
				if s.Key != "examplecom" {
					t.Errorf("Key = %q, want %q", s.Key, "examplecom")
				}
			},
		},
		{
			name: "fallback key counter",
			csvData: `url,username,password,totp,extra,name,grouping,fav
,user,pass,,,,,`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  1,
			wantWarnings: 0,
			wantError:    false,
			checkFirst: func(t *testing.T, s *ImportedSecret) {
				if s.Key != "imported_item_1" {
					t.Errorf("Key = %q, want %q", s.Key, "imported_item_1")
				}
			},
		},
		{
			name:         "missing header",
			csvData:      "",
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  0,
			wantWarnings: 0,
			wantError:    true,
		},
		{
			name: "missing required name column",
			csvData: `url,username,password
https://github.com,user,pass`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  0,
			wantWarnings: 0,
			wantError:    true,
		},
		{
			name:         "UTF-8 BOM handling",
			csvData:      "\xef\xbb\xbfurl,username,password,totp,extra,name,grouping,fav\nhttps://github.com,user,pass,,,GitHub,,",
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  1,
			wantWarnings: 0,
			wantError:    false,
		},
		{
			name: "column count mismatch",
			csvData: `url,username,password,totp,extra,name,grouping,fav
https://github.com,user`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  0,
			wantWarnings: 1,
			wantError:    false,
		},
		{
			name: "case insensitive header",
			csvData: `URL,USERNAME,PASSWORD,TOTP,EXTRA,NAME,GROUPING,FAV
https://github.com,user,pass,,,GitHub,,`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  1,
			wantWarnings: 0,
			wantError:    false,
		},
		{
			name: "no TOTP exported (common in free LastPass)",
			csvData: `url,username,password,extra,name,grouping,fav
https://github.com,user,pass,notes,GitHub,,`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  1,
			wantWarnings: 0,
			wantError:    false,
			checkFirst: func(t *testing.T, s *ImportedSecret) {
				if _, exists := s.Fields["totp"]; exists {
					t.Error("totp field should not exist when column is missing")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &LastPassParser{}
			result, err := p.Parse([]byte(tt.csvData), tt.opts)

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.Secrets) != tt.wantSecrets {
				t.Errorf("Secrets count = %d, want %d", len(result.Secrets), tt.wantSecrets)
			}

			if len(result.Warnings) != tt.wantWarnings {
				t.Errorf("Warnings count = %d, want %d", len(result.Warnings), tt.wantWarnings)
			}

			if tt.checkFirst != nil && len(result.Secrets) > 0 {
				tt.checkFirst(t, result.Secrets[0])
			}
		})
	}
}

func TestLastPassParser_Deduplication(t *testing.T) {
	csvData := `url,username,password,totp,extra,name,grouping,fav
https://github.com,user1,pass1,,,GitHub,,
https://github.com,user2,pass2,,,GitHub,,
https://github.com,user3,pass3,,,GitHub,,`

	p := &LastPassParser{}
	result, err := p.Parse([]byte(csvData), ParseOptions{PreserveCase: false})
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

	expectedKeys := []string{"github", "github_1", "github_2"}
	for _, k := range expectedKeys {
		if !keys[k] {
			t.Errorf("expected key %q not found", k)
		}
	}
}

func TestLastPassParser_FieldSensitivity(t *testing.T) {
	csvData := `url,username,password,totp,extra,name,grouping,fav
https://test.com,myuser,mypass,otpauth://test,secret notes,Test,,`

	p := &LastPassParser{}
	result, err := p.Parse([]byte(csvData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 1 {
		t.Fatalf("Secrets count = %d, want 1", len(result.Secrets))
	}

	s := result.Secrets[0]

	// Check sensitivity according to ADR-006
	sensitivityTests := []struct {
		field     string
		sensitive bool
	}{
		{"username", false},
		{"password", true},
		{"totp", true},
		{"notes", true},
		{"url", false},
	}

	for _, st := range sensitivityTests {
		field, exists := s.Fields[st.field]
		if !exists {
			t.Errorf("field %q not found", st.field)
			continue
		}
		if field.Sensitive != st.sensitive {
			t.Errorf("field %q Sensitive = %v, want %v", st.field, field.Sensitive, st.sensitive)
		}
	}
}

func TestLastPassParser_PasswordKind(t *testing.T) {
	csvData := `url,username,password,totp,extra,name,grouping,fav
https://test.com,user,secretpass,,,Test,,`

	p := &LastPassParser{}
	result, err := p.Parse([]byte(csvData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 1 {
		t.Fatalf("Secrets count = %d, want 1", len(result.Secrets))
	}

	if result.Secrets[0].Fields["password"].Kind != "password" {
		t.Errorf("password Kind = %q, want %q", result.Secrets[0].Fields["password"].Kind, "password")
	}
}

func TestLastPassParser_TOTPHint(t *testing.T) {
	csvData := `url,username,password,totp,extra,name,grouping,fav
https://test.com,user,pass,JBSWY3DPEHPK3PXP,,Test,,`

	p := &LastPassParser{}
	result, err := p.Parse([]byte(csvData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 1 {
		t.Fatalf("Secrets count = %d, want 1", len(result.Secrets))
	}

	if result.Secrets[0].Fields["totp"].Hint != "TOTP seed" {
		t.Errorf("totp Hint = %q, want %q", result.Secrets[0].Fields["totp"].Hint, "TOTP seed")
	}
}

func TestLastPassParser_NotesInputType(t *testing.T) {
	csvData := `url,username,password,totp,extra,name,grouping,fav
https://test.com,user,pass,,Some extra notes,Test,,`

	p := &LastPassParser{}
	result, err := p.Parse([]byte(csvData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 1 {
		t.Fatalf("Secrets count = %d, want 1", len(result.Secrets))
	}

	if result.Secrets[0].Fields["notes"].InputType != "textarea" {
		t.Errorf("notes InputType = %q, want %q", result.Secrets[0].Fields["notes"].InputType, "textarea")
	}
}

func TestLastPassParser_LazyQuotes(t *testing.T) {
	// Test handling of malformed CSV with unbalanced quotes
	csvData := `url,username,password,totp,extra,name,grouping,fav
https://test.com,user,pass,,"Note with "embedded" quotes",Test,,`

	p := &LastPassParser{}
	result, err := p.Parse([]byte(csvData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// LazyQuotes should handle this gracefully
	if len(result.Secrets) != 1 {
		t.Errorf("Secrets count = %d, want 1", len(result.Secrets))
	}
}

func TestLastPassParser_LargeFile(t *testing.T) {
	// Generate a large CSV with 1000 entries
	var sb strings.Builder
	sb.WriteString("url,username,password,totp,extra,name,grouping,fav\n")
	for i := 0; i < 1000; i++ {
		sb.WriteString("https://example.com,user,pass,,notes,Entry")
		sb.WriteString(string(rune('0' + i%10)))
		sb.WriteString(",Work,0\n")
	}

	p := &LastPassParser{}
	result, err := p.Parse([]byte(sb.String()), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 1000 {
		t.Errorf("Secrets count = %d, want 1000", len(result.Secrets))
	}
}

func TestLastPassParser_SpecialCharactersInValues(t *testing.T) {
	csvData := `url,username,password,totp,extra,name,grouping,fav
https://test.com,"user,with,commas","pass""with""quotes",,"extra with
newline",Test,,`

	p := &LastPassParser{}
	result, err := p.Parse([]byte(csvData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 1 {
		t.Fatalf("Secrets count = %d, want 1", len(result.Secrets))
	}

	s := result.Secrets[0]

	if s.Fields["username"].Value != "user,with,commas" {
		t.Errorf("username = %q, want %q", s.Fields["username"].Value, "user,with,commas")
	}

	if s.Fields["password"].Value != `pass"with"quotes` {
		t.Errorf("password = %q, want %q", s.Fields["password"].Value, `pass"with"quotes`)
	}

	expectedNotes := "extra with\nnewline"
	if s.Fields["notes"].Value != expectedNotes {
		t.Errorf("notes = %q, want %q", s.Fields["notes"].Value, expectedNotes)
	}
}

func TestLastPassParser_URLMetadata(t *testing.T) {
	csvData := `url,username,password,totp,extra,name,grouping,fav
https://github.com/login,user,pass,,,GitHub,,`

	p := &LastPassParser{}
	result, err := p.Parse([]byte(csvData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 1 {
		t.Fatalf("Secrets count = %d, want 1", len(result.Secrets))
	}

	s := result.Secrets[0]

	// URL should be in both fields and metadata
	if s.Fields["url"].Value != "https://github.com/login" {
		t.Errorf("url field = %q, want %q", s.Fields["url"].Value, "https://github.com/login")
	}

	if s.Metadata == nil || s.Metadata.URL != "https://github.com/login" {
		t.Errorf("Metadata.URL = %v, want https://github.com/login", s.Metadata)
	}
}
