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
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRadixCache_New_ZeroMaxSize(t *testing.T) {
	assert.PanicsWithValue(t, "Invalid maxSize", func() {
		lru.NewRadixCache(0)
	})
}

func TestRadixCache_Overwrite_SmallerSize(t *testing.T) {
	cache := setupRadixCacheTest(t)

	// Insert key1 with size 30 and key2 with size 15 (total size = 45 <= testMaxSize 50)
	insertAndAssert(t, cache, "key1", testData{Value: 1, DataSize: 30}, []int64{}, nil)
	insertAndAssert(t, cache, "key2", testData{Value: 2, DataSize: 15}, []int64{}, nil)

	// Overwrite key1 with a smaller size (10).
	// Total size should become 10 + 15 = 25.
	insertAndAssert(t, cache, "key1", testData{Value: 100, DataSize: 10}, []int64{}, nil)

	assert.Equal(t, int64(100), cache.LookUp("key1").(testData).Value)
	assert.Equal(t, int64(2), cache.LookUp("key2").(testData).Value)

	// We can insert a new key3 of size 25 without evicting anything (25 + 25 = 50 <= 50).
	insertAndAssert(t, cache, "key3", testData{Value: 3, DataSize: 25}, []int64{}, nil)
	assert.NotNil(t, cache.LookUp("key1"))
	assert.NotNil(t, cache.LookUp("key2"))
	assert.NotNil(t, cache.LookUp("key3"))
}

func TestRadixCache_PrefixSplittingAndCompression(t *testing.T) {
	cache := setupRadixCacheTest(t)

	// Insert keys that force prefix splitting in the radix tree:
	// "foo/bar/1", "foo/bar/2", "foo/baz", "foo/b", "foo"
	insertAndAssert(t, cache, "foo/bar/1", testData{Value: 10, DataSize: 5}, []int64{}, nil)
	insertAndAssert(t, cache, "foo/bar/2", testData{Value: 20, DataSize: 5}, []int64{}, nil)
	insertAndAssert(t, cache, "foo/baz", testData{Value: 30, DataSize: 5}, []int64{}, nil)
	insertAndAssert(t, cache, "foo/b", testData{Value: 40, DataSize: 5}, []int64{}, nil)
	insertAndAssert(t, cache, "foo", testData{Value: 50, DataSize: 5}, []int64{}, nil)

	assert.Equal(t, int64(10), cache.LookUp("foo/bar/1").(testData).Value)
	assert.Equal(t, int64(20), cache.LookUp("foo/bar/2").(testData).Value)
	assert.Equal(t, int64(30), cache.LookUp("foo/baz").(testData).Value)
	assert.Equal(t, int64(40), cache.LookUp("foo/b").(testData).Value)
	assert.Equal(t, int64(50), cache.LookUp("foo").(testData).Value)

	// Erase leaf node "foo/bar/1", leaving "foo/bar/2" which should compress routing node "foo/bar/"
	cache.Erase("foo/bar/1")
	assert.Nil(t, cache.LookUp("foo/bar/1"))
	assert.Equal(t, int64(20), cache.LookUp("foo/bar/2").(testData).Value)

	// Erase value-bearing ancestor node "foo/b"
	cache.Erase("foo/b")
	assert.Nil(t, cache.LookUp("foo/b"))
	assert.Equal(t, int64(30), cache.LookUp("foo/baz").(testData).Value)
	assert.Equal(t, int64(50), cache.LookUp("foo").(testData).Value)

	// Erase remaining keys to trigger upward path compression to root
	cache.Erase("foo/bar/2")
	cache.Erase("foo/baz")
	cache.Erase("foo")

	assert.Nil(t, cache.LookUp("foo/bar/2"))
	assert.Nil(t, cache.LookUp("foo/baz"))
	assert.Nil(t, cache.LookUp("foo"))
}

func TestRadixCache_UpdateSize_CascadingEviction(t *testing.T) {
	cache := setupRadixCacheTest(t)

	// Insert 4 items of size 10 (total 40 <= 50)
	data1 := &testData{Value: 1, DataSize: 10}
	data2 := &testData{Value: 2, DataSize: 10}
	data3 := &testData{Value: 3, DataSize: 10}
	data4 := &testData{Value: 4, DataSize: 10}

	_, err := cache.Insert("key1", data1)
	require.NoError(t, err)
	_, err = cache.Insert("key2", data2)
	require.NoError(t, err)
	_, err = cache.Insert("key3", data3)
	require.NoError(t, err)
	_, err = cache.Insert("key4", data4)
	require.NoError(t, err)

	// Now increase key4's size by 30 (new DataSize = 40, total size would be 70).
	// This should cause multiple least recently used items (key1, key2, key3) to be evicted
	// until currentSize <= 50.
	data4.DataSize = 40
	err = cache.UpdateSize("key4", 30)
	assert.NoError(t, err)

	// key4 should remain, and enough older entries evicted to bring total size <= 50.
	assert.NotNil(t, cache.LookUp("key4"))
	// With key4=40, only 1 other item of size 10 can remain (50 - 40 = 10).
	// Since key1, key2, key3 were LRU ordered, key1 and key2 should be evicted.
	assert.Nil(t, cache.LookUp("key1"))
	assert.Nil(t, cache.LookUp("key2"))
	assert.NotNil(t, cache.LookUp("key3"))
}

func TestRadixCache_EraseEntriesWithGivenPrefix_PartialMatch(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "dir/sub/file1", testData{Value: 1, DataSize: 10}, []int64{}, nil)
	insertAndAssert(t, cache, "dir/sub/file2", testData{Value: 2, DataSize: 10}, []int64{}, nil)
	insertAndAssert(t, cache, "dir/other/file3", testData{Value: 3, DataSize: 10}, []int64{}, nil)

	// Erase prefix "dir/s" which splits the node prefix "dir/sub/file1"
	cache.EraseEntriesWithGivenPrefix("dir/s")

	assert.Nil(t, cache.LookUp("dir/sub/file1"))
	assert.Nil(t, cache.LookUp("dir/sub/file2"))
	assert.NotNil(t, cache.LookUp("dir/other/file3"))
}

func TestRadixCache_EraseEntriesWithGivenPrefix_DivergingPrefix(t *testing.T) {
	cache := setupRadixCacheTest(t)
	insertAndAssert(t, cache, "dir/sub/file1", testData{Value: 1, DataSize: 10}, []int64{}, nil)

	// Erase prefix "dir/something" which shares "dir/s" but diverges
	cache.EraseEntriesWithGivenPrefix("dir/something")

	assert.NotNil(t, cache.LookUp("dir/sub/file1"))
}

func TestRadixCache_ConcurrentReadWriteErase(t *testing.T) {
	// locker.EnableInvariantsCheck()
	cache := lru.NewRadixCache(1000)
	const numOps = 200

	var wg sync.WaitGroup
	wg.Add(4)

	// Goroutine 1: Insert keys
	go func() {
		defer wg.Done()
		for i := 0; i < numOps; i++ {
			key := fmt.Sprintf("dir/%d/file", i%10)
			_, _ = cache.Insert(key, testData{Value: int64(i), DataSize: 10})
		}
	}()

	// Goroutine 2: Lookups
	go func() {
		defer wg.Done()
		for i := 0; i < numOps; i++ {
			key := fmt.Sprintf("dir/%d/file", i%10)
			_ = cache.LookUp(key)
			_ = cache.LookUpWithoutChangingOrder(key)
		}
	}()

	// Goroutine 3: Erase individual keys
	go func() {
		defer wg.Done()
		for i := 0; i < numOps; i++ {
			key := fmt.Sprintf("dir/%d/file", i%10)
			_ = cache.Erase(key)
		}
	}()

	// Goroutine 4: Erase by prefix
	go func() {
		defer wg.Done()
		for i := 0; i < numOps/10; i++ {
			prefix := fmt.Sprintf("dir/%d/", i%5)
			cache.EraseEntriesWithGivenPrefix(prefix)
		}
	}()

	wg.Wait()
}
