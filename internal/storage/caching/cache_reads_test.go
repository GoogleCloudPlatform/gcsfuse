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

package caching_test

import (
	"context"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cacheRead is one recorded metadata_cache/read_count event.
type cacheRead struct {
	hit    bool
	status metrics.EntryStatus
	detail metrics.LookupDetail
}

var (
	positiveHit     = cacheRead{true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr}
	negativeHit     = cacheRead{true, metrics.EntryStatusNegativeAttr, metrics.LookupDetailFoundAttr}
	absent          = cacheRead{false, metrics.EntryStatusAttr, metrics.LookupDetailNotFoundAttr}
	expiredPositive = cacheRead{false, metrics.EntryStatusPositiveAttr, metrics.LookupDetailTtlExpiredAttr}
)

// readCountRecorder records the metadata_cache/read_count events it receives
// and ignores every other metric.
type readCountRecorder struct {
	metrics.MetricHandle
	events []cacheRead
}

func (r *readCountRecorder) MetadataCacheReadCount(inc int64, cacheHit bool, entryStatus metrics.EntryStatus, lookupDetail metrics.LookupDetail) {
	r.events = append(r.events, cacheRead{cacheHit, entryStatus, lookupDetail})
}

// eventsOf runs op as a single FUSE op would, collecting its stat-cache reads,
// and returns the metadata_cache/read_count events the op records.
func eventsOf(ctx context.Context, op func(ctx context.Context)) []cacheRead {
	cacheReads := metadata.NewCacheReads(ctx)
	op(cacheReads)
	recorder := &readCountRecorder{MetricHandle: metrics.NewNoopMetrics()}
	cacheReads.Emit(recorder)
	return recorder.events
}

// probe stats name from the stat cache only, as lookups do before going to GCS.
func probe(ctx context.Context, bucket gcs.Bucket, name string) error {
	_, _, err := bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: name, FetchOnlyFromCache: true})
	return err
}

func TestCacheReads_StatObjectOfExistingObject(t *testing.T) {
	deps := setupIntegrationTest(t)
	const name = "taco"
	_, err := storageutil.CreateObject(deps.ctx, deps.wrapped, name, []byte{})
	require.NoError(t, err)
	stat := func(ctx context.Context) {
		_, _, err := deps.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: name})
		require.NoError(t, err)
	}

	assert.Equal(t, []cacheRead{absent}, eventsOf(deps.ctx, stat), "cold")
	assert.Equal(t, []cacheRead{positiveHit}, eventsOf(deps.ctx, stat), "warm")
	deps.clock.AdvanceTime(primaryCacheTTL + time.Second)
	assert.Equal(t, []cacheRead{expiredPositive}, eventsOf(deps.ctx, stat), "expired")
}

func TestCacheReads_StatObjectOfMissingObject(t *testing.T) {
	deps := setupIntegrationTest(t)
	stat := func(ctx context.Context) {
		_, _, err := deps.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: "taco"})
		var notFoundErr *gcs.NotFoundError
		require.ErrorAs(t, err, &notFoundErr)
	}

	assert.Equal(t, []cacheRead{absent}, eventsOf(deps.ctx, stat), "cold")
	assert.Equal(t, []cacheRead{negativeHit}, eventsOf(deps.ctx, stat), "warm")
}

// A cache-only stat that misses makes no GCS call, so it is not a miss.
func TestCacheReads_CacheOnlyMissRecordsNothing(t *testing.T) {
	deps := setupIntegrationTest(t)

	events := eventsOf(deps.ctx, func(ctx context.Context) {
		var cacheMissErr *caching.CacheMissError
		assert.ErrorAs(t, probe(ctx, deps.bucket, "taco"), &cacheMissErr)
	})

	assert.Empty(t, events)
}

// The shape of a lookup after a listing, which caches "taco" but not "taco/":
// the directory probe misses, the file probe hits, and GCS is never called.
func TestCacheReads_CacheOnlyMissAnsweredByHitIsHit(t *testing.T) {
	deps := setupIntegrationTest(t)
	_, err := storageutil.CreateObject(deps.ctx, deps.wrapped, "taco", []byte{})
	require.NoError(t, err)
	_, _, err = deps.bucket.StatObject(deps.ctx, &gcs.StatObjectRequest{Name: "taco"})
	require.NoError(t, err)

	events := eventsOf(deps.ctx, func(ctx context.Context) {
		var cacheMissErr *caching.CacheMissError
		assert.ErrorAs(t, probe(ctx, deps.bucket, "taco/"), &cacheMissErr)
		assert.NoError(t, probe(ctx, deps.bucket, "taco"))
	})

	assert.Equal(t, []cacheRead{positiveHit}, events)
}

// A cache-only probe that finds the entry expired erases it, so the stat that
// then fetches it from GCS finds nothing. The op is still a TTL expiry.
func TestCacheReads_ExpiredEntryProbedThenFetchedIsTTLExpiry(t *testing.T) {
	deps := setupIntegrationTest(t)
	_, err := storageutil.CreateObject(deps.ctx, deps.wrapped, "taco", []byte{})
	require.NoError(t, err)
	_, _, err = deps.bucket.StatObject(deps.ctx, &gcs.StatObjectRequest{Name: "taco"})
	require.NoError(t, err)
	deps.clock.AdvanceTime(primaryCacheTTL + time.Second)

	events := eventsOf(deps.ctx, func(ctx context.Context) {
		var cacheMissErr *caching.CacheMissError
		assert.ErrorAs(t, probe(ctx, deps.bucket, "taco"), &cacheMissErr)
		_, _, err := deps.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: "taco"})
		require.NoError(t, err)
	})

	assert.Equal(t, []cacheRead{expiredPositive}, events)
}

func TestCacheReads_ForcedGCSStatRecordsNothing(t *testing.T) {
	deps := setupIntegrationTest(t)
	_, err := storageutil.CreateObject(deps.ctx, deps.wrapped, "taco", []byte{})
	require.NoError(t, err)

	events := eventsOf(deps.ctx, func(ctx context.Context) {
		_, _, err := deps.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: "taco", ForceFetchFromGcs: true})
		require.NoError(t, err)
	})

	assert.Empty(t, events)
}

func TestCacheReads_ListObjectsRecordsNothing(t *testing.T) {
	deps := setupIntegrationTest(t)
	_, err := storageutil.CreateObject(deps.ctx, deps.wrapped, "dir/taco", []byte{})
	require.NoError(t, err)

	events := eventsOf(deps.ctx, func(ctx context.Context) {
		_, err := deps.bucket.ListObjects(ctx, &gcs.ListObjectsRequest{Prefix: "dir/", Delimiter: "/"})
		require.NoError(t, err)
	})

	assert.Empty(t, events)
}

func TestCacheReads_GetFolder(t *testing.T) {
	deps := setupIntegrationTest(t)
	_, err := deps.bucket.CreateFolder(deps.ctx, "dir/")
	require.NoError(t, err)

	hitEvents := eventsOf(deps.ctx, func(ctx context.Context) {
		_, err := deps.bucket.GetFolder(ctx, &gcs.GetFolderRequest{Name: "dir/"})
		require.NoError(t, err)
	})
	cacheOnlyMissEvents := eventsOf(deps.ctx, func(ctx context.Context) {
		_, err := deps.bucket.GetFolder(ctx, &gcs.GetFolderRequest{Name: "other/", FetchOnlyFromCache: true})
		var cacheMissErr *caching.CacheMissError
		require.ErrorAs(t, err, &cacheMissErr)
	})
	fetchedMissEvents := eventsOf(deps.ctx, func(ctx context.Context) {
		_, err := deps.bucket.GetFolder(ctx, &gcs.GetFolderRequest{Name: "other/"})
		var notFoundErr *gcs.NotFoundError
		require.ErrorAs(t, err, &notFoundErr)
	})

	assert.Equal(t, []cacheRead{positiveHit}, hitEvents)
	assert.Empty(t, cacheOnlyMissEvents)
	assert.Equal(t, []cacheRead{absent}, fetchedMissEvents)
}

func TestCacheReads_OneEventForManyReads(t *testing.T) {
	deps := setupIntegrationTest(t)
	_, err := storageutil.CreateObject(deps.ctx, deps.wrapped, "taco", []byte{})
	require.NoError(t, err)
	_, _, err = deps.bucket.StatObject(deps.ctx, &gcs.StatObjectRequest{Name: "taco"})
	require.NoError(t, err)

	// A warm hit followed by a miss fetched from GCS within one op is a single
	// miss.
	events := eventsOf(deps.ctx, func(ctx context.Context) {
		_, _, err := deps.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: "taco"})
		require.NoError(t, err)
		_, _, err = deps.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: "burrito"})
		var notFoundErr *gcs.NotFoundError
		require.ErrorAs(t, err, &notFoundErr)
	})

	assert.Equal(t, []cacheRead{absent}, events)
}
