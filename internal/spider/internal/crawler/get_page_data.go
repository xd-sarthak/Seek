package crawler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// maxResponseBodySize is the maximum number of bytes to read from a response body (10MB).
const maxResponseBodySize = 10 * 1024 * 1024

// httpClient is a shared HTTP client with proper timeouts to prevent goroutine leaks.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	},
}

// getPageData fetches a web page and returns its HTML body, HTTP status code,
// content type, and any error encountered.
//
// The function enforces several safety constraints:
//   - 30-second total request timeout
//   - 10 MB response body limit (prevents OOM)
//   - Only text/html content types are accepted
//   - HTTP 4xx/5xx responses are treated as errors
//   - User-Agent is set to "SearchEngineSpider/1.0"
//
// Returns:
//   - html: the page body as a string (empty on error)
//   - statusCode: HTTP status code (0 if request failed entirely)
//   - contentType: the Content-Type header value
//   - err: non-nil if the page could not be fetched or is not HTML
func getPageData(rawURL string) (string, int, string, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "SearchEngineSpider/1.0")
	req.Header.Set("Accept", "text/html")

	res, err := httpClient.Do(req)

	if err != nil {
		return "", 0, "", fmt.Errorf("failed to fetch URL: %w", err)
	}

	defer res.Body.Close()

	if res.StatusCode > 399 {
		return "", res.StatusCode, "", fmt.Errorf("HTTP error: %d %s", res.StatusCode, http.StatusText(res.StatusCode))
	}

	contentType := res.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		return "", res.StatusCode, contentType, fmt.Errorf("invalid content type: %s", contentType)
	}

	// Limit response body to prevent OOM from very large pages
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBodySize))

	if err != nil {
		return "", res.StatusCode, "text/html", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), res.StatusCode, "text/html", nil
}