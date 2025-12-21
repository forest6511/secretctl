// Package cli provides shared utilities for CLI commands.
package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ExpandPattern expands a glob pattern against available keys.
// If the pattern contains glob characters (*?[), it performs glob matching.
// Otherwise, it performs exact matching.
func ExpandPattern(pattern string, availableKeys []string) ([]string, error) {
	// Validate pattern syntax
	if _, err := filepath.Match(pattern, ""); err != nil {
		return nil, fmt.Errorf("invalid pattern '%s': %w", pattern, err)
	}

	// Check if pattern contains glob characters
	hasGlob := strings.ContainsAny(pattern, "*?[")

	if !hasGlob {
		// Exact match - verify key exists
		for _, key := range availableKeys {
			if key == pattern {
				return []string{pattern}, nil
			}
		}
		return nil, fmt.Errorf("key '%s' not found", pattern)
	}

	// Glob matching
	var matches []string
	for _, key := range availableKeys {
		matched, err := filepath.Match(pattern, key)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, key)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no keys match pattern '%s'", pattern)
	}

	return matches, nil
}

// ExpandPatterns expands multiple glob patterns against available keys.
// Returns unique matching keys preserving order of first match.
func ExpandPatterns(patterns []string, availableKeys []string) ([]string, error) {
	seen := make(map[string]bool)
	var result []string

	for _, pattern := range patterns {
		matches, err := ExpandPattern(pattern, availableKeys)
		if err != nil {
			return nil, err
		}
		for _, key := range matches {
			if !seen[key] {
				seen[key] = true
				result = append(result, key)
			}
		}
	}

	return result, nil
}

// SortKeys returns a sorted copy of the keys slice.
func SortKeys(keys []string) []string {
	sorted := make([]string, len(keys))
	copy(sorted, keys)
	sort.Strings(sorted)
	return sorted
}

// MapKeys extracts keys from a map and returns them sorted.
func MapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
