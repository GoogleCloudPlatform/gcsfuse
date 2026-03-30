package metadata_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

func generateObjectNames(prefixCount, itemsPerPrefix, depth int) []string {
	var keys []string
	for p := 0; p < prefixCount; p++ {
		prefix := ""
		for d := 0; d < depth; d++ {
			prefix += fmt.Sprintf("dir%d/", p*depth+d)
		}
		for i := 0; i < itemsPerPrefix; i++ {
			keys = append(keys, fmt.Sprintf("%sobj%d", prefix, i))
		}
	}
	// Shuffle to simulate random access
	rand.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})
	return keys
}

func benchmarkStatCacheInsert(b *testing.B, cache metadata.StatCache, keys []string) {
	b.ResetTimer()
	now := time.Now()
	for i := 0; i < b.N; i++ {
		name := keys[i%len(keys)]
		obj := &gcs.MinObject{Name: name}
		cache.Insert(obj, now)
	}
}

func benchmarkStatCacheLookup(b *testing.B, cache metadata.StatCache, keys []string) {
	now := time.Now()
	for _, name := range keys {
		cache.Insert(&gcs.MinObject{Name: name}, now.Add(time.Hour))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := keys[i%len(keys)]
		cache.LookUp(name, now)
	}
}

func benchmarkStatCacheErasePrefix(b *testing.B, cache metadata.StatCache, keys []string, prefixes []string) {
	now := time.Now()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		for _, name := range keys {
			cache.Insert(&gcs.MinObject{Name: name}, now)
		}
		prefix := prefixes[i%len(prefixes)]
		b.StartTimer()
		cache.EraseEntriesWithGivenPrefix(prefix)
	}
}

func runStatCacheBenchmarks(b *testing.B, name string, depth int) {
	prefixCount := 100
	itemsPerPrefix := 100
	keys := generateObjectNames(prefixCount, itemsPerPrefix, depth)

	var prefixes []string
	for p := 0; p < prefixCount; p++ {
		prefix := ""
		for d := 0; d < depth; d++ {
			prefix += fmt.Sprintf("dir%d/", p*depth+d)
		}
		prefixes = append(prefixes, prefix)
	}

	capacity := uint64(len(keys) * 1000)

	b.Run(name+"_MapLRU_Insert", func(b *testing.B) {
		benchmarkStatCacheInsert(b, metadata.NewStatCacheBucketView(lru.NewCache(capacity), ""), keys)
	})
	b.Run(name+"_RadixLRU_Insert", func(b *testing.B) {
		benchmarkStatCacheInsert(b, metadata.NewStatCacheBucketView(lru.NewRadixCache(capacity), ""), keys)
	})

	b.Run(name+"_MapLRU_Lookup", func(b *testing.B) {
		benchmarkStatCacheLookup(b, metadata.NewStatCacheBucketView(lru.NewCache(capacity), ""), keys)
	})
	b.Run(name+"_RadixLRU_Lookup", func(b *testing.B) {
		benchmarkStatCacheLookup(b, metadata.NewStatCacheBucketView(lru.NewRadixCache(capacity), ""), keys)
	})

	b.Run(name+"_MapLRU_ErasePrefix", func(b *testing.B) {
		benchmarkStatCacheErasePrefix(b, metadata.NewStatCacheBucketView(lru.NewCache(capacity), ""), keys, prefixes)
	})
	b.Run(name+"_RadixLRU_ErasePrefix", func(b *testing.B) {
		benchmarkStatCacheErasePrefix(b, metadata.NewStatCacheBucketView(lru.NewRadixCache(capacity), ""), keys, prefixes)
	})
}

func BenchmarkStatCacheCompare_Flat(b *testing.B) {
	runStatCacheBenchmarks(b, "Flat", 0)
}

func BenchmarkStatCacheCompare_Nested(b *testing.B) {
	runStatCacheBenchmarks(b, "Nested", 2)
}

func BenchmarkStatCacheCompare_DeeplyNested(b *testing.B) {
	runStatCacheBenchmarks(b, "DeeplyNested", 10)
}
