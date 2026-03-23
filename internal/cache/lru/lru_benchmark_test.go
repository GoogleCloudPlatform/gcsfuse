// Copyright 2023 Google LLC
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
	"runtime"
	"strconv"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
)

type benchData struct{}

func (b *benchData) Size() uint64 {
	return 1
}

func BenchmarkEraseEntriesWithGivenPrefix(b *testing.B) {
	const totalEntries = 2000000
	const entriesToDelete = 1000000
	const prefix = "delete_me_"
	const keepPrefix = "keep_me_"

	keepKeys := make([]string, totalEntries-entriesToDelete)
	deleteKeys := make([]string, entriesToDelete)

	numWorkers := runtime.NumCPU()
	var wg sync.WaitGroup

	// Pre-generate keys once across threads using fast byte buffering.
	// We do this globally before the b.N loop so no allocations happen during b.N iterations!

	// Shard string generation for keepKeys
	chunkKeep := len(keepKeys) / numWorkers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			startIdx := workerID * chunkKeep
			endIdx := startIdx + chunkKeep
			if workerID == numWorkers-1 {
				endIdx = len(keepKeys) // Handle remainder
			}
			buf := make([]byte, 0, 32)
			for j := startIdx; j < endIdx; j++ {
				buf = append(buf[:0], keepPrefix...)
				buf = strconv.AppendInt(buf, int64(j), 10)
				keepKeys[j] = string(buf)
			}
		}(w)
	}

	// Shard string generation for deleteKeys
	chunkDel := len(deleteKeys) / numWorkers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			startIdx := workerID * chunkDel
			endIdx := startIdx + chunkDel
			if workerID == numWorkers-1 {
				endIdx = len(deleteKeys) // Handle remainder
			}
			buf := make([]byte, 0, 32)
			for j := startIdx; j < endIdx; j++ {
				buf = append(buf[:0], prefix...)
				buf = strconv.AppendInt(buf, int64(j), 10)
				deleteKeys[j] = string(buf)
			}
		}(w)
	}

	wg.Wait()

	// Setup our own fast benchData blocks instead of allocating millions of `testData{}` interface implementations.
	td := &benchData{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		cache := lru.NewCache(uint64(totalEntries))

		wg.Add(2)

		go func() {
			defer wg.Done()
			// Insert entries to keep
			for j := 0; j < totalEntries-entriesToDelete; j++ {
				// Re-use pre-allocated pointers globally instead of creating new ones for GC
				cache.Insert(keepKeys[j], td)
			}
		}()

		go func() {
			defer wg.Done()
			// Insert entries to delete
			for j := 0; j < entriesToDelete; j++ {
				cache.Insert(deleteKeys[j], td)
			}
		}()

		wg.Wait()

		b.StartTimer()
		cache.EraseEntriesWithGivenPrefix(prefix)
	}
}
