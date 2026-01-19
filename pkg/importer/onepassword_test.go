package importer

import (
	"strings"
	"testing"
)

func TestOnePasswordParser_Source(t *testing.T) {
	p := &OnePasswordParser{}
	if p.Source() != Source1Password {
		t.Errorf("Source() = %q, want %q", p.Source(), Source1Password)
	}
}

func TestOnePasswordParser_Parse(t *testing.T) {
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
			csvData: `Title,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes
GitHub,https://github.com,johndoe,mysecretpass123,otpauth://totp/GitHub?secret=ABC123,false,false,work,My GitHub account`,
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
				if s.Fields["totp"].Value != "otpauth://totp/GitHub?secret=ABC123" {
					t.Errorf("totp = %q, want %q", s.Fields["totp"].Value, "otpauth://totp/GitHub?secret=ABC123")
				}
				if !s.Fields["notes"].Sensitive {
					t.Error("notes should be sensitive")
				}
				if len(s.Tags) != 1 || s.Tags[0] != "work" {
					t.Errorf("Tags = %v, want [work]", s.Tags)
				}
				if s.Metadata == nil || s.Metadata.URL != "https://github.com" {
					t.Errorf("Metadata.URL = %v, want https://github.com", s.Metadata)
				}
			},
		},
		{
			name: "preserve case",
			csvData: `Title,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes
GitHub_API,https://github.com,johndoe,pass123,,false,false,,`,
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
			name: "multiple entries",
			csvData: `Title,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes
GitHub,https://github.com,user1,pass1,,false,false,work,notes1
GitLab,https://gitlab.com,user2,pass2,,false,false,work,notes2
Bitbucket,https://bitbucket.org,user3,pass3,,false,false,personal,notes3`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  3,
			wantWarnings: 0,
			wantError:    false,
		},
		{
			name: "skip empty entries",
			csvData: `Title,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes
GitHub,https://github.com,,,,,false,false,,
GitLab,https://gitlab.com,user2,pass2,,false,false,,notes2`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  1,
			wantWarnings: 1, // skipped warning for GitHub
			wantError:    false,
			checkFirst: func(t *testing.T, s *ImportedSecret) {
				if s.Key != "gitlab" {
					t.Errorf("Key = %q, want %q", s.Key, "gitlab")
				}
			},
		},
		{
			name: "multiple tags",
			csvData: `Title,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes
Test,https://test.com,user,pass,,false,false,"work, personal, important",notes`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  1,
			wantWarnings: 0,
			wantError:    false,
			checkFirst: func(t *testing.T, s *ImportedSecret) {
				if len(s.Tags) != 3 {
					t.Errorf("Tags length = %d, want 3", len(s.Tags))
				}
			},
		},
		{
			name: "fallback key from URL",
			csvData: `Title,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes
,https://example.com,user,pass,,false,false,,`,
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
			csvData: `Title,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes
,,user,pass,,false,false,,`,
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
			name: "missing required Title column",
			csvData: `Website,Username,Password
https://github.com,user,pass`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  0,
			wantWarnings: 0,
			wantError:    true,
		},
		{
			name:         "UTF-8 BOM handling",
			csvData:      "\xef\xbb\xbfTitle,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes\nGitHub,https://github.com,user,pass,,false,false,,",
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  1,
			wantWarnings: 0,
			wantError:    false,
		},
		{
			name: "column count mismatch",
			csvData: `Title,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes
GitHub,https://github.com,user`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  0,
			wantWarnings: 1,
			wantError:    false,
		},
		{
			name: "only password entry",
			csvData: `Title,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes
Test,,,,,,false,false,,`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  0,
			wantWarnings: 1, // skipped
			wantError:    false,
		},
		{
			name: "notes only entry",
			csvData: `Title,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes
Secure Note,,,,,false,false,,This is a secret note`,
			opts:         ParseOptions{PreserveCase: false},
			wantSecrets:  1,
			wantWarnings: 0,
			wantError:    false,
			checkFirst: func(t *testing.T, s *ImportedSecret) {
				if s.Fields["notes"].Value != "This is a secret note" {
					t.Errorf("notes = %q, want %q", s.Fields["notes"].Value, "This is a secret note")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &OnePasswordParser{}
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

func TestOnePasswordParser_Deduplication(t *testing.T) {
	csvData := `Title,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes
GitHub,https://github.com,user1,pass1,,false,false,,
GitHub,https://github.com,user2,pass2,,false,false,,
GitHub,https://github.com,user3,pass3,,false,false,,`

	p := &OnePasswordParser{}
	result, err := p.Parse([]byte(csvData), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 3 {
		t.Fatalf("Secrets count = %d, want 3", len(result.Secrets))
	}

	// Check deduplication
	keys := make(map[string]bool)
	for _, s := range result.Secrets {
		if keys[s.Key] {
			t.Errorf("duplicate key found: %q", s.Key)
		}
		keys[s.Key] = true
	}

	// Should have github, github_1, github_2
	expectedKeys := []string{"github", "github_1", "github_2"}
	for _, k := range expectedKeys {
		if !keys[k] {
			t.Errorf("expected key %q not found", k)
		}
	}
}

func TestOnePasswordParser_FieldSensitivity(t *testing.T) {
	csvData := `Title,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes
Test,https://test.com,myuser,mypass,otpauth://test,false,false,,my notes`

	p := &OnePasswordParser{}
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

func TestOnePasswordParser_LargeFile(t *testing.T) {
	// Generate a large CSV with 1000 entries
	var sb strings.Builder
	sb.WriteString("Title,Website,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes\n")
	for i := 0; i < 1000; i++ {
		sb.WriteString("Entry" + string(rune('0'+i%10)))
		sb.WriteString(",https://example.com,user,pass,,false,false,tag,notes\n")
	}

	p := &OnePasswordParser{}
	result, err := p.Parse([]byte(sb.String()), ParseOptions{PreserveCase: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Secrets) != 1000 {
		t.Errorf("Secrets count = %d, want 1000", len(result.Secrets))
	}
}
