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
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type challengerTestData struct {
	id   int64
	size uint64
}

func (d challengerTestData) Size() uint64 {
	return d.size
}

// TestChallenger_UpdateSizeOnEvictedAndMidEvictKeys stress tests repeated UpdateSize
// calls on keys that get evicted mid-update or right before/after UpdateSize.
func TestChallenger_UpdateSizeOnEvictedAndMidEvictKeys(t *testing.T) {
	const totalMaxSize = 200
	cache := lru.NewShardedRadixCacheWithCustomShards(totalMaxSize, 4)
	defer func() {
		if cache != nil {
			cache.Close()
		}
	}()

	// 1. Synchronous single-key self-eviction test
	// Fill cache up to capacity (200 bytes) using 10 entries of size 20
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("fill_key_%d", i)
		_, err := cache.Insert(key, challengerTestData{id: int64(i), size: 20})
		require.NoError(t, err)
	}

	// Expand fill_key_0 by 190 bytes (total desired size 210 > totalMaxSize 200)
	// UpdateSize should trigger evictions (including fill_key_0 if needed or other tail keys)
	err := cache.UpdateSize("fill_key_0", 190)
	// UpdateSize can succeed by evicting items until currentSize <= totalMaxSize.
	// If fill_key_0 itself was evicted during eviction, subsequent UpdateSize calls on fill_key_0 should fail with ErrEntryNotExist.
	assert.NoError(t, err)

	// Sub-call: update size on non-existent or evicted key must return ErrEntryNotExist
	err = cache.UpdateSize("fill_key_0", 10)
	if cache.LookUp("fill_key_0") == nil {
		assert.True(t, errors.Is(err, lru.ErrEntryNotExist), "UpdateSize on evicted key must return ErrEntryNotExist")
	}

	// 2. High concurrency multi-worker stress test
	var wg sync.WaitGroup
	numWorkers := 16
	opsPerWorker := 1000
	var panics atomic.Int64

	// Pre-insert a shared pool of keys
	for i := 0; i < 30; i++ {
		key := fmt.Sprintf("shared_evict_key_%d", i)
		_, _ = cache.Insert(key, challengerTestData{id: int64(i), size: 10})
	}

	start := make(chan struct{})

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panics.Add(1)
					t.Errorf("Panic in worker %d: %v", workerID, r)
				}
			}()

			<-start
			r := rand.New(rand.NewSource(int64(workerID * 777)))

			for i := 0; i < opsPerWorker; i++ {
				keyIdx := r.Intn(40) // Some exist, some evicted, some new
				key := fmt.Sprintf("shared_evict_key_%d", keyIdx)

				if r.Float32() < 0.6 {
					// Call UpdateSize with random expansion
					delta := uint64(r.Intn(50) + 1)
					uErr := cache.UpdateSize(key, delta)
					if uErr != nil && !errors.Is(uErr, lru.ErrEntryNotExist) {
						t.Errorf("Unexpected error from UpdateSize on %s: %v", key, uErr)
					}
				} else {
					// Re-insert or new insert
					sz := uint64(r.Intn(30) + 5)
					_, _ = cache.Insert(key, challengerTestData{id: int64(i), size: sz})
				}
			}
		}(w)
	}

	close(start)
	wg.Wait()

	assert.Equal(t, int64(0), panics.Load(), "Zero panics expected during concurrent UpdateSize mid-eviction stress")

	// Verify capacity invariant
	var calculatedSize uint64
	for i := 0; i < 40; i++ {
		key := fmt.Sprintf("shared_evict_key_%d", i)
		v := cache.LookUpWithoutChangingOrder(key)
		if v != nil {
			calculatedSize += v.Size()
		}
	}
	assert.LessOrEqual(t, calculatedSize, uint64(totalMaxSize), "Calculated total size of remaining keys must be <= totalMaxSize")
}

// TestChallenger_InterleavedInsertAndUpdateSizeOnSameKeys tests rapid interleaved Insert
// overwrites and UpdateSize expansions running concurrently on identical key sets.
func TestChallenger_InterleavedInsertAndUpdateSizeOnSameKeys(t *testing.T) {
	const totalMaxSize = 300
	cache := lru.NewShardedRadixCache(totalMaxSize)
	defer func() {
		if c, ok := cache.(*lru.ShardedRadixCache); ok {
			c.Close()
		}
	}()

	var wg sync.WaitGroup
	numWriters := 12
	numUpdaters := 12
	numReaders := 8
	opsPerWorker := 1000

	keys := make([]string, 20)
	for i := 0; i < 20; i++ {
		keys[i] = fmt.Sprintf("interleaved_key_%d", i)
	}

	start := make(chan struct{})
	var panicCount atomic.Int64

	// Writer goroutines: perform Insert overwrites
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
				}
			}()
			<-start
			r := rand.New(rand.NewSource(int64(id * 101)))
			for i := 0; i < opsPerWorker; i++ {
				k := keys[r.Intn(len(keys))]
				sz := uint64(r.Intn(40) + 1)
				_, _ = cache.Insert(k, challengerTestData{id: int64(i), size: sz})
			}
		}(w)
	}

	// Updater goroutines: perform UpdateSize expansions & UpdateWithoutChangingOrder
	for u := 0; u < numUpdaters; u++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
				}
			}()
			<-start
			r := rand.New(rand.NewSource(int64(id * 202)))
			for i := 0; i < opsPerWorker; i++ {
				k := keys[r.Intn(len(keys))]
				if r.Float32() < 0.7 {
					delta := uint64(r.Intn(15) + 1)
					_ = cache.UpdateSize(k, delta)
				} else {
					// Fetch current value to match size for UpdateWithoutChangingOrder
					curr := cache.LookUpWithoutChangingOrder(k)
					if curr != nil {
						_ = cache.UpdateWithoutChangingOrder(k, challengerTestData{id: int64(i), size: curr.Size()})
					}
				}
			}
		}(u)
	}

	// Reader goroutines: perform LookUp & LookUpWithoutChangingOrder
	for rIdx := 0; rIdx < numReaders; rIdx++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
				}
			}()
			<-start
			r := rand.New(rand.NewSource(int64(id * 303)))
			for i := 0; i < opsPerWorker; i++ {
				k := keys[r.Intn(len(keys))]
				if i%2 == 0 {
					_ = cache.LookUp(k)
				} else {
					_ = cache.LookUpWithoutChangingOrder(k)
				}
			}
		}(rIdx)
	}

	close(start)
	wg.Wait()

	assert.Equal(t, int64(0), panicCount.Load(), "Zero panics expected during interleaved Insert and UpdateSize stress")

	// Post-execution capacity invariant validation
	var totalCalculated uint64
	for _, k := range keys {
		if v := cache.LookUpWithoutChangingOrder(k); v != nil {
			totalCalculated += v.Size()
		}
	}
	assert.LessOrEqual(t, totalCalculated, uint64(totalMaxSize), "Sum of sizes of items in cache must be <= totalMaxSize")
}

// TestChallenger_RapidPrefixEraseParallelWithUpdateSizeAndInsert tests high frequency EraseEntriesWithGivenPrefix
// operations running concurrently with parallel Insert, UpdateSize, and LookUp workloads.
func TestChallenger_RapidPrefixEraseParallelWithUpdateSizeAndInsert(t *testing.T) {
	const totalMaxSize = 1000
	cache := lru.NewShardedRadixCache(totalMaxSize)
	defer func() {
		if c, ok := cache.(*lru.ShardedRadixCache); ok {
			c.Close()
		}
	}()

	var wg sync.WaitGroup
	duration := 2 * time.Second
	stop := make(chan struct{})
	var panics atomic.Int64

	prefixes := []string{
		"tenant_a/folder_1/",
		"tenant_a/folder_2/",
		"tenant_a/",
		"tenant_b/folder_1/",
		"tenant_b/",
		"",
	}

	// 8 Writers doing Insert
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panics.Add(1)
				}
			}()
			r := rand.New(rand.NewSource(int64(id * 11)))
			i := 0
			for {
				select {
				case <-stop:
					return
				default:
					tenant := "tenant_a"
					if id%2 == 1 {
						tenant = "tenant_b"
					}
					folder := fmt.Sprintf("folder_%d", r.Intn(3)+1)
					file := fmt.Sprintf("file_%d.dat", r.Intn(20))
					key := fmt.Sprintf("%s/%s/%s", tenant, folder, file)
					sz := uint64(r.Intn(50) + 10)
					_, _ = cache.Insert(key, challengerTestData{id: int64(i), size: sz})
					i++
				}
			}
		}(w)
	}

	// 8 Updaters doing UpdateSize
	for u := 0; u < 8; u++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panics.Add(1)
				}
			}()
			r := rand.New(rand.NewSource(int64(id * 22)))
			for {
				select {
				case <-stop:
					return
				default:
					tenant := "tenant_a"
					if id%2 == 1 {
						tenant = "tenant_b"
					}
					folder := fmt.Sprintf("folder_%d", r.Intn(3)+1)
					file := fmt.Sprintf("file_%d.dat", r.Intn(20))
					key := fmt.Sprintf("%s/%s/%s", tenant, folder, file)
					delta := uint64(r.Intn(20) + 1)
					_ = cache.UpdateSize(key, delta)
				}
			}
		}(u)
	}

	// 4 Erasers doing rapid EraseEntriesWithGivenPrefix
	for e := 0; e < 4; e++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panics.Add(1)
				}
			}()
			r := rand.New(rand.NewSource(int64(id * 33)))
			for {
				select {
				case <-stop:
					return
				default:
					pfx := prefixes[r.Intn(len(prefixes))]
					cache.EraseEntriesWithGivenPrefix(pfx)
					time.Sleep(500 * time.Microsecond)
				}
			}
		}(e)
	}

	// 8 Readers doing LookUp
	for rIdx := 0; rIdx < 8; rIdx++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panics.Add(1)
				}
			}()
			r := rand.New(rand.NewSource(int64(id * 44)))
			for {
				select {
				case <-stop:
					return
				default:
					tenant := "tenant_a"
					if id%2 == 1 {
						tenant = "tenant_b"
					}
					folder := fmt.Sprintf("folder_%d", r.Intn(3)+1)
					file := fmt.Sprintf("file_%d.dat", r.Intn(20))
					key := fmt.Sprintf("%s/%s/%s", tenant, folder, file)
					_ = cache.LookUp(key)
				}
			}
		}(rIdx)
	}

	time.Sleep(duration)
	close(stop)
	wg.Wait()

	assert.Equal(t, int64(0), panics.Load(), "Zero panics expected during rapid prefix erasure stress test")

	// Invariant validation: calculate remaining sizes across all possible key combinations
	var totalCalculated uint64
	for _, tName := range []string{"tenant_a", "tenant_b"} {
		for fIdx := 1; fIdx <= 3; fIdx++ {
			for fileIdx := 0; fileIdx < 20; fileIdx++ {
				key := fmt.Sprintf("%s/folder_%d/file_%d.dat", tName, fIdx, fileIdx)
				if v := cache.LookUpWithoutChangingOrder(key); v != nil {
					totalCalculated += v.Size()
				}
			}
		}
	}
	assert.LessOrEqual(t, totalCalculated, uint64(totalMaxSize), "Sum of sizes of items remaining in cache must be <= totalMaxSize")
}
