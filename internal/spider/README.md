# Go Search Engine Spider

A distributed web crawler written in Go that collects webpages, extracts links and images, and pushes page data into a Redis-backed indexing pipeline.

The crawler is designed for **high concurrency** and **horizontal scaling**. It uses a BFS (breadth-first search) strategy with a priority-scored frontier, goroutine-based worker pools, and Redis as both a URL queue and a persistence layer for downstream services.

> **Note:** TF-IDF scoring and ranking are not yet implemented. The spider focuses exclusively on crawling, extraction, and data ingestion.

---

## Architecture Pipeline

```
Seed URL
    ↓
Crawler Frontier (Redis Sorted Set — priority queue)
    ↓
Worker Pool (N goroutines, configurable via --max-concurrency)
    ↓
HTTP Fetch (30s timeout, 10MB body limit)
    ↓
HTML Parser (golang.org/x/net/html)
    ↓
Link & Image Extraction
    ↓
URL Normalization (lowercase host, strip www/fragments/tracking params)
    ↓
Deduplication (Redis hash-based visited check)
    ↓
Redis Index Queue (pages_queue — consumed by indexer service)
```

### Stage Descriptions

**Seed URL** — The initial URL that boots the crawler. Configurable via the `STARTING_URL` environment variable. Defaults to `https://en.wikipedia.org/wiki/Kamen_Rider`. Pushed into the Redis sorted set frontier with a score of `0` (highest priority).

**Crawler Frontier** — A Redis sorted set (`spider_queue`) that functions as a priority queue. URLs are scored by depth level — lower scores are popped first via `BZPopMin`. This ensures breadth-first traversal. New URLs are scored as `parent_depth + 1`, clamped between `-1000` and `10000`.

**Worker Pool** — `N` goroutines (default 10) run the `Crawl()` loop concurrently. Each worker independently pops URLs from the frontier, fetches pages, and extracts data. Workers share a mutex-protected `CrawlerConfig` struct. Workers terminate when `MaxPages` (default 100) is reached for the current batch.

**HTTP Fetch** — Uses a shared `http.Client` with a 30-second total timeout, TLS handshake and response header timeouts of 10 seconds each, connection pooling (100 idle connections, 10 per host), and a 10 MB response body limit. Identifies as `SearchEngineSpider/1.0`. Only `text/html` responses are processed.

**HTML Parser** — The fetched HTML is parsed using Go's `golang.org/x/net/html` tokenizer. The parser performs a depth-first traversal of the DOM tree, extracting `<a href>` links and `<img src>` / `<img alt>` attributes. Malformed URLs and non-ASCII URLs are skipped for safety.

**Link & Image Extraction** — Links are resolved against the base URL (relative → absolute) and deduplicated using an in-memory set during extraction. Images are collected with their `src` (normalized) and `alt` text, keyed by source URL to prevent duplicates per page.

**URL Normalization** — URLs undergo canonical normalization: scheme validation (`http`/`https` only), lowercased hostnames, `www.` prefix removal, trailing slash removal, fragment stripping, and removal of tracking query parameters (`utm_*`, `fbclid`, `gclid`, `ref`). This ensures the same logical page is never crawled twice.

**Deduplication** — Before crawling, each URL is checked against a Redis hash (`normalized_url:<url>`) with a `visited` field. After successful crawling, the URL is marked as visited. This provides global, persistent deduplication across crawler restarts and multiple instances.

**Redis Index Queue** — Crawled page data is written to Redis hashes (`page_data:<url>`) and their keys are pushed to a Redis list (`pages_queue`) for consumption by a downstream indexer service. Backlinks, outlinks, and image metadata are also persisted via separate Redis structures.

---

## Quick Start

### Prerequisites

- Go 1.25+
- Redis 7+

### Run

```bash
# Start Redis
redis-server

# Run the crawler
cd internal/spider
go run ./cmd/main.go --max-concurrency=10 --max-pages=100
```

### Environment Variables

| Variable         | Default                                                  | Description                      |
|------------------|----------------------------------------------------------|----------------------------------|
| `REDIS_HOST`     | `localhost`                                              | Redis server hostname            |
| `REDIS_PORT`     | `6379`                                                   | Redis server port                |
| `REDIS_PASSWORD` | (empty)                                                  | Redis authentication password    |
| `REDIS_DB`       | `0`                                                      | Redis database index             |
| `STARTING_URL`   | `https://en.wikipedia.org/wiki/Kamen_Rider`              | Seed URL for crawling            |
| `ALLOWED_DOMAINS`| `github.com,raw.githubusercontent.com,stackoverflow.com,developer.mozilla.org,docs.python.org,react.dev,nodejs.org,golang.org,docs.docker.com,kubernetes.io,pkg.go.dev,docs.rs` | Comma-separated host allowlist |

### CLI Flags

| Flag                | Default | Description                              |
|---------------------|---------|------------------------------------------|
| `--max-concurrency` | `10`    | Number of concurrent worker goroutines   |
| `--max-pages`       | `100`   | Maximum pages crawled per batch cycle    |

---

## Documentation

| Document                                            | Description                              |
|-----------------------------------------------------|------------------------------------------|
| [Architecture](docs/architecture.md)                | System components and data flow          |
| [Crawling Strategy](docs/crawling-strategy.md)      | Deduplication, frontier, backpressure    |
| [Redis Schema](docs/redis-schema.md)                | Redis key layout and data model          |
| [Development Process](docs/dev-process.md)          | Local setup, env vars, testing           |
| [Design Decisions](docs/design-decisions.md)        | Rationale for key architectural choices  |

---

## Project Status

- [x] Concurrent BFS crawling with goroutine worker pool
- [x] HTML parsing and link/image extraction
- [x] URL normalization and deduplication
- [x] Redis-backed priority queue frontier
- [x] Backpressure based on indexer queue size
- [x] Pipelined Redis writes for pages, links, and images
- [ ] TF-IDF scoring
- [ ] PageRank computation
- [ ] robots.txt compliance
- [ ] Rate limiting per domain
