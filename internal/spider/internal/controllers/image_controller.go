package controllers

import (
	"spider/internal/database"
	"spider/internal/utils"
	"spider/internal/crawler"
	"log"
	"time"
)

// ImageController handles persisting extracted image metadata to Redis.
// Each image is stored as a hash with a 1-hour TTL, and a reverse index
// maps pages to their images via page_images:<url> sets.
type ImageController struct {
	db *database.Database
}

// NewImageController creates a new ImageController with the given database connection.
func NewImageController(db *database.Database) *ImageController {
	return &ImageController{
		db: db,
	}
}

// SaveImages flushes all extracted images from the batch to Redis using a pipeline.
// For each image, it writes:
//   - image_data:<source_url> hash with page_url and alt fields (1h TTL)
//   - page_images:<page_url> set with the image source URL
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
