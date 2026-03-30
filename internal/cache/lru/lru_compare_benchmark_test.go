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
		for d := 0; d < depth; d++ {
			prefix += fmt.Sprintf("dir%d/", p*depth+d)
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

func benchmarkInsert(b *testing.B, cache lru.Cache, keys []string) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]
		_, _ = cache.Insert(key, stringValue{key})
	}
}

func benchmarkLookup(b *testing.B, cache lru.Cache, keys []string) {
	for _, key := range keys {
		_, _ = cache.Insert(key, stringValue{key})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]
		_ = cache.LookUp(key)
	}
}

func benchmarkErasePrefix(b *testing.B, cache lru.Cache, keys []string, prefixes []string) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		for _, key := range keys {
			_, _ = cache.Insert(key, stringValue{key})
		}
		prefix := prefixes[i%len(prefixes)]
		b.StartTimer()
		cache.EraseEntriesWithGivenPrefix(prefix)
	}
}

func runBenchmarks(b *testing.B, name string, depth int) {
	prefixCount := 100
	itemsPerPrefix := 100
	keys := generateKeys(prefixCount, itemsPerPrefix, depth)

	var prefixes []string
	for p := 0; p < prefixCount; p++ {
		prefix := ""
		for d := 0; d < depth; d++ {
			prefix += fmt.Sprintf("dir%d/", p*depth+d)
		}
		prefixes = append(prefixes, prefix)
	}

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
		benchmarkErasePrefix(b, lru.NewCache(capacity), keys, prefixes)
	})
	b.Run(name+"_RadixLRU_ErasePrefix", func(b *testing.B) {
		benchmarkErasePrefix(b, lru.NewRadixCache(capacity), keys, prefixes)
	})
}

func BenchmarkLRUCompare_Flat(b *testing.B) {
	runBenchmarks(b, "Flat", 0)
}

func BenchmarkLRUCompare_Nested(b *testing.B) {
	runBenchmarks(b, "Nested", 2)
}

func BenchmarkLRUCompare_DeeplyNested(b *testing.B) {
	runBenchmarks(b, "DeeplyNested", 10)
}
