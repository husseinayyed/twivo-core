package shard

import (
	"fmt"
	"testing"
)

func TestReadCuckoo(t *testing.T) {
	const key = "testKey"
	const value = "testValue"
	WriteLRU(key, value, false)
	if _, ok := ReadLRU(key); !ok {
		t.Errorf("expected %q to be found in cuckoo filter", key)
	}

	const missing = "nonExistentKey"
	if _, ok := ReadLRU(missing); ok {
		t.Errorf("expected %q not to be found in cuckoo filter", missing)
	}

	const n = 100000
	for i := 0; i < n; i++ {
		k := fmt.Sprintf("key%d", i)
		WriteLRU(k, fmt.Sprintf("value%d", i), false)
	}

	if val, ok := ReadLRU(key); !ok || val != value {
		t.Errorf("expected %q to have value %q, got %q", key, value, val)
	}

	var missingCount int
	for i := 0; i < n; i++ {
		k := fmt.Sprintf("key%d", i)
		if _, ok := ReadLRU(k); !ok {
			missingCount++
		}
	}
	if missingCount > 0 {
		t.Errorf("%d of %d inserted keys were not found (filter may be full or evicting)", missingCount, n)
	}
}