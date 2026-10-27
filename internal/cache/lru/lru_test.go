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

func setupCacheTest(t *testing.T) *lru.Cache {
	t.Helper()
	testOpts := locker.Options{
		EnableInvariantsCheck: true,
	}
	return lru.NewCacheWithOptions(MaxSize, testOpts)
}

// insertAndAssert inserts the given key,value in the cache and assert based on
// the expected eviction and error.
func insertAndAssert(t *testing.T, cache *lru.Cache, key string, val lru.ValueType, evictedValues []int64, expectedError error) {
	t.Helper()
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

type testSparseData struct {
	Value    int64
	DataSize uint64
}

func (td *testSparseData) Size() uint64 {
	return td.DataSize
}

func TestUpdateSizeTOCTOU_EraseBeforeUpdateSizeDoesNotUnderflow(t *testing.T) {
	cache := setupCacheTest(t)
	data := &testSparseData{Value: 1, DataSize: 0}
	insertAndAssert(t, cache, "key", data, []int64{}, nil)
	data.DataSize = 1000

	deletedEntry := cache.Erase("key")
	err := cache.UpdateSize("key", 1000)

	require.NotNil(t, deletedEntry)
	assert.ErrorIs(t, err, lru.ErrEntryNotExist)
}

func TestUpdateSizeTOCTOU_ConcurrentUpdateAndErase(t *testing.T) {
	cache := setupCacheTest(t)
	data := &testSparseData{Value: 1, DataSize: 0}
	insertAndAssert(t, cache, "key", data, []int64{}, nil)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for range 10 {
			data.DataSize += 2
			_ = cache.UpdateSize("key", 2)
		}
	}()
	go func() {
		defer wg.Done()
		_ = cache.Erase("key")
	}()

	wg.Wait()
	// Assert: Invariant checks (sum(size) == currentSize and currentSize <= maxSize)
	// run automatically on lock release via setupCacheTest.
}

func TestUpdateWithoutChangingOrder_NilValue(t *testing.T) {
	cache := setupCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)

	err := cache.UpdateWithoutChangingOrder("burrito", nil)

	assert.ErrorIs(t, err, lru.ErrInvalidEntry)
	assert.Equal(t, int64(23), cache.LookUp("burrito").(testData).Value)
}

func TestZeroSizeEntriesExceedMaxSizeCount(t *testing.T) {
	cache := lru.NewCacheWithOptions(5, locker.Options{EnableInvariantsCheck: true})
	for i := range 20 {
		insertAndAssert(t, cache, fmt.Sprintf("zero-%d", i), testData{Value: int64(i), DataSize: 0}, []int64{}, nil)
	}
	for i := range 20 {
		v := cache.LookUpWithoutChangingOrder(fmt.Sprintf("zero-%d", i))
		require.NotNil(t, v)
		assert.Equal(t, int64(i), v.(testData).Value)
	}
}

func TestZeroMaxSizeCache(t *testing.T) {
	cache := lru.NewCache(0)
	insertAndAssert(t, cache, "non-zero", testData{Value: 1, DataSize: 1}, []int64{}, lru.ErrInvalidEntrySize)
	insertAndAssert(t, cache, "zero", testData{Value: 2, DataSize: 0}, []int64{}, nil)
	assert.Equal(t, int64(2), cache.LookUp("zero").(testData).Value)
}

func TestLargeMaxSizeCache(t *testing.T) {
	cache := lru.NewCacheWithOptions(uint64(1)<<30, locker.Options{EnableInvariantsCheck: true})
	insertAndAssert(t, cache, "item", testData{Value: 42, DataSize: 1024}, []int64{}, nil)
	assert.Equal(t, int64(42), cache.LookUp("item").(testData).Value)
}

func TestUpdateSize_DeferredOverflowAndEvictionOnInsert(t *testing.T) {
	cache := setupCacheTest(t)
	insertAndAssert(t, cache, "k1", testData{Value: 1, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "k2", testData{Value: 2, DataSize: 20}, []int64{}, nil)

	// Grow k1 twice so total size (20 + 20 + 15 + 10 = 65) exceeds MaxSize (50),
	// and also grow k2 while deferred overflow is active (65 + 5 = 70).
	require.NoError(t, cache.UpdateSize("k1", 15))
	require.NoError(t, cache.UpdateSize("k1", 10))
	require.NoError(t, cache.UpdateSize("k2", 5))

	// Both entries remain present without changing LRU order (k1 is still LRU, k2 is MRU).
	assert.Equal(t, int64(1), cache.LookUpWithoutChangingOrder("k1").(testData).Value)
	assert.Equal(t, int64(2), cache.LookUpWithoutChangingOrder("k2").(testData).Value)

	// Inserting k3 (size 10) brings total to 80 > 50: evicts k1 (size 45), leaving k2 (25) + k3 (10) = 35 <= 50,
	// and reconciles k2's deferred size in the underlying cache.
	insertAndAssert(t, cache, "k3", testData{Value: 3, DataSize: 10}, []int64{1}, nil)
	assert.Nil(t, cache.LookUp("k1"))
	assert.Equal(t, int64(2), cache.LookUp("k2").(testData).Value)
	assert.Equal(t, int64(3), cache.LookUp("k3").(testData).Value)
}

func TestUpdateSize_DeferredOverflowOverwriteAndReconcile(t *testing.T) {
	cache := setupCacheTest(t)
	insertAndAssert(t, cache, "k1", testData{Value: 1, DataSize: 30}, []int64{}, nil)
	insertAndAssert(t, cache, "k2", testData{Value: 2, DataSize: 15}, []int64{}, nil)

	// Grow k1 by 20 -> total 65 > 50.
	require.NoError(t, cache.UpdateSize("k1", 20))
	// UpdateWithoutChangingOrder on k1 with its updated size (50) while deferred overflow is active.
	require.NoError(t, cache.UpdateWithoutChangingOrder("k1", testData{Value: 11, DataSize: 50}))

	// Overwrite k1 with smaller size 10 -> total becomes 10 + 15 = 25 <= 50, reconciling deferred state without evictions.
	insertAndAssert(t, cache, "k1", testData{Value: 12, DataSize: 10}, []int64{}, nil)
	assert.Equal(t, int64(12), cache.LookUp("k1").(testData).Value)
	assert.Equal(t, int64(2), cache.LookUp("k2").(testData).Value)
}

func TestUpdateSize_DeferredOverflowEraseAndPrefixErase(t *testing.T) {
	cache := setupCacheTest(t)
	insertAndAssert(t, cache, "p/1", testData{Value: 1, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "p/2", testData{Value: 2, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "other", testData{Value: 3, DataSize: 5}, []int64{}, nil)

	// Grow p/1 by 20 -> total 65 > 50.
	require.NoError(t, cache.UpdateSize("p/1", 20))

	// EraseEntriesWithGivenPrefix while deferred overflow is active removes p/1 and p/2,
	// bringing currentSize to 5 <= 50 and reconciling deferred state.
	cache.EraseEntriesWithGivenPrefix("p/")
	assert.Nil(t, cache.LookUp("p/1"))
	assert.Nil(t, cache.LookUp("p/2"))
	assert.Equal(t, int64(3), cache.LookUp("other").(testData).Value)
}
