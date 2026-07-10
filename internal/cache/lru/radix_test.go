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

package lru_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/stretchr/testify/assert"
)

////////////////////////////////////////////////////////////////////////
// CACHE INTERFACE TESTS
////////////////////////////////////////////////////////////////////////

const testMaxSize = 50
const testOperationCount = 100

func setupRadixCacheTest(t *testing.T) lru.Cache {
	locker.EnableInvariantsCheck()
	return lru.NewRadixCache(testMaxSize)
}

func TestRadixCache_LookUpInEmptyCache(t *testing.T) {
	cache := setupRadixCacheTest(t)
	assert.Nil(t, cache.LookUp(""))
	assert.Nil(t, cache.LookUp("taco"))
}

func TestRadixCache_InsertNilValue(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "taco", nil, []int64{}, lru.ErrInvalidEntry)
}

func TestRadixCache_InsertEmptyKey(t *testing.T) {
	cache := setupRadixCacheTest(t)

	insertAndAssert(t, cache, "", testData{Value: 42, DataSize: 10}, []int64{}, nil)

	assert.Equal(t, int64(42), cache.LookUp("").(testData).Value)
	assert.Nil(t, cache.LookUp("taco"))
}

func TestRadixCache_LookUpUnknownKey(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", testData{Value: 23, DataSize: 8}, []int64{}, nil)

	assert.Nil(t, cache.LookUp(""))
	assert.Nil(t, cache.LookUp("enchilada"))
}

func TestRadixCache_FillUpToCapacity(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", testData{Value: 26, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "enchilada", testData{Value: 28, DataSize: 26}, []int64{}, nil)

	assert.Equal(t, int64(23), cache.LookUp("burrito").(testData).Value)
	assert.Equal(t, int64(26), cache.LookUp("taco").(testData).Value)
	assert.Equal(t, int64(28), cache.LookUp("enchilada").(testData).Value)
}

func TestRadixCache_ExpiresLeastRecentlyUsed(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)

	// Least recent.
	insertAndAssert(t, cache, "taco", testData{Value: 26, DataSize: 20}, []int64{}, nil)

	// Second most recent.
	insertAndAssert(t, cache, "enchilada", testData{Value: 28, DataSize: 26}, []int64{}, nil)

	assert.Equal(t, int64(23), cache.LookUp("burrito").(testData).Value) // Most recent

	// Insert another.
	insertAndAssert(t, cache, "queso", testData{Value: 34, DataSize: 5}, []int64{26}, nil)

	// See what's left.
	assert.Nil(t, cache.LookUp("taco"))
	assert.Equal(t, int64(23), cache.LookUp("burrito").(testData).Value)
	assert.Equal(t, int64(28), cache.LookUp("enchilada").(testData).Value)
	assert.Equal(t, int64(34), cache.LookUp("queso").(testData).Value)
}

func TestRadixCache_Overwrite(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", testData{Value: 26, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "enchilada", testData{Value: 28, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "burrito", testData{Value: 33, DataSize: 6}, []int64{}, nil)

	// Increase the DataSize while modifying, so eviction should happen
	insertAndAssert(t, cache, "burrito", testData{Value: 33, DataSize: 12}, []int64{26}, nil)

	assert.Nil(t, cache.LookUp("taco"))
	assert.Equal(t, int64(33), cache.LookUp("burrito").(testData).Value)
	assert.Equal(t, int64(28), cache.LookUp("enchilada").(testData).Value)
}

func TestRadixCache_MultipleEviction(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", testData{Value: 26, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "enchilada", testData{Value: 28, DataSize: 20}, []int64{}, nil)

	// Increase the DataSize while modifying, so eviction should happen
	insertAndAssert(t, cache, "large_data", testData{Value: 33, DataSize: 45}, []int64{23, 26, 28}, nil)

	assert.Nil(t, cache.LookUp("taco"))
	assert.Nil(t, cache.LookUp("burrito"))
	assert.Nil(t, cache.LookUp("enchilada"))
	assert.Equal(t, int64(33), cache.LookUp("large_data").(testData).Value)
}

func TestRadixCache_WhenEntrySizeMoreThanCacheMaxSize(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)

	// Insert entry with size greater than maxSize of cache.
	insertAndAssert(t, cache, "taco", testData{Value: 26, DataSize: testMaxSize + 1}, []int64{}, lru.ErrInvalidEntrySize)

	assert.Equal(t, int64(23), cache.LookUp("burrito").(testData).Value)
}

func TestRadixCache_EraseWhenKeyPresent(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)

	deletedEntry := cache.Erase("burrito")

	assert.Equal(t, int64(23), deletedEntry.(testData).Value)
	assert.Nil(t, cache.LookUp("burrito"))
}

func TestRadixCache_EraseCacheWithGivenPrefix(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "a", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "a/b", testData{Value: 26, DataSize: 5}, []int64{}, nil)
	insertAndAssert(t, cache, "a/b/d", testData{Value: 22, DataSize: 6}, []int64{}, nil)
	insertAndAssert(t, cache, "a/c", testData{Value: 20, DataSize: 6}, []int64{}, nil)
	insertAndAssert(t, cache, "b", testData{Value: 21, DataSize: 2}, []int64{}, nil)

	cache.EraseEntriesWithGivenPrefix("a")

	assert.Nil(t, cache.LookUp("a"))
	assert.Nil(t, cache.LookUp("a/b"))
	assert.Nil(t, cache.LookUp("a/b/d"))
	assert.Nil(t, cache.LookUp("a/c"))
	assert.Equal(t, uint64(2), cache.LookUp("b").Size())
}

func TestRadixCache_EraseCacheWithEmptyPrefix(t *testing.T) {
	cache := setupRadixCacheTest(t)

	insertAndAssert(t, cache, "a", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "a/b", testData{Value: 26, DataSize: 5}, []int64{}, nil)
	insertAndAssert(t, cache, "b", testData{Value: 21, DataSize: 2}, []int64{}, nil)

	cache.EraseEntriesWithGivenPrefix("")

	assert.Nil(t, cache.LookUp("a"))
	assert.Nil(t, cache.LookUp("a/b"))
	assert.Nil(t, cache.LookUp("b"))
}

func TestRadixCache_EraseCacheWhereNoEntriesExistWithGivenPrefix(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "a", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "a/b", testData{Value: 26, DataSize: 5}, []int64{}, nil)
	insertAndAssert(t, cache, "b", testData{Value: 21, DataSize: 2}, []int64{}, nil)

	cache.EraseEntriesWithGivenPrefix("c")

	assert.Equal(t, uint64(4), cache.LookUp("a").Size())
	assert.Equal(t, uint64(5), cache.LookUp("a/b").Size())
	assert.Equal(t, uint64(2), cache.LookUp("b").Size())
}

func TestRadixCache_EraseCacheWithGivenPrefixWithSomeEntriesEvictedDueToCacheSize(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "a", testData{Value: 23, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "a/b", testData{Value: 26, DataSize: 10}, []int64{}, nil)
	insertAndAssert(t, cache, "a/b/d", testData{Value: 22, DataSize: 5}, []int64{}, nil)
	insertAndAssert(t, cache, "a/c", testData{Value: 20, DataSize: 10}, []int64{}, nil)
	insertAndAssert(t, cache, "b", testData{Value: 21, DataSize: 15}, []int64{23}, nil)

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
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)

	deletedEntry := cache.Erase("taco")
	assert.Nil(t, deletedEntry)

	assert.Equal(t, int64(23), cache.LookUp("burrito").(testData).Value)
}

func TestRadixCache_UpdateSize(t *testing.T) {
	t.Run("NonExistentKey", func(t *testing.T) {
		cache := lru.NewRadixCache(100)

		err := cache.UpdateSize("key1", 20)

		assert.ErrorIs(t, err, lru.ErrEntryNotExist)
	})

	t.Run("Immediate Eviction", func(t *testing.T) {
		cache := lru.NewRadixCache(100)
		data1 := testData{Value: 1, DataSize: 10}
		data2 := testData{Value: 2, DataSize: 70}

		_, _ = cache.Insert("key1", data1)
		_, _ = cache.Insert("key2", data2)

		errUpdate := cache.UpdateSize("key1", 30)

		assert.NoError(t, errUpdate)
		assert.Nil(t, cache.LookUp("key1"))
		assert.NotNil(t, cache.LookUp("key2"))
	})
}

func TestRadixCache_UpdateWhenKeyPresent(t *testing.T) {
	cache := setupRadixCacheTest(t)
	key := "burrito"
	data := testData{Value: 23, DataSize: 4}
	insertAndAssert(t, cache, key, data, []int64{}, nil)
	newData := testData{Value: 2, DataSize: 4}

	err := cache.UpdateWithoutChangingOrder(key, newData)

	assert.Nil(t, err)
	assert.Equal(t, int64(2), cache.LookUp(key).(testData).Value)
}

func TestRadixCache_UpdateWhenKeyNotPresent(t *testing.T) {
	cache := setupRadixCacheTest(t)
	key := "burrito"
	data := testData{Value: 23, DataSize: 4}

	err := cache.UpdateWithoutChangingOrder(key, data)

	assert.ErrorIs(t, err, lru.ErrEntryNotExist)
}

func TestRadixCache_UpdateWhenSizeIsDifferent(t *testing.T) {
	cache := setupRadixCacheTest(t)
	key := "burrito"
	data := testData{Value: 23, DataSize: 4}
	insertAndAssert(t, cache, key, data, []int64{}, nil)
	newData := testData{Value: 2, DataSize: 3}

	err := cache.UpdateWithoutChangingOrder(key, newData)

	assert.ErrorIs(t, err, lru.ErrInvalidUpdateEntrySize)
}

func TestRadixCache_UpdateNotChangeOrder(t *testing.T) {
	cache := setupRadixCacheTest(t)
	key1 := "burrito1"
	data1 := testData{Value: 23, DataSize: 10}
	insertAndAssert(t, cache, key1, data1, []int64{}, nil)
	key2 := "burrito2"
	data2 := testData{Value: 2, DataSize: 40}
	insertAndAssert(t, cache, key2, data2, []int64{}, nil)

	newData := testData{Value: 7, DataSize: 10}
	err := cache.UpdateWithoutChangingOrder(key1, newData)

	assert.Nil(t, err)
	// inserting again which should evict key1 because key1 is updated without
	// changing order
	key3 := "burrito3"
	data3 := testData{Value: 3, DataSize: 5}
	insertAndAssert(t, cache, key3, data3, []int64{7}, nil)
}

func TestRadixCache_LookUpWithoutChangingOrder_WhenKeyPresent(t *testing.T) {
	cache := setupRadixCacheTest(t)
	key := "burrito"
	data := testData{Value: 23, DataSize: 4}
	insertAndAssert(t, cache, key, data, []int64{}, nil)

	value := cache.LookUpWithoutChangingOrder(key)

	assert.Equal(t, int64(23), value.(testData).Value)
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
	data1 := testData{Value: 23, DataSize: 10}
	insertAndAssert(t, cache, key1, data1, []int64{}, nil)
	key2 := "burrito2"
	data2 := testData{Value: 2, DataSize: 40}
	insertAndAssert(t, cache, key2, data2, []int64{}, nil)

	value := cache.LookUpWithoutChangingOrder(key1)

	assert.Equal(t, int64(23), value.(testData).Value)
	// inserting again which should evict key1 because key1 is looked up without
	// changing order
	key3 := "burrito3"
	data3 := testData{Value: 3, DataSize: 5}
	insertAndAssert(t, cache, key3, data3, []int64{23}, nil)
}

// This will detect race if we run the test with `-race` flag.
// We get the race condition failure if we remove lock from Insert or Erase method.
func TestRadixCache_RaceCondition(t *testing.T) {
	cache := setupRadixCacheTest(t)
	var wg sync.WaitGroup
	wg.Add(7)

	go func() {
		defer wg.Done()
		for range testOperationCount {
			_ = cache.UpdateSize("key", uint64(rand.Intn(testMaxSize)))
		}
	}()

	go func() {
		defer wg.Done()
		for range testOperationCount {
			cache.EraseEntriesWithGivenPrefix("k")
		}
	}()

	go func() {
		defer wg.Done()
		for i := range testOperationCount {
			_, err := cache.Insert("key", testData{
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
			_ = cache.UpdateWithoutChangingOrder("key", testData{
				Value:    int64(i),
				DataSize: uint64(rand.Intn(testMaxSize)),
			})
		}
	}()

	wg.Wait()
}

func TestRadixCache_EraseEntriesWithGivenPrefix_Concurrent(t *testing.T) {
	c := lru.NewRadixCache(100000)

	// Pre-fill the cache
	for i := range 1000 {
		_, _ = c.Insert(fmt.Sprintf("dir1/file%d", i), testData{10, 10})
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 1000; i < 2000; i++ {
			_, _ = c.Insert(fmt.Sprintf("dir2/file%d", i), testData{10, 10})
		}
	}()

	go func() {
		defer wg.Done()
		c.EraseEntriesWithGivenPrefix("dir1/")
	}()

	wg.Wait()
}
