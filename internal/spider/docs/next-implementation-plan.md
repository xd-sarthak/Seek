# Seek Next Implementation Plan

This document translates the post-crawler parts of `moogle` into concrete implementation steps for `Seek`.

Scope:

- Keep the current `Seek` crawler as the completed foundation.
- Use the same architectural direction as `moogle`.
- Do not rewrite the crawler.
- Build the next modules in small, testable increments.

The first target is not "build the whole search engine".
The first target is:

`crawler -> indexer -> MongoDB`

Once that works, add TF-IDF, backlinks, PageRank, and query serving.

---

## Immediate Next Step

Build a minimal `indexer` service that:

1. pops page keys from Redis `pages_queue`
2. reads `page_data:<normalized_url>`
3. parses HTML into searchable text
4. writes durable MongoDB documents
5. deletes Redis transient data only after Mongo writes succeed

This is the first step that converts your crawler into a real search pipeline.

---

## Proposed Repo Layout

Your repo currently has only:

- `internal/spider/`

Add the next services beside it:

```text
Seek/
  internal/
    spider/
    indexer/
      cmd/
        main.py
      data/
        mongo_client.py
        redis_client.py
      models/
        page.py
        metadata.py
        outlinks.py
      utils/
        constants.py
        utils.py
        nlp_utils.py
      README.md
      requirements.txt
      Dockerfile
      docker-compose.yml
    tfidf/
    backlinks-processor/
    page-rank/
    query-engine/
```

Do not build all of these at once.
Build them in the order below.

---

## Phase 1: Minimal Indexer

### Goal

Consume crawler output from Redis and materialize searchable page data into MongoDB.

### Files to create in `Seek`

- `internal/indexer/cmd/main.py`
- `internal/indexer/data/redis_client.py`
- `internal/indexer/data/mongo_client.py`
- `internal/indexer/models/page.py`
- `internal/indexer/models/metadata.py`
- `internal/indexer/models/outlinks.py`
- `internal/indexer/utils/constants.py`
- `internal/indexer/utils/utils.py`
- `internal/indexer/utils/nlp_utils.py`
- `internal/indexer/requirements.txt`
- `internal/indexer/README.md`
- `internal/indexer/Dockerfile`
- `internal/indexer/docker-compose.yml`

### What each file should do

#### `internal/indexer/cmd/main.py`

Responsibilities:

- initialize Redis connection
- initialize Mongo connection
- block on `pages_queue`
- fetch page payload from Redis
- parse HTML
- build word frequency map
- write `metadata`, `words`, and `outlinks`
- delete Redis keys after success
- push a page id to a future image queue if you want image indexing later

Use these `moogle` references:

- [services/indexer/main.py](/home/sarthak/dev/moogle/moogle/services/indexer/main.py)

Implementation notes:

- Copy the processing loop shape, not the code blindly.
- Keep the first version single-purpose and simple.
- Do not add batching complexity beyond Mongo bulk writes.

#### `internal/indexer/data/redis_client.py`

Responsibilities:

- `pop_page()`
- `get_page_data(key)`
- `get_outlinks(normalized_url)`
- `delete_page_data(key)`
- `delete_outlinks(normalized_url)`
- `get_queue_size()`

Use these `moogle` references:

- [services/indexer/data/redis_client.py](/home/sarthak/dev/moogle/moogle/services/indexer/data/redis_client.py)

Implementation notes:

- Match your crawler's Redis schema exactly.
- Your source of truth for Redis key names is the Seek spider docs:
  - [internal/spider/docs/redis-schema.md](/home/sarthak/dev/seek-search-engine/Seek/internal/spider/docs/redis-schema.md)

#### `internal/indexer/data/mongo_client.py`

Responsibilities:

- connect to MongoDB
- create indexes for `words`
- provide bulk upsert operations for:
  - `words`
  - `metadata`
  - `outlinks`
  - `dictionary`

Use these `moogle` references:

- [services/indexer/data/mongo_client.py](/home/sarthak/dev/moogle/moogle/services/indexer/data/mongo_client.py)

Implementation notes:

- Start with these Mongo collections:
  - `words`
  - `metadata`
  - `outlinks`
  - `dictionary`
- Add the same compound unique index on `(word, url)`.

#### `internal/indexer/models/page.py`

Responsibilities:

- deserialize Redis `page_data:*` hash into a Python model

Use these `moogle` references:

- [services/indexer/models/page.py](/home/sarthak/dev/moogle/moogle/services/indexer/models/page.py)

Implementation notes:

- Keep field names aligned to the Go spider output:
  - `normalized_url`
  - `html`
  - `content_type`
  - `status_code`
  - `last_crawled`

#### `internal/indexer/models/metadata.py`

Responsibilities:

- define the page metadata document written to MongoDB

Use these `moogle` references:

- [services/indexer/models/metadata.py](/home/sarthak/dev/moogle/moogle/services/indexer/models/metadata.py)

First version fields:

- `_id`
- `title`
- `description`
- `summary_text`
- `last_crawled`
- `keywords`

#### `internal/indexer/models/outlinks.py`

Responsibilities:

- define a Mongo document for page outlinks

Use these `moogle` references:

- [services/indexer/models/outlinks.py](/home/sarthak/dev/moogle/moogle/services/indexer/models/outlinks.py)

#### `internal/indexer/utils/utils.py`

Responsibilities:

- parse HTML
- extract title and description
- create summary text
- tokenize page text
- remove stop words
- detect language
- split words from URL if you want URL token boosts later

Use these `moogle` references:

- [services/indexer/utils/utils.py](/home/sarthak/dev/moogle/moogle/services/indexer/utils/utils.py)

Implementation notes:

- For the first version, keep:
  - `get_html_data(html)`
  - `split_url(url)`
- You do not need to implement every helper from `moogle`.

#### `internal/indexer/utils/nlp_utils.py`

Responsibilities:

- ensure NLTK resources exist
- initialize stop words

Use these `moogle` references:

- [services/indexer/utils/nlp_utils.py](/home/sarthak/dev/moogle/moogle/services/indexer/utils/nlp_utils.py)

#### `internal/indexer/utils/constants.py`

Responsibilities:

- centralize queue names and collection names

Suggested constants:

- `INDEXER_QUEUE_KEY = "pages_queue"`
- `OUTLINKS_PREFIX = "outlinks"`
- `WORDS_COLLECTION = "words"`
- `METADATA_COLLECTION = "metadata"`
- `OUTLINKS_COLLECTION = "outlinks"`
- `DICTIONARY_COLLECTION = "dictionary"`

### First milestone

When the indexer is done, one crawled page should produce:

- one `metadata` document
- many `words` documents
- one `outlinks` document

You should be able to verify this by crawling a few pages and inspecting Mongo manually.

---

## Phase 2: Corpus Statistics and TF-IDF

### Goal

Add corpus-level scoring without slowing down indexing.

### Files to create in `Seek`

- `internal/tfidf/main.py`
- `internal/tfidf/data/mongo_client.py`
- `internal/tfidf/requirements.txt`
- `internal/tfidf/README.md`
- `internal/tfidf/Dockerfile`
- `internal/tfidf/docker-compose.yml`

### What each file should do

#### `internal/tfidf/main.py`

Responsibilities:

- count total indexed documents
- iterate unique words
- compute `idf`
- update each `(word, url)` entry with `tfidf`

Use these `moogle` references:

- [services/tfidf/main.py](/home/sarthak/dev/moogle/moogle/services/tfidf/main.py)

Implementation notes:

- Keep this as a batch job.
- Do not compute TF-IDF inline in the indexer.
- First version can run manually after indexing.

#### `internal/tfidf/data/mongo_client.py`

Responsibilities:

- load unique words
- count docs per word
- read word entries
- bulk update `idf` and `tfidf`

Use these `moogle` references:

- [services/tfidf/data/mongo_client.py](/home/sarthak/dev/moogle/moogle/services/tfidf/data/mongo_client.py)

### Second milestone

After TF-IDF runs, `words` entries should contain:

- `tf`
- `idf`
- `tfidf`

---

## Phase 3: Backlinks Materialization

### Goal

Convert crawl-time backlink sets into durable MongoDB graph data.

### Files to create in `Seek`

- `internal/backlinks-processor/main.py`
- `internal/backlinks-processor/data/redis_client.py`
- `internal/backlinks-processor/data/mongo_client.py`
- `internal/backlinks-processor/models/backlinks.py`
- `internal/backlinks-processor/requirements.txt`
- `internal/backlinks-processor/README.md`
- `internal/backlinks-processor/Dockerfile`
- `internal/backlinks-processor/docker-compose.yml`

### What each file should do

#### `internal/backlinks-processor/main.py`

Responsibilities:

- scan Redis `backlinks:*`
- materialize backlink sets into MongoDB
- delete processed Redis backlink keys

Use these `moogle` references:

- [services/backlinks-processor/main.py](/home/sarthak/dev/moogle/moogle/services/backlinks-processor/main.py)

Implementation notes:

- Keep it periodic and simple.
- This is a transfer process, not a ranking process.

### Third milestone

Mongo now contains:

- `metadata`
- `words`
- `outlinks`
- `backlinks`

---

## Phase 4: PageRank

### Goal

Compute authority signals from the link graph.

### Files to create in `Seek`

- `internal/page-rank/go.mod`
- `internal/page-rank/cmd/main.go`
- `internal/page-rank/README.md`
- `internal/page-rank/Dockerfile`
- `internal/page-rank/docker-compose.yml`

### What each file should do

#### `internal/page-rank/cmd/main.go`

Responsibilities:

- load link graph from MongoDB
- compute PageRank scores
- write scores back to MongoDB

Use these `moogle` references:

- [services/page-rank/cmd/page-rank/main.go](/home/sarthak/dev/moogle/moogle/services/page-rank/cmd/page-rank/main.go)

Implementation notes:

- Keep this in Go to stay aligned with your existing Go codebase.
- First version can be a batch job invoked manually.

### Fourth milestone

Mongo has a `page_rank` collection keyed by URL.

---

## Phase 5: Query Engine

### Goal

Expose the first usable search endpoint.

### Files to create in `Seek`

You have two valid choices:

#### Option A: stay close to `moogle`

- `internal/query-engine/` as a Laravel app

Use these `moogle` references:

- [services/query-engine/app/Http/Controllers/QuerySearchController.php](/home/sarthak/dev/moogle/moogle/services/query-engine/app/Http/Controllers/QuerySearchController.php)
- [services/query-engine/routes/api.php](/home/sarthak/dev/moogle/moogle/services/query-engine/routes/api.php)

#### Option B: keep Seek simpler first

- build a small Python or Go JSON API that queries Mongo directly

This deviates from `moogle`, but it is a better sequencing choice if your goal is to get search working first.

### Recommended first endpoint

- `GET /search?q=term1 term2`

Responsibilities:

- split query into words
- load matching `words` entries
- aggregate candidate URLs
- join `metadata`
- combine TF-IDF and PageRank
- return ranked results

Use these `moogle` references:

- [services/query-engine/app/Http/Controllers/QuerySearchController.php](/home/sarthak/dev/moogle/moogle/services/query-engine/app/Http/Controllers/QuerySearchController.php)
- [services/query-engine/routes/api.php](/home/sarthak/dev/moogle/moogle/services/query-engine/routes/api.php)

### Fifth milestone

You can issue a query and receive ranked URLs plus titles and snippets.

---

## Suggested Build Order

Do these in order:

1. `internal/indexer`
2. `internal/tfidf`
3. `internal/backlinks-processor`
4. `internal/page-rank`
5. `internal/query-engine`

Do not start PageRank or the query engine before the indexer is producing durable data.

---

## Exact Checklist

### Indexer

- [ ] Create `internal/indexer/` service skeleton
- [ ] Implement Redis consumer client
- [ ] Implement Mongo write client
- [ ] Implement `Page` model from Redis hash
- [ ] Implement HTML parsing helper
- [ ] Implement stop-word filtering and tokenization
- [ ] Write `metadata` documents to Mongo
- [ ] Write `(word, url, tf)` documents to Mongo
- [ ] Write `outlinks` documents to Mongo
- [ ] Add Mongo indexes for `words`
- [ ] Delete Redis page data only after successful writes
- [ ] Verify one crawled page becomes durable Mongo data

### TF-IDF

- [ ] Create `internal/tfidf/` service skeleton
- [ ] Read unique words from Mongo
- [ ] Count total documents
- [ ] Compute `idf`
- [ ] Update `tfidf` on `words`
- [ ] Verify scores exist in Mongo

### Backlinks

- [ ] Create `internal/backlinks-processor/` service skeleton
- [ ] Read `backlinks:*` from Redis
- [ ] Write `backlinks` collection in Mongo
- [ ] Delete processed Redis backlink keys
- [ ] Verify backlinks exist per URL

### PageRank

- [ ] Create `internal/page-rank/` Go service skeleton
- [ ] Read graph data from Mongo
- [ ] Compute PageRank
- [ ] Write `page_rank` results to Mongo
- [ ] Verify authority scores exist

### Query Engine

- [ ] Create `internal/query-engine/`
- [ ] Add `/search` endpoint
- [ ] Read `words`, `metadata`, and `page_rank` from Mongo
- [ ] Rank results by lexical score plus authority
- [ ] Return JSON results
- [ ] Verify end-to-end crawl to search flow

---

## Moogle File Map

Use this table as your reverse-engineering index:

| Seek task | Moogle reference |
|---|---|
| Redis queue consumer | [services/indexer/data/redis_client.py](/home/sarthak/dev/moogle/moogle/services/indexer/data/redis_client.py) |
| Indexing loop | [services/indexer/main.py](/home/sarthak/dev/moogle/moogle/services/indexer/main.py) |
| Mongo writes and indexes | [services/indexer/data/mongo_client.py](/home/sarthak/dev/moogle/moogle/services/indexer/data/mongo_client.py) |
| HTML parsing and tokenization | [services/indexer/utils/utils.py](/home/sarthak/dev/moogle/moogle/services/indexer/utils/utils.py) |
| NLP bootstrap | [services/indexer/utils/nlp_utils.py](/home/sarthak/dev/moogle/moogle/services/indexer/utils/nlp_utils.py) |
| Page model | [services/indexer/models/page.py](/home/sarthak/dev/moogle/moogle/services/indexer/models/page.py) |
| Metadata model | [services/indexer/models/metadata.py](/home/sarthak/dev/moogle/moogle/services/indexer/models/metadata.py) |
| Outlinks model | [services/indexer/models/outlinks.py](/home/sarthak/dev/moogle/moogle/services/indexer/models/outlinks.py) |
| TF-IDF batch loop | [services/tfidf/main.py](/home/sarthak/dev/moogle/moogle/services/tfidf/main.py) |
| Backlinks transfer loop | [services/backlinks-processor/main.py](/home/sarthak/dev/moogle/moogle/services/backlinks-processor/main.py) |
| PageRank job | [services/page-rank/cmd/page-rank/main.go](/home/sarthak/dev/moogle/moogle/services/page-rank/cmd/page-rank/main.go) |
| Query controller | [services/query-engine/app/Http/Controllers/QuerySearchController.php](/home/sarthak/dev/moogle/moogle/services/query-engine/app/Http/Controllers/QuerySearchController.php) |
| API routes | [services/query-engine/routes/api.php](/home/sarthak/dev/moogle/moogle/services/query-engine/routes/api.php) |

---

## Recommendation

Do not branch out into query serving yet.

The correct next move is:

1. build `internal/indexer/`
2. prove Redis crawler output becomes durable Mongo search data
3. only then continue to TF-IDF and ranking

If you get Phase 1 right, the rest of the system becomes straightforward.
