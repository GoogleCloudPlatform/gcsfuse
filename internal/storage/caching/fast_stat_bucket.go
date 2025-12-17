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

package caching

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"golang.org/x/net/context"

	"github.com/jacobsa/timeutil"
)

// Create a bucket that caches object records returned by the supplied wrapped
// bucket. Records are invalidated when modifications are made through this
// bucket, and after the supplied TTL.
func NewFastStatBucket(
	primaryCacheTTL time.Duration,
	cache metadata.StatCache,
	clock timeutil.Clock,
	wrapped gcs.Bucket,
	negativeCacheTTL time.Duration,
) (b gcs.Bucket) {
	fsb := &fastStatBucket{
		cache:            cache,
		clock:            clock,
		wrapped:          wrapped,
		primaryCacheTTL:  primaryCacheTTL,
		negativeCacheTTL: negativeCacheTTL,
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

	// TTL for entries for existing files and folders in the cache.
	primaryCacheTTL time.Duration
	// TTL for entries for non-existing files and folders in the cache.
	negativeCacheTTL time.Duration
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) insertMultiple(objs []*gcs.Object) {
	b.mu.Lock()
	defer b.mu.Unlock()

	expiration := b.clock.Now().Add(b.primaryCacheTTL)
	for _, o := range objs {
		m := storageutil.ConvertObjToMinObject(o)
		b.cache.Insert(m, expiration)
	}
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) insertMultipleMinObjects(listing *gcs.Listing, dirName string) {
	// logger.Info("insertMultipleMinObjects: ", listing)
	b.mu.Lock()
	defer b.mu.Unlock()

	minObjectNames := make(map[string]struct{})
	expiration := b.clock.Now().Add(b.primaryCacheTTL)

	if len(listing.MinObjects) != 0 {
		// logger.Info("MinObject name: ", listing.MinObjects[0].Name, dirName)
		if listing.MinObjects[0].Name != dirName {
			// logger.Info("In Implicit dir caching", dirName)
			f := &gcs.MinObject{
				Name: dirName,
			}
			b.cache.Insert(f, expiration)
		}
	}

	for _, o := range listing.MinObjects {
		b.cache.Insert(o, expiration)
		minObjectNames[o.Name] = struct{}{}
	}

	for _, p := range listing.CollapsedRuns {
		// logger.Info("CollapsedRuns: ", p)
		// If a MinObject with the same name as the CollapsedRun already exists,
		// we don't need to insert it again as a Folder.
		if _, ok := minObjectNames[p]; ok {
			continue
		}
		if !strings.HasSuffix(p, "/") {
			// log the error for incorrect prefix but don't fail the operation
			logger.Errorf("error in prefix name: %s", p)
		} else {
			// logger.Info("In Implicit dir caching", p)
			f := &gcs.MinObject{
				Name: p,
			}
			b.cache.Insert(f, expiration)
		}
	}

}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) eraseEntriesWithGivenPrefix(folderName string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cache.EraseEntriesWithGivenPrefix(folderName)
}

// insertHierarchicalListing saves the objects in cache excluding zero byte objects corresponding to folders
// by iterating objects present in listing and saves prefixes as folders (all prefixes are folders in hns) by
// iterating collapsedRuns of listing.
func (b *fastStatBucket) insertHierarchicalListing(listing *gcs.Listing) {
	b.mu.Lock()
	defer b.mu.Unlock()

	expiration := b.clock.Now().Add(b.primaryCacheTTL)

	for _, o := range listing.MinObjects {
		if !strings.HasSuffix(o.Name, "/") {
			b.cache.Insert(o, expiration)
		}
	}

	for _, p := range listing.CollapsedRuns {
		if !strings.HasSuffix(p, "/") {
			// log the error for incorrect prefix but don't fail the operation
			logger.Errorf("error in prefix name: %s", p)
		} else {
			f := &gcs.Folder{
				Name: p,
			}
			b.cache.InsertFolder(f, expiration)
		}
	}

}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) insert(o *gcs.Object) {
	b.insertMultiple([]*gcs.Object{o})
}

func (b *fastStatBucket) insertMinObject(o *gcs.MinObject) {
	b.mu.Lock()
	defer b.mu.Unlock()

	expiration := b.clock.Now().Add(b.primaryCacheTTL)
	b.cache.Insert(o, expiration)
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) insertFolder(f *gcs.Folder) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cache.InsertFolder(f, b.clock.Now().Add(b.primaryCacheTTL))
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) addNegativeEntry(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	expiration := b.clock.Now().Add(b.negativeCacheTTL)
	b.cache.AddNegativeEntry(name, expiration)
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) addNegativeEntryForFolder(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	expiration := b.clock.Now().Add(b.negativeCacheTTL)
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

func (b *fastStatBucket) NewReaderWithReadHandle(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (rd gcs.StorageReader, err error) {
	rd, err = b.wrapped.NewReaderWithReadHandle(ctx, req)

	var notFoundError *gcs.NotFoundError
	if errors.As(err, &notFoundError) {
		b.invalidate(req.Name)
	}
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

	// Record the new object.
	b.insert(o)

	return
}

func (b *fastStatBucket) CreateObjectChunkWriter(ctx context.Context, req *gcs.CreateObjectRequest, chunkSize int, callBack func(bytesUploadedSoFar int64)) (gcs.Writer, error) {
	return b.wrapped.CreateObjectChunkWriter(ctx, req, chunkSize, callBack)
}

func (b *fastStatBucket) CreateAppendableObjectWriter(ctx context.Context, req *gcs.CreateObjectChunkWriterRequest) (gcs.Writer, error) {
	return b.wrapped.CreateAppendableObjectWriter(ctx, req)
}

func (b *fastStatBucket) FinalizeUpload(ctx context.Context, writer gcs.Writer) (*gcs.MinObject, error) {
	name := writer.ObjectName()
	// Throw away any existing record for this object.
	b.invalidate(name)

	o, err := b.wrapped.FinalizeUpload(ctx, writer)

	// Record the new object if err is nil.
	if err == nil {
		b.insertMinObject(o)
	}

	return o, err
}

func (b *fastStatBucket) FlushPendingWrites(ctx context.Context, writer gcs.Writer) (*gcs.MinObject, error) {
	name := writer.ObjectName()
	// Throw away any existing record for this object.
	b.invalidate(name)

	o, err := b.wrapped.FlushPendingWrites(ctx, writer)

	// Record the new object if err is nil.
	if err == nil {
		b.insertMinObject(o)
	}
	return o, err
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
	// logger.Debugf("StatObject")
	// If fetching from gcs is enabled, directly make a call to GCS.
	if req.ForceFetchFromGcs {
		logger.Debugf("In force fetch GCS")
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
			b.addNegativeEntry(req.Name)
			err = &gcs.NotFoundError{
				Err: fmt.Errorf("negative cache entry for %v", req.Name),
			}

			return
		}

		// update the stat cache with new expiry.
		b.insertMinObject(entry)
		// Otherwise, return MinObject and nil ExtendedObjectAttributes.
		m = entry
		return
	}

	if req.ForceFetchFromCache {
		return nil, nil, &gcs.NotFoundCacheError{Err: fmt.Errorf("Not found cache entry for %v", req.Name)}
	}

	return b.StatObjectFromGcs(ctx, req)
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) ListObjects(
	ctx context.Context,
	req *gcs.ListObjectsRequest) (*gcs.Listing, error) {
	// If ForceFetchFromCache is true, we will try to serve listing from cache.
	if req.ForceFetchFromCache {
		var hit bool
		var entry any
		// logger.Infof("req Prefix: ", req.Prefix)
		hit, entry = b.lookUp(req.Prefix)

		if hit {
			// Negative entries result in NotFoundError.
			// logger.Infof("entry: ", entry)
			if entry.(*gcs.MinObject) == nil {
				b.addNegativeEntry(req.Prefix)
				return nil, &gcs.NotFoundError{
					Err: fmt.Errorf("negative cache entry for %v", req.Prefix),
				}
			}
			if minObject, ok := entry.(*gcs.MinObject); ok {
				b.insertMinObject(entry.(*gcs.MinObject))
				if minObject.Generation == 0 { // Assumed to be a directory-like object from a collapsed run.
					return &gcs.Listing{CollapsedRuns: []string{minObject.Name}}, nil
				}
				return &gcs.Listing{MinObjects: []*gcs.MinObject{minObject}}, nil
			}
		}
		return nil, &gcs.NotFoundCacheError{Err: fmt.Errorf("Not found cache entry for %v", req.Prefix)}
	}

	// Fetch the listing.
	listing, err := b.wrapped.ListObjects(ctx, req)
	if err != nil {
		return nil, err
	}
	if b.BucketType().Hierarchical {
		b.insertHierarchicalListing(listing)
		return listing, nil
	}

	// note anything we found.
	b.insertMultipleMinObjects(listing, req.Prefix)
	return listing, nil
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
	if req.DeleteFromCache {
		b.addNegativeEntry(req.Name)
		return nil
	}
	err = b.wrapped.DeleteObject(ctx, req)
	if err != nil {
		b.invalidate(req.Name)
	} else {
		b.addNegativeEntry(req.Name)
	}
	return
}

func (b *fastStatBucket) MoveObject(ctx context.Context, req *gcs.MoveObjectRequest) (*gcs.Object, error) {
	// Throw away any existing record for the source and destination name.
	b.invalidate(req.SrcName)
	b.invalidate(req.DstName)

	// Move the object.
	o, err := b.wrapped.MoveObject(ctx, req)
	if err != nil {
		return nil, err
	}

	// Record the new version.
	b.insert(o)

	return o, nil
}

func (b *fastStatBucket) DeleteFolder(ctx context.Context, folderName string) error {
	err := b.wrapped.DeleteFolder(ctx, folderName)
	// In case of an error; invalidate the cached entry. This will make sure that
	// gcsfuse is not caching possibly erroneous status of the folder and next
	// call will hit GCS backend to probe the latest status.
	if err != nil {
		b.invalidate(folderName)
	} else {
		b.addNegativeEntryForFolder(folderName)
	}
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
	b.insertMinObject(m)

	return
}

func (b *fastStatBucket) GetFolder(ctx context.Context, req *gcs.GetFolderRequest) (*gcs.Folder, error) {
	if hit, entry := b.lookUpFolder(req.Name); hit {
		// Negative entries result in NotFoundError.
		if entry == nil {
			err := &gcs.NotFoundError{
				Err: fmt.Errorf("negative cache entry for folder %v", req.Name),
			}
			b.addNegativeEntryForFolder(req.Name)

			return nil, err
		}
		b.insertFolder(entry)
		return entry, nil
	}

	if req.ForceFetchFromCache {
		return nil, &gcs.NotFoundCacheError{Err: fmt.Errorf("Not found cache entry for %v", req.Name)}
	}

	// Fetch the Folder from GCS
	return b.getFolderFromGCS(ctx, req)
}

func (b *fastStatBucket) getFolderFromGCS(ctx context.Context, req *gcs.GetFolderRequest) (*gcs.Folder, error) {
	f, err := b.wrapped.GetFolder(ctx, req)

	if err == nil {
		b.insertFolder(f)
		return f, nil
	}

	// Special case: NotFoundError -> negative entry.
	if _, ok := err.(*gcs.NotFoundError); ok {
		b.addNegativeEntryForFolder(req.Name)
	}
	return nil, err
}

func (b *fastStatBucket) CreateFolder(ctx context.Context, folderName string) (f *gcs.Folder, err error) {
	// Throw away any existing record for this folder.
	b.invalidate(folderName)

	f, err = b.wrapped.CreateFolder(ctx, folderName)
	if err != nil {
		return
	}

	// Record the new folder.
	b.insertFolder(f)

	return
}

func (b *fastStatBucket) RenameFolder(ctx context.Context, folderName string, destinationFolderId string) (*gcs.Folder, error) {
	f, err := b.wrapped.RenameFolder(ctx, folderName, destinationFolderId)
	if err != nil {
		return nil, err
	}

	// Invalidate cache for old directory.
	b.eraseEntriesWithGivenPrefix(folderName)
	// Insert destination folder.
	b.insertFolder(f)

	return f, err
}

func (b *fastStatBucket) NewMultiRangeDownloader(
	ctx context.Context, req *gcs.MultiRangeDownloaderRequest) (mrd gcs.MultiRangeDownloader, err error) {
	mrd, err = b.wrapped.NewMultiRangeDownloader(ctx, req)
	return
}

func (b *fastStatBucket) GCSName(obj *gcs.MinObject) string {
	return b.wrapped.GCSName(obj)
}
