package pages

import (
	"time"
)

type Page struct {
	NormalizedURL string
	HTML          string
	ContentType   string
	StatusCode    int
	LastCrawled   time.Time
}

func CreatePage(normalizedUrl, html, contentType string, statusCode int) *Page {
    return &Page {
        NormalizedURL:  normalizedUrl,
        HTML:           html,
        ContentType:    contentType,
        StatusCode:     statusCode,
        LastCrawled:    time.Now(),
    }
}

func HashPage(page *Page) (map[string]interface{}, error) {
    // Convert it to a redis hash
    return map[string]interface{}{
        "normalized_url":   page.NormalizedURL,
        "html":             page.HTML,
        "content_type":     page.ContentType,
        "status_code":      page.StatusCode,
        "last_crawled":     page.LastCrawled.Format(time.RFC1123),
    }, nil
}
