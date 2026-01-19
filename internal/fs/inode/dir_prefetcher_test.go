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

package inode

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

type DirPrefetchTest struct {
	ctx    context.Context
	bucket gcsx.SyncerBucket
	fake   gcs.Bucket
	clock  timeutil.SimulatedClock
	in     *dirInode
	config *cfg.Config
	suite.Suite
}

func (t *DirPrefetchTest) setup(enablePrefetch bool, ttl time.Duration) (d *dirInode) {
	t.T().Helper()
	t.ctx = context.Background()
	t.clock.SetTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	t.fake = fake.NewFakeBucket(&t.clock, "some_bucket", gcs.BucketType{})
	t.bucket = gcsx.NewSyncerBucket(1, 10, ".gcsfuse_tmp/", t.fake)

	t.config = &cfg.Config{
		MetadataCache: cfg.MetadataCacheConfig{
			EnableMetadataPrefetch:       enablePrefetch,
			TypeCacheMaxSizeMb:           400,
			TtlSecs:                      60,
			MetadataPrefetchEntriesLimit: 5000,
		},
	}

	in := NewDirInode(
		dirInodeID,
		NewDirName(NewRootName(""), "dir/"),
		fuseops.InodeAttributes{Mode: dirMode},
		true, // implicitDirs
		false,
		ttl,
		&t.bucket,
		&t.clock,
		&t.clock,
		semaphore.NewWeighted(10),
		t.config,
	)
	return in.(*dirInode)
}

func (t *DirPrefetchTest) SetupTest() {
	t.in = t.setup(true, time.Minute)
	// Setup GCS state: a directory with files.
	files := []string{"dir/file0001", "dir/file0002", "dir/implicitDir0003/a", "dir/file0004"}
	require.NoError(t.T(), storageutil.CreateEmptyObjects(t.ctx, t.bucket, files))
}

func (t *DirPrefetchTest) TearDownTest() {
	err := t.in.Destroy()
	require.NoError(t.T(), err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func TestDirPrefetch(t *testing.T) {
	suite.Run(t, new(DirPrefetchTest))
}

// Tests that LookUpChild triggers the background prefetch and populates siblings.
func (t *DirPrefetchTest) TestPrefetch_TriggersOnUnknownType() {
	// Trigger LookUpChild for "implicitDir0003" which is not in cache.
	t.in.mu.Lock()
	_, err := t.in.LookUpChild(t.ctx, "implicitDir0003")
	t.in.mu.Unlock()
	require.NoError(t.T(), err)

	// Wait for the background worker to populate the cache for siblings.
	assert.Eventually(t.T(), func() bool {
		t.in.mu.RLock()
		defer t.in.mu.RUnlock()
		return t.in.cache.Get(t.clock.Now(), "file0001") == metadata.RegularFileType &&
			t.in.cache.Get(t.clock.Now(), "file0002") == metadata.RegularFileType &&
			t.in.cache.Get(t.clock.Now(), "implicitDir0003") == metadata.ImplicitDirType &&
			t.in.cache.Get(t.clock.Now(), "file0004") == metadata.RegularFileType
	}, 2*time.Second, 10*time.Millisecond, "Prefetch should populate all siblings in cache")
}

// Tests that if the directory is marked as large, prefetch starts from the looked-up object.
func (t *DirPrefetchTest) TestPrefetch_LargeDirUsesOffset() {
	// Set the inode to LargeDir mode.
	t.in.prefetcher.isLargeDir.Store(true)
	// Trigger LookUpChild for "file0002" which is not in cache.
	t.in.mu.Lock()
	_, err := t.in.LookUpChild(t.ctx, "file0002")
	t.in.mu.Unlock()
	require.NoError(t.T(), err)

	// Wait for the background worker to populate the cache for siblings.
	assert.Eventually(t.T(), func() bool {
		return t.in.prefetcher.state.Load() == prefetchReady
	}, 2*time.Second, 10*time.Millisecond)
	t.in.mu.RLock()
	defer t.in.mu.RUnlock()
	assert.Equal(t.T(), metadata.RegularFileType, t.in.cache.Get(t.clock.Now(), "file0002"))
	assert.Equal(t.T(), metadata.RegularFileType, t.in.cache.Get(t.clock.Now(), "file0004"))
	assert.Equal(t.T(), metadata.ImplicitDirType, t.in.cache.Get(t.clock.Now(), "implicitDir0003"))
	assert.Equal(t.T(), metadata.UnknownType, t.in.cache.Get(t.clock.Now(), "file0001"), "Objects before StartOffset should not be prefetched")
}

// Tests that if prefetch is disabled in config, LookUpChild doesn't trigger it.
func (t *DirPrefetchTest) TestPrefetch_Disabled() {
	t.in = t.setup(false, time.Minute) // Prefetch OFF

	t.in.mu.Lock()
	_, err := t.in.LookUpChild(t.ctx, "file0001")
	require.NoError(t.T(), err)
	t.in.mu.Unlock()

	// Give it a moment to ensure no background task runs.
	time.Sleep(100 * time.Millisecond)
	t.in.mu.RLock()
	defer t.in.mu.RUnlock()
	assert.Equal(t.T(), metadata.UnknownType, t.in.cache.Get(t.clock.Now(), "file0002"), "Sibling should not be cached when prefetch is disabled")
}

// Tests that only one prefetch can run at a time using the atomic state.
func (t *DirPrefetchTest) TestPrefetch_ConcurrentSafety() {
	// 1. Manually set state to InProgress.
	t.in.prefetcher.state.Store(prefetchInProgress)

	// 2. Call runOnDemandPrefetch. It should return immediately because of the state.
	t.in.prefetcher.Run(NewFileName(t.in.Name(), "file0001").GcsObjectName())

	// 3. Cache should remain empty because the function exited early.
	t.in.mu.RLock()
	defer t.in.mu.RUnlock()
	assert.Equal(t.T(), metadata.UnknownType, t.in.cache.Get(t.clock.Now(), "file0001"))
}

// Tests that the prefetcher respects the ctx and stops when Inode is destroyed.
func (t *DirPrefetchTest) TestPrefetch_CancellationOnDestroy() {
	// Trigger a prefetch.
	t.in.mu.Lock()
	_, _ = t.in.LookUpChild(t.ctx, "file0001")
	t.in.mu.Unlock()

	// Destroy the inode, which calls cancel().
	err := t.in.Destroy()
	require.NoError(t.T(), err)

	// The state should eventually return to Ready and context should be cancelled.
	assert.Equal(t.T(), t.in.prefetcher.ctx.Err(), context.Canceled)
	assert.Eventually(t.T(), func() bool {
		return t.in.prefetcher.state.Load() == prefetchReady
	}, 1*time.Second, 5*time.Millisecond)
}

func (t *DirPrefetchTest) TestPrefetch_RespectsMaxPrefetchCount() {
	t.in.prefetcher.maxPrefetchCount = 2 // Set to a small value.

	t.in.mu.Lock()
	_, err := t.in.LookUpChild(t.ctx, "file0001")
	t.in.mu.Unlock()
	require.NoError(t.T(), err)

	assert.Eventually(t.T(), func() bool {
		return t.in.prefetcher.state.Load() == prefetchReady
	}, 2*time.Second, 10*time.Millisecond)

	t.in.mu.RLock()
	defer t.in.mu.RUnlock()
	assert.Equal(t.T(), metadata.RegularFileType, t.in.cache.Get(t.clock.Now(), "file0001"))
	assert.Equal(t.T(), metadata.RegularFileType, t.in.cache.Get(t.clock.Now(), "file0002"))
	assert.Equal(t.T(), metadata.UnknownType, t.in.cache.Get(t.clock.Now(), "implicitDir003"), "Sibling implicitDir003 should NOT be cached (limit reached)")
	assert.Equal(t.T(), metadata.UnknownType, t.in.cache.Get(t.clock.Now(), "file0004"), "Sibling file0002 should NOT be cached (limit reached)")
	assert.True(t.T(), t.in.prefetcher.isLargeDir.Load())
}

func (t *DirPrefetchTest) TestPrefetch_HandlesMultiplePages() {
	// 1. Create 6000 objects with padded names to ensure consistent string sorting.
	// Names will be: dir/file0000, dir/file0001, ... dir/file5999
	files := make([]string, 0, 6000)
	for i := 0; i < 6000; i++ {
		files = append(files, fmt.Sprintf("dir/file%04d", i))
	}
	require.NoError(t.T(), storageutil.CreateEmptyObjects(t.ctx, t.bucket, files))
	// 2. Set maxPrefetchCount to 5500.
	// This forces the prefetcher to perform:
	// Page 1: 5000 results (MaxResultsForListObjectsCall)
	// Page 2: 500 results (Remainder)
	t.in.prefetcher.maxPrefetchCount = 5500

	// 3. Trigger LookUpChild. We use a name that starts at the beginning
	// of the sequence to ensure we fetch from file0000 onwards.
	t.in.mu.Lock()
	_, err := t.in.LookUpChild(t.ctx, "file0000")
	t.in.mu.Unlock()
	require.NoError(t.T(), err)

	// 4. Wait for the background worker to finish.
	assert.Eventually(t.T(), func() bool {
		return t.in.prefetcher.state.Load() == prefetchReady
	}, 2*time.Second, 20*time.Millisecond)
	// 5. Verify the large dir flag was set because listing wasn't finished.
	assert.True(t.T(), t.in.prefetcher.isLargeDir.Load(), "Inode should be marked as large directory")
	// 5. Verify the cache boundaries.
	t.in.mu.RLock()
	defer t.in.mu.RUnlock()
	// Check the first item.
	assert.Equal(t.T(), metadata.RegularFileType, t.in.cache.Get(t.clock.Now(), "file0000"))
	// Check an item right at the first page boundary (5000).
	assert.Equal(t.T(), metadata.RegularFileType, t.in.cache.Get(t.clock.Now(), "file4999"))
	// Check the last item that SHOULD be cached (5499 is the 5500th item).
	assert.Equal(t.T(), metadata.RegularFileType, t.in.cache.Get(t.clock.Now(), "file5499"))
	// Check the first item that SHOULD NOT be cached (5500).
	assert.Equal(t.T(), metadata.UnknownType, t.in.cache.Get(t.clock.Now(), "file5500"),
		"Should have stopped prefetching after 5500 items")
}

// mockListFunc returns a function that blocks until the provided channel is closed.
// This allows us to simulate long-running GCS calls to test concurrency limits.
func blockingListFunc(blockChan chan struct{}) func(context.Context, string, string, int) (map[Name]*Core, []string, string, error) {
	return func(ctx context.Context, tok string, start string, limit int) (map[Name]*Core, []string, string, error) {
		<-blockChan
		return make(map[Name]*Core), nil, "", nil
	}
}

func (t *DirPrefetchTest) TestMetadataPrefetcher_ConcurrencyLimit() {
	// Setup: Semaphore with a limit of 2 workers.
	limit := int64(2)
	sem := semaphore.NewWeighted(limit)
	blockChan := make(chan struct{})
	p1 := NewMetadataPrefetcher(t.config, sem, blockingListFunc(blockChan))
	p2 := NewMetadataPrefetcher(t.config, sem, blockingListFunc(blockChan))
	p3 := NewMetadataPrefetcher(t.config, sem, blockingListFunc(blockChan))
	// 1. Run two prefetches to fill up the limit.
	p1.Run("dir1/obj1")
	p2.Run("dir2/obj2")
	// 2. Wait until the semaphore is fully occupied.
	time.Sleep(10 * time.Millisecond)
	assert.False(t.T(), sem.TryAcquire(1), "Expected semaphore to be full")

	// 3. Trigger a third prefetch.
	// Because TryAcquire(1) is used in Run(), it should skip immediately if full.
	p3.Run("dir3/obj3")

	// 4. Release the blocked workers.
	close(blockChan)
	// 5. Use Eventually to wait until all permits are released.
	t.Eventually(func() bool {
		// Attempt to acquire the full weight. If successful, workers have finished.
		if sem.TryAcquire(limit) {
			sem.Release(limit)
			return true
		}
		return false
	}, 50*time.Millisecond, 5*time.Millisecond, "Expected all workers to release permits")
}

func (t *DirPrefetchTest) TestMetadataPrefetcher_RespectsMaxParallelPrefetchesConfig() {
	// User configures max 1 parallel prefetch.
	sem := semaphore.NewWeighted(1)
	blockChan := make(chan struct{})
	// Track how many times listFunc was actually entered.
	var callCount int
	var mu sync.Mutex
	listFunc := func(ctx context.Context, tok string, start string, limit int) (map[Name]*Core, []string, string, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		<-blockChan
		return nil, nil, "", nil
	}
	p := NewMetadataPrefetcher(t.config, sem, listFunc)

	// Trigger multiple runs on the same prefetcher (simulating different objects in same dir)
	// and different prefetchers.
	p.Run("a/1")
	p.Run("a/2") // Will be skipped by atomic state check anyway
	p2 := NewMetadataPrefetcher(t.config, sem, listFunc)
	p2.Run("b/1") // Should be skipped by semaphore check

	time.Sleep(10 * time.Millisecond)
	mu.Lock()
	assert.Equal(t.T(), 1, callCount, "Expected only 1 concurrent call based on semaphore")
	mu.Unlock()
	close(blockChan)
}
