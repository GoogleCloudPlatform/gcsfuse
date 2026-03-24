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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
)

func BenchmarkInsert(b *testing.B) {
	cache := lru.NewCache(10000000) // 10MB
	data := testData{Value: 1, DataSize: 10}

	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("key-%d", i)
		_, _ = cache.Insert(key, data)
	}
}

func BenchmarkLookUp(b *testing.B) {
	cache := lru.NewCache(10000000) // 10MB
	data := testData{Value: 1, DataSize: 10}

	// Pre-populate
	for i := range 10000 {
		key := fmt.Sprintf("key-%d", i)
		_, _ = cache.Insert(key, data)
	}

	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("key-%d", i%10000)
		_ = cache.LookUp(key)
	}
}

func BenchmarkErase(b *testing.B) {
	cache := lru.NewCache(10000000) // 10MB
	data := testData{Value: 1, DataSize: 10}

	b.ResetTimer()
	for i := range b.N {
		b.StopTimer()
		key := fmt.Sprintf("key-%d", i)
		_, _ = cache.Insert(key, data)
		b.StartTimer()

		_ = cache.Erase(key)
	}
}

func BenchmarkConcurrency(b *testing.B) {
	cache := lru.NewCache(50000000) // 50MB
	data := testData{Value: 1, DataSize: 10}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			op := r.Intn(100)
			key := fmt.Sprintf("key-%d", r.Intn(10000))
			if op < 30 {
				// 30% inserts
				_, _ = cache.Insert(key, data)
			} else if op < 90 {
				// 60% lookups
				_ = cache.LookUp(key)
			} else {
				// 10% erases
				_ = cache.Erase(key)
			}
		}
	})
}

/*
func BenchmarkInsert1Million(b *testing.B) {
	const numEntries = 1000000
	data := testData{Value: 1, DataSize: 10}
	cacheMaxSize := uint64(numEntries * 20) // ensure enough size so no evictions

	for range b.N {
		b.StopTimer()
		cache := lru.NewCache(cacheMaxSize)

		// Insert 1 million entries
		for j := range numEntries {
			key := fmt.Sprintf("prefix/key-%d", j)
			_, _ = cache.Insert(key, data)
		}

		b.StartTimer()

	}
}

func BenchmarkEraseEntriesWithGivenPrefix_1Million(b *testing.B) {
	const numEntries = 1000000
	data := testData{Value: 1, DataSize: 10}
	cacheMaxSize := uint64(numEntries * 20) // ensure enough size so no evictions

	for range b.N {
		b.StopTimer()
		cache := lru.NewCache(cacheMaxSize)

		// Insert 1 million entries
		for j := range numEntries {
			var key string
			// Add a specific prefix to half the keys
			if j%2 == 0 {
				key = fmt.Sprintf("prefix/key-%d", j)
			} else {
				key = fmt.Sprintf("other/key-%d", j)
			}
			_, _ = cache.Insert(key, data)
		}

		b.StartTimer()

		// Delete entries with "prefix/"
		cache.EraseEntriesWithGivenPrefix("prefix/")
	}
}
*/
