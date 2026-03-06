package controllers

import (
	"log"
	"spider/internal/crawler"
	"spider/internal/database"
	"spider/internal/pages"
	"spider/internal/utils"
)

type PageController struct {
	db *database.Database
}

func NewPageController(db *database.Database) *PageController {
	return &PageController{
		db: db,
	}
}

//saviing the page data to redis
func (pc *PageController) SavePages(crawcfg *crawler.CrawlerConfig){

	data := crawcfg.Pages
	log.Printf("Writing %d entries to the db...\n", len(data))

	// Process the redis entries using a pipeline
    pipeline := pc.db.Client.Pipeline()

	for _,page := range data {
		pageHash,err := pages.HashPage(page)
		if err != nil {
			log.Printf("Error hashing page %s: %v", page.NormalizedURL, err)
            continue
		}

		// Append command to pipeline
        pipeline.HSet(pc.db.Context, utils.PagePrefix + ":"+page.NormalizedURL, pageHash)

        // Push to the indexer queue
        // NOTE: For some weird reason "indexer_queue" does not work, but any other name does :/
        // res, err := pgc.db.Client.LPush(pgc.db.Context, utils.IndexerQueueKey, utils.PagePrefix + ":" +page.NormalizedURL).Result()
        pc.db.Client.LPush(pc.db.Context, utils.IndexerQueueKey, utils.PagePrefix + ":" +page.NormalizedURL).Result()

	}

	// Execute the pipeline
	_, err := pipeline.Exec(pc.db.Context)
    if err != nil {
        log.Printf("Error executing pipeline: %v", err)
    } else {
        log.Printf("Successfully written %d entries to the db!", len(data))
    }
}