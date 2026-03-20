// Copyright 2024 Google LLC
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

package metadata_test

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// measureMemStats runs the given function and logs the memory consumed.
func measureMemStats(t *testing.T, name string, count int, f func(cache metadata.StatCache, expiration time.Time)) {
	var m1, m2 runtime.MemStats

	// Run GC to get a clean baseline
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// We use a large cache capacity so nothing gets evicted during the test.
	// Capacity is in bytes, so we give it a very large size (e.g., 10GB).
	capacity := uint64(10 * 1024 * 1024 * 1024)
	cache := lru.NewCache(capacity)
	statCache := metadata.NewStatCacheBucketView(cache, "")
	expiration := time.Now().Add(time.Hour)

	// Run the population function
	f(statCache, expiration)

	// Run GC again to ensure only reachable objects are counted
	runtime.GC()
	runtime.ReadMemStats(&m2)

	consumed := int64(m2.Alloc) - int64(m1.Alloc)
	var consumedMB, bytesPerEntry float64
	if consumed > 0 {
		consumedMB = float64(consumed) / (1024 * 1024)
		bytesPerEntry = float64(consumed) / float64(count)
	}

	t.Logf("Test: %s, Entries: %d, Memory Consumed: %.2f MB, Bytes per Entry: %.2f bytes", name, count, consumedMB, bytesPerEntry)

	// Ensure the cache is not garbage-collected before we read the final memory stats.
	runtime.KeepAlive(cache)
}

func generatePath(nestingLevel int, index int) string {
	if nestingLevel == 0 {
		return fmt.Sprintf("file_%d", index)
	}

	path := ""
	for i := 0; i < nestingLevel; i++ {
		path += fmt.Sprintf("dir_%d/", index)
	}
	path += fmt.Sprintf("file_%d", index)
	return path
}

func TestStatCacheMemory_Flat(t *testing.T) {
	counts := []int{10000, 100000, 1000000}

	for _, count := range counts {
		t.Run(fmt.Sprintf("Count_%d", count), func(t *testing.T) {
			measureMemStats(t, "Flat", count, func(cache metadata.StatCache, expiration time.Time) {
				for i := 0; i < count; i++ {
					name := fmt.Sprintf("file_%d", i)
					m := &gcs.MinObject{Name: name, Generation: 1, MetaGeneration: 1}
					cache.Insert(m, expiration)
				}
			})
		})
	}
}

func TestStatCacheMemory_Nested5(t *testing.T) {
	counts := []int{10000, 100000, 1000000}

	for _, count := range counts {
		t.Run(fmt.Sprintf("Count_%d", count), func(t *testing.T) {
			measureMemStats(t, "Nested_5", count, func(cache metadata.StatCache, expiration time.Time) {
				for i := 0; i < count; i++ {
					name := generatePath(5, i)
					m := &gcs.MinObject{Name: name, Generation: 1, MetaGeneration: 1}
					cache.Insert(m, expiration)
				}
			})
		})
	}
}

func TestStatCacheMemory_Nested20(t *testing.T) {
	counts := []int{10000, 100000, 1000000}

	for _, count := range counts {
		t.Run(fmt.Sprintf("Count_%d", count), func(t *testing.T) {
			measureMemStats(t, "Nested_20", count, func(cache metadata.StatCache, expiration time.Time) {
				for i := 0; i < count; i++ {
					name := generatePath(20, i)
					m := &gcs.MinObject{Name: name, Generation: 1, MetaGeneration: 1}
					cache.Insert(m, expiration)
				}
			})
		})
	}
}

func TestStatCacheMemory_Folders_Flat(t *testing.T) {
	counts := []int{10000, 100000, 1000000}

	for _, count := range counts {
		t.Run(fmt.Sprintf("Count_%d", count), func(t *testing.T) {
			measureMemStats(t, "Folders_Flat", count, func(cache metadata.StatCache, expiration time.Time) {
				for i := 0; i < count; i++ {
					name := fmt.Sprintf("dir_%d/", i)
					f := &gcs.Folder{Name: name}
					cache.InsertFolder(f, expiration)
				}
			})
		})
	}
}

func TestStatCacheMemory_ImplicitDirs_Flat(t *testing.T) {
	counts := []int{10000, 100000, 1000000}

	for _, count := range counts {
		t.Run(fmt.Sprintf("Count_%d", count), func(t *testing.T) {
			measureMemStats(t, "ImplicitDirs_Flat", count, func(cache metadata.StatCache, expiration time.Time) {
				for i := 0; i < count; i++ {
					name := fmt.Sprintf("dir_%d/", i)
					cache.InsertImplicitDir(name, expiration)
				}
			})
		})
	}
}

func TestStatCacheMemory_NegativeEntries_Flat(t *testing.T) {
	counts := []int{10000, 100000, 1000000}

	for _, count := range counts {
		t.Run(fmt.Sprintf("Count_%d", count), func(t *testing.T) {
			measureMemStats(t, "NegativeEntries_Flat", count, func(cache metadata.StatCache, expiration time.Time) {
				for i := 0; i < count; i++ {
					name := fmt.Sprintf("file_%d", i)
					cache.AddNegativeEntry(name, expiration)
				}
			})
		})
	}
}
