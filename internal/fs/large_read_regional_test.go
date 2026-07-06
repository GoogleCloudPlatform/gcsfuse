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
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLargeReadRegional_Success(t *testing.T) {
	ctx := context.Background()
	bucketName := "regional-bucket"
	// Zonal: false -> Regional bucket
	bucket := fake.NewFakeBucket(timeutil.RealClock(), bucketName, gcs.BucketType{Zonal: false, Hierarchical: false})

	serverCfg := &fs.ServerConfig{
		NewConfig: &cfg.Config{
			Write: cfg.WriteConfig{
				GlobalMaxBlocks: 1,
			},
			FileSystem: cfg.FileSystemConfig{
				EnableKernelReader: true,
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
	fileContent := "regional bucket content for vectored read verification."
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

	// Dst is nil, requesting full size.
	readOp := &fuseops.ReadFileOp{
		Inode:  lookUpOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 0,
		Size:   int64(len(fileContent)),
		Dst:    nil,
	}
	err = server.ReadFile(ctx, readOp)
	require.NoError(t, err)
	assert.Equal(t, len(fileContent), readOp.BytesRead)
	require.Len(t, readOp.Data, 1)
	assert.Equal(t, []byte(fileContent), readOp.Data[0])

	if readOp.Callback != nil {
		readOp.Callback()
	}
}

func TestLargeReadRegional_NotSupportedForZonal(t *testing.T) {
	ctx := context.Background()
	bucketName := "zonal-bucket"
	// Zonal: true -> Zonal bucket
	bucket := fake.NewFakeBucket(timeutil.RealClock(), bucketName, gcs.BucketType{Zonal: true, Hierarchical: false})

	serverCfg := &fs.ServerConfig{
		NewConfig: &cfg.Config{
			Write: cfg.WriteConfig{
				GlobalMaxBlocks: 1,
			},
			FileSystem: cfg.FileSystemConfig{
				EnableKernelReader: true,
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
	fileContent := "zonal bucket content."
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

	// Dst is nil, requesting full size. This should return ENOTSUPP/EOPNOTSUPP.
	readOp := &fuseops.ReadFileOp{
		Inode:  lookUpOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 0,
		Size:   int64(len(fileContent)),
		Dst:    nil,
	}
	err = server.ReadFile(ctx, readOp)
	assert.ErrorIs(t, err, syscall.EOPNOTSUPP)
}

func TestLargeReadRegional_NotSupportedWhenKernelReaderDisabled(t *testing.T) {
	ctx := context.Background()
	bucketName := "regional-bucket"
	bucket := fake.NewFakeBucket(timeutil.RealClock(), bucketName, gcs.BucketType{Zonal: false, Hierarchical: false})

	serverCfg := &fs.ServerConfig{
		NewConfig: &cfg.Config{
			Write: cfg.WriteConfig{
				GlobalMaxBlocks: 1,
			},
			FileSystem: cfg.FileSystemConfig{
				EnableKernelReader: false, // disabled
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
	fileContent := "regional bucket content."
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

	// Dst is nil, requesting full size. This should return ENOTSUPP/EOPNOTSUPP.
	readOp := &fuseops.ReadFileOp{
		Inode:  lookUpOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 0,
		Size:   int64(len(fileContent)),
		Dst:    nil,
	}
	err = server.ReadFile(ctx, readOp)
	assert.ErrorIs(t, err, syscall.EOPNOTSUPP)
}
