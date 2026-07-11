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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type advValue struct {
	bytes uint64
}

func (v advValue) Size() uint64 {
	return v.bytes
}

// 1. Boundary Condition: 100% Capacity Fill & Eviction Behavior
func TestAdv_CapacityFill100Percent(t *testing.T) {
	const maxSize uint64 = 100
	cache := lru.NewRadixCache(maxSize)

	// Fill to exactly 100% (10 items of size 10)
	for i := range 10 {
		key := fmt.Sprintf("item_%d", i)
		evicted, err := cache.Insert(key, advValue{bytes: 10})
		require.NoError(t, err)
		assert.Empty(t, evicted, "No eviction expected before capacity exceeded")
	}

	// Verify all 10 items are present
	for i := range 10 {
		key := fmt.Sprintf("item_%d", i)
		assert.NotNil(t, cache.LookUp(key), "Key %s should be present at 100%% capacity", key)
	}

	// Adding 1 more item of size 1 should trigger eviction of the least recently used item (item_0)
	evicted, err := cache.Insert("item_10", advValue{bytes: 1})
	require.NoError(t, err)
	require.Len(t, evicted, 1)
	assert.Equal(t, uint64(10), evicted[0].Size())
	assert.Nil(t, cache.LookUp("item_0"), "item_0 should have been evicted")
	assert.NotNil(t, cache.LookUp("item_10"), "item_10 should be present")

	// Insert single entry equal to 100% of maxSize (100 bytes)
	cache100 := lru.NewRadixCache(100)
	evicted, err = cache100.Insert("full_item", advValue{bytes: 100})
	require.NoError(t, err)
	assert.Empty(t, evicted)
	assert.NotNil(t, cache100.LookUp("full_item"))

	// Inserting 1 byte into cache filled to 100% with a single item must evict the large item
	evicted, err = cache100.Insert("tiny_item", advValue{bytes: 1})
	require.NoError(t, err)
	require.Len(t, evicted, 1)
	assert.Equal(t, uint64(100), evicted[0].Size())
	assert.Nil(t, cache100.LookUp("full_item"))
	assert.NotNil(t, cache100.LookUp("tiny_item"))
}

// 2. Boundary Condition: Node Promotion (Hub -> Payload) and Demotion (Payload -> Hub)
func TestAdv_NodePromotionAndDemotion(t *testing.T) {
	cache := lru.NewRadixCache(1000)

	// Setup tree with child nodes sharing a common routing prefix:
	// "dir/sub/file1" and "dir/sub/file2" create an internal routing node (hub) at "dir/sub/"
	_, err := cache.Insert("dir/sub/file1", advValue{bytes: 10})
	require.NoError(t, err)
	_, err = cache.Insert("dir/sub/file2", advValue{bytes: 20})
	require.NoError(t, err)

	// At this point "dir/sub/" is a hub node without value
	assert.Nil(t, cache.LookUpWithoutChangingOrder("dir/sub/"))

	// PROMOTION: Insert value for "dir/sub/" directly -> hub becomes payload node
	evicted, err := cache.Insert("dir/sub/", advValue{bytes: 30})
	require.NoError(t, err)
	assert.Empty(t, evicted)
	val := cache.LookUp("dir/sub/")
	require.NotNil(t, val, "Promoted node 'dir/sub/' must return value")
	assert.Equal(t, uint64(30), val.Size())

	// Children must still be accessible
	assert.NotNil(t, cache.LookUp("dir/sub/file1"))
	assert.NotNil(t, cache.LookUp("dir/sub/file2"))

	// DEMOTION: Erase "dir/sub/" -> payload node becomes hub node again
	erasedVal := cache.Erase("dir/sub/")
	require.NotNil(t, erasedVal)
	assert.Equal(t, uint64(30), erasedVal.Size())

	// "dir/sub/" should no longer return a value
	assert.Nil(t, cache.LookUp("dir/sub/"))

	// Traversal through the demoted hub node to children MUST still succeed
	assert.NotNil(t, cache.LookUp("dir/sub/file1"), "Child file1 must survive hub demotion")
	assert.NotNil(t, cache.LookUp("dir/sub/file2"), "Child file2 must survive hub demotion")
}

// 3. Boundary Condition: Single File vs Directory Prefix Invalidation
func TestAdv_PrefixInvalidationBoundaries(t *testing.T) {
	cache := lru.NewRadixCache(1000)

	// Population
	keys := []string{
		"photos/2026/jan/1.jpg",
		"photos/2026/jan/2.jpg",
		"photos/2026/feb/1.jpg",
		"photos/2026_archive.tar",
		"photos_backup/file.txt",
	}
	for _, k := range keys {
		_, err := cache.Insert(k, advValue{bytes: 10})
		require.NoError(t, err)
	}

	// Invalidate directory prefix "photos/2026/jan/"
	cache.EraseEntriesWithGivenPrefix("photos/2026/jan/")

	// Verify exact boundary behavior
	assert.Nil(t, cache.LookUp("photos/2026/jan/1.jpg"))
	assert.Nil(t, cache.LookUp("photos/2026/jan/2.jpg"))
	assert.NotNil(t, cache.LookUp("photos/2026/feb/1.jpg"), "Sibling directory must remain intact")
	assert.NotNil(t, cache.LookUp("photos/2026_archive.tar"), "Similar prefix without trailing slash must remain intact")
	assert.NotNil(t, cache.LookUp("photos_backup/file.txt"), "Neighbor directory must remain intact")

	// Single file erase vs prefix erase
	erased := cache.Erase("photos/2026/feb/1.jpg")
	assert.NotNil(t, erased)
	assert.Nil(t, cache.LookUp("photos/2026/feb/1.jpg"))
	assert.NotNil(t, cache.LookUp("photos/2026_archive.tar"))
}

// 4. Boundary Condition: Zero-Length Strings (Empty Key & Empty Prefix)
func TestAdv_ZeroLengthStrings(t *testing.T) {
	cache := lru.NewRadixCache(100)

	// Empty string key insertion
	_, err := cache.Insert("", advValue{bytes: 15})
	require.NoError(t, err)

	val := cache.LookUp("")
	require.NotNil(t, val)
	assert.Equal(t, uint64(15), val.Size())

	// Non-empty key alongside empty key
	_, err = cache.Insert("file", advValue{bytes: 25})
	require.NoError(t, err)
	assert.NotNil(t, cache.LookUp(""))
	assert.NotNil(t, cache.LookUp("file"))

	// Erase empty key
	erased := cache.Erase("")
	assert.NotNil(t, erased)
	assert.Nil(t, cache.LookUp(""))
	assert.NotNil(t, cache.LookUp("file"))

	// Test EraseEntriesWithGivenPrefix("") resets entire cache
	_, err = cache.Insert("a/b", advValue{bytes: 10})
	require.NoError(t, err)
	cache.EraseEntriesWithGivenPrefix("")
	assert.Nil(t, cache.LookUp("file"))
	assert.Nil(t, cache.LookUp("a/b"))
}

// 5. Boundary Condition: Panic on Zero MaxSize & Exceptional Inputs
func TestAdv_RadixCacheCornerCases(t *testing.T) {
	assert.Panics(t, func() {
		lru.NewRadixCache(0)
	}, "NewRadixCache(0) must panic")

	cache := lru.NewRadixCache(50)

	// Nil value insert error
	_, err := cache.Insert("key", nil)
	assert.ErrorIs(t, err, lru.ErrInvalidEntry)

	// Update non-existent key error
	err = cache.UpdateWithoutChangingOrder("nonexistent", advValue{bytes: 10})
	assert.ErrorIs(t, err, lru.ErrEntryNotExist)

	// Update size for non-existent key error
	err = cache.UpdateSize("nonexistent", 5)
	assert.ErrorIs(t, err, lru.ErrEntryNotExist)
}
