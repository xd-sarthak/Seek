package database

import (
	"context"
	"fmt"
	"log"
	"spider/internal/utils"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// setting up redis
type Database struct {
	Client  *redis.Client
	Context context.Context
}

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

	db.Context = context.Background()

	_, err = db.Client.Ping(db.Context).Result()
	if err != nil {
		return fmt.Errorf("Couldn't connect to shit [%v, %v]: %v", redisHost, redisPassword, err)
	}

	log.Println("Successfully connected to Redis!")
	return nil
}

// pushing url method
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
	err = db.Client.ZAdd(db.Context, utils.SpiderQueueKey, redis.Z{
		Score:  score,
		Member: normalizedURL,
	}).Err()

	if err != nil {
		return fmt.Errorf("Could not add URL to spider queue: %w", err)
	}

	fmt.Print("Pushed %v (%v) to queue\n", rawURL, normalizedURL)

	return nil
}

// check if exists in Queue
func (db *Database) ExistsInQueue(rawURL string) (float64, error) {
	normalizedURL, err := utils.NormalizeURL(rawURL)
	if err != nil {
		return 0.0, err
	}

	exists, err := db.Client.ZScore(db.Context, utils.SpiderQueueKey, normalizedURL).Result()
	if err != nil {
		return 0.0, err
	}

	return exists, nil
}

// get indexer queue size
func (db *Database) GetIndexerQueueSize() (int64, error) {
	size, err := db.Client.LLen(db.Context, utils.IndexerQueueKey).Result()

	if err != nil {
		return -1, fmt.Errorf("could not get %v size: %v", utils.IndexerQueueKey, err)
	}

	return size, nil
}

// pop signal
func (db *Database) PopSignalQueue() (string, error) {
	signal, err := db.Client.BRPop(db.Context, 0, utils.SignalQueueKey).Result()
	if err != nil {
		return "", fmt.Errorf("Coult not pop from signal queue: %v\n", err)
	}
	return signal[1], nil
}


//get next entry in the queue
func (db *Database) PopURL() (string, float64, string, error) {
	//get the next normalized url from the priority queue
	result, err := db.Client.BZPopMin(db.Context, utils.Timeout, utils.SpiderQueueKey).Result()
	if err != nil {
		return "", 0.0, "", fmt.Errorf("Could not pop URL from queue: %v\n", err)
	}

	normalizedURL := result.Z.Member.(string)
	raw_url := fmt.Sprintf("https://%v", normalizedURL)

	// Raw url is just the normalized url + https:// so we'll do it manually due to performance issues

	return raw_url, result.Z.Score, normalizedURL, nil
}

// ------------------- VISIT PAGE -------------------
func (db *Database) HasURLBeenVisited(normalizedURL string) (bool, error) {
	// FIXME: This is a temporary fix
	return false, nil
	normalized_url_key := utils.NormalizedURLPrefix + ":" + normalizedURL
	result, err := db.Client.HGet(db.Context, normalized_url_key, "visited").Result()

	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("Could not fetch %v from Redis: %v\n", normalized_url_key, err)
	}

	visited, err := strconv.Atoi(result)
	if err != nil {
		return false, fmt.Errorf("Could not parse 'visited' value: %v", err)
	}

	if visited == 0 {
		return false, nil
	}

	return true, nil
}


func (db *Database) VisitPage(normalizedURL string) error {
	return nil // FIXME: This is a temporary fix
	normalized_url_key := utils.NormalizedURLPrefix + ":" + normalizedURL
	_, err := db.Client.HSet(db.Context, normalized_url_key, "visited", 1).Result()

	if err != nil {
		return fmt.Errorf("Could not update visit %v from Redis: %v\n", normalized_url_key, err)
	}

	return nil
}
