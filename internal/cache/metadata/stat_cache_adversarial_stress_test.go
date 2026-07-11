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

type dummyVal struct {
	bytes uint64
}

func (d dummyVal) Size() uint64 {
	return d.bytes
}

// TestChallenger_StatCacheBucketView_Fuzz_100kOps_ZeroSizeDrift performs 60,000 randomized fuzzing iterations
// across StatCacheBucketView instances backed by ShardedRadixCache and RadixCache, verifying zero size drift
// and complete entry reclamation after full prefix sweeps.
func TestChallenger_StatCacheBucketView_Fuzz_100kOps_ZeroSizeDrift(t *testing.T) {
	const capacity = 1000
	cacheSize := uint64(cfg.AverageSizeOfPositiveStatCacheEntry+cfg.AverageSizeOfNegativeStatCacheEntry) * uint64(capacity)

	underlyingCaches := map[string]func() lru.Cache{
		"ShardedRadixCache": func() lru.Cache { return lru.NewShardedRadixCache(cacheSize) },
		"RadixCache":        func() lru.Cache { return lru.NewRadixCache(cacheSize) },
	}

	for name, newLRUFn := range underlyingCaches {
		t.Run(name, func(t *testing.T) {
			sharedLRU := newLRUFn()
			defer sharedLRU.Close()
			sc := metadata.NewStatCacheBucketView(sharedLRU, "test-bucket")
			r := rand.New(rand.NewSource(1234567))

			paths := []string{
				"folder1/file1.txt", "folder1/file2.txt", "folder1/sub/file3.txt",
				"folder2/data.csv", "root.txt", "folder3/nested/deep/file.bin",
			}
			prefixes := []string{
				"folder1/", "folder1/sub/", "folder2/", "folder3/", "",
			}

			exp := time.Now().Add(time.Hour)
			now := time.Now()

			const numOps = 30000
			for i := 0; i < numOps; i++ {
				op := r.Intn(7)
				path := paths[r.Intn(len(paths))]

				switch op {
				case 0: // Insert MinObject
					obj := &gcs.MinObject{
						Name:       path,
						Generation: int64(r.Intn(50) + 1),
						Size:       uint64(r.Intn(1000)),
					}
					sc.Insert(obj, exp)
				case 1: // AddNegativeEntry
					sc.AddNegativeEntry(path, exp)
				case 2: // InsertImplicitDir
					sc.InsertImplicitDir(path, exp)
				case 3: // InsertFolder
					sc.InsertFolder(&gcs.Folder{Name: path + "/"}, exp)
				case 4: // Erase key
					sc.Erase(path)
				case 5: // Sweep prefix
					pfx := prefixes[r.Intn(len(prefixes))]
					sc.EraseEntriesWithGivenPrefix(pfx)
				case 6: // LookUp
					_, _ = sc.LookUp(path, now)
				}
			}

			// Full sweep
			sc.EraseEntriesWithGivenPrefix("")

			for _, p := range paths {
				hit, _ := sc.LookUp(p, now)
				if hit {
					t.Fatalf("Key %s should be erased after sweep in %s", p, name)
				}
			}

			// Capacity verification
			// Inserting full capacity item into underlying cache
			evicted, err := sharedLRU.Insert("capacity_check", dummyVal{bytes: cacheSize})
			if err != nil {
				t.Fatalf("Failed to insert full capacity item after StatCacheBucketView sweep: %v", err)
			}
			if len(evicted) > 0 {
				t.Fatalf("Phantom size leak in StatCacheBucketView backed by %s! Evicted count: %d", name, len(evicted))
			}
		})
	}
}

