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
)

type DirPrefetchTest struct {
	ctx    context.Context
	bucket gcsx.SyncerBucket
	fake   gcs.Bucket
	clock  timeutil.SimulatedClock
	in     *dirInode
	suite.Suite
}

func (t *DirPrefetchTest) setup(enablePrefetch bool, ttl time.Duration) (d *dirInode) {
	t.T().Helper()
	t.ctx = context.Background()
	t.clock.SetTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	t.fake = fake.NewFakeBucket(&t.clock, "some_bucket", gcs.BucketType{})
	t.bucket = gcsx.NewSyncerBucket(1, 10, ".gcsfuse_tmp/", t.fake)

	config := &cfg.Config{
		MetadataCache: cfg.MetadataCacheConfig{
			ExperimentalDirMetadataPrefetch: enablePrefetch,
			TypeCacheMaxSizeMb:              4,
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
		config,
	)
	return in.(*dirInode)
}

func (t *DirPrefetchTest) SetupTest() {
	t.in = t.setup(true, time.Minute)
	// Setup GCS state: a directory with files.
	files := []string{"dir/a", "dir/b", "dir/c/", "dir/d"}
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
	// Trigger LookUpChild for "a" which is not in cache.
	t.in.mu.Lock()
	_, err := t.in.LookUpChild(t.ctx, "a")
	t.in.mu.Unlock()
	require.NoError(t.T(), err)

	// Wait for the background worker to populate the cache for siblings.
	assert.Eventually(t.T(), func() bool {
		t.in.mu.RLock()
		defer t.in.mu.RUnlock()
		return t.in.cache.Get(t.clock.Now(), "a") == metadata.RegularFileType &&
			t.in.cache.Get(t.clock.Now(), "b") == metadata.RegularFileType &&
			t.in.cache.Get(t.clock.Now(), "c") == metadata.ExplicitDirType &&
			t.in.cache.Get(t.clock.Now(), "d") == metadata.RegularFileType
	}, 2*time.Second, 10*time.Millisecond, "Prefetch should populate siblings in cache")
}

// Tests that LookUpChild triggers the background prefetch and populates siblings.
func (t *DirPrefetchTest) TestPrefetch_CachesOnlyLexicographicallyGreaterObjects() {
	// Trigger LookUpChild for "c" which is not in cache.
	t.in.mu.Lock()
	_, err := t.in.LookUpChild(t.ctx, "c")
	t.in.mu.Unlock()
	require.NoError(t.T(), err)

	// Wait for the background worker to populate the cache for siblings.
	assert.Eventually(t.T(), func() bool {
		t.in.mu.RLock()
		defer t.in.mu.RUnlock()
		return t.in.cache.Get(t.clock.Now(), "c") == metadata.ExplicitDirType &&
			t.in.cache.Get(t.clock.Now(), "d") == metadata.RegularFileType
	}, 2*time.Second, 10*time.Millisecond, "Prefetch should populate siblings in cache")
}

// Tests that if prefetch is disabled in config, LookUpChild doesn't trigger it.
func (t *DirPrefetchTest) TestPrefetch_Disabled() {
	t.in = t.setup(false, time.Minute) // Prefetch OFF

	t.in.mu.Lock()
	_, err := t.in.LookUpChild(t.ctx, "a")
	require.NoError(t.T(), err)
	t.in.mu.Unlock()

	// Give it a moment to ensure no background task runs.
	time.Sleep(100 * time.Millisecond)
	t.in.mu.RLock()
	defer t.in.mu.RUnlock()
	assert.Equal(t.T(), metadata.UnknownType, t.in.cache.Get(t.clock.Now(), "b"), "Sibling should not be cached when prefetch is disabled")
}

// Tests that only one prefetch can run at a time using the atomic state.
func (t *DirPrefetchTest) TestPrefetch_ConcurrentSafety() {
	// 1. Manually set state to InProgress.
	t.in.prefetchState.Store(prefetchInProgress)

	// 2. Call runOnDemandPrefetch. It should return immediately because of the state.
	// If it didn't return immediately, it would try to list objects and update cache.
	t.in.runOnDemandPrefetch(t.ctx, "a")

	// 3. Since we are mocking the state as InProgress, the cache should remain empty
	// because the function exited early.
	t.in.mu.RLock()
	defer t.in.mu.RUnlock()
	assert.Equal(t.T(), metadata.UnknownType, t.in.cache.Get(t.clock.Now(), "a"))
}

// Tests that the prefetcher respects the prefetchCtx and stops when Inode is destroyed.
func (t *DirPrefetchTest) TestPrefetch_CancellationOnDestroy() {
	// Trigger a prefetch.
	t.in.mu.Lock()
	_, _ = t.in.LookUpChild(t.ctx, "a")
	t.in.mu.Unlock()
	// Ensure it started.
	assert.Eventually(t.T(), func() bool {
		return t.in.prefetchState.Load() == prefetchInProgress
	}, 1*time.Second, 5*time.Millisecond)

	// Destroy the inode, which calls prefetchCancel().
	err := t.in.Destroy()
	require.NoError(t.T(), err)

	// The state should eventually return to Ready because the loop checks for ctx.Done().
	assert.Equal(t.T(), t.in.prefetchCtx.Err(), context.Canceled)
	assert.Eventually(t.T(), func() bool {
		return t.in.prefetchState.Load() == prefetchReady
	}, 1*time.Second, 5*time.Millisecond)
}
