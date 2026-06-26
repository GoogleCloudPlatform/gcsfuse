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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
)

func generateKeys(prefixCount, itemsPerPrefix, depth int) []string {
	var keys []string
	for p := 0; p < prefixCount; p++ {
		prefix := ""
		if depth == 0 {
			prefix = fmt.Sprintf("prefix%d-", p)
		} else {
			for d := 0; d < depth; d++ {
				prefix += fmt.Sprintf("dir%d/", p*depth+d)
			}
		}
		for i := 0; i < itemsPerPrefix; i++ {
			keys = append(keys, fmt.Sprintf("%sfile%d", prefix, i))
		}
	}
	// Shuffle to simulate random access
	rand.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})
	return keys
}

func benchmarkInsert(b *testing.B, cache *lru.Cache, keys []string) {
	data := testData{Value: 1, DataSize: 10}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]
		_, _ = cache.Insert(key, data)
	}
}

func benchmarkLookup(b *testing.B, cache *lru.Cache, keys []string) {
	data := testData{Value: 1, DataSize: 10}
	for _, key := range keys {
		_, _ = cache.Insert(key, data)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]
		_ = cache.LookUp(key)
	}
}

func benchmarkErasePrefix(b *testing.B, cache *lru.Cache, prefixMap map[string][]string, prefixes []string) {
	data := testData{Value: 1, DataSize: 10}

	for _, keysInPrefix := range prefixMap {
		for _, key := range keysInPrefix {
			_, _ = cache.Insert(key, data)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prefix := prefixes[i%len(prefixes)]

		b.StartTimer()
		cache.EraseEntriesWithGivenPrefix(prefix)
		//reset the timer so that we only measure erasure time
		b.StopTimer()

		//restore the map
		keysToRestore := prefixMap[prefix]
		for _, key := range keysToRestore {
			_, _ = cache.Insert(key, data)
		}
	}
}

func runBenchmarks(b *testing.B, name string, depth int) {
	prefixCount := 1000
	itemsPerPrefix := 1000
	keys := generateKeys(prefixCount, itemsPerPrefix, depth)

	// Pre-calculate which keys belong to which prefix so we can
	// quickly repair the cache in benchmarkErasePrefix
	var prefixes []string
	prefixMap := make(map[string][]string)

	for p := 0; p < prefixCount; p++ {
		prefix := ""
		if depth == 0 {
			prefix = fmt.Sprintf("prefix%d-", p)
		} else {
			for d := 0; d < depth; d++ {
				prefix += fmt.Sprintf("dir%d/", p*depth+d)
			}
		}
		prefixes = append(prefixes, prefix)

		var keysForPrefix []string
		for i := 0; i < itemsPerPrefix; i++ {
			keysForPrefix = append(keysForPrefix, fmt.Sprintf("%sfile%d", prefix, i))
		}
		prefixMap[prefix] = keysForPrefix
	}

	capacity := uint64(len(keys) * 100)

	b.Run(name+"_MapLRU_Insert", func(b *testing.B) {
		benchmarkInsert(b, lru.NewCache(capacity), keys)
	})

	b.Run(name+"_MapLRU_Lookup", func(b *testing.B) {
		benchmarkLookup(b, lru.NewCache(capacity), keys)
	})

	b.Run(name+"_MapLRU_ErasePrefix", func(b *testing.B) {
		benchmarkErasePrefix(b, lru.NewCache(capacity), prefixMap, prefixes)
	})
}

func BenchmarkLRU_Flat(b *testing.B) {
	runBenchmarks(b, "Flat", 0)
}

func BenchmarkLRU_Nested(b *testing.B) {
	runBenchmarks(b, "Nested", 2)
}

func BenchmarkLRU_DeeplyNested(b *testing.B) {
	runBenchmarks(b, "DeeplyNested", 10)
}
