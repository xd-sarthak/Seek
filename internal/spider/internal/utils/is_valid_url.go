package utils

import (
	"strings"
	"unicode"
)

// IsValidURL performs fast pre-filtering on a URL to determine if it should
// be enqueued for crawling.
//
// A URL is considered invalid if:
//   - It contains "w/index.php" (Wikipedia special/action pages)
//   - It contains non-ASCII characters (codepoint > 127)
//   - It contains percent-encoded sequences (%XX)
//   - It contains characters that are neither letters, digits, nor allowed symbols
func IsValidURL(link string) bool {
	// Ignore this en.wikipedia.org/w/index.php
	if strings.Contains(link, "w/index.php") {
		return false
	}
	for _, r := range link {
		if r > 127 || (!unicode.IsLetter(r) && !unicode.IsDigit(r) && !isAllowedSymbol(r)) {
			return false
		}
	}

	if strings.Contains(link, "%") {
		return false
	}

	return true
}

// isAllowedSymbol checks whether a rune is in the set of allowed URL characters:
// -._~:/?#[]@!$&'()*+,;= and all printable ASCII characters.
func isAllowedSymbol(r rune) bool {
	allowed := "-._~:/?#[]@!$&'()*+,;="
	return (r < 127 && unicode.IsPrint(r)) || containsRune(allowed, r)
}

func containsRune(str string, r rune) bool {
	for _, c := range str {
		if c == r {
			return true
		}
	}

	return false
}
