package security

import "testing"

func TestEdition_String(t *testing.T) {
	tests := []struct {
		edition Edition
		want    string
	}{
		{EditionFree, "Free"},
		{EditionTeam, "Team"},
		{Edition(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.edition.String(); got != tt.want {
				t.Errorf("Edition.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLimits(t *testing.T) {
	tests := []struct {
		name    string
		edition Edition
		want    Limits
	}{
		{
			name:    "free_edition",
			edition: EditionFree,
			want: Limits{
				DuplicateLimit: 3,
				WeakLimit:      3,
				ExportEnabled:  false,
			},
		},
		{
			name:    "team_edition",
			edition: EditionTeam,
			want: Limits{
				DuplicateLimit: 0,
				WeakLimit:      0,
				ExportEnabled:  true,
			},
		},
		{
			name:    "unknown_defaults_to_free",
			edition: Edition(99),
			want: Limits{
				DuplicateLimit: 3,
				WeakLimit:      3,
				ExportEnabled:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetLimits(tt.edition)
			if got.DuplicateLimit != tt.want.DuplicateLimit {
				t.Errorf("DuplicateLimit = %v, want %v", got.DuplicateLimit, tt.want.DuplicateLimit)
			}
			if got.WeakLimit != tt.want.WeakLimit {
				t.Errorf("WeakLimit = %v, want %v", got.WeakLimit, tt.want.WeakLimit)
			}
			if got.ExportEnabled != tt.want.ExportEnabled {
				t.Errorf("ExportEnabled = %v, want %v", got.ExportEnabled, tt.want.ExportEnabled)
			}
		})
	}
}

func TestLimits_IsLimited(t *testing.T) {
	tests := []struct {
		name   string
		limits Limits
		want   bool
	}{
		{
			name:   "free_is_limited",
			limits: GetLimits(EditionFree),
			want:   true,
		},
		{
			name:   "team_not_limited",
			limits: GetLimits(EditionTeam),
			want:   false,
		},
		{
			name:   "custom_limited",
			limits: Limits{DuplicateLimit: 5, WeakLimit: 0},
			want:   true,
		},
		{
			name:   "custom_unlimited",
			limits: Limits{DuplicateLimit: 0, WeakLimit: 0},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.limits.IsLimited(); got != tt.want {
				t.Errorf("Limits.IsLimited() = %v, want %v", got, tt.want)
			}
		})
	}
}
