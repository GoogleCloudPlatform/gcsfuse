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
	"strings"
	"sync"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
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
func (b *fastStatBucket) insertMultiple(objs []*gcs.Object) {
	b.mu.Lock()
	defer b.mu.Unlock()

	expiration := b.clock.Now().Add(b.ttl)
	for _, o := range objs {
		if b.BucketType() == gcs.Hierarchical {
			if strings.HasSuffix(o.Name, "/") {
				f := &gcs.Folder{
					Name:           o.Name,
					MetaGeneration: 0,
					UpdateTime:     time.Now(),
				}
				b.cache.InsertFolder(f, expiration)
			} else {
				m := storageutil.ConvertObjToMinObject(o)
				b.cache.Insert(m, expiration)
			}
		} else {
			m := storageutil.ConvertObjToMinObject(o)
			b.cache.Insert(m, expiration)
		}
	}
}

func (b *fastStatBucket) insertMultipleFolders(folders []*gcs.Folder) {
	b.mu.Lock()
	defer b.mu.Unlock()

	expiration := b.clock.Now().Add(b.ttl)
	for _, f := range folders {
		b.cache.InsertFolder(f, expiration)
	}
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) insertFolder(o *gcs.Folder) {
	b.insertMultipleFolders([]*gcs.Folder{o})
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) insert(o *gcs.Object) {
	b.insertMultiple([]*gcs.Object{o})
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) addNegativeEntry(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	expiration := b.clock.Now().Add(b.ttl)
	b.cache.AddNegativeEntry(name, expiration)
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) addNegativeEntryForFolder(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	expiration := b.clock.Now().Add(b.ttl)
	b.cache.AddNegativeEntryForFolder(name, expiration)
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) invalidate(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cache.Erase(name)
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) lookUp(name string) (hit bool, m *gcs.MinObject) {
	b.mu.Lock()
	defer b.mu.Unlock()

	hit, m = b.cache.LookUp(name, b.clock.Now())
	return
}

func (b *fastStatBucket) lookUpFolder(name string) (bool, *gcs.Folder) {
	b.mu.Lock()
	defer b.mu.Unlock()

	hit, f := b.cache.LookUpFolder(name, b.clock.Now())
	return hit, f
}

////////////////////////////////////////////////////////////////////////
// Bucket interface
////////////////////////////////////////////////////////////////////////

func (b *fastStatBucket) Name() string {
	return b.wrapped.Name()
}

func (b *fastStatBucket) BucketType() gcs.BucketType {
	return b.wrapped.BucketType()
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
	req *gcs.CreateObjectRequest) (o *gcs.Object, err error) {
	// Throw away any existing record for this object.
	b.invalidate(req.Name)

	// TODO: create object to be replaced with create folder api once integrated
	o, err = b.wrapped.CreateObject(ctx, req)
	if err != nil {
		return
	}

	b.insert(o)

	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) CopyObject(
	ctx context.Context,
	req *gcs.CopyObjectRequest) (o *gcs.Object, err error) {
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
	req *gcs.ComposeObjectsRequest) (o *gcs.Object, err error) {
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
	req *gcs.StatObjectRequest) (m *gcs.MinObject, e *gcs.ExtendedObjectAttributes, err error) {
	// If ExtendedObjectAttributes are requested without fetching from gcs enabled, panic.
	if !req.ForceFetchFromGcs && req.ReturnExtendedObjectAttributes {
		panic("invalid StatObjectRequest: ForceFetchFromGcs: false and ReturnExtendedObjectAttributes: true")
	}
	// If fetching from gcs is enabled, directly make a call to GCS.
	if req.ForceFetchFromGcs {
		m, e, err = b.StatObjectFromGcs(ctx, req)
		if !req.ReturnExtendedObjectAttributes {
			e = nil
		}
		return
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

		// Otherwise, return MinObject and nil ExtendedObjectAttributes.
		m = entry
		return
	}

	return b.StatObjectFromGcs(ctx, req)
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
	req *gcs.UpdateObjectRequest) (o *gcs.Object, err error) {
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

func (b *fastStatBucket) DeleteFolder(ctx context.Context, folderName string) error {
	err := b.wrapped.DeleteFolder(ctx, folderName)
	if err != nil {
		return err
	}
	// Add negative entry in the cache.
	b.cache.Erase(folderName)

	return err
}

func (b *fastStatBucket) StatObjectFromGcs(ctx context.Context,
	req *gcs.StatObjectRequest) (m *gcs.MinObject, e *gcs.ExtendedObjectAttributes, err error) {
	m, e, err = b.wrapped.StatObject(ctx, req)
	if err != nil {
		// Special case: NotFoundError -> negative entry.
		if _, ok := err.(*gcs.NotFoundError); ok {
			b.addNegativeEntry(req.Name)
		}

		return
	}

	// Put the object in cache.
	o := storageutil.ConvertMinObjectToObject(m)
	b.insert(o)

	return
}

func (b *fastStatBucket) GetFolder(
	ctx context.Context,
	prefix string) (*gcs.Folder, error) {

	if hit, entry := b.lookUpFolder(prefix); hit {
		// Negative entries result in NotFoundError.
		if entry == nil {
			err := &gcs.NotFoundError{
				Err: fmt.Errorf("negative cache entry for folder %v", prefix),
			}

			return nil, err
		}

		return entry, nil
	}

	// Fetch the Folder
	f, err := b.wrapped.GetFolder(ctx, prefix)
	if err != nil {
		// Special case: NotFoundError -> negative entry.
		if _, ok := err.(*gcs.NotFoundError); ok {
			b.addNegativeEntryForFolder(prefix)
		}
		return nil, err
	}

	b.insertFolder(f)

	return f, err
}

func (b *fastStatBucket) CreateFolder(ctx context.Context, folderName string) (f *gcs.Folder, err error) {
	// Throw away any existing record for this folder.
	b.invalidate(folderName)

	expiration := b.clock.Now().Add(b.ttl)
	f, err = b.wrapped.CreateFolder(ctx, folderName)
	if err != nil {
		return
	}

	// Record the new folder.
	b.cache.InsertFolder(f, expiration)

	return
}

func (b *fastStatBucket) RenameFolder(ctx context.Context, folderName string, destinationFolderId string) (o *gcs.Folder, err error) {
	o, err = b.wrapped.RenameFolder(ctx, folderName, destinationFolderId)

	// Invalidate cache for old directory.
	b.cache.EraseEntriesWithGivenPrefix(folderName)

	return o, err
}
