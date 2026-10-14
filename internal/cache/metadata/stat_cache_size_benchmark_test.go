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
	"math/rand"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

// The benchmarks below cover every way an entry enters or leaves the stat
// cache, i.e. every path that records metadata_cache/size.

// runWithHandles runs bench once per way of recording metadata_cache/size:
// "none" builds the shared cache without a metric handle, hence without a size
// observer; "noop" with the handle used when metrics are disabled; "otel" with
// the OTel handle. Each run gets a new stat cache of the given capacity.
func runWithHandles(b *testing.B, capacity uint64, bench func(b *testing.B, sc metadata.StatCache)) {
	otelHandle, err := metrics.NewOTelMetrics(context.Background(), 1, 100)
	if err != nil {
		b.Fatal(err)
	}
	handles := []struct {
		name string
		mh   metrics.MetricHandle
	}{
		{"none", nil},
		{"noop", metrics.NewNoopMetrics()},
		{"otel", otelHandle},
	}
	for _, h := range handles {
		b.Run("handle="+h.name, func(b *testing.B) {
			b.ReportAllocs()
			bench(b, metadata.NewStatCacheBucketView(metadata.NewSharedStatCache(capacity, h.mh), ""))
		})
	}
}

func sizeBenchKeys(prefix string, n int) []string {
	keys := make([]string, n)
	for i := range n {
		keys[i] = fmt.Sprintf("%sobject-%d", prefix, i)
	}
	return keys
}

func sizeBenchObjects(keys []string) []*gcs.MinObject {
	objs := make([]*gcs.MinObject, len(keys))
	for i, k := range keys {
		objs[i] = &gcs.MinObject{Name: k, Size: 1024, Generation: 1, MetaGeneration: 1}
	}
	return objs
}

// A positive entry replaced by a positive entry: -old, +new.
func BenchmarkSizeMetric_OverwritePositive(b *testing.B) {
	objs := sizeBenchObjects(sizeBenchKeys("", 10000))
	exp := time.Now().Add(time.Hour)
	runWithHandles(b, 64<<20, func(b *testing.B, sc metadata.StatCache) {
		for _, o := range objs {
			sc.Insert(o, exp)
		}
		i := 0
		for b.Loop() {
			sc.Insert(objs[i%len(objs)], exp)
			i++
		}
	})
}

// A negative entry replaced by a negative entry: -old, +new.
func BenchmarkSizeMetric_OverwriteNegative(b *testing.B) {
	keys := sizeBenchKeys("", 10000)
	exp := time.Now().Add(time.Hour)
	runWithHandles(b, 64<<20, func(b *testing.B, sc metadata.StatCache) {
		i := 0
		for b.Loop() {
			sc.AddNegativeEntry(keys[i%len(keys)], exp)
			i++
		}
	})
}

// The same key flips positive -> negative -> positive: its bytes move between
// the two entry_status series.
func BenchmarkSizeMetric_FlipStatus(b *testing.B) {
	keys := sizeBenchKeys("", 10000)
	objs := sizeBenchObjects(keys)
	exp := time.Now().Add(time.Hour)
	runWithHandles(b, 64<<20, func(b *testing.B, sc metadata.StatCache) {
		i := 0
		for b.Loop() {
			j := i % len(objs)
			sc.Insert(objs[j], exp)
			sc.AddNegativeEntry(keys[j], exp)
			i++
		}
	})
}

// A full cache: every insert of a new key evicts the least recently used entry.
func BenchmarkSizeMetric_InsertWithEviction(b *testing.B) {
	objs := sizeBenchObjects(sizeBenchKeys("", 100000))
	exp := time.Now().Add(time.Hour)
	// Room for ~1000 positive entries, so that each insert evicts one.
	runWithHandles(b, 1000*1500, func(b *testing.B, sc metadata.StatCache) {
		i := 0
		for b.Loop() {
			sc.Insert(objs[i%len(objs)], exp)
			i++
		}
	})
}

// An insert then an invalidation, as on object updates and deletes: +size,
// -size.
func BenchmarkSizeMetric_InsertErase(b *testing.B) {
	keys := sizeBenchKeys("", 10000)
	objs := sizeBenchObjects(keys)
	exp := time.Now().Add(time.Hour)
	runWithHandles(b, 64<<20, func(b *testing.B, sc metadata.StatCache) {
		i := 0
		for b.Loop() {
			j := i % len(objs)
			sc.Insert(objs[j], exp)
			sc.Erase(keys[j])
			i++
		}
	})
}

// A look-up that finds its entry expired erases it: +size, -size.
func BenchmarkSizeMetric_LookUpExpired(b *testing.B) {
	keys := sizeBenchKeys("", 10000)
	objs := sizeBenchObjects(keys)
	now := time.Now()
	past := now.Add(-time.Second)
	runWithHandles(b, 64<<20, func(b *testing.B, sc metadata.StatCache) {
		i := 0
		for b.Loop() {
			j := i % len(objs)
			sc.Insert(objs[j], past)
			_, _ = sc.LookUp(keys[j], now)
			i++
		}
	})
}

// Look-up hits change nothing in the cache, so they record nothing.
func BenchmarkSizeMetric_LookUpHit(b *testing.B) {
	keys := sizeBenchKeys("", 10000)
	objs := sizeBenchObjects(keys)
	now := time.Now()
	runWithHandles(b, 64<<20, func(b *testing.B, sc metadata.StatCache) {
		for _, o := range objs {
			sc.Insert(o, now.Add(time.Hour))
		}
		i := 0
		for b.Loop() {
			_, _ = sc.LookUp(keys[i%len(keys)], now)
			i++
		}
	})
}

// A folder rename or delete erases a whole subtree: -size per entry.
func BenchmarkSizeMetric_EraseEntriesWithGivenPrefix(b *testing.B) {
	const n = 10000
	inside := sizeBenchObjects(sizeBenchKeys("dir/", n))
	outside := sizeBenchObjects(sizeBenchKeys("other/", n))
	exp := time.Now().Add(time.Hour)
	runWithHandles(b, 64<<20, func(b *testing.B, sc metadata.StatCache) {
		for _, o := range outside {
			sc.Insert(o, exp)
		}
		for b.Loop() {
			b.StopTimer()
			for _, o := range inside {
				sc.Insert(o, exp)
			}
			b.StartTimer()
			sc.EraseEntriesWithGivenPrefix("dir/")
		}
	})
}

// A concurrent mixed workload on one shared cache: 25% positive inserts, 10%
// negative inserts, 55% look-ups and 10% erases.
func BenchmarkSizeMetric_ParallelMixed(b *testing.B) {
	keys := sizeBenchKeys("", 20000)
	objs := sizeBenchObjects(keys)
	now := time.Now()
	exp := now.Add(time.Hour)
	// Room for ~10000 positive entries, so that inserts also evict.
	runWithHandles(b, 10000*1500, func(b *testing.B, sc metadata.StatCache) {
		b.RunParallel(func(pb *testing.PB) {
			r := rand.New(rand.NewSource(rand.Int63()))
			for pb.Next() {
				j := r.Intn(len(keys))
				switch op := r.Intn(100); {
				case op < 25:
					sc.Insert(objs[j], exp)
				case op < 35:
					sc.AddNegativeEntry(keys[j], exp)
				case op < 90:
					_, _ = sc.LookUp(keys[j], now)
				default:
					sc.Erase(keys[j])
				}
			}
		})
	})
}
