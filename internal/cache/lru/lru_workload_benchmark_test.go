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

func generateKeys(prefixCount, itemsPerPrefix, depth int) (keys []string, prefixMap map[string][]string, prefixes []string) {
	prefixMap = make(map[string][]string)

	for p := range prefixCount {
		prefix := ""
		if depth == 0 {
			prefix = fmt.Sprintf("prefix%d-", p)
		} else {
			for d := range depth {
				prefix += fmt.Sprintf("dir%d/", p*depth+d)
			}
		}
		prefixes = append(prefixes, prefix)

		var keysForPrefix []string
		for i := range itemsPerPrefix {
			key := fmt.Sprintf("%sfile%d", prefix, i)
			keysForPrefix = append(keysForPrefix, key)
			keys = append(keys, key)
		}
		prefixMap[prefix] = keysForPrefix
	}

	//shuffle the paths to simulate a realistic, unpredictable workload
	//inserting keys in perfectly sorted lexicographical order can benefit or penalize certain tree/map data structures
	rand.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})
	return keys, prefixMap, prefixes
}

func benchmarkInsert(b *testing.B, cache lru.Cache, keys []string) {
	data := testData{Value: 1, DataSize: 10}

	i := 0
	for b.Loop() {
		key := keys[i%len(keys)]
		_, _ = cache.Insert(key, data)
		i++
	}
}

func benchmarkLookup(b *testing.B, cache lru.Cache, keys []string) {
	data := testData{Value: 1, DataSize: 10}
	for _, key := range keys {
		_, _ = cache.Insert(key, data)
	}

	i := 0
	for b.Loop() {
		key := keys[i%len(keys)]
		_ = cache.LookUp(key)
		i++
	}
}

func benchmarkErasePrefix(b *testing.B, cache lru.Cache, prefixMap map[string][]string, prefixes []string) {
	data := testData{Value: 1, DataSize: 10}

	for _, keysInPrefix := range prefixMap {
		for _, key := range keysInPrefix {
			_, _ = cache.Insert(key, data)
		}
	}

	i := 0
	for b.Loop() {
		prefix := prefixes[i%len(prefixes)]
		cache.EraseEntriesWithGivenPrefix(prefix)

		// UNTIMED: Pause clock, restore erased keys, and restart timer for next iteration
		b.StopTimer()
		for _, key := range prefixMap[prefix] {
			_, _ = cache.Insert(key, data)
		}
		i++
		b.StartTimer() // Timer MUST be running when b.Loop() evaluates next!
	}
}

func runBenchmarks(b *testing.B, name string, depth int) {
	prefixCount := 1000
	itemsPerPrefix := 1000
	keys, prefixMap, prefixes := generateKeys(prefixCount, itemsPerPrefix, depth)

	capacity := uint64(len(keys) * 100)

	b.Run(name+"_MapLRU_Insert", func(b *testing.B) {
		benchmarkInsert(b, lru.NewCache(capacity), keys)
	})
	b.Run(name+"_RadixLRU_Insert", func(b *testing.B) {
		benchmarkInsert(b, lru.NewRadixCache(capacity), keys)
	})

	b.Run(name+"_MapLRU_Lookup", func(b *testing.B) {
		benchmarkLookup(b, lru.NewCache(capacity), keys)
	})
	b.Run(name+"_RadixLRU_Lookup", func(b *testing.B) {
		benchmarkLookup(b, lru.NewRadixCache(capacity), keys)
	})

	b.Run(name+"_MapLRU_ErasePrefix", func(b *testing.B) {
		benchmarkErasePrefix(b, lru.NewCache(capacity), prefixMap, prefixes)
	})
	b.Run(name+"_RadixLRU_ErasePrefix", func(b *testing.B) {
		benchmarkErasePrefix(b, lru.NewRadixCache(capacity), prefixMap, prefixes)
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
