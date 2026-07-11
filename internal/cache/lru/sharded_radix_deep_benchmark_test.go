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

type deepBenchValue struct{}

func (v deepBenchValue) Size() uint64 {
	return 100
}

func generateDeepHierarchyPaths(numDirs, filesPerDir int) []string {
	paths := make([]string, 0, numDirs*filesPerDir)
	for dirIdx := range numDirs {
		catIdx := dirIdx / 100
		batchIdx := (dirIdx / 10) % 10
		grpIdx := dirIdx % 10

		dirPath := fmt.Sprintf("projects/ai-vision/dataset/train/cat_%03d/sub_00/batch_%02d/group_%02d/sub_00/set_00",
			catIdx, batchIdx, grpIdx)

		for f := range filesPerDir {
			paths = append(paths, fmt.Sprintf("%s/img_%04d.jpg", dirPath, f))
		}
	}

	// Deterministic shuffle with seed
	r := rand.New(rand.NewSource(42))
	r.Shuffle(len(paths), func(i, j int) {
		paths[i], paths[j] = paths[j], paths[i]
	})
	return paths
}

type deepMemSnapshot struct {
	HeapAllocMB float64
	HeapSysMB   float64
	RSSMB       float64
	MaxRSSMB    float64
}

func getDeepLinuxRSSBytes() uint64 {
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

func captureMemSnapshotNoGC() deepMemSnapshot {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	var ru syscall.Rusage
	var maxRssBytes uint64
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err == nil {
		maxRssBytes = uint64(ru.Maxrss) * 1024
	}

	const mb = 1024 * 1024
	return deepMemSnapshot{
		HeapAllocMB: float64(ms.HeapAlloc) / mb,
		HeapSysMB:   float64(ms.HeapSys) / mb,
		RSSMB:       float64(getDeepLinuxRSSBytes()) / mb,
		MaxRSSMB:    float64(maxRssBytes) / mb,
	}
}

func captureMemSnapshotClean() deepMemSnapshot {
	runtime.GC()
	runtime.GC()
	debug.FreeOSMemory()
	time.Sleep(20 * time.Millisecond)
	return captureMemSnapshotNoGC()
}

func TestComparativeRSS_DeepHierarchy(t *testing.T) {
	if flag.Lookup("test.bench") != nil && flag.Lookup("test.bench").Value.String() != "" {
		t.Skip("Skipping memory footprint evaluation during benchmark execution")
	}

	const numDirs = 1000
	const filesPerDir = 50
	const totalFiles = numDirs * filesPerDir // 50,000 files
	capacity := uint64(totalFiles * 1000)

	paths := generateDeepHierarchyPaths(numDirs, filesPerDir)
	require.Len(t, paths, totalFiles)

	t.Logf("\n==================================================================================")
	t.Logf("DEEP HIERARCHY COMPARATIVE RSS & HEAP FOOTPRINT EVALUATION (%d files across %d dirs, 10 levels deep)", totalFiles, numDirs)
	t.Logf("==================================================================================")

	caches := []struct {
		name string
		ctor func(uint64) lru.Cache
	}{
		{"MapCache", lru.NewCache},
		{"ShardedRadixCache", lru.NewShardedRadixCache},
	}

	results := make(map[string]struct {
		peakHeapDelta   float64
		peakRSSDelta    float64
		postGCHeapDelta float64
		postGCRSSDelta  float64
		peakMaxRSS      float64
	})

	for _, c := range caches {
		baseSnap := captureMemSnapshotClean()

		cache := c.ctor(capacity)
		for _, p := range paths {
			_, err := cache.Insert(p, deepBenchValue{})
			require.NoError(t, err)
		}

		peakSnap := captureMemSnapshotNoGC()
		postGCSnap := captureMemSnapshotClean()

		peakHeapDelta := peakSnap.HeapAllocMB - baseSnap.HeapAllocMB
		peakRSSDelta := peakSnap.RSSMB - baseSnap.RSSMB
		postGCHeapDelta := postGCSnap.HeapAllocMB - baseSnap.HeapAllocMB
		postGCRSSDelta := postGCSnap.RSSMB - baseSnap.RSSMB

		results[c.name] = struct {
			peakHeapDelta   float64
			peakRSSDelta    float64
			postGCHeapDelta float64
			postGCRSSDelta  float64
			peakMaxRSS      float64
		}{
			peakHeapDelta:   peakHeapDelta,
			peakRSSDelta:    peakRSSDelta,
			postGCHeapDelta: postGCHeapDelta,
			postGCRSSDelta:  postGCRSSDelta,
			peakMaxRSS:      postGCSnap.MaxRSSMB,
		}

		t.Logf("[%s]", c.name)
		t.Logf("  Baseline  - HeapAlloc: %8.2f MB | RSS: %8.2f MB | MaxRSS: %8.2f MB", baseSnap.HeapAllocMB, baseSnap.RSSMB, baseSnap.MaxRSSMB)
		t.Logf("  Peak      - HeapAlloc: %8.2f MB | RSS: %8.2f MB | MaxRSS: %8.2f MB | Delta Heap: %8.2f MB | Delta RSS: %8.2f MB",
			peakSnap.HeapAllocMB, peakSnap.RSSMB, peakSnap.MaxRSSMB, peakHeapDelta, peakRSSDelta)
		t.Logf("  Post-GC   - HeapAlloc: %8.2f MB | RSS: %8.2f MB | MaxRSS: %8.2f MB | Delta Heap: %8.2f MB | Delta RSS: %8.2f MB",
			postGCSnap.HeapAllocMB, postGCSnap.RSSMB, postGCSnap.MaxRSSMB, postGCHeapDelta, postGCRSSDelta)

		runtime.KeepAlive(cache)
		runtime.KeepAlive(paths)
		require.NoError(t, cache.Close())
	}

	mapRes := results["MapCache"]
	radixRes := results["ShardedRadixCache"]

	peakHeapSavings := 100.0 * (1.0 - radixRes.peakHeapDelta/mapRes.peakHeapDelta)
	postGCHeapSavings := 100.0 * (1.0 - radixRes.postGCHeapDelta/mapRes.postGCHeapDelta)
	peakRSSSavings := 100.0 * (1.0 - radixRes.peakRSSDelta/mapRes.peakRSSDelta)
	postGCRSSSavings := 100.0 * (1.0 - radixRes.postGCRSSDelta/mapRes.postGCRSSDelta)

	t.Logf("\n==================================================================================")
	t.Logf("SUMMARY SAVINGS (ShardedRadixCache vs MapCache):")
	t.Logf("  Peak Heap Reduction:    %.2f%% (Map: %.2f MB vs ShardedRadix: %.2f MB)", peakHeapSavings, mapRes.peakHeapDelta, radixRes.peakHeapDelta)
	t.Logf("  Post-GC Heap Reduction: %.2f%% (Map: %.2f MB vs ShardedRadix: %.2f MB)", postGCHeapSavings, mapRes.postGCHeapDelta, radixRes.postGCHeapDelta)
	t.Logf("  Peak RSS Reduction:     %.2f%% (Map: %.2f MB vs ShardedRadix: %.2f MB)", peakRSSSavings, mapRes.peakRSSDelta, radixRes.peakRSSDelta)
	t.Logf("  Post-GC RSS Reduction:  %.2f%% (Map: %.2f MB vs ShardedRadix: %.2f MB)", postGCRSSSavings, mapRes.postGCRSSDelta, radixRes.postGCRSSDelta)
	t.Logf("==================================================================================\n")
}

func BenchmarkDeepHierarchy(b *testing.B) {
	const numDirs = 1000
	const filesPerDir = 50
	const totalFiles = numDirs * filesPerDir // 50,000 files
	capacity := uint64(totalFiles * 1000)

	paths := generateDeepHierarchyPaths(numDirs, filesPerDir)
	val := deepBenchValue{}

	caches := []struct {
		name string
		ctor func(uint64) lru.Cache
	}{
		{"MapCache", lru.NewCache},
		{"ShardedRadixCache", lru.NewShardedRadixCache},
	}

	b.Run("Insert", func(b *testing.B) {
		for _, c := range caches {
			b.Run(c.name, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					cache := c.ctor(capacity)
					for _, p := range paths {
						_, _ = cache.Insert(p, val)
					}
					_ = cache.Close()
				}
			})
		}
	})

	b.Run("LookUp", func(b *testing.B) {
		for _, c := range caches {
			b.Run(c.name, func(b *testing.B) {
				cache := c.ctor(capacity)
				for _, p := range paths {
					_, _ = cache.Insert(p, val)
				}
				b.ReportAllocs()
				b.ResetTimer()

				b.RunParallel(func(pb *testing.PB) {
					var i uint64
					for pb.Next() {
						idx := atomic.AddUint64(&i, 1) % uint64(len(paths))
						_ = cache.LookUp(paths[idx])
					}
				})

				b.StopTimer()
				_ = cache.Close()
			})
		}
	})

	b.Run("LookUpWithoutChangingOrder", func(b *testing.B) {
		for _, c := range caches {
			b.Run(c.name, func(b *testing.B) {
				cache := c.ctor(capacity)
				for _, p := range paths {
					_, _ = cache.Insert(p, val)
				}
				b.ReportAllocs()
				b.ResetTimer()

				b.RunParallel(func(pb *testing.PB) {
					var i uint64
					for pb.Next() {
						idx := atomic.AddUint64(&i, 1) % uint64(len(paths))
						_ = cache.LookUpWithoutChangingOrder(paths[idx])
					}
				})

				b.StopTimer()
				_ = cache.Close()
			})
		}
	})
}
