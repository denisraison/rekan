package agent

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var accentStripper = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// normalizeForMatch strips accents, lowercases, and trims for fuzzy comparison.
// Safe for sequential use (agent processes messages one at a time via debouncer).
func normalizeForMatch(s string) string {
	accentStripper.Reset()
	result, _, _ := transform.String(accentStripper, s)
	return strings.ToLower(strings.TrimSpace(result))
}
