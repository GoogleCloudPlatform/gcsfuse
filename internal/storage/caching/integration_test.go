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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

func TestIntegration(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type IntegrationTest struct {
	ctx context.Context

	clock   timeutil.SimulatedClock
	wrapped gcs.Bucket

	bucket gcs.Bucket
}

func init() { RegisterTestSuite(&IntegrationTest{}) }

func (t *IntegrationTest) SetUp(ti *TestInfo) {
	t.ctx = context.Background()
	bucketName := "some_bucket"

	// Set up a fixed, non-zero time.
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))

	// Set up dependencies.
	const cacheCapacity = 100
	lruCache := lru.NewTrieCache(cfg.AverageSizeOfPositiveStatCacheEntry * cacheCapacity)
	cache := metadata.NewStatCacheBucketView(lruCache, "")
	t.wrapped = fake.NewFakeBucket(&t.clock, bucketName, gcs.BucketType{})

	t.bucket = caching.NewFastStatBucket(
		primaryCacheTTL,
		cache,
		&t.clock,
		t.wrapped,
		negativeCacheTTL,
	)
}

func (t *IntegrationTest) stat(name string) (o *gcs.Object, err error) {
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	m, _, err := t.bucket.StatObject(t.ctx, req)
	o = storageutil.ConvertMinObjectToObject(m)
	return
}

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *IntegrationTest) CreateInsertsIntoCache() {
	const name = "taco"
	var err error

	// Create an object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, name, []byte{})
	AssertEq(nil, err)

	// Delete it through the back door.
	err = t.wrapped.DeleteObject(t.ctx, &gcs.DeleteObjectRequest{Name: name})
	AssertEq(nil, err)

	// StatObject should still see it.
	o, err := t.stat(name)
	AssertEq(nil, err)
	ExpectNe(nil, o)
}

func (t *IntegrationTest) StatInsertsIntoCache() {
	const name = "foo"
	var err error

	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte{})
	AssertEq(nil, err)

	// Stat it so that it's in cache.
	_, err = t.stat(name)
	AssertEq(nil, err)

	// Delete it through the back door.
	err = t.wrapped.DeleteObject(t.ctx, &gcs.DeleteObjectRequest{Name: name})
	AssertEq(nil, err)

	// StatObject should still see it.
	o, err := t.stat(name)
	AssertEq(nil, err)
	ExpectNe(nil, o)
}

func (t *IntegrationTest) ListInsertsIntoCache() {
	const name = "taco"
	var err error

	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte{})
	AssertEq(nil, err)

	// List so that it's in cache.
	_, err = t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	// Delete the object through the back door.
	err = t.wrapped.DeleteObject(t.ctx, &gcs.DeleteObjectRequest{Name: name})
	AssertEq(nil, err)

	// StatObject should still see it.
	o, err := t.stat(name)
	AssertEq(nil, err)
	ExpectNe(nil, o)
}

func (t *IntegrationTest) UpdateUpdatesCache() {
	const name = "taco"
	var err error

	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte{})
	AssertEq(nil, err)

	// Update it, putting the new version in cache.
	updateReq := &gcs.UpdateObjectRequest{
		Name: name,
	}

	_, err = t.bucket.UpdateObject(t.ctx, updateReq)
	AssertEq(nil, err)

	// Delete the object through the back door.
	err = t.wrapped.DeleteObject(t.ctx, &gcs.DeleteObjectRequest{Name: name})
	AssertEq(nil, err)

	// StatObject should still see it.
	o, err := t.stat(name)
	AssertEq(nil, err)
	ExpectNe(nil, o)
}

func (t *IntegrationTest) PositiveCacheExpiration() {
	const name = "taco"
	var err error

	// Create an object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, name, []byte{})
	AssertEq(nil, err)

	// Delete it through the back door.
	err = t.wrapped.DeleteObject(t.ctx, &gcs.DeleteObjectRequest{Name: name})
	AssertEq(nil, err)

	// Advance time.
	t.clock.AdvanceTime(primaryCacheTTL + time.Millisecond)

	// StatObject should no longer see it.
	_, err = t.stat(name)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *IntegrationTest) CreateInvalidatesNegativeCache() {
	const name = "taco"
	var err error

	// Stat an unknown object, getting it into the negative cache.
	_, err = t.stat(name)
	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))

	// Create the object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, name, []byte{})
	AssertEq(nil, err)

	// Now StatObject should see it.
	o, err := t.stat(name)
	AssertEq(nil, err)
	ExpectNe(nil, o)
}

func (t *IntegrationTest) StatAddsToNegativeCache() {
	const name = "taco"
	var err error

	// Stat an unknown object, getting it into the negative cache.
	_, err = t.stat(name)
	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))

	// Create the object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte{})
	AssertEq(nil, err)

	// StatObject should still not see it yet.
	_, err = t.stat(name)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *IntegrationTest) ListInvalidatesNegativeCache() {
	const name = "taco"
	var err error

	// Stat an unknown object, getting it into the negative cache.
	_, err = t.stat(name)
	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))

	// Create the object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte{})
	AssertEq(nil, err)

	// List the bucket.
	_, err = t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	// Now StatObject should see it.
	o, err := t.stat(name)
	AssertEq(nil, err)
	ExpectNe(nil, o)
}

func (t *IntegrationTest) UpdateInvalidatesNegativeCache() {
	const name = "taco"
	var err error

	// Stat an unknown object, getting it into the negative cache.
	_, err = t.stat(name)
	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))

	// Create the object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte{})
	AssertEq(nil, err)

	// Update the object.
	updateReq := &gcs.UpdateObjectRequest{
		Name: name,
	}

	_, err = t.bucket.UpdateObject(t.ctx, updateReq)
	AssertEq(nil, err)

	// Now StatObject should see it.
	o, err := t.stat(name)
	AssertEq(nil, err)
	ExpectNe(nil, o)
}

func (t *IntegrationTest) NegativeCacheExpiration() {
	const name = "taco"
	var err error

	// Stat an unknown object, getting it into the negative cache.
	_, err = t.stat(name)
	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))

	// Create the object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte{})
	AssertEq(nil, err)

	// Advance time.
	t.clock.AdvanceTime(negativeCacheTTL + time.Millisecond)

	// Now StatObject should see it.
	o, err := t.stat(name)
	AssertEq(nil, err)
	ExpectNe(nil, o)
}

func (t *IntegrationTest) TestRenameFolderInvalidatesPrefix() {
	// 1. Create a folder "src/".
	_, err := t.bucket.CreateFolder(t.ctx, "src/")
	AssertEq(nil, err)

	// 2. Create an object "src/obj".
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "src/obj", []byte("content"))
	AssertEq(nil, err)

	// 3. Stat "src/obj" to populate the cache.
	_, err = t.stat("src/obj")
	AssertEq(nil, err)

	// Verify it's in cache (implicitly, by stating again and ensuring no error).
	_, err = t.stat("src/obj")
	AssertEq(nil, err)

	// 4. Rename "src/" to "dst/".
	_, err = t.bucket.RenameFolder(t.ctx, "src/", "dst/")
	AssertEq(nil, err)

	// 5. Stat "src/obj". Should fail with NotFound.
	// If the cache was NOT invalidated, this would return the cached object
	// because FastStatBucket checks cache first.
	_, err = t.stat("src/obj")
	AssertNe(nil, err)
	var notFoundError *gcs.NotFoundError
	AssertTrue(errors.As(err, &notFoundError), fmt.Sprintf("Expected NotFoundError, got: %v", err))

	// 6. Verify "dst/obj" exists (optional, but good for sanity).
	_, err = t.stat("dst/obj")
	AssertEq(nil, err)
}
