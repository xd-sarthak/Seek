# Focused Crawler Plan

## Goal

Shift the current general-purpose crawler toward a developer-focused crawler by making crawl decisions based on host and URL structure, while preserving the existing Redis frontier and BFS traversal model.

## Current State

- Host-level allowlisting is implemented.
- Seed URLs are validated against the allowlist before entering the frontier.
- Discovered links are filtered by allowlist before being enqueued.
- The crawler still treats all allowed URLs with the same priority beyond BFS depth.

## Design Principles

1. Keep BFS as the base traversal strategy.
2. Add focused-crawl behavior at enqueue time, not fetch time.
3. Use numeric score adjustments instead of enum priority buckets.
4. Keep crawl policy separate from generic utility code.
5. Match rules from most specific to least specific.
6. Use only host and path in v1 policy decisions.
7. Treat query normalization as a URL-normalization concern, not a policy concern.

## Target Architecture

The URL decision pipeline should be:

1. Link extracted from HTML
2. URL normalized
3. Host allowlist checked
4. Policy evaluator applied
5. Numeric score adjustment computed
6. Final frontier score calculated
7. URL enqueued into Redis

## Policy Package

Create a dedicated package:

`internal/spider/internal/policy`

Suggested files:

- `decision.go`
- `rules.go`
- `evaluator.go`
- `policy_test.go`

This package should own:

- domain/path rules
- focused crawl decisions
- URL scoring adjustments
- rule evaluation order

It should not own:

- generic URL normalization helpers
- Redis queue logic
- crawler batch state

## Decision Model

Use a numeric scoring model:

```go
type URLDecision struct {
    Allowed         bool
    ScoreAdjustment float64
    Reason          string
}
```

Meaning:

- `Allowed=false` means the URL must not be enqueued.
- `ScoreAdjustment<0` means higher priority.
- `ScoreAdjustment=0` means neutral.
- `ScoreAdjustment>0` means lower priority.

Final score:

```go
score = parentDepth + 1 + ScoreAdjustment
```

Clamp after adjustment using the existing score bounds.

## Rule Model

Start with simple ordered rules:

```go
type URLRule struct {
    Host            string
    PathPrefix      string
    Allowed         bool
    ScoreAdjustment float64
    Reason          string
}
```

Rule semantics:

- first matching rule wins
- rules must be ordered most specific to least specific
- no query-based matching in v1

## Initial Domains

Support only:

- `github.com`
- `stackoverflow.com`

## Initial Rule Set

### GitHub

Deny first:

- `/login`
- `/settings`
- `/search`
- `/notifications`
- `/sessions`

Deprioritize:

- `/issues`
- `/pulls`
- `/discussions`
- `/marketplace`
- `/topics`
- `/orgs`
- `/users`

Prioritize:

- repository-like paths
- `/blob/`
- `/tree/`
- `/readme`

Fallback:

- neutral for any remaining allowed GitHub path

### Stack Overflow

Deny first:

- `/search`
- `/users/login`

Deprioritize:

- `/users/`
- `/tags/`

Prioritize:

- `/questions/`

Fallback:

- neutral for any remaining allowed Stack Overflow path

## GitHub Path Shape Handling

Static prefixes alone are not enough for GitHub.

The evaluator should include lightweight path-shape checks for:

- repo root: `/<owner>/<repo>`
- blob pages: `/<owner>/<repo>/blob/...`
- tree pages: `/<owner>/<repo>/tree/...`
- issue/pr/discussion areas

This should remain simple and readable. Avoid regex-heavy logic in v1.

## Integration Plan

### Step 1: Create the policy package

Add the new package under `internal/spider/internal/policy` with decision types, rule tables, and the evaluator.

### Step 2: Move URL decision logic out of `utils`

Keep host normalization helpers in `utils`, but move focused-crawl decisions to the policy package.

### Step 3: Evaluate URLs during enqueue

In `internal/spider/internal/crawler/crawl.go`:

- replace the allowlist-only check with policy evaluation
- skip enqueue for denied URLs
- apply `ScoreAdjustment` before pushing to Redis
- clamp after adjustment

### Step 4: Keep startup validation simple

In `internal/spider/cmd/main.go`:

- continue validating the seed host against the allowlist
- do not reject startup for neutral or low-priority URLs
- only reject malformed or non-allowed seed URLs

### Step 5: Add tests

Test:

- allowlisted host with no matching rule returns neutral
- disallowed host is denied
- malformed URL returns error
- GitHub login/settings/search are denied
- GitHub repo/blob/tree paths are prioritized
- GitHub issues/pulls/discussions are deprioritized
- Stack Overflow question pages are prioritized
- Stack Overflow users/tags are deprioritized
- rule ordering works as expected

### Step 6: Validate frontier behavior

Run a small crawl and inspect the Redis sorted set ordering to confirm that better developer URLs are receiving lower scores.

## Non-Goals For v1

- query-based policy rules
- content-based page scoring
- freshness-based scoring
- inlink-based scoring
- dynamic rule loading
- regex-heavy rule definitions

## Risks

1. Overbroad GitHub rules can prioritize too much low-value content.
2. Weak rule ordering can accidentally allow junk pages.
3. Path-prefix-only logic can miss important GitHub path shapes if not supplemented with minimal structural checks.
4. Aggressive deprioritization may hide useful developer content if scores are tuned badly.

## Success Criteria

1. The crawler still behaves as BFS by default.
2. Allowed developer URLs can be ranked ahead of generic allowed URLs at the same depth.
3. Known junk surfaces such as login and search pages are kept out of the frontier.
4. Rules are easy to read, test, and extend per domain.
5. The policy code is isolated from utility helpers and crawler infrastructure.

## Immediate Next Task

Implement the `internal/spider/internal/policy` package and wire it into `crawl.go` so enqueue decisions use `Allowed` plus `ScoreAdjustment` instead of allowlist-only filtering.
