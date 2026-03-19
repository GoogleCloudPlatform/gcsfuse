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

func measureMemory(f func()) uint64 {
	runtime.GC()
	runtime.GC() // call twice
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	f()

	runtime.ReadMemStats(&m2)

	// Ensure we don't underflow if GC happened to collect something else
	if m2.Alloc > m1.Alloc {
		return m2.Alloc - m1.Alloc
	}
	return 0
}

func generateGCSPaths(numDirs, filesPerDir int) []string {
	var paths []string
	for i := 0; i < numDirs; i++ {
		dir := fmt.Sprintf("dataset-v%d/folder-%04d/", i/10, i)
		for j := 0; j < filesPerDir; j++ {
			paths = append(paths, fmt.Sprintf("%sfile-%04d.txt", dir, j))
		}
	}
	return paths
}

func TestStatCacheTrieMemoryUsageIsLowerThanMap(t *testing.T) {
	// Generate 100 directories, each with 100 files = 10,000 realistic keys.
	keys := generateGCSPaths(100, 100)

	// Since we are measuring memory allocation, we need to create the objects beforehand
	// to ensure they don't skew the results.
	objects := make([]*gcs.MinObject, len(keys))
	for i, key := range keys {
		objects[i] = &gcs.MinObject{
			Name:       key,
			Generation: 1,
			Size:       1024,
		}
	}

	mapMemory := measureMemory(func() {
		// Create cache with old map index
		sharedCache := lru.NewCacheWithIndex(uint64(1<<30), false)
		statCache := metadata.NewStatCacheBucketView(sharedCache, "my-bucket")

		for _, key := range keys {
			// Copy key string to accurately represent incoming new strings from gcs API
			k := fmt.Sprintf("%s", key)
			obj := &gcs.MinObject{
				Name:       k,
				Generation: 1,
				Size:       1024,
			}
			statCache.Insert(obj, time.Now().Add(time.Hour))
		}
		// Keep alive to prevent GC
		runtime.KeepAlive(sharedCache)
		runtime.KeepAlive(statCache)
	})

	trieMemory := measureMemory(func() {
		// Create cache with new trie index
		sharedCache := lru.NewCacheWithIndex(uint64(1<<30), true)
		statCache := metadata.NewStatCacheBucketView(sharedCache, "my-bucket")

		for _, key := range keys {
			// Copy key string to accurately represent incoming new strings from gcs API
			k := fmt.Sprintf("%s", key)
			obj := &gcs.MinObject{
				Name:       k,
				Generation: 1,
				Size:       1024,
			}
			statCache.Insert(obj, time.Now().Add(time.Hour))
		}
		// Keep alive to prevent GC
		runtime.KeepAlive(sharedCache)
		runtime.KeepAlive(statCache)
	})

	t.Logf("StatCache Map memory usage:  %d bytes", mapMemory)
	t.Logf("StatCache Trie memory usage: %d bytes", trieMemory)

	// Log the difference
	if trieMemory >= mapMemory {
		overhead := float64(trieMemory-mapMemory) / float64(mapMemory) * 100
		t.Logf("StatCache Trie uses %.2f%% MORE memory compared to Map in this specific scenario", overhead)
	} else {
		savings := float64(mapMemory-trieMemory) / float64(mapMemory) * 100
		t.Logf("StatCache Trie saves %.2f%% memory compared to Map", savings)
	}
}
