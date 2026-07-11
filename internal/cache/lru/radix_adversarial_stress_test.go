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
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
)

// TestRadixCache_AdversarialStress performs massive high-concurrency race testing
// on Sharded/RadixCache under rapid parallel insertions, lookups, prefix invalidations,
// leaf prunings, and LRU evictions.
func TestRadixCache_AdversarialStress(t *testing.T) {
	const capacity = 10000
	const numGoroutinesPerOp = 8
	const duration = 2 * time.Second

	cache := lru.NewRadixCache(capacity)

	prefixes := []string{
		"a/",
		"a/b/",
		"a/b/c/",
		"a/b/c/d/",
		"a/x/",
		"b/c/d/",
		"deeply/nested/directory/structure/level1/level2/level3/",
		"",
	}

	keys := []string{
		"a/b/c/d/file1.txt",
		"a/b/c/d/file2.txt",
		"a/b/c/file3.txt",
		"a/b/file4.txt",
		"a/file5.txt",
		"a/x/file6.txt",
		"b/c/d/file7.txt",
		"deeply/nested/directory/structure/level1/level2/level3/file8.txt",
		"unrelated/path/file9.txt",
		"single_file.txt",
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Timer to stop after duration
	go func() {
		time.Sleep(duration)
		close(stop)
	}()

	// 1. Rapid Insertions & Evictions (High churn)
	for g := 0; g < numGoroutinesPerOp; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(id)))
			for {
				select {
				case <-stop:
					return
				default:
					key := keys[r.Intn(len(keys))]
					val := testData{Value: int64(r.Intn(10000)), DataSize: uint64(1 + r.Intn(30))}
					_, _ = cache.Insert(key, val)
				}
			}
		}(g)
	}

	// 2. Continuous Lookups (Mutating LRU order)
	for g := 0; g < numGoroutinesPerOp; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(100 + id)))
			for {
				select {
				case <-stop:
					return
				default:
					key := keys[r.Intn(len(keys))]
					_ = cache.LookUp(key)
				}
			}
		}(g)
	}

	// 3. Lookups Without Changing Order (Read-lock path)
	for g := 0; g < numGoroutinesPerOp; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(200 + id)))
			for {
				select {
				case <-stop:
					return
				default:
					key := keys[r.Intn(len(keys))]
					_ = cache.LookUpWithoutChangingOrder(key)
				}
			}
		}(g)
	}

	// 4. Prefix Invalidations (Detaching subtrees & sweeping)
	for g := 0; g < numGoroutinesPerOp; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(300 + id)))
			for {
				select {
				case <-stop:
					return
				default:
					prefix := prefixes[r.Intn(len(prefixes))]
					cache.EraseEntriesWithGivenPrefix(prefix)
				}
			}
		}(g)
	}

	// 5. Individual Leaf Prunings (Erase)
	for g := 0; g < numGoroutinesPerOp; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(400 + id)))
			for {
				select {
				case <-stop:
					return
				default:
					key := keys[r.Intn(len(keys))]
					_ = cache.Erase(key)
				}
			}
		}(g)
	}

	// 6. Incremental Size Updates (UpdateSize & UpdateWithoutChangingOrder)
	for g := 0; g < numGoroutinesPerOp; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(500 + id)))
			for {
				select {
				case <-stop:
					return
				default:
					key := keys[r.Intn(len(keys))]
					if r.Float32() < 0.5 {
						_ = cache.UpdateSize(key, uint64(r.Intn(5)))
					} else {
						val := testData{Value: int64(r.Intn(1000)), DataSize: 10}
						_ = cache.UpdateWithoutChangingOrder(key, val)
					}
				}
			}
		}(g)
	}

	wg.Wait()
}

// TestRadixCache_DeepHierarchyConcurrently tests extreme path splitting and compression.
func TestRadixCache_DeepHierarchyConcurrently(t *testing.T) {
	cache := lru.NewRadixCache(10000)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for iter := 0; iter < 200; iter++ {
				path := fmt.Sprintf("dir_%d/subdir_%d/nested_%d/file_%d.dat", iter%5, iter%10, iter%3, workerID)
				_, _ = cache.Insert(path, testData{Value: int64(workerID), DataSize: 10})
				_ = cache.LookUp(path)
				if iter%10 == 0 {
					cache.EraseEntriesWithGivenPrefix(fmt.Sprintf("dir_%d/", iter%5))
				}
			}
		}(i)
	}

	wg.Wait()
}
