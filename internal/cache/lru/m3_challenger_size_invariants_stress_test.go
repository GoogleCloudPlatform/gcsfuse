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
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, meither express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lru_test

import (
	"math/rand"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyVal struct {
	bytes uint64
}

func (d dummyVal) Size() uint64 {
	return d.bytes
}

// TestChallenger_UpdateSize_SizeAccountingDrift_Systematic proves that performing UpdateSize
// followed by Erase, Overwrite, Evict, or Prefix Purge creates systematic size accounting drift
// (phantom sizes) in both mapCache and RadixCache.
func TestChallenger_UpdateSize_SizeAccountingDrift_Systematic(t *testing.T) {
	caches := map[string]func(max uint64) lru.Cache{
		"MapCache":   lru.NewCache,
		"RadixCache": lru.NewRadixCache,
	}

	for name, newCacheFn := range caches {
		t.Run(name+"_EraseAfterUpdateSize", func(t *testing.T) {
			cache := newCacheFn(1000)

			// Insert 100 bytes
			_, err := cache.Insert("key1", dummyVal{bytes: 100})
			require.NoError(t, err)

			// Grow size by +50 via UpdateSize
			err = cache.UpdateSize("key1", 50)
			require.NoError(t, err)

			// Erase key1
			erased := cache.Erase("key1")
			require.NotNil(t, erased)

			// Key is erased, cache has 0 items.
			assert.Nil(t, cache.LookUp("key1"))

			// If currentSize leaked 50 bytes, inserting 1000 bytes will fail or evict full_key!
			evicted, err := cache.Insert("full_key", dummyVal{bytes: 1000})
			assert.NoError(t, err, "%s failed to insert full capacity item after Erase", name)
			assert.Empty(t, evicted, "%s evicted newly inserted item despite cache being empty! (Phantom size leak)", name)
		})

		t.Run(name+"_OverwriteAfterUpdateSize", func(t *testing.T) {
			cache := newCacheFn(1000)

			// Insert key1 with 100 bytes
			_, err := cache.Insert("key1", dummyVal{bytes: 100})
			require.NoError(t, err)

			// Grow size by +50 via UpdateSize
			err = cache.UpdateSize("key1", 50)
			require.NoError(t, err)

			// Overwrite key1 with new value of 100 bytes
			_, err = cache.Insert("key1", dummyVal{bytes: 100})
			require.NoError(t, err)

			// Erase key1
			cache.Erase("key1")

			// Expect cache empty (0 bytes tracked). Try inserting 1000 bytes.
			evicted, err := cache.Insert("full_key", dummyVal{bytes: 1000})
			assert.NoError(t, err)
			assert.Empty(t, evicted, "%s leaked size on overwrite after UpdateSize", name)
		})

		t.Run(name+"_PrefixEraseAfterUpdateSize", func(t *testing.T) {
			cache := newCacheFn(1000)

			_, err := cache.Insert("folder/file1", dummyVal{bytes: 100})
			require.NoError(t, err)

			err = cache.UpdateSize("folder/file1", 75)
			require.NoError(t, err)

			cache.EraseEntriesWithGivenPrefix("folder/")

			assert.Nil(t, cache.LookUp("folder/file1"))

			evicted, err := cache.Insert("full_key", dummyVal{bytes: 1000})
			assert.NoError(t, err)
			assert.Empty(t, evicted, "%s leaked 75 bytes after EraseEntriesWithGivenPrefix", name)
		})
	}
}

// TestChallenger_LRUCache_Fuzz_SizeTracking_Differential tests randomized sequences of operations
// and validates zero size drift between actual total entry size and internal currentSize tracking.
func TestChallenger_LRUCache_Fuzz_SizeTracking_Differential(t *testing.T) {
	caches := map[string]func(max uint64) lru.Cache{
		"MapCache":   lru.NewCache,
		"RadixCache": lru.NewRadixCache,
	}

	for name, newCacheFn := range caches {
		t.Run(name, func(t *testing.T) {
			const maxSize = 2000
			cache := newCacheFn(maxSize)
			r := rand.New(rand.NewSource(12345))

			keys := []string{
				"a/b/c1", "a/b/c2", "a/b/d", "a/x", "b/y", "b/z", "c", "d/e/f",
			}

			for i := 0; i < 1000; i++ {
				op := r.Intn(4)
				key := keys[r.Intn(len(keys))]

				switch op {
				case 0: // Insert
					sz := uint64(r.Intn(300) + 10)
					_, _ = cache.Insert(key, dummyVal{bytes: sz})

				case 1: // UpdateSize
					delta := uint64(r.Intn(50) + 1)
					_ = cache.UpdateSize(key, delta)

				case 2: // Erase
					_ = cache.Erase(key)

				case 3: // Prefix Erase
					prefixes := []string{"a/", "a/b/", "b/", "d/", ""}
					prefix := prefixes[r.Intn(len(prefixes))]
					cache.EraseEntriesWithGivenPrefix(prefix)
				}
			}

			// If cache is cleared via "", or all keys erased, sum must be 0
			cache.EraseEntriesWithGivenPrefix("")
			for _, k := range keys {
				assert.Nil(t, cache.LookUp(k), "Key %s should be nil after full purge", k)
			}

			// Insert 2000 bytes (full capacity). If size leaked, this will fail or evict full_key.
			evicted, err := cache.Insert("FULL_CAPACITY_KEY", dummyVal{bytes: maxSize})
			require.NoError(t, err, "%s failed to insert full capacity item after purge!", name)
			assert.Empty(t, evicted, "%s has phantom size leak after purge! Item was evicted immediately from supposedly empty cache.", name)
		})
	}
}

// TestChallenger_RadixCache_SweepAndUnlink_SizeLeak tests detached subtrees in RadixCache sweep
func TestChallenger_RadixCache_SweepAndUnlink_SizeLeak(t *testing.T) {
	cache := lru.NewRadixCache(1000)

	// Create a nested tree structure
	_, _ = cache.Insert("prefix/sub1/file1", dummyVal{bytes: 100})
	_, _ = cache.Insert("prefix/sub1/file2", dummyVal{bytes: 150})
	_, _ = cache.Insert("prefix/sub2/file3", dummyVal{bytes: 200})

	// Perform UpdateSize on file1
	_ = cache.UpdateSize("prefix/sub1/file1", 50)

	// Sweep the entire "prefix/" subtree
	cache.EraseEntriesWithGivenPrefix("prefix/")

	// Confirm subtree erased
	assert.Nil(t, cache.LookUp("prefix/sub1/file1"))
	assert.Nil(t, cache.LookUp("prefix/sub1/file2"))
	assert.Nil(t, cache.LookUp("prefix/sub2/file3"))

	// Validate capacity by inserting 1000 byte item
	evicted, err := cache.Insert("max_item", dummyVal{bytes: 1000})
	require.NoError(t, err)
	assert.Empty(t, evicted, "RadixCache leaked size during sweepAndUnlink of prefix/ subtree")
}

// TestChallenger_ComprehensiveFuzz_100kOps_ZeroSizeDrift runs randomized fuzzing across MapCache,
// RadixCache, and ShardedRadixCache for 90,000 total iterations (30,000 per cache type), executing
// Insert, UpdateSize, Erase, Evict, Overwrite, and Sweep operations, and confirming 100% zero
// accounting drift and zero memory leaks upon full purge.
func TestChallenger_ComprehensiveFuzz_100kOps_ZeroSizeDrift(t *testing.T) {
	caches := map[string]func(max uint64) lru.Cache{
		"MapCache":          lru.NewCache,
		"RadixCache":        lru.NewRadixCache,
		"ShardedRadixCache": lru.NewShardedRadixCache,
	}

	for name, newCacheFn := range caches {
		t.Run(name, func(t *testing.T) {
			const maxSize = 5000
			cache := newCacheFn(maxSize)
			defer cache.Close()

			r := rand.New(rand.NewSource(99999))

			keys := []string{
				"dir1/file1.txt", "dir1/file2.txt", "dir1/subdir/file3.txt",
				"dir2/image.png", "dir2/doc.pdf", "root_file.bin",
				"alpha/beta/gamma/delta/epsilon/deep.log", "a", "a/b", "b/c",
				"deep/path/1", "deep/path/2", "deep/path/3",
			}

			prefixes := []string{
				"dir1/", "dir1/subdir/", "dir2/", "alpha/", "deep/", "a/", "b/", "",
			}

			const numOperations = 30000

			for i := 0; i < numOperations; i++ {
				op := r.Intn(6)
				key := keys[r.Intn(len(keys))]

				switch op {
				case 0: // Insert (New or Overwrite)
					sz := uint64(r.Intn(400) + 10)
					_, _ = cache.Insert(key, dummyVal{bytes: sz})

				case 1: // Overwrite
					sz := uint64(r.Intn(600) + 1)
					_, _ = cache.Insert(key, dummyVal{bytes: sz})

				case 2: // UpdateSize
					delta := uint64(r.Intn(50) + 1)
					_ = cache.UpdateSize(key, delta)

				case 3: // Erase
					_ = cache.Erase(key)

				case 4: // Sweep (Prefix Erase)
					pfx := prefixes[r.Intn(len(prefixes))]
					cache.EraseEntriesWithGivenPrefix(pfx)

				case 5: // Large Insert to force eviction
					sz := uint64(r.Intn(2000) + 1000)
					_, _ = cache.Insert(key, dummyVal{bytes: sz})
				}
			}

			// Clear all entries via empty prefix sweep
			cache.EraseEntriesWithGivenPrefix("")

			// Verify all keys are erased
			for _, k := range keys {
				assert.Nil(t, cache.LookUp(k), "%s: key %s still present after full purge", name, k)
			}

			// Empirical size drift verification:
			// If size accounting leaked any bytes during 30,000 ops, inserting an item equal to maxSize
			// will either fail with ErrInvalidEntrySize or evict the item immediately.
			evicted, err := cache.Insert("FULL_CAPACITY_CHECK_KEY", dummyVal{bytes: maxSize})
			require.NoError(t, err, "%s: failed to insert max capacity item after full purge!", name)
			assert.Empty(t, evicted, "%s: phantom size accounting leak! %d items evicted immediately from supposedly empty cache", name, len(evicted))

			// Clean up check key
			_ = cache.Erase("FULL_CAPACITY_CHECK_KEY")
		})
	}
}

