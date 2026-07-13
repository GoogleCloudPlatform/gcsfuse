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
	"sync/atomic"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	locker.EnableInvariantsCheck()
}

// Scenario 1: Parallel UpdateSize operations on existing nodes concurrent with heavy lookups, insertions, and deletions.
func TestStress_ParallelUpdateSizeWithHeavyConcurrentOps(t *testing.T) {
	const totalMaxSize = 5000
	const numShards = 16
	cache := lru.NewShardedRadixCacheWithCustomShards(totalMaxSize, numShards)
	defer cache.Close()

	// Initial population
	const numKeys = 200
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("tenant_%d/folder_%d/file_%d.dat", i%4, i%10, i)
		_, err := cache.Insert(key, stressTestData{val: int64(i), size: 20})
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	numWorkers := 32
	opsPerWorker := 800
	var totalUpdateSizeOps atomic.Uint64

	start := make(chan struct{})

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-start
			r := rand.New(rand.NewSource(int64(workerID * 7919)))

			for i := 0; i < opsPerWorker; i++ {
				keyID := r.Intn(numKeys * 2) // Some existing, some new
				key := fmt.Sprintf("tenant_%d/folder_%d/file_%d.dat", keyID%4, keyID%10, keyID)

				op := r.Intn(100)
				switch {
				case op < 40:
					// UpdateSize operation
					sizeDelta := uint64(r.Intn(50) + 1)
					err := cache.UpdateSize(key, sizeDelta)
					if err == nil {
						totalUpdateSizeOps.Add(1)
					}
				case op < 70:
					// LookUp operation
					_ = cache.LookUp(key)
				case op < 85:
					// Insert operation
					size := uint64(r.Intn(100) + 10)
					_, _ = cache.Insert(key, stressTestData{val: int64(i), size: size})
				case op < 95:
					// Erase operation
					_ = cache.Erase(key)
				default:
					// Prefix erasure
					prefix := fmt.Sprintf("tenant_%d/folder_%d/", r.Intn(4), r.Intn(10))
					cache.EraseEntriesWithGivenPrefix(prefix)
				}
			}
		}(w)
	}

	close(start)
	wg.Wait()

	t.Logf("Completed stress test with %d successful UpdateSize ops", totalUpdateSizeOps.Load())

	// Invariant validation: calculate actual size of all remaining items across all keys
	var calculatedSize uint64
	var count int
	for i := 0; i < numKeys*2; i++ {
		key := fmt.Sprintf("tenant_%d/folder_%d/file_%d.dat", i%4, i%10, i)
		v := cache.LookUpWithoutChangingOrder(key)
		if v != nil {
			calculatedSize += v.Size()
			count++
		}
	}

	t.Logf("Final cache count: %d, total calculated size: %d, maxSize: %d", count, calculatedSize, totalMaxSize)
	assert.LessOrEqual(t, calculatedSize, uint64(totalMaxSize), "Calculated total size exceeds totalMaxSize invariant")
}

// Scenario 2: Dynamic entry size expansions driving cache capacity to maxSize and totalMaxSize.
func TestStress_DynamicEntrySizeExpansionsToMaxSize(t *testing.T) {
	const totalMaxSize = 2000
	const numShards = 8
	cache := lru.NewShardedRadixCacheWithCustomShards(totalMaxSize, numShards)
	defer cache.Close()

	// Insert 10 base files of size 50 each (total size 500)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("file_%d.txt", i)
		_, err := cache.Insert(key, stressTestData{val: int64(i), size: 50})
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	// Workers will dynamically expand file sizes aggressively to hit totalMaxSize
	numWorkers := 16
	duration := 2 * time.Second
	stop := make(chan struct{})

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(workerID * 104729)))

			for {
				select {
				case <-stop:
					return
				default:
					fileIdx := r.Intn(15) // Expand existing or insert new
					key := fmt.Sprintf("file_%d.txt", fileIdx)

					if r.Float32() < 0.6 {
						// Expand entry size up to 500 bytes at a time
						expansion := uint64(r.Intn(450) + 50)
						_ = cache.UpdateSize(key, expansion)
					} else {
						// Insert entry directly near max capacity limit
						size := uint64(r.Intn(800) + 100)
						_, _ = cache.Insert(key, stressTestData{val: int64(fileIdx), size: size})
					}
				}
			}
		}(w)
	}

	time.Sleep(duration)
	close(stop)
	wg.Wait()

	// Invariant validation: ensure final total size does not exceed totalMaxSize
	var calculatedSize uint64
	for i := 0; i < 15; i++ {
		key := fmt.Sprintf("file_%d.txt", i)
		v := cache.LookUpWithoutChangingOrder(key)
		if v != nil {
			calculatedSize += v.Size()
		}
	}

	t.Logf("Dynamic expansion final calculated size: %d / totalMaxSize: %d", calculatedSize, totalMaxSize)
	assert.LessOrEqual(t, calculatedSize, uint64(totalMaxSize), "Capacity invariant violated after aggressive size expansions")
}

// Scenario 3: Multi-shard global fallback eviction triggering under high parallel write load across shards.
func TestStress_MultiShardGlobalFallbackEvictionUnderHighParallelLoad(t *testing.T) {
	const totalMaxSize = 1200
	const numShards = 16
	cache := lru.NewShardedRadixCacheWithCustomShards(totalMaxSize, numShards)
	defer cache.Close()

	// Pre-fill cache across all shards with small items
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("item_%d", i)
		_, err := cache.Insert(key, stressTestData{val: int64(i), size: 10})
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	numWriters := 24
	opsPerWriter := 500

	start := make(chan struct{})

	// Parallel writers inserting medium/large entries across shards, forcing multi-shard global fallback evictions
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			<-start
			r := rand.New(rand.NewSource(int64(writerID * 31337)))

			for i := 0; i < opsPerWriter; i++ {
				shardTargetKey := fmt.Sprintf("writer_%d_item_%d", writerID, i%20)
				// Insertion size up to 400 bytes, which exceeds single shard average capacity and triggers multi-shard fallback
				size := uint64(r.Intn(350) + 50)
				_, _ = cache.Insert(shardTargetKey, stressTestData{val: int64(i), size: size})

				if i%5 == 0 {
					// Interleave UpdateSize that further stresses global capacity limits
					_ = cache.UpdateSize(shardTargetKey, uint64(r.Intn(100)+10))
				}
			}
		}(w)
	}

	close(start)
	wg.Wait()

	// Final verification of total capacity across all shards
	var calculatedSize uint64
	for w := 0; w < numWriters; w++ {
		for i := 0; i < 20; i++ {
			key := fmt.Sprintf("writer_%d_item_%d", w, i)
			if v := cache.LookUpWithoutChangingOrder(key); v != nil {
				calculatedSize += v.Size()
			}
		}
	}
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("item_%d", i)
		if v := cache.LookUpWithoutChangingOrder(key); v != nil {
			calculatedSize += v.Size()
		}
	}

	t.Logf("Multi-shard global fallback eviction final calculated size: %d / totalMaxSize: %d", calculatedSize, totalMaxSize)
	assert.LessOrEqual(t, calculatedSize, uint64(totalMaxSize), "Global fallback eviction failed to enforce totalMaxSize bound")
}
