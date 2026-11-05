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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util/diskutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
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

func newFileInodeForTest(id fuseops.InodeID, name inode.Name, local bool) *inode.FileInode {
	bucket := gcsx.NewSyncerBucket(1, 10, 10, ".gcsfuse_tmp/", fake.NewFakeBucket(timeutil.RealClock(), "some_bucket", gcs.BucketType{}))
	return inode.NewFileInode(
		id,
		name,
		&gcs.MinObject{Name: name.GcsObjectName()},
		&bucket,
		false,
		nil,
		timeutil.RealClock(),
		local,
		&cfg.Config{},
		semaphore.NewWeighted(1),
		nil,
		tracing.NewNoopTracer(),
		metrics.NewNoopMetrics())
}

func TestPromoteToGenerationBacked(t *testing.T) {
	name := inode.NewFileName(inode.NewRootName("some_bucket"), "foo")
	// promoted is the inode being promoted after a successful sync. other is
	// another inode with the same name.
	const (
		none = iota
		promoted
		other
	)
	for _, tc := range []struct {
		name                      string
		localFileInode            int
		generationBackedInode     int
		wantLocalFileInode        int
		wantGenerationBackedInode int
	}{
		{
			name:                      "LocalFileIsIndexedByName",
			localFileInode:            promoted,
			generationBackedInode:     none,
			wantLocalFileInode:        none,
			wantGenerationBackedInode: promoted,
		},
		{
			name:                      "LocalFileReplacesStaleInodeWithSameName",
			localFileInode:            promoted,
			generationBackedInode:     other,
			wantLocalFileInode:        none,
			wantGenerationBackedInode: promoted,
		},
		{
			name:                      "SyncedFileStaysIndexed",
			localFileInode:            none,
			generationBackedInode:     promoted,
			wantLocalFileInode:        none,
			wantGenerationBackedInode: promoted,
		},
		{
			name:                      "SyncedFileIsIndexedWhenNameIsFree",
			localFileInode:            none,
			generationBackedInode:     none,
			wantLocalFileInode:        none,
			wantGenerationBackedInode: promoted,
		},
		{
			name:                      "SyncedFileDoesNotReplaceOtherInodeWithSameName",
			localFileInode:            none,
			generationBackedInode:     other,
			wantLocalFileInode:        none,
			wantGenerationBackedInode: other,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inodes := map[int]*inode.FileInode{
				promoted: newFileInodeForTest(fuseops.InodeID(10), name, tc.localFileInode == promoted),
				other:    newFileInodeForTest(fuseops.InodeID(11), name, tc.localFileInode == other),
			}
			fs := &fileSystem{
				mu:                     locker.New("FS", func() {}),
				localFileInodes:        make(map[inode.Name]inode.Inode),
				generationBackedInodes: make(map[inode.Name]inode.GenerationBackedInode),
			}
			if tc.localFileInode != none {
				fs.localFileInodes[name] = inodes[tc.localFileInode]
			}
			if tc.generationBackedInode != none {
				fs.generationBackedInodes[name] = inodes[tc.generationBackedInode]
			}

			fs.promoteToGenerationBacked(inodes[promoted])

			if tc.wantLocalFileInode == none {
				assert.NotContains(t, fs.localFileInodes, name)
			} else {
				assert.Same(t, inodes[tc.wantLocalFileInode], fs.localFileInodes[name])
			}
			assert.Same(t, inodes[tc.wantGenerationBackedInode], fs.generationBackedInodes[name])
		})
	}
}
