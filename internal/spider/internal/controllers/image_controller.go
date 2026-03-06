package controllers

import (
	"spider/internal/database"
	"spider/internal/utils"
	"spider/internal/crawler"
	"log"
	"time"
)

type ImageController struct {
	db *database.Database
}

func NewImageController(db *database.Database) *ImageController {
	return &ImageController{
		db: db,
	}
}

func (pgc *ImageController) SaveImages(crawcfg *crawler.CrawlerConfig) {
	pipeline := pgc.db.Client.Pipeline()

	log.Printf("Saving images...\n")
	data := crawcfg.Images
	count := 0

	for normalizedURL, imageData := range data {
		for _, image := range imageData {
			imageKey := utils.ImagePrefix + ":" + image.NormalizedSourceURL
			pipeline.HSet(pgc.db.Context, imageKey, map[string]interface{}{
				"page_url": image.NormalizedPageURL,
				"alt":      image.Alt,
			})

			pipeline.Expire(pgc.db.Context, imageKey, 1*time.Hour) // 1 hour TTL

			count += 1

			// Store the image under a page image set
			pageImagesKey := utils.PageImagesPrefix + ":" + normalizedURL
			pipeline.SAdd(pgc.db.Context, pageImagesKey, image.NormalizedSourceURL)
		}
	}

	_, err := pipeline.Exec(pgc.db.Context)
	if err != nil {
		log.Printf("Error saving images: %v\n", err)
	} else {
		log.Printf("Successfully written %d entries to the db!", count)
	}
}
