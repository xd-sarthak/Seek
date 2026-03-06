package utils

import (
	"strings"
	"unicode"
)

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
