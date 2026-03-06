package pages

import (
    // "fmt"
)

// Image represents an image extracted from a crawled page.
// Each image records the page it was found on, its normalized source URL,
// and its alt text (if present). Images are stored in Redis with a 1-hour TTL.
type Image struct {
    NormalizedPageURL   string // Canonical URL of the page containing this image
    NormalizedSourceURL string // Canonical URL of the image source
    Alt                 string // Alt text from the <img> tag (may be empty)
}