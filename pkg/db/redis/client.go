package redis

import (
	"context"
	"os"
	"time"

	"github.com/husseinayyed/twivo-core/pkg/breaker"
	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var (
	RedisClient *goredis.Client
	REDIS_URL   = os.Getenv("REDIS_URL")
)

func Connect() (*goredis.Client, error) {
	redisHost := REDIS_URL

	if redisHost == "" {
		log.Fatal().Msg("REDIS_URL environment variable must be set")
	}

	result, err := breaker.Redis.Execute(func() (any, error) {
		opt, err := goredis.ParseURL(redisHost)
		if err != nil {
			return nil, err
		}

		opt.PoolSize = 20
		opt.MinIdleConns = 5
		opt.MaxIdleConns = 10
		opt.ConnMaxIdleTime = 5 * time.Minute

		client := goredis.NewClient(opt)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.Ping(ctx).Err(); err != nil {
			client.Close()
			return nil, err
		}

		return client, nil
	})
	if err != nil {
		log.Fatal().Err(err).Msg("error connecting to Redis")
		os.Exit(1)
	}

	client, ok := result.(*goredis.Client)
	if !ok {
		log.Fatal().Msg("redis breaker returned an invalid client type")
	}

	RedisClient = client
	log.Info().Msg("connected to Redis successfully")
	return RedisClient, nil
}
