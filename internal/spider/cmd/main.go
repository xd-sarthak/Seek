package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"spider/internal/controllers"
	"spider/internal/crawler"
	"spider/internal/database"
	"spider/internal/pages"
	"spider/internal/ratelimiter"
	"spider/internal/robotstxt"
	"spider/internal/utils"
	"sync"
	"syscall"
)

// getEnv retrieves the value of an environment variable or returns a fallback value if not set.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return fallback
}

func main() {
	// Parse the flags
	maxConcurrency := flag.Int("max-concurrency", 10, "Maximum number of concurrent workers")
	maxPages := flag.Int("max-pages", 100, "Maximum number of pages per batch")
	flag.Parse()

	// Get the environment variables
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnv("REDIS_DB", "0")
	startingURL := getEnv("STARTING_URL", "https://en.wikipedia.org/wiki/Kamen_Rider")
	allowedDomains := utils.ParseAllowedDomains(getEnv("ALLOWED_DOMAINS", ""))

	// Set up graceful shutdown via context
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Connect to redis
	db := &database.Database{}
	err := db.ConnectToRedis(redisHost, redisPort, redisPassword, redisDB)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	allowed, err := utils.IsURLAllowed(startingURL, allowedDomains)
	if err != nil {
		log.Printf("Invalid STARTING_URL: %v\n", err)
		return
	}

	if !allowed {
		log.Printf("STARTING_URL host is not in ALLOWED_DOMAINS: %v\n", startingURL)
		return
	}

	// Add an entry to message queue
	err = db.PushURL(startingURL, 0)
	if err != nil {
		log.Printf("Error seeding crawler: %v\n", err)
		return
	}
	log.Printf("PUSH %v\n", startingURL)

	// Instantiate robots.txt checker and rate limiter
	robots := robotstxt.NewRobotsChecker("SearchEngineSpider/1.0")
	limiter := ratelimiter.NewDomainLimiter(utils.DefaultRatePerSecond, utils.DefaultBurstSize)

	// Instantiate the controllers
	pageController := controllers.NewPageController(db)
	linksController := controllers.NewLinksController(db)
	imageController := controllers.NewImageController(db)

	// Instantiate crawler
	crawlerCfg := &crawler.CrawlerConfig{
		Mu:             &sync.Mutex{},
		Wg:             &sync.WaitGroup{},
		Pages:          make(map[string]*pages.Page),
		Outlinks:       make(map[string]*pages.PageNode),
		Backlinks:      make(map[string]*pages.PageNode),
		Images:         make(map[string][]*pages.Image),
		AllowedDomains: allowedDomains,
		MaxPages:       *maxPages,
		MaxConcurrency: *maxConcurrency,
	}

	// Infinite loop to keep the crawler running
	for {
		// Check if shutdown has been requested
		select {
		case <-ctx.Done():
			log.Printf("Shutdown signal received, exiting...\n")
			return
		default:
		}

		// Check how many urls are in the indexer queue
		log.Printf("checking number of entries...\n")
		queueSize, err := db.GetIndexerQueueSize(ctx)

		if err != nil {
			if ctx.Err() != nil {
				log.Printf("Shutdown during queue size check\n")
				break
			}
			log.Printf("Error getting indexer queue: %v\n", err)
			return
		}

		if queueSize >= utils.MaxIndexerQueueSize {
			log.Printf("Indexer queue has %v entries, waiting...\n", queueSize)
			// Wait until we receive a signal that the indexer queue has been processed
			for {
				select {
				case <-ctx.Done():
					log.Printf("Shutdown signal received while waiting for indexer\n")
					return
				default:
				}

				sig, err := db.PopSignalQueue(ctx)
				if err != nil {
					if ctx.Err() != nil {
						log.Printf("Shutdown during signal wait\n")
						return
					}
					log.Printf("Error popping signal queue: %v\n", err)
					return
				}

				if sig == utils.ResumeCrawl {
					log.Printf("Received RESUME_CRAWL signal, resuming...\n")
					break
				}
			}
		}

		log.Printf("Spawning workers...\n")
		for range crawlerCfg.MaxConcurrency {
			crawlerCfg.Wg.Add(1)
			go crawlerCfg.Crawl(ctx, db, robots, limiter)
		}

		crawlerCfg.Wg.Wait()

		// Write entries to the database (always flush, even on shutdown)
		log.Printf("Flushing batch data to Redis...\n")
		pageController.SavePages(crawlerCfg)
		linksController.SaveLinks(crawlerCfg)
		imageController.SaveImages(crawlerCfg)

		// Clean visited pages by this runner
		crawlerCfg.Pages = make(map[string]*pages.Page)
		crawlerCfg.Outlinks = make(map[string]*pages.PageNode)
		crawlerCfg.Backlinks = make(map[string]*pages.PageNode)
		crawlerCfg.Images = make(map[string][]*pages.Image)

		// After flushing, if shutdown was requested, exit cleanly
		if ctx.Err() != nil {
			log.Printf("Batch flushed. Shutting down gracefully.\n")
			return
		}
	}
}
