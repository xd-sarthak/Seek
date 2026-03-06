package utils

import (
	"fmt"
	"strings"
	"net/url"
)

func NormalizeURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)

    if err != nil {
        return "", fmt.Errorf("Could not parse raw URL [%w]", err) 
    }

    if u.Scheme != "https" && u.Scheme != "http" {
        return "", fmt.Errorf("URL has invalid field 'Scheme'")
    }

    if u.Host == "" {
        return "", fmt.Errorf("URL has no field 'Host'")
    }

	host := u.Host
    host = strings.TrimPrefix(host, "www.")

    normalizedURL := host

    if u.Path != "" {
        trimmedPath := strings.TrimSuffix(u.Path, "/")
        normalizedURL += trimmedPath
    }

    return normalizedURL, nil
}