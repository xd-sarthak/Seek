package crawler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"strings"
	"time"

	"spider/internal/utils"
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

// isRetryable returns true if the error or status code indicates the request
// should be retried (server errors, timeouts, connection issues).
func isRetryable(err error, statusCode int) bool {
	// Server errors are retryable
	if statusCode >= 500 {
		return true
	}

	if err == nil {
		return false
	}

	// Timeout errors
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// Connection refused / reset
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// Context deadline exceeded (but not cancelled — that means shutdown)
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	return false
}

// backoffDuration calculates the sleep duration for a retry attempt using
// exponential backoff with jitter: InitialBackoff * 2^attempt * (1 + rand(0, 0.25))
func backoffDuration(attempt int) time.Duration {
	backoff := utils.InitialBackoff * (1 << attempt)
	if backoff > utils.MaxBackoff {
		backoff = utils.MaxBackoff
	}

	// Add up to 25% jitter
	jitter := time.Duration(rand.Float64() * 0.25 * float64(backoff))
	return backoff + jitter
}

// getPageDataWithRetry wraps getPageData with exponential backoff retry logic.
// It retries on server errors (5xx) and transient network errors, but NOT on
// client errors (4xx) or content-type mismatches.
//
// Returns immediately if the context is cancelled (graceful shutdown).
func getPageDataWithRetry(ctx context.Context, rawURL string) (string, int, string, error) {
	var lastErr error

	for attempt := range utils.MaxRetries {
		// Check if context has been cancelled (graceful shutdown)
		select {
		case <-ctx.Done():
			return "", 0, "", fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		html, statusCode, contentType, err := getPageData(ctx, rawURL)
		if err == nil {
			return html, statusCode, contentType, nil
		}

		lastErr = err

		// Don't retry client errors (4xx) or content-type mismatches
		if !isRetryable(err, statusCode) {
			return html, statusCode, contentType, err
		}

		// Don't sleep after the last attempt
		if attempt < utils.MaxRetries-1 {
			sleepDur := backoffDuration(attempt)
			log.Printf("[retry] Attempt %d/%d failed for %s: %v (retrying in %v)",
				attempt+1, utils.MaxRetries, rawURL, err, sleepDur)

			select {
			case <-ctx.Done():
				return "", 0, "", fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
			case <-time.After(sleepDur):
			}
		}
	}

	return "", 0, "", fmt.Errorf("all %d attempts failed for %s: %w", utils.MaxRetries, rawURL, lastErr)
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
func getPageData(ctx context.Context, rawURL string) (string, int, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
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