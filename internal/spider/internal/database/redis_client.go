package database

import (
	"context"
	"fmt"
	"log"
	"spider/internal/utils"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// Database wraps a Redis client and provides domain-specific
// methods for the crawler's queue operations, visited tracking, and
// inter-service signaling.
type Database struct {
	Client *redis.Client
}

// ConnectToRedis establishes a connection to the Redis server.
// It parses the database index from redisDB and verifies connectivity via PING.
func (db *Database) ConnectToRedis(redisHost, redisPort, redisPassword, redisDB string) error {
	log.Println("Connecting to redis")

	dbIndex, err := strconv.Atoi(redisDB)

	if err != nil {
		return fmt.Errorf("Could not parse DB value: %v\n", err)
	}

	db.Client = redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: redisPassword,
		DB:       dbIndex,
	})

	ctx := context.Background()
	_, err = db.Client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("couldn't connect to Redis [%v]: %v", redisHost, err)
	}

	log.Println("Successfully connected to Redis!")
	return nil
}

// PushURL strips fragments from rawURL, normalizes it, and adds it
// to the spider_queue sorted set with the given priority score.
// Lower scores are popped first (BFS depth ordering).
func (db *Database) PushURL(rawURL string, score float64) error {

	//remove fragments and queries from rawURL
	rawURL, err := utils.StripURL(rawURL)

	if err != nil {
		return fmt.Errorf("Could not strip URL: %w", err)
	}

	//normalise the url
	normalizedURL, err := utils.NormalizeURL(rawURL)
	if err != nil {
		return fmt.Errorf("Could not normalize URL: %w", err)
	}

	//add the normalized url to a sorted set with 0 as a priority score
	ctx := context.Background()
	err = db.Client.ZAdd(ctx, utils.SpiderQueueKey, redis.Z{
		Score:  score,
		Member: normalizedURL,
	}).Err()

	if err != nil {
		return fmt.Errorf("Could not add URL to spider queue: %w", err)
	}

	log.Printf("Pushed %v (%v) to queue\n", rawURL, normalizedURL)

	return nil
}

// ExistsInQueue checks whether a URL (after normalization) exists in the
// spider_queue sorted set. Returns the URL's current score and true if found,
// or (0.0, false) if the URL is not in the queue.
func (db *Database) ExistsInQueue(rawURL string) (float64, bool) {
	normalizedURL, err := utils.NormalizeURL(rawURL)
	if err != nil {
		return 0.0, false
	}

	ctx := context.Background()
	score, err := db.Client.ZScore(ctx, utils.SpiderQueueKey, normalizedURL).Result()
	if err != nil {
		// redis.Nil means the member does not exist in the sorted set
		return 0.0, false
	}

	return score, true
}

// GetIndexerQueueSize returns the number of entries in the pages_queue list.
// Used by the main loop for backpressure: if the size exceeds
// MaxIndexerQueueSize, crawling is paused until a RESUME_CRAWL signal arrives.
func (db *Database) GetIndexerQueueSize(ctx context.Context) (int64, error) {
	size, err := db.Client.LLen(ctx, utils.IndexerQueueKey).Result()

	if err != nil {
		return -1, fmt.Errorf("could not get %v size: %v", utils.IndexerQueueKey, err)
	}

	return size, nil
}

// PopSignalQueue performs a blocking pop (BRPOP) on the signal_queue list
// with no timeout. Returns the signal value (e.g., "RESUME_CRAWL").
// Blocks indefinitely until a signal is available or the context is cancelled.
func (db *Database) PopSignalQueue(ctx context.Context) (string, error) {
	signal, err := db.Client.BRPop(ctx, 0, utils.SignalQueueKey).Result()
	if err != nil {
		return "", fmt.Errorf("could not pop from signal queue: %v\n", err)
	}
	return signal[1], nil
}

// PopURL performs a blocking pop of the lowest-score URL from spider_queue.
// Uses BZPopMin with a 5-second timeout.
//
// Returns:
//   - rawURL: the URL to fetch (same as normalizedURL in current implementation)
//   - score: the URL's depth/priority score
//   - normalizedURL: the canonical form of the URL
//   - err: non-nil if the queue is empty after timeout or a Redis error occurs
func (db *Database) PopURL(ctx context.Context) (string, float64, string, error) {
	//get the next normalized url from the priority queue
	result, err := db.Client.BZPopMin(ctx, utils.Timeout, utils.SpiderQueueKey).Result()
	if err != nil {
		return "", 0.0, "", fmt.Errorf("could not pop URL from queue: %w", err)
	}

	// The stored member is the full normalized URL (scheme://host/path?query)
	normalizedURL := result.Z.Member.(string)

	// Use the normalized URL directly as the raw URL for fetching
	return normalizedURL, result.Z.Score, normalizedURL, nil
}

// HasURLBeenVisited checks whether a normalized URL has been crawled before
// by reading the "visited" field from the normalized_url:<url> hash in Redis.
// Returns false if the key or field does not exist.
func (db *Database) HasURLBeenVisited(ctx context.Context, normalizedURL string) (bool, error) {
	normalized_url_key := utils.NormalizedURLPrefix + ":" + normalizedURL
	result, err := db.Client.HGet(ctx, normalized_url_key, "visited").Result()

	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("could not fetch %v from Redis: %w", normalized_url_key, err)
	}

	visited, err := strconv.Atoi(result)
	if err != nil {
		return false, fmt.Errorf("could not parse 'visited' value: %w", err)
	}

	return visited != 0, nil
}

// VisitPage marks a normalized URL as visited by setting the "visited" field
// to 1 in the normalized_url:<url> hash. This prevents future workers from
// re-crawling the same URL.
func (db *Database) VisitPage(ctx context.Context, normalizedURL string) error {
	normalized_url_key := utils.NormalizedURLPrefix + ":" + normalizedURL
	_, err := db.Client.HSet(ctx, normalized_url_key, "visited", 1).Result()

	if err != nil {
		return fmt.Errorf("could not update visit %v from Redis: %w", normalized_url_key, err)
	}

	return nil
}
