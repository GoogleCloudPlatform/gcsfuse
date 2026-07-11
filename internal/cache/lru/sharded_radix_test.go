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
	"github.com/stretchr/testify/require"
)

func setupShardedRadixCacheTest(t *testing.T) lru.Cache {
	locker.EnableInvariantsCheck()
	return lru.NewShardedRadixCache(testMaxSize)
}

func TestShardedRadixCache_LookUpInEmptyCache(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	assert.Nil(t, cache.LookUp(""))
	assert.Nil(t, cache.LookUp("taco"))
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_InsertNilValue(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	insertAndAssert(t, cache, "taco", nil, []int64{}, lru.ErrInvalidEntry)
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_InsertEmptyKey(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	insertAndAssert(t, cache, "", testData{Value: 42, DataSize: 10}, []int64{}, nil)
	assert.Equal(t, int64(42), cache.LookUp("").(testData).Value)
	assert.Nil(t, cache.LookUp("taco"))
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_LookUpUnknownKey(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", testData{Value: 23, DataSize: 8}, []int64{}, nil)
	assert.Nil(t, cache.LookUp(""))
	assert.Nil(t, cache.LookUp("enchilada"))
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_FillUpToCapacity(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", testData{Value: 26, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "enchilada", testData{Value: 28, DataSize: 26}, []int64{}, nil)
	assert.Equal(t, int64(23), cache.LookUp("burrito").(testData).Value)
	assert.Equal(t, int64(26), cache.LookUp("taco").(testData).Value)
	assert.Equal(t, int64(28), cache.LookUp("enchilada").(testData).Value)
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_ExpiresLeastRecentlyUsed(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", testData{Value: 26, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "enchilada", testData{Value: 28, DataSize: 26}, []int64{}, nil)

	assert.Equal(t, int64(23), cache.LookUp("burrito").(testData).Value)

	insertAndAssert(t, cache, "queso", testData{Value: 34, DataSize: 5}, []int64{26}, nil)
	assert.Nil(t, cache.LookUp("taco"))
	assert.Equal(t, int64(23), cache.LookUp("burrito").(testData).Value)
	assert.Equal(t, int64(28), cache.LookUp("enchilada").(testData).Value)
	assert.Equal(t, int64(34), cache.LookUp("queso").(testData).Value)
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_Overwrite(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", testData{Value: 26, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "enchilada", testData{Value: 28, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "burrito", testData{Value: 33, DataSize: 6}, []int64{}, nil)
	insertAndAssert(t, cache, "burrito", testData{Value: 33, DataSize: 12}, []int64{26}, nil)
	assert.Nil(t, cache.LookUp("taco"))
	assert.Equal(t, int64(33), cache.LookUp("burrito").(testData).Value)
	assert.Equal(t, int64(28), cache.LookUp("enchilada").(testData).Value)
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_MultipleEviction(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", testData{Value: 26, DataSize: 20}, []int64{}, nil)
	insertAndAssert(t, cache, "enchilada", testData{Value: 28, DataSize: 20}, []int64{}, nil)

	ret, err := cache.Insert("large_data", testData{Value: 33, DataSize: 45})
	require.NoError(t, err)
	var evictedVals []int64
	for _, v := range ret {
		evictedVals = append(evictedVals, v.(testData).Value)
	}
	assert.ElementsMatch(t, []int64{23, 26, 28}, evictedVals)

	assert.Nil(t, cache.LookUp("taco"))
	assert.Nil(t, cache.LookUp("burrito"))
	assert.Nil(t, cache.LookUp("enchilada"))
	assert.Equal(t, int64(33), cache.LookUp("large_data").(testData).Value)
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_WhenEntrySizeMoreThanCacheMaxSize(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "taco", testData{Value: 26, DataSize: testMaxSize + 1}, []int64{}, lru.ErrInvalidEntrySize)
	assert.Equal(t, int64(23), cache.LookUp("burrito").(testData).Value)
	assert.Nil(t, cache.LookUp("taco"))
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_EraseWhenKeyPresent(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	deletedEntry := cache.Erase("burrito")
	assert.Equal(t, int64(23), deletedEntry.(testData).Value)
	assert.Nil(t, cache.LookUp("burrito"))
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_EraseCacheWithGivenPrefix(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
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
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_EraseCacheWithEmptyPrefix(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	insertAndAssert(t, cache, "a", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	insertAndAssert(t, cache, "a/b", testData{Value: 26, DataSize: 5}, []int64{}, nil)
	insertAndAssert(t, cache, "b", testData{Value: 21, DataSize: 2}, []int64{}, nil)

	cache.EraseEntriesWithGivenPrefix("")

	assert.Nil(t, cache.LookUp("a"))
	assert.Nil(t, cache.LookUp("a/b"))
	assert.Nil(t, cache.LookUp("b"))
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_EraseWhenKeyNotPresent(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	insertAndAssert(t, cache, "burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	deletedEntry := cache.Erase("taco")
	assert.Nil(t, deletedEntry)
	assert.Equal(t, int64(23), cache.LookUp("burrito").(testData).Value)
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_UpdateSize(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	err := cache.UpdateSize("key", 10)
	assert.Equal(t, lru.ErrEntryNotExist, err)

	insertAndAssert(t, cache, "key1", testData{Value: 1, DataSize: 10}, []int64{}, nil)
	insertAndAssert(t, cache, "key2", testData{Value: 2, DataSize: 30}, []int64{}, nil)
	err = cache.UpdateSize("key1", 20)
	require.NoError(t, err)
	assert.Nil(t, cache.LookUp("key2"))
	assert.NotNil(t, cache.LookUp("key1"))
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_UpdateWithoutChangingOrder(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	err := cache.UpdateWithoutChangingOrder("key", testData{Value: 1, DataSize: 10})
	assert.Equal(t, lru.ErrEntryNotExist, err)

	insertAndAssert(t, cache, "key", testData{Value: 1, DataSize: 10}, []int64{}, nil)
	err = cache.UpdateWithoutChangingOrder("key", testData{Value: 2, DataSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), cache.LookUp("key").(testData).Value)
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_LookUpWithoutChangingOrder(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	insertAndAssert(t, cache, "key", testData{Value: 1, DataSize: 10}, []int64{}, nil)
	val := cache.LookUpWithoutChangingOrder("key")
	assert.Equal(t, int64(1), val.(testData).Value)
	assert.Nil(t, cache.LookUpWithoutChangingOrder("nonexistent"))
	require.NoError(t, cache.Close())
}

type dynamicTestData struct {
	mu       sync.Mutex
	Value    int64
	DataSize uint64
}

func (d *dynamicTestData) Size() uint64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.DataSize
}

func TestShardedRadixCache_RaceCondition(t *testing.T) {
	cache := setupShardedRadixCacheTest(t)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range testOperationCount {
			val := cache.LookUpWithoutChangingOrder("key")
			if val != nil {
				if d, ok := val.(*dynamicTestData); ok {
					delta := uint64(rand.Intn(10))
					d.mu.Lock()
					d.DataSize += delta
					d.mu.Unlock()
					_ = cache.UpdateSize("key", delta)
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range testOperationCount {
			_ = cache.LookUp("key")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range testOperationCount {
			_, _ = cache.Insert("key", &dynamicTestData{
				Value:    int64(i),
				DataSize: uint64(rand.Intn(testMaxSize)),
			})
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range testOperationCount {
			_ = cache.Erase("key")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range testOperationCount {
			_ = cache.UpdateWithoutChangingOrder("key", &dynamicTestData{
				Value:    int64(i),
				DataSize: uint64(rand.Intn(testMaxSize)),
			})
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range testOperationCount {
			cache.EraseEntriesWithGivenPrefix("k")
		}
	}()

	wg.Wait()
	require.NoError(t, cache.Close())
}

func TestShardedRadixCache_EraseEntriesWithGivenPrefix_Concurrent(t *testing.T) {
	c := setupShardedRadixCacheTest(t)
	var wg sync.WaitGroup

	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("dir1/file%d", i)
		_, _ = c.Insert(key, testData{Value: int64(i), DataSize: 1})
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.EraseEntriesWithGivenPrefix("dir1/")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			key := fmt.Sprintf("dir2/file%d", i)
			_, _ = c.Insert(key, testData{Value: int64(i), DataSize: 1})
		}
	}()

	wg.Wait()
	require.NoError(t, c.Close())
}

func TestShardedRadixCache_ShardPadding(t *testing.T) {
	assert.Equal(t, uintptr(128), lru.GetShardSize(), "cacheShard struct must be padded to exactly 128 bytes to eliminate false sharing")
}

func TestStripBucketPrefix_EdgeCases(t *testing.T) {
	testCases := []struct {
		name       string
		key        string
		bucketName string
		expected   string
	}{
		{
			name:       "standard file key",
			key:        "my-bucket/dir/file.txt",
			bucketName: "my-bucket",
			expected:   "dir/file.txt",
		},
		{
			name:       "directory with trailing slash preserved",
			key:        "my-bucket/datasets/imagenet/train/",
			bucketName: "my-bucket",
			expected:   "datasets/imagenet/train/",
		},
		{
			name:       "bucket root with trailing slash",
			key:        "my-bucket/",
			bucketName: "my-bucket",
			expected:   "",
		},
		{
			name:       "bucket name without trailing slash",
			key:        "my-bucket",
			bucketName: "my-bucket",
			expected:   "my-bucket",
		},
		{
			name:       "similar prefix without slash boundary",
			key:        "my-bucket-other/file.txt",
			bucketName: "my-bucket",
			expected:   "my-bucket-other/file.txt",
		},
		{
			name:       "empty bucket name",
			key:        "my-bucket/dir/file.txt",
			bucketName: "",
			expected:   "my-bucket/dir/file.txt",
		},
		{
			name:       "empty key",
			key:        "",
			bucketName: "my-bucket",
			expected:   "",
		},
		{
			name:       "unrelated bucket key",
			key:        "other-bucket/foo/bar",
			bucketName: "my-bucket",
			expected:   "other-bucket/foo/bar",
		},
		{
			name:       "dotdot and double slash path without cleaning",
			key:        "my-bucket/dir/../file//txt/",
			bucketName: "my-bucket",
			expected:   "dir/../file//txt/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := lru.StripBucketPrefix(tc.key, tc.bucketName)
			assert.Equal(t, tc.expected, res)
		})
	}
}

func TestStripBucketPrefix_ZeroAllocation(t *testing.T) {
	key := "my-bucket/datasets/imagenet/train/0001.jpg"
	bucket := "my-bucket"

	allocs := testing.AllocsPerRun(1000, func() {
		_ = lru.StripBucketPrefix(key, bucket)
	})

	assert.Equal(t, float64(0), allocs, "StripBucketPrefix must not allocate any heap memory")
}

func TestShardedRadixCache_UpdateSize_AccountingSafety(t *testing.T) {
	c := lru.NewShardedRadixCache(256)
	defer c.(interface{ Close() error }).Close()

	// Insert entry of size 30
	_, err := c.Insert("dir/file1", testData{Value: 1, DataSize: 30})
	assert.NoError(t, err)

	// Update size by 20 (total accounting size = 50)
	err = c.UpdateSize("dir/file1", 20)
	assert.NoError(t, err)

	// Erase dir/file1. Size accounting must subtract 50 (30 base + 20 delta).
	val := c.Erase("dir/file1")
	assert.NotNil(t, val)

	// Overwrite test: insert entry of size 40, update size by 20 (total 60), then Insert new value of size 40
	_, err = c.Insert("dir/file2", testData{Value: 2, DataSize: 40})
	assert.NoError(t, err)

	err = c.UpdateSize("dir/file2", 20)
	assert.NoError(t, err)

	// Overwrite dir/file2 with size 40. The extraSize (20) must be properly swapped and accounted for.
	_, err = c.Insert("dir/file2", testData{Value: 22, DataSize: 40})
	assert.NoError(t, err)

	val2 := c.Erase("dir/file2")
	assert.NotNil(t, val2)
}

