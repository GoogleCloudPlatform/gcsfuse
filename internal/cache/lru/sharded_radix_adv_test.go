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
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShardedRadixCache_Adv_EmptyKeys tests all cache operations on empty string key "".
func TestShardedRadixCache_Adv_EmptyKeys(t *testing.T) {
	cache := lru.NewShardedRadixCache(100)
	defer cache.Close()

	// 1. Lookups on empty cache
	assert.Nil(t, cache.LookUp(""))
	assert.Nil(t, cache.LookUpWithoutChangingOrder(""))
	assert.Equal(t, lru.ErrEntryNotExist, cache.UpdateWithoutChangingOrder("", testData{Value: 1, DataSize: 10}))
	assert.Equal(t, lru.ErrEntryNotExist, cache.UpdateSize("", 5))
	assert.Nil(t, cache.Erase(""))

	// 2. Insert empty key
	evicted, err := cache.Insert("", testData{Value: 100, DataSize: 20})
	require.NoError(t, err)
	assert.Empty(t, evicted)
	assert.Equal(t, int64(100), cache.LookUp("").(testData).Value)
	assert.Equal(t, int64(100), cache.LookUpWithoutChangingOrder("").(testData).Value)

	// 3. UpdateWithoutChangingOrder on empty key
	err = cache.UpdateWithoutChangingOrder("", testData{Value: 200, DataSize: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(200), cache.LookUp("").(testData).Value)

	// 4. UpdateSize on empty key
	err = cache.UpdateSize("", 10) // increases size from 20 to 30
	require.NoError(t, err)
	assert.Equal(t, int64(200), cache.LookUp("").(testData).Value)

	// 5. Erase empty key
	erased := cache.Erase("")
	assert.NotNil(t, erased)
	assert.Equal(t, int64(200), erased.(testData).Value)
	assert.Nil(t, cache.LookUp(""))

	// 6. Eviction involving empty key
	_, _ = cache.Insert("", testData{Value: 1, DataSize: 60})
	_, _ = cache.Insert("key2", testData{Value: 2, DataSize: 50}) // total 110 > 100, should evict "" (least recently used)
	assert.Nil(t, cache.LookUp(""))
	assert.NotNil(t, cache.LookUp("key2"))
}

// TestShardedRadixCache_Adv_EmptyPrefixErasure tests EraseEntriesWithGivenPrefix("") with items in cache.
// BUG DETECTED: In EraseEntriesWithGivenPrefix(""), s.currentSize and s.len are reset to 0,
// and s.sieveHead/s.sieveTail are set to nil BEFORE calling s.processDetachedSubtreesBatchLocked.
// When processDetachedSubtreesBatchLocked runs on oldRoot nodes, it subtracts item sizes from s.currentSize (causing uint64 underflow),
// decrements s.len (causing negative len), and calls sieveRemove which resurrects old pointers into s.sieveHead/s.sieveTail.
// Consequently, any subsequent Insert into an affected shard sees s.currentSize > s.maxSize and immediately evicts newly inserted items.
func TestShardedRadixCache_Adv_EmptyPrefixErasure(t *testing.T) {
	cache := lru.NewShardedRadixCache(262144) // 256KB so local shard eviction regime (maxSize >= 1024) is active
	defer cache.Close()

	// Insert many items to populate shards and test detached subtree processing
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("key-%d", i)
		_, err := cache.Insert(key, testData{Value: int64(i), DataSize: 100})
		require.NoError(t, err)
	}

	// Clear entire cache using empty prefix
	cache.EraseEntriesWithGivenPrefix("")

	// Verify all items are gone
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("key-%d", i)
		assert.Nil(t, cache.LookUp(key), "key %s should be erased", key)
	}

	// Insert new items to verify shard state (currentSize, len, sieve lists) is not corrupted or underflowed
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("new-key-%d", i)
		evicted, err := cache.Insert(key, testData{Value: int64(i), DataSize: 100})
		require.NoError(t, err)
		assert.Empty(t, evicted, "BUG: Entries evicted immediately after EraseEntriesWithGivenPrefix('') due to s.currentSize underflow")
	}
}

// TestShardedRadixCache_Adv_100PercentCapacity tests 100% capacity eviction regimes.
func TestShardedRadixCache_Adv_100PercentCapacity(t *testing.T) {
	// Cache of size 100
	cache := lru.NewShardedRadixCache(100)
	defer cache.Close()

	// 1. Single entry exactly equal to 100% of maxSize
	evicted, err := cache.Insert("exact", testData{Value: 1, DataSize: 100})
	require.NoError(t, err)
	assert.Empty(t, evicted)
	assert.Equal(t, int64(1), cache.LookUp("exact").(testData).Value)

	// 2. Insert another entry when cache is at 100% capacity
	evicted, err = cache.Insert("next", testData{Value: 2, DataSize: 50})
	require.NoError(t, err)
	assert.NotEmpty(t, evicted)
	assert.Nil(t, cache.LookUp("exact"))
	assert.Equal(t, int64(2), cache.LookUp("next").(testData).Value)

	// 3. UpdateSize pushing total cache size over 100% capacity -> triggers eviction
	err = cache.UpdateSize("next", 60) // increases size by 60, total 110 > 100
	require.NoError(t, err)
	// Because cache size 110 > 100 and only "next" is present, evictGlobal runs.
	// In evictGlobal, since "next" is in the protected shard, it is evicted on the second pass.
	assert.Nil(t, cache.LookUp("next"), "Entry should be evicted when UpdateSize pushes cache over 100% capacity")
}

// TestShardedRadixCache_Adv_HybridEvictionAccounting tests shard local vs global eviction accounting when shard maxSize >= 1024.
func TestShardedRadixCache_Adv_HybridEvictionAccounting(t *testing.T) {
	// With maxSize = 256 * 1024 = 262144, each shard has maxSize = 1024.
	// This activates the condition `s.maxSize >= 1024 && s.currentSize > s.maxSize`.
	cache := lru.NewShardedRadixCache(262144)
	defer cache.Close()

	// Find two keys that hash to the SAME shard
	var k1, k2 string
	var shardIdx int
	keysInShard := make(map[int][]string)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("collision-%d", i)
		shardKey := key
		if idxSlash := strings.LastIndex(key, "/"); idxSlash >= 0 {
			shardKey = key[:idxSlash+1]
		}
		hash := uint32(2166136261)
		for j := 0; j < len(shardKey); j++ {
			hash ^= uint32(shardKey[j])
			hash *= 16777619
		}
		idx := int(hash & 0xFF)
		keysInShard[idx] = append(keysInShard[idx], key)
		if len(keysInShard[idx]) >= 2 {
			k1 = keysInShard[idx][0]
			k2 = keysInShard[idx][1]
			shardIdx = idx
			break
		}
	}
	require.NotEmpty(t, k1)
	require.NotEmpty(t, k2)
	t.Logf("Found colliding keys for shard %d: %s and %s", shardIdx, k1, k2)

	// Insert k1 with size 800 (under shard maxSize 1024)
	evicted, err := cache.Insert(k1, testData{Value: 1, DataSize: 800})
	require.NoError(t, err)
	assert.Empty(t, evicted)

	// Insert k2 into the same shard with size 500 (total in shard = 1300 > 1024).
	// Because shard maxSize >= 1024, local shard eviction SHOULD trigger and evict k1 (least recently used in shard).
	evicted, err = cache.Insert(k2, testData{Value: 2, DataSize: 500})
	require.NoError(t, err)
	assert.NotEmpty(t, evicted, "Local shard eviction should evict k1")
	assert.Nil(t, cache.LookUp(k1), "k1 should be evicted by local shard eviction")
	assert.NotNil(t, cache.LookUp(k2), "k2 should be present")
}

// TestShardedRadixCache_Adv_ZeroMaxSize tests boundary condition where maxSize == 0.
func TestShardedRadixCache_Adv_ZeroMaxSize(t *testing.T) {
	assert.Panics(t, func() {
		_ = lru.NewShardedRadixCache(0)
	}, "NewShardedRadixCache(0) should panic")
}

// TestShardedRadixCache_Adv_UpdateWithoutChangingOrder_Errors tests boundary conditions for UpdateWithoutChangingOrder.
func TestShardedRadixCache_Adv_UpdateWithoutChangingOrder_Errors(t *testing.T) {
	cache := lru.NewShardedRadixCache(100)
	defer cache.Close()

	// Nil entry
	err := cache.UpdateWithoutChangingOrder("any", nil)
	assert.Equal(t, lru.ErrInvalidEntry, err)

	// Insert valid entry
	_, err = cache.Insert("key", testData{Value: 1, DataSize: 10})
	require.NoError(t, err)

	// Size mismatch
	err = cache.UpdateWithoutChangingOrder("key", testData{Value: 2, DataSize: 20})
	assert.Equal(t, lru.ErrInvalidUpdateEntrySize, err)
}

// TestShardedRadixCache_Adv_UpdateSize_ExceedsMaxSize tests UpdateSize triggering eviction when total size exceeds maxSize.
func TestShardedRadixCache_Adv_UpdateSize_ExceedsMaxSize(t *testing.T) {
	cache := lru.NewShardedRadixCache(100)
	defer cache.Close()

	_, err := cache.Insert("k1", testData{Value: 1, DataSize: 40})
	require.NoError(t, err)
	_, err = cache.Insert("k2", testData{Value: 2, DataSize: 40})
	require.NoError(t, err)

	// Update k2 size by 30 (total 110 > 100), should evict k1 (least recently used)
	err = cache.UpdateSize("k2", 30)
	require.NoError(t, err)
	assert.Nil(t, cache.LookUp("k1"))
	assert.NotNil(t, cache.LookUp("k2"))
}

// TestShardedRadixCache_Adv_ErasePrefix_PartialAndDiverging tests EraseEntriesWithGivenPrefix with partial and diverging prefixes.
func TestShardedRadixCache_Adv_ErasePrefix_PartialAndDiverging(t *testing.T) {
	cache := lru.NewShardedRadixCache(1000)
	defer cache.Close()

	_, _ = cache.Insert("app/vendor/lib1", testData{Value: 1, DataSize: 10})
	_, _ = cache.Insert("app/vendor/lib2", testData{Value: 2, DataSize: 10})
	_, _ = cache.Insert("app/version", testData{Value: 3, DataSize: 10})

	// Diverging prefix: "app/venX" matches nothing
	cache.EraseEntriesWithGivenPrefix("app/venX")
	assert.NotNil(t, cache.LookUp("app/vendor/lib1"))
	assert.NotNil(t, cache.LookUp("app/vendor/lib2"))
	assert.NotNil(t, cache.LookUp("app/version"))

	// Partial match: "app/vendor" should erase lib1 and lib2, leave version
	cache.EraseEntriesWithGivenPrefix("app/vendor")
	assert.Nil(t, cache.LookUp("app/vendor/lib1"))
	assert.Nil(t, cache.LookUp("app/vendor/lib2"))
	assert.NotNil(t, cache.LookUp("app/version"))
}

// TestShardedRadixCache_Adv_SmallShardNoLocalEviction tests eviction regime when shard maxSize < 1024.
func TestShardedRadixCache_Adv_SmallShardNoLocalEviction(t *testing.T) {
	// Total maxSize = 25600 -> shard maxSize = 100 (< 1024)
	cache := lru.NewShardedRadixCache(25600)
	defer cache.Close()

	// Find two keys that hash to the SAME shard
	var k1, k2 string
	keysInShard := make(map[int][]string)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("small-coll-%d", i)
		shardKey := key
		if idxSlash := strings.LastIndex(key, "/"); idxSlash >= 0 {
			shardKey = key[:idxSlash+1]
		}
		hash := uint32(2166136261)
		for j := 0; j < len(shardKey); j++ {
			hash ^= uint32(shardKey[j])
			hash *= 16777619
		}
		idx := int(hash & 0xFF)
		keysInShard[idx] = append(keysInShard[idx], key)
		if len(keysInShard[idx]) >= 2 {
			k1 = keysInShard[idx][0]
			k2 = keysInShard[idx][1]
			break
		}
	}
	require.NotEmpty(t, k1)
	require.NotEmpty(t, k2)

	// Insert k1 of size 60, k2 of size 60 (total in shard = 120 > shard maxSize 100).
	// Because shard maxSize < 1024, local shard eviction should NOT trigger!
	evicted, err := cache.Insert(k1, testData{Value: 1, DataSize: 60})
	require.NoError(t, err)
	assert.Empty(t, evicted)

	evicted, err = cache.Insert(k2, testData{Value: 2, DataSize: 60})
	require.NoError(t, err)
	assert.Empty(t, evicted, "When shard maxSize < 1024, local shard eviction must not trigger")
	assert.NotNil(t, cache.LookUp(k1))
	assert.NotNil(t, cache.LookUp(k2))
}
