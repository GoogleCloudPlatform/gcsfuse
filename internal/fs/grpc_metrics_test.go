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
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

const (
	readObjectMethod  = "google.storage.v2.Storage/ReadObject"
	writeObjectMethod = "google.storage.v2.Storage/WriteObject"
)

// mockGrpcBucket intercepts gcs.Bucket calls to artificially emit grpc metrics
// based on standard otel grpc plugin behavior for unit testing framework testing purposes.
type mockGrpcBucket struct {
	gcs.Bucket
	started      otelmetric.Int64Counter
	callDur      otelmetric.Int64Histogram
	defaultPicks otelmetric.Int64Counter
	failedPicks  otelmetric.Int64Counter
}

func (m *mockGrpcBucket) emitGrpcMetric(ctx context.Context, method string, isFailed bool) {
	attr := otelmetric.WithAttributes(attribute.String("grpc_method", method))
	m.started.Add(ctx, 1, attr)
	m.callDur.Record(ctx, 1, attr)
	m.defaultPicks.Add(ctx, 1, attr)
	if isFailed {
		m.failedPicks.Add(ctx, 1, attr)
	}
}

func (m *mockGrpcBucket) StatObject(ctx context.Context, req *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error) {
	method := readObjectMethod
	minObj, extAttr, err := m.Bucket.StatObject(ctx, req)
	m.emitGrpcMetric(ctx, method, err != nil)
	return minObj, extAttr, err
}

func (m *mockGrpcBucket) NewReaderWithReadHandle(ctx context.Context, req *gcs.ReadObjectRequest) (gcs.StorageReader, error) {
	method := readObjectMethod
	rd, err := m.Bucket.NewReaderWithReadHandle(ctx, req)
	m.emitGrpcMetric(ctx, method, err != nil)
	return rd, err
}

func (m *mockGrpcBucket) CreateObject(ctx context.Context, req *gcs.CreateObjectRequest) (*gcs.Object, error) {
	method := writeObjectMethod
	obj, err := m.Bucket.CreateObject(ctx, req)
	m.emitGrpcMetric(ctx, method, err != nil)
	return obj, err
}

type fakeGrpcBucketManager struct {
	buckets map[string]gcs.Bucket
	meter   otelmetric.Meter
}

func (bm *fakeGrpcBucketManager) SetUpBucket(
	ctx context.Context,
	name string,
	isMultibucketMount bool,
	mh metrics.MetricHandle) (sb gcsx.SyncerBucket, err error) {
	bucket, ok := bm.buckets[name]
	if !ok {
		err = fmt.Errorf("Bucket %q does not exist", name)
		return sb, err
	}

	started, _ := bm.meter.Int64Counter("grpc_client_attempt_started")
	callDur, _ := bm.meter.Int64Histogram("grpc_client_call_duration")
	defaultPicks, _ := bm.meter.Int64Counter("grpc_lb_rls_default_target_picks")
	failedPicks, _ := bm.meter.Int64Counter("grpc_lb_rls_failed_picks")

	mockGrpcBkt := &mockGrpcBucket{
		Bucket:       bucket,
		started:      started,
		callDur:      callDur,
		defaultPicks: defaultPicks,
		failedPicks:  failedPicks,
	}

	sb = gcsx.NewSyncerBucket(
		0, 10, ".gcsfuse_tmp/",
		gcsx.NewContentTypeBucket(monitor.NewMonitoringBucket(mockGrpcBkt, mh)),
	)
	return sb, err
}

func (bm *fakeGrpcBucketManager) ShutDown() {}

func createTestFileSystemWithGrpcMetrics(ctx context.Context, t *testing.T, params *serverConfigParams) (gcs.Bucket, fuseutil.FileSystem, metrics.MetricHandle, *metric.ManualReader) {
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

	// Apply experimental flag
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
			EnableNewReader: true,
			Metrics: cfg.MetricsConfig{
				ExperimentalEnableGrpcMetrics: true,
			},
		},
		MetricHandle: mh,
		CacheClock:   &timeutil.SimulatedClock{},
		BucketName:   bucketName,
		BucketManager: &fakeGrpcBucketManager{
			buckets: map[string]gcs.Bucket{
				bucketName: bucket,
			},
			meter: provider.Meter("gcsfuse"),
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
			MaxSizeMb:                              100,
			CacheFileForRangeRead:                  true,
			ExperimentalEnableChunkCache:           params.enableSparseFileCache,
			DownloadChunkSizeMb:                    1,
			EnableParallelDownloads:                params.enableParallelDownloads,
			ExperimentalParallelDownloadsDefaultOn: params.enableParallelDownloadsBlocking,
			ParallelDownloadsPerFile:               16,
		}
	}
	if serverCfg.NewConfig.MetadataCache.TtlSecs == 0 {
		serverCfg.NewConfig.MetadataCache.TtlSecs = 60
	}

	server, err := fs.NewFileSystem(ctx, serverCfg)
	require.NoError(t, err, "NewFileSystem")
	return bucket, server, mh, reader
}

// TestGrpcMetrics_LookUpInode tests multiple gRPC metrics emitted during a LookUpInode operation.
// We verify that grpc_client_attempt_started, grpc_client_call_duration, and grpc_lb_rls_default_target_picks are recorded.
func TestGrpcMetrics_LookUpInode(t *testing.T) {
	ctx := context.Background()
	params := defaultServerConfigParams()

	bucket, server, mh, reader := createTestFileSystemWithGrpcMetrics(ctx, t, params)
	server = wrappers.WithMonitoring(server, mh)

	fileName := "test.txt"
	createWithContents(ctx, t, bucket, fileName, "test content")

	lookupOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}

	err := server.LookUpInode(ctx, lookupOp)
	require.NoError(t, err)
	waitForMetricsProcessing()

	t.Run("grpc_client_attempt_started", func(t *testing.T) {
		// 3 started attempts due to LookUpInode making StatObject calls (1 dir check + 1 file check + 1 attribute refresh)
		metrics.VerifyCounterMetric(t, ctx, reader, "grpc_client_attempt_started",
			attribute.NewSet(attribute.String("grpc_method", readObjectMethod)),
			3)
	})

	t.Run("grpc_client_call_duration", func(t *testing.T) {
		// 3 histogram records
		metrics.VerifyHistogramMetric(t, ctx, reader, "grpc_client_call_duration",
			attribute.NewSet(attribute.String("grpc_method", readObjectMethod)),
			3)
	})

	t.Run("grpc_lb_rls_default_target_picks", func(t *testing.T) {
		// 3 default component picks
		metrics.VerifyCounterMetric(t, ctx, reader, "grpc_lb_rls_default_target_picks",
			attribute.NewSet(attribute.String("grpc_method", readObjectMethod)),
			3)
	})
}

// TestGrpcMetrics_LbRlsFailedPicks tests the grpc_lb_rls_failed_picks metric.
// We verify that the grpc_lb_rls_failed_picks is correctly emitted when picks fail.
func TestGrpcMetrics_LbRlsFailedPicks(t *testing.T) {
	ctx := context.Background()
	params := defaultServerConfigParams()

	_, server, mh, reader := createTestFileSystemWithGrpcMetrics(ctx, t, params)
	server = wrappers.WithMonitoring(server, mh)

	// Trigger a lookup for a non-existing object, which can cause pick fallback or failure scenario in GRPC.
	lookupOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   "non_existent.txt",
	}

	// This operation is expected to fail or hit fallback
	err := server.LookUpInode(ctx, lookupOp)
	require.Error(t, err)
	waitForMetricsProcessing()

	// LookUpInode on a non-existent file triggers a failed StatObject for directory prefix, then a failed StatObject for the file itself.
	// Therefore, we get 2 failures.
	metrics.VerifyCounterMetric(t, ctx, reader, "grpc_lb_rls_failed_picks",
		attribute.NewSet(attribute.String("grpc_method", readObjectMethod)),
		2)
}

// TestGrpcMetrics_FileCache_Read tests gRPC metrics under file cache scenario.
// Ensures that reading objects directly contributes to the gRPC call statistics.
func TestGrpcMetrics_FileCache_Read(t *testing.T) {
	ctx := context.Background()
	params := defaultServerConfigParams()
	params.enableFileCache = true

	bucket, server, mh, reader := createTestFileSystemWithGrpcMetrics(ctx, t, params)
	server = wrappers.WithMonitoring(server, mh)

	fileName := "test_cache.txt"
	content := "test content for caching!"
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

	// For a cache miss download, NewReaderWithReadHandle is triggered which contributes to grpc metrics.
	// 3 for Stat (LookUp) + 1 for NewReader
	metrics.VerifyCounterMetric(t, ctx, reader, "grpc_client_attempt_started",
		attribute.NewSet(attribute.String("grpc_method", readObjectMethod)),
		4)
}

// TestGrpcMetrics_CreateObject validates gRPC metrics for writing files.
func TestGrpcMetrics_CreateObject(t *testing.T) {
	ctx := context.Background()
	params := defaultServerConfigParams()

	_, server, mh, reader := createTestFileSystemWithGrpcMetrics(ctx, t, params)
	server = wrappers.WithMonitoring(server, mh)

	fileName := "new_file.txt"

	// CreateFile
	createOp := &fuseops.CreateFileOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
		Mode:   0644,
	}
	err := server.CreateFile(ctx, createOp)
	require.NoError(t, err)

	// Sync or Close triggers upload
	syncOp := &fuseops.SyncFileOp{
		Inode:  createOp.Entry.Child,
		Handle: createOp.Handle,
	}
	err = server.SyncFile(ctx, syncOp)
	require.NoError(t, err)
	waitForMetricsProcessing()

	// 1 for WriteObject
	metrics.VerifyCounterMetric(t, ctx, reader, "grpc_client_attempt_started",
		attribute.NewSet(attribute.String("grpc_method", writeObjectMethod)),
		1)
}
