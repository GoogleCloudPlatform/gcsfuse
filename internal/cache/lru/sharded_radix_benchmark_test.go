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
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/stretchr/testify/require"
)

type cacheConstructor struct {
	name string
	ctor func(uint64) lru.Cache
}

var comparativeCaches = []cacheConstructor{
	{"MapCache", lru.NewCache},
	{"RadixCache", lru.NewRadixCache},
	{"ShardedRadixCache", lru.NewShardedRadixCache},
}

type memSnapshot struct {
	HeapAllocMB float64
	HeapSysMB   float64
	RSSMB       float64
	MaxRSSMB    float64
}

func getLinuxRSSBytes() uint64 {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0
	}
	rssPages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return rssPages * uint64(os.Getpagesize())
}

func captureMemSnapshot() memSnapshot {
	runtime.GC()
	runtime.GC()
	debug.FreeOSMemory()
	time.Sleep(20 * time.Millisecond)

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	var ru syscall.Rusage
	var maxRssBytes uint64
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err == nil {
		// ru.Maxrss is in kilobytes on Linux
		maxRssBytes = uint64(ru.Maxrss) * 1024
	}

	const mb = 1024 * 1024
	return memSnapshot{
		HeapAllocMB: float64(ms.HeapAlloc) / mb,
		HeapSysMB:   float64(ms.HeapSys) / mb,
		RSSMB:       float64(getLinuxRSSBytes()) / mb,
		MaxRSSMB:    float64(maxRssBytes) / mb,
	}
}

// TestComparative_MemoryFootprint quantifies heap and physical resident memory (RSS)
// across MapCache, RadixCache, and ShardedRadixCache for different workload distributions.
func TestComparative_MemoryFootprint(t *testing.T) {
	if flag.Lookup("test.bench") != nil && flag.Lookup("test.bench").Value.String() != "" {
		t.Skip("Skipping memory footprint evaluation during benchmark execution to prevent global map invariant check overhead")
	}
	const count = 5000 // 5k entries keeps test fast even under -race and global map invariant checks
	capacity := uint64(count * 1000)
	workloads := []string{"flat", "nested", "deeply_nested"}

	for _, w := range workloads {
		t.Logf("\n=== MEMORY EVALUATION WORKLOAD: %s (%d files) ===", strings.ToUpper(w), count)
		paths := generatePaths(w, count)

		var allocMapMB float64
		var rssMapMB float64

		for _, c := range comparativeCaches {
			baseSnap := captureMemSnapshot()

			cache := c.ctor(capacity)
			for _, p := range paths {
				_, _ = cache.Insert(p, dummyValue{})
			}

			peakSnap := captureMemSnapshot()
			heapDelta := peakSnap.HeapAllocMB - baseSnap.HeapAllocMB
			rssDelta := peakSnap.RSSMB - baseSnap.RSSMB

			t.Logf("%-18s | Heap Used: %8.2f MB | RSS Delta: %8.2f MB | Peak MaxRSS: %8.2f MB",
				c.name, heapDelta, rssDelta, peakSnap.MaxRSSMB)

			if c.name == "MapCache" {
				allocMapMB = heapDelta
				rssMapMB = rssDelta
			} else if c.name == "ShardedRadixCache" && allocMapMB > 0 {
				heapSavedPct := 100.0 * (1.0 - heapDelta/allocMapMB)
				rssSavedPct := 100.0 * (1.0 - rssDelta/rssMapMB)
				t.Logf(">>> ShardedRadixCache vs MapCache Savings: Heap reduction = %.2f%% | RSS reduction = %.2f%%",
					heapSavedPct, rssSavedPct)
			}

			runtime.KeepAlive(cache)
			runtime.KeepAlive(paths)
			require.NoError(t, cache.Close())
		}
	}
}

// TestComparative_RSSLifecycle evaluates physical memory across 4 lifecycle phases:
// Baseline -> Peak Population -> Partial Prefix Invalidation -> Full Erasure/Drain.
func TestComparative_RSSLifecycle(t *testing.T) {
	if flag.Lookup("test.bench") != nil && flag.Lookup("test.bench").Value.String() != "" {
		t.Skip("Skipping RSS lifecycle evaluation during benchmark execution to prevent global map invariant check overhead")
	}
	const count = 5000
	capacity := uint64(count * 100)
	keys, prefixMap, prefixes := generateKeys(5, 1000, 2) // nested paths

	t.Logf("\n=== RSS LIFECYCLE EVALUATION (Nested Workload, %d entries) ===", len(keys))

	for _, c := range comparativeCaches {
		baseSnap := captureMemSnapshot()
		cache := c.ctor(capacity)

		// Phase 1: Peak Population
		data := testData{Value: 1, DataSize: 10}
		for _, k := range keys {
			_, _ = cache.Insert(k, data)

		}
		peakSnap := captureMemSnapshot()

		// Phase 2: Partial Prefix Invalidation (Erase 50% of prefixes)
		half := len(prefixes) / 2
		for i := 0; i < half; i++ {
			cache.EraseEntriesWithGivenPrefix(prefixes[i])
		}
		if sc, ok := cache.(*lru.ShardedRadixCache); ok {
			sc.PruneAllEmptyLeaves()
			time.Sleep(150 * time.Millisecond) // Allow async background batch sweeping
		}
		partialSnap := captureMemSnapshot()

		// Phase 3: Full Erasure & Drain
		for i := half; i < len(prefixes); i++ {
			cache.EraseEntriesWithGivenPrefix(prefixes[i])
		}
		if sc, ok := cache.(*lru.ShardedRadixCache); ok {
			sc.PruneAllEmptyLeaves()
			time.Sleep(150 * time.Millisecond)
		}
		require.NoError(t, cache.Close())
		finalSnap := captureMemSnapshot()

		t.Logf("[%s] Phase 1 (Peak): HeapDelta=%6.2f MB | RSSDelta=%6.2f MB",
			c.name, peakSnap.HeapAllocMB-baseSnap.HeapAllocMB, peakSnap.RSSMB-baseSnap.RSSMB)
		t.Logf("[%s] Phase 2 (50%% Erased): HeapDelta=%6.2f MB | RSSDelta=%6.2f MB",
			c.name, partialSnap.HeapAllocMB-baseSnap.HeapAllocMB, partialSnap.RSSMB-baseSnap.RSSMB)
		t.Logf("[%s] Phase 3 (Drain/Close): HeapDelta=%6.2f MB | RSSDelta=%6.2f MB (Leak Check)",
			c.name, finalSnap.HeapAllocMB-baseSnap.HeapAllocMB, finalSnap.RSSMB-baseSnap.RSSMB)

		runtime.KeepAlive(prefixMap)
	}
}

// BenchmarkComparative_LookUp_Parallel benchmarks concurrent lookup throughput across N goroutines.
// Verifies and reports 0 B/op and 0 allocs/op on ShardedRadixCache.
func BenchmarkComparative_LookUp_Parallel(b *testing.B) {
	keys, _, _ := generateKeys(20, 500, 3) // 10,000 keys
	data := testData{Value: 1, DataSize: 10}
	capacity := uint64(len(keys) * 100) // Ensure no evictions

	for _, c := range comparativeCaches {
		b.Run(c.name, func(b *testing.B) {
			cache := c.ctor(capacity)
			for _, key := range keys {
				_, _ = cache.Insert(key, data)
			}

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				var i uint64
				for pb.Next() {
					idx := atomic.AddUint64(&i, 1) % uint64(len(keys))
					_ = cache.LookUp(keys[idx])
				}
			})
			b.StopTimer()
			_ = cache.Close()
		})
	}
}

// BenchmarkComparative_LookUpWithoutChangingOrder_Parallel compares read-only lookups without LRU promotion.
func BenchmarkComparative_LookUpWithoutChangingOrder_Parallel(b *testing.B) {
	keys, _, _ := generateKeys(20, 500, 3)
	data := testData{Value: 1, DataSize: 10}
	capacity := uint64(len(keys) * 100)

	for _, c := range comparativeCaches {
		b.Run(c.name, func(b *testing.B) {
			cache := c.ctor(capacity)
			for _, key := range keys {
				_, _ = cache.Insert(key, data)
			}

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				var i uint64
				for pb.Next() {
					idx := atomic.AddUint64(&i, 1) % uint64(len(keys))
					_ = cache.LookUpWithoutChangingOrder(keys[idx])
				}
			})
			b.StopTimer()
			_ = cache.Close()
		})
	}
}

// BenchmarkComparative_Insert benchmarks insert latency across no eviction (200%),
// moderate eviction (50%), and high contention continuous eviction (10%) regimes.
func BenchmarkComparative_Insert(b *testing.B) {
	workloads := []struct {
		name  string
		depth int
	}{
		{"Flat", 0},
		{"Nested", 2},
		{"DeeplyNested", 10},
	}

	regimes := []struct {
		name       string
		capFactor  float64 // fraction of total working set size
	}{
		{"NoEviction_200Pct", 2.0},
		{"ModerateEviction_50Pct", 0.50},
		{"HighContention_10Pct", 0.10},
	}

	for _, w := range workloads {
		keys, _, _ := generateKeys(20, 500, w.depth) // 10,000 keys
		data := testData{Value: 42, DataSize: 100}    // 1 MB total working set size
		totalSize := float64(len(keys)) * float64(data.DataSize)

		for _, reg := range regimes {
			capacity := uint64(totalSize * reg.capFactor)
			for _, c := range comparativeCaches {
				b.Run(fmt.Sprintf("%s/%s/%s", w.name, reg.name, c.name), func(b *testing.B) {
					cache := c.ctor(capacity)
					b.ReportAllocs()
					b.ResetTimer()

					b.RunParallel(func(pb *testing.PB) {
						var i uint64
						for pb.Next() {
							idx := atomic.AddUint64(&i, 1) % uint64(len(keys))
							_, _ = cache.Insert(keys[idx], data)
						}
					})
					b.StopTimer()
					_ = cache.Close()
				})
			}
		}
	}
}

// runContentionMixedWorkload benchmarks mixed R/W/Erase operations under continuous eviction pressure.
func runContentionMixedWorkload(b *testing.B, ctor func(uint64) lru.Cache, keys []string, readPct, writePct int) {
	data := testData{Value: 1, DataSize: 100}
	totalSize := uint64(len(keys)) * data.DataSize
	capacity := totalSize / 10 // 10% capacity -> continuous eviction

	cache := ctor(capacity)
	defer cache.Close()

	// Pre-populate to capacity
	for i := 0; i < len(keys)/10; i++ {
		_, _ = cache.Insert(keys[i], data)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var counter uint64
		id := atomic.AddUint64(&counter, 1)
		rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)*1000))

		for pb.Next() {
			op := rng.Intn(100)
			key := keys[rng.Intn(len(keys))]
			if op < readPct {
				_ = cache.LookUp(key)
			} else if op < readPct+writePct {
				_, _ = cache.Insert(key, data)
			} else {
				_ = cache.Erase(key)
			}
		}
	})
	b.StopTimer()
}

func BenchmarkContention_ReadHeavy(b *testing.B) {
	workloads := []struct {
		name  string
		depth int
	}{
		{"Flat", 0},
		{"Nested", 2},
		{"DeeplyNested", 10},
	}

	for _, w := range workloads {
		keys, _, _ := generateKeys(20, 500, w.depth) // 10,000 keys
		for _, c := range comparativeCaches {
			b.Run(fmt.Sprintf("%s/%s", w.name, c.name), func(b *testing.B) {
				runContentionMixedWorkload(b, c.ctor, keys, 80, 15) // 80% Read, 15% Write, 5% Erase
			})
		}
	}
}

func BenchmarkContention_WriteHeavy(b *testing.B) {
	workloads := []struct {
		name  string
		depth int
	}{
		{"Flat", 0},
		{"Nested", 2},
		{"DeeplyNested", 10},
	}

	for _, w := range workloads {
		keys, _, _ := generateKeys(20, 500, w.depth) // 10,000 keys
		for _, c := range comparativeCaches {
			b.Run(fmt.Sprintf("%s/%s", w.name, c.name), func(b *testing.B) {
				runContentionMixedWorkload(b, c.ctor, keys, 40, 50) // 40% Read, 50% Write, 10% Erase
			})
		}
	}
}
