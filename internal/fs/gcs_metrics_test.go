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
	"fmt"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/wrappers"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
)

type fakeBucketManagerWithMetrics struct {
	buckets map[string]gcs.Bucket
}

func (bm *fakeBucketManagerWithMetrics) SetUpBucket(
	ctx context.Context,
	name string,
	isMultibucketMount bool,
	mh metrics.MetricHandle) (sb gcsx.SyncerBucket, err error) {
	bucket, ok := bm.buckets[name]
	if !ok {
		err = fmt.Errorf("Bucket %q does not exist", name)
		return
	}

	// Wrap bucket with monitor.NewMonitoringBucket to enable GCS metrics.
	sb = gcsx.NewSyncerBucket(
		0, 10, ".gcsfuse_tmp/",
		gcsx.NewContentTypeBucket(monitor.NewMonitoringBucket(bucket, mh)),
	)
	return
}

func (bm *fakeBucketManagerWithMetrics) ShutDown() {}

func createTestFileSystemWithMonitoredBucket(ctx context.Context, t *testing.T, params *serverConfigParams) (gcs.Bucket, fuseutil.FileSystem, metrics.MetricHandle, *metric.ManualReader) {
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
		BucketManager: &fakeBucketManagerWithMetrics{
			buckets: map[string]gcs.Bucket{
				bucketName: bucket,
			},
		},
		SequentialReadSizeMb: 200,
	}

	if params.enableFileCache || params.enableSparseFileCache {
		cacheDir := t.TempDir()
		t.Cleanup(func() {
			os.RemoveAll(cacheDir)
		})
		serverCfg.NewConfig.CacheDir = cfg.ResolvedPath(cacheDir)
		serverCfg.NewConfig.FileCache = cfg.FileCacheConfig{
			MaxSizeMb:                    100,
			CacheFileForRangeRead:        true,
			ExperimentalEnableBlockCache: params.enableSparseFileCache,
			DownloadChunkSizeMb:          1, // 1MB chunks for testing
		}
	}

	server, err := fs.NewFileSystem(ctx, serverCfg)
	require.NoError(t, err, "NewFileSystem")
	return bucket, server, mh, reader
}

func TestGCSMetrics_RequestCount_StatObject(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMonitoredBucket(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test.txt"
	createWithContents(ctx, t, bucket, fileName, "test")

	lookupOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}

	err := server.LookUpInode(ctx, lookupOp)
	require.NoError(t, err)

	waitForMetricsProcessing()

	// LookUpInode typically triggers StatObject on the object name.
	// It triggers multiple stats (e.g. checking for implicit directories).
	metrics.VerifyCounterMetric(t, ctx, reader, "gcs/request_count",
		attribute.NewSet(attribute.String("gcs_method", "StatObject")),
		3)
}

func TestGCSMetrics_RequestCount_CreateObject(t *testing.T) {
	ctx := context.Background()
	_, server, mh, reader := createTestFileSystemWithMonitoredBucket(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "new_file.txt"

	// CreateFile -> CreateObject
	createOp := &fuseops.CreateFileOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
		Mode:   0644,
	}

	err := server.CreateFile(ctx, createOp)
	require.NoError(t, err)

	// Sync or Close to trigger upload to GCS
	syncOp := &fuseops.SyncFileOp{
		Inode:  createOp.Entry.Child,
		Handle: createOp.Handle,
	}
	err = server.SyncFile(ctx, syncOp)
	require.NoError(t, err)

	waitForMetricsProcessing()

	metrics.VerifyCounterMetric(t, ctx, reader, "gcs/request_count",
		attribute.NewSet(attribute.String("gcs_method", "CreateObject")),
		1)
}

func TestGCSMetrics_RequestLatencies(t *testing.T) {
	ctx := context.Background()
	bucket, server, mh, reader := createTestFileSystemWithMonitoredBucket(ctx, t, defaultServerConfigParams())
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test.txt"
	createWithContents(ctx, t, bucket, fileName, "test")

	lookupOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}

	err := server.LookUpInode(ctx, lookupOp)
	require.NoError(t, err)

	waitForMetricsProcessing()

	metrics.VerifyHistogramMetric(t, ctx, reader, "gcs/request_latencies",
		attribute.NewSet(attribute.String("gcs_method", "StatObject")),
		3)
}

func TestGCSMetrics_DownloadBytesCount_Explicit(t *testing.T) {
	// Explicitly test gcs/download_bytes_count using monitored bucket.
	ctx := context.Background()
	params := defaultServerConfigParams()
	params.enableBufferedRead = true
	bucket, server, mh, reader := createTestFileSystemWithMonitoredBucket(ctx, t, params)
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test.txt"
	content := "1234567890"
	createWithContents(ctx, t, bucket, fileName, content)

	lookupOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	err := server.LookUpInode(ctx, lookupOp)
	require.NoError(t, err)
	openOp := &fuseops.OpenFileOp{
		Inode: lookupOp.Entry.Child,
	}
	err = server.OpenFile(ctx, openOp)
	require.NoError(t, err)

	readOp := &fuseops.ReadFileOp{
		Inode:  lookupOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 0,
		Dst:    make([]byte, len(content)),
	}

	err = server.ReadFile(ctx, readOp)
	require.NoError(t, err)

	waitForMetricsProcessing()

	metrics.VerifyCounterMetric(t, ctx, reader, "gcs/download_bytes_count",
		attribute.NewSet(attribute.String("read_type", string(metrics.ReadTypeBufferedAttr))),
		int64(len(content)))
}
