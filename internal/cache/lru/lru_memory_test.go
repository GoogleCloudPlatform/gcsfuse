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
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
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
		for i := range count {
			paths = append(paths, fmt.Sprintf("file_%d.txt", i))
		}
	case "nested":
		filesPerDir := 1000
		for i := range count {
			d := i / filesPerDir
			f := i % filesPerDir
			paths = append(paths, fmt.Sprintf("dir_%04d/file_%04d.txt", d, f))
		}
	case "deeply_nested":
		batches := 100
		images := 200
		for i := range count {
			c := i / (batches * images)
			b := (i / images) % batches
			img := i % images
			paths = append(paths, fmt.Sprintf("projects/ai-vision/training/class_%04d/batch_%04d/img_%06d.jpg", c, b, img))
		}
	}

	//shuffle the paths to simulate a realistic, unpredictable workload
	//inserting keys in perfectly sorted lexicographical order can benefit or penalize certain tree/map data structures
	rand.Shuffle(len(paths), func(i, j int) {
		paths[i], paths[j] = paths[j], paths[i]
	})

	return paths
}

type dummyValue struct{}

func (d dummyValue) Size() uint64 {
	return 10
}

func TestMapCacheWorkloads(t *testing.T) {
	count := 1000000 // 1 Million files
	capacity := uint64(count * 1000)

	tests := []struct {
		name     string
		workload string
		isRadix  bool
	}{
		{"Flat_Map", "flat", false},
		{"Flat_Radix", "flat", true},
		{"Nested_Map", "nested", false},
		{"Nested_Radix", "nested", true},
		{"DeeplyNested_Map", "deeply_nested", false},
		{"DeeplyNested_Radix", "deeply_nested", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			paths := generatePaths(tc.workload, count)
			baseMem := getMemStats()
			var cache lru.Cache
			if tc.isRadix {
				cache = lru.NewRadixCache(capacity)
			} else {
				cache = lru.NewCache(capacity)
			}

			// Act
			for _, p := range paths {
				_, _ = cache.Insert(p, dummyValue{})
			}

			// Assert (or Log in this case)
			alloc := getMemStats()
			pureMem := alloc - baseMem
			runtime.KeepAlive(cache)
			runtime.KeepAlive(paths)

			cacheType := "MapLRU"
			if tc.isRadix {
				cacheType = "RadixLRU"
			}
			t.Logf("%-20s Heap Used: %10.2f MB\n", cacheType, float64(pureMem)/(1024*1024))
		})
	}
}
