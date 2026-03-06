package crawler

import (
	"spider/internal/pages"
	"spider/internal/utils"
	"sync"
	"fmt"
)

// CrawlerConfig holds the shared state for a single crawl batch.
// All map fields are protected by Mu and must only be accessed under lock.
// When Pages reaches MaxPages, workers stop crawling, the batch is flushed
// to Redis via controllers, and the maps are reset for the next cycle.
type CrawlerConfig struct {
	Mu             *sync.Mutex                // Protects all map fields below
	Wg             *sync.WaitGroup            // Tracks active worker goroutines
	Pages          map[string]*pages.Page     // Pages crawled in this batch, keyed by normalized URL
	Outlinks       map[string]*pages.PageNode // Outgoing links per page, keyed by source URL
	Backlinks      map[string]*pages.PageNode // Incoming links per page, keyed by target URL
	Images         map[string][]*pages.Image  // Images per page, keyed by page URL
	MaxPages       int                        // Maximum pages to crawl per batch cycle
	MaxConcurrency int                        // Number of concurrent worker goroutines
}

func (crawcfg *CrawlerConfig) lenPages () int {
	crawcfg.Mu.Lock()
	defer crawcfg.Mu.Unlock()

	return len(crawcfg.Pages)
}

func (crawcfg *CrawlerConfig) maxPagesReached() (bool) {
    crawcfg.Mu.Lock()
    defer crawcfg.Mu.Unlock()

    if len(crawcfg.Pages) >= crawcfg.MaxPages {
        // Can't add more pages because max pages has been reached
        return true
    }

    // Max pages has not been reached
    return false
}



// AddImages extracts image metadata from the raw imagesMap and stores
// them in the batch under the given page URL. Thread-safe via Mu.
//
// imagesMap keys are normalized image source URLs; values contain
// "src" and optionally "alt" entries.
func (crawcfg* CrawlerConfig) AddImages(normalizedCurrentURL string, imagesMap map[string]map[string]string) {
    crawcfg.Mu.Lock()
    defer crawcfg.Mu.Unlock()

    for imgURL, imgAttrs := range imagesMap {
        imgAlt := ""
        if alt, exists := imgAttrs["alt"]; exists {
            imgAlt = alt
        }

        image := &pages.Image {
            NormalizedPageURL:   normalizedCurrentURL,
            NormalizedSourceURL: imgURL,
            Alt:                 imgAlt,

        }

        crawcfg.Images[normalizedCurrentURL] = append(crawcfg.Images[normalizedCurrentURL], image)
    }
}


// UpdateLinks builds the outlink and backlink graph for the current page.
// For each valid outgoing link, it creates an outlink entry on the current
// page and a backlink entry on the target page. Self-links are skipped.
// Thread-safe via Mu.
func (crawcfg *CrawlerConfig) UpdateLinks(normalizedCurrentURL string, outgoingLinks []string) {
    crawcfg.Mu.Lock()
    defer crawcfg.Mu.Unlock()

    crawcfg.Outlinks[normalizedCurrentURL] = pages.CreatePageNode(normalizedCurrentURL)
    for _, link := range outgoingLinks {
        if utils.IsValidURL(link) {
            // normalize url
            normalizedOutgoingURL, err := utils.NormalizeURL(link)
            if err != nil {
                continue
            }

            if normalizedOutgoingURL == normalizedCurrentURL {
                continue
            }

            // If the entry does not exist
            if _, exists := crawcfg.Backlinks[normalizedOutgoingURL]; !exists {
                crawcfg.Backlinks[normalizedOutgoingURL] = pages.CreatePageNode(normalizedOutgoingURL)
            }

            crawcfg.Backlinks[normalizedOutgoingURL].AppendLink(normalizedCurrentURL)
            crawcfg.Outlinks[normalizedCurrentURL].AppendLink(normalizedOutgoingURL)
        }
    }
}


func (crawcfg *CrawlerConfig) addPage(page *pages.Page) error {
    crawcfg.Mu.Lock()
    defer crawcfg.Mu.Unlock()

    normalizedURL := page.NormalizedURL

    if _, visited := crawcfg.Pages[normalizedURL]; visited {
        return fmt.Errorf("Page already visited")
    }

    if len(crawcfg.Pages) >= crawcfg.MaxPages {
        // Can't add more pages because max pages has been reached
        return fmt.Errorf("Max pages reached")
    }

    crawcfg.Pages[normalizedURL] = page
    return nil
}