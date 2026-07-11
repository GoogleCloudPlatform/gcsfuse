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
	"sync/atomic"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
)

// TestStatCache_ExtremeConcurrencyStress stress-tests statCacheBucketView under heavy multi-goroutine concurrency
// (100 goroutines concurrently executing inserts, lookups, negative entries, folder ops, prefix erasures).
func TestStatCache_ExtremeConcurrencyStress(t *testing.T) {
	sharedCache := lru.NewShardedRadixCache(100000)
	defer sharedCache.Close()

	sc1 := metadata.NewStatCacheBucketView(sharedCache, "bucket1")
	sc2 := metadata.NewStatCacheBucketView(sharedCache, "bucket2")

	const numGoroutines = 100
	const opsPerGoroutine = 1000

	keys := []string{
		"file1.txt", "file2.txt", "dir1/fileA.txt", "dir1/fileB.txt",
		"dir2/sub/fileC.txt", "dir2/sub/fileD.txt", "folder1/", "folder2/",
	}

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id)))

			targetSC := sc1
			if id%2 == 1 {
				targetSC = sc2
			}

			for op := 0; op < opsPerGoroutine; op++ {
				key := keys[rng.Intn(len(keys))]
				crcVal := uint32(rng.Uint32())
				now := time.Now()

				switch rng.Intn(7) {
				case 0: // Insert MinObject
					mo := &gcs.MinObject{
						Name:           key,
						Size:           uint64(rng.Intn(10000)),
						Generation:     int64(rng.Intn(100) + 1),
						MetaGeneration: 1,
						Updated:        now,
						CRC32C:         &crcVal,
					}
					targetSC.Insert(mo, now.Add(time.Minute))

				case 1: // LookUp MinObject
					hit, mo := targetSC.LookUp(key, now)
					if hit && mo.Name != "" {
						assert.Equal(t, key, mo.Name)
					}

				case 2: // AddNegativeEntry
					targetSC.AddNegativeEntry(key, now.Add(time.Minute))

				case 3: // Insert Folder
					f := &gcs.Folder{
						Name:       key,
						UpdateTime: now,
					}
					targetSC.InsertFolder(f, now.Add(time.Minute))

				case 4: // LookUp Folder
					hit, f := targetSC.LookUpFolder(key, now)
					if hit && f.Name != "" {
						assert.Equal(t, key, f.Name)
					}

				case 5: // Erase
					targetSC.Erase(key)

				case 6: // EraseEntriesWithGivenPrefix
					targetSC.EraseEntriesWithGivenPrefix("dir1/")
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestStatCache_ScalarValueSemantics_ThreadIsolated demonstrates that scalar fields on returned value structs
// (gcs.MinObject and gcs.Folder) are completely thread-isolated under extreme 100-goroutine concurrency with -race.
func TestStatCache_ScalarValueSemantics_ThreadIsolated(t *testing.T) {
	sharedCache := lru.NewShardedRadixCache(100000)
	defer sharedCache.Close()

	sc := metadata.NewStatCacheBucketView(sharedCache, "test-bucket")
	now := time.Now()

	crcInitial := uint32(12345)
	initialObj := &gcs.MinObject{
		Name:           "shared-file.txt",
		Size:           500,
		Generation:     1,
		MetaGeneration: 1,
		Updated:        now,
		CRC32C:         &crcInitial,
	}
	sc.Insert(initialObj, now.Add(time.Hour))

	initialFolder := &gcs.Folder{
		Name:       "shared-folder/",
		UpdateTime: now,
	}
	sc.InsertFolder(initialFolder, now.Add(time.Hour))

	const numGoroutines = 100
	var wg sync.WaitGroup
	var stop int32

	// Mutating Readers for MinObject scalar fields
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for atomic.LoadInt32(&stop) == 0 {
				hit, mo := sc.LookUp("shared-file.txt", now)
				if hit && mo.Name != "" {
					// Mutate value copy scalar fields
					mo.Name = fmt.Sprintf("mutated-by-%d", id)
					mo.Size += uint64(id)
					mo.Generation = 9999
					mo.Updated = time.Now()
				}
				time.Sleep(10 * time.Microsecond)
			}
		}(i)
	}

	// Mutating Readers for Folder scalar fields
	for i := 0; i < numGoroutines/4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for atomic.LoadInt32(&stop) == 0 {
				hit, f := sc.LookUpFolder("shared-folder/", now)
				if hit && f.Name != "" {
					f.Name = fmt.Sprintf("folder-mutated-%d", id)
					f.UpdateTime = time.Now()
				}
				time.Sleep(10 * time.Microsecond)
			}
		}(i)
	}

	// Concurrent Writers and Invalidation
	for i := 0; i < numGoroutines/4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id + 500)))
			for atomic.LoadInt32(&stop) == 0 {
				crc := uint32(rng.Uint32())
				freshObj := &gcs.MinObject{
					Name:           "shared-file.txt",
					Size:           uint64(rng.Intn(1000) + 100),
					Generation:     int64(rng.Intn(100) + 2),
					MetaGeneration: 1,
					Updated:        time.Now(),
					CRC32C:         &crc,
				}
				sc.Insert(freshObj, now.Add(time.Hour))
				time.Sleep(50 * time.Microsecond)
			}
		}(i)
	}

	time.Sleep(500 * time.Millisecond)
	atomic.StoreInt32(&stop, 1)
	wg.Wait()
}

// TestStatCache_ReferenceField_DataRace demonstrates that returning gcs.MinObject by value (*e.m)
// performs a shallow struct copy that leaks map pointer (Metadata) and scalar pointer (CRC32C),
// producing data races and concurrent map write panics when callers access/mutate reference fields.
func TestStatCache_ReferenceField_DataRace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping data race demonstration in short mode")
	}

	sharedCache := lru.NewShardedRadixCache(10000)
	defer sharedCache.Close()

	sc := metadata.NewStatCacheBucketView(sharedCache, "race-bucket")
	now := time.Now()

	crcInitial := uint32(100)
	initialObj := &gcs.MinObject{
		Name:       "race-file.txt",
		Generation: 1,
		CRC32C:     &crcInitial,
		Metadata: map[string]string{
			"initial": "true",
		},
	}
	sc.Insert(initialObj, now.Add(time.Hour))

	var wg sync.WaitGroup
	var stop int32

	// Goroutines mutating shared Metadata map
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for atomic.LoadInt32(&stop) == 0 {
				hit, mo := sc.LookUp("race-file.txt", now)
				if hit && mo.Metadata != nil {
					// Shallow copy leak: mo.Metadata points to cached map
					mo.Metadata[fmt.Sprintf("worker-%d", id)] = "val"
				}
				time.Sleep(10 * time.Microsecond)
			}
		}(i)
	}

	time.Sleep(100 * time.Millisecond)
	atomic.StoreInt32(&stop, 1)
	wg.Wait()
}
