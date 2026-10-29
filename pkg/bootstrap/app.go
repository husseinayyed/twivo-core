package bootstrap

import (
	"errors"
	"os"

	"github.com/husseinayyed/twivo-core/pkg/breaker"
	"github.com/husseinayyed/twivo-core/pkg/db/mongo"
	"github.com/husseinayyed/twivo-core/pkg/db/redis"
	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	mongodb "go.mongodb.org/mongo-driver/v2/mongo"
)
var (
	Breaker = breaker.NewGroup()
	Mongo_Client *mongodb.Client
	Redis_Client *goredis.Client
)
func getEnv() (string, string, string, string,error) {
	MONGO_URL := os.Getenv("MONGO_URL")
	MONGO_USER := os.Getenv("MONGO_USER")
	MONGO_PASSWORD := os.Getenv("MONGO_PASSWORD")
	REDIS_URL := os.Getenv("REDIS_URL")

	if MONGO_URL == "" || MONGO_USER == "" || MONGO_PASSWORD == "" || REDIS_URL == "" {
		return "", "", "", "",  errors.New("missing required environment variables for MongoDB or Redis connection")
	}
	return MONGO_URL, MONGO_USER, MONGO_PASSWORD, REDIS_URL, nil
}

func Run() {
	var err error
	MONGO_URL, MONGO_USER, MONGO_PASSWORD, REDIS_URL, err := getEnv()
	if err != nil {
		log.Fatal().Msg(err.Error())
		os.Exit(1)
	}

	// Initialize MongoDB connection
	Mongo_Client, err = mongo.Connect(MONGO_URL, MONGO_USER, MONGO_PASSWORD)
	if err != nil {
		log.Error().Service("MongoDB").Msg(err.Error())
	}
	// Initialize Redis connection
	Redis_Client, err = redis.Connect(REDIS_URL)
	if err != nil {
		log.Error().Service("Redis").Msg(err.Error())
	}
}