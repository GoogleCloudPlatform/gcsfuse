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
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type empiricalVal struct {
	data string
	size uint64
}

func (v empiricalVal) Size() uint64 {
	return v.size
}


// TestShardedRadixCache_DeepFolderTrees_ConcurrentEviction tests multi-goroutine insertion, lookup,
// and prefix erasure on deep folder structures (up to 25 directory levels) under low cache capacity
// forcing constant shard-local and global SIEVE evictions.
func TestShardedRadixCache_DeepFolderTrees_ConcurrentEviction(t *testing.T) {
	// Small maxSize forcing continuous global and local evictions
	const maxSize = 20000
	cache := lru.NewShardedRadixCache(maxSize)
	defer cache.Close()

	const numWorkers = 30
	const opsPerWorker = 300
	var wg sync.WaitGroup

	// Helper to generate deep directory path
	makeDeepPath := func(rng *rand.Rand, depth int, id int) string {
		parts := make([]string, depth)
		for d := 0; d < depth; d++ {
			parts[d] = fmt.Sprintf("dir_lvl_%d_%d", d, rng.Intn(3))
		}
		parts = append(parts, fmt.Sprintf("file_%d.dat", id))
		return strings.Join(parts, "/")
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(workerID * 777)))
			for op := 0; op < opsPerWorker; op++ {
				depth := rng.Intn(20) + 1 // 1 to 20 levels deep
				path := makeDeepPath(rng, depth, rng.Intn(50))
				valSize := uint64(rng.Intn(300) + 10)
				val := empiricalVal{data: path, size: valSize}

				switch rng.Intn(5) {
				case 0, 1: // Insert
					_, err := cache.Insert(path, val)
					if err != nil {
						assert.Equal(t, lru.ErrInvalidEntrySize, err)
					}
				case 2: // LookUp & LookUpWithoutChangingOrder
					_ = cache.LookUp(path)
					_ = cache.LookUpWithoutChangingOrder(path)
				case 3: // Erase single key
					_ = cache.Erase(path)
				case 4: // Subtree prefix erase
					prefixDepth := rng.Intn(depth) + 1
					parts := strings.Split(path, "/")
					if prefixDepth <= len(parts) {
						prefix := strings.Join(parts[:prefixDepth], "/") + "/"
						cache.EraseEntriesWithGivenPrefix(prefix)
					}
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestShardedRadixCache_AdversarialKeys_HighConcurrency tests boundary/edge-case keys under multi-goroutine load:
// empty string, single slash, repeating prefixes, very long keys, non-ASCII strings.
func TestShardedRadixCache_AdversarialKeys_HighConcurrency(t *testing.T) {
	cache := lru.NewShardedRadixCache(50000)
	defer cache.Close()

	adversarialKeys := []string{
		"",
		"/",
		"//",
		"///",
		"a",
		"a/b",
		"a/b/c",
		"a/b/c/d",
		"a/b/c/d/e",
		strings.Repeat("long_prefix_part/", 50) + "file.txt",
		"spaces in path/sub folder/my file.txt",
		"unicode_日本語_test/こんにちは/world.bin",
		"very/similar/path1",
		"very/similar/path2",
		"very/similar/path3",
	}

	const numGoroutines = 40
	var wg sync.WaitGroup
	var stopFlag atomic.Bool

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(gID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(gID * 12345)))
			for !stopFlag.Load() {
				key := adversarialKeys[rng.Intn(len(adversarialKeys))]
				val := empiricalVal{data: key, size: uint64(rng.Intn(100) + 1)}

				action := rng.Intn(7)
				switch action {
				case 0:
					_, _ = cache.Insert(key, val)
				case 1:
					_ = cache.LookUp(key)
				case 2:
					_ = cache.LookUpWithoutChangingOrder(key)
				case 3:
					_ = cache.Erase(key)
				case 4:
					_ = cache.UpdateWithoutChangingOrder(key, val)
				case 5:
					_ = cache.UpdateSize(key, uint64(rng.Intn(20)))
				case 6:
					if key != "" {
						cache.EraseEntriesWithGivenPrefix(lru.ParentDirectoryPrefix(key))
					}
				}
			}
		}(i)
	}

	time.Sleep(1500 * time.Millisecond)
	stopFlag.Store(true)
	wg.Wait()
}

// TestShardedRadixCache_GlobalVsLocalEviction_DeadlockStress tests cross-shard locks and background worker interaction.
// Global eviction attempts to acquire locks across all 256 shards while workers concurrently acquire shard locks.
func TestShardedRadixCache_GlobalVsLocalEviction_DeadlockStress(t *testing.T) {
	// maxSize set to 262144 (256KB) so each shard's maxSize = 1024 (activating local eviction path)
	cache := lru.NewShardedRadixCache(262144)
	defer cache.Close()

	const numWorkers = 50
	var wg sync.WaitGroup
	var stopFlag atomic.Bool

	// Spawn workers generating entries that hash across all 256 shards
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(wID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(wID * 9999)))
			step := 0
			for !stopFlag.Load() {
				step++
				// Diversify shard keys by using different parent directory prefixes
				key := fmt.Sprintf("shard_dir_%d/sub_%d/item_%d", rng.Intn(300), rng.Intn(10), step)
				// Large sizes to trigger rapid capacity limits and global eviction loops
				sz := uint64(rng.Intn(2000) + 100)
				val := empiricalVal{data: key, size: sz}

				_, _ = cache.Insert(key, val)

				if step%10 == 0 {
					pfx := fmt.Sprintf("shard_dir_%d/", rng.Intn(300))
					cache.EraseEntriesWithGivenPrefix(pfx)
				}
			}
		}(i)
	}

	// Concurrent pruner thread calling PruneAllEmptyLeaves continuously
	wg.Add(1)
	go func() {
		defer wg.Done()
		for !stopFlag.Load() {
			if sc, ok := cache.(*lru.ShardedRadixCache); ok {
				sc.PruneAllEmptyLeaves()
			}
			time.Sleep(1 * time.Millisecond)
		}
	}()

	time.Sleep(2 * time.Second)
	stopFlag.Store(true)
	wg.Wait()
}

// TestShardedRadixCache_ConcurrentPrefixErase_SubtreeDetachment stress tests background detached subtree sweeping
// with simultaneous EraseEntriesWithGivenPrefix on shared parent prefixes.
func TestShardedRadixCache_ConcurrentPrefixErase_SubtreeDetachment(t *testing.T) {
	const maxSize = 500000
	cache := lru.NewShardedRadixCache(maxSize)
	defer cache.Close()

	const numProducers = 15
	const numErasers = 10
	var wg sync.WaitGroup
	var stopFlag atomic.Bool

	// Producers inserting deep branching subtrees
	for i := 0; i < numProducers; i++ {
		wg.Add(1)
		go func(pID int) {
			defer wg.Done()
			count := 0
			for !stopFlag.Load() {
				count++
				key := fmt.Sprintf("tenant_%d/bucket_%d/folder_%d/obj_%d", pID%5, count%10, count%20, count)
				val := empiricalVal{data: key, size: 50}
				_, _ = cache.Insert(key, val)
			}
		}(i)
	}

	// Erasers clearing entire tenant and bucket subtrees
	for i := 0; i < numErasers; i++ {
		wg.Add(1)
		go func(eID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(eID * 8765)))
			for !stopFlag.Load() {
				tenantID := rng.Intn(5)
				switch rng.Intn(3) {
				case 0:
					cache.EraseEntriesWithGivenPrefix(fmt.Sprintf("tenant_%d/", tenantID))
				case 1:
					bucketID := rng.Intn(10)
					cache.EraseEntriesWithGivenPrefix(fmt.Sprintf("tenant_%d/bucket_%d/", tenantID, bucketID))
				case 2:
					// Clear root
					cache.EraseEntriesWithGivenPrefix("")
				}
				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}

	time.Sleep(2 * time.Second)
	stopFlag.Store(true)
	wg.Wait()
}

// TestShardedRadixCache_HighContention_SizeAccounting Integrity check to ensure currentSize stays consistent
// and doesn't crash or underflow under race detector.
func TestShardedRadixCache_HighContention_SizeAccounting(t *testing.T) {
	const maxSize = 100000
	cache := lru.NewShardedRadixCache(maxSize)
	defer cache.Close()

	const numWorkers = 20
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(wID int) {
			defer wg.Done()
			for k := 0; k < 100; k++ {
				key := fmt.Sprintf("work_%d_%d", wID, k)
				val := empiricalVal{data: key, size: 1000}
				_, err := cache.Insert(key, val)
				require.NoError(t, err)

				// UpdateSize
				_ = cache.UpdateSize(key, 500)

				// Erase
				_ = cache.Erase(key)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(50 * time.Millisecond) // Let background sweep finish

	// Clear everything
	cache.EraseEntriesWithGivenPrefix("")
	time.Sleep(100 * time.Millisecond)

	// Inserting a new item after complete clearing should succeed without immediate eviction
	_, err := cache.Insert("fresh_key", empiricalVal{data: "fresh", size: 50})
	require.NoError(t, err)
	assert.NotNil(t, cache.LookUp("fresh_key"), "Freshly inserted key after full purge must be retrievable")
}
