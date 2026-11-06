package shard

import (
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/linvon/cuckoo-filter"
	"sync"
)

type ShardManager struct {
	lastId    int
	lastCount int
	mutex     sync.RWMutex
	lru       []*lru.Cache[string, any]
	cuckoo    []*cuckoo.Filter
}

var (
	// GlobalShard is the global instance managing all keys and caches.
	Shard = newShard()
)

func appendShard() {
	lruCache, _ := lru.New[string, any](100000)
	Shard.lru = append(Shard.lru, lruCache)
	cuckooFilter := cuckoo.NewFilter(4, 8, 100000, cuckoo.TableTypeSingle)
	Shard.cuckoo = append(Shard.cuckoo, cuckooFilter)
}
func newShard() *ShardManager {
	manager := &ShardManager{
		lastId:    0,
		lastCount: 0,
		lru:       make([]*lru.Cache[string, any], 0),
		cuckoo:    make([]*cuckoo.Filter, 0),
	}

	lruCache, _ := lru.New[string, any](100000)
	manager.lru = append(manager.lru, lruCache)

	cuckooFilter := cuckoo.NewFilter(4, 8, 100000, cuckoo.TableTypeSingle)
	manager.cuckoo = append(manager.cuckoo, cuckooFilter)
	return manager
}

/*
	 ReadCuckoo checks if a key exists in the cuckoo filter. It returns true if the key is found, and false otherwise. If the filter is empty, it will return false.
		- Key: The key to be checked in the cuckoo filter.
		- Returns: A boolean indicating whether the key exists in the filter (true) or not (false).
*/
func ReadCuckoo(key string) bool {
	Shard.mutex.RLock()
	defer Shard.mutex.RUnlock()
	if Shard.lastId == 0 {
		return Shard.cuckoo[Shard.lastId].Contain([]byte(key))
	} else {
		for i := 0; i <= Shard.lastId; i++ {
			if Shard.cuckoo[i].Contain([]byte(key)) {
				return true
			}
		}
	}
	return false
}

/*
	 WriteCuckoo adds a key to the cuckoo filter. It returns true if the key already exists in the filter, and false if it was added successfully. If the filter is full, it will create a new filter and add the key to it.
		- Key: The key to be added to the cuckoo filter.
		- Returns: A boolean indicating whether the key already exists in the filter (true) or was added successfully (false).
*/
func WriteCuckoo(key string) bool {
	Shard.mutex.Lock()
	defer Shard.mutex.Unlock()
	if Shard.lastCount >= 100000 {
		Shard.lastId++
		Shard.lastCount = 0
		appendShard()
	}
	if Shard.lastId == 0 {
		Shard.cuckoo[0].Add([]byte(key))
	} else {

		for i := 0; i <= Shard.lastId; i++ {
			if !Shard.cuckoo[i].Contain([]byte(key)) {
				Shard.cuckoo[i].Add([]byte(key))
				return false
			}
		}
	}
	return true
}

/*
	 ReadLRU retrieves a value from the LRU cache based on the provided key. It returns the value and a boolean indicating whether the key was found in the cache. If the cache is empty, it will return nil and false.
		- Key: The key to be looked up in the LRU cache.
		- Returns: The value associated with the key (if found) and a boolean indicating whether the key exists in the cache (true) or not (false).
*/
func ReadLRU(key string) (any, bool) {
	Shard.mutex.RLock()
	defer Shard.mutex.RUnlock()
	if Shard.lastId == 0 {
		return Shard.lru[Shard.lastId].Get(key)
	} else {
		for i := 0; i <= Shard.lastId; i++ {
			if value, ok := Shard.lru[i].Get(key); ok {
				return value, true
			}
		}
	}
	return nil, false
}

/*
	 WriteLRU adds a key-value pair to the LRU cache. If the cache is full, it will create a new shard and add the pair to it.
		- Key: The key to be added to the LRU cache.
		- Value: The value to be added to the LRU cache.
		- Force: A boolean indicating whether to force the addition of the key-value pair even if it already exists in the cache.
*/
func WriteLRU(key string, value any, force bool) {
	Shard.mutex.Lock()
	defer Shard.mutex.Unlock()
	if Shard.lastCount >= 100000 {
		Shard.lastId++
		Shard.lastCount = 0
		appendShard()
	}
	if Shard.lastId == 0 {
		Shard.lru[Shard.lastId].Add(key, value)
	} else {
		for i := 0; i <= Shard.lastId; i++ {
			if _, ok := Shard.lru[i].Get(key); !ok || force {
				Shard.lru[i].Add(key, value)
				return
			}
		}
	}
}

/*
	 Read retrieves a value from the LRU cache based on the provided key. It first checks if the key exists in the cuckoo filter. If it does, it will then look up the value in the LRU cache. If the key does not exist in the cuckoo filter, it will return nil and false.
		- Key: The key to be looked up in the LRU cache.
		- Returns: The value associated with the key (if found) and a boolean indicating whether the key exists in the cache (true) or not (false).
*/
func Read(key string, value any) (any, bool) {
	exists := ReadCuckoo(key)
	if !exists {
		return nil, false
	}
	return ReadLRU(key)
}

/*
	 Write adds a key-value pair to both the cuckoo filter and the LRU cache. It first checks if the key already exists in the cuckoo filter. If it does, it will not add the key again. If it doesn't exist, it will add the key to both the cuckoo filter and the LRU cache.
		- Key: The key to be added.
		- Value: The value to be associated with the key.
		- Force: A boolean indicating whether to force the addition of the key-value pair even if it already exists in the cache.
*/
func Write(key string, value any, force bool) {
	WriteCuckoo(key)
	WriteLRU(key, value, force)
}