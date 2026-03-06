package crawler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// getPageData fetches the HTML, status code, and content type of a given URL
func getPageData(rawURL string) (string, int, string, error) {
    res, err := http.Get(rawURL)

    if err != nil {
        return "", 0, "", fmt.Errorf("failed to fetch URL: %w", err)
    }

    defer res.Body.Close() // Close the body to prevent memory leaks or something I don't remember

    if res.StatusCode > 399 {
        return "", res.StatusCode, "", fmt.Errorf("HTTP error: %d %s", res.StatusCode, http.StatusText(res.StatusCode))
    }

    contentType := res.Header.Get("Content-Type")
    if !strings.HasPrefix(contentType, "text/html") {
        return "", res.StatusCode, contentType, fmt.Errorf("invalid content type: %s", contentType)
    }

    body, err := io.ReadAll(res.Body)

    if err != nil {
        return "", res.StatusCode, "text/html", fmt.Errorf("failed to read response body: %w", err)
    }

    return string(body), res.StatusCode, "text/html", nil
}