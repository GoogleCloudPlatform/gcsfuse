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
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

type RecursiveCancellationTest struct {
	suite.Suite
	bucket gcsx.SyncerBucket
	fake   gcs.Bucket
	clock  timeutil.SimulatedClock
	config *cfg.Config
}

func (t *RecursiveCancellationTest) SetupTest() {
	t.clock.SetTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	t.fake = fake.NewFakeBucket(&t.clock, "some_bucket", gcs.BucketType{})
	t.bucket = gcsx.NewSyncerBucket(1, 10, ".gcsfuse_tmp/", t.fake)
	t.config = &cfg.Config{
		MetadataCache: cfg.MetadataCacheConfig{
			EnableMetadataPrefetch:       true,
			TypeCacheMaxSizeMb:           400,
			StatCacheMaxSizeMb:           400,
			TtlSecs:                      60,
			MetadataPrefetchEntriesLimit: 5000,
		},
	}
}

func (t *RecursiveCancellationTest) createDirInode(name Name, parentCtx context.Context) *dirInode {
	in := NewDirInode(
		fuseops.RootInodeID+1, // ID doesn't matter much
		name,
		parentCtx,
		fuseops.InodeAttributes{Mode: dirMode},
		true, // implicitDirs
		false,
		time.Minute,
		&t.bucket,
		&t.clock,
		&t.clock,
		semaphore.NewWeighted(10),
		t.config,
	)
	return in.(*dirInode)
}

func (t *RecursiveCancellationTest) TestRecursiveCancellation() {
	// Root dir
	rootDir := t.createDirInode(NewRootName(""), nil)

	// Child dir
	childName := NewDirName(rootDir.Name(), "child/")
	childDir := t.createDirInode(childName, rootDir.Context())

	// Grandchild dir
	grandChildName := NewDirName(childDir.Name(), "grandchild/")
	grandChildDir := t.createDirInode(grandChildName, childDir.Context())

	// Start prefetches (simulated by checking if contexts are active)
	assert.NoError(t.T(), rootDir.Context().Err())
	assert.NoError(t.T(), childDir.Context().Err())
	assert.NoError(t.T(), grandChildDir.Context().Err())

	// Cancel root
	rootDir.CancelSubdirectoryPrefetches()

	// Verify cancellation propagated
	assert.ErrorIs(t.T(), rootDir.Context().Err(), context.Canceled)
	assert.ErrorIs(t.T(), childDir.Context().Err(), context.Canceled)
	assert.ErrorIs(t.T(), grandChildDir.Context().Err(), context.Canceled)
}

func TestRecursiveCancellationSuite(t *testing.T) {
	suite.Run(t, new(RecursiveCancellationTest))
}
