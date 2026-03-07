package controllers

import (
	"context"
	"log"
	"spider/internal/crawler"
	"spider/internal/database"
	"spider/internal/pages"
	"spider/internal/utils"
)

// PageController handles persisting crawled page data to Redis.
// It converts Page structs to Redis hashes and pushes page keys
// to the indexer queue for downstream processing.
type PageController struct {
	db *database.Database
}

// NewPageController creates a new PageController with the given database connection.
func NewPageController(db *database.Database) *PageController {
	return &PageController{
		db: db,
	}
}

// SavePages flushes all crawled pages from the batch to Redis using a pipeline.
// For each page, it writes a Redis hash (page_data:<url>) and pushes the key
// to the pages_queue list for the indexer to consume.
func (pc *PageController) SavePages(crawcfg *crawler.CrawlerConfig) {

	data := crawcfg.Pages
	log.Printf("Writing %d entries to the db...\n", len(data))

	ctx := context.Background()

	// Process the redis entries using a pipeline
	pipeline := pc.db.Client.Pipeline()

	for _, page := range data {
		pageHash, err := pages.HashPage(page)
		if err != nil {
			log.Printf("Error hashing page %s: %v", page.NormalizedURL, err)
			continue
		}

		// Append commands to pipeline
		pipeline.HSet(ctx, utils.PagePrefix+":"+page.NormalizedURL, pageHash)
		pipeline.LPush(ctx, utils.IndexerQueueKey, utils.PagePrefix+":"+page.NormalizedURL)

	}

	// Execute the pipeline
	_, err := pipeline.Exec(ctx)
	if err != nil {
		log.Printf("Error executing pipeline: %v", err)
	} else {
		log.Printf("Successfully written %d entries to the db!", len(data))
	}
}