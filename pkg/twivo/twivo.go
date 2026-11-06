package twivo

import (
	"github.com/husseinayyed/twivo-core/pkg/bootstrap"
	"github.com/husseinayyed/twivo-core/pkg/shard"
)

/*
SetCache sets a key-value pair in the cache with a specified time-to-live (TTL). It writes the key-value pair to the shard and also attempts to set it in Redis. If Redis is unavailable, it will still store the value in the shard.
  - Key: The key to be set in the cache.
  - Value: The value associated with the key.
*/
func SetCache(key string, value any, ttl int) error {
	shard.Write(key, value, false)
	bootstrap.Breaker("redis", func() error {
		return bootstrap.Redis.Set(key, value, ttl).Err()
	})
	return nil
}

/*
GetCache retrieves a value from the cache based on the provided key. It first checks if the key exists in the shard. If it does, it returns the value. If not, it attempts to retrieve the value from Redis. If Redis is unavailable or the key does not exist in Redis, it will return an error.
  - Key: The key to be looked up in the cache.
  - Returns: The value associated with the key (if found) and an error if any occurred during the retrieval process.
*/
func GetCache(ctx context.Context, key string) (any, error) {
	if _, okay := ctx.DeadLine(); !okay {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		val, exists := shard.Read(key)
		if exists {
			return val, nil
		}
		var value any
		exists, err := bootstrap.Breaker("redis", func() error {
			var err error
			value, err = bootstrap.Redis.Get(key).Result()
			return err
		})
		if err != nil {
			return nil, err
		}
		shard.Write(key, value, false)
		return value, nil
	}

}
