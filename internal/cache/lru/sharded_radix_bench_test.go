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
	"sync/atomic"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
)

type cacheConstructor struct {
	name    string
	factory func(capacity uint64) (lru.Cache, func())
}

var testImplementations = []cacheConstructor{
	{
		name: "SingleMutex_RadixCache",
		factory: func(capacity uint64) (lru.Cache, func()) {
			c := lru.NewRadixCache(capacity)
			cleanup := func() {
				if rc, ok := c.(interface{ Close() }); ok {
					rc.Close()
				}
			}
			return c, cleanup
		},
	},
	{
		name: "ShardedRadixCache_16Shards",
		factory: func(capacity uint64) (lru.Cache, func()) {
			c := lru.NewShardedRadixCacheWithCustomShards(capacity, 16)
			cleanup := func() {
				c.Close()
			}
			return c, cleanup
		},
	},
}

var parallelismLevels = []struct {
	name        string
	parallelism int
}{
	{"Parallelism_1", 1},
	{"Parallelism_4", 4},
	{"Parallelism_16", 16},
	{"Parallelism_64", 64},
}

func pregenerateKeys(n int) []string {
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		keys[i] = fmt.Sprintf("folder_%d/subfolder_%d/file_%d.txt", i%20, i%100, i)
	}
	return keys
}

// BenchmarkConcurrentRead benchmarks parallel LookUp operations across single-mutex RadixCache
// and 16-shard ShardedRadixCache under varying parallelism levels.
func BenchmarkConcurrentRead(b *testing.B) {
	const numKeys = 10000
	keys := pregenerateKeys(numKeys)
	data := testData{Value: 42, DataSize: 10}
	capacity := uint64(numKeys * 20)

	for _, impl := range testImplementations {
		b.Run(impl.name, func(b *testing.B) {
			for _, p := range parallelismLevels {
				b.Run(p.name, func(b *testing.B) {
					cache, cleanup := impl.factory(capacity)
					defer cleanup()

					for _, key := range keys {
						_, _ = cache.Insert(key, data)
					}

					var counter atomic.Uint64
					b.ReportAllocs()
					b.ResetTimer()
					b.SetParallelism(p.parallelism)

					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							idx := counter.Add(1)
							key := keys[idx%uint64(numKeys)]
							_ = cache.LookUp(key)
						}
					})
				})
			}
		})
	}
}

// BenchmarkConcurrentWrite benchmarks parallel Insert operations triggering cache eviction
// across single-mutex RadixCache and 16-shard ShardedRadixCache under varying parallelism levels.
func BenchmarkConcurrentWrite(b *testing.B) {
	const totalKeys = 10000
	const maxCapacityItems = 1000
	keys := pregenerateKeys(totalKeys)
	data := testData{Value: 100, DataSize: 10}
	capacity := uint64(maxCapacityItems * 10) // Small capacity forces continuous LRU eviction

	for _, impl := range testImplementations {
		b.Run(impl.name, func(b *testing.B) {
			for _, p := range parallelismLevels {
				b.Run(p.name, func(b *testing.B) {
					cache, cleanup := impl.factory(capacity)
					defer cleanup()

					// Pre-fill cache to capacity so evictions occur immediately
					for i := 0; i < maxCapacityItems; i++ {
						_, _ = cache.Insert(keys[i], data)
					}

					var counter atomic.Uint64
					b.ReportAllocs()
					b.ResetTimer()
					b.SetParallelism(p.parallelism)

					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							idx := counter.Add(1)
							key := keys[idx%uint64(totalKeys)]
							_, _ = cache.Insert(key, data)
						}
					})
				})
			}
		})
	}
}

// BenchmarkConcurrentMixedWorkload benchmarks parallel mixed operations (80% Lookups, 15% Inserts, 5% Erases)
// across single-mutex RadixCache and 16-shard ShardedRadixCache under varying parallelism levels.
func BenchmarkConcurrentMixedWorkload(b *testing.B) {
	const totalKeys = 10000
	const initialKeys = 2500
	keys := pregenerateKeys(totalKeys)
	data := testData{Value: 200, DataSize: 10}
	capacity := uint64(5000 * 10)

	for _, impl := range testImplementations {
		b.Run(impl.name, func(b *testing.B) {
			for _, p := range parallelismLevels {
				b.Run(p.name, func(b *testing.B) {
					cache, cleanup := impl.factory(capacity)
					defer cleanup()

					for i := 0; i < initialKeys; i++ {
						_, _ = cache.Insert(keys[i], data)
					}

					var counter atomic.Uint64
					b.ReportAllocs()
					b.ResetTimer()
					b.SetParallelism(p.parallelism)

					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							idx := counter.Add(1)
							key := keys[idx%uint64(totalKeys)]
							op := idx % 100
							if op < 80 {
								_ = cache.LookUp(key)
							} else if op < 95 {
								_, _ = cache.Insert(key, data)
							} else {
								_ = cache.Erase(key)
							}
						}
					})
				})
			}
		})
	}
}
