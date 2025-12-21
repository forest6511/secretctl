package cli

import (
	"testing"
)

func TestExpandPattern(t *testing.T) {
	availableKeys := []string{
		"AWS_ACCESS_KEY",
		"AWS_SECRET_KEY",
		"DB_PASSWORD",
		"API_KEY",
		"CONFIG",
	}

	tests := []struct {
		name     string
		pattern  string
		expected []string
		wantErr  bool
	}{
		{
			name:     "exact match",
			pattern:  "API_KEY",
			expected: []string{"API_KEY"},
		},
		{
			name:     "wildcard prefix",
			pattern:  "AWS_*",
			expected: []string{"AWS_ACCESS_KEY", "AWS_SECRET_KEY"},
		},
		{
			name:     "wildcard suffix",
			pattern:  "*_KEY",
			expected: []string{"AWS_ACCESS_KEY", "AWS_SECRET_KEY", "API_KEY"},
		},
		{
			name:     "wildcard middle",
			pattern:  "AWS_*_KEY",
			expected: []string{"AWS_ACCESS_KEY", "AWS_SECRET_KEY"},
		},
		{
			name:     "question mark",
			pattern:  "DB_????????",
			expected: []string{"DB_PASSWORD"},
		},
		{
			name:     "match all",
			pattern:  "*",
			expected: []string{"AWS_ACCESS_KEY", "AWS_SECRET_KEY", "DB_PASSWORD", "API_KEY", "CONFIG"},
		},
		{
			name:    "no match glob",
			pattern: "NONEXISTENT_*",
			wantErr: true,
		},
		{
			name:    "no match exact",
			pattern: "NONEXISTENT",
			wantErr: true,
		},
		{
			name:    "invalid pattern",
			pattern: "[invalid",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ExpandPattern(tc.pattern, availableKeys)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tc.expected) {
				t.Errorf("got %d results, want %d", len(result), len(tc.expected))
				return
			}

			for _, exp := range tc.expected {
				found := false
				for _, r := range result {
					if r == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("missing expected key: %s", exp)
				}
			}
		})
	}
}

func TestExpandPatterns(t *testing.T) {
	availableKeys := []string{"a", "b", "c", "ab", "bc"}

	tests := []struct {
		name     string
		patterns []string
		expected []string
		wantErr  bool
	}{
		{
			name:     "single pattern",
			patterns: []string{"a"},
			expected: []string{"a"},
		},
		{
			name:     "multiple patterns",
			patterns: []string{"a", "b"},
			expected: []string{"a", "b"},
		},
		{
			name:     "overlapping patterns",
			patterns: []string{"a*", "ab"},
			expected: []string{"a", "ab"},
		},
		{
			name:     "glob pattern",
			patterns: []string{"*b"},
			expected: []string{"b", "ab"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ExpandPatterns(tc.patterns, availableKeys)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tc.expected) {
				t.Errorf("got %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestSortKeys(t *testing.T) {
	input := []string{"z", "a", "m", "b"}
	result := SortKeys(input)

	// Check original is unchanged
	if input[0] != "z" {
		t.Error("original slice was modified")
	}

	// Check sorted result
	expected := []string{"a", "b", "m", "z"}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("position %d: got %s, want %s", i, v, expected[i])
		}
	}
}

func TestMapKeys(t *testing.T) {
	m := map[string]int{"z": 1, "a": 2, "m": 3}
	result := MapKeys(m)

	expected := []string{"a", "m", "z"}
	if len(result) != len(expected) {
		t.Errorf("got %d keys, want %d", len(result), len(expected))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("position %d: got %s, want %s", i, v, expected[i])
		}
	}
}
