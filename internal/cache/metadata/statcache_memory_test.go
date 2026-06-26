// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package metadata_test

import (
	"fmt"
	"math/rand"
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

func getMemStats() uint64 {
	runtime.GC()
	runtime.GC()
	debug.FreeOSMemory()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

func generatePaths(workload string, count int) []string {
	var paths []string

	switch workload {
	case "flat":
		for i := 0; i < count; i++ {
			paths = append(paths, fmt.Sprintf("file_%d.txt", i))
		}
	case "nested":
		// e.g. 1000 dirs, 1000 files each
		for d := 0; d < 1000; d++ {
			for f := 0; f < 1000; f++ {
				paths = append(paths, fmt.Sprintf("dir_%04d/file_%04d.txt", d, f))
			}
		}
	case "deeply_nested":
		// e.g. 50 classes, 100 batches, 200 images = 1,000,000 keys
		for c := 0; c < 50; c++ {
			for b := 0; b < 100; b++ {
				for i := 0; i < 200; i++ {
					paths = append(paths, fmt.Sprintf("projects/ai-vision/training/class_%04d/batch_%04d/img_%06d.jpg", c, b, i))
				}
			}
		}
	}

	rand.Shuffle(len(paths), func(i, j int) {
		paths[i], paths[j] = paths[j], paths[i]
	})

	return paths
}

func TestStatCacheMemoryWorkloads(t *testing.T) {
	count := 1000000                   // 1 Million files
	capacity := uint64(count * 100000) // high capacity to avoid eviction
	workloads := []string{"flat", "nested", "deeply_nested"}
	expiration := time.Now().Add(time.Hour)

	for _, w := range workloads {
		t.Logf("=== WORKLOAD: %s (%d files) ===", w, count)

		// 1. StatCache
		paths := generatePaths(w, count)
		baseMem := getMemStats()

		sharedCache := lru.NewCache(capacity)
		statCache := metadata.NewStatCacheBucketView(sharedCache, "")

		for _, p := range paths {
			statCache.Insert(&gcs.MinObject{Name: p}, expiration)
		}

		alloc := getMemStats()
		pureMem := alloc - baseMem
		runtime.KeepAlive(statCache)
		statCache = nil

		t.Logf("%-20s Heap Used: %10.2f MB", "StatCache", float64(pureMem)/(1024*1024))
		t.Logf("")
	}
}
