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
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v4/cfg"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/util/diskutil"
	"github.com/googlecloudplatform/gcsfuse/v4/metrics"
	"github.com/googlecloudplatform/gcsfuse/v4/tracing"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type failingUploadBucket struct {
	gcs.Bucket
	mu  sync.Mutex
	err error
}

func (b *failingUploadBucket) setErr(err error) {
	b.mu.Lock()
	b.err = err
	b.mu.Unlock()
}

func (b *failingUploadBucket) getErr() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.err
}

func (b *failingUploadBucket) CreateObject(ctx context.Context, req *gcs.CreateObjectRequest) (*gcs.Object, error) {
	if err := b.getErr(); err != nil {
		return nil, err
	}
	return b.Bucket.CreateObject(ctx, req)
}

func (b *failingUploadBucket) FinalizeUpload(ctx context.Context, w gcs.Writer) (*gcs.MinObject, error) {
	if err := b.getErr(); err != nil {
		return nil, err
	}
	return b.Bucket.FinalizeUpload(ctx, w)
}

func (b *failingUploadBucket) FlushPendingWrites(ctx context.Context, w gcs.Writer) (*gcs.MinObject, error) {
	if err := b.getErr(); err != nil {
		return nil, err
	}
	return b.Bucket.FlushPendingWrites(ctx, w)
}

type singleBucketManager struct{ bucket gcs.Bucket }

func (bm *singleBucketManager) SetUpBucket(_ context.Context, _ string, _ bool, _ metrics.MetricHandle) (gcsx.SyncerBucket, error) {
	return gcsx.NewSyncerBucket(0, 120, 10, ".gcsfuse_tmp/", gcsx.NewContentTypeBucket(bm.bucket), false), nil
}

func (bm *singleBucketManager) ShutDown() {}

func newTestFSWithInvariants(t *testing.T, streamingWrites bool, bucketType gcs.BucketType) (*fileSystem, *failingUploadBucket) {
	t.Helper()
	locker.EnableInvariantsCheck()

	clock := &timeutil.SimulatedClock{}
	clock.SetTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	fb := &failingUploadBucket{Bucket: fake.NewFakeBucket(clock, "test-bucket", bucketType)}

	server, err := NewFileSystem(context.Background(), &ServerConfig{
		NewConfig: &cfg.Config{
			Write: cfg.WriteConfig{
				EnableStreamingWrites: streamingWrites,
				BlockSizeMb:           1,
				MaxBlocksPerFile:      2,
				GlobalMaxBlocks:       4,
			},
		},
		CacheClock:    clock,
		BucketName:    "test-bucket",
		BucketManager: &singleBucketManager{bucket: fb},
		MetricHandle:  metrics.NewNoopMetrics(),
		TraceHandle:   tracing.NewNoopTracer(),
		TempDir:       t.TempDir(),
	})
	require.NoError(t, err)
	return server.(*fileSystem), fb
}

func createAndWriteLocalFile(t *testing.T, ctx context.Context, fs *fileSystem, name string, data []byte) *fuseops.CreateFileOp {
	t.Helper()
	createOp := &fuseops.CreateFileOp{Parent: fuseops.RootInodeID, Name: name, Mode: 0644}
	require.NoError(t, fs.CreateFile(ctx, createOp))
	require.NoError(t, fs.WriteFile(ctx, &fuseops.WriteFileOp{
		Inode:  createOp.Entry.Child,
		Handle: createOp.Handle,
		Data:   data,
	}))
	return createOp
}

func TestLocalFile_UploadFailurePreservesInvariants(t *testing.T) {
	ctx := context.Background()
	uploadErr := errors.New("simulated upload failure")

	t.Run("staged_writes", func(t *testing.T) {
		fs, fb := newTestFSWithInvariants(t, false, gcs.BucketType{})
		fileName := inode.NewFileName(inode.NewRootName(""), "foo.txt")
		first := createAndWriteLocalFile(t, ctx, fs, "foo.txt", []byte("first"))

		// Failed SyncFile returns error without panicking and keeps the inode linked.
		fb.setErr(uploadErr)
		require.ErrorIs(t, fs.SyncFile(ctx, &fuseops.SyncFileOp{Inode: first.Entry.Child, Handle: first.Handle}), uploadErr)
		fs.mu.Lock()
		assert.Equal(t, fs.inodes[first.Entry.Child], fs.localFileInodes[fileName])
		fs.mu.Unlock()

		// Failed FlushFile returns error without panicking, unlinks the inode, and removes it from localFileInodes.
		require.ErrorIs(t, fs.FlushFile(ctx, &fuseops.FlushFileOp{Inode: first.Entry.Child, Handle: first.Handle}), uploadErr)
		fs.mu.Lock()
		assert.Nil(t, fs.localFileInodes[fileName])
		firstInode := fs.inodes[first.Entry.Child].(*inode.FileInode)
		fs.mu.Unlock()
		firstInode.Lock()
		assert.True(t, firstInode.IsLocal() && firstInode.IsUnlinked())
		firstInode.Unlock()

		// Creating a second local file with the same name and closing the first fd must not evict the second file.
		second := createAndWriteLocalFile(t, ctx, fs, "foo.txt", []byte("second"))
		require.NotEqual(t, first.Entry.Child, second.Entry.Child)
		require.NoError(t, fs.FlushFile(ctx, &fuseops.FlushFileOp{Inode: first.Entry.Child, Handle: first.Handle}))
		require.NoError(t, fs.ReleaseFileHandle(ctx, &fuseops.ReleaseFileHandleOp{Handle: first.Handle}))
		fs.mu.Lock()
		assert.Equal(t, fs.inodes[second.Entry.Child], fs.localFileInodes[fileName])
		fs.mu.Unlock()

		fb.setErr(nil)
		require.NoError(t, fs.FlushFile(ctx, &fuseops.FlushFileOp{Inode: second.Entry.Child, Handle: second.Handle}))
	})

	t.Run("streaming_writes", func(t *testing.T) {
		fs, fb := newTestFSWithInvariants(t, true, gcs.BucketType{})
		op := createAndWriteLocalFile(t, ctx, fs, "bar.txt", []byte("hello"))

		fb.setErr(uploadErr)
		require.ErrorIs(t, fs.FlushFile(ctx, &fuseops.FlushFileOp{Inode: op.Entry.Child, Handle: op.Handle}), uploadErr)

		fs.mu.Lock()
		in := fs.inodes[op.Entry.Child].(*inode.FileInode)
		assert.Nil(t, fs.localFileInodes[in.Name()])
		fs.mu.Unlock()
		in.Lock()
		assert.True(t, in.IsLocal() && in.IsUnlinked())
		in.Unlock()
	})

	t.Run("zonal_streaming_writes", func(t *testing.T) {
		fs, fb := newTestFSWithInvariants(t, true, gcs.BucketType{Zonal: true})

		// Failed CreateFile during initial sync must clean up localFileInodes and inodes.
		fb.setErr(uploadErr)
		require.ErrorIs(t, fs.CreateFile(ctx, &fuseops.CreateFileOp{Parent: fuseops.RootInodeID, Name: "fail.txt", Mode: 0644}), uploadErr)
		fs.mu.Lock()
		assert.Empty(t, fs.localFileInodes)
		assert.Len(t, fs.inodes, 1)
		fs.mu.Unlock()

		// Failed SyncFile on an open zonal file returns the error without panicking.
		fb.setErr(nil)
		op := createAndWriteLocalFile(t, ctx, fs, "zonal.txt", []byte("hello"))
		fb.setErr(uploadErr)
		require.ErrorIs(t, fs.SyncFile(ctx, &fuseops.SyncFileOp{Inode: op.Entry.Child, Handle: op.Handle}), uploadErr)
	})
}
