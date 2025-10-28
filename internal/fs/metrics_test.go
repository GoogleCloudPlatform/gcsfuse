// Copyright 2025 Google LLC
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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/wrappers"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
)

// serverConfigParams holds parameters for creating a test file system.
type serverConfigParams struct {
	enableBufferedRead bool
	enableNewReader    bool
}

func defaultServerConfigParams() *serverConfigParams {
	return &serverConfigParams{
		enableBufferedRead: false,
		enableNewReader:    true,
	}
}

func createTestFileSystemWithMetrics(ctx context.Context, t *testing.T, params *serverConfigParams) (gcs.Bucket, fuseutil.FileSystem, metrics.MetricHandle, *metric.ManualReader) {
	t.Helper()
	origProvider := otel.GetMeterProvider()
	t.Cleanup(func() { otel.SetMeterProvider(origProvider) })
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)

	mh, err := metrics.NewOTelMetrics(ctx, 1, 100)
	require.NoError(t, err, "metrics.NewOTelMetrics")
	bucketName := "test-bucket"
	bucket := fake.NewFakeBucket(timeutil.RealClock(), bucketName, gcs.BucketType{Hierarchical: false})
	serverCfg := &fs.ServerConfig{
		NewConfig: &cfg.Config{
			Write: cfg.WriteConfig{
				GlobalMaxBlocks: 1,
			},
			Read: cfg.ReadConfig{
				EnableBufferedRead: params.enableBufferedRead,
				GlobalMaxBlocks:    1,
				BlockSizeMb:        1,
				MaxBlocksPerHandle: 10,
			},
			EnableNewReader: params.enableNewReader,
		},
		MetricHandle: mh,
		CacheClock:   &timeutil.SimulatedClock{},
		BucketName:   bucketName,
		BucketManager: &fakeBucketManager{
			buckets: map[string]gcs.Bucket{
				bucketName: bucket,
			},
		},
		SequentialReadSizeMb: 200,
	}
	server, err := fs.NewFileSystem(ctx, serverCfg)
	require.NoError(t, err, "NewFileSystem")
	return bucket, server, mh, reader
}

func createWithContents(ctx context.Context, t *testing.T, bucket gcs.Bucket, name string, contents string) {
	err := storageutil.CreateObjects(ctx, bucket, map[string][]byte{name: []byte(contents)})
	require.NoError(t, err, "CreateObjects")
}

func waitForMetricsProcessing() {
	time.Sleep(time.Millisecond)
}

func TestLookUpInode_Metrics(t *testing.T) {
	testCases := []struct {
		name          string
		fileName      string
		createFile    bool
		expectedError error
	}{
		{
			name:          "non-existent file",
			fileName:      "non_existent",
			createFile:    false,
			expectedError: fuse.ENOENT,
		},
		{
			name:          "existing file",
			fileName:      "test",
			createFile:    true,
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
			content := "test"
			if tc.createFile {
				createWithContents(ctx, t, bucket, tc.fileName, content)
			}
			server = wrappers.WithMonitoring(server, mh)
			op := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   tc.fileName,
			}

			err := server.LookUpInode(ctx, op)
			waitForMetricsProcessing()

			assert.Equal(t, tc.expectedError, err)
			attrs := attribute.NewSet(attribute.String("fs_op", "LookUpInode"))
			metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
			metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
		})
	}
}

func TestReadFile_BufferedReadMetrics(t *testing.T) {
	ctx := context.Background()
	params := defaultServerConfigParams()
	params.enableBufferedRead = true
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, params)
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test.txt"
	content := "test content"
	createWithContents(ctx, t, bucket, fileName, content)
	lookupOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookupOp)
	require.NoError(t, err, "LookUpInode")
	openOp := &fuseops.OpenFileOp{
		Inode: lookupOp.Entry.Child,
	}
	err = server.OpenFile(ctx, openOp)
	require.NoError(t, err, "OpenFile")
	readOp := &fuseops.ReadFileOp{
		Inode:  lookupOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 0,
		Dst:    make([]byte, len(content)),
	}

	err = server.ReadFile(ctx, readOp)
	waitForMetricsProcessing()

	require.NoError(t, err, "ReadFile")
	metrics.VerifyCounterMetric(t, ctx, reader, "gcs/download_bytes_count", attribute.NewSet(attribute.String("read_type", string(metrics.ReadTypeBufferedAttr))), int64(len(content)))
	metrics.VerifyCounterMetric(t, ctx, reader, "gcs/read_bytes_count", attribute.NewSet(attribute.String("reader", string(metrics.ReaderBufferedAttr))), int64(len(content)))
	metrics.VerifyHistogramMetric(t, ctx, reader, "buffered_read/read_latency", attribute.NewSet(), uint64(1))
}

func TestReadFile_GCSReaderSequentialReadMetrics(t *testing.T) {
	testCases := []struct {
		name            string
		enableNewReader bool
	}{
		{"NewReader", true},
		{"OldReader", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			params := defaultServerConfigParams()
			params.enableNewReader = tc.enableNewReader
			bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, params)
			server = wrappers.WithMonitoring(server, mh)
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookupOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := server.LookUpInode(ctx, lookupOp)
			require.NoError(t, err, "LookUpInode")
			openOp := &fuseops.OpenFileOp{
				Inode: lookupOp.Entry.Child,
			}
			err = server.OpenFile(ctx, openOp)
			require.NoError(t, err, "OpenFile")
			readOp := &fuseops.ReadFileOp{
				Inode:  lookupOp.Entry.Child,
				Handle: openOp.Handle,
				Offset: 0,
				Dst:    make([]byte, len(content)),
			}

			err = server.ReadFile(ctx, readOp)
			waitForMetricsProcessing()

			require.NoError(t, err, "ReadFile")
			metrics.VerifyCounterMetric(t, ctx, reader, "gcs/read_count", attribute.NewSet(attribute.String("read_type", string(metrics.ReadTypeSequentialAttr))), int64(1))
			metrics.VerifyCounterMetric(t, ctx, reader, "gcs/download_bytes_count", attribute.NewSet(attribute.String("read_type", string(metrics.ReadTypeSequentialAttr))), int64(len(content)))
		})
	}
}

func TestReadFile_GCSReaderRandomReadMetrics(t *testing.T) {
	testCases := []struct {
		name            string
		enableNewReader bool
	}{
		{"NewReader", true},
		{"OldReader", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			params := defaultServerConfigParams()
			params.enableNewReader = tc.enableNewReader
			bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, params)
			server = wrappers.WithMonitoring(server, mh)
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookupOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := server.LookUpInode(ctx, lookupOp)
			require.NoError(t, err, "LookUpInode")
			openOp := &fuseops.OpenFileOp{
				Inode: lookupOp.Entry.Child,
			}
			err = server.OpenFile(ctx, openOp)
			require.NoError(t, err, "OpenFile")

			// Perform a random read at offset 10, 5, 3, and 0 in order.
			readOp := &fuseops.ReadFileOp{
				Inode:  lookupOp.Entry.Child,
				Handle: openOp.Handle,
				Offset: 10,
				Dst:    make([]byte, len(content)),
			}
			err = server.ReadFile(ctx, readOp) // Sequential read of 2 bytes (12 - 10).
			require.NoError(t, err, "ReadFile")
			readOp.Offset = 5
			err = server.ReadFile(ctx, readOp) // Sequential read of 7 bytes (12 - 5).
			require.NoError(t, err, "ReadFile")
			readOp.Offset = 3
			err = server.ReadFile(ctx, readOp) // Random read of 9 bytes (12 - 3).
			require.NoError(t, err, "ReadFile")
			readOp.Offset = 0
			err = server.ReadFile(ctx, readOp) // Random read of 12 bytes (12 - 0).
			require.NoError(t, err, "ReadFile")
			waitForMetricsProcessing()

			metrics.VerifyCounterMetric(t, ctx, reader, "gcs/read_count", attribute.NewSet(attribute.String("read_type", string(metrics.ReadTypeSequentialAttr))), int64(2))
			metrics.VerifyCounterMetric(t, ctx, reader, "gcs/read_count", attribute.NewSet(attribute.String("read_type", string(metrics.ReadTypeRandomAttr))), int64(2))
			metrics.VerifyCounterMetric(t, ctx, reader, "gcs/download_bytes_count", attribute.NewSet(attribute.String("read_type", string(metrics.ReadTypeSequentialAttr))), int64(9))
			metrics.VerifyCounterMetric(t, ctx, reader, "gcs/download_bytes_count", attribute.NewSet(attribute.String("read_type", string(metrics.ReadTypeRandomAttr))), int64(21))
		})
	}
}

func TestGetInodeAttributes_Metrics(t *testing.T) {
	ctx := context.Background()
	_, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	op := &fuseops.GetInodeAttributesOp{
		Inode: fuseops.RootInodeID,
	}

	err := server.GetInodeAttributes(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "GetInodeAttributes"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestRemoveXattr_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	createWithContents(ctx, t, bucket, fileName, "test")
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)
	op := &fuseops.RemoveXattrOp{
		Inode: lookUpOp.Entry.Child,
		Name:  "user.test",
	}

	err = server.RemoveXattr(ctx, op)
	waitForMetricsProcessing()

	// The operation is not implemented, so we expect an error.
	assert.Error(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "Others"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestListXattr_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	createWithContents(ctx, t, bucket, fileName, "test")
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)
	op := &fuseops.ListXattrOp{
		Inode: lookUpOp.Entry.Child,
	}

	err = server.ListXattr(ctx, op)
	waitForMetricsProcessing()

	// The operation is not implemented, so we expect an error.
	assert.NotNil(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "Others"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestSetXattr_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	createWithContents(ctx, t, bucket, fileName, "test")
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)
	op := &fuseops.SetXattrOp{
		Inode: lookUpOp.Entry.Child,
		Name:  "user.test",
		Value: []byte("test"),
	}

	err = server.SetXattr(ctx, op)
	waitForMetricsProcessing()

	// The operation is not implemented, so we expect an error.
	assert.NotNil(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "Others"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestGetXattr_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	createWithContents(ctx, t, bucket, fileName, "test")
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)
	op := &fuseops.GetXattrOp{
		Inode: lookUpOp.Entry.Child,
		Name:  "user.test",
	}

	err = server.GetXattr(ctx, op)
	waitForMetricsProcessing()

	// The operation is not implemented, so we expect an error.
	assert.NotNil(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "Others"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestFallocate_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	createWithContents(ctx, t, bucket, fileName, "test")
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)
	openOp := &fuseops.OpenFileOp{
		Inode: lookUpOp.Entry.Child,
	}
	err = server.OpenFile(ctx, openOp)
	require.NoError(t, err)
	op := &fuseops.FallocateOp{
		Inode:  lookUpOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 0,
		Length: 10,
		Mode:   0,
	}

	err = server.Fallocate(ctx, op)
	waitForMetricsProcessing()

	// The operation is not implemented, so we expect an error.
	assert.Error(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "Others"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestCreateLink_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	createWithContents(ctx, t, bucket, fileName, "test")
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)
	op := &fuseops.CreateLinkOp{
		Parent: fuseops.RootInodeID,
		Name:   "link",
		Target: lookUpOp.Entry.Child,
	}

	err = server.CreateLink(ctx, op)
	waitForMetricsProcessing()

	// The operation is not implemented, so we expect an error.
	assert.Error(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "CreateLink"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestStatFS_Metrics(t *testing.T) {
	ctx := context.Background()
	_, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	op := &fuseops.StatFSOp{}

	err := server.StatFS(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "Others"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestReleaseFileHandle_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	createWithContents(ctx, t, bucket, fileName, "")
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)
	openOp := &fuseops.OpenFileOp{
		Inode: lookUpOp.Entry.Child,
	}
	err = server.OpenFile(ctx, openOp)
	require.NoError(t, err)
	op := &fuseops.ReleaseFileHandleOp{
		Handle: openOp.Handle,
	}

	err = server.ReleaseFileHandle(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "ReleaseFileHandle"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestFlushFile_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	createWithContents(ctx, t, bucket, fileName, "")
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)
	openOp := &fuseops.OpenFileOp{
		Inode: lookUpOp.Entry.Child,
	}
	err = server.OpenFile(ctx, openOp)
	require.NoError(t, err)
	op := &fuseops.FlushFileOp{
		Inode:  lookUpOp.Entry.Child,
		Handle: openOp.Handle,
	}

	err = server.FlushFile(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "FlushFile"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestSyncFile_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	createWithContents(ctx, t, bucket, fileName, "")
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookUpOp)
	assert.NoError(t, err)
	op := &fuseops.SyncFileOp{
		Inode: lookUpOp.Entry.Child,
	}

	err = server.SyncFile(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "SyncFile"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestWriteFile_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	createWithContents(ctx, t, bucket, fileName, "")
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)
	openOp := &fuseops.OpenFileOp{
		Inode: lookUpOp.Entry.Child,
	}
	err = server.OpenFile(ctx, openOp)
	require.NoError(t, err)
	op := &fuseops.WriteFileOp{
		Inode:  lookUpOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 0,
		Data:   []byte("test"),
	}

	err = server.WriteFile(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "WriteFile"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestReadSymlink_Metrics(t *testing.T) {
	ctx := context.Background()
	_, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	symlinkName := "test"
	target := "target"
	createSymlinkOp := &fuseops.CreateSymlinkOp{
		Parent: fuseops.RootInodeID,
		Name:   symlinkName,
		Target: target,
	}
	err := server.CreateSymlink(ctx, createSymlinkOp)
	require.NoError(t, err)
	op := &fuseops.ReadSymlinkOp{
		Inode: createSymlinkOp.Entry.Child,
	}

	err = server.ReadSymlink(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "ReadSymlink"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestReadFile_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	content := "test content"
	createWithContents(ctx, t, bucket, fileName, content)
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)
	openOp := &fuseops.OpenFileOp{
		Inode: lookUpOp.Entry.Child,
	}
	err = server.OpenFile(ctx, openOp)
	require.NoError(t, err)
	op := &fuseops.ReadFileOp{
		Inode:  lookUpOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 0,
		Dst:    make([]byte, len(content)),
	}

	err = server.ReadFile(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "ReadFile"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestOpenFile_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	createWithContents(ctx, t, bucket, fileName, "test")
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)
	op := &fuseops.OpenFileOp{
		Inode: lookUpOp.Entry.Child,
	}

	err = server.OpenFile(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "OpenFile"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestReleaseDirHandle_Metrics(t *testing.T) {
	ctx := context.Background()
	_, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	openOp := &fuseops.OpenDirOp{
		Inode: fuseops.RootInodeID,
	}
	err := server.OpenDir(ctx, openOp)
	require.NoError(t, err)
	op := &fuseops.ReleaseDirHandleOp{
		Handle: openOp.Handle,
	}

	err = server.ReleaseDirHandle(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "ReleaseDirHandle"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestReadDirPlus_Metrics(t *testing.T) {
	ctx := context.Background()
	_, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	openOp := &fuseops.OpenDirOp{
		Inode: fuseops.RootInodeID,
	}
	err := server.OpenDir(ctx, openOp)
	require.NoError(t, err)
	op := &fuseops.ReadDirPlusOp{
		ReadDirOp: fuseops.ReadDirOp{
			Inode:  fuseops.RootInodeID,
			Handle: openOp.Handle,
			Offset: 0,
			Dst:    make([]byte, 1024),
		},
	}

	err = server.ReadDirPlus(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "ReadDirPlus"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestReadDir_Metrics(t *testing.T) {
	ctx := context.Background()
	_, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	openOp := &fuseops.OpenDirOp{
		Inode: fuseops.RootInodeID,
	}
	err := server.OpenDir(ctx, openOp)
	require.NoError(t, err)
	op := &fuseops.ReadDirOp{
		Inode:  fuseops.RootInodeID,
		Handle: openOp.Handle,
		Offset: 0,
		Dst:    make([]byte, 1024),
	}

	err = server.ReadDir(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "ReadDir"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestOpenDir_Metrics(t *testing.T) {
	ctx := context.Background()
	_, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	op := &fuseops.OpenDirOp{
		Inode: fuseops.RootInodeID,
	}

	err := server.OpenDir(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "OpenDir"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestForgetInode_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	createWithContents(ctx, t, bucket, fileName, "test")
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)
	op := &fuseops.ForgetInodeOp{
		Inode: lookUpOp.Entry.Child,
		N:     1,
	}

	err = server.ForgetInode(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "ForgetInode"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestRename_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	oldName := "old"
	newName := "new"
	createWithContents(ctx, t, bucket, oldName, "test")
	op := &fuseops.RenameOp{
		OldParent: fuseops.RootInodeID,
		OldName:   oldName,
		NewParent: fuseops.RootInodeID,
		NewName:   newName,
	}

	err := server.Rename(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "Rename"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestUnlink_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	createWithContents(ctx, t, bucket, fileName, "test")
	op := &fuseops.UnlinkOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}

	err := server.Unlink(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "Unlink"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestRmDir_Metrics(t *testing.T) {
	ctx := context.Background()
	_, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	dirName := "test"
	mkDirOp := &fuseops.MkDirOp{
		Parent: fuseops.RootInodeID,
		Name:   dirName,
	}
	err := server.MkDir(ctx, mkDirOp)
	require.NoError(t, err)
	op := &fuseops.RmDirOp{
		Parent: fuseops.RootInodeID,
		Name:   dirName,
	}

	err = server.RmDir(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "RmDir"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestCreateSymlink_Metrics(t *testing.T) {
	ctx := context.Background()
	_, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	op := &fuseops.CreateSymlinkOp{
		Parent: fuseops.RootInodeID,
		Name:   "test",
		Target: "target",
	}

	err := server.CreateSymlink(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "CreateSymlink"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestCreateFile_Metrics(t *testing.T) {
	ctx := context.Background()
	_, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	op := &fuseops.CreateFileOp{
		Parent: fuseops.RootInodeID,
		Name:   "test",
	}

	err := server.CreateFile(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "CreateFile"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestMkNode_Metrics(t *testing.T) {
	ctx := context.Background()
	_, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	op := &fuseops.MkNodeOp{
		Parent: fuseops.RootInodeID,
		Name:   "test",
	}

	err := server.MkNode(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "MkNode"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestMkDir_Metrics(t *testing.T) {
	ctx := context.Background()
	_, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	op := &fuseops.MkDirOp{
		Parent: fuseops.RootInodeID,
		Name:   "test",
	}

	err := server.MkDir(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "MkDir"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}

func TestSetInodeAttributes_Metrics(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test"
	createWithContents(ctx, t, bucket, fileName, "test")
	lookUpOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookUpOp)
	require.NoError(t, err)
	op := &fuseops.SetInodeAttributesOp{
		Inode: lookUpOp.Entry.Child,
	}

	err = server.SetInodeAttributes(ctx, op)
	waitForMetricsProcessing()

	assert.NoError(t, err)
	attrs := attribute.NewSet(attribute.String("fs_op", "SetInodeAttributes"))
	metrics.VerifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
	metrics.VerifyHistogramMetric(t, ctx, reader, "fs/ops_latency", attrs, 1)
}
