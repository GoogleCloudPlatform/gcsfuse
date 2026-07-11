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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyWrappedBucket struct {
	gcs.Bucket
}

func (d *dummyWrappedBucket) Name() string { return "dummy-bucket" }
func (d *dummyWrappedBucket) BucketType() gcs.BucketType {
	return gcs.BucketType{Hierarchical: true}
}
func (d *dummyWrappedBucket) StatObject(ctx context.Context, req *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error) {
	crc := uint32(100)
	return &gcs.MinObject{
		Name:           req.Name,
		Size:           1024,
		Generation:     1,
		MetaGeneration: 1,
		Updated:        time.Now(),
		CRC32C:         &crc,
	}, nil, nil
}
func (d *dummyWrappedBucket) GetFolder(ctx context.Context, req *gcs.GetFolderRequest) (*gcs.Folder, error) {
	return &gcs.Folder{
		Name:       req.Name,
		UpdateTime: time.Now(),
	}, nil
}
func (d *dummyWrappedBucket) CreateObject(ctx context.Context, req *gcs.CreateObjectRequest) (*gcs.Object, error) {
	return &gcs.Object{
		Name:       req.Name,
		Size:       2048,
		Generation: 2,
	}, nil
}
func (d *dummyWrappedBucket) DeleteObject(ctx context.Context, req *gcs.DeleteObjectRequest) error {
	return nil
}
func (d *dummyWrappedBucket) DeleteFolder(ctx context.Context, name string) error {
	return nil
}
func (d *dummyWrappedBucket) CreateFolder(ctx context.Context, name string) (*gcs.Folder, error) {
	return &gcs.Folder{
		Name:       name,
		UpdateTime: time.Now(),
	}, nil
}

// TestFastStatBucket_ExtremeConcurrencyStress stress-tests FastStatBucket backed by ShardedRadixCache
// under 100 concurrent goroutines executing high-rate Stat, Create, Delete, GetFolder operations.
func TestFastStatBucket_ExtremeConcurrencyStress(t *testing.T) {
	sc := lru.NewShardedRadixCache(100000)
	defer sc.Close()

	statCache := metadata.NewStatCacheBucketView(sc, "test-bucket")
	clock := timeutil.RealClock()
	wrapped := &dummyWrappedBucket{}

	fsb := caching.NewFastStatBucket(
		time.Minute,
		statCache,
		clock,
		wrapped,
		time.Minute,
		true,
		true,
	)

	const numGoroutines = 100
	const opsPerRoutine = 500
	ctx := context.Background()

	keys := []string{
		"file1.dat", "file2.dat", "dir1/subfile.txt", "dir2/another.txt",
		"folder1/", "folder2/",
	}

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id)))

			for op := 0; op < opsPerRoutine; op++ {
				key := keys[rng.Intn(len(keys))]

				switch rng.Intn(6) {
				case 0: // StatObject (hits cache or fallback)
					req := &gcs.StatObjectRequest{Name: key}
					m, _, err := fsb.StatObject(ctx, req)
					if err == nil {
						require.NotNil(t, m)
						assert.Equal(t, key, m.Name)
					}

				case 1: // GetFolder
					req := &gcs.GetFolderRequest{Name: key}
					f, err := fsb.GetFolder(ctx, req)
					if err == nil {
						require.NotNil(t, f)
					}

				case 2: // CreateObject
					req := &gcs.CreateObjectRequest{Name: key}
					obj, err := fsb.CreateObject(ctx, req)
					require.NoError(t, err)
					require.NotNil(t, obj)

				case 3: // DeleteObject
					req := &gcs.DeleteObjectRequest{Name: key}
					err := fsb.DeleteObject(ctx, req)
					require.NoError(t, err)

				case 4: // CreateFolder
					f, err := fsb.CreateFolder(ctx, key)
					require.NoError(t, err)
					require.NotNil(t, f)

				case 5: // DeleteFolder
					err := fsb.DeleteFolder(ctx, key)
					require.NoError(t, err)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestFastStatBucket_ScalarThreadIsolation verifies that scalar mutations on returned pointer
// to stack-copied MinObject and Folder structs are thread-isolated and do not race.
func TestFastStatBucket_ScalarThreadIsolation(t *testing.T) {
	sc := lru.NewShardedRadixCache(100000)
	defer sc.Close()

	statCache := metadata.NewStatCacheBucketView(sc, "test-bucket")
	clock := timeutil.RealClock()
	wrapped := &dummyWrappedBucket{}

	fsb := caching.NewFastStatBucket(
		time.Minute,
		statCache,
		clock,
		wrapped,
		time.Minute,
		true,
		true,
	)

	ctx := context.Background()

	// Pre-populate cache via StatObject
	req := &gcs.StatObjectRequest{Name: "shared.txt"}
	m0, _, err := fsb.StatObject(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, m0)

	f0, err := fsb.GetFolder(ctx, &gcs.GetFolderRequest{Name: "shared-dir/"})
	require.NoError(t, err)
	require.NotNil(t, f0)

	const numGoroutines = 80
	var wg sync.WaitGroup
	var stop int32

	// Readers mutators on StatObject returned pointer scalar fields
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for atomic.LoadInt32(&stop) == 0 {
				m, _, err := fsb.StatObject(ctx, &gcs.StatObjectRequest{Name: "shared.txt"})
				if err == nil && m != nil {
					// Mutate scalar fields of the returned struct copy
					m.Name = fmt.Sprintf("mutated-%d", id)
					m.Size += uint64(id)
					m.Generation += int64(id)
				}
				time.Sleep(10 * time.Microsecond)
			}
		}(i)
	}

	// Readers mutators on GetFolder returned pointer scalar fields
	for i := 0; i < numGoroutines/4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for atomic.LoadInt32(&stop) == 0 {
				f, err := fsb.GetFolder(ctx, &gcs.GetFolderRequest{Name: "shared-dir/"})
				if err == nil && f != nil {
					f.Name = fmt.Sprintf("folder-mut-%d", id)
					f.UpdateTime = time.Now()
				}
				time.Sleep(10 * time.Microsecond)
			}
		}(i)
	}

	time.Sleep(500 * time.Millisecond)
	atomic.StoreInt32(&stop, 1)
	wg.Wait()
}

// TestFastStatBucket_PointerAliasing_DataRace demonstrates that taking the address of a local stack struct
// (m = &entry) and passing m into sc.Insert(m) stores the raw pointer in the cache, causing data races
// when m is subsequently mutated while cache invalidations/size updates access entry.m.
func TestFastStatBucket_PointerAliasing_DataRace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping pointer aliasing data race demonstration in short mode")
	}

	sc := lru.NewShardedRadixCache(10000)
	defer sc.Close()

	statCache := metadata.NewStatCacheBucketView(sc, "race-bucket")
	clock := timeutil.RealClock()
	wrapped := &dummyWrappedBucket{}

	fsb := caching.NewFastStatBucket(
		time.Minute,
		statCache,
		clock,
		wrapped,
		time.Minute,
		true,
		true,
	)

	ctx := context.Background()

	var wg sync.WaitGroup
	var stop int32

	// Goroutine A: calls StatObject, gets m = &entry, mutates m
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for atomic.LoadInt32(&stop) == 0 {
				m, _, err := fsb.StatObject(ctx, &gcs.StatObjectRequest{Name: "shared-race.txt"})
				if err == nil && m != nil {
					m.Name = fmt.Sprintf("mutated-name-%d", id)
					m.Generation += int64(id)
				}
				time.Sleep(10 * time.Microsecond)
			}
		}(i)
	}

	// Goroutine B: calls CreateObject / DeleteObject which trigger sc.Insert(m) and entry.Size()
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for atomic.LoadInt32(&stop) == 0 {
				_, _ = fsb.CreateObject(ctx, &gcs.CreateObjectRequest{Name: "shared-race.txt"})
				_ = fsb.DeleteObject(ctx, &gcs.DeleteObjectRequest{Name: "shared-race.txt"})
				time.Sleep(50 * time.Microsecond)
			}
		}(i)
	}

	time.Sleep(200 * time.Millisecond)
	atomic.StoreInt32(&stop, 1)
	wg.Wait()
}
