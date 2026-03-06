package main

import (
	"flag"
	"log"
	"os"
	"spider/internal/controllers"
	"spider/internal/crawler"
	"spider/internal/database"
	"spider/internal/pages"
	"spider/internal/utils"
	"sync"
)

// getEnv retrieves the value of an environment variable or returns a fallback value if not set.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return fallback
}

func main(){
	//parse the flags 
	maxConcurrency := flag.Int("max-concurrency", 10, "Maximum number of concurrent workers")
	maxPages := flag.Int("max-pages", 100, "Maximum number of pages per batch")
	flag.Parse()

	//get the environment variables
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnv("REDIS_DB", "0")
	startingURL := getEnv("STARTING_URL", "https://en.wikipedia.org/wiki/Kamen_Rider")

	//connect to redis
	db := &database.Database{}
	err := db.ConnectToRedis(redisHost, redisPort, redisPassword, redisDB)
	if err != nil {
		log.Printf("Error: %v\n",err)
		return
	}

	//Add an entry to message queue
	db.PushURL(startingURL,0)
	log.Printf("PUSH %v\n", startingURL)

	//instantiate the controllers
	pageController := controllers.NewPageController(db)
	linksController := controllers.NewLinksController(db)
	imageController := controllers.NewImageController(db)

	
	// Instantiate crawler
	crawler := &crawler.CrawlerConfig{
		Mu:             &sync.Mutex{},
		Wg:             &sync.WaitGroup{},
		Pages:          make(map[string]*pages.Page),
		Outlinks:       make(map[string]*pages.PageNode),
		Backlinks:      make(map[string]*pages.PageNode),
		Images:         make(map[string][]*pages.Image),
		MaxPages:       *maxPages,
		MaxConcurrency: *maxConcurrency,
	}

	//infinite loop to keep the crawler running
	for {
		//check how many urls are in the indexer queue
		log.Printf("checking number of entries...\n")
		queueSize,err := db.GetIndexerQueueSize()

		if err != nil {
			log.Printf("Error getting indexer queue: %v\n",err)
			return
		}

		if queueSize >= utils.MaxIndexerQueueSize {
			log.Printf("Indexer queue has %v entries, waiting...\n",queueSize)
			//wait until we receive a signal that the indexer queue has been processed
			for {
				sig, err := db.PopSignalQueue()
				if err != nil {
					log.Printf("Error popping signal queue: %v\n",err)
					return
				}

				if sig == utils.ResumeCrawl {
					log.Printf("could not get signal: %v\n", err)
					break
				}
			}
		}

		log.Printf("Spawning workers...\n")
		for range crawler.MaxConcurrency {
			crawler.Wg.Add(1)
			go crawler.Crawl(db)
		}

		crawler.Wg.Wait()

		//write entries to the database
		pageController.SavePages(crawler)
		linksController.SaveLinks(crawler)
		imageController.SaveImages(crawler)

		// Clean visited pages by this runner
		crawler.Pages = make(map[string]*pages.Page)
		crawler.Outlinks = make(map[string]*pages.PageNode)
		crawler.Backlinks = make(map[string]*pages.PageNode)
		crawler.Images = make(map[string][]*pages.Image)
	}
}
