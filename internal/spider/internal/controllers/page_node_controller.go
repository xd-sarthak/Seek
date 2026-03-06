package controllers

import (
	"spider/internal/database"
	"spider/internal/utils"
	"spider/internal/crawler"
	"log"
)

type LinksController struct {
	db *database.Database
}

func NewLinksController(db *database.Database) *LinksController {
	return &LinksController{
		db: db,
	}
}

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


