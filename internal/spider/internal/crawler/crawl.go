package crawler

import (
	"context"
	"log"
	"math"
	"spider/internal/database"
	"spider/internal/pages"
	"spider/internal/ratelimiter"
	"spider/internal/robotstxt"
	"spider/internal/utils"
)

// Crawl runs the BFS crawl loop for a single worker goroutine.
// Each invocation pops URLs from the Redis frontier, fetches pages,
// extracts links and images, and stores results in the shared CrawlerConfig.
//
// The worker exits when:
//   - MaxPages is reached for the current batch
//   - The frontier queue is empty (BZPopMin times out)
//   - The context is cancelled (graceful shutdown)
//
// Steps per iteration:
//  1. Check context cancellation (graceful shutdown)
//  2. Check batch page limit
//  3. Pop next URL from Redis sorted set (BZPopMin)
//  4. Check deduplication (Redis visited hash)
//  5. Check robots.txt compliance
//  6. Wait for per-domain rate limiter
//  7. Fetch HTML via HTTP GET with retry
//  8. Parse HTML and extract links + images + meta directives
//  9. Respect noindex/nofollow directives
//  10. Store images and update link graph in CrawlerConfig
//  11. Mark URL as visited in Redis
//  12. Enqueue discovered URLs with depth-based scores
func (crawcfg *CrawlerConfig) Crawl(ctx context.Context, db *database.Database, robots *robotstxt.RobotsChecker, limiter *ratelimiter.DomainLimiter) {
	defer crawcfg.Wg.Done()

	for {
		// Check if shutdown has been requested
		select {
		case <-ctx.Done():
			log.Printf("Worker shutting down: %v\n", ctx.Err())
			return
		default:
		}

		log.Printf("Crawling...\n")

		// Check if we have reached the maximum number of pages
		if crawcfg.maxPagesReached() {
			log.Printf("Max pages reached, waiting for workers to finish...\n")
			return
		}

		// Get the next url to crawl from the message queue
		log.Printf("Waiting for message queue...\n")
		rawCurrentURL, depthLevel, normalizedCurrentURL, err := db.PopURL(ctx)
		if err != nil {
			// Context cancelled or queue timeout
			if ctx.Err() != nil {
				log.Printf("Worker shutting down during PopURL: %v\n", ctx.Err())
				return
			}
			log.Printf("No more URLs in the queue: %v\n", err)
			return
		}

		visited, err := db.HasURLBeenVisited(ctx, normalizedCurrentURL)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("Error: [%v] - skipping...\n", err)
			continue
		}

		if visited {
			log.Printf("Skipping %v - already visited\n", normalizedCurrentURL)
			continue
		}

		// Check robots.txt before fetching
		if !robots.IsAllowed(rawCurrentURL) {
			log.Printf("Blocked by robots.txt: %v\n", normalizedCurrentURL)
			continue
		}

		// Honor Crawl-delay from robots.txt by updating the rate limiter
		if crawlDelay := robots.GetCrawlDelay(rawCurrentURL); crawlDelay > 0 {
			limiter.SetDomainRate(rawCurrentURL, crawlDelay)
		}

		// Wait for rate limiter
		if err := limiter.Wait(ctx, rawCurrentURL); err != nil {
			if ctx.Err() != nil {
				log.Printf("Worker shutting down during rate limit: %v\n", ctx.Err())
				return
			}
			log.Printf("Rate limiter error for %v: %v\n", rawCurrentURL, err)
			continue
		}

		log.Printf("Crawling %v (depth: %v)\n", normalizedCurrentURL, depthLevel)

		// Fetch HTML with retry logic
		html, statusCode, contentType, err := getPageDataWithRetry(ctx, rawCurrentURL)
		if err != nil {
			if ctx.Err() != nil {
				log.Printf("Worker shutting down during fetch: %v\n", ctx.Err())
				return
			}
			log.Printf("Error fetching %v data: %v\n", rawCurrentURL, err)
			continue
		}

		// Fetch the links of the current page + meta directives
		outgoingLinks, imagesMap, directives, err := getURLsFromHTML(html, rawCurrentURL)
		if err != nil {
			log.Printf("Error getting links from HTML: %v\n", err)
			continue
		}

		// Respect noindex: don't store this page
		if directives.NoIndex {
			log.Printf("Skipping storage of %v (noindex directive)\n", normalizedCurrentURL)
			// Still mark as visited to avoid re-crawling
			_ = db.VisitPage(ctx, normalizedCurrentURL)
			continue
		}

		// Store images
		crawcfg.AddImages(normalizedCurrentURL, imagesMap)

		// Create outlinks and update backlinks
		crawcfg.UpdateLinks(normalizedCurrentURL, outgoingLinks)

		// Create page struct
		pg := pages.CreatePage(normalizedCurrentURL, html, contentType, statusCode)

		// Add page visit
		err = crawcfg.addPage(pg)
		if err != nil {
			log.Printf("\tError adding page visit: %v\n", err)
			continue
		}

		err = db.VisitPage(ctx, normalizedCurrentURL)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("\tError adding page visit: %v\n", err)
			continue
		}

		// Respect nofollow: don't enqueue discovered links
		if directives.NoFollow {
			log.Printf("Skipping link enqueue from %v (nofollow directive)\n", normalizedCurrentURL)
			continue
		}

		log.Printf("Adding links from %v (%v)...\n", normalizedCurrentURL, rawCurrentURL)

		// Add links to url queue
		for _, rawCurrentLink := range outgoingLinks {
			// Check if the url is valid
			if !utils.IsValidURL(rawCurrentLink) {
				continue
			}

			// Check if the URL already exists in the queue
			score, exists := db.ExistsInQueue(rawCurrentLink)
			if !exists {
				// New URL: assign score based on depth level
				score = depthLevel + 1
			}

			score = math.Max(utils.MinScore, math.Min(score, utils.MaxScore))

			// Update score based on depth
			_ = db.PushURL(rawCurrentLink, score)
		}
	}
}