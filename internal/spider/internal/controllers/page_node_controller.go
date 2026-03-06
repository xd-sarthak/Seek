package controllers

import (
	"spider/internal/database"
	"spider/internal/utils"
	"spider/internal/crawler"
	"log"
)

// LinksController handles persisting link graph data (backlinks and outlinks)
// to Redis. Uses SADD to store directional links as Redis sets.
type LinksController struct {
	db *database.Database
}

// NewLinksController creates a new LinksController with the given database connection.
func NewLinksController(db *database.Database) *LinksController {
	return &LinksController{
		db: db,
	}
}

// SaveLinks flushes all backlink and outlink data from the batch to Redis
// using a pipeline. For each page node:
//   - backlinks:<url> set stores all pages that link TO this URL
//   - outlinks:<url> set stores all pages that this URL links TO
func (pgc *LinksController) SaveLinks(crawcfg *crawler.CrawlerConfig) {
    pipeline := pgc.db.Client.Pipeline()

    log.Printf("Saving backlinks...\n")
    data := crawcfg.Backlinks
    count := len(data)
    for key, backlinks := range data {
        for _, link := range backlinks.GetLinks() {
            pipeline.SAdd(pgc.db.Context, utils.BacklinksPrefix + ":" + key, link)
        }
    }

    log.Printf("Saving outlinks...\n")
    data = crawcfg.Outlinks
    count += len(data)
    for key, outlinks := range data {
        for _, link := range outlinks.GetLinks() {
            pipeline.SAdd(pgc.db.Context, utils.OutlinksPrefix + ":" + key, link)
        }
    }

    _, err := pipeline.Exec(pgc.db.Context)
    if err != nil {
        log.Printf("Error executing pipeline: %v", err)
    } else {
        log.Printf("Successfully written %d entries to the db!", count)
    }
}


