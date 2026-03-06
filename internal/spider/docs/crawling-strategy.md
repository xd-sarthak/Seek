# Crawling Strategy

This document explains the design decisions behind the crawler's traversal strategy, URL handling, concurrency model, and throughput control.

---

## Traversal: Breadth-First Search (BFS)

The crawler uses a **BFS strategy** implemented through a Redis sorted set (`spider_queue`) acting as a priority queue. URLs are scored by their depth from the seed:

- Seed URL is pushed with score `0`
- Each discovered URL is scored as `parent_depth + 1`
- `BZPopMin` always pops the lowest-score URL first

This ensures the crawler explores pages level by level, fully traversing all links at depth `N` before moving to depth `N+1`. The score is clamped between `MinScore` (-1000) and `MaxScore` (10000) to prevent score overflow during very deep crawls.

---

## URL Normalization

All URLs pass through `NormalizeURL()` before being used as keys or compared for equality. The normalization pipeline ensures that semantically identical URLs collapse to the same canonical form.

### Normalization Steps

| Step                          | Example                                                   |
|-------------------------------|-----------------------------------------------------------|
| Parse and validate scheme     | `ftp://example.com` → **rejected** (only `http`/`https`)  |
| Lowercase hostname            | `EXAMPLE.COM/Page` → `example.com/Page`                   |
| Strip `www.` prefix           | `www.example.com/page` → `example.com/page`               |
| Remove trailing slash         | `example.com/page/` → `example.com/page`                  |
| Strip fragments               | `example.com/page#section` → `example.com/page`           |
| Remove tracking query params  | `example.com?utm_source=x&q=foo` → `example.com?q=foo`    |
| Preserve non-tracking params  | `example.com/search?q=test` → preserved as-is             |

### Stripped Tracking Parameters

The following query parameters are removed during normalization:

```
utm_source, utm_medium, utm_campaign, utm_term, utm_content
fbclid, gclid, ref
```

### Fragment Stripping (`StripURL`)

Before normalization, `StripURL()` removes URL fragments (`#section`) while preserving the scheme, host, path, and query parameters. This is called during `PushURL()` before the URL enters the frontier.

---

## URL Validation

`IsValidURL()` performs fast pre-filtering before a URL is enqueued:

1. **Wikipedia filter** — Rejects URLs containing `w/index.php` (special/action pages)
2. **ASCII-only** — Every character must be in the range `0x00–0x7F`, printable, and either alphanumeric or in the allowed symbol set: `-._~:/?#[]@!$&'()*+,;=`
3. **No percent-encoding** — URLs containing `%` are rejected to avoid double-encoding issues

Additionally, during HTML parsing, the following URLs are skipped:
- URLs containing spaces, `<`, `>`, or `"` characters
- URLs with non-ASCII characters (filtered by regex `[^\x20-\x7E]`)

---

## Deduplication

The crawler uses a **two-layer deduplication** strategy:

### Layer 1: Redis Global Deduplication

Before fetching any URL, the worker queries a Redis hash:

```
Key:    normalized_url:<normalized_url>
Field:  visited
Value:  1 (visited) or absent (not visited)
```

This provides **persistent, cross-instance deduplication**. If multiple crawler instances are running, they all share the same visited set via Redis, preventing redundant crawling.

### Layer 2: In-Memory Batch Deduplication

Within a single batch, `CrawlerConfig.Pages` acts as a secondary dedup layer:

```go
if _, visited := crawcfg.Pages[normalizedURL]; visited {
    return fmt.Errorf("Page already visited")
}
```

This prevents the same page from being added to the batch twice when multiple workers discover it simultaneously before either marks it as visited in Redis.

### Layer 3: Extraction-Time Deduplication

During HTML parsing, extracted links are stored in a `map[string]struct{}` (Go set pattern), which naturally deduplicates links within a single page.

---

## Frontier Control

### Worker Pool

The crawler uses a fixed-size goroutine pool:

```go
for range crawler.MaxConcurrency {
    crawler.Wg.Add(1)
    go crawler.Crawl(db)
}
crawler.Wg.Wait()
```

`MaxConcurrency` (default: 10) controls the number of parallel workers. Each worker independently:
1. Pops a URL from the shared Redis frontier
2. Fetches and processes the page
3. Repeats until `MaxPages` is reached

### Batch Cycling

The crawler operates in **batch cycles**:

1. Spawn `N` workers
2. Workers crawl until `MaxPages` pages are collected
3. All workers finish (`WaitGroup.Wait()`)
4. Controllers flush data to Redis via pipelined writes
5. In-memory maps are reset
6. Next batch begins

This batch model bounds memory usage — in-memory data never exceeds `MaxPages` entries, regardless of total crawl depth.

---

## Backpressure

The crawler implements **backpressure** to prevent overwhelming downstream services (the indexer):

```
┌──────────┐   checks   ┌──────────────┐   if size >= 5000   ┌──────────────┐
│  Crawler │──────────→  │  pages_queue  │ ──────────────────→ │  BLOCK on    │
│  (main)  │            │  (LLEN)       │                     │  signal_queue │
└──────────┘            └──────────────┘                     └──────┬───────┘
                                                                    │
                                                              RESUME_CRAWL
                                                                    │
                                                              ┌─────┴──────┐
                                                              │  Continue  │
                                                              │  crawling  │
                                                              └────────────┘
```

### Mechanism

Before each batch cycle, the main loop:

1. Queries `LLEN pages_queue` for the indexer queue size
2. If the size is **≥ 5000** (`MaxIndexerQueueSize`), the crawler **blocks**
3. It performs a blocking `BRPOP` on `signal_queue` (no timeout)
4. When the indexer pushes a `RESUME_CRAWL` signal, crawling resumes

This ensures the crawler does not produce data faster than the indexer can consume it. The threshold of 5000 is defined in `utils/constants.go`.

---

## Concurrency Safety

All shared state in `CrawlerConfig` is protected by a single `sync.Mutex`:

| Operation           | Lock Scope                                                  |
|---------------------|-------------------------------------------------------------|
| `lenPages()`        | Read-lock on `Pages` map                                    |
| `maxPagesReached()` | Read-lock on `Pages` map                                    |
| `addPage()`         | Write-lock: check + insert into `Pages`                     |
| `AddImages()`       | Write-lock: append to `Images` map                          |
| `UpdateLinks()`     | Write-lock: modify `Outlinks` and `Backlinks` maps          |

Redis operations (`BZPopMin`, `ZADD`, `HGET`, `HSET`) are inherently atomic and do not require application-level locking.

---

## Score Clamping

Discovered URLs are scored using depth-based priority:

```go
score = depthLevel + 1                                    // new URLs
score = math.Max(MinScore, math.Min(score, MaxScore))     // clamp
```

| Constant   | Value   | Purpose                                        |
|------------|---------|------------------------------------------------|
| `MinScore` | -1000   | Floor for priority score (highest priority)    |
| `MaxScore` | 10000   | Ceiling for priority score (lowest priority)   |

If a URL already exists in the queue, its current score is retrieved and clamped. This prevents unbounded score growth for deeply nested pages.
