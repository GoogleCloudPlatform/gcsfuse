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
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/wrappers"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
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

func createTestFileSystemWithMetadataCache(
	ctx context.Context,
	t *testing.T,
	ttl time.Duration,
	clock timeutil.Clock,
) (gcs.Bucket, fuseutil.FileSystem, metrics.MetricHandle, *metric.ManualReader) {
	t.Helper()
	origProvider := otel.GetMeterProvider()
	t.Cleanup(func() { otel.SetMeterProvider(origProvider) })
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)

	mh, err := metrics.NewOTelMetrics(ctx, 1, 100)
	require.NoError(t, err, "metrics.NewOTelMetrics")
	bucketName := "test-bucket"
	bucketType := gcs.BucketType{Hierarchical: false}
	uncachedBucket := fake.NewFakeBucket(clock, bucketName, bucketType)

	lruCache := lru.NewCache(uint64(1000 * cfg.AverageSizeOfPositiveStatCacheEntry))
	statCache := metadata.NewStatCacheBucketView(lruCache, "")
	cachedBucket := caching.NewFastStatBucket(
		ttl,
		statCache,
		clock,
		uncachedBucket,
		ttl,
		true,
		true,
		false,
	)

	serverCfg := &fs.ServerConfig{
		NewConfig: &cfg.Config{
			EnableTypeCacheDeprecation: true,
			MetadataCache: cfg.MetadataCacheConfig{
				TtlSecs:                int64(ttl.Seconds()),
				TypeCacheMaxSizeMb:     4,
				StatCacheMaxSizeMb:     32,
				NegativeTtlSecs:        int64(ttl.Seconds()),
				EnableMetadataPrefetch:  false,
			},
		},
		MetricHandle: mh,
		TraceHandle:  tracing.NewNoopTracer(),
		CacheClock:   clock,
		BucketName:   bucketName,
		BucketManager: &fakeBucketManagerWithMetrics{
			buckets: map[string]gcs.Bucket{
				bucketName: cachedBucket,
			},
		},
	}

	server, err := fs.NewFileSystem(ctx, serverCfg)
	require.NoError(t, err, "NewFileSystem")
	return uncachedBucket, server, mh, reader
}

func verifyMetadataCacheMetric(
	t *testing.T,
	ctx context.Context,
	reader *metric.ManualReader,
	cacheHit bool,
	entryStatus metrics.EntryStatus,
	lookupDetail metrics.LookupDetail,
	expected int64,
) {
	t.Helper()
	var attrs attribute.Set
	if entryStatus == metrics.EntryStatusAttr {
		attrs = attribute.NewSet(
			attribute.Bool("cache_hit", cacheHit),
			attribute.String("lookup_detail", string(lookupDetail)),
		)
	} else {
		attrs = attribute.NewSet(
			attribute.Bool("cache_hit", cacheHit),
			attribute.String("entry_status", string(entryStatus)),
			attribute.String("lookup_detail", string(lookupDetail)),
		)
	}
	metrics.VerifyCounterMetric(t, ctx, reader, "metadata_cache/read_count", attrs, expected)
}

func TestMetadataCache_EndToEndScenarios(t *testing.T) {
	ctx := context.Background()
	clock := &timeutil.SimulatedClock{}
	clock.SetTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	ttl := 120 * time.Second
	bucket, server, mh, reader := createTestFileSystemWithMetadataCache(ctx, t, ttl, clock)
	server = wrappers.WithMonitoring(server, mh)

	// Scenario 1: Cold Nonexistent Stat
	// LookUpInode("nonexistent.txt") -> miss (not_found)
	t.Run("Scenario 1: Cold Nonexistent Stat", func(t *testing.T) {
		op := &fuseops.LookUpInodeOp{
			Parent: fuseops.RootInodeID,
			Name:   "nonexistent.txt",
		}
		err := server.LookUpInode(ctx, op)
		assert.Equal(t, fuse.ENOENT, err)
		waitForMetricsProcessing()

		verifyMetadataCacheMetric(t, ctx, reader, false, metrics.EntryStatusAttr, metrics.LookupDetailNotFoundAttr, 1)
	})

	// Scenario 2: Warm Nonexistent Stat
	// LookUpInode("nonexistent.txt") -> hit negative entry (negative, found)
	t.Run("Scenario 2: Warm Nonexistent Stat", func(t *testing.T) {
		op := &fuseops.LookUpInodeOp{
			Parent: fuseops.RootInodeID,
			Name:   "nonexistent.txt",
		}
		err := server.LookUpInode(ctx, op)
		assert.Equal(t, fuse.ENOENT, err)
		waitForMetricsProcessing()

		verifyMetadataCacheMetric(t, ctx, reader, true, metrics.EntryStatusNegativeAttr, metrics.LookupDetailFoundAttr, 1)
	})

	// Scenario 3: External GCS Create
	// Create object in bucket directly -> 0 FUSE calls
	t.Run("Scenario 3: External GCS Create", func(t *testing.T) {
		err := storageutil.CreateObjects(ctx, bucket, map[string][]byte{"existing_file.txt": []byte("hello world")})
		require.NoError(t, err)
		waitForMetricsProcessing()

		// Verify no additional metric increments
		verifyMetadataCacheMetric(t, ctx, reader, false, metrics.EntryStatusAttr, metrics.LookupDetailNotFoundAttr, 1)
	})

	// Scenario 4: Cold Stat on Existing File
	// LookUpInode("existing_file.txt") -> miss (not_found), then caches positive entry
	var existingInodeID fuseops.InodeID
	t.Run("Scenario 4: Cold Stat on Existing File", func(t *testing.T) {
		op := &fuseops.LookUpInodeOp{
			Parent: fuseops.RootInodeID,
			Name:   "existing_file.txt",
		}
		err := server.LookUpInode(ctx, op)
		require.NoError(t, err)
		existingInodeID = op.Entry.Child
		waitForMetricsProcessing()

		// Delta +1 for cold miss
		verifyMetadataCacheMetric(t, ctx, reader, false, metrics.EntryStatusAttr, metrics.LookupDetailNotFoundAttr, 2)
	})

	// Scenario 5: Warm Stat on Existing File
	// LookUpInode("existing_file.txt") -> hit positive entry (positive, found)
	t.Run("Scenario 5: Warm Stat on Existing File", func(t *testing.T) {
		op := &fuseops.LookUpInodeOp{
			Parent: fuseops.RootInodeID,
			Name:   "existing_file.txt",
		}
		err := server.LookUpInode(ctx, op)
		require.NoError(t, err)
		waitForMetricsProcessing()

		// Delta +1 for warm positive hit
		verifyMetadataCacheMetric(t, ctx, reader, true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr, 1)
	})

	// Scenario 6: Inode Attributes (fstat) on Existing File
	// GetInodeAttributes(existingInodeID) -> hit positive entry (positive, found)
	t.Run("Scenario 6: Inode Attributes on Existing File", func(t *testing.T) {
		op := &fuseops.GetInodeAttributesOp{
			Inode: existingInodeID,
		}
		err := server.GetInodeAttributes(ctx, op)
		require.NoError(t, err)
		waitForMetricsProcessing()

		// Delta +1 for positive hit during clobbered check
		verifyMetadataCacheMetric(t, ctx, reader, true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr, 2)
	})

	// Scenario 7: Stat Cache TTL Expiration
	// Advance clock past TTL -> LookUpInode("existing_file.txt") -> miss (ttl_expired)
	t.Run("Scenario 7: Stat Cache TTL Expiration", func(t *testing.T) {
		clock.AdvanceTime(ttl + 10*time.Second)

		op := &fuseops.LookUpInodeOp{
			Parent: fuseops.RootInodeID,
			Name:   "existing_file.txt",
		}
		err := server.LookUpInode(ctx, op)
		require.NoError(t, err)
		waitForMetricsProcessing()

		// Delta +1 for TTL expired
		verifyMetadataCacheMetric(t, ctx, reader, false, metrics.EntryStatusPositiveAttr, metrics.LookupDetailTtlExpiredAttr, 1)
	})

	// Scenario 8: Directory Creation & Lookup
	t.Run("Scenario 8: Directory Creation & Lookup", func(t *testing.T) {
		// MkDir "new_dir" -> No duplicate lookup counts
		mkdirOp := &fuseops.MkDirOp{
			Parent: fuseops.RootInodeID,
			Name:   "new_dir",
			Mode:   0755,
		}
		err := server.MkDir(ctx, mkdirOp)
		require.NoError(t, err)
		waitForMetricsProcessing()

		// Cold lookup of "new_dir" (after advance/mint) -> hit positive entry
		lookupOp := &fuseops.LookUpInodeOp{
			Parent: fuseops.RootInodeID,
			Name:   "new_dir",
		}
		err = server.LookUpInode(ctx, lookupOp)
		require.NoError(t, err)
		waitForMetricsProcessing()

		// Positive hit count incremented
		verifyMetadataCacheMetric(t, ctx, reader, true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr, 3)
	})

	// Scenario 9: Directory Deletion & Negative Lookup
	t.Run("Scenario 9: Directory Deletion & Negative Lookup", func(t *testing.T) {
		rmdirOp := &fuseops.RmDirOp{
			Parent: fuseops.RootInodeID,
			Name:   "new_dir",
		}
		err := server.RmDir(ctx, rmdirOp)
		require.NoError(t, err)
		waitForMetricsProcessing()

		// Warm lookup of deleted dir -> hit negative entry
		lookupOp := &fuseops.LookUpInodeOp{
			Parent: fuseops.RootInodeID,
			Name:   "new_dir",
		}
		err = server.LookUpInode(ctx, lookupOp)
		assert.Equal(t, fuse.ENOENT, err)
		waitForMetricsProcessing()

		verifyMetadataCacheMetric(t, ctx, reader, true, metrics.EntryStatusNegativeAttr, metrics.LookupDetailFoundAttr, 2)
	})

	// Scenario 10: File Creation & Warm Lookup
	t.Run("Scenario 10: File Creation & Warm Lookup", func(t *testing.T) {
		createOp := &fuseops.CreateFileOp{
			Parent: fuseops.RootInodeID,
			Name:   "new_file.txt",
			Mode:   0644,
		}
		err := server.CreateFile(ctx, createOp)
		require.NoError(t, err)
		waitForMetricsProcessing()
		t.Logf("After CreateFile")

		writeOp := &fuseops.WriteFileOp{
			Inode:  createOp.Entry.Child,
			Handle: createOp.Handle,
			Data:   []byte("test content"),
			Offset: 0,
		}
		err = server.WriteFile(ctx, writeOp)
		require.NoError(t, err)
		waitForMetricsProcessing()
		t.Logf("After WriteFile")

		syncOp := &fuseops.SyncFileOp{
			Inode:  createOp.Entry.Child,
			Handle: createOp.Handle,
		}
		err = server.SyncFile(ctx, syncOp)
		require.NoError(t, err)
		waitForMetricsProcessing()
		t.Logf("After SyncFile")

		// Warm lookup of created file -> hit positive entry
		lookupOp := &fuseops.LookUpInodeOp{
			Parent: fuseops.RootInodeID,
			Name:   "new_file.txt",
		}
		err = server.LookUpInode(ctx, lookupOp)
		require.NoError(t, err)
		waitForMetricsProcessing()
		t.Logf("After LookUpInode")

		verifyMetadataCacheMetric(t, ctx, reader, true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr, 5)
	})

	// Scenario 11: File Deletion & Negative Lookup
	t.Run("Scenario 11: File Deletion & Negative Lookup", func(t *testing.T) {
		unlinkOp := &fuseops.UnlinkOp{
			Parent: fuseops.RootInodeID,
			Name:   "new_file.txt",
		}
		err := server.Unlink(ctx, unlinkOp)
		require.NoError(t, err)
		waitForMetricsProcessing()

		// Warm lookup of unlinked file -> hit negative entry
		lookupOp := &fuseops.LookUpInodeOp{
			Parent: fuseops.RootInodeID,
			Name:   "new_file.txt",
		}
		err = server.LookUpInode(ctx, lookupOp)
		assert.Equal(t, fuse.ENOENT, err)
		waitForMetricsProcessing()

		verifyMetadataCacheMetric(t, ctx, reader, true, metrics.EntryStatusNegativeAttr, metrics.LookupDetailFoundAttr, 3)
	})

	// Scenario 12: SetInodeAttributes (Truncate)
	t.Run("Scenario 12: SetInodeAttributes Truncate", func(t *testing.T) {
		newSize := uint64(5)
		setOp := &fuseops.SetInodeAttributesOp{
			Inode: existingInodeID,
			Size:  &newSize,
		}
		err := server.SetInodeAttributes(ctx, setOp)
		require.NoError(t, err)
		waitForMetricsProcessing()

		// Positive hit count incremented via clobbered check
		verifyMetadataCacheMetric(t, ctx, reader, true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr, 6)
	})

	// Scenario 13: Symlink Creation & Lookup
	t.Run("Scenario 13: Symlink Creation & Lookup", func(t *testing.T) {
		symlinkOp := &fuseops.CreateSymlinkOp{
			Parent: fuseops.RootInodeID,
			Name:   "symlink_target",
			Target: "existing_file.txt",
		}
		err := server.CreateSymlink(ctx, symlinkOp)
		require.NoError(t, err)
		waitForMetricsProcessing()

		// Warm lookup of symlink -> positive hit
		lookupOp := &fuseops.LookUpInodeOp{
			Parent: fuseops.RootInodeID,
			Name:   "symlink_target",
		}
		err = server.LookUpInode(ctx, lookupOp)
		require.NoError(t, err)
		waitForMetricsProcessing()

		verifyMetadataCacheMetric(t, ctx, reader, true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr, 7)
	})

	// Scenario 14: Negative TTL Expiration
	t.Run("Scenario 14: Negative TTL Expiration", func(t *testing.T) {
		clock.AdvanceTime(ttl + 10*time.Second)

		// LookUpInode("nonexistent.txt") after negative TTL expiration -> miss (ttl_expired)
		op := &fuseops.LookUpInodeOp{
			Parent: fuseops.RootInodeID,
			Name:   "nonexistent.txt",
		}
		err := server.LookUpInode(ctx, op)
		assert.Equal(t, fuse.ENOENT, err)
		waitForMetricsProcessing()

		verifyMetadataCacheMetric(t, ctx, reader, false, metrics.EntryStatusNegativeAttr, metrics.LookupDetailTtlExpiredAttr, 1)
	})

	// Scenario 15: Open Existing File & Warm Lookup
	t.Run("Scenario 15: Open Existing File & Warm Lookup", func(t *testing.T) {
		// First lookup refreshes expired cache entry from Scenario 14
		lookupOp := &fuseops.LookUpInodeOp{
			Parent: fuseops.RootInodeID,
			Name:   "existing_file.txt",
		}
		err := server.LookUpInode(ctx, lookupOp)
		require.NoError(t, err)

		// Second lookup hits the refreshed warm cache
		err = server.LookUpInode(ctx, lookupOp)
		require.NoError(t, err)
		waitForMetricsProcessing()

		verifyMetadataCacheMetric(t, ctx, reader, true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr, 8)
	})

	// Scenario 16: ReadDirPlus does not inflate clobbered counts
	t.Run("Scenario 16: ReadDirPlus does not inflate clobbered counts", func(t *testing.T) {
		openDirOp := &fuseops.OpenDirOp{
			Inode: fuseops.RootInodeID,
		}
		err := server.OpenDir(ctx, openDirOp)
		require.NoError(t, err)

		readDirOp := &fuseops.ReadDirOp{
			Inode:  fuseops.RootInodeID,
			Handle: openDirOp.Handle,
			Dst:    make([]byte, 4096),
		}
		err = server.ReadDir(ctx, readDirOp)
		require.NoError(t, err)
		waitForMetricsProcessing()

		// Verify positive hit count did not increase from listing
		verifyMetadataCacheMetric(t, ctx, reader, true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr, 8)
	})

	// Scenario 17: Nested File Cold Stat
	var subDirInodeID fuseops.InodeID
	t.Run("Scenario 17: Nested File Cold Stat", func(t *testing.T) {
		mkdirOp := &fuseops.MkDirOp{
			Parent: fuseops.RootInodeID,
			Name:   "nested_dir",
			Mode:   0755,
		}
		err := server.MkDir(ctx, mkdirOp)
		require.NoError(t, err)
		subDirInodeID = mkdirOp.Entry.Child

		// Create object nested_dir/child.txt in bucket
		err = storageutil.CreateObjects(ctx, bucket, map[string][]byte{"nested_dir/child.txt": []byte("nested content")})
		require.NoError(t, err)
		waitForMetricsProcessing()

		// Cold lookup of child.txt in nested_dir -> miss (not_found)
		lookupOp := &fuseops.LookUpInodeOp{
			Parent: subDirInodeID,
			Name:   "child.txt",
		}
		err = server.LookUpInode(ctx, lookupOp)
		require.NoError(t, err)
		waitForMetricsProcessing()

		verifyMetadataCacheMetric(t, ctx, reader, false, metrics.EntryStatusAttr, metrics.LookupDetailNotFoundAttr, 3)
	})

	// Scenario 18: Nested File Warm Stat
	t.Run("Scenario 18: Nested File Warm Stat", func(t *testing.T) {
		lookupOp := &fuseops.LookUpInodeOp{
			Parent: subDirInodeID,
			Name:   "child.txt",
		}
		err := server.LookUpInode(ctx, lookupOp)
		require.NoError(t, err)
		waitForMetricsProcessing()

		verifyMetadataCacheMetric(t, ctx, reader, true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr, 9)
	})

	// Scenario 19: Nested Nonexistent Stat (Cold)
	t.Run("Scenario 19: Nested Nonexistent Stat Cold", func(t *testing.T) {
		lookupOp := &fuseops.LookUpInodeOp{
			Parent: subDirInodeID,
			Name:   "nonexistent_child.txt",
		}
		err := server.LookUpInode(ctx, lookupOp)
		assert.Equal(t, fuse.ENOENT, err)
		waitForMetricsProcessing()

		verifyMetadataCacheMetric(t, ctx, reader, false, metrics.EntryStatusAttr, metrics.LookupDetailNotFoundAttr, 4)
	})

	// Scenario 20: Nested Nonexistent Stat (Warm Negative Hit)
	t.Run("Scenario 20: Nested Nonexistent Stat Warm", func(t *testing.T) {
		lookupOp := &fuseops.LookUpInodeOp{
			Parent: subDirInodeID,
			Name:   "nonexistent_child.txt",
		}
		err := server.LookUpInode(ctx, lookupOp)
		assert.Equal(t, fuse.ENOENT, err)
		waitForMetricsProcessing()

		verifyMetadataCacheMetric(t, ctx, reader, true, metrics.EntryStatusNegativeAttr, metrics.LookupDetailFoundAttr, 4)
	})

	// Scenario 21: Conflicting Filename Lookup
	t.Run("Scenario 21: Conflicting Filename Lookup", func(t *testing.T) {
		// Lookup for a conflicting name suffix
		lookupOp := &fuseops.LookUpInodeOp{
			Parent: fuseops.RootInodeID,
			Name:   "conflict\n",
		}
		err := server.LookUpInode(ctx, lookupOp)
		assert.Equal(t, fuse.ENOENT, err)
		waitForMetricsProcessing()
	})

	// Scenario 22: Unlinked local file Attributes
	t.Run("Scenario 22: Unlinked local file Attributes", func(t *testing.T) {
		createOp := &fuseops.CreateFileOp{
			Parent: fuseops.RootInodeID,
			Name:   "local_unlink.txt",
			Mode:   0644,
		}
		err := server.CreateFile(ctx, createOp)
		require.NoError(t, err)

		unlinkOp := &fuseops.UnlinkOp{
			Parent: fuseops.RootInodeID,
			Name:   "local_unlink.txt",
		}
		err = server.Unlink(ctx, unlinkOp)
		require.NoError(t, err)

		// GetInodeAttributes on unlinked file should succeed with nlink=0 without extra cache increments
		getOp := &fuseops.GetInodeAttributesOp{
			Inode: createOp.Entry.Child,
		}
		err = server.GetInodeAttributes(ctx, getOp)
		require.NoError(t, err)
		assert.Equal(t, uint32(0), getOp.Attributes.Nlink)
	})
}
