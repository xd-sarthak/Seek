package utils

import (
	"time"
)

const (
	// Crawler constants
	Timeout  = 5 * time.Second
	MaxScore = 10000
	MinScore = -1000

	// Default crawl scope
	DefaultAllowedDomains = "github.com,raw.githubusercontent.com,stackoverflow.com,developer.mozilla.org,docs.python.org,react.dev,nodejs.org,golang.org,docs.docker.com,kubernetes.io,pkg.go.dev,docs.rs"

	// Retry constants
	MaxRetries     = 3
	InitialBackoff = 500 * time.Millisecond
	MaxBackoff     = 8 * time.Second

	// Rate limiter constants
	DefaultRatePerSecond = 2.0
	DefaultBurstSize     = 1

	// Message Queues
	SpiderQueueKey      = "spider_queue"
	IndexerQueueKey     = "pages_queue"
	SignalQueueKey      = "signal_queue"
	ResumeCrawl         = "RESUME_CRAWL"
	MaxIndexerQueueSize = 5000

	// Redis Data: some keys stay in Redis indefinitely, while others are transfer to MongoDB by other services
	NormalizedURLPrefix = "normalized_url" // Stays in Redis indefinitely
	PagePrefix          = "page_data"      // Transferred by the indexer
	ImagePrefix         = "image_data"     // Transferred by the image indexer
	PageImagesPrefix    = "page_images"    // Transferred by the image indexer
	BacklinksPrefix     = "backlinks"      // Transferred by the backlinks processor
	OutlinksPrefix      = "outlinks"       // Transferred by the indexer
)
