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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/wrappers"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
)

const metadataCacheTTL = 120 * time.Second

// metadataCacheFSOptions configures newProductionLikeMetadataCacheFS.
type metadataCacheFSOptions struct {
	// ttl of the metadata cache entries. Zero disables the metadata cache.
	ttl                       time.Duration
	implicitDirs              bool
	enableEmptyManagedFolders bool
}

// newProductionLikeMetadataCacheFS builds a file system wrapped with
// monitoring and with ignore-interrupts on, as gcsfuse runs by default. Its
// metadata cache is set up from opts as the bucket manager does it.
func newProductionLikeMetadataCacheFS(ctx context.Context, t *testing.T, opts metadataCacheFSOptions) *metadataCacheTestEnv {
	t.Helper()
	origProvider := otel.GetMeterProvider()
	t.Cleanup(func() { otel.SetMeterProvider(origProvider) })
	reader := metric.NewManualReader()
	otel.SetMeterProvider(metric.NewMeterProvider(metric.WithReader(reader)))
	mh, err := metrics.NewOTelMetrics(ctx, 1, 100)
	require.NoError(t, err, "metrics.NewOTelMetrics")

	clock := &timeutil.SimulatedClock{}
	clock.SetTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	const bucketName = "test-bucket"
	uncachedBucket := fake.NewFakeBucket(clock, bucketName, gcs.BucketType{})
	// Like the bucket manager, only put the stat cache in front of the bucket
	// when the metadata cache is enabled.
	var bucket gcs.Bucket = uncachedBucket
	if opts.ttl != 0 {
		statCache := metadata.NewStatCacheBucketView(lru.NewCache(uint64(1000*cfg.AverageSizeOfPositiveStatCacheEntry)), "")
		bucket = caching.NewFastStatBucket(opts.ttl, statCache, clock, uncachedBucket, opts.ttl, true, opts.implicitDirs, opts.enableEmptyManagedFolders)
	}

	server, err := fs.NewFileSystem(ctx, &fs.ServerConfig{
		NewConfig: &cfg.Config{
			EnableTypeCacheDeprecation: true,
			FileSystem:                 cfg.FileSystemConfig{IgnoreInterrupts: true},
			ImplicitDirs:               opts.implicitDirs,
			List:                       cfg.ListConfig{EnableEmptyManagedFolders: opts.enableEmptyManagedFolders},
			MetadataCache: cfg.MetadataCacheConfig{
				TtlSecs:            int64(opts.ttl.Seconds()),
				NegativeTtlSecs:    int64(opts.ttl.Seconds()),
				TypeCacheMaxSizeMb: 4,
				StatCacheMaxSizeMb: 32,
			},
		},
		ImplicitDirectories: opts.implicitDirs,
		MetricHandle:        mh,
		TraceHandle:         tracing.NewNoopTracer(),
		CacheClock:          clock,
		BucketName:          bucketName,
		BucketManager:       &fakeBucketManagerWithMetrics{buckets: map[string]gcs.Bucket{bucketName: bucket}},
	})
	require.NoError(t, err, "NewFileSystem")
	return &metadataCacheTestEnv{
		bucket: uncachedBucket,
		server: wrappers.WithMonitoring(server, mh),
		reader: reader,
		clock:  clock,
		ttl:    opts.ttl,
	}
}

// With ignore-interrupts on, ops run on a context detached from the FUSE
// request's. Their metadata cache reads must still be counted.
func TestMetadataCacheReadCount_IgnoreInterrupts(t *testing.T) {
	ctx := context.Background()
	env := newProductionLikeMetadataCacheFS(ctx, t, metadataCacheFSOptions{ttl: metadataCacheTTL})
	env.putObject(ctx, t, "existing_file.txt")

	for range 2 {
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "existing_file.txt")
		require.NoError(t, err)
	}

	env.assertReads(ctx, t,
		wantRead{false, statusNone, detailNotFound, 1},
		wantRead{true, statusPositive, detailFound, 1},
	)
}

// A rename reads the metadata cache for several names, but is a single FUSE op
// and so records a single event.
func TestMetadataCacheReadCount_RenameRecordsOneEvent(t *testing.T) {
	ctx := context.Background()
	env := newProductionLikeMetadataCacheFS(ctx, t, metadataCacheFSOptions{ttl: metadataCacheTTL})
	env.putObject(ctx, t, "old.txt")
	_, err := env.lookUp(ctx, fuseops.RootInodeID, "old.txt")
	require.NoError(t, err)
	waitForMetricsProcessing()
	before := totalMetadataCacheReads(t, ctx, env.reader)

	err = env.server.Rename(ctx, &fuseops.RenameOp{
		OldParent: fuseops.RootInodeID, OldName: "old.txt",
		NewParent: fuseops.RootInodeID, NewName: "new.txt",
	})

	require.NoError(t, err)
	waitForMetricsProcessing()
	assert.Equal(t, before+1, totalMetadataCacheReads(t, ctx, env.reader))
}

// With implicit directories, a lookup that the stat cache cannot answer alone
// checks for the directory with a listing, which goes to GCS without reading
// the stat cache. Such a lookup is a miss, even when the stat cache answers
// for the file.
func TestMetadataCacheReadCount_ImplicitDirs(t *testing.T) {
	ctx := context.Background()

	t.Run("listing after a negative hit is a miss", func(t *testing.T) {
		// Empty managed folders are never cached as negative entries, so every
		// lookup of a missing name lists it, although from the second lookup
		// on the stat cache answers for the file.
		env := newProductionLikeMetadataCacheFS(ctx, t, metadataCacheFSOptions{
			ttl: metadataCacheTTL, implicitDirs: true, enableEmptyManagedFolders: true,
		})

		for range 3 {
			_, err := env.lookUp(ctx, fuseops.RootInodeID, "missing")
			require.Equal(t, fuse.ENOENT, err)
		}

		env.assertReads(ctx, t, wantRead{false, statusNone, detailNotFound, 3})
	})

	t.Run("listing after the directory entry expired is a TTL miss", func(t *testing.T) {
		env := newProductionLikeMetadataCacheFS(ctx, t, metadataCacheFSOptions{ttl: metadataCacheTTL, implicitDirs: true})
		env.putObject(ctx, t, "deleted")
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "deleted")
		require.NoError(t, err)
		// Deleting the file caches a fresh negative entry for "deleted", but
		// not for "deleted/", whose entry from the first lookup expires first.
		env.clock.AdvanceTime(metadataCacheTTL / 2)
		require.NoError(t, env.server.Unlink(ctx, &fuseops.UnlinkOp{Parent: fuseops.RootInodeID, Name: "deleted"}))
		env.clock.AdvanceTime(metadataCacheTTL/2 + 10*time.Second)

		_, err = env.lookUp(ctx, fuseops.RootInodeID, "deleted")

		assert.Equal(t, fuse.ENOENT, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{false, statusNegative, detailTTLExpired, 1},
		)
	})

	t.Run("implicit directory answered by the stat cache is a hit", func(t *testing.T) {
		env := newProductionLikeMetadataCacheFS(ctx, t, metadataCacheFSOptions{ttl: metadataCacheTTL, implicitDirs: true})
		env.putObject(ctx, t, "dir/file")

		for range 2 {
			_, err := env.lookUp(ctx, fuseops.RootInodeID, "dir")
			require.NoError(t, err)
		}

		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{true, statusPositive, detailFound, 1},
		)
	})

	t.Run("disabled metadata cache records nothing", func(t *testing.T) {
		env := newProductionLikeMetadataCacheFS(ctx, t, metadataCacheFSOptions{implicitDirs: true})

		_, err := env.lookUp(ctx, fuseops.RootInodeID, "missing")

		assert.Equal(t, fuse.ENOENT, err)
		env.assertReads(ctx, t)
	})
}
