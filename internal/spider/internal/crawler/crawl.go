package crawler

import (
	"log"
	"math"
	"spider/internal/pages"
	"spider/internal/utils"
	"spider/internal/database"
)

// Crawl runs the BFS crawl loop for a single worker goroutine.
// Each invocation pops URLs from the Redis frontier, fetches pages,
// extracts links and images, and stores results in the shared CrawlerConfig.
//
// The worker exits when:
//   - MaxPages is reached for the current batch
//   - The frontier queue is empty (BZPopMin times out)
//
// Steps per iteration:
//  1. Check batch page limit
//  2. Pop next URL from Redis sorted set (BZPopMin)
//  3. Check deduplication (Redis visited hash)
//  4. Fetch HTML via HTTP GET
//  5. Parse HTML and extract links + images
//  6. Store images and update link graph in CrawlerConfig
//  7. Mark URL as visited in Redis
//  8. Enqueue discovered URLs with depth-based scores
func (crawcfg *CrawlerConfig) Crawl(db *database.Database) {
	//starting a new webcrawler
	defer crawcfg.Wg.Done()

	//bfs loop
	for {
		log.Printf("Crawling...\n")

		//check if we have reached the maximum number of pages
		if crawcfg.maxPagesReached() {
			log.Printf("Max pages reached, waiting for workers to finish...\n")
			return
		}

		//get the next url to crawl from the message queue
		log.Printf("Waiting for message queue...\n")
		rawCurrentURL, depthLevel, normalizedCurrentURL, err := db.PopURL()
		if err != nil {
			log.Printf("No more URLs in the queue: %v\n", err)
			return
		}

		visited, err := db.HasURLBeenVisited(normalizedCurrentURL)
		if err != nil {
			log.Printf("Error: [%v] - skipping...\n", err)
			continue
		}

		if visited {
			log.Printf("Skipping %v - already visited\n", normalizedCurrentURL)
			continue
		}

		log.Printf("Crawling %v (depth: %v)\n", normalizedCurrentURL, depthLevel)

		//Fetch HTML, Status Code, and content type
		html, statusCode, contentType, err := getPageData(rawCurrentURL)
		if err != nil {
			// Skip if we couldn't fetch the data
			log.Printf("Error fetching %v data: %v\n", rawCurrentURL, err)
			continue
		}

		// Fetch the links of the current page
		outgoingLinks, imagesMap, err := getURLsFromHTML(html, rawCurrentURL)
		if err != nil {
			log.Printf("Error getting links from HTML: %v\n", err)
			continue
		}

		// Store images
		crawcfg.AddImages(normalizedCurrentURL, imagesMap)

		// Create outlinks and update backlinks
		crawcfg.UpdateLinks(normalizedCurrentURL, outgoingLinks)

		//create page struct
		pg := pages.CreatePage(normalizedCurrentURL, html, contentType, statusCode)

		// Add page visit
		err = crawcfg.addPage(pg)
		if err != nil {
			log.Printf("\tError adding page visit: %v\n", err)
			continue
		}

		err = db.VisitPage(normalizedCurrentURL)
		if err != nil {
			log.Printf("\tError adding page visit: %v\n", err)
			continue
		}


		log.Printf("Adding links from %v (%v)...\n", normalizedCurrentURL, rawCurrentURL)

		
		// Add links to url queue
		for _, rawCurrentLink := range outgoingLinks {
			// Check if the url is valid
			if !utils.IsValidURL(rawCurrentLink) {
				// If it's not valid, process the next link
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