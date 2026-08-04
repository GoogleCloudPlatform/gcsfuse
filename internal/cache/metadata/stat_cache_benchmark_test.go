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

package metadata_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

func createBenchmarkMinObject(name string) *gcs.MinObject {
	return &gcs.MinObject{
		Name:           name,
		Size:           1024,
		Generation:     1,
		MetaGeneration: 1,
		Updated:        time.Now().UnixNano(),
	}
}

// BenchmarkStatCacheOperations measures throughput and latency of StatCache with metric tracking.
func BenchmarkStatCacheOperations(b *testing.B) {
	const cacheSizeMB = 100
	const maxEntries = 50000
	exp := time.Now().Add(time.Hour)

	b.Run("Insert_Positive", func(b *testing.B) {
		cache := lru.NewCache(cacheSizeMB * 1024 * 1024)
		sc := metadata.NewStatCacheBucketView(cache, "test-bucket", metrics.NewNoopMetrics())

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			obj := createBenchmarkMinObject(fmt.Sprintf("file_%d.txt", i%maxEntries))
			sc.Insert(obj, exp)
		}
	})

	b.Run("Insert_Negative", func(b *testing.B) {
		cache := lru.NewCache(cacheSizeMB * 1024 * 1024)
		sc := metadata.NewStatCacheBucketView(cache, "test-bucket", metrics.NewNoopMetrics())

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			sc.AddNegativeEntry(fmt.Sprintf("nonexistent_%d.txt", i%maxEntries), exp)
		}
	})

	b.Run("LookUp_Hit", func(b *testing.B) {
		cache := lru.NewCache(cacheSizeMB * 1024 * 1024)
		sc := metadata.NewStatCacheBucketView(cache, "test-bucket", metrics.NewNoopMetrics())
		for i := range maxEntries {
			sc.Insert(createBenchmarkMinObject(fmt.Sprintf("file_%d.txt", i)), exp)
		}

		now := time.Now()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			sc.LookUp(fmt.Sprintf("file_%d.txt", i%maxEntries), now)
		}
	})

	b.Run("Parallel_Insert_And_LookUp", func(b *testing.B) {
		cache := lru.NewCache(cacheSizeMB * 1024 * 1024)
		sc := metadata.NewStatCacheBucketView(cache, "test-bucket", metrics.NewNoopMetrics())

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			now := time.Now()
			for pb.Next() {
				name := fmt.Sprintf("file_%d.txt", i%maxEntries)
				if i%2 == 0 {
					sc.Insert(createBenchmarkMinObject(name), exp)
				} else {
					sc.LookUp(name, now)
				}
				i++
			}
		})
	})
}
