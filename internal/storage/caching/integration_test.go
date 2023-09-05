// Copyright 2023 Google Inc. All Rights Reserved.
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

	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/bucket"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/object"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/requests"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
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

	cache   caching.StatCache
	clock   timeutil.SimulatedClock
	wrapped bucket.Bucket

	bucket bucket.Bucket
}

func init() { RegisterTestSuite(&IntegrationTest{}) }

func (t *IntegrationTest) SetUp(ti *TestInfo) {
	t.ctx = context.Background()

	// Set up a fixed, non-zero time.
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))

	// Set up dependencies.
	const cacheCapacity = 100
	t.cache = caching.NewStatCache(cacheCapacity)
	t.wrapped = fake.NewFakeBucket(&t.clock, "some_bucket")

	t.bucket = caching.NewFastStatBucket(
		ttl,
		t.cache,
		&t.clock,
		t.wrapped)
}

func (t *IntegrationTest) stat(name string) (o *object.Object, err error) {
	req := &requests.StatObjectRequest{
		Name: name,
	}

	o, err = t.bucket.StatObject(t.ctx, req)
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
	err = t.wrapped.DeleteObject(t.ctx, &requests.DeleteObjectRequest{Name: name})
	AssertEq(nil, err)

	// StatObject should still see it.
	o, err := t.stat(name)
	AssertEq(nil, err)
	ExpectNe(nil, o)
}

func (t *IntegrationTest) StatInsertsIntoCache() {
	const name = "taco"
	var err error

	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte{})
	AssertEq(nil, err)

	// Stat it so that it's in cache.
	_, err = t.stat(name)
	AssertEq(nil, err)

	// Delete it through the back door.
	err = t.wrapped.DeleteObject(t.ctx, &requests.DeleteObjectRequest{Name: name})
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
	_, err = t.bucket.ListObjects(t.ctx, &requests.ListObjectsRequest{})
	AssertEq(nil, err)

	// Delete the object through the back door.
	err = t.wrapped.DeleteObject(t.ctx, &requests.DeleteObjectRequest{Name: name})
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
	updateReq := &requests.UpdateObjectRequest{
		Name: name,
	}

	_, err = t.bucket.UpdateObject(t.ctx, updateReq)
	AssertEq(nil, err)

	// Delete the object through the back door.
	err = t.wrapped.DeleteObject(t.ctx, &requests.DeleteObjectRequest{Name: name})
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
	err = t.wrapped.DeleteObject(t.ctx, &requests.DeleteObjectRequest{Name: name})
	AssertEq(nil, err)

	// Advance time.
	t.clock.AdvanceTime(ttl + time.Millisecond)

	// StatObject should no longer see it.
	_, err = t.stat(name)
	ExpectThat(err, HasSameTypeAs(&storage.NotFoundError{}))
}

func (t *IntegrationTest) CreateInvalidatesNegativeCache() {
	const name = "taco"
	var err error

	// Stat an unknown object, getting it into the negative cache.
	_, err = t.stat(name)
	AssertThat(err, HasSameTypeAs(&storage.NotFoundError{}))

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
	AssertThat(err, HasSameTypeAs(&storage.NotFoundError{}))

	// Create the object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte{})
	AssertEq(nil, err)

	// StatObject should still not see it yet.
	_, err = t.stat(name)
	ExpectThat(err, HasSameTypeAs(&storage.NotFoundError{}))
}

func (t *IntegrationTest) ListInvalidatesNegativeCache() {
	const name = "taco"
	var err error

	// Stat an unknown object, getting it into the negative cache.
	_, err = t.stat(name)
	AssertThat(err, HasSameTypeAs(&storage.NotFoundError{}))

	// Create the object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte{})
	AssertEq(nil, err)

	// List the bucket.
	_, err = t.bucket.ListObjects(t.ctx, &requests.ListObjectsRequest{})
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
	AssertThat(err, HasSameTypeAs(&storage.NotFoundError{}))

	// Create the object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte{})
	AssertEq(nil, err)

	// Update the object.
	updateReq := &requests.UpdateObjectRequest{
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
	AssertThat(err, HasSameTypeAs(&storage.NotFoundError{}))

	// Create the object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte{})
	AssertEq(nil, err)

	// Advance time.
	t.clock.AdvanceTime(ttl + time.Millisecond)

	// Now StatObject should see it.
	o, err := t.stat(name)
	AssertEq(nil, err)
	ExpectNe(nil, o)
}
