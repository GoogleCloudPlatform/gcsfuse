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

package caching_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/timeutil"
)

// benchmarkDummyBucket is a lock-free synthetic bucket used for benchmarking.
// It prevents backend lock contention (such as fake.Bucket.mu) from masking
// caching layer throughput.
type benchmarkDummyBucket struct {
	gcs.Bucket
}

func (d *benchmarkDummyBucket) StatObject(ctx context.Context, req *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error) {
	now := time.Now()
	return &gcs.MinObject{
		Name:           req.Name,
		Size:           1024,
		Generation:     1,
		MetaGeneration: 1,
		Updated:        now,
	}, nil, nil
}

func (d *benchmarkDummyBucket) CreateObject(ctx context.Context, req *gcs.CreateObjectRequest) (*gcs.Object, error) {
	now := time.Now()
	return &gcs.Object{
		Name:           req.Name,
		Size:           1024,
		Generation:     1,
		MetaGeneration: 1,
		Updated:        now,
	}, nil
}

type bucketCacheConstructor struct {
	name string
	ctor func(uint64) lru.Cache
}

var comparativeBucketCaches = []bucketCacheConstructor{
	{"MapCache", lru.NewCache},
	{"ShardedRadixCache", lru.NewShardedRadixCache},
}

func generateBenchmarkKeys(count, depth int) []string {
	keys := make([]string, count)
	for i := 0; i < count; i++ {
		if depth == 0 {
			keys[i] = fmt.Sprintf("file_%d.txt", i)
		} else {
			keys[i] = fmt.Sprintf("dir_%d/subdir_%d/file_%d.txt", i%20, i%100, i)
		}
	}
	return keys
}

func createBenchFastStatBucket(ctor func(uint64) lru.Cache, capacity uint64) (gcs.Bucket, lru.Cache) {
	lruCache := ctor(capacity)
	statCache := metadata.NewStatCacheBucketView(lruCache, "")
	dummy := &benchmarkDummyBucket{}
	clock := &timeutil.SimulatedClock{}
	clock.SetTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	fsb := caching.NewFastStatBucket(
		time.Hour,
		statCache,
		clock,
		dummy,
		time.Hour,
		true,
		true,
	)
	return fsb, lruCache
}

// BenchmarkFastStatBucket_Concurrent_LookUp benchmarks multi-threaded lookup throughput
// across MapCache vs ShardedRadixCache on 100% cache hits.
func BenchmarkFastStatBucket_Concurrent_LookUp(b *testing.B) {
	const count = 50000
	keys := generateBenchmarkKeys(count, 2)
	capacity := uint64(count) * cfg.AverageSizeOfPositiveStatCacheEntry * 2
	ctx := context.Background()

	for _, c := range comparativeBucketCaches {
		b.Run(c.name, func(b *testing.B) {
			fsb, lruCache := createBenchFastStatBucket(c.ctor, capacity)
			defer lruCache.Close()

			// Pre-populate all keys into cache via StatObjectFromGcs
			for _, k := range keys {
				_, _, _ = fsb.StatObject(ctx, &gcs.StatObjectRequest{Name: k, ForceFetchFromGcs: true})
			}

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				var idx uint64
				for pb.Next() {
					i := atomic.AddUint64(&idx, 1) % uint64(count)
					_, _, _ = fsb.StatObject(ctx, &gcs.StatObjectRequest{Name: keys[i]})
				}
			})
			b.StopTimer()
		})
	}
}

var reqPool = sync.Pool{
	New: func() interface{} {
		return &gcs.StatObjectRequest{}
	},
}

func BenchmarkFastStatBucket_Concurrent_LookUp_Pooled(b *testing.B) {
	const count = 50000
	keys := generateBenchmarkKeys(count, 2)
	capacity := uint64(count) * cfg.AverageSizeOfPositiveStatCacheEntry * 2
	ctx := context.Background()

	for _, c := range comparativeBucketCaches {
		b.Run(c.name, func(b *testing.B) {
			fsb, lruCache := createBenchFastStatBucket(c.ctor, capacity)
			defer lruCache.Close()

			for _, k := range keys {
				_, _, _ = fsb.StatObject(ctx, &gcs.StatObjectRequest{Name: k, ForceFetchFromGcs: true})
			}

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				req := reqPool.Get().(*gcs.StatObjectRequest)
				defer reqPool.Put(req)

				i := rand.Intn(count)
				for pb.Next() {
					i = (i + 1) % count
					req.Name = keys[i]
					_, _, _ = fsb.StatObject(ctx, req)
				}
			})
			b.StopTimer()
		})
	}
}

// BenchmarkFastStatBucket_Concurrent_Insert benchmarks multi-threaded object creation
// and insertion throughput under NoEviction (200%) and HighContention (10%) regimes.
func BenchmarkFastStatBucket_Concurrent_Insert(b *testing.B) {
	const count = 50000
	workloads := []struct {
		name  string
		depth int
	}{
		{"Flat", 0},
		{"Nested", 2},
	}

	regimes := []struct {
		name      string
		capFactor float64
	}{
		{"NoEviction_200Pct", 2.0},
		{"HighContention_10Pct", 0.10},
	}

	ctx := context.Background()

	for _, w := range workloads {
		keys := generateBenchmarkKeys(count, w.depth)
		totalSize := float64(count) * float64(cfg.AverageSizeOfPositiveStatCacheEntry)

		for _, reg := range regimes {
			capacity := uint64(totalSize * reg.capFactor)
			for _, c := range comparativeBucketCaches {
				b.Run(fmt.Sprintf("%s/%s/%s", w.name, reg.name, c.name), func(b *testing.B) {
					fsb, lruCache := createBenchFastStatBucket(c.ctor, capacity)
					defer lruCache.Close()

					b.ReportAllocs()
					b.ResetTimer()
					b.RunParallel(func(pb *testing.PB) {
						var idx uint64
						for pb.Next() {
							i := atomic.AddUint64(&idx, 1) % uint64(count)
							_, _ = fsb.CreateObject(ctx, &gcs.CreateObjectRequest{Name: keys[i]})
						}
					})
					b.StopTimer()
				})
			}
		}
	}
}

// BenchmarkFastStatBucket_Concurrent_Mixed benchmarks a mixed 80% Read / 15% Write / 5% Delete
// workload under continuous eviction pressure.
func BenchmarkFastStatBucket_Concurrent_Mixed(b *testing.B) {
	const count = 50000
	keys := generateBenchmarkKeys(count, 2)
	capacity := uint64(count) * cfg.AverageSizeOfPositiveStatCacheEntry / 10 // 10% capacity
	ctx := context.Background()

	for _, c := range comparativeBucketCaches {
		b.Run(c.name, func(b *testing.B) {
			fsb, lruCache := createBenchFastStatBucket(c.ctor, capacity)
			defer lruCache.Close()

			// Pre-populate initial working set
			for i := 0; i < count/10; i++ {
				_, _, _ = fsb.StatObject(ctx, &gcs.StatObjectRequest{Name: keys[i], ForceFetchFromGcs: true})
			}

			b.ReportAllocs()
			b.ResetTimer()
			var counter uint64
			b.RunParallel(func(pb *testing.PB) {
				id := atomic.AddUint64(&counter, 1)
				rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)*1000))

				for pb.Next() {
					op := rng.Intn(100)
					key := keys[rng.Intn(count)]
					if op < 80 {
						_, _, _ = fsb.StatObject(ctx, &gcs.StatObjectRequest{Name: key})
					} else if op < 95 {
						_, _ = fsb.CreateObject(ctx, &gcs.CreateObjectRequest{Name: key})
					} else {
						_ = fsb.DeleteObject(ctx, &gcs.DeleteObjectRequest{Name: key, OnlyDeleteFromCache: true})
					}
				}
			})
			b.StopTimer()
		})
	}
}
