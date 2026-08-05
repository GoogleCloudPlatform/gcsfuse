// Copyright 2020 Google LLC
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
	"syscall"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	chunkRetryDeadlineSecs   = 120
	chunkTransferTimeoutSecs = 10
)

type fakeBucketManager struct {
	buckets    map[string]gcsx.SyncerBucket
	setupTimes int
}

func (bm *fakeBucketManager) SetUpBucket(
	ctx context.Context,
	name string, isMultibucketMount bool, _ metrics.MetricHandle) (sb gcsx.SyncerBucket, err error) {
	bm.setupTimes++

	var ok bool
	sb, ok = bm.buckets[name]
	if ok {
		return
	}
	err = fmt.Errorf("Cannot open bucket %q", name)
	return
}

func (bm *fakeBucketManager) ShutDown() {}

func (bm *fakeBucketManager) SetUpTimes() int {
	return bm.setupTimes
}

func setupBaseDirTest(t *testing.T) (context.Context, *timeutil.SimulatedClock, *fakeBucketManager, DirInode) {
	ctx := context.Background()
	clock := &timeutil.SimulatedClock{}
	clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.UTC))

	bm := &fakeBucketManager{
		buckets: make(map[string]gcsx.SyncerBucket),
	}
	bm.buckets["bucketA"] = gcsx.NewSyncerBucket(
		1,
		chunkRetryDeadlineSecs,
		chunkTransferTimeoutSecs,
		".gcsfuse_tmp/",
		fake.NewFakeBucket(clock, "bucketA", gcs.BucketType{}),
	)
	bm.buckets["bucketB"] = gcsx.NewSyncerBucket(
		1,
		chunkRetryDeadlineSecs,
		chunkTransferTimeoutSecs,
		".gcsfuse_tmp/",
		fake.NewFakeBucket(clock, "bucketB", gcs.BucketType{}),
	)

	in := NewBaseDirInode(
		dirInodeID,
		NewRootName(""),
		bm,
		metrics.NewNoopMetrics(),
		isTypeCacheDeprecationEnabled,
	)

	// Cast to sync.Locker to satisfy the linter which struggles with interface embedding.
	if locker, ok := in.(sync.Locker); ok {
		locker.Lock()
	} else {
		t.Fatal("DirInode does not implement sync.Locker")
	}

	return ctx, clock, bm, in
}

func runWithBaseDirInode(t *testing.T, testFunc func(ctx context.Context, clock *timeutil.SimulatedClock, bm *fakeBucketManager, in DirInode)) {
	ctx, clock, bm, in := setupBaseDirTest(t)
	defer func() {
		if locker, ok := in.(sync.Locker); ok {
			locker.Unlock()
		}
	}()
	testFunc(ctx, clock, bm, in)
}

func TestBaseDir_ID(t *testing.T) {
	runWithBaseDirInode(t, func(ctx context.Context, clock *timeutil.SimulatedClock, bm *fakeBucketManager, in DirInode) {
		assert.Equal(t, fuseops.InodeID(dirInodeID), in.ID())
	})
}

func TestBaseDir_Name(t *testing.T) {
	runWithBaseDirInode(t, func(ctx context.Context, clock *timeutil.SimulatedClock, bm *fakeBucketManager, in DirInode) {
		assert.Equal(t, "", in.Name().LocalName())
	})
}

func TestBaseDir_LookupCount(t *testing.T) {
	runWithBaseDirInode(t, func(ctx context.Context, clock *timeutil.SimulatedClock, bm *fakeBucketManager, in DirInode) {
		// Increment thrice. The count should now be three.
		in.IncrementLookupCount()
		in.IncrementLookupCount()
		in.IncrementLookupCount()

		// Decrementing twice shouldn't cause destruction. But one more should.
		require.False(t, in.DecrementLookupCount(2))
		assert.True(t, in.DecrementLookupCount(1))
	})
}

func TestBaseDir_Attributes_ClobberedCheckTrue(t *testing.T) {
	runWithBaseDirInode(t, func(ctx context.Context, clock *timeutil.SimulatedClock, bm *fakeBucketManager, in DirInode) {
		size, _, nlink, err := in.Attributes(ctx, true)

		require.NoError(t, err)
		assert.Equal(t, uint64(0), size)
		assert.Equal(t, uint32(1), nlink)
	})
}

func TestBaseDir_Attributes_ClobberedCheckFalse(t *testing.T) {
	runWithBaseDirInode(t, func(ctx context.Context, clock *timeutil.SimulatedClock, bm *fakeBucketManager, in DirInode) {
		size, _, nlink, err := in.Attributes(ctx, false)

		require.NoError(t, err)
		assert.Equal(t, uint64(0), size)
		assert.Equal(t, uint32(1), nlink)
	})
}

func TestBaseDir_LookUpChild_NonExistent(t *testing.T) {
	runWithBaseDirInode(t, func(ctx context.Context, clock *timeutil.SimulatedClock, bm *fakeBucketManager, in DirInode) {
		result, err := in.LookUpChild(ctx, "missing_bucket")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, 1, bm.SetUpTimes())
	})
}

func TestBaseDir_LookUpChild_BucketFound(t *testing.T) {
	runWithBaseDirInode(t, func(ctx context.Context, clock *timeutil.SimulatedClock, bm *fakeBucketManager, in DirInode) {
		result, err := in.LookUpChild(ctx, "bucketA")

		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "bucketA", result.Bucket.Name())
		assert.True(t, result.FullName.IsBucketRoot())
		assert.Equal(t, "bucketA/", result.FullName.LocalName())
		assert.Equal(t, "", result.FullName.GcsObjectName())
		assert.Nil(t, result.MinObject)
		assert.Equal(t, metadata.ImplicitDirType, result.Type())

		result, err = in.LookUpChild(ctx, "bucketB")

		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "bucketB", result.Bucket.Name())
		assert.True(t, result.FullName.IsBucketRoot())
		assert.Equal(t, "bucketB/", result.FullName.LocalName())
		assert.Equal(t, "", result.FullName.GcsObjectName())
		assert.Nil(t, result.MinObject)
		assert.Equal(t, metadata.ImplicitDirType, result.Type())
	})
}

func TestBaseDir_LookUpChild_BucketCached(t *testing.T) {
	runWithBaseDirInode(t, func(ctx context.Context, clock *timeutil.SimulatedClock, bm *fakeBucketManager, in DirInode) {
		_, _ = in.LookUpChild(ctx, "bucketA")
		assert.Equal(t, 1, bm.SetUpTimes())
		_, _ = in.LookUpChild(ctx, "bucketA")
		assert.Equal(t, 1, bm.SetUpTimes())
		_, _ = in.LookUpChild(ctx, "bucketB")
		assert.Equal(t, 2, bm.SetUpTimes())
		_, _ = in.LookUpChild(ctx, "bucketB")
		assert.Equal(t, 2, bm.SetUpTimes())
		_, _ = in.LookUpChild(ctx, "missing_bucket")
		assert.Equal(t, 3, bm.SetUpTimes())
	})
}

func TestBaseDir_ShouldInvalidateKernelListCache(t *testing.T) {
	runWithBaseDirInode(t, func(ctx context.Context, clock *timeutil.SimulatedClock, bm *fakeBucketManager, in DirInode) {
		ttl := time.Second
		assert.True(t, in.ShouldInvalidateKernelListCache(ttl))
	})
}

func TestBaseDir_ShouldInvalidateKernelListCache_TtlExpired(t *testing.T) {
	runWithBaseDirInode(t, func(ctx context.Context, clock *timeutil.SimulatedClock, bm *fakeBucketManager, in DirInode) {
		ttl := time.Second
		clock.AdvanceTime(10 * time.Second)

		assert.True(t, in.ShouldInvalidateKernelListCache(ttl))
	})
}

func TestBaseDir_ReadEntryCores(t *testing.T) {
	runWithBaseDirInode(t, func(ctx context.Context, clock *timeutil.SimulatedClock, bm *fakeBucketManager, in DirInode) {
		cores, unsupportedPaths, newTok, err := in.ReadEntryCores(ctx, "")

		// Should return ENOTSUP because listing is unsupported.
		assert.Nil(t, cores)
		assert.Nil(t, unsupportedPaths)
		assert.Equal(t, "", newTok)
		assert.Equal(t, syscall.ENOTSUP, err)
	})
}

func TestBaseDir_IsTypeCacheDeprecated_false(t *testing.T) {
	bm := &fakeBucketManager{}
	dInode := NewBaseDirInode(
		dirInodeID,
		NewRootName(""),
		bm,
		metrics.NewNoopMetrics(),
		false,
	)

	assert.False(t, dInode.IsTypeCacheDeprecated())
}

func TestBaseDir_IsTypeCacheDeprecated_true(t *testing.T) {
	bm := &fakeBucketManager{}
	dInode := NewBaseDirInode(
		dirInodeID,
		NewRootName(""),
		bm,
		metrics.NewNoopMetrics(),
		true,
	)

	assert.True(t, dInode.IsTypeCacheDeprecated())
}
