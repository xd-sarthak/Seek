package pages

import (
	"time"
)

// Page represents a crawled web page with its content and metadata.
// Pages are collected in-memory during a batch cycle and flushed to
// Redis via PageController.SavePages.
type Page struct {
	NormalizedURL string    // Canonical URL after normalization
	HTML          string    // Full HTML body (up to 10 MB)
	ContentType   string    // HTTP Content-Type header (always "text/html")
	StatusCode    int       // HTTP response status code
	LastCrawled   time.Time // Timestamp when the page was fetched
}

// CreatePage constructs a new Page with the current time as LastCrawled.
func CreatePage(normalizedUrl, html, contentType string, statusCode int) *Page {
    return &Page {
        NormalizedURL:  normalizedUrl,
        HTML:           html,
        ContentType:    contentType,
        StatusCode:     statusCode,
        LastCrawled:    time.Now(),
    }
}

// HashPage converts a Page into a map suitable for Redis HSET.
// The LastCrawled field is formatted as RFC 1123.
func HashPage(page *Page) (map[string]interface{}, error) {
    return map[string]interface{}{
        "normalized_url":   page.NormalizedURL,
        "html":             page.HTML,
        "content_type":     page.ContentType,
        "status_code":      page.StatusCode,
        "last_crawled":     page.LastCrawled.Format(time.RFC1123),
    }, nil
}
