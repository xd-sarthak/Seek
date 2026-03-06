package utils

import (
	"fmt"
	"net/url"
	"strings"
)

// trackingParams are query parameters that should be stripped during normalization
var trackingParams = map[string]struct{}{
	"utm_source":   {},
	"utm_medium":   {},
	"utm_campaign": {},
	"utm_term":     {},
	"utm_content":  {},
	"fbclid":       {},
	"gclid":        {},
	"ref":          {},
}

// NormalizeURL cleans and canonicalizes a URL to ensure duplicate URLs
// are not crawled multiple times.
//
// Steps:
//  1. Parse and validate scheme (http/https only)
//  2. Lowercase the hostname
//  3. Strip "www." prefix
//  4. Remove trailing slash from path
//  5. Remove tracking query parameters (utm_*, fbclid, gclid, ref)
//  6. Reassemble the canonical URL
//
// Examples:
//
//	https://WWW.Example.Com/Page/ -> https://example.com/Page
//	https://example.com/page?utm_source=twitter&q=go -> https://example.com/page?q=go
//	https://example.com/page#section -> https://example.com/page
func NormalizeURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)

	if err != nil {
		return "", fmt.Errorf("could not parse raw URL: %w", err)
	}

	if u.Scheme != "https" && u.Scheme != "http" {
		return "", fmt.Errorf("URL has invalid scheme: %s", u.Scheme)
	}

	if u.Host == "" {
		return "", fmt.Errorf("URL has no host")
	}

	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")

	path := strings.TrimSuffix(u.Path, "/")

	normalizedURL := u.Scheme + "://" + host + path

	// Preserve query parameters (sorted via url.Values.Encode), excluding tracking params
	if u.RawQuery != "" {
		params := u.Query()
		for key := range params {
			lk := strings.ToLower(key)
			if _, isTracking := trackingParams[lk]; isTracking {
				params.Del(key)
			}
		}
		if encoded := params.Encode(); encoded != "" {
			normalizedURL += "?" + encoded
		}
	}

	return normalizedURL, nil
}