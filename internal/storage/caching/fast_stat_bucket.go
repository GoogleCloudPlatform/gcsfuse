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

// A *CacheMissError value is an error that indicates an object name or a
// particular generation for that name were not found from cache.
type CacheMissError struct {
	Err error
}

func (cme *CacheMissError) Error() string {
	return fmt.Sprintf("CacheMissError: %v", cme.Err)
}

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
// insertListing caches all objects and sub-directories discovered during a GCS listing.
// It explicitly handles the "implicit directory" edge case where a directory exists
// only as a prefix to other objects.
func (b *fastStatBucket) insertListing(listing *gcs.Listing, dirName string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	expiration := b.clock.Now().Add(b.primaryCacheTTL)

	// Track object names to avoid redundant caching of prefixes
	// that already exist as explicit objects.
	minObjectNames := make(map[string]struct{})

	// 1. Parent Directory Inference (Implicit Check)
	// If the listing contains objects or sub-directories but the directory itself
	// is not returned as an explicit object, we infer and cache it as an
	// implicit directory.
	dirHasContents := len(listing.MinObjects) > 0 || len(listing.CollapsedRuns) > 0
	isDirInListing := len(listing.MinObjects) > 0 && listing.MinObjects[0].Name == dirName
	if dirHasContents && !isDirInListing {
		m := &gcs.MinObject{
			Name: dirName,
		}
		b.cache.Insert(m, expiration)
	}

	// 2. Cache Explicit Objects
	for _, o := range listing.MinObjects {
		b.cache.Insert(o, expiration)
		minObjectNames[o.Name] = struct{}{}
	}

	// 3. Cache Sub-directories (Collapsed Runs)
	// These represent folders discovered via prefixes in the ListObjects response.
	for _, p := range listing.CollapsedRuns {
		// Skip if this name was already cached as an explicit object.
		if _, exists := minObjectNames[p]; exists {
			continue
		}

		// Ensure the prefix follows directory naming conventions (trailing slash).
		// Although 'collapsedRuns' is expected to contain only directories, we perform
		// this defensive check to prevent processing malformed prefixes.
		if !strings.HasSuffix(p, "/") {
			logger.Errorf("fastStatBucket: ignoring malformed prefix name: %s", p)
			continue
		}

		// Cache the prefix as a minimal object (implicit directory marker).
		m := &gcs.MinObject{
			Name: p,
		}
		b.cache.Insert(m, expiration)
	}
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) insertMultipleMinObjects(minObjs []*gcs.MinObject) {
	b.mu.Lock()
	defer b.mu.Unlock()

	expiration := b.clock.Now().Add(b.primaryCacheTTL)
	for _, o := range minObjs {
		b.cache.Insert(o, expiration)
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
	b.insertMultipleMinObjects([]*gcs.MinObject{o})
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
	// TODO: create object to be replaced with create folder api once integrated
	o, err = b.wrapped.CreateObject(ctx, req)
	// Throw away any existing record for this object even if there was an error but do it after the API call.
	b.invalidate(req.Name)
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
	o, err := b.wrapped.FinalizeUpload(ctx, writer)
	// Throw away any existing record for this object.
	name := writer.ObjectName()
	b.invalidate(name)
	// Record the new object only if err is nil.
	if err == nil {
		b.insertMinObject(o)
	}

	return o, err
}

func (b *fastStatBucket) FlushPendingWrites(ctx context.Context, writer gcs.Writer) (*gcs.MinObject, error) {
	o, err := b.wrapped.FlushPendingWrites(ctx, writer)

	// Throw away any existing record for this object even if there was an error but do it after the API call.
	name := writer.ObjectName()
	b.invalidate(name)

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
	o, err = b.wrapped.CopyObject(ctx, req)
	// Throw away any existing record for the destination name even if there was an error but do it after the API call.
	b.invalidate(req.DstName)
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
	o, err = b.wrapped.ComposeObjects(ctx, req)
	// Throw away any existing record for the destination name even if there was an error but do it after the API call.
	b.invalidate(req.DstName)
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
				Err: fmt.Errorf("negative cache entry for %v", req.Name),
			}

			return
		}

		// Otherwise, return MinObject and nil ExtendedObjectAttributes.
		m = entry
		return
	}

	// Cache Miss Handling
	if req.FetchOnlyFromCache {
		return nil, nil, &CacheMissError{
			Err: fmt.Errorf("cache miss for %q", req.Name),
		}
	}

	// Standard fallback to GCS.
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

	if b.BucketType().Hierarchical {
		b.insertHierarchicalListing(listing)
		return
	}

	if req.IsTypeCacheDeprecated {
		b.insertListing(listing, req.Prefix)
	} else {
		// note anything we found.
		b.insertMultipleMinObjects(listing.MinObjects)
	}
	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *fastStatBucket) UpdateObject(
	ctx context.Context,
	req *gcs.UpdateObjectRequest) (o *gcs.Object, err error) {
	o, err = b.wrapped.UpdateObject(ctx, req)
	// Throw away any existing record for the destination name even if there was an error but do it after the API call.
	b.invalidate(req.Name)
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
	if req.OnlyDeleteFromCache {
		b.addNegativeEntry(req.Name)
		return nil
	}
	err = b.wrapped.DeleteObject(ctx, req)
	// In case of successful delete, add a negative entry to the cache.
	if err == nil {
		b.addNegativeEntry(req.Name)
		return
	}
	// If the delete failed due to a precondition error or not found error,
	// invalidate the cache entry as the object's state is uncertain.
	// For other errors, we don't touch the cache because the object likely
	// still exists.
	var preconditionErr *gcs.PreconditionError
	var notFoundErr *gcs.NotFoundError
	if errors.As(err, &preconditionErr) || errors.As(err, &notFoundErr) {
		b.invalidate(req.Name)
	}
	return
}

func (b *fastStatBucket) MoveObject(ctx context.Context, req *gcs.MoveObjectRequest) (*gcs.Object, error) {
	o, err := b.wrapped.MoveObject(ctx, req)
	// Throw away any existing record for the source and destination name even if there was an error but do it after the API call.
	b.invalidate(req.SrcName)
	b.invalidate(req.DstName)
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
	// Cache Lookup
	if hit, entry := b.lookUpFolder(req.Name); hit {
		// Negative entries result in NotFoundError.
		if entry == nil {
			err := &gcs.NotFoundError{
				Err: fmt.Errorf("negative cache entry for folder %q", req.Name),
			}

			return nil, err
		}

		return entry, nil
	}

	if req.FetchOnlyFromCache {
		return nil, &CacheMissError{
			Err: fmt.Errorf("cache miss for %q", req.Name),
		}
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
	f, err = b.wrapped.CreateFolder(ctx, folderName)
	// Throw away any existing record for this folder even if there was an error but do it after the API call.
	b.invalidate(folderName)
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

	var notFoundError *gcs.NotFoundError
	if errors.As(err, &notFoundError) {
		b.invalidate(req.Name)
	}
	return
}

func (b *fastStatBucket) GCSName(obj *gcs.MinObject) string {
	return b.wrapped.GCSName(obj)
}
