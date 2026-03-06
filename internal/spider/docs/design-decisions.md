# Design Decisions

This document explains the rationale behind key architectural and technology choices in the Go Search Engine Spider.

---

## Why Go?

**Decision:** Use Go as the implementation language for the web crawler.

**Rationale:**

- **Goroutines** — Go's lightweight goroutines (~4 KB stack) make it trivial to run hundreds of concurrent crawl workers without the overhead of OS threads. The crawler spawns `N` goroutines per batch cycle, each independently popping URLs, fetching pages, and processing HTML.
- **Built-in concurrency primitives** — `sync.Mutex`, `sync.WaitGroup`, and channels provide precise control over shared state without external libraries. The `CrawlerConfig` struct uses a single mutex to protect all shared maps.
- **Efficient networking** — Go's `net/http` package provides connection pooling, keep-alive, and configurable timeouts out of the box. The crawler's `httpClient` reuses connections across goroutines without additional configuration.
- **Static compilation** — Go produces a single statically-linked binary, simplifying deployment to containers and remote machines.
- **Standard library quality** — HTML parsing (`golang.org/x/net/html`), URL parsing (`net/url`), and HTTP are all available as first-party or quasi-first-party packages, reducing external dependency risk.

**Alternative considered:** Python with `asyncio` + `aiohttp`. Rejected due to the GIL, higher memory usage per concurrent task, and weaker type safety for a long-running system.

---

## Why Redis?

**Decision:** Use Redis as the URL frontier, visited set, inter-service queue, and data staging area.

**Rationale:**

- **Sub-millisecond operations** — `ZADD`, `BZPopMin`, `HSET`, `SADD`, and `LLEN` all run in O(1) or O(log N), providing the performance needed for high-throughput crawling.
- **Sorted Set as priority queue** — Redis sorted sets provide natural priority queue semantics: unique members, score-based ordering, and atomic pop-min via `BZPopMin`. This eliminates the need for an external message broker like RabbitMQ or Kafka for the URL frontier.
- **Blocking primitives** — `BZPopMin` and `BRPOP` allow workers and the main loop to block efficiently without polling, reducing CPU waste.
- **Shared state across instances** — Multiple crawler instances can connect to the same Redis, sharing the frontier and visited set. This enables horizontal scaling without application-level coordination.
- **Pipelined writes** — Redis pipelines batch multiple commands into a single round-trip, used by all three controllers to flush batch data efficiently.
- **Use as inter-service bus** — The `pages_queue` list and `signal_queue` list serve as lightweight message queues between the crawler and indexer. This avoids deploying a dedicated message broker for simple producer-consumer patterns.

**Alternative considered:** PostgreSQL for visited URLs + RabbitMQ for queues. Rejected to avoid operational complexity — Redis serves all roles with a single dependency.

---

## Why Sorted Set for the Frontier (Not a List)?

**Decision:** Use a Redis sorted set (`spider_queue`) instead of a list for the URL frontier.

**Rationale:**

- **Priority ordering** — Lists only support FIFO or LIFO. Sorted sets allow depth-based scoring, ensuring BFS traversal (pop lowest score first).
- **Built-in deduplication** — Sorted set members are unique. If the same URL is pushed multiple times (e.g., from different pages), it only appears once. With a list, the crawler would need to handle duplicate URLs at pop time.
- **Score updates** — If a URL is discovered at a shallower depth later, its score can be updated via `ZADD` with the new score. Lists don't support this.

---

## Why Hash for Visited Tracking (Not a Set)?

**Decision:** Track visited URLs using Redis hashes (`normalized_url:<url>` with `visited` field) instead of a simple set.

**Rationale:**

- **Extensibility** — A hash key per URL allows storing additional metadata in the future: `last_crawled`, `crawl_count`, `http_status`, `error_reason`, etc. A set only stores membership.
- **Per-URL metadata** — Current code only uses the `visited` field, but the hash structure is ready for richer per-URL state without migration.

**Trade-off:** Higher memory usage than a set, since each hash has key overhead. For very large-scale crawls (billions of URLs), a Redis set or a Bloom filter would be more memory-efficient.

---

## Why Batch Cycling?

**Decision:** The crawler operates in fixed-size batches (`MaxPages` per cycle) instead of continuous streaming.

**Rationale:**

- **Bounded memory** — In-memory maps (`Pages`, `Outlinks`, `Backlinks`, `Images`) grow only to `MaxPages` entries. Without batching, a continuous crawl could accumulate unbounded state.
- **Write coalescing** — Pipelining batch writes to Redis is more efficient than writing each page individually. The controllers execute a single `pipeline.Exec()` per batch.
- **Clean worker lifecycle** — Workers start, crawl, stop, and data is flushed. This makes error recovery simpler — if the process crashes mid-batch, at most `MaxPages` entries are lost, and the URLs remain in the frontier for re-crawling.
- **Backpressure checkpoints** — The backpressure check happens between batches, not during crawling. This avoids complex mid-crawl pausing logic.

---

## Why a Single Mutex (Not Per-Map Locks)?

**Decision:** All shared maps in `CrawlerConfig` are protected by a single `sync.Mutex`.

**Rationale:**

- **Simplicity** — A single lock eliminates the risk of lock ordering deadlocks that arise with multiple mutexes.
- **Short critical sections** — Lock operations are brief (map lookups, inserts, length checks). Contention is low because the bottleneck is HTTP fetching (30-second timeout), not in-memory operations.
- **Correctness over performance** — At `MaxConcurrency=10`, a single mutex has negligible overhead. The crawler is I/O-bound, not CPU-bound. Optimizing for lock-free data structures would add complexity without measurable throughput gains.

**Alternative considered:** `sync.RWMutex` with read locks for `maxPagesReached()` and `lenPages()`. Deferred as a future optimization if profiling shows lock contention.

---

## Why `BZPopMin` with a Timeout?

**Decision:** Use `BZPopMin` with a 5-second timeout (not infinite blocking) for popping URLs from the frontier.

**Rationale:**

- **Graceful termination** — If the frontier is empty, workers return an error after 5 seconds and exit cleanly. Infinite blocking would keep workers alive indefinitely when there are no more URLs to crawl.
- **Batch completion** — Combined with `maxPagesReached()`, the timeout ensures workers don't block forever when the batch is finishing. Workers naturally wind down as the frontier drains.

---

## Why `10 MB` Body Limit?

**Decision:** Limit HTTP response bodies to 10 MB via `io.LimitReader`.

**Rationale:**

- **OOM prevention** — Without a limit, a malicious or misconfigured server could return a multi-gigabyte response, exhausting crawler memory.
- **Reasonable page size** — The vast majority of web pages are under 5 MB. Pages exceeding 10 MB are typically non-HTML content (videos, downloads) that the crawler can't process anyway.
- **Combined with content-type check** — The crawler also rejects non-`text/html` responses, so the 10 MB limit is a secondary safety net.

---

## Why Pipeline Redis Writes?

**Decision:** Use Redis pipelines to batch all writes at the end of each crawl cycle.

**Rationale:**

- **Reduced round-trips** — A batch of 100 pages generates ~200+ Redis commands (page hashes, queue pushes, backlinks, outlinks, images). Without pipelining, each command is a separate network round-trip. The pipeline sends all commands in a single round-trip and reads all responses at once.
- **Atomic flush** — While Redis pipelines are not transactional, they execute commands back-to-back without interleaving commands from other clients, providing a quasi-atomic write behavior.
- **Throughput** — Pipelining can achieve 10–100x higher write throughput compared to sequential commands, depending on network latency.

---

## Why Image TTL of 1 Hour?

**Decision:** Image metadata keys (`image_data:<url>`) have a 1-hour TTL via `EXPIRE`.

**Rationale:**

- **Garbage collection** — Image data is transient: it's produced by the crawler and consumed by a separate image indexer. If the image indexer is slow or offline, stale image keys are automatically cleaned up.
- **Storage management** — Without TTLs, image keys would accumulate indefinitely, potentially consuming significant Redis memory.
- **Data is non-critical** — If an image key expires before being consumed, the image will be re-discovered on the next crawl of the parent page. No data is permanently lost.

---

## Why Separate Controllers?

**Decision:** Use three separate controllers (`PageController`, `LinksController`, `ImageController`) instead of a single monolithic write function.

**Rationale:**

- **Single responsibility** — Each controller handles one data type: pages, link graphs, or images. This makes the code easier to test, modify, and extend independently.
- **Independent pipelines** — Each controller creates its own Redis pipeline. If link writes fail, page writes are unaffected.
- **Future extensibility** — New data types (e.g., metadata, scripts, stylesheets) can be added by creating a new controller without modifying existing ones.
