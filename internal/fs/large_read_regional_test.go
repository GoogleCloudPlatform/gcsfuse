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

package fs_test

import (
	"context"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLargeRead(ctx context.Context, t *testing.T, isZonal bool, enableKernelReader bool, fileContent string) (fuseutil.FileSystem, *fuseops.ReadFileOp) {
	t.Helper()
	bucketName := "regional-bucket"
	if isZonal {
		bucketName = "zonal-bucket"
	}
	bucket := fake.NewFakeBucket(timeutil.RealClock(), bucketName, gcs.BucketType{Zonal: isZonal, Hierarchical: false})

	serverCfg := &fs.ServerConfig{
		NewConfig: &cfg.Config{
			Write: cfg.WriteConfig{
				GlobalMaxBlocks: 1,
			},
			FileSystem: cfg.FileSystemConfig{
				EnableKernelReader: enableKernelReader,
			},
		},
		CacheClock: &timeutil.SimulatedClock{},
		BucketName: bucketName,
		BucketManager: &fakeBucketManager{
			buckets: map[string]gcs.Bucket{
				bucketName: bucket,
			},
		},
		SequentialReadSizeMb: 200,
		TraceHandle:          tracing.NewOTELTracer(),
		MetricHandle:         metrics.NewNoopMetrics(),
	}

	server, err := fs.NewFileSystem(ctx, serverCfg)
	require.NoError(t, err)

	fileName := "large-file"
	createWithContents(ctx, t, bucket, fileName, fileContent)
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err = server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)

	openOp := &fuseops.OpenFileOp{
		Inode: lookUpOp.Entry.Child,
	}
	err = server.OpenFile(ctx, openOp)
	require.NoError(t, err)

	readOp := &fuseops.ReadFileOp{
		Inode:  lookUpOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 0,
		Size:   int64(len(fileContent)),
		Dst:    nil,
	}

	return server, readOp
}

func TestLargeReadRegional_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	fileContent := "regional bucket content for vectored read verification."
	server, readOp := setupLargeRead(ctx, t, false, true, fileContent)

	// Act
	err := server.ReadFile(ctx, readOp)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, len(fileContent), readOp.BytesRead)
	require.Len(t, readOp.Data, 1)
	assert.Equal(t, []byte(fileContent), readOp.Data[0])
	if readOp.Callback != nil {
		readOp.Callback()
	}
}

func TestLargeReadRegional_NotSupportedForZonal(t *testing.T) {
	// Arrange
	ctx := context.Background()
	server, readOp := setupLargeRead(ctx, t, true, true, "zonal bucket content.")

	// Act
	err := server.ReadFile(ctx, readOp)

	// Assert
	assert.ErrorIs(t, err, syscall.ENOTSUP)
}

func TestLargeReadRegional_NotSupportedWhenKernelReaderDisabled(t *testing.T) {
	// Arrange
	ctx := context.Background()
	server, readOp := setupLargeRead(ctx, t, false, false, "regional bucket content.")

	// Act
	err := server.ReadFile(ctx, readOp)

	// Assert
	assert.ErrorIs(t, err, syscall.ENOTSUP)
}

func TestNewFileSystem_FuseMaxRequestSizeKbNotSupportedForRapid(t *testing.T) {
	// Arrange
	ctx := context.Background()
	bucketName := "zonal-bucket"
	bucket := fake.NewFakeBucket(timeutil.RealClock(), bucketName, gcs.BucketType{Zonal: true, Hierarchical: false})
	// Create a server config with fuse-max-request-size-kb set explicitly above default.
	serverCfg := &fs.ServerConfig{
		NewConfig: &cfg.Config{
			FileSystem: cfg.FileSystemConfig{
				FuseMaxRequestSizeKb: 16384,
			},
		},
		CacheClock: &timeutil.SimulatedClock{},
		BucketName: bucketName,
		BucketManager: &fakeBucketManager{
			buckets: map[string]gcs.Bucket{
				bucketName: bucket,
			},
		},
		SequentialReadSizeMb: 200,
		TraceHandle:          tracing.NewOTELTracer(),
		MetricHandle:         metrics.NewNoopMetrics(),
	}

	// Act
	_, err := fs.NewFileSystem(ctx, serverCfg)

	// Assert
	assert.ErrorContains(t, err, "fuse-max-request-size-kb is not supported for rapid buckets")
}

func TestNewFileSystem_DefaultFuseMaxRequestSizeKbAllowedForRapid(t *testing.T) {
	// Arrange
	ctx := context.Background()
	bucketName := "zonal-bucket"
	bucket := fake.NewFakeBucket(timeutil.RealClock(), bucketName, gcs.BucketType{Zonal: true, Hierarchical: false})
	// Create a server config with default fuse-max-request-size-kb (1024).
	serverCfg := &fs.ServerConfig{
		NewConfig: &cfg.Config{
			FileSystem: cfg.FileSystemConfig{
				FuseMaxRequestSizeKb: int64(cfg.StorageClassRapid.DefaultFuseMaxRequestSizeKb()),
			},
		},
		CacheClock: &timeutil.SimulatedClock{},
		BucketName: bucketName,
		BucketManager: &fakeBucketManager{
			buckets: map[string]gcs.Bucket{
				bucketName: bucket,
			},
		},
		SequentialReadSizeMb: 200,
		TraceHandle:          tracing.NewOTELTracer(),
		MetricHandle:         metrics.NewNoopMetrics(),
	}

	// Act
	_, err := fs.NewFileSystem(ctx, serverCfg)

	// Assert
	assert.NoError(t, err)
}
