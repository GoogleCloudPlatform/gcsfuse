// Copyright 2024 Google LLC
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
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
)

func TestCacheTrieConcurrency(t *testing.T) {
	cache := lru.NewCacheWithIndex(uint64(10000000), true)

	const numGoroutines = 50
	const numOpsPerGoroutine = 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// We use testData struct from lru_test.go which satisfies ValueType interface
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			// Use a local random source to avoid lock contention on global rand
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(goroutineID)))

			for j := 0; j < numOpsPerGoroutine; j++ {
				// Generate random paths to simulate file system structures
				path := fmt.Sprintf("dir%d/subdir%d/file%d", r.Intn(10), r.Intn(10), r.Intn(100))

				op := r.Intn(4)
				switch op {
				case 0:
					// Insert
					val := testData{Value: int64(goroutineID*1000 + j), DataSize: 10}
					_, err := cache.Insert(path, val)
					if err != nil {
						t.Errorf("Insert failed: %v", err)
					}
				case 1:
					// LookUp
					cache.LookUp(path)
				case 2:
					// Erase
					cache.Erase(path)
				case 3:
					// EraseEntriesWithGivenPrefix
					prefix := fmt.Sprintf("dir%d/subdir%d/", r.Intn(10), r.Intn(10))
					cache.EraseEntriesWithGivenPrefix(prefix)
				}
			}
		}(i)
	}

	wg.Wait()
}
