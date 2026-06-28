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
	"fmt"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

type quotaAcceptanceBucketManager struct {
	bucket  gcs.Bucket
	onlyDir string
}

func (bm quotaAcceptanceBucketManager) ShutDown() {}

func (bm quotaAcceptanceBucketManager) SetUpBucket(
	_ context.Context,
	name string,
	_ bool,
	_ metrics.MetricHandle) (gcsx.SyncerBucket, error) {
	if bm.bucket.Name() != name {
		return gcsx.SyncerBucket{}, fmt.Errorf("bucket %q does not exist", name)
	}
	bucket := bm.bucket
	if bm.onlyDir != "" {
		var err error
		bucket, err = gcsx.NewPrefixBucket(path.Clean(bm.onlyDir)+"/", bucket)
		if err != nil {
			return gcsx.SyncerBucket{}, err
		}
	}
	return gcsx.NewSyncerBucket(0, 0, 10, ".gcsfuse_tmp/", gcsx.NewContentTypeBucket(bucket)), nil
}

func newQuotaAcceptanceFileSystem(t *testing.T, bucket gcs.Bucket, configure func(*cfg.Config)) *fileSystem {
	t.Helper()
	return newQuotaAcceptanceFileSystemWithOnlyDir(t, bucket, "", configure)
}

func newQuotaAcceptanceFileSystemWithOnlyDir(t *testing.T, bucket gcs.Bucket, onlyDir string, configure func(*cfg.Config)) *fileSystem {
	t.Helper()

	config := &cfg.Config{
		EnableNewReader: true,
		FileCache: cfg.FileCacheConfig{
			MaxSizeMb: -1,
		},
		MetadataCache: cfg.MetadataCacheConfig{
			MetadataPrefetchMaxWorkers: 1,
			StatCacheMaxSizeMb:         33,
			TypeCacheMaxSizeMb:         4,
		},
		Read: cfg.ReadConfig{
			GlobalMaxBlocks: 1,
		},
		Write: cfg.WriteConfig{
			GlobalMaxBlocks: 1,
		},
	}
	configure(config)

	rawFS, err := NewFileSystem(context.Background(), &ServerConfig{
		CacheClock:    timeutil.RealClock(),
		BucketManager: quotaAcceptanceBucketManager{bucket: bucket, onlyDir: onlyDir},
		BucketName:    bucket.Name(),
		NewConfig:     config,
		MetricHandle:  metrics.NewNoopMetrics(),
		TraceHandle:   tracing.NewNoopTracer(),
		Uid:           uint32(os.Getuid()),
		Gid:           uint32(os.Getgid()),
		FilePerms:     0644,
		DirPerms:      0755,
	})
	require.NoError(t, err)

	fs, ok := rawFS.(*fileSystem)
	require.True(t, ok)
	return fs
}

func TestLogicalQuotaAcceptanceOnlyDirScopesColdStartScan(t *testing.T) {
	ctx := context.Background()
	bucket := fake.NewFakeBucket(timeutil.RealClock(), "bucket", gcs.BucketType{})
	_, err := storageutil.CreateObject(ctx, bucket, "quota-prefix/existing", make([]byte, 4<<20))
	require.NoError(t, err)
	_, err = storageutil.CreateObject(ctx, bucket, "outside-prefix/ignored", make([]byte, 4<<20))
	require.NoError(t, err)

	fs := newQuotaAcceptanceFileSystemWithOnlyDir(t, bucket, "quota-prefix", func(config *cfg.Config) {
		config.FileSystem.ExperimentalMaxSizeMb = 5
	})

	stat := &fuseops.StatFSOp{}
	require.NoError(t, fs.StatFS(ctx, stat))
	assert.Equal(t, uint64(40), stat.Blocks)
	assert.Equal(t, uint64(8), stat.BlocksFree)

	create := &fuseops.CreateFileOp{
		Parent:    fuseops.RootInodeID,
		Name:      "new",
		OpenFlags: syscall.O_WRONLY,
	}
	require.NoError(t, fs.CreateFile(ctx, create))
	require.NoError(t, fs.WriteFile(ctx, &fuseops.WriteFileOp{
		Inode:  create.Entry.Child,
		Handle: create.Handle,
		Data:   make([]byte, 1<<20),
	}))
	require.NoError(t, fs.StatFS(ctx, stat))
	assert.Equal(t, uint64(0), stat.BlocksFree)
}

func TestLogicalQuotaAcceptanceSizeLimit(t *testing.T) {
	ctx := context.Background()
	bucket := fake.NewFakeBucket(timeutil.RealClock(), "bucket", gcs.BucketType{})
	_, err := storageutil.CreateObject(ctx, bucket, "existing", make([]byte, 4<<20))
	require.NoError(t, err)
	fs := newQuotaAcceptanceFileSystem(t, bucket, func(config *cfg.Config) {
		config.FileSystem.ExperimentalMaxSizeMb = 5
	})

	stat := &fuseops.StatFSOp{}
	require.NoError(t, fs.StatFS(ctx, stat))
	assert.Equal(t, uint64(40), stat.Blocks)
	assert.Equal(t, uint64(8), stat.BlocksFree)

	create := &fuseops.CreateFileOp{
		Parent:    fuseops.RootInodeID,
		Name:      "new",
		OpenFlags: syscall.O_WRONLY,
	}
	require.NoError(t, fs.CreateFile(ctx, create))

	err = fs.WriteFile(ctx, &fuseops.WriteFileOp{
		Inode:  create.Entry.Child,
		Handle: create.Handle,
		Data:   make([]byte, 2<<20),
	})
	assert.ErrorIs(t, err, syscall.ENOSPC)

	require.NoError(t, fs.WriteFile(ctx, &fuseops.WriteFileOp{
		Inode:  create.Entry.Child,
		Handle: create.Handle,
		Data:   make([]byte, 1<<20),
	}))
	require.NoError(t, fs.StatFS(ctx, stat))
	assert.Equal(t, uint64(0), stat.BlocksFree)

	require.NoError(t, fs.Unlink(ctx, &fuseops.UnlinkOp{
		Parent: fuseops.RootInodeID,
		Name:   "new",
	}))
	require.NoError(t, fs.StatFS(ctx, stat))
	assert.Equal(t, uint64(8), stat.BlocksFree)
}

func TestLogicalQuotaAcceptanceInitialOverLimitAllowsDeleteRecovery(t *testing.T) {
	ctx := context.Background()
	bucket := fake.NewFakeBucket(timeutil.RealClock(), "bucket", gcs.BucketType{})
	_, err := storageutil.CreateObject(ctx, bucket, "too-large", make([]byte, 6<<20))
	require.NoError(t, err)
	fs := newQuotaAcceptanceFileSystem(t, bucket, func(config *cfg.Config) {
		config.FileSystem.ExperimentalMaxSizeMb = 5
	})

	stat := &fuseops.StatFSOp{}
	require.NoError(t, fs.StatFS(ctx, stat))
	assert.Equal(t, uint64(0), stat.BlocksFree)

	create := &fuseops.CreateFileOp{
		Parent:    fuseops.RootInodeID,
		Name:      "new",
		OpenFlags: syscall.O_WRONLY,
	}
	require.NoError(t, fs.CreateFile(ctx, create))
	err = fs.WriteFile(ctx, &fuseops.WriteFileOp{
		Inode:  create.Entry.Child,
		Handle: create.Handle,
		Data:   []byte("x"),
	})
	assert.ErrorIs(t, err, syscall.ENOSPC)

	require.NoError(t, fs.Unlink(ctx, &fuseops.UnlinkOp{
		Parent: fuseops.RootInodeID,
		Name:   "too-large",
	}))
	require.NoError(t, fs.StatFS(ctx, stat))
	assert.Equal(t, uint64(40), stat.BlocksFree)
}

func TestLogicalQuotaAcceptanceRenameOverExistingReleasesDestination(t *testing.T) {
	ctx := context.Background()
	bucket := fake.NewFakeBucket(timeutil.RealClock(), "bucket", gcs.BucketType{})
	_, err := storageutil.CreateObject(ctx, bucket, "src", make([]byte, 1<<20))
	require.NoError(t, err)
	_, err = storageutil.CreateObject(ctx, bucket, "dst", make([]byte, 4<<20))
	require.NoError(t, err)
	fs := newQuotaAcceptanceFileSystem(t, bucket, func(config *cfg.Config) {
		config.FileSystem.ExperimentalMaxSizeMb = 5
	})

	stat := &fuseops.StatFSOp{}
	require.NoError(t, fs.StatFS(ctx, stat))
	assert.Equal(t, uint64(0), stat.BlocksFree)

	require.NoError(t, fs.Rename(ctx, &fuseops.RenameOp{
		OldParent: fuseops.RootInodeID,
		OldName:   "src",
		NewParent: fuseops.RootInodeID,
		NewName:   "dst",
	}))
	require.NoError(t, fs.StatFS(ctx, stat))
	assert.Equal(t, uint64(32), stat.BlocksFree)
}

func TestLogicalQuotaAcceptanceFileLimit(t *testing.T) {
	ctx := context.Background()
	bucket := fake.NewFakeBucket(timeutil.RealClock(), "bucket", gcs.BucketType{})
	_, err := storageutil.CreateObject(ctx, bucket, "existing", []byte("data"))
	require.NoError(t, err)
	fs := newQuotaAcceptanceFileSystem(t, bucket, func(config *cfg.Config) {
		config.FileSystem.ExperimentalMaxFileCount = 2
	})

	first := &fuseops.CreateFileOp{
		Parent:    fuseops.RootInodeID,
		Name:      "first",
		OpenFlags: syscall.O_WRONLY,
	}
	require.NoError(t, fs.CreateFile(ctx, first))

	err = fs.CreateFile(ctx, &fuseops.CreateFileOp{
		Parent:    fuseops.RootInodeID,
		Name:      "second",
		OpenFlags: syscall.O_WRONLY,
	})
	assert.ErrorIs(t, err, syscall.ENOSPC)

	require.NoError(t, fs.Unlink(ctx, &fuseops.UnlinkOp{
		Parent: fuseops.RootInodeID,
		Name:   "first",
	}))
	require.NoError(t, fs.CreateFile(ctx, &fuseops.CreateFileOp{
		Parent:    fuseops.RootInodeID,
		Name:      "third",
		OpenFlags: syscall.O_WRONLY,
	}))
}
