// Copyright 2024 Google LLC
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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/wrappers"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
)

// A fake implementation of gcsx.BucketManager.
type fakeBucketManager struct {
	gcsx.BucketManager
	buckets map[string]gcs.Bucket
}

func (bm *fakeBucketManager) ShutDown() {}
func (bm *fakeBucketManager) SetUpBucket(
	ctx context.Context,
	name string,
	skipCheck bool,
	metricHandle metrics.MetricHandle) (b gcsx.SyncerBucket, err error) {
	gcsBucket, ok := bm.buckets[name]
	if !ok {
		err = fmt.Errorf("Uknown bucket: %s", name)
	}
	b = gcsx.NewSyncerBucket(
		1,
		10,
		".gcsfuse_tmp/",
		gcsBucket)
	return
}

func createTestFileSystemWithFileCacheMetrics(ctx context.Context, t *testing.T, cacheDir string) (gcs.Bucket, fuseutil.FileSystem, metrics.MetricHandle, *metric.ManualReader) {
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
			CacheDir: cfg.ResolvedPath(cacheDir),
			FileCache: cfg.FileCacheConfig{
				MaxSizeMb:             100,
				CacheFileForRangeRead: true,
			},
			Write: cfg.WriteConfig{
				GlobalMaxBlocks: 1,
			},
			Read: cfg.ReadConfig{
				GlobalMaxBlocks:    1,
				BlockSizeMb:        1,
				MaxBlocksPerHandle: 10,
			},
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

func TestReadFile_SequentialReadMetrics(t *testing.T) {
	ctx := context.Background()
	cacheDir := t.TempDir()
	bucket, server, mh, reader := createTestFileSystemWithFileCacheMetrics(ctx, t, cacheDir)
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

	// First read should be a cache miss.
	err = server.ReadFile(ctx, readOp)
	waitForMetricsProcessing()

	require.NoError(t, err, "ReadFile")
	attrs := attribute.NewSet(attribute.String("read_type", "Sequential"), attribute.Bool("cache_hit", false))
	metrics.VerifyCounterMetric(t, ctx, reader, "file_cache/read_count", attrs, 1)
	attrs = attribute.NewSet(attribute.Bool("cache_hit", false))
	metrics.VerifyHistogramMetric(t, ctx, reader, "file_cache/read_latencies", attrs, 1)

	// Second read should be a cache hit.
	err = server.ReadFile(ctx, readOp)
	waitForMetricsProcessing()

	require.NoError(t, err, "ReadFile")
	// The read_bytes_count is cumulative. The first read (cache miss) reads from
	// the backend and writes to the cache, and the second read (cache hit) reads
	// from the cache. Both reads contribute to the read_bytes_count.
	attrs = attribute.NewSet(attribute.String("read_type", "Sequential"))
	metrics.VerifyCounterMetric(t, ctx, reader, "file_cache/read_bytes_count", attrs, int64(2*len(content)))
	attrs = attribute.NewSet(attribute.String("read_type", "Sequential"), attribute.Bool("cache_hit", true))
	metrics.VerifyCounterMetric(t, ctx, reader, "file_cache/read_count", attrs, 1)
	attrs = attribute.NewSet(attribute.Bool("cache_hit", true))
	metrics.VerifyHistogramMetric(t, ctx, reader, "file_cache/read_latencies", attrs, 1)
}

func TestReadFile_RandomReadMetrics(t *testing.T) {
	ctx := context.Background()
	cacheDir := t.TempDir()
	bucket, server, mh, reader := createTestFileSystemWithFileCacheMetrics(ctx, t, cacheDir)
	server = wrappers.WithMonitoring(server, mh)
	fileName := "test.txt"
	content := "some large content for testing random reads in the cache."
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
	readSize := 25
	readOp := &fuseops.ReadFileOp{
		Inode:  lookupOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 10,
		Dst:    make([]byte, readSize),
	}
	// First read should be a cache miss.
	err = server.ReadFile(ctx, readOp)
	waitForMetricsProcessing()

	require.NoError(t, err, "ReadFile")
	attrs := attribute.NewSet(attribute.String("read_type", "Random"), attribute.Bool("cache_hit", false))
	metrics.VerifyCounterMetric(t, ctx, reader, "file_cache/read_count", attrs, 1)
	attrs = attribute.NewSet(attribute.Bool("cache_hit", false))
	metrics.VerifyHistogramMetric(t, ctx, reader, "file_cache/read_latencies", attrs, 1)

	// Second read should be a cache hit.
	err = server.ReadFile(ctx, readOp)
	waitForMetricsProcessing()

	require.NoError(t, err, "ReadFile")
	attrs = attribute.NewSet(attribute.String("read_type", "Random"), attribute.Bool("cache_hit", true))
	metrics.VerifyCounterMetric(t, ctx, reader, "file_cache/read_count", attrs, 1)
	attrs = attribute.NewSet(attribute.String("read_type", "Random"))
	metrics.VerifyCounterMetric(t, ctx, reader, "file_cache/read_bytes_count", attrs, int64(readSize))
	attrs = attribute.NewSet(attribute.Bool("cache_hit", true))
	metrics.VerifyHistogramMetric(t, ctx, reader, "file_cache/read_latencies", attrs, 1)
}
