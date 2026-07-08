// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lru

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockValue implements the ValueType interface for testing
type mockValue struct{ size uint64 }

func (m mockValue) Size() uint64 { return m.size }

func TestRadixCache_InsertNode(t *testing.T) {
	c := NewRadixCache(100).(*radixCache)
	val1 := mockValue{10}

	node1, isNew := c.insertNode("foo/bar", val1)

	assert.True(t, isNew)
	assert.NotNil(t, node1)
	assert.Equal(t, 1, c.size)
}

func TestRadixCache_InsertNil(t *testing.T) {
	c := NewRadixCache(100).(*radixCache)

	node, isNew := c.insertNode("foo/bar", nil)

	assert.False(t, isNew)
	assert.Nil(t, node)
	assert.Equal(t, 0, c.size)
}

func TestRadixCache_GetNode(t *testing.T) {
	c := NewRadixCache(100).(*radixCache)
	val1 := mockValue{10}
	c.insertNode("foo/bar", val1)

	gotNode, ok := c.getNode("foo/bar")

	assert.True(t, ok)
	assert.Equal(t, val1, gotNode.value)
}

func TestRadixCache_OverwriteNode(t *testing.T) {
	c := NewRadixCache(100).(*radixCache)
	c.insertNode("foo/bar", mockValue{10})
	val2 := mockValue{20}

	node2, isNew2 := c.insertNode("foo/bar", val2)

	assert.False(t, isNew2)
	assert.Equal(t, val2, node2.value)
	assert.Equal(t, 1, c.size)
}

func TestRadixCache_GetNonExistent(t *testing.T) {
	c := NewRadixCache(100).(*radixCache)
	c.insertNode("foo/bar", mockValue{10})

	_, ok := c.getNode("foo/baz")

	assert.False(t, ok)
}

func TestRadixCache_PrefixSplitting(t *testing.T) {
	c := NewRadixCache(100).(*radixCache)
	c.insertNode("foo/bar", mockValue{1})

	c.insertNode("foo/baz", mockValue{2}) // Splits "foo/ba" -> "r", "z"
	n1, _ := c.getNode("foo/bar")
	n2, _ := c.getNode("foo/baz")

	assert.Equal(t, 2, c.size)
	assert.Equal(t, n1.parent, n2.parent)
	assert.Equal(t, "foo/ba", n1.parent.prefix)
	assert.Nil(t, n1.parent.value) // Parent is just a routing node
}

func TestRadixCache_DeleteAndCompress(t *testing.T) {
	c := NewRadixCache(100).(*radixCache)
	c.insertNode("foo/bar", mockValue{1})
	c.insertNode("foo/baz", mockValue{2})
	n1, _ := c.getNode("foo/bar")

	c.deleteNode(n1)
	_, ok := c.getNode("foo/bar")
	n2, _ := c.getNode("foo/baz")

	assert.Equal(t, 1, c.size)
	assert.False(t, ok)
	assert.Equal(t, "foo/baz", n2.prefix)
	assert.Equal(t, c.root, n2.parent) // Path compressed all the way up to the root!
}

func TestRadixCache_LRU_PushFront(t *testing.T) {
	c := NewRadixCache(100).(*radixCache)
	val1 := mockValue{10}
	val2 := mockValue{20}
	node1, _ := c.insertNode("foo/1", val1)
	node2, _ := c.insertNode("foo/2", val2)

	c.pushFront(node1)
	c.pushFront(node2)

	assert.Equal(t, 2, c.len)
	assert.Equal(t, node2, c.head)
	assert.Equal(t, node1, c.tail)
	assert.Equal(t, node1, node2.next)
	assert.Equal(t, node2, node1.prev)
}

func TestRadixCache_LRU_MoveToFront(t *testing.T) {
	t.Run("move tail to front", func(t *testing.T) {
		c := NewRadixCache(100).(*radixCache)
		node1, _ := c.insertNode("foo/1", mockValue{10})
		node2, _ := c.insertNode("foo/2", mockValue{20})
		c.pushFront(node1)
		c.pushFront(node2)

		c.moveToFront(node1)

		assert.Equal(t, node1, c.head)
		assert.Equal(t, node2, c.tail)
		assert.Nil(t, node1.prev)
		assert.Equal(t, node2, node1.next)
		assert.Equal(t, node1, node2.prev)
		assert.Nil(t, node2.next)
	})

	t.Run("move middle to front", func(t *testing.T) {
		c := NewRadixCache(100).(*radixCache)
		node1, _ := c.insertNode("foo/1", mockValue{10})
		node2, _ := c.insertNode("foo/2", mockValue{20})
		node3, _ := c.insertNode("foo/3", mockValue{30})
		c.pushFront(node1)
		c.pushFront(node2)
		c.pushFront(node3)

		c.moveToFront(node2)

		assert.Equal(t, node2, c.head)
		assert.Equal(t, node1, c.tail)
		assert.Nil(t, node2.prev)
		assert.Equal(t, node3, node2.next)
		assert.Equal(t, node2, node3.prev)
		assert.Equal(t, node1, node3.next)
		assert.Equal(t, node3, node1.prev)
	})

	t.Run("move head to front", func(t *testing.T) {
		c := NewRadixCache(100).(*radixCache)
		node1, _ := c.insertNode("foo/1", mockValue{10})
		node2, _ := c.insertNode("foo/2", mockValue{20})
		c.pushFront(node1)
		c.pushFront(node2)

		c.moveToFront(node2)

		assert.Equal(t, node2, c.head)
		assert.Equal(t, node1, c.tail)
	})
}

func TestRadixCache_LRU_Remove(t *testing.T) {
	t.Run("remove only node", func(t *testing.T) {
		c := NewRadixCache(100).(*radixCache)
		node1, _ := c.insertNode("foo/1", mockValue{10})
		c.pushFront(node1)

		c.remove(node1)

		assert.Equal(t, 0, c.len)
		assert.Nil(t, c.head)
		assert.Nil(t, c.tail)
	})

	t.Run("remove middle node", func(t *testing.T) {
		c := NewRadixCache(100).(*radixCache)
		node1, _ := c.insertNode("foo/1", mockValue{10})
		node2, _ := c.insertNode("foo/2", mockValue{20})
		node3, _ := c.insertNode("foo/3", mockValue{30})
		c.pushFront(node1)
		c.pushFront(node2)
		c.pushFront(node3)

		c.remove(node2)

		assert.Equal(t, 2, c.len)
		assert.Equal(t, node3, c.head)
		assert.Equal(t, node1, c.tail)
		assert.Equal(t, node1, node3.next)
		assert.Equal(t, node3, node1.prev)
	})

	t.Run("remove node not in list", func(t *testing.T) {
		c := NewRadixCache(100).(*radixCache)
		node1, _ := c.insertNode("foo/1", mockValue{10})
		node2, _ := c.insertNode("foo/2", mockValue{20})
		c.pushFront(node1)

		c.remove(node2)

		assert.Equal(t, 1, c.len)
		assert.Equal(t, node1, c.head)
		assert.Equal(t, node1, c.tail)
	})
}

func TestRadixCache_LRU_EvictOne(t *testing.T) {
	c := NewRadixCache(100).(*radixCache)
	node1, _ := c.insertNode("foo/1", mockValue{10})
	node2, _ := c.insertNode("foo/2", mockValue{20})
	c.pushFront(node1)
	c.pushFront(node2)
	c.currentSize = 30

	evictedValue := c.evictOne()

	assert.Equal(t, 1, c.len)
	assert.Equal(t, uint64(20), c.currentSize)   // 30 - 10
	assert.Equal(t, mockValue{10}, evictedValue) // node1 was tail (least recently used)
	assert.Equal(t, node2, c.head)
	assert.Equal(t, node2, c.tail)
}

////////////////////////////////////////////////////////////////////////
// CACHE INTERFACE TESTS
////////////////////////////////////////////////////////////////////////

const testMaxSize = 50
const testOperationCount = 100

type cacheTestData struct {
	Value    int64
	DataSize uint64
}

func (td cacheTestData) Size() uint64 {
	return td.DataSize
}

func setupRadixCacheTest(t *testing.T) Cache {
	locker.EnableInvariantsCheck()
	return NewRadixCache(testMaxSize)
}

// insertAndAssert inserts the given key,value in the cache and assert based on
// the expected eviction and error.
func insertAndAssert(t *testing.T, cache Cache, key string, val ValueType, evictedValues []int64, expectedError error) {
	ret, err := cache.Insert(key, val)

	require.ErrorIs(t, err, expectedError)
	require.Equal(t, len(evictedValues), len(ret))
	for index, value := range ret {
		assert.Equal(t, evictedValues[index], value.(cacheTestData).Value)
	}
}

func TestRadixCache_LookUpInEmptyCache(t *testing.T) {
	cache := setupRadixCacheTest(t)
	assert.Nil(t, cache.LookUp(""))
	assert.Nil(t, cache.LookUp("taco"))
}

func TestRadixCache_InsertNilValue(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "taco", nil, []int64{}, ErrInvalidEntry)
}

func TestRadixCache_LookUpUnknownKey(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", cacheTestData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", cacheTestData{Value: 23, DataSize: 8}, []int64{}, nil)

	assert.Nil(t, cache.LookUp(""))
	assert.Nil(t, cache.LookUp("enchilada"))
}

func TestRadixCache_FillUpToCapacity(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", cacheTestData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", cacheTestData{Value: 26, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "enchilada", cacheTestData{Value: 28, DataSize: 26}, []int64{}, nil)

	assert.Equal(t, int64(23), cache.LookUp("burrito").(cacheTestData).Value)
	assert.Equal(t, int64(26), cache.LookUp("taco").(cacheTestData).Value)
	assert.Equal(t, int64(28), cache.LookUp("enchilada").(cacheTestData).Value)
}

func TestRadixCache_ExpiresLeastRecentlyUsed(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", cacheTestData{Value: 23, DataSize: 4}, []int64{}, nil)

	// Least recent.
	insertAndAssert(t, cache, "taco", cacheTestData{Value: 26, DataSize: 20}, []int64{}, nil)

	// Second most recent.
	insertAndAssert(t, cache, "enchilada", cacheTestData{Value: 28, DataSize: 26}, []int64{}, nil)

	assert.Equal(t, int64(23), cache.LookUp("burrito").(cacheTestData).Value) // Most recent

	// Insert another.
	insertAndAssert(t, cache, "queso", cacheTestData{Value: 34, DataSize: 5}, []int64{26}, nil)

	// See what's left.
	assert.Nil(t, cache.LookUp("taco"))
	assert.Equal(t, int64(23), cache.LookUp("burrito").(cacheTestData).Value)
	assert.Equal(t, int64(28), cache.LookUp("enchilada").(cacheTestData).Value)
	assert.Equal(t, int64(34), cache.LookUp("queso").(cacheTestData).Value)
}

func TestRadixCache_Overwrite(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", cacheTestData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", cacheTestData{Value: 26, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "enchilada", cacheTestData{Value: 28, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "burrito", cacheTestData{Value: 33, DataSize: 6}, []int64{}, nil)

	// Increase the DataSize while modifying, so eviction should happen
	insertAndAssert(t, cache, "burrito", cacheTestData{Value: 33, DataSize: 12}, []int64{26}, nil)

	assert.Nil(t, cache.LookUp("taco"))
	assert.Equal(t, int64(33), cache.LookUp("burrito").(cacheTestData).Value)
	assert.Equal(t, int64(28), cache.LookUp("enchilada").(cacheTestData).Value)
}

func TestRadixCache_MultipleEviction(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", cacheTestData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", cacheTestData{Value: 26, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "enchilada", cacheTestData{Value: 28, DataSize: 20}, []int64{}, nil)

	// Increase the DataSize while modifying, so eviction should happen
	insertAndAssert(t, cache, "large_data", cacheTestData{Value: 33, DataSize: 45}, []int64{23, 26, 28}, nil)

	assert.Nil(t, cache.LookUp("taco"))
	assert.Nil(t, cache.LookUp("burrito"))
	assert.Nil(t, cache.LookUp("enchilada"))
	assert.Equal(t, int64(33), cache.LookUp("large_data").(cacheTestData).Value)
}

func TestRadixCache_WhenEntrySizeMoreThanCacheMaxSize(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", cacheTestData{Value: 23, DataSize: 4}, []int64{}, nil)

	// Insert entry with size greater than maxSize of cache.
	insertAndAssert(t, cache, "taco", cacheTestData{Value: 26, DataSize: testMaxSize + 1}, []int64{}, ErrInvalidEntrySize)

	assert.Equal(t, int64(23), cache.LookUp("burrito").(cacheTestData).Value)
}

func TestRadixCache_EraseWhenKeyPresent(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", cacheTestData{Value: 23, DataSize: 4}, []int64{}, nil)

	deletedEntry := cache.Erase("burrito")

	assert.Equal(t, int64(23), deletedEntry.(cacheTestData).Value)
	assert.Nil(t, cache.LookUp("burrito"))
}

func TestRadixCache_EraseCacheWithGivenPrefix(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "a", cacheTestData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "a/b", cacheTestData{Value: 26, DataSize: 5}, []int64{}, nil)
	insertAndAssert(t, cache, "a/b/d", cacheTestData{Value: 22, DataSize: 6}, []int64{}, nil)
	insertAndAssert(t, cache, "a/c", cacheTestData{Value: 20, DataSize: 6}, []int64{}, nil)
	insertAndAssert(t, cache, "b", cacheTestData{Value: 21, DataSize: 2}, []int64{}, nil)

	cache.EraseEntriesWithGivenPrefix("a")

	assert.Nil(t, cache.LookUp("a"))
	assert.Nil(t, cache.LookUp("a/b"))
	assert.Nil(t, cache.LookUp("a/b/d"))
	assert.Nil(t, cache.LookUp("a/c"))
	assert.Equal(t, uint64(2), cache.LookUp("b").Size())
}

func TestRadixCache_EraseCacheWhereNoEntriesExistWithGivenPrefix(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "a", cacheTestData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "a/b", cacheTestData{Value: 26, DataSize: 5}, []int64{}, nil)
	insertAndAssert(t, cache, "b", cacheTestData{Value: 21, DataSize: 2}, []int64{}, nil)

	cache.EraseEntriesWithGivenPrefix("c")

	assert.Equal(t, uint64(4), cache.LookUp("a").Size())
	assert.Equal(t, uint64(5), cache.LookUp("a/b").Size())
	assert.Equal(t, uint64(2), cache.LookUp("b").Size())
}

func TestRadixCache_EraseCacheWithGivenPrefixWithSomeEntriesEvictedDueToCacheSize(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "a", cacheTestData{Value: 23, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "a/b", cacheTestData{Value: 26, DataSize: 10}, []int64{}, nil)
	insertAndAssert(t, cache, "a/b/d", cacheTestData{Value: 22, DataSize: 5}, []int64{}, nil)
	insertAndAssert(t, cache, "a/c", cacheTestData{Value: 20, DataSize: 10}, []int64{}, nil)
	insertAndAssert(t, cache, "b", cacheTestData{Value: 21, DataSize: 15}, []int64{23}, nil)

	// As entry "a" was already evicted by the insertion of "b", only three entries will be removed.
	cache.EraseEntriesWithGivenPrefix("a")

	assert.Nil(t, cache.LookUp("a"))
	assert.Nil(t, cache.LookUp("a/b"))
	assert.Nil(t, cache.LookUp("a/b/d"))
	assert.Nil(t, cache.LookUp("a/c"))
	assert.Equal(t, uint64(15), cache.LookUp("b").Size())
}

func TestRadixCache_EraseWhenKeyNotPresent(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", cacheTestData{Value: 23, DataSize: 4}, []int64{}, nil)

	deletedEntry := cache.Erase("taco")
	assert.Nil(t, deletedEntry)

	assert.Equal(t, int64(23), cache.LookUp("burrito").(cacheTestData).Value)
}

func TestRadixCache_UpdateWhenKeyPresent(t *testing.T) {
	cache := setupRadixCacheTest(t)
	key := "burrito"
	data := cacheTestData{Value: 23, DataSize: 4}
	insertAndAssert(t, cache, key, data, []int64{}, nil)
	newData := cacheTestData{Value: 2, DataSize: 4}

	err := cache.UpdateWithoutChangingOrder(key, newData)

	assert.Nil(t, err)
	assert.Equal(t, int64(2), cache.LookUp(key).(cacheTestData).Value)
}

func TestRadixCache_UpdateWhenKeyNotPresent(t *testing.T) {
	cache := setupRadixCacheTest(t)
	key := "burrito"
	data := cacheTestData{Value: 23, DataSize: 4}

	err := cache.UpdateWithoutChangingOrder(key, data)

	assert.ErrorIs(t, err, ErrEntryNotExist)
}

func TestRadixCache_UpdateWhenSizeIsDifferent(t *testing.T) {
	cache := setupRadixCacheTest(t)
	key := "burrito"
	data := cacheTestData{Value: 23, DataSize: 4}
	insertAndAssert(t, cache, key, data, []int64{}, nil)
	newData := cacheTestData{Value: 2, DataSize: 3}

	err := cache.UpdateWithoutChangingOrder(key, newData)

	assert.ErrorIs(t, err, ErrInvalidUpdateEntrySize)
}

func TestRadixCache_UpdateNotChangeOrder(t *testing.T) {
	cache := setupRadixCacheTest(t)
	key1 := "burrito1"
	data1 := cacheTestData{Value: 23, DataSize: 10}
	insertAndAssert(t, cache, key1, data1, []int64{}, nil)
	key2 := "burrito2"
	data2 := cacheTestData{Value: 2, DataSize: 40}
	insertAndAssert(t, cache, key2, data2, []int64{}, nil)

	newData := cacheTestData{Value: 7, DataSize: 10}
	err := cache.UpdateWithoutChangingOrder(key1, newData)

	assert.Nil(t, err)
	// inserting again which should evict key1 because key1 is updated without
	// changing order
	key3 := "burrito3"
	data3 := cacheTestData{Value: 3, DataSize: 5}
	insertAndAssert(t, cache, key3, data3, []int64{7}, nil)
}

func TestRadixCache_LookUpWithoutChangingOrder_WhenKeyPresent(t *testing.T) {
	cache := setupRadixCacheTest(t)
	key := "burrito"
	data := cacheTestData{Value: 23, DataSize: 4}
	insertAndAssert(t, cache, key, data, []int64{}, nil)

	value := cache.LookUpWithoutChangingOrder(key)

	assert.Equal(t, int64(23), value.(cacheTestData).Value)
}

func TestRadixCache_LookUpWithoutChangingOrder_WhenKeyNotPresent(t *testing.T) {
	cache := setupRadixCacheTest(t)
	key := "burrito"

	value := cache.LookUpWithoutChangingOrder(key)

	assert.Nil(t, value)
}

func TestRadixCache_LookUpWithoutChangingOrder_NotChangeOrder(t *testing.T) {
	cache := setupRadixCacheTest(t)
	key1 := "burrito1"
	data1 := cacheTestData{Value: 23, DataSize: 10}
	insertAndAssert(t, cache, key1, data1, []int64{}, nil)
	key2 := "burrito2"
	data2 := cacheTestData{Value: 2, DataSize: 40}
	insertAndAssert(t, cache, key2, data2, []int64{}, nil)

	value := cache.LookUpWithoutChangingOrder(key1)

	assert.Equal(t, int64(23), value.(cacheTestData).Value)
	// inserting again which should evict key1 because key1 is looked up without
	// changing order
	key3 := "burrito3"
	data3 := cacheTestData{Value: 3, DataSize: 5}
	insertAndAssert(t, cache, key3, data3, []int64{23}, nil)
}

// This will detect race if we run the test with `-race` flag.
// We get the race condition failure if we remove lock from Insert or Erase method.
func TestRadixCache_RaceCondition(t *testing.T) {
	cache := setupRadixCacheTest(t)
	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		defer wg.Done()
		for i := range testOperationCount {
			_, err := cache.Insert("key", cacheTestData{
				Value:    int64(i),
				DataSize: uint64(rand.Intn(testMaxSize)),
			})
			assert.NoError(t, err)
		}
	}()

	go func() {
		defer wg.Done()
		for range testOperationCount {
			cache.Erase("key")
		}
	}()

	go func() {
		defer wg.Done()
		for range testOperationCount {
			cache.LookUp("key")
		}
	}()

	go func() {
		defer wg.Done()
		for range testOperationCount {
			cache.LookUpWithoutChangingOrder("key")
		}
	}()

	go func() {
		defer wg.Done()
		for i := range testOperationCount {
			_ = cache.UpdateWithoutChangingOrder("key", cacheTestData{
				Value:    int64(i),
				DataSize: uint64(rand.Intn(testMaxSize)),
			})
		}
	}()

	wg.Wait()
}

func TestRadixCache_EraseEntriesWithGivenPrefix_Concurrent(t *testing.T) {
	c := NewRadixCache(100000)

	// Pre-fill the cache
	for i := range 1000 {
		_, _ = c.Insert(fmt.Sprintf("dir1/file%d", i), cacheTestData{10, 10})
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 1000; i < 2000; i++ {
			_, _ = c.Insert(fmt.Sprintf("dir2/file%d", i), cacheTestData{10, 10})
		}
	}()

	go func() {
		defer wg.Done()
		c.EraseEntriesWithGivenPrefix("dir1/")
	}()

	wg.Wait()
}
