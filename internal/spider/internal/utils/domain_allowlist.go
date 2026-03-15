package utils

import (
	"fmt"
	"net/url"
	"strings"
)

// ParseAllowedDomains converts a comma-separated env var into a normalized host set.
func ParseAllowedDomains(raw string) map[string]struct{} {
	allowedDomains := make(map[string]struct{})

	for _, domain := range strings.Split(raw, ",") {
		normalizedDomain := NormalizeHost(domain)
		if normalizedDomain == "" {
			continue
		}

		allowedDomains[normalizedDomain] = struct{}{}
	}

	return allowedDomains
}

// IsURLAllowed reports whether rawURL belongs to an allowed host.
// An empty allowlist means "allow all" to preserve the current crawler behavior.
func IsURLAllowed(rawURL string, allowedDomains map[string]struct{}) (bool, error) {
	if len(allowedDomains) == 0 {
		return true, nil
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false, fmt.Errorf("could not parse URL: %w", err)
	}

	host := NormalizeHost(parsedURL.Host)
	if host == "" {
		return false, fmt.Errorf("URL has no host")
	}

	return IsHostAllowed(host, allowedDomains), nil
}

// IsHostAllowed reports whether a normalized host belongs to the allowlist.
// An empty allowlist means "allow all" to preserve the current crawler behavior.
func IsHostAllowed(host string, allowedDomains map[string]struct{}) bool {
	if len(allowedDomains) == 0 {
		return true
	}

	_, allowed := allowedDomains[host]
	return allowed
}

// NormalizeHost canonicalizes a host for allowlist and policy matching.
func NormalizeHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimSuffix(host, ".")

	if host == "" {
		return ""
	}

	if strings.Contains(host, "://") {
		if parsedURL, err := url.Parse(host); err == nil {
			host = parsedURL.Host
		}
	}

	if parsedHost, _, found := strings.Cut(host, ":"); found && parsedHost != "" {
		host = parsedHost
	}

	return host
}
