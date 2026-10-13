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
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
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

// The benchmarks below cover the metadata_cache/read_count recording, which
// runs on every stat-cache read and every FUSE op.

// lookUpCase is a set of keys whose stat-cache reads have the same outcome.
type lookUpCase struct {
	name string
	keys []string
}

// setupLookUpCases returns a stat cache filled at now, with keys that hit a
// positive entry, hit a negative entry, or find no entry in it.
func setupLookUpCases(now time.Time) (metadata.StatCache, []lookUpCase) {
	const n = 10000
	cache := setupStatCache(2 * n * 500)
	positive, negative, absent := make([]string, n), make([]string, n), make([]string, n)
	for i := range n {
		positive[i] = fmt.Sprintf("object-%d", i)
		negative[i] = fmt.Sprintf("deleted-%d", i)
		absent[i] = fmt.Sprintf("absent-%d", i)
		cache.Insert(&gcs.MinObject{Name: positive[i]}, now.Add(time.Hour))
		cache.AddNegativeEntry(negative[i], now.Add(time.Hour))
	}
	return cache, []lookUpCase{{"PositiveHit", positive}, {"NegativeHit", negative}, {"Absent", absent}}
}

// BenchmarkStatCache_LookUpOutcomes compares LookUpDetail, which also returns
// the outcome for metadata_cache/read_count, with LookUp.
func BenchmarkStatCache_LookUpOutcomes(b *testing.B) {
	now := time.Now()
	cache, cases := setupLookUpCases(now)
	for _, c := range cases {
		b.Run(c.name+"/LookUp", func(b *testing.B) {
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				_, _ = cache.LookUp(c.keys[i%len(c.keys)], now)
				i++
			}
		})
		b.Run(c.name+"/LookUpDetail", func(b *testing.B) {
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				_, _, _, _ = cache.LookUpDetail(c.keys[i%len(c.keys)], now)
				i++
			}
		})
	}
}

// statCacheReader reads a stat cache as fastStatBucket.lookUp does.
type statCacheReader struct {
	mu    sync.Mutex
	cache metadata.StatCache
}

func (r *statCacheReader) lookUp(ctx context.Context, name string, now time.Time) (hit bool, m *gcs.MinObject) {
	r.mu.Lock()
	defer r.mu.Unlock()

	hit, m, entryStatus, lookupDetail := r.cache.LookUpDetail(name, now)
	metadata.RecordCacheRead(ctx, hit, entryStatus, lookupDetail)
	return
}

// BenchmarkStatCache_ReadPath measures a stat-cache read as a FUSE op makes
// it: under the bucket's mutex, recorded with the op's collector. In parallel,
// each goroutine is an op with its own collector, sharing the cache and mutex.
func BenchmarkStatCache_ReadPath(b *testing.B) {
	now := time.Now()
	cache, cases := setupLookUpCases(now)
	r := &statCacheReader{cache: cache}
	for _, c := range cases {
		b.Run(c.name+"/serial", func(b *testing.B) {
			b.ReportAllocs()
			ctx := metadata.NewCacheReads(context.Background())
			i := 0
			for b.Loop() {
				r.lookUp(ctx, c.keys[i%len(c.keys)], now)
				i++
			}
		})
		b.Run(c.name+"/parallel", func(b *testing.B) {
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				ctx := metadata.NewCacheReads(context.Background())
				i := 0
				for pb.Next() {
					r.lookUp(ctx, c.keys[i%len(c.keys)], now)
					i++
				}
			})
		})
	}
}

type depthKey int

// BenchmarkCacheReads_RecordCacheRead measures recording one read, with the
// op's collector depth contexts above the reading one, or without a collector.
func BenchmarkCacheReads_RecordCacheRead(b *testing.B) {
	for _, collector := range []bool{true, false} {
		for _, depth := range []int{0, 4, 16} {
			ctx := context.Background()
			name := "NoCollector"
			if collector {
				ctx = metadata.NewCacheReads(ctx)
				name = "Collector"
			}
			for i := range depth {
				ctx = context.WithValue(ctx, depthKey(i), i)
			}
			b.Run(fmt.Sprintf("%s/depth=%d", name, depth), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					metadata.RecordCacheRead(ctx, true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr)
				}
			})
		}
	}
}

// BenchmarkCacheReads_PerOp measures what the metric costs a FUSE op: collect
// its reads and emit one event. NoReads is the cost for ops that do not read
// the stat cache, like ReadFile.
func BenchmarkCacheReads_PerOp(b *testing.B) {
	mh, err := metrics.NewOTelMetrics(context.Background(), 1, 100)
	if err != nil {
		b.Fatal(err)
	}
	ops := []struct {
		name    string
		reads   []cacheRead
		fetched bool
	}{
		{name: "NoReads"},
		// A warm lookup of a file: "name/" is negative, "name" and its
		// attributes are positive.
		{name: "WarmHit", reads: []cacheRead{negativeHit, positiveHit, positiveHit}},
		{name: "ColdMiss", reads: []cacheRead{absent, absent, absent}, fetched: true},
	}
	for _, op := range ops {
		runOp := func() {
			cacheReads := metadata.NewCacheReads(context.Background())
			record(cacheReads, op.reads...)
			if op.fetched {
				metadata.RecordGCSFetch(cacheReads)
			}
			cacheReads.Emit(mh)
		}
		b.Run(op.name+"/serial", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				runOp()
			}
		})
		b.Run(op.name+"/parallel", func(b *testing.B) {
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					runOp()
				}
			})
		})
	}
}

// BenchmarkCacheReads_WithCacheReadsFrom measures carrying an op's collector
// to the context that getInterruptlessContext detaches from the FUSE request.
func BenchmarkCacheReads_WithCacheReadsFrom(b *testing.B) {
	cacheReads := metadata.NewCacheReads(context.Background())
	b.ReportAllocs()
	for b.Loop() {
		ctxSink = metadata.WithCacheReadsFrom(context.Background(), cacheReads)
	}
}
