package ratelimiter

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// DomainLimiter enforces per-host request rate limits to avoid overwhelming
// target servers. Each domain gets its own token-bucket rate limiter.
// Thread-safe for concurrent use by multiple crawler goroutines.
type DomainLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      float64 // requests per second per domain
	burst    int
}

// NewDomainLimiter creates a new DomainLimiter with the given default rate
// (requests per second) and burst size per domain.
func NewDomainLimiter(requestsPerSecond float64, burst int) *DomainLimiter {
	return &DomainLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      requestsPerSecond,
		burst:    burst,
	}
}

// Wait blocks until the rate limiter for the URL's host allows a request,
// or until the context is cancelled. Returns an error if the context is
// cancelled or the URL cannot be parsed.
func (dl *DomainLimiter) Wait(ctx context.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("could not parse URL for rate limiting: %w", err)
	}

	host := u.Host
	limiter := dl.getLimiter(host)

	return limiter.Wait(ctx)
}

// SetDomainRate overrides the rate limit for a specific host.
// This is useful for honoring robots.txt Crawl-delay directives.
func (dl *DomainLimiter) SetDomainRate(host string, delay time.Duration) {
	if delay <= 0 {
		return
	}

	rps := 1.0 / delay.Seconds()

	dl.mu.Lock()
	defer dl.mu.Unlock()

	dl.limiters[host] = rate.NewLimiter(rate.Limit(rps), 1)
}

// getLimiter returns the rate limiter for a host, creating one with the
// default rate if it doesn't exist yet.
func (dl *DomainLimiter) getLimiter(host string) *rate.Limiter {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	if limiter, exists := dl.limiters[host]; exists {
		return limiter
	}

	limiter := rate.NewLimiter(rate.Limit(dl.rps), dl.burst)
	dl.limiters[host] = limiter
	return limiter
}
