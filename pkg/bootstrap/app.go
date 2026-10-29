package bootstrap

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/husseinayyed/twivo-core/pkg/breaker"
	"github.com/husseinayyed/twivo-core/pkg/db/mongo"
	"github.com/husseinayyed/twivo-core/pkg/db/redis"
	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	mongodb "go.mongodb.org/mongo-driver/v2/mongo"
)
var (
	Breaker      = breaker.Group
	Mongo_Client *mongodb.Client
	Redis_Client *goredis.Client
)
func getEnv() (string, string, string, string, error) {
	mongoURL := os.Getenv("MONGO_URL")
	mongoUser := os.Getenv("MONGO_USER")
	mongoPassword := os.Getenv("MONGO_PASSWORD")
	redisAddress := os.Getenv("REDIS_URL")
	redisPassword := os.Getenv("REDIS_PASS")

	if mongoURL == "" || mongoUser == "" || mongoPassword == "" || redisAddress == "" || redisPassword == "" {
		return "", "", "", "", errors.New("missing required environment variables for MongoDB or Redis connection")
	}

	parsedRedisURL, err := url.Parse(redisAddress)
	if err != nil || parsedRedisURL.Scheme == "" || parsedRedisURL.Host == "" {
		return "", "", "", "", errors.New("REDIS_URL must be a valid Redis URL with a host")
	}
	parsedRedisURL.User = url.UserPassword("", redisPassword)

	return mongoURL, mongoUser, mongoPassword, parsedRedisURL.String(), nil
}

// Run initializes MongoDB and Redis and returns any startup error to the caller.
func Run() error {
	MONGO_URL, MONGO_USER, MONGO_PASSWORD, REDIS_URL, err := getEnv()
	if err != nil {
		return err
	}

	// Initialize MongoDB connection
	Mongo_Client, err = mongo.Connect(MONGO_URL, MONGO_USER, MONGO_PASSWORD)
	if err != nil {
		return fmt.Errorf("initialize MongoDB: %w", err)
	}
	// Initialize Redis connection
	Redis_Client, err = redis.Connect(REDIS_URL)
	if err != nil {
		return fmt.Errorf("initialize Redis: %w", err)
	}
	return nil
}

// MustRun initializes MongoDB and Redis, exiting the process if startup fails.
// Use Run when the caller needs to choose how startup errors are handled.
func MustRun() {
	if err := Run(); err != nil {
		log.Fatal().Err(err).Msg("bootstrap failed")
	}
}