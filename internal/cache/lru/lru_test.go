// Copyright 2023 Google LLC
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
	"github.com/stretchr/testify/require"
)

const MaxSize = 50
const OperationCount = 100

type testData struct {
	Value    int64
	DataSize uint64
}

func (td testData) Size() uint64 {
	return td.DataSize
}

func setupCacheTest(t *testing.T) lru.Cache {
	locker.EnableInvariantsCheck()
	return lru.NewCache(MaxSize)
}

// insertAndAssert inserts the given key,value in the cache and assert based on
// the expected eviction and error.
func insertAndAssert(t *testing.T, cache lru.Cache, key string, val lru.ValueType, evictedValues []int64, expectedError error) {
	ret, err := cache.Insert(key, val)

	require.ErrorIs(t, err, expectedError)
	require.Equal(t, len(evictedValues), len(ret))
	for index, value := range ret {
		assert.Equal(t, evictedValues[index], value.(testData).Value)
	}
}

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func TestLookUpInEmptyCache(t *testing.T) {
	cache := setupCacheTest(t)
	assert.Nil(t, cache.LookUp(""))
	assert.Nil(t, cache.LookUp("taco"))
}

func TestInsertNilValue(t *testing.T) {
	cache := setupCacheTest(t)
	insertAndAssert(t, cache, "taco", nil, []int64{}, lru.ErrInvalidEntry)
}

func TestLookUpUnknownKey(t *testing.T) {
	cache := setupCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", testData{Value: 23, DataSize: 8}, []int64{}, nil)

	assert.Nil(t, cache.LookUp(""))
	assert.Nil(t, cache.LookUp("enchilada"))
}

func TestFillUpToCapacity(t *testing.T) {
	cache := setupCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", testData{Value: 26, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "enchilada", testData{Value: 28, DataSize: 26}, []int64{}, nil)

	assert.Equal(t, int64(23), cache.LookUp("burrito").(testData).Value)
	assert.Equal(t, int64(26), cache.LookUp("taco").(testData).Value)
	assert.Equal(t, int64(28), cache.LookUp("enchilada").(testData).Value)
}

func TestExpiresLeastRecentlyUsed(t *testing.T) {
	cache := setupCacheTest(t)
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

func TestOverwrite(t *testing.T) {
	cache := setupCacheTest(t)
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

func TestMultipleEviction(t *testing.T) {
	cache := setupCacheTest(t)
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

func TestWhenEntrySizeMoreThanCacheMaxSize(t *testing.T) {
	cache := setupCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)

	// Insert entry with size greater than maxSize of cache.
	insertAndAssert(t, cache, "taco", testData{Value: 26, DataSize: MaxSize + 1}, []int64{}, lru.ErrInvalidEntrySize)

	assert.Equal(t, int64(23), cache.LookUp("burrito").(testData).Value)
}

func TestEraseWhenKeyPresent(t *testing.T) {
	cache := setupCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)

	deletedEntry := cache.Erase("burrito")

	assert.Equal(t, int64(23), deletedEntry.(testData).Value)
	assert.Nil(t, cache.LookUp("burrito"))
}

func TestEraseCacheWithGivenPrefix(t *testing.T) {
	cache := setupCacheTest(t)
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

func TestEraseCacheWhereNoEntriesExistWithGivenPrefix(t *testing.T) {
	cache := setupCacheTest(t)
	insertAndAssert(t, cache, "a", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "a/b", testData{Value: 26, DataSize: 5}, []int64{}, nil)
	insertAndAssert(t, cache, "b", testData{Value: 21, DataSize: 2}, []int64{}, nil)

	cache.EraseEntriesWithGivenPrefix("c")

	assert.Equal(t, uint64(4), cache.LookUp("a").Size())
	assert.Equal(t, uint64(5), cache.LookUp("a/b").Size())
	assert.Equal(t, uint64(2), cache.LookUp("b").Size())
}

func TestEraseCacheWithGivenPrefixWithSomeEntriesEvictedDueToCacheSize(t *testing.T) {
	cache := setupCacheTest(t)
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

func TestEraseWhenKeyNotPresent(t *testing.T) {
	cache := setupCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)

	deletedEntry := cache.Erase("taco")
	assert.Nil(t, deletedEntry)

	assert.Equal(t, int64(23), cache.LookUp("burrito").(testData).Value)
}

func TestUpdateWhenKeyPresent(t *testing.T) {
	cache := setupCacheTest(t)
	key := "burrito"
	data := testData{Value: 23, DataSize: 4}
	insertAndAssert(t, cache, key, data, []int64{}, nil)
	newData := testData{Value: 2, DataSize: 4}

	err := cache.UpdateWithoutChangingOrder(key, newData)

	assert.Nil(t, err)
	assert.Equal(t, int64(2), cache.LookUp(key).(testData).Value)
}

func TestUpdateWhenKeyNotPresent(t *testing.T) {
	cache := setupCacheTest(t)
	key := "burrito"
	data := testData{Value: 23, DataSize: 4}

	err := cache.UpdateWithoutChangingOrder(key, data)

	assert.ErrorIs(t, err, lru.ErrEntryNotExist)
}

func TestUpdateWhenSizeIsDifferent(t *testing.T) {
	cache := setupCacheTest(t)
	key := "burrito"
	data := testData{Value: 23, DataSize: 4}
	insertAndAssert(t, cache, key, data, []int64{}, nil)
	newData := testData{Value: 2, DataSize: 3}

	err := cache.UpdateWithoutChangingOrder(key, newData)

	assert.ErrorIs(t, err, lru.ErrInvalidUpdateEntrySize)
}

func TestUpdateNotChangeOrder(t *testing.T) {
	cache := setupCacheTest(t)
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

func TestLookUpWithoutChangingOrder_WhenKeyPresent(t *testing.T) {
	cache := setupCacheTest(t)
	key := "burrito"
	data := testData{Value: 23, DataSize: 4}
	insertAndAssert(t, cache, key, data, []int64{}, nil)

	value := cache.LookUpWithoutChangingOrder(key)

	assert.Equal(t, int64(23), value.(testData).Value)
}

func TestLookUpWithoutChangingOrder_WhenKeyNotPresent(t *testing.T) {
	cache := setupCacheTest(t)
	key := "burrito"

	value := cache.LookUpWithoutChangingOrder(key)

	assert.Nil(t, value)
}

func TestLookUpWithoutChangingOrder_NotChangeOrder(t *testing.T) {
	cache := setupCacheTest(t)
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
func TestRaceCondition(t *testing.T) {
	cache := setupCacheTest(t)
	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		defer wg.Done()
		for i := range OperationCount {
			_, err := cache.Insert("key", testData{
				Value:    int64(i),
				DataSize: uint64(rand.Intn(MaxSize)),
			})
			assert.NoError(t, err)
		}
	}()

	go func() {
		defer wg.Done()
		for range OperationCount {
			cache.Erase("key")
		}
	}()

	go func() {
		defer wg.Done()
		for range OperationCount {
			cache.LookUp("key")
		}
	}()

	go func() {
		defer wg.Done()
		for range OperationCount {
			cache.LookUpWithoutChangingOrder("key")
		}
	}()

	go func() {
		defer wg.Done()
		for i := range OperationCount {
			_ = cache.UpdateWithoutChangingOrder("key", testData{
				Value:    int64(i),
				DataSize: uint64(rand.Intn(MaxSize)),
			})
		}
	}()

	wg.Wait()
}

func Test_EraseEntriesWithGivenPrefix_Concurrent(t *testing.T) {
	c := lru.NewCache(100000)

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

func Test_UpdateSize_AccountingSafety(t *testing.T) {
	c := lru.NewCache(100)

	// Insert entry of size 30
	_, err := c.Insert("file1", testData{Value: 1, DataSize: 30})
	assert.NoError(t, err)

	// Update size by 20 (total accounting size = 50)
	err = c.UpdateSize("file1", 20)
	assert.NoError(t, err)

	// Erase file1. Size accounting must subtract 50 (30 base + 20 delta), leaving 0.
	val := c.Erase("file1")
	assert.NotNil(t, val)

	// Inserting an entry of size 100 should now succeed completely without exceeding maxSize (100) or panicking invariants
	evicted, err := c.Insert("file2", testData{Value: 2, DataSize: 100})
	assert.NoError(t, err)
	assert.Empty(t, evicted)

	// Test UpdateSize triggering eviction when currentSize exceeds maxSize
	err = c.UpdateSize("file2", 10)
	assert.NoError(t, err)
	assert.Nil(t, c.LookUp("file2"))
}

func TestMapCache_UpdateSize_ImmediateTailEviction(t *testing.T) {
	cache := lru.NewCache(50)

	_, err := cache.Insert("key1", testData{Value: 10, DataSize: 30})
	require.NoError(t, err)
	_, err = cache.Insert("key2", testData{Value: 20, DataSize: 15})
	require.NoError(t, err)

	// Look up key1 so key1 becomes MRU and key2 becomes LRU (tail)
	_ = cache.LookUp("key1")

	// Update size of key1 by 20. Total size becomes 65 > 50.
	// Immediate tail eviction should evict key2 (least recently used tail).
	err = cache.UpdateSize("key1", 20)
	require.NoError(t, err)

	assert.Nil(t, cache.LookUp("key2"), "Tail entry key2 must be immediately evicted")
	assert.NotNil(t, cache.LookUp("key1"), "Key1 should remain in cache")
}

func TestRadixAndMapCache_SizeAccounting_OverwriteExtraSize(t *testing.T) {
	caches := []struct {
		name string
		c    lru.Cache
	}{
		{"mapCache", lru.NewCache(100)},
		{"radixCache", lru.NewRadixCache(100)},
	}

	for _, tc := range caches {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Insert entry of size 20
			_, err := tc.c.Insert("file1", testData{Value: 1, DataSize: 20})
			require.NoError(t, err)

			// Step 2: Expand size by 40 (total 60)
			err = tc.c.UpdateSize("file1", 40)
			require.NoError(t, err)

			// Step 3: Overwrite with size 10
			evicted, err := tc.c.Insert("file1", testData{Value: 2, DataSize: 10})
			require.NoError(t, err)
			assert.Empty(t, evicted)

			// Step 4: Insert another item of size 90 (10 + 90 = 100 <= 100 capacity)
			evicted, err = tc.c.Insert("file2", testData{Value: 3, DataSize: 90})
			require.NoError(t, err)
			assert.Empty(t, evicted, "file1 should NOT be evicted because overwritten size was 10, not 60")

			assert.NotNil(t, tc.c.LookUp("file1"))
			assert.NotNil(t, tc.c.LookUp("file2"))
		})
	}
}


