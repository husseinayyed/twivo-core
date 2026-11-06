package twivo

import (
	"context"
	"time"

	"github.com/husseinayyed/twivo-core/pkg/bootstrap"
	"github.com/husseinayyed/twivo-core/pkg/breaker"
	"github.com/husseinayyed/twivo-core/pkg/shard"
)

const cacheOperationTimeout = 3 * time.Second

/*
	 runWithContext executes the provided operation function with a context that has a specified timeout. If the context is nil, a background context is used. The function returns the result of the operation and any error encountered during its execution.
		- Parameters:
			- ctx: The context to use for the operation. If nil, a background context is used.
			- timeout: The duration after which the context will be canceled.
			- operation: A function that takes a context and returns a result and an error.
		- Returns:
			- The result of the operation (if successful) and any error encountered during its execution.
*/
func runWithContext(ctx context.Context, timeout time.Duration, operation func(context.Context) (any, error)) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	timerCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case <-timerCtx.Done():
		return nil, timerCtx.Err()
	default:
	}

	result, err := operation(timerCtx)
	if timerCtx.Err() != nil {
		return nil, timerCtx.Err()
	}
	return result, err
}

/*
	 SetCache sets a key-value pair in the cache with a specified time-to-live (TTL). It first writes the value to the local shard and then attempts to set it in Redis. If the Redis operation fails, it returns an error.
		- Parameters:
			- ctx: The context to use for the operation. If nil, a background context is used.
			- key: The key to set in the cache.
			- value: The value to associate with the key.
			- ttl: The time-to-live for the cache entry, in seconds.
		- Returns:
			- An error if the operation fails; otherwise, nil.
*/
func SetCache(ctx context.Context, key string, value any, ttl int) error {
	_, err := runWithContext(ctx, cacheOperationTimeout, func(ctx context.Context) (any, error) {
		shard.Write(key, value, false)
		return breaker.Group.Execute("Redis", func() (any, error) {
			return nil, bootstrap.Redis_Client.Set(ctx, key, value, time.Duration(ttl)*time.Second).Err()
		})
	})
	return err
}

/*
	 GetCache retrieves a value from the cache for a given key. It first checks the local shard for the value; if not found, it attempts to retrieve it from Redis. If the Redis operation fails, it returns an error.
		- Parameters:
			- ctx: The context to use for the operation. If nil, a background context is used.
			- key: The key to retrieve from the cache.
		- Returns:
			- The value associated with the key (if found) and an error if the operation fails; otherwise, nil.
*/
func GetCache(ctx context.Context, key string) (any, error) {
	return runWithContext(ctx, cacheOperationTimeout, func(ctx context.Context) (any, error) {
		if value, found := shard.Read(key); found {
			return value, nil
		}

		value, err := breaker.Group.Execute("Redis", func() (any, error) {
			return bootstrap.Redis_Client.Get(ctx, key).Result()
		})
		if err != nil {
			return nil, err
		}

		shard.Write(key, value, false)
		return value, nil
	})
}
