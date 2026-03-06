package utils

import (
	"fmt"
	"net/url"
	"strings"
)

//removes queries and fragments
func StripURL(rawURL string) (string, error) {
	u,err := url.Parse(rawURL)

	if err != nil {
		return "",fmt.Errorf("Could not parse URL [%w]", err)
	}

	if u.Scheme == "" {
        return "", fmt.Errorf("URL has no field 'Scheme'")
    }

    if u.Host == "" {
        return "", fmt.Errorf("URL has no field 'Host'")
    }

	strippedURL := u.Scheme + "://" + u.Host

	if u.Path != "" {
		trimmedPath := strings.TrimSuffix(u.Path,"/")
		strippedURL += trimmedPath
	}

	return strippedURL,nil
}