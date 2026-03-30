package lru_test

import (
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stringValue struct {
	val string
}

func (s stringValue) Size() uint64 {
	return uint64(len(s.val))
}

func TestRadixLRU_Basic(t *testing.T) {
	cache := lru.NewRadixCache(100)

	// Insert
	evicted, err := cache.Insert("key1", stringValue{"value1"})
	require.NoError(t, err)
	assert.Empty(t, evicted)

	// Lookup
	val := cache.LookUp("key1")
	assert.NotNil(t, val)
	assert.Equal(t, "value1", val.(stringValue).val)

	// Update
	err = cache.UpdateWithoutChangingOrder("key1", stringValue{"value2"})
	require.NoError(t, err)

	val = cache.LookUp("key1")
	assert.Equal(t, "value2", val.(stringValue).val)

	// Erase
	erased := cache.Erase("key1")
	assert.Equal(t, "value2", erased.(stringValue).val)

	val = cache.LookUp("key1")
	assert.Nil(t, val)
}

func TestRadixLRU_Eviction(t *testing.T) {
	cache := lru.NewRadixCache(10) // capacity 10 bytes

	// Insert 6 bytes
	evicted, err := cache.Insert("k1", stringValue{"123456"})
	require.NoError(t, err)
	assert.Empty(t, evicted)

	// Insert 6 more bytes (total 12) -> should evict k1
	evicted, err = cache.Insert("k2", stringValue{"123456"})
	require.NoError(t, err)
	require.Len(t, evicted, 1)
	assert.Equal(t, "123456", evicted[0].(stringValue).val)

	assert.Nil(t, cache.LookUp("k1"))
	assert.NotNil(t, cache.LookUp("k2"))
}

func TestRadixLRU_ErasePrefix(t *testing.T) {
	cache := lru.NewRadixCache(100)
	_, _ = cache.Insert("a/b/c", stringValue{"1"})
	_, _ = cache.Insert("a/b/d", stringValue{"2"})
	_, _ = cache.Insert("a/e", stringValue{"3"})
	_, _ = cache.Insert("f/g", stringValue{"4"})

	cache.EraseEntriesWithGivenPrefix("a/b/")

	assert.Nil(t, cache.LookUp("a/b/c"))
	assert.Nil(t, cache.LookUp("a/b/d"))
	assert.NotNil(t, cache.LookUp("a/e"))
	assert.NotNil(t, cache.LookUp("f/g"))
}

func TestRadixLRU_Concurrency(t *testing.T) {
	cache := lru.NewRadixCache(1000)
	var wg sync.WaitGroup

	// Concurrent Inserts
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key" + string(rune(i))
			_, _ = cache.Insert(key, stringValue{"val"})
		}(i)
	}
	wg.Wait()

	// Concurrent Lookups and Erases
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key" + string(rune(i))
			if i%2 == 0 {
				cache.LookUp(key)
			} else {
				cache.Erase(key)
			}
		}(i)
	}
	wg.Wait()

	// Test passes if no races or panics
}
