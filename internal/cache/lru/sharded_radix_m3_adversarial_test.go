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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type m3AdversarialData struct {
	id   int64
	size uint64
}

func (d m3AdversarialData) Size() uint64 {
	return d.size
}

// pregenerateCollidingKeys produces keys that all hash to the exact same shard index (shard 0)
// for fnv1a mod 32.
func pregenerateCollidingKeys(count int, targetShard uint32, numShards int) []string {
	keys := make([]string, 0, count)
	mask := uint32(numShards - 1)
	i := 0
	for len(keys) < count {
		candidate := fmt.Sprintf("colliding_key_candidate_%d", i)
		hash := uint32(2166136261)
		for j := 0; j < len(candidate); j++ {
			hash ^= uint32(candidate[j])
			hash *= 16777619
		}
		if (hash & mask) == targetShard {
			keys = append(keys, candidate)
		}
		i++
	}
	return keys
}

// TestM3_PathologicalHashCollisionHotspot tests correctness and capacity invariants when all operations
// target keys that hash to the exact same shard.
func TestM3_PathologicalHashCollisionHotspot(t *testing.T) {
	const totalMaxSize = 500
	const numShards = 16
	cache := lru.NewShardedRadixCacheWithCustomShards(totalMaxSize, numShards)
	defer cache.Close()

	collidingKeys := pregenerateCollidingKeys(50, 0, numShards)

	var wg sync.WaitGroup
	numWorkers := 16
	opsPerWorker := 500
	var panicCount atomic.Int64

	start := make(chan struct{})

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
					t.Errorf("Panic in collision worker %d: %v", workerID, r)
				}
			}()

			<-start
			r := rand.New(rand.NewSource(int64(workerID * 4321)))

			for i := 0; i < opsPerWorker; i++ {
				key := collidingKeys[r.Intn(len(collidingKeys))]
				op := r.Float32()

				if op < 0.4 {
					_ = cache.LookUp(key)
				} else if op < 0.7 {
					sz := uint64(r.Intn(30) + 5)
					_, _ = cache.Insert(key, m3AdversarialData{id: int64(i), size: sz})
				} else if op < 0.9 {
					delta := uint64(r.Intn(20) + 1)
					_ = cache.UpdateSize(key, delta)
				} else {
					_ = cache.Erase(key)
				}
			}
		}(w)
	}

	close(start)
	wg.Wait()

	assert.Equal(t, int64(0), panicCount.Load(), "Zero panics expected during pathological hash collision stress")

	// Capacity invariant validation
	var calculatedTotal uint64
	for _, k := range collidingKeys {
		if v := cache.LookUpWithoutChangingOrder(k); v != nil {
			calculatedTotal += v.Size()
		}
	}
	assert.LessOrEqual(t, calculatedTotal, uint64(totalMaxSize), "Calculated total size of items remaining in single colliding shard must be <= totalMaxSize")
}

// TestM3_ContinuousMultiShardFallbackThrashing tests behavior under intentional extreme capacity pressure
// where EVERY insert/update exceeds local shard capacity and forces multi-shard fallback global evictions.
func TestM3_ContinuousMultiShardFallbackThrashing(t *testing.T) {
	const totalMaxSize = 100 // Very tiny capacity relative to item sizes
	const numShards = 16
	cache := lru.NewShardedRadixCacheWithCustomShards(totalMaxSize, numShards)
	defer cache.Close()

	var wg sync.WaitGroup
	numWorkers := 32
	duration := 1500 * time.Millisecond
	stop := make(chan struct{})
	var panicCount atomic.Int64
	var totalOps atomic.Uint64

	keys := make([]string, 64)
	for i := 0; i < 64; i++ {
		keys[i] = fmt.Sprintf("thrash_key_%d", i)
	}

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
				}
			}()

			r := rand.New(rand.NewSource(int64(workerID * 8888)))

			for {
				select {
				case <-stop:
					return
				default:
					totalOps.Add(1)
					key := keys[r.Intn(len(keys))]
					// Items of size 40-80 out of 100 total capacity force continuous multi-shard locks
					sz := uint64(r.Intn(40) + 40)

					if r.Float32() < 0.6 {
						_, err := cache.Insert(key, m3AdversarialData{id: int64(workerID), size: sz})
						if err != nil && err != lru.ErrInvalidEntrySize {
							t.Errorf("Unexpected Insert error during thrashing: %v", err)
						}
					} else {
						_ = cache.UpdateSize(key, uint64(r.Intn(10)+1))
					}
				}
			}
		}(w)
	}

	time.Sleep(duration)
	close(stop)
	wg.Wait()

	assert.Equal(t, int64(0), panicCount.Load(), "Zero panics expected during continuous multi-shard thrashing")
	require.Greater(t, totalOps.Load(), uint64(100), "Operations must execute during thrashing test")

	// Capacity invariant validation
	var totalCalculated uint64
	for _, k := range keys {
		if v := cache.LookUpWithoutChangingOrder(k); v != nil {
			totalCalculated += v.Size()
		}
	}
	assert.LessOrEqual(t, totalCalculated, uint64(totalMaxSize), "Calculated total size post-thrashing must be <= totalMaxSize")
}

// BenchmarkAdversarialHashCollision benchmarks performance degradation of ShardedRadixCache
// under 100% hash key collision compared to uniform distribution and single-mutex RadixCache.
func BenchmarkAdversarialHashCollision(b *testing.B) {
	const numKeys = 1000
	const numShards = 32
	capacity := uint64(numKeys * 50)
	data := m3AdversarialData{id: 1, size: 10}

	collidingKeys := pregenerateCollidingKeys(numKeys, 0, numShards)
	uniformKeys := make([]string, numKeys)
	for i := 0; i < numKeys; i++ {
		uniformKeys[i] = fmt.Sprintf("uniform_key_path_%d/file_%d.txt", i%10, i)
	}

	b.Run("ShardedRadix_UniformDistribution", func(b *testing.B) {
		cache := lru.NewShardedRadixCacheWithCustomShards(capacity, numShards)
		defer cache.Close()
		for _, k := range uniformKeys {
			_, _ = cache.Insert(k, data)
		}
		var counter atomic.Uint64
		b.ReportAllocs()
		b.ResetTimer()
		b.SetParallelism(32)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				idx := counter.Add(1)
				k := uniformKeys[idx%uint64(numKeys)]
				_ = cache.LookUp(k)
			}
		})
	})

	b.Run("ShardedRadix_100PercentHashCollision", func(b *testing.B) {
		cache := lru.NewShardedRadixCacheWithCustomShards(capacity, numShards)
		defer cache.Close()
		for _, k := range collidingKeys {
			_, _ = cache.Insert(k, data)
		}
		var counter atomic.Uint64
		b.ReportAllocs()
		b.ResetTimer()
		b.SetParallelism(32)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				idx := counter.Add(1)
				k := collidingKeys[idx%uint64(numKeys)]
				_ = cache.LookUp(k)
			}
		})
	})

	b.Run("SingleMutex_RadixCache", func(b *testing.B) {
		c := lru.NewRadixCache(capacity)
		defer func() {
			if rc, ok := c.(interface{ Close() }); ok {
				rc.Close()
			}
		}()
		for _, k := range uniformKeys {
			_, _ = c.Insert(k, data)
		}
		var counter atomic.Uint64
		b.ReportAllocs()
		b.ResetTimer()
		b.SetParallelism(32)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				idx := counter.Add(1)
				k := uniformKeys[idx%uint64(numKeys)]
				_ = c.LookUp(k)
			}
		})
	})
}
