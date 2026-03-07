package robotstxt

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// robotRules holds parsed directives from a single robots.txt file.
type robotRules struct {
	AllowPaths    []string
	DisallowPaths []string
	CrawlDelay    time.Duration
	FetchedAt     time.Time
}

const cacheTTL = 1 * time.Hour

// RobotsChecker fetches, caches, and evaluates robots.txt rules per domain.
// Thread-safe for concurrent use by multiple crawler goroutines.
type RobotsChecker struct {
	mu        sync.RWMutex
	cache     map[string]*robotRules // keyed by host
	userAgent string
	client    *http.Client
}

// NewRobotsChecker creates a new RobotsChecker with the given user agent string.
func NewRobotsChecker(userAgent string) *RobotsChecker {
	return &RobotsChecker{
		cache:     make(map[string]*robotRules),
		userAgent: userAgent,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// IsAllowed checks whether the given URL may be crawled according to the
// host's robots.txt. Returns true if allowed, false if disallowed.
// On fetch failure, defaults to allowing the URL.
func (rc *RobotsChecker) IsAllowed(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true // can't parse, let the crawler handle it
	}

	host := u.Host
	rules := rc.getRules(host, u.Scheme)
	if rules == nil {
		return true // no rules = allow all
	}

	path := u.Path
	if path == "" {
		path = "/"
	}

	// Check allow rules first (more specific wins in standard robots.txt)
	for _, allow := range rules.AllowPaths {
		if strings.HasPrefix(path, allow) {
			return true
		}
	}

	// Check disallow rules
	for _, disallow := range rules.DisallowPaths {
		if disallow == "" {
			continue
		}
		if strings.HasPrefix(path, disallow) {
			return false
		}
	}

	return true
}

// GetCrawlDelay returns the Crawl-delay directive for the given host, if any.
// Returns 0 if no delay is specified or the host has not been fetched.
func (rc *RobotsChecker) GetCrawlDelay(rawURL string) time.Duration {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0
	}

	rules := rc.getRules(u.Host, u.Scheme)
	if rules == nil {
		return 0
	}
	return rules.CrawlDelay
}

// getRules returns cached rules for a host, fetching and parsing robots.txt
// if not cached or expired.
func (rc *RobotsChecker) getRules(host, scheme string) *robotRules {
	// Fast path: read lock
	rc.mu.RLock()
	rules, exists := rc.cache[host]
	rc.mu.RUnlock()

	if exists && time.Since(rules.FetchedAt) < cacheTTL {
		return rules
	}

	// Slow path: fetch and cache
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Double-check after acquiring write lock
	if rules, exists := rc.cache[host]; exists && time.Since(rules.FetchedAt) < cacheTTL {
		return rules
	}

	fetched := rc.fetchAndParse(host, scheme)
	rc.cache[host] = fetched
	return fetched
}

// fetchAndParse downloads and parses robots.txt for the given host.
// Returns a permissive rule set on any error.
func (rc *RobotsChecker) fetchAndParse(host, scheme string) *robotRules {
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", scheme, host)

	resp, err := rc.client.Get(robotsURL)
	if err != nil {
		log.Printf("[robots.txt] Failed to fetch %s: %v (defaulting to allow all)", robotsURL, err)
		return &robotRules{FetchedAt: time.Now()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[robots.txt] %s returned %d (defaulting to allow all)", robotsURL, resp.StatusCode)
		return &robotRules{FetchedAt: time.Now()}
	}

	rules := &robotRules{FetchedAt: time.Now()}
	scanner := bufio.NewScanner(resp.Body)

	// Track which user-agent group we're in
	inRelevantGroup := false
	foundRelevantGroup := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first colon
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch key {
		case "user-agent":
			ua := strings.ToLower(value)
			// Match our specific user-agent or wildcard
			if ua == "*" || strings.Contains(strings.ToLower(rc.userAgent), ua) {
				inRelevantGroup = true
				foundRelevantGroup = true
			} else {
				// Only leave the group if we already found ours
				if foundRelevantGroup {
					inRelevantGroup = false
				}
			}

		case "disallow":
			if inRelevantGroup && value != "" {
				rules.DisallowPaths = append(rules.DisallowPaths, value)
			}

		case "allow":
			if inRelevantGroup && value != "" {
				rules.AllowPaths = append(rules.AllowPaths, value)
			}

		case "crawl-delay":
			if inRelevantGroup {
				if delay, err := strconv.ParseFloat(value, 64); err == nil && delay > 0 {
					rules.CrawlDelay = time.Duration(delay * float64(time.Second))
				}
			}
		}
	}

	log.Printf("[robots.txt] Parsed %s: %d allow, %d disallow, delay=%v",
		robotsURL, len(rules.AllowPaths), len(rules.DisallowPaths), rules.CrawlDelay)

	return rules
}
