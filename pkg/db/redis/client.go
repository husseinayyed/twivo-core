package redis

import (
	"context"
	"errors"
	"time"

	"github.com/husseinayyed/twivo-core/pkg/breaker"
	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)


func Connect(redisURL string) (*goredis.Client, error) {
	redisHost := redisURL

	if redisHost == "" {
		return nil, errors.New("missing required environment variable REDIS_URL for Redis connection")
	}

	result, err := breaker.Group.Execute("Redis", func() (any, error) {
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
		return nil, err
	}

	client, ok := result.(*goredis.Client)
	if !ok {
		return nil, errors.New("redis breaker returned an invalid client type")
	}

	log.Info().Str("service", "Redis").Msg("connected to Redis successfully")
	return client, nil
}
