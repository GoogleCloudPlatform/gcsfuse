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
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type FastStatBucketEdgeTest struct {
	suite.Suite
	cache   metadata.StatCache
	clock   timeutil.SimulatedClock
	wrapped gcs.Bucket
	fsb     gcs.Bucket
}

func TestFastStatBucketEdge(t *testing.T) {
	suite.Run(t, new(FastStatBucketEdgeTest))
}

func (t *FastStatBucketEdgeTest) SetupTest() {
	capacity := 1000
	lruCache := lru.NewCache((cfg.AverageSizeOfPositiveStatCacheEntry + cfg.AverageSizeOfNegativeStatCacheEntry) * uint64(capacity))
	t.cache = metadata.NewStatCacheBucketView(lruCache, "")
	t.clock.SetTime(time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC))
	t.wrapped = fake.NewFakeBucket(&t.clock, "test_bucket", gcs.BucketType{})

	t.fsb = caching.NewFastStatBucket(
		10*time.Minute,
		t.cache,
		&t.clock,
		t.wrapped,
		5*time.Minute,
		true,
		true,
	)
}

// 1. Implicit Directory Hit Path
func (t *FastStatBucketEdgeTest) Test_StatObject_ImplicitDirectoryHit() {
	ctx := context.Background()
	dirName := "implicit_dir/"

	// Insert implicit directory
	t.cache.InsertImplicitDir(dirName, t.clock.Now().Add(10*time.Minute))

	// StatObject should return MinObject with Name=dirName and nil error from cache
	req := &gcs.StatObjectRequest{
		Name:               dirName,
		FetchOnlyFromCache: true,
	}

	minObj, extAttr, err := t.fsb.StatObject(ctx, req)
	require.NoError(t.T(), err)
	assert.Nil(t.T(), extAttr)
	require.NotNil(t.T(), minObj)
	assert.Equal(t.T(), dirName, minObj.Name)
	assert.Equal(t.T(), int64(0), minObj.Generation)
}

// 2. Ambiguity with Empty Name vs Negative Entry
func (t *FastStatBucketEdgeTest) Test_StatObject_EmptyNameAmbiguity() {
	ctx := context.Background()

	// Add negative entry for empty string object ""
	t.cache.AddNegativeEntry("", t.clock.Now().Add(5*time.Minute))

	req := &gcs.StatObjectRequest{
		Name:               "",
		FetchOnlyFromCache: true,
	}

	_, _, err := t.fsb.StatObject(ctx, req)
	require.Error(t.T(), err)
	var notFound *gcs.NotFoundError
	assert.ErrorAs(t.T(), err, &notFound)

	// Now insert implicit directory for ""
	t.cache.InsertImplicitDir("", t.clock.Now().Add(10*time.Minute))

	// When calling StatObject for "", the lookup returns MinObject{Name:"", Generation:0}
	// fastStatBucket checks entry.Name == "" && entry.Generation == 0 -> causing it to return NotFoundError!
	_, _, err2 := t.fsb.StatObject(ctx, req)
	require.Error(t.T(), err2)
	assert.ErrorAs(t.T(), err2, &notFound, "Zero value ambiguity: implicit dir '' is treated as negative cache entry")
}

// 3. Negative Cache Hits for StatObject and GetFolder
func (t *FastStatBucketEdgeTest) Test_NegativeCacheHits() {
	ctx := context.Background()

	// 1. Negative Object Entry
	t.cache.AddNegativeEntry("missing_file.txt", t.clock.Now().Add(5*time.Minute))
	reqStat := &gcs.StatObjectRequest{
		Name:               "missing_file.txt",
		FetchOnlyFromCache: true,
	}
	minObj, extAttr, errStat := t.fsb.StatObject(ctx, reqStat)
	assert.Nil(t.T(), minObj)
	assert.Nil(t.T(), extAttr)
	require.Error(t.T(), errStat)
	var notFound *gcs.NotFoundError
	assert.ErrorAs(t.T(), errStat, &notFound)
	assert.Contains(t.T(), errStat.Error(), "negative cache entry for missing_file.txt")

	// 2. Negative Folder Entry
	t.cache.AddNegativeEntryForFolder("missing_folder/", t.clock.Now().Add(5*time.Minute))
	reqFolder := &gcs.GetFolderRequest{
		Name:               "missing_folder/",
		FetchOnlyFromCache: true,
	}
	folder, errFolder := t.fsb.GetFolder(ctx, reqFolder)
	assert.Nil(t.T(), folder)
	require.Error(t.T(), errFolder)
	assert.ErrorAs(t.T(), errFolder, &notFound)
	assert.Contains(t.T(), errFolder.Error(), "negative cache entry for folder \"missing_folder/\"")
}

// 4. Folder Lookups & Expiration
func (t *FastStatBucketEdgeTest) Test_FolderLookup_AndExpiration() {
	ctx := context.Background()
	folderName := "photos/"
	f := &gcs.Folder{Name: folderName, UpdateTime: t.clock.Now()}

	t.cache.InsertFolder(f, t.clock.Now().Add(10*time.Minute))

	// Fetch from cache before expiration
	folder, err := t.fsb.GetFolder(ctx, &gcs.GetFolderRequest{
		Name:               folderName,
		FetchOnlyFromCache: true,
	})
	require.NoError(t.T(), err)
	require.NotNil(t.T(), folder)
	assert.Equal(t.T(), folderName, folder.Name)

	// Advance time past primaryCacheTTL (10m)
	t.clock.AdvanceTime(11 * time.Minute)

	// Fetch from cache after expiration -> should return CacheMissError
	_, errExpired := t.fsb.GetFolder(ctx, &gcs.GetFolderRequest{
		Name:               folderName,
		FetchOnlyFromCache: true,
	})
	require.Error(t.T(), errExpired)
	var cacheMiss *caching.CacheMissError
	assert.ErrorAs(t.T(), errExpired, &cacheMiss)
}

// 5. Context Cancellation Handling
func (t *FastStatBucketEdgeTest) Test_ContextCancellationHandling() {
	ctx := context.Background()

	// 1. Call ListObjects with cancelled context
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	reqList := &gcs.ListObjectsRequest{
		Prefix: "dir_canceled/",
	}
	// ListObjects wrapping fake bucket
	_, _ = t.fsb.ListObjects(canceledCtx, reqList)

	// Verify entries were NOT inserted into cache
	_, _, err1 := t.fsb.StatObject(context.Background(), &gcs.StatObjectRequest{
		Name:               "dir_canceled/",
		FetchOnlyFromCache: true,
	})
	require.Error(t.T(), err1)
	assert.ErrorAs(t.T(), err1, new(*caching.CacheMissError))
}

// 6. Stress Test on FastStatBucket with Concurrency
func (t *FastStatBucketEdgeTest) Test_FastStatBucket_ConcurrentStressHarness() {
	ctx := context.Background()
	var wg sync.WaitGroup
	numGoroutines := 20
	iterations := 200

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		workerID := i
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				name := fmt.Sprintf("stress_obj_%d_%d", workerID, j%10)
				folderName := fmt.Sprintf("stress_folder_%d_%d/", workerID, j%10)
				exp := t.clock.Now().Add(10 * time.Minute)

				// Concurrent read / write / stat / delete operations
				switch j % 4 {
				case 0:
					t.cache.Insert(&gcs.MinObject{Name: name, Generation: int64(j)}, exp)
					t.cache.InsertFolder(&gcs.Folder{Name: folderName, UpdateTime: t.clock.Now()}, exp)
				case 1:
					t.cache.AddNegativeEntry(name, exp)
					t.cache.AddNegativeEntryForFolder(folderName, exp)
				case 2:
					m, _, _ := t.fsb.StatObject(ctx, &gcs.StatObjectRequest{Name: name, FetchOnlyFromCache: true})
					if m != nil && m.Name != "" {
						assert.Equal(t.T(), name, m.Name)
					}
					f, _ := t.fsb.GetFolder(ctx, &gcs.GetFolderRequest{Name: folderName, FetchOnlyFromCache: true})
					if f != nil && f.Name != "" {
						assert.Equal(t.T(), folderName, f.Name)
					}
				case 3:
					t.cache.Erase(name)
					t.cache.Erase(folderName)
				}
			}
		}()
	}

	wg.Wait()
}
