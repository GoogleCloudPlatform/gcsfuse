// Copyright 2023 Google LLC
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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

type integrationTestDeps struct {
	ctx     context.Context
	clock   *timeutil.SimulatedClock
	wrapped gcs.Bucket
	bucket  gcs.Bucket
}

func setupIntegrationTest(t *testing.T) *integrationTestDeps {
	ctx := context.Background()
	bucketName := "some_bucket"

	clock := &timeutil.SimulatedClock{}
	clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))

	const cacheCapacity = 100
	lruCache := lru.NewCache(cfg.AverageSizeOfPositiveStatCacheEntry * cacheCapacity)
	cache := metadata.NewStatCacheBucketView(lruCache, "")
	wrapped := fake.NewFakeBucket(clock, bucketName, gcs.BucketType{})

	bucket := caching.NewFastStatBucket(
		primaryCacheTTL,
		cache,
		clock,
		wrapped,
		negativeCacheTTL,
		isTypeCacheDeprecated,
		isImplicitDir,
	)

	return &integrationTestDeps{
		ctx:     ctx,
		clock:   clock,
		wrapped: wrapped,
		bucket:  bucket,
	}
}

func statObject(ctx context.Context, bucket gcs.Bucket, name string) (*gcs.Object, error) {
	req := &gcs.StatObjectRequest{
		Name: name,
	}
	m, _, err := bucket.StatObject(ctx, req)
	if err != nil {
		return nil, err
	}
	return storageutil.ConvertMinObjectToObject(m), nil
}

func TestIntegration_CreateInsertsIntoCache(t *testing.T) {
	deps := setupIntegrationTest(t)
	const name = "taco"

	// Create an object.
	_, err := storageutil.CreateObject(deps.ctx, deps.bucket, name, []byte{})
	require.NoError(t, err)

	// Delete it through the back door.
	err = deps.wrapped.DeleteObject(deps.ctx, &gcs.DeleteObjectRequest{Name: name})
	require.NoError(t, err)

	// StatObject should still see it.
	o, err := statObject(deps.ctx, deps.bucket, name)
	require.NoError(t, err)
	assert.NotNil(t, o)
}

func TestIntegration_StatInsertsIntoCache(t *testing.T) {
	deps := setupIntegrationTest(t)
	const name = "foo"

	// Create an object through the back door.
	_, err := storageutil.CreateObject(deps.ctx, deps.wrapped, name, []byte{})
	require.NoError(t, err)

	// Stat it so that it's in cache.
	_, err = statObject(deps.ctx, deps.bucket, name)
	require.NoError(t, err)

	// Delete it through the back door.
	err = deps.wrapped.DeleteObject(deps.ctx, &gcs.DeleteObjectRequest{Name: name})
	require.NoError(t, err)

	// StatObject should still see it.
	o, err := statObject(deps.ctx, deps.bucket, name)
	require.NoError(t, err)
	assert.NotNil(t, o)
}

func TestIntegration_ListInsertsIntoCache(t *testing.T) {
	deps := setupIntegrationTest(t)
	const name = "taco"

	// Create an object through the back door.
	_, err := storageutil.CreateObject(deps.ctx, deps.wrapped, name, []byte{})
	require.NoError(t, err)

	// List so that it's in cache.
	_, err = deps.bucket.ListObjects(deps.ctx, &gcs.ListObjectsRequest{})
	require.NoError(t, err)

	// Delete the object through the back door.
	err = deps.wrapped.DeleteObject(deps.ctx, &gcs.DeleteObjectRequest{Name: name})
	require.NoError(t, err)

	// StatObject should still see it.
	o, err := statObject(deps.ctx, deps.bucket, name)
	require.NoError(t, err)
	assert.NotNil(t, o)
}

func TestIntegration_UpdateUpdatesCache(t *testing.T) {
	deps := setupIntegrationTest(t)
	const name = "taco"

	// Create an object through the back door.
	_, err := storageutil.CreateObject(deps.ctx, deps.wrapped, name, []byte{})
	require.NoError(t, err)

	// Update it, putting the new version in cache.
	updateReq := &gcs.UpdateObjectRequest{
		Name: name,
	}
	_, err = deps.bucket.UpdateObject(deps.ctx, updateReq)
	require.NoError(t, err)

	// Delete the object through the back door.
	err = deps.wrapped.DeleteObject(deps.ctx, &gcs.DeleteObjectRequest{Name: name})
	require.NoError(t, err)

	// StatObject should still see it.
	o, err := statObject(deps.ctx, deps.bucket, name)
	require.NoError(t, err)
	assert.NotNil(t, o)
}

func TestIntegration_PositiveCacheExpiration(t *testing.T) {
	deps := setupIntegrationTest(t)
	const name = "taco"

	// Create an object.
	_, err := storageutil.CreateObject(deps.ctx, deps.bucket, name, []byte{})
	require.NoError(t, err)

	// Delete it through the back door.
	err = deps.wrapped.DeleteObject(deps.ctx, &gcs.DeleteObjectRequest{Name: name})
	require.NoError(t, err)

	// Advance time.
	deps.clock.AdvanceTime(primaryCacheTTL + time.Millisecond)

	// StatObject should no longer see it.
	_, err = statObject(deps.ctx, deps.bucket, name)
	assert.IsType(t, &gcs.NotFoundError{}, err)
}

func TestIntegration_CreateInvalidatesNegativeCache(t *testing.T) {
	deps := setupIntegrationTest(t)
	const name = "taco"

	// Stat an unknown object, getting it into the negative cache.
	_, err := statObject(deps.ctx, deps.bucket, name)
	assert.IsType(t, &gcs.NotFoundError{}, err)

	// Create the object.
	_, err = storageutil.CreateObject(deps.ctx, deps.bucket, name, []byte{})
	require.NoError(t, err)

	// Now StatObject should see it.
	o, err := statObject(deps.ctx, deps.bucket, name)
	require.NoError(t, err)
	assert.NotNil(t, o)
}

func TestIntegration_StatAddsToNegativeCache(t *testing.T) {
	deps := setupIntegrationTest(t)
	const name = "taco"

	// Stat an unknown object, getting it into the negative cache.
	_, err := statObject(deps.ctx, deps.bucket, name)
	assert.IsType(t, &gcs.NotFoundError{}, err)

	// Create the object through the back door.
	_, err = storageutil.CreateObject(deps.ctx, deps.wrapped, name, []byte{})
	require.NoError(t, err)

	// StatObject should still not see it yet.
	_, err = statObject(deps.ctx, deps.bucket, name)
	assert.IsType(t, &gcs.NotFoundError{}, err)
}

func TestIntegration_ListInvalidatesNegativeCache(t *testing.T) {
	deps := setupIntegrationTest(t)
	const name = "taco"

	// Stat an unknown object, getting it into the negative cache.
	_, err := statObject(deps.ctx, deps.bucket, name)
	assert.IsType(t, &gcs.NotFoundError{}, err)

	// Create the object through the back door.
	_, err = storageutil.CreateObject(deps.ctx, deps.wrapped, name, []byte{})
	require.NoError(t, err)

	// List the bucket.
	_, err = deps.bucket.ListObjects(deps.ctx, &gcs.ListObjectsRequest{})
	require.NoError(t, err)

	// Now StatObject should see it.
	o, err := statObject(deps.ctx, deps.bucket, name)
	require.NoError(t, err)
	assert.NotNil(t, o)
}

func TestIntegration_UpdateInvalidatesNegativeCache(t *testing.T) {
	deps := setupIntegrationTest(t)
	const name = "taco"

	// Stat an unknown object, getting it into the negative cache.
	_, err := statObject(deps.ctx, deps.bucket, name)
	assert.IsType(t, &gcs.NotFoundError{}, err)

	// Create the object through the back door.
	_, err = storageutil.CreateObject(deps.ctx, deps.wrapped, name, []byte{})
	require.NoError(t, err)

	// Update the object.
	updateReq := &gcs.UpdateObjectRequest{
		Name: name,
	}
	_, err = deps.bucket.UpdateObject(deps.ctx, updateReq)
	require.NoError(t, err)

	// Now StatObject should see it.
	o, err := statObject(deps.ctx, deps.bucket, name)
	require.NoError(t, err)
	assert.NotNil(t, o)
}

func TestIntegration_NegativeCacheExpiration(t *testing.T) {
	deps := setupIntegrationTest(t)
	const name = "taco"

	// Stat an unknown object, getting it into the negative cache.
	_, err := statObject(deps.ctx, deps.bucket, name)
	assert.IsType(t, &gcs.NotFoundError{}, err)

	// Create the object through the back door.
	_, err = storageutil.CreateObject(deps.ctx, deps.wrapped, name, []byte{})
	require.NoError(t, err)

	// Advance time.
	deps.clock.AdvanceTime(negativeCacheTTL + time.Millisecond)

	// Now StatObject should see it.
	o, err := statObject(deps.ctx, deps.bucket, name)
	require.NoError(t, err)
	assert.NotNil(t, o)
}
