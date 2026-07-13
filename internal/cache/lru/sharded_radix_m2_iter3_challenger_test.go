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

type iter3TestData struct {
	val  int64
	size uint64
}

func (d iter3TestData) Size() uint64 {
	return d.size
}

// TestM2Iter3_TailKeyParallelInsertAndUpdateSize stress-tests high-throughput parallel
// Insert and UpdateSize operations specifically targeting keys sitting at the LRU tail of shards.
func TestM2Iter3_TailKeyParallelInsertAndUpdateSize(t *testing.T) {
	const totalMaxSize = 600
	const numShards = 4
	cache := lru.NewShardedRadixCacheWithCustomShards(totalMaxSize, numShards)
	defer cache.Close()

	var wg sync.WaitGroup
	numWorkers := 16
	opsPerWorker := 800
	var panicCount atomic.Int64

	// Generate a pool of keys distributed across shards
	keys := make([]string, 40)
	for i := 0; i < 40; i++ {
		keys[i] = fmt.Sprintf("tail_stress_key_%d", i)
	}

	// Initial fill to saturate cache capacity
	for i, k := range keys {
		_, err := cache.Insert(k, iter3TestData{val: int64(i), size: 15})
		require.NoError(t, err)
	}

	start := make(chan struct{})

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
					t.Errorf("Panic in worker %d: %v", workerID, r)
				}
			}()

			<-start
			r := rand.New(rand.NewSource(int64(workerID * 1337)))

			for i := 0; i < opsPerWorker; i++ {
				key := keys[r.Intn(len(keys))]

				// By routinely looking up head/middle keys, we force non-accessed keys to the tail
				if r.Float32() < 0.3 {
					headKey := keys[r.Intn(10)]
					_ = cache.LookUp(headKey)
				}

				if r.Float32() < 0.5 {
					// Call UpdateSize on key (which may be at tail and subject to concurrent eviction)
					delta := uint64(r.Intn(30) + 1)
					uErr := cache.UpdateSize(key, delta)
					if uErr != nil && !errors.Is(uErr, lru.ErrEntryNotExist) {
						t.Errorf("Unexpected UpdateSize error on %s: %v", key, uErr)
					}
				} else {
					// Perform Insert overwrite with variable size
					newSize := uint64(r.Intn(60) + 5)
					_, err := cache.Insert(key, iter3TestData{val: int64(i), size: newSize})
					if err != nil && !errors.Is(err, lru.ErrInvalidEntrySize) {
						t.Errorf("Unexpected Insert error on %s: %v", key, err)
					}
				}
			}
		}(w)
	}

	close(start)
	wg.Wait()

	assert.Equal(t, int64(0), panicCount.Load(), "Zero panics expected during tail key stress")

	// Verify exact capacity bound
	var totalSize uint64
	for _, k := range keys {
		if v := cache.LookUpWithoutChangingOrder(k); v != nil {
			totalSize += v.Size()
		}
	}
	assert.LessOrEqual(t, totalSize, uint64(totalMaxSize), "Calculated remaining total size must not exceed totalMaxSize")
}

// TestM2Iter3_DynamicSizeExpansionMultiShardFallbackEviction stress-tests dynamic entry size
// expansions and overwrites under heavy multi-shard fallback eviction where local shard capacity
// is insufficient and calcDelta() must dynamically recalculate pending delta under multi-shard locks.
func TestM2Iter3_DynamicSizeExpansionMultiShardFallbackEviction(t *testing.T) {
	const totalMaxSize = 400
	const numShards = 4
	cache := lru.NewShardedRadixCacheWithCustomShards(totalMaxSize, numShards)
	defer cache.Close()

	var wg sync.WaitGroup
	numWorkers := 16
	duration := 1500 * time.Millisecond
	stop := make(chan struct{})
	var panicCount atomic.Int64

	// Pool of keys targeting different shards
	keys := make([]string, 24)
	for i := 0; i < 24; i++ {
		keys[i] = fmt.Sprintf("multishard_fallback_key_%d", i)
	}

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
					t.Errorf("Panic in fallback worker %d: %v", workerID, r)
				}
			}()

			r := rand.New(rand.NewSource(int64(workerID * 9999)))
			opCount := 0

			for {
				select {
				case <-stop:
					return
				default:
					opCount++
					key := keys[r.Intn(len(keys))]

					// 20% of operations perform huge expansions (300+ bytes) requiring global multi-shard eviction
					var sz uint64
					if r.Float32() < 0.2 {
						sz = uint64(r.Intn(100) + 250) // 250-350 bytes out of 400 total
					} else {
						sz = uint64(r.Intn(30) + 5)
					}

					if opCount%2 == 0 {
						_, err := cache.Insert(key, iter3TestData{val: int64(opCount), size: sz})
						if err != nil && !errors.Is(err, lru.ErrInvalidEntrySize) {
							t.Errorf("Unexpected error during multi-shard fallback Insert: %v", err)
						}
					} else {
						delta := uint64(r.Intn(40) + 1)
						err := cache.UpdateSize(key, delta)
						if err != nil && !errors.Is(err, lru.ErrEntryNotExist) {
							t.Errorf("Unexpected error during multi-shard fallback UpdateSize: %v", err)
						}
					}
				}
			}
		}(w)
	}

	time.Sleep(duration)
	close(stop)
	wg.Wait()

	assert.Equal(t, int64(0), panicCount.Load(), "Zero panics expected during multi-shard fallback eviction stress")

	// Validate remaining total size invariant
	var calculatedSize uint64
	for _, k := range keys {
		if v := cache.LookUpWithoutChangingOrder(k); v != nil {
			calculatedSize += v.Size()
		}
	}
	assert.LessOrEqual(t, calculatedSize, uint64(totalMaxSize), "Calculated total size of remaining keys must be <= totalMaxSize")
}

// TestM2Iter3_RapidPrefixEraseConcurrentWithKeyInsertions stress-tests high-frequency
// prefix erasures concurrent with key insertions, overwrites, and size updates.
func TestM2Iter3_RapidPrefixEraseConcurrentWithKeyInsertions(t *testing.T) {
	const totalMaxSize = 1000
	cache := lru.NewShardedRadixCache(totalMaxSize)
	defer func() {
		if c, ok := cache.(*lru.ShardedRadixCache); ok {
			c.Close()
		}
	}()

	var wg sync.WaitGroup
	duration := 1500 * time.Millisecond
	stop := make(chan struct{})
	var panicCount atomic.Int64

	prefixes := []string{
		"root/tenant_a/",
		"root/tenant_b/",
		"root/tenant_a/shard_1/",
		"root/tenant_b/shard_2/",
		"root/",
		"",
	}

	// 10 Writers executing Insert
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
				}
			}()

			r := rand.New(rand.NewSource(int64(id * 43)))
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
					shardIdx := r.Intn(4) + 1
					fileIdx := r.Intn(20)
					key := fmt.Sprintf("root/%s/shard_%d/file_%d.dat", tenant, shardIdx, fileIdx)
					sz := uint64(r.Intn(60) + 10)
					_, _ = cache.Insert(key, iter3TestData{val: int64(i), size: sz})
					i++
				}
			}
		}(w)
	}

	// 6 Updaters executing UpdateSize
	for u := 0; u < 6; u++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
				}
			}()

			r := rand.New(rand.NewSource(int64(id * 87)))
			for {
				select {
				case <-stop:
					return
				default:
					tenant := "tenant_a"
					if id%2 == 1 {
						tenant = "tenant_b"
					}
					shardIdx := r.Intn(4) + 1
					fileIdx := r.Intn(20)
					key := fmt.Sprintf("root/%s/shard_%d/file_%d.dat", tenant, shardIdx, fileIdx)
					delta := uint64(r.Intn(15) + 1)
					_ = cache.UpdateSize(key, delta)
				}
			}
		}(u)
	}

	// 4 Erasers executing rapid EraseEntriesWithGivenPrefix
	for e := 0; e < 4; e++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
				}
			}()

			r := rand.New(rand.NewSource(int64(id * 109)))
			for {
				select {
				case <-stop:
					return
				default:
					pfx := prefixes[r.Intn(len(prefixes))]
					cache.EraseEntriesWithGivenPrefix(pfx)
					time.Sleep(300 * time.Microsecond)
				}
			}
		}(e)
	}

	time.Sleep(duration)
	close(stop)
	wg.Wait()

	assert.Equal(t, int64(0), panicCount.Load(), "Zero panics expected during rapid prefix erasure stress")

	// Validate capacity bound post-erasure
	var calculatedTotal uint64
	for _, tenant := range []string{"tenant_a", "tenant_b"} {
		for sIdx := 1; sIdx <= 4; sIdx++ {
			for fIdx := 0; fIdx < 20; fIdx++ {
				key := fmt.Sprintf("root/%s/shard_%d/file_%d.dat", tenant, sIdx, fIdx)
				if v := cache.LookUpWithoutChangingOrder(key); v != nil {
					calculatedTotal += v.Size()
				}
			}
		}
	}
	assert.LessOrEqual(t, calculatedTotal, uint64(totalMaxSize), "Calculated total size of remaining keys must be <= totalMaxSize")
}
