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

package inode

import (
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
	"golang.org/x/net/context"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////
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
	// 1. Setup GCS state: a directory with 3 files.
	files := []string{"dir/file1.txt", "dir/file2.txt", "dir/dir/"}
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

func (t *DirPrefetchTest) TestPrefetch_TriggersOnUnknownType() {
	// 1. Trigger LookUpChild for a name not in cache.
	// This should trigger the background prefetch worker.
	t.in.mu.Lock()
	_, err := t.in.LookUpChild(t.ctx, "file1.txt")
	t.in.mu.Unlock()
	require.NoError(t.T(), err)

	// 2. Wait for the background worker to populate the cache.
	assert.Eventually(t.T(), func() bool {
		t.in.mu.RLock()
		defer t.in.mu.RUnlock()
		return t.in.cache.Get(t.clock.Now(), "file2.txt") == metadata.RegularFileType &&
			t.in.cache.Get(t.clock.Now(), "dir") == metadata.ExplicitDirType
	}, 2*time.Second, 10*time.Millisecond, "Prefetch should populate siblings in cache")
}

func (t *DirPrefetchTest) TestPrefetch_RespectsTTL() {
	// First call triggers prefetch.
	t.in.mu.Lock()
	_, err := t.in.LookUpChild(t.ctx, "file1.txt")
	require.NoError(t.T(), err)
	t.in.mu.Unlock()
	// Advance clock but stay within TTL.
	t.clock.AdvanceTime(time.Minute / 2)
	// Reset prefetch state manually to simulate a finished task.
	t.in.prefetchState.Store(prefetchReady)
	// Second call should NOT trigger prefetch because lastPrefetchTime is recent.
	// We check this by seeing if lastPrefetchTime changes.
	originalPrefetchTime := t.in.lastPrefetchTime

	t.in.mu.Lock()
	_, err = t.in.LookUpChild(t.ctx, "file1.txt")
	require.NoError(t.T(), err)
	t.in.mu.Unlock()

	assert.Equal(t.T(), originalPrefetchTime, t.in.lastPrefetchTime)
}

func (t *DirPrefetchTest) TestPrefetch_Disabled() {
	t.in = t.setup(false, time.Minute) // Prefetch OFF

	t.in.mu.Lock()
	_, err := t.in.LookUpChild(t.ctx, "file1.txt")
	require.NoError(t.T(), err)
	t.in.mu.Unlock()

	// Give it a moment to ensure no background task runs.
	time.Sleep(100 * time.Millisecond)
	t.in.mu.RLock()
	assert.Equal(t.T(), metadata.UnknownType, t.in.cache.Get(t.clock.Now(), "file2.txt"), "Sibling should not be cached")
	t.in.mu.RUnlock()
}

func (t *DirPrefetchTest) TestPrefetch_ConcurrentSafety() {
	// Simulate prefetch already in progress.
	t.in.prefetchState.Store(prefetchInProgress)

	// Call runOnDemandPrefetch. It should return immediately without doing work.
	t.in.runOnDemandPrefetch(t.ctx, "")

	// lastPrefetchTime should remain Zero.
	assert.True(t.T(), t.in.lastPrefetchTime.IsZero())
}
