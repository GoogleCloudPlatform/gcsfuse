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

package caching

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"golang.org/x/net/context"

	"github.com/jacobsa/timeutil"
)

// Create a bucket that caches object records returned by the supplied wrapped
// bucket. Records are invalidated when modifications are made through this
// bucket, and after the supplied TTL.
func NewFastStatBucket(
	ttl time.Duration,
	cache metadata.StatCache,
	clock timeutil.Clock,
	wrapped gcs.Bucket) (b gcs.Bucket) {
	fsb := &fastStatBucket{
		cache:   cache,
		clock:   clock,
		wrapped: wrapped,
		ttl:     ttl,
	}

	b = fsb
	return
}

type fastStatBucket struct {
	mu sync.Mutex

	/////////////////////////
	// Dependencies
	/////////////////////////

	// GUARDED_BY(mu)
	cache metadata.StatCache

	clock   timeutil.Clock
	wrapped gcs.Bucket

	/////////////////////////
	// Constant data
	/////////////////////////

	ttl time.Duration
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) insertMultiple(objs []*gcs.MinObject) {
	b.mu.Lock()
	defer b.mu.Unlock()

	expiration := b.clock.Now().Add(b.ttl)
	for _, o := range objs {
		b.cache.Insert(o, expiration)
	}
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) insert(o *gcs.MinObject) {
	b.insertMultiple([]*gcs.MinObject{o})
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) addNegativeEntry(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	expiration := b.clock.Now().Add(b.ttl)
	b.cache.AddNegativeEntry(name, expiration)
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) invalidate(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cache.Erase(name)
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) lookUp(name string) (hit bool, o *gcs.MinObject) {
	b.mu.Lock()
	defer b.mu.Unlock()

	hit, o = b.cache.LookUp(name, b.clock.Now())
	return
}

////////////////////////////////////////////////////////////////////////
// Bucket interface
////////////////////////////////////////////////////////////////////////

func (b *fastStatBucket) Name() string {
	return b.wrapped.Name()
}

func (b *fastStatBucket) NewReader(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (rc io.ReadCloser, err error) {
	rc, err = b.wrapped.NewReader(ctx, req)
	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) CreateObject(
	ctx context.Context,
	req *gcs.CreateObjectRequest) (o *gcs.MinObject, err error) {
	// Throw away any existing record for this object.
	b.invalidate(req.Name)

	// Create the new object.
	o, err = b.wrapped.CreateObject(ctx, req)
	if err != nil {
		return
	}

	// Record the new object.
	b.insert(o)

	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) CopyObject(
	ctx context.Context,
	req *gcs.CopyObjectRequest) (o *gcs.MinObject, err error) {
	// Throw away any existing record for the destination name.
	b.invalidate(req.DstName)

	// Copy the object.
	o, err = b.wrapped.CopyObject(ctx, req)
	if err != nil {
		return
	}

	// Record the new version.
	b.insert(o)

	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) ComposeObjects(
	ctx context.Context,
	req *gcs.ComposeObjectsRequest) (o *gcs.MinObject, err error) {
	// Throw away any existing record for the destination name.
	b.invalidate(req.DstName)

	// Copy the object.
	o, err = b.wrapped.ComposeObjects(ctx, req)
	if err != nil {
		return
	}

	// Record the new version.
	b.insert(o)

	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) StatObject(
	ctx context.Context,
	req *gcs.StatObjectRequest) (o *gcs.MinObject, err error) {
	// If fetching from gcs is enabled, directly make a call to GCS.
	if req.ForceFetchFromGcs {
		return b.StatObject(ctx, req)
	}

	// Do we have an entry in the cache?
	if hit, entry := b.lookUp(req.Name); hit {
		// Negative entries result in NotFoundError.
		if entry == nil {
			err = &gcs.NotFoundError{
				Err: fmt.Errorf("Negative cache entry for %v", req.Name),
			}

			return
		}

		// Otherwise, return the object.
		o = entry
		return
	}

	return b.StatObjectFromGcs(ctx, req)
}

// LOCKS_EXCLUDED(b.mu)
// Cache lookup not supported.
func (b *fastStatBucket) OldStatObject(
	ctx context.Context,
	req *gcs.StatObjectRequest) (o *gcs.Object, err error) {
	// If fetching from gcs is enabled, directly make a call to GCS.
	if req.ForceFetchFromGcs {
		return b.OldStatObjectFromGcs(ctx, req)
	}

	return b.OldStatObjectFromGcs(ctx, req)
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) ListObjects(
	ctx context.Context,
	req *gcs.ListObjectsRequest) (listing *gcs.Listing, err error) {
	// Fetch the listing.
	listing, err = b.wrapped.ListObjects(ctx, req)
	if err != nil {
		return
	}

	// Note anything we found.
	b.insertMultiple(listing.Objects)

	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) UpdateObject(
	ctx context.Context,
	req *gcs.UpdateObjectRequest) (o *gcs.MinObject, err error) {
	// Throw away any existing record for this object.
	b.invalidate(req.Name)

	// Update the object.
	o, err = b.wrapped.UpdateObject(ctx, req)
	if err != nil {
		return
	}

	// Record the new version.
	b.insert(o)

	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) DeleteObject(
	ctx context.Context,
	req *gcs.DeleteObjectRequest) (err error) {
	b.invalidate(req.Name)
	err = b.wrapped.DeleteObject(ctx, req)
	return
}

func (b *fastStatBucket) StatObjectFromGcs(ctx context.Context, req *gcs.StatObjectRequest) (o *gcs.MinObject, err error) {
	o, err = b.wrapped.StatObject(ctx, req)
	if err != nil {
		// Special case: NotFoundError -> negative entry.
		if _, ok := err.(*gcs.NotFoundError); ok {
			b.addNegativeEntry(req.Name)
		}

		return
	}

	// Put the object in cache.
	b.insert(o)

	return
}

func (b *fastStatBucket) OldStatObjectFromGcs(ctx context.Context, req *gcs.StatObjectRequest) (o *gcs.Object, err error) {
	o, err = b.wrapped.OldStatObject(ctx, req)
	if err != nil {
		// Special case: NotFoundError -> negative entry.
		if _, ok := err.(*gcs.NotFoundError); ok {
			b.addNegativeEntry(req.Name)
		}

		return
	}

	return
}
