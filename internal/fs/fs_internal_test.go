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

package fs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	storagemock "github.com/googlecloudplatform/gcsfuse/v3/internal/storage/mock"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util/diskutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

func TestCacheDirVolumeBlockSize(t *testing.T) {
	cacheDir := t.TempDir()
	actualBlockSize := diskutil.GetVolumeBlockSize(cacheDir)

	for _, tc := range []struct {
		name                         string
		disableSizeCalculationFix    bool
		enableExperimentalChunkCache bool
		expectedBlockSize            uint64
	}{
		{
			name:                         "SizeCalcFixEnabled_NotSparse",
			disableSizeCalculationFix:    false,
			enableExperimentalChunkCache: false,
			expectedBlockSize:            actualBlockSize,
		},
		{
			name:                         "SizeCalcFixDisabled",
			disableSizeCalculationFix:    true,
			enableExperimentalChunkCache: false,
			expectedBlockSize:            1,
		},
		{
			name:                         "SparseModeEnabled",
			disableSizeCalculationFix:    false,
			enableExperimentalChunkCache: true,
			expectedBlockSize:            1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serverCfg := &ServerConfig{
				NewConfig: &cfg.Config{
					FileCache: cfg.FileCacheConfig{
						ExperimentalDisableSizeCalculationFix: tc.disableSizeCalculationFix,
						ExperimentalEnableChunkCache:          tc.enableExperimentalChunkCache,
					},
				},
			}

			blockSize := cacheDirVolumeBlockSize(serverCfg, cacheDir)

			assert.Equal(t, tc.expectedBlockSize, blockSize)
		})
	}
}

func TestInitBufferedWriteHandlerAndSyncFileIfEligible_DestroysBWHOnSyncFailure(t *testing.T) {
	ctx := context.Background()
	mockBucket := new(storagemock.TestifyMockBucket)
	mockBucket.On("BucketType").Return(gcs.BucketType{Zonal: true})

	// StatObject succeeds for an empty unfinalized object, but CreateAppendableObjectWriter fails.
	mockBucket.On("StatObject", ctx, mock.AnythingOfType("*gcs.StatObjectRequest")).
		Return(&gcs.MinObject{
			Name:           "foo",
			Size:           0,
			Generation:     1,
			MetaGeneration: 1,
			Finalized:      0,
		}, &gcs.ExtendedObjectAttributes{}, nil)
	precondErr := &gcs.PreconditionError{Err: errors.New("takeover offset 52428800 does not match requested offset 0")}
	mockBucket.On("CreateAppendableObjectWriter", mock.Anything, mock.Anything).Return(nil, precondErr)

	clock := &timeutil.SimulatedClock{}
	clock.SetTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	syncerBucket := gcsx.NewSyncerBucket(1, 120, 10, ".gcsfuse_tmp/", mockBucket)

	newCfg := &cfg.Config{
		Write: cfg.WriteConfig{
			EnableStreamingWrites: true,
			EnableRapidAppends:    true,
			BlockSizeMb:           1,
			MaxBlocksPerFile:      5,
			GlobalMaxBlocks:       5,
		},
	}
	sem := semaphore.NewWeighted(5)

	fs := &fileSystem{
		mtimeClock:              clock,
		cacheClock:              clock,
		contentCache:            contentcache.New("", clock),
		inodes:                  make(map[fuseops.InodeID]inode.Inode),
		generationBackedInodes:  make(map[inode.Name]inode.GenerationBackedInode),
		implicitDirInodes:       make(map[inode.Name]inode.DirInode),
		folderInodes:            make(map[inode.Name]inode.DirInode),
		localFileInodes:         make(map[inode.Name]inode.Inode),
		handles:                 make(map[fuseops.HandleID]any),
		newConfig:               newCfg,
		metricHandle:            metrics.NewNoopMetrics(),
		traceHandle:             tracing.NewNoopTracer(),
		globalMaxWriteBlocksSem: sem,
	}
	fs.mu = locker.New("FS", func() {})

	backingObj := &gcs.MinObject{
		Name:           "foo",
		Size:           0,
		Generation:     1,
		MetaGeneration: 1,
		Finalized:      0,
	}
	f := inode.NewFileInode(
		fuseops.InodeID(2),
		inode.NewFileName(inode.NewRootName(""), "foo"),
		backingObj,
		&syncerBucket,
		false,
		fs.contentCache,
		clock,
		false,
		newCfg,
		sem,
		nil,
		fs.traceHandle,
		fs.metricHandle,
	)
	f.Lock()
	defer f.Unlock()

	err := fs.initBufferedWriteHandlerAndSyncFileIfEligible(ctx, f, util.NewOpenMode(util.ReadWrite, 0))

	require.Error(t, err)
	var clobberedErr *gcsfuse_errors.FileClobberedError
	assert.ErrorAs(t, err, &clobberedErr)
	// Broken BWH must be destroyed and cleared so subsequent ops do not reuse it or leak semaphore permits.
	assert.False(t, f.IsUsingBWH())
	assert.True(t, sem.TryAcquire(5))
	sem.Release(5)
	mockBucket.AssertExpectations(t)
}
