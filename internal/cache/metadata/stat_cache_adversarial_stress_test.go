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

package metadata_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// TestMultiBucketStatCache_AdversarialStress tests concurrent bucket views sharing a RadixCache
// under high-throughput concurrent insertions, lookups, prefix invalidations, and negative entries.
func TestMultiBucketStatCache_AdversarialStress(t *testing.T) {
	const capacity = 500
	const numBuckets = 5
	const workersPerBucket = 4
	const duration = 2 * time.Second

	cacheSize := uint64(cfg.AverageSizeOfPositiveStatCacheEntry+cfg.AverageSizeOfNegativeStatCacheEntry) * uint64(capacity)
	sharedLRU := lru.NewShardedRadixCache(cacheSize)
	now := time.Now()
	exp := now.Add(time.Hour)

	buckets := make([]metadata.StatCache, numBuckets)
	for i := 0; i < numBuckets; i++ {
		bName := fmt.Sprintf("bucket-%d", i)
		buckets[i] = metadata.NewStatCacheBucketView(sharedLRU, bName)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	go func() {
		time.Sleep(duration)
		close(stop)
	}()

	var bName string
	for bIdx := 0; bIdx < numBuckets; bIdx++ {
		sc := buckets[bIdx]
		bName = fmt.Sprintf("bucket-%d", bIdx)

		for w := 0; w < workersPerBucket; w++ {
			wg.Add(1)
			go func(bucketID int, workerID int) {
				defer wg.Done()
				r := rand.New(rand.NewSource(int64(bucketID*100 + workerID)))

				paths := []string{
					"dir1/dir2/file1.txt",
					"dir1/dir2/file2.txt",
					"dir1/file3.txt",
					"file4.txt",
					"dir3/sub/file5.txt",
				}

				prefixes := []string{
					"dir1/",
					"dir1/dir2/",
					"dir3/",
					"",
				}

				for {
					select {
					case <-stop:
						return
					default:
						op := r.Intn(6)
						path := paths[r.Intn(len(paths))]
						prefix := prefixes[r.Intn(len(prefixes))]

						switch op {
						case 0:
							// Insert positive MinObject
							obj := &gcs.MinObject{
								Name:       path,
								Generation: int64(r.Intn(100) + 1),
								Size:       uint64(r.Intn(1000)),
							}
							sc.Insert(obj, exp)
						case 1:
							// Insert negative entry
							sc.AddNegativeEntry(path, exp)
						case 2:
							// Insert implicit dir
							sc.InsertImplicitDir(path, exp)
						case 3:
							// Lookup
							_, _ = sc.LookUp(path, now)
						case 4:
							// Erase prefix
							sc.EraseEntriesWithGivenPrefix(prefix)
						case 5:
							// Erase individual key
							sc.Erase(path)
						}
					}
				}
			}(bIdx, w)
		}
	}

	wg.Wait()
	_ = bName
}
