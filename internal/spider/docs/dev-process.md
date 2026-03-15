# Development Process

This document covers local development setup, environment configuration, and testing for the Go Search Engine Spider.

---

## Prerequisites

| Dependency | Version | Purpose                              |
|------------|---------|--------------------------------------|
| Go         | 1.25+   | Compiler and runtime                 |
| Redis      | 7+      | URL frontier, data storage, queues   |

---

## Local Setup

### 1. Start Redis

```bash
# Using Docker
docker run -d --name redis -p 6379:6379 redis:7-alpine

# Or using system Redis
redis-server
```

### 2. Install Go Dependencies

```bash
cd internal/spider
go mod download
```

### 3. Run the Crawler

```bash
go run ./cmd/main.go
```

With custom settings:

```bash
go run ./cmd/main.go --max-concurrency=20 --max-pages=500
```

---

## Environment Variables

All configuration is done via environment variables with sensible defaults:

| Variable         | Default                                            | Description                                  |
|------------------|----------------------------------------------------|----------------------------------------------|
| `REDIS_HOST`     | `localhost`                                        | Redis server hostname                        |
| `REDIS_PORT`     | `6379`                                             | Redis server port                            |
| `REDIS_PASSWORD` | _(empty)_                                          | Redis authentication password                |
| `REDIS_DB`       | `0`                                                | Redis database index (0–15)                  |
| `STARTING_URL`   | `https://en.wikipedia.org/wiki/Kamen_Rider`        | Seed URL for the crawler                     |
| `ALLOWED_DOMAINS`| _(empty)_                                          | Comma-separated host allowlist for focused crawling |

### Example: Custom Redis and Seed URL

```bash
REDIS_HOST=redis.prod.internal \
REDIS_PORT=6380 \
REDIS_PASSWORD=secret \
STARTING_URL=https://example.com \
ALLOWED_DOMAINS=example.com,developer.mozilla.org \
go run ./cmd/main.go --max-concurrency=50 --max-pages=1000
```

---

## CLI Flags

| Flag                | Type | Default | Description                                             |
|---------------------|------|---------|---------------------------------------------------------|
| `--max-concurrency` | int  | `10`    | Number of concurrent goroutines in the worker pool      |
| `--max-pages`       | int  | `100`   | Maximum number of pages to crawl per batch cycle        |

---

## Build

```bash
cd internal/spider

# Build binary
go build -o spider ./cmd/main.go

# Run binary
./spider --max-concurrency=10 --max-pages=100
```

---

## Testing

```bash
cd internal/spider

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test -v ./internal/utils/...

# Run tests with race detector
go test -race ./...
```

---

## Redis Monitoring

Useful commands for monitoring crawler state during development:

```bash
# Monitor all Redis commands in real-time
redis-cli MONITOR

# Check frontier queue size
redis-cli ZCARD spider_queue

# View top 10 pending URLs (lowest score = next to crawl)
redis-cli ZRANGE spider_queue 0 9 WITHSCORES

# Check indexer queue size (backpressure metric)
redis-cli LLEN pages_queue

# Count visited URLs
redis-cli KEYS "normalized_url:*" | wc -l

# View a specific page's crawled data
redis-cli HGETALL "page_data:https://example.com/about"

# Check backlinks for a URL
redis-cli SMEMBERS "backlinks:https://example.com/about"

# Check outlinks for a URL
redis-cli SMEMBERS "outlinks:https://example.com/about"

# View images for a page
redis-cli SMEMBERS "page_images:https://example.com/about"

# Flush all data (CAUTION: destructive)
redis-cli FLUSHDB
```

---

## Project Structure

```
internal/spider/
│
├── cmd/
│   └── main.go                  # Entrypoint: config, workers, batch loop
│
├── internal/
│   ├── controllers/
│   │   ├── page_controllers.go  # Flush pages to Redis
│   │   ├── page_node_controller.go  # Flush backlinks/outlinks to Redis
│   │   └── image_controller.go  # Flush images to Redis
│   │
│   ├── crawler/
│   │   ├── crawler.go           # CrawlerConfig struct and shared-state methods
│   │   ├── crawl.go             # BFS crawl loop (per-worker)
│   │   ├── get_page_data.go     # HTTP fetch with timeouts and limits
│   │   └── get_urls_html.go     # HTML parsing, link and image extraction
│   │
│   ├── database/
│   │   └── redis_client.go      # Redis client: queue ops, visited tracking
│   │
│   ├── pages/
│   │   ├── page.go              # Page struct and hash serialization
│   │   ├── page_nodes.go        # PageNode struct for link graph
│   │   └── image.go             # Image struct
│   │
│   └── utils/
│       ├── constants.go         # Redis keys, timeouts, score bounds
│       ├── normalize_url.go     # URL canonicalization
│       ├── strip_url.go         # Fragment removal
│       └── is_valid_url.go      # URL validation
│
├── go.mod
├── go.sum
└── README.md
```

### Package Responsibilities

| Package       | Responsibility                                                  |
|---------------|-----------------------------------------------------------------|
| `cmd`         | Application entrypoint, CLI parsing, batch orchestration        |
| `crawler`     | Core crawl loop, HTTP fetching, HTML parsing, link extraction   |
| `controllers` | Persist batch data to Redis via pipelined writes                |
| `database`    | Redis client wrapper with queue, set, and hash operations       |
| `pages`       | Data models (`Page`, `PageNode`, `Image`) and serialization     |
| `utils`       | URL normalization, validation, stripping, and constants         |
