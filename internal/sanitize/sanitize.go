// Package sanitize provides multi-layer output sanitization for secretctl.
// It detects and redacts secrets even when encoded or fragmented.
package sanitize

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// Config holds sanitization configuration.
type Config struct {
	MaxOutputLength int
	MaxOutputLines  int
	DetectBase64    bool
	DetectHex       bool
	DetectFragments bool
}

// DefaultConfig returns the default sanitization configuration.
var DefaultConfig = Config{
	MaxOutputLength: 5120,
	MaxOutputLines:  200,
	DetectBase64:    true,
	DetectHex:       true,
	DetectFragments: true,
}

// Pre-compiled regex patterns for detection
var (
	base64Pattern    = regexp.MustCompile(`[A-Za-z0-9+/]{20,}={0,2}`)
	hexPattern       = regexp.MustCompile(`[0-9a-fA-F]{40,}`)
	fragmentPatterns = []*regexp.Regexp{
		regexp.MustCompile(`PART\d*:\s*(\S+)`),
		regexp.MustCompile(`\d+:\s*([a-zA-Z0-9\-]{2,4})`),
		regexp.MustCompile(`([a-zA-Z0-9\-]{1,2})(?:\|)+`),
		regexp.MustCompile(`([a-zA-Z0-9\-]{1,2})(?:\s){3,}`),
	}
)

// SanitizeMultiLayer applies multiple sanitization layers to detect and redact secrets.
// Layers:
// 1. Exact string matching
// 2. Base64 encoding detection
// 3. Hex encoding detection
// 4. Fragment pattern detection
func SanitizeMultiLayer(output string, secrets map[string]string, config Config) string {
	if len(output) == 0 {
		return output
	}

	sanitized := output

	// Layer 1: Exact string matching
	for name, value := range secrets {
		if len(value) >= 4 {
			sanitized = strings.ReplaceAll(sanitized, value, fmt.Sprintf("[REDACTED:%s]", name))
		}
	}

	// Layer 2: Base64 detection
	if config.DetectBase64 {
		sanitized = detectAndSanitizeBase64(sanitized, secrets)
	}

	// Layer 3: Hex detection
	if config.DetectHex {
		sanitized = detectAndSanitizeHex(sanitized, secrets)
	}

	// Layer 4: Fragment detection
	if config.DetectFragments {
		sanitized = detectAndSanitizeFragments(sanitized, secrets)
	}

	return sanitized
}

// detectAndSanitizeBase64 detects base64-encoded content and checks if it contains secrets.
func detectAndSanitizeBase64(output string, secrets map[string]string) string {
	return base64Pattern.ReplaceAllStringFunc(output, func(match string) string {
		decoded, err := base64.StdEncoding.DecodeString(match)
		if err != nil {
			// Try URL-safe encoding
			decoded, err = base64.RawURLEncoding.DecodeString(strings.TrimRight(match, "="))
			if err != nil {
				return match
			}
		}

		decodedStr := string(decoded)
		for name, value := range secrets {
			if len(value) >= 4 && strings.Contains(decodedStr, value) {
				return fmt.Sprintf("[REDACTED:%s]", name)
			}
		}

		// Recursive check for nested encoding
		if base64Pattern.MatchString(decodedStr) {
			decodedStr = detectAndSanitizeBase64(decodedStr, secrets)
			for name, value := range secrets {
				if len(value) >= 4 && strings.Contains(decodedStr, value) {
					return fmt.Sprintf("[REDACTED:%s]", name)
				}
			}
		}

		return match
	})
}

// detectAndSanitizeHex detects hex-encoded content and checks if it contains secrets.
func detectAndSanitizeHex(output string, secrets map[string]string) string {
	return hexPattern.ReplaceAllStringFunc(output, func(match string) string {
		decoded, err := hex.DecodeString(match)
		if err != nil {
			return match
		}

		decodedStr := string(decoded)
		for name, value := range secrets {
			if len(value) >= 4 && strings.Contains(decodedStr, value) {
				return fmt.Sprintf("[REDACTED:%s]", name)
			}
		}

		return match
	})
}

// detectAndSanitizeFragments detects fragmentation patterns that could reconstruct secrets.
func detectAndSanitizeFragments(output string, secrets map[string]string) string {
	sanitized := output

	for _, pattern := range fragmentPatterns {
		matches := pattern.FindAllString(output, -1)
		if len(matches) >= 10 {
			// Check if fragments can reconstruct any secret
			for name, value := range secrets {
				if len(value) < 4 {
					continue
				}

				// Try to reconstruct from fragments
				reconstructed := strings.Join(matches, "")
				if strings.Contains(reconstructed, value) {
					for _, match := range matches {
						sanitized = strings.ReplaceAll(sanitized, match, fmt.Sprintf("[REDACTED:%s]", name))
					}
					continue
				}

				// Try normalized reconstruction (remove separators)
				normalized := normalizeForFragmentCheck(reconstructed)
				normalizedValue := normalizeForFragmentCheck(value)
				if strings.Contains(normalized, normalizedValue) {
					for _, match := range matches {
						sanitized = strings.ReplaceAll(sanitized, match, fmt.Sprintf("[REDACTED:%s]", name))
					}
				}
			}
		}
	}

	return sanitized
}

// normalizeForFragmentCheck removes separators and lowercases for comparison.
func normalizeForFragmentCheck(s string) string {
	var result strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result.WriteRune(c)
		}
	}
	return strings.ToLower(result.String())
}

// ContainsSecret checks if output contains any secret (direct or encoded).
func ContainsSecret(output string, secrets map[string]string) bool {
	if len(output) == 0 {
		return false
	}

	for _, value := range secrets {
		if len(value) >= 4 && strings.Contains(output, value) {
			return true
		}
	}

	// Check base64
	base64Matches := base64Pattern.FindAllString(output, -1)
	for _, match := range base64Matches {
		decoded, err := base64.StdEncoding.DecodeString(match)
		if err != nil {
			decoded, err = base64.RawURLEncoding.DecodeString(strings.TrimRight(match, "="))
			if err != nil {
				continue
			}
		}
		for _, value := range secrets {
			if len(value) >= 4 && strings.Contains(string(decoded), value) {
				return true
			}
		}
	}

	// Check hex
	hexMatches := hexPattern.FindAllString(output, -1)
	for _, match := range hexMatches {
		decoded, err := hex.DecodeString(match)
		if err != nil {
			continue
		}
		for _, value := range secrets {
			if len(value) >= 4 && strings.Contains(string(decoded), value) {
				return true
			}
		}
	}

	return false
}
