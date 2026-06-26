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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLargeRead_NilOrSmallDst(t *testing.T) {
	ctx := context.Background()
	bucketName := "test-bucket"
	bucket := fake.NewFakeBucket(timeutil.RealClock(), bucketName, gcs.BucketType{Hierarchical: false})

	serverCfg := &fs.ServerConfig{
		NewConfig: &cfg.Config{
			Write: cfg.WriteConfig{
				GlobalMaxBlocks: 1,
			},
			Read: cfg.ReadConfig{
				EnableBufferedRead: false,
			},
			EnableNewReader: true,
			FileSystem: cfg.FileSystemConfig{
				IgnoreInterrupts: true,
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

	// Write a file to the fake bucket using the shared createWithContents helper.
	fileName := "large-file"
	fileContent := "hello world! this is a test of the new large read path where Dst is nil."
	createWithContents(ctx, t, bucket, fileName, fileContent)

	// Look up the file inode.
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err = server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)

	// Open the file.
	openOp := &fuseops.OpenFileOp{
		Inode: lookUpOp.Entry.Child,
	}
	err = server.OpenFile(ctx, openOp)
	require.NoError(t, err)

	// 1. Test case: Dst is nil, requesting full size.
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

	// Invoke callback if present (to return to pool).
	if readOp.Callback != nil {
		readOp.Callback()
	}

	// 2. Test case: Dst has insufficient capacity (e.g. cap 5, but size 10).
	readOp2 := &fuseops.ReadFileOp{
		Inode:  lookUpOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 0,
		Size:   10,
		Dst:    make([]byte, 2, 5), // len 2, cap 5 (insufficient for size 10)
	}
	err = server.ReadFile(ctx, readOp2)
	require.NoError(t, err)
	assert.Equal(t, 10, readOp2.BytesRead)
	require.Len(t, readOp2.Data, 1)
	assert.Equal(t, []byte(fileContent[:10]), readOp2.Data[0])
	if readOp2.Callback != nil {
		readOp2.Callback()
	}

	// 3. Test case: Dst has sufficient capacity (traditional path).
	dstBuf := make([]byte, len(fileContent))
	readOp3 := &fuseops.ReadFileOp{
		Inode:  lookUpOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 0,
		Size:   int64(len(fileContent)),
		Dst:    dstBuf,
	}
	err = server.ReadFile(ctx, readOp3)
	require.NoError(t, err)
	assert.Equal(t, len(fileContent), readOp3.BytesRead)
	assert.Equal(t, []byte(fileContent), readOp3.Dst[:readOp3.BytesRead])
}
