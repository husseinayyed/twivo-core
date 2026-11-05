package twivo

import (
    "fmt"
    lru "github.com/hashicorp/golang-lru/v2"
    "github.com/linvon/cuckoo-filter"
)

// cacheWrapper is a non-generic interface so different T types can live in the same map.
type cacheWrapper interface {
    Get(key string) (any, bool)
    Add(key string, value any) error
    Len() int
}

// params manages the cuckoo filters and wrapped caches.
type params struct {
    cuckooFilter map[string]*cuckoo.Filter
    lruCache     map[string]cacheWrapper
    currentSize  int
}

// Core is the global instance managing all keys and caches.
var Core = params{
    cuckooFilter: make(map[string]*cuckoo.Filter),
    lruCache:     make(map[string]cacheWrapper),
}

// typedCache implements cacheWrapper while preserving strong typing for T internally.
type typedCache[T any] struct {
    cache *lru.Cache[string, T]
}

func (tc *typedCache[T]) Get(key string) (any, bool) {
    return tc.cache.Get(key)
}

func (tc *typedCache[T]) Add(key string, value any) error {
    v, ok := value.(T)
    if !ok {
        return fmt.Errorf("invalid type: expected %T, got %T", (*T)(nil), value)
    }
    tc.cache.Add(key, v)
    return nil
}

func (tc *typedCache[T]) Len() int {
    return tc.cache.Len()
}

// NewKey creates a strongly-typed LRU cache for a specific key.
func (c *params) NewKey[T any](key string, cacheSize int, _ int, _ int) error {
    if c.cuckooFilter == nil {
        c.cuckooFilter = make(map[string]*cuckoo.Filter)
    }
    if c.lruCache == nil {
        c.lruCache = make(map[string]cacheWrapper)
    }

    filter := c.cuckooFilter[key]
    if filter == nil {
        filter = cuckoo.NewFilter(4, 12, uint(cacheSize), cuckoo.TableTypeSingle)
        c.cuckooFilter[key] = filter
    }

    cache, err := lru.NewWithEvict[string, T](cacheSize, func(itemKey string, _ T) {
        filter.Delete([]byte(itemKey))
    })
    if err != nil {
        return err
    }

    c.lruCache[key] = &typedCache[T]{cache: cache}
    return nil
}

// GetTyped retrieves and safely casts the value to type T.
func (c *params) GetTyped[T any](key string, itemKey string) (T, bool) {
    var zero T
    wrapped, exists := c.lruCache[key]
    if !exists {
        return zero, false
    }
    val, found := wrapped.Get(itemKey)
    if !found {
        return zero, false
    }
    typedVal, ok := val.(T)
    if !ok {
        return zero, false
    }
    return typedVal, true
}