package metadata_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
)

func getMemStats() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

func populateCache(cache metadata.StatCache, keys []string) {
	now := time.Now()
	for _, key := range keys {
		cache.Insert(&gcs.MinObject{Name: key}, now)
	}
}

func TestStatCache_MemoryUsage(t *testing.T) {
	// Generate keys with common prefixes (deeply nested)
	prefixCount := 100
	itemsPerPrefix := 100
	depth := 10
	keys := generateObjectNames(prefixCount, itemsPerPrefix, depth)

	capacity := uint64(len(keys) * 1000)

	// Baseline memory
	baseMem := getMemStats()

	// 1. Measure MapLRU Memory
	mapCache := metadata.NewStatCacheBucketView(lru.NewCache(capacity), "")
	populateCache(mapCache, keys)

	mapMemUsed := getMemStats() - baseMem
	runtime.KeepAlive(mapCache)

	// Reset memory
	mapCache = nil
	baseMem = getMemStats()

	// 2. Measure RadixLRU Memory
	radixCache := metadata.NewStatCacheBucketView(lru.NewRadixCache(capacity), "")
	populateCache(radixCache, keys)

	radixMemUsed := getMemStats() - baseMem
	runtime.KeepAlive(radixCache)

	// Logging to output for manual inspection
	t.Logf("MapLRU   Memory Used: %d bytes\n", mapMemUsed)
	t.Logf("RadixLRU Memory Used: %d bytes\n", radixMemUsed)

	// Ensure that RadixLRU uses less or equal memory for highly-nested prefixes.
	// We add a minor buffer to account for Go runtime allocation variability.
	assert.LessOrEqual(t, float64(radixMemUsed), float64(mapMemUsed)*1.10, "Radix cache should be more or equally memory efficient for shared prefixes.")
}
