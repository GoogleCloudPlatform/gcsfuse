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

package block

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
)

// BenchmarkGenBlockPool_Get_Sequential provides a sequential baseline measurement
// of Get() and Release() cycles on GenBlockPool.
func BenchmarkGenBlockPool_Get_Sequential(b *testing.B) {
	b.ReportAllocs()

	globalMaxBlocksSem := NewBlockSemaphore(100)
	bp, err := NewGenBlockPool(1024, 10, 2, globalMaxBlocksSem, createBlock)
	if err != nil {
		b.Fatalf("failed to create GenBlockPool: %v", err)
	}
	defer func() {
		_ = bp.ClearFreeBlockChannel(true)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blk, err := bp.Get()
		if err != nil {
			b.Fatalf("Get() failed: %v", err)
		}
		bp.Release(blk)
	}
}

// BenchmarkGenBlockPool_Get_Concurrent benchmarks Get() under high concurrency using b.RunParallel
// with varying goroutine levels (10, 50, 100, 500) competing for blocks from a pool with local and global limits.
func BenchmarkGenBlockPool_Get_Concurrent(b *testing.B) {
	concurrencyLevels := []int{10, 50, 100, 500}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Goroutines_%d", concurrency), func(b *testing.B) {
			b.ReportAllocs()

			globalMaxBlocksSem := NewBlockSemaphore(100)
			bp, err := NewGenBlockPool(1024, 20, 5, globalMaxBlocksSem, createBlock)
			if err != nil {
				b.Fatalf("failed to create GenBlockPool: %v", err)
			}
			defer func() {
				_ = bp.ClearFreeBlockChannel(true)
			}()

			numCPU := runtime.GOMAXPROCS(0)
			parallelism := concurrency / numCPU
			if parallelism < 1 {
				parallelism = 1
			}
			b.SetParallelism(parallelism)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					blk, err := bp.Get()
					if err != nil {
						b.Fatalf("Get() failed: %v", err)
					}
					bp.Release(blk)
				}
			})
		})
	}
}

// BenchmarkGenBlockPool_Get_MultiPool_Concurrent benchmarks multiple GenBlockPool instances
// competing concurrently for permits from a single shared global BlockSemaphore.
func BenchmarkGenBlockPool_Get_MultiPool_Concurrent(b *testing.B) {
	concurrencyLevels := []int{10, 50, 100, 500}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Goroutines_%d", concurrency), func(b *testing.B) {
			b.ReportAllocs()

			globalMaxBlocksSem := NewBlockSemaphore(100)
			const numPools = 5
			pools := make([]*GenBlockPool[Block], numPools)
			for i := 0; i < numPools; i++ {
				p, err := NewGenBlockPool(1024, 10, 2, globalMaxBlocksSem, createBlock)
				if err != nil {
					b.Fatalf("failed to create pool %d: %v", i, err)
				}
				pools[i] = p
			}
			defer func() {
				for _, p := range pools {
					_ = p.ClearFreeBlockChannel(true)
				}
			}()

			numCPU := runtime.GOMAXPROCS(0)
			parallelism := concurrency / numCPU
			if parallelism < 1 {
				parallelism = 1
			}
			b.SetParallelism(parallelism)

			b.ResetTimer()
			var counter uint64
			b.RunParallel(func(pb *testing.PB) {
				idx := atomic.AddUint64(&counter, 1)
				pool := pools[idx%numPools]
				for pb.Next() {
					blk, err := pool.Get()
					if err != nil {
						b.Fatalf("Get() failed: %v", err)
					}
					pool.Release(blk)
				}
			})
		})
	}
}
