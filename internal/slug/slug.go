package slug

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Generate creates a URL-friendly slug from a French string.
func Generate(s string) string {
	// Normalize to NFD to decompose accented characters
	s = norm.NFD.String(s)

	var b strings.Builder
	prevDash := false

	for _, r := range s {
		switch {
		case unicode.Is(unicode.Mn, r):
			// Skip combining marks (accents)
			continue
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(unicode.ToLower(r))
			prevDash = false
		case r == ' ' || r == '-' || r == '_' || r == '.':
			if !prevDash && b.Len() > 0 {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}

	result := b.String()
	// Trim trailing dash
	result = strings.TrimRight(result, "-")
	return result
}
