package utils

import (
	"fmt"
	"net/url"
)

// StripURL removes the fragment component (#...) from a URL while preserving
// the scheme, host, path, and query parameters.
//
// This is called before normalization to ensure fragments do not create
// false URL duplicates in the frontier.
//
// Example:
//
//	https://example.com/page#section -> https://example.com/page
//	https://example.com/page?q=foo#top -> https://example.com/page?q=foo
func StripURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)

	if err != nil {
		return "", fmt.Errorf("could not parse URL: %w", err)
	}

	if u.Scheme == "" {
		return "", fmt.Errorf("URL has no scheme")
	}

	if u.Host == "" {
		return "", fmt.Errorf("URL has no host")
	}

	// Only strip the fragment; preserve query parameters
	u.Fragment = ""
	u.RawFragment = ""

	return u.String(), nil
}