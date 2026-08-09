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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

func generateKeys(count int) []string {
	keys := make([]string, count)
	for i := 0; i < count; i++ {
		keys[i] = fmt.Sprintf("object-%d", i)
	}
	return keys
}

func setupStatCache(capacity uint64) metadata.StatCache {
	// 1000 items * approx 120 bytes each = ~120,000 bytes capacity
	lruCache := lru.NewCache(capacity)
	return metadata.NewStatCacheBucketView(lruCache, "")
}

func BenchmarkStatCache_Insert(b *testing.B) {
	keys := generateKeys(10000)
	cache := setupStatCache(uint64(len(keys) * 500))
	now := time.Now()

	objects := make([]*gcs.MinObject, len(keys))
	for i, key := range keys {
		objects[i] = &gcs.MinObject{Name: key}
	}

	i := 0
	for b.Loop() {
		m := objects[i%len(objects)]
		cache.Insert(m, now)
		i++
	}
}

func BenchmarkStatCache_AddNegativeEntry(b *testing.B) {
	keys := generateKeys(10000)
	cache := setupStatCache(uint64(len(keys) * 500))
	now := time.Now()

	i := 0
	for b.Loop() {
		key := keys[i%len(keys)]
		cache.AddNegativeEntry(key, now)
		i++
	}
}

func BenchmarkStatCache_LookUp(b *testing.B) {
	keys := generateKeys(10000)
	cache := setupStatCache(uint64(len(keys) * 500))
	now := time.Now()

	// Pre-fill the cache with unexpired entries.
	for _, key := range keys {
		m := &gcs.MinObject{Name: key}
		cache.Insert(m, now.Add(time.Hour))
	}

	i := 0
	for b.Loop() {
		key := keys[i%len(keys)]
		_, _ = cache.LookUp(key, now)
		i++
	}
}
