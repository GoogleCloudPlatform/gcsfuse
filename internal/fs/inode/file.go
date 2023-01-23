// Copyright 2015 Google Inc. All Rights Reserved.
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

package inode

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

// A GCS object metadata key for file mtimes. mtimes are UTC, and are stored in
// the format defined by time.RFC3339Nano.
const FileMtimeMetadataKey = gcsx.MtimeMetadataKey

type FileInode struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	bucket     gcsx.SyncerBucket
	mtimeClock timeutil.Clock

	/////////////////////////
	// Constant data
	/////////////////////////

	id           fuseops.InodeID
	name         Name
	attrs        fuseops.InodeAttributes
	contentCache *contentcache.ContentCache
	// TODO (#640) remove bool flag and refactor contentCache to support two implementations:
	// one implementation with original functionality and one with new persistent disk content cache
	localFileCache bool

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A mutex that must be held when calling certain methods. See documentation
	// for each method.
	mu syncutil.InvariantMutex

	// GUARDED_BY(mu)
	lc lookupCount

	// The source object from which this inode derives.
	//
	// INVARIANT: src.Name == name.GcsObjectName()
	//
	// GUARDED_BY(mu)
	src storage.MinObject

	// The current content of this inode, or nil if the source object is still
	// authoritative.
	content gcsx.TempFile

	// Has Destroy been called?
	//
	// GUARDED_BY(mu)
	destroyed bool
}

var _ Inode = &FileInode{}

// Create a file inode for the given object in GCS. The initial lookup count is
// zero.
//
// REQUIRES: o != nil
// REQUIRES: o.Generation > 0
// REQUIRES: o.MetaGeneration > 0
// REQUIRES: len(o.Name) > 0
// REQUIRES: o.Name[len(o.Name)-1] != '/'
func NewFileInode(
	id fuseops.InodeID,
	name Name,
	o *gcs.Object,
	attrs fuseops.InodeAttributes,
	bucket gcsx.SyncerBucket,
	localFileCache bool,
	contentCache *contentcache.ContentCache,
	mtimeClock timeutil.Clock) (f *FileInode) {
	// Set up the basic struct.
	f = &FileInode{
		bucket:         bucket,
		mtimeClock:     mtimeClock,
		id:             id,
		name:           name,
		attrs:          attrs,
		localFileCache: localFileCache,
		contentCache:   contentCache,
		src:            convertObjToMinObject(o),
	}

	f.lc.Init(id)

	// Set up invariant checking.
	f.mu = syncutil.NewInvariantMutex(f.checkInvariants)

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) checkInvariants() {
	if f.destroyed {
		return
	}

	// Make sure the name is legal.
	name := f.Name()
	if !name.IsFile() {
		panic("Illegal file name: " + name.String())
	}

	// INVARIANT: src.Name == name
	if f.src.Name != name.GcsObjectName() {
		panic(fmt.Sprintf(
			"Name mismatch: %q vs. %q",
			f.src.Name,
			name.GcsObjectName(),
		))
	}

	// INVARIANT: content.CheckInvariants() does not panic
	if f.content != nil {
		f.content.CheckInvariants()
	}
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) clobbered(ctx context.Context, forceFetchFromGcs bool) (o *gcs.Object, b bool, err error) {
	// Stat the object in GCS. ForceFetchFromGcs ensures object is fetched from
	// gcs and not cache.
	req := &gcs.StatObjectRequest{
		Name:              f.name.GcsObjectName(),
		ForceFetchFromGcs: forceFetchFromGcs,
	}
	o, err = f.bucket.StatObject(ctx, req)

	// Special case: "not found" means we have been clobbered.
	var notFoundErr *gcs.NotFoundError
	if errors.As(err, &notFoundErr) {
		err = nil
		b = true
		return
	}

	// Propagate other errors.
	if err != nil {
		err = fmt.Errorf("StatObject: %w", err)
		return
	}

	// We are clobbered iff the generation doesn't match our source generation.
	oGen := Generation{o.Generation, o.MetaGeneration}
	b = f.SourceGeneration().Compare(oGen) != 0

	return
}

// Open a reader for the generation of object we care about.
func (f *FileInode) openReader(ctx context.Context) (io.ReadCloser, error) {
	rc, err := f.bucket.NewReader(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       f.src.Name,
			Generation: f.src.Generation,
		})
	if err != nil {
		err = fmt.Errorf("NewReader: %w", err)
	}
	return rc, err
}

// Ensure that content exists and is not stale
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) ensureContent(ctx context.Context) (err error) {
	if f.localFileCache {
		// Fetch content from the cache after validating generation numbers again
		// Generation validation first occurs at inode creation/destruction
		cacheObjectKey := &contentcache.CacheObjectKey{BucketName: f.bucket.Name(), ObjectName: f.name.objectName}
		if cacheObject, exists := f.contentCache.Get(cacheObjectKey); exists {
			if cacheObject.ValidateGeneration(f.src.Generation, f.src.MetaGeneration) {
				f.content = cacheObject.CacheFile
				return
			}
		}

		rc, err := f.openReader(ctx)
		if err != nil {
			err = fmt.Errorf("openReader Error: %w", err)
			return err
		}

		// Insert object into content cache
		tf, err := f.contentCache.AddOrReplace(cacheObjectKey, f.src.Generation, f.src.MetaGeneration, rc)
		if err != nil {
			err = fmt.Errorf("AddOrReplace cache error: %w", err)
			return err
		}

		// Update state.
		f.content = tf.CacheFile
	} else {
		// Local filecache is not enabled
		if f.content != nil {
			return
		}

		rc, err := f.openReader(ctx)
		if err != nil {
			err = fmt.Errorf("openReader Error: %w", err)
			return err
		}

		tf, err := f.contentCache.NewTempFile(rc)
		if err != nil {
			err = fmt.Errorf("NewTempFile: %w", err)
			return err
		}
		// Update state.
		f.content = tf
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (f *FileInode) Lock() {
	f.mu.Lock()
}

func (f *FileInode) Unlock() {
	f.mu.Unlock()
}

func (f *FileInode) ID() fuseops.InodeID {
	return f.id
}

func (f *FileInode) Name() Name {
	return f.name
}

// Source returns a record for the GCS object from which this inode is branched. The
// record is guaranteed not to be modified, and users must not modify it.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Source() *storage.MinObject {
	// Make a copy, since we modify f.src.
	o := f.src
	return &o
}

// If true, it is safe to serve reads directly from the object given by
// f.Source(), rather than calling f.ReadAt. Doing so may be more efficient,
// because f.ReadAt may cause the entire object to be faulted in and requires
// the inode to be locked during the read.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) SourceGenerationIsAuthoritative() bool {
	return f.content == nil
}

// Equivalent to the generation returned by f.Source().
//
// LOCKS_REQUIRED(f)
func (f *FileInode) SourceGeneration() (g Generation) {
	g.Object = f.src.Generation
	g.Metadata = f.src.MetaGeneration
	return
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) IncrementLookupCount() {
	f.lc.Inc()
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) DecrementLookupCount(n uint64) (destroy bool) {
	destroy = f.lc.Dec(n)
	return
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Destroy() (err error) {
	f.destroyed = true
	if f.localFileCache {
		cacheObjectKey := &contentcache.CacheObjectKey{BucketName: f.bucket.Name(), ObjectName: f.name.objectName}
		f.contentCache.Remove(cacheObjectKey)
	} else if f.content != nil {
		f.content.Destroy()
	}
	return
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Attributes(
	ctx context.Context) (attrs fuseops.InodeAttributes, err error) {
	attrs = f.attrs

	// Obtain default information from the source object.
	attrs.Mtime = f.src.Updated
	attrs.Size = uint64(f.src.Size)

	// We require only that atime and ctime be "reasonable".
	attrs.Atime = attrs.Mtime
	attrs.Ctime = attrs.Mtime

	// If the source object has an mtime metadata key, use that instead of its
	// update time.
	// If the file was copied via gsutil, we'll have goog-reserved-file-mtime
	if strTimestamp, ok := f.src.Metadata["goog-reserved-file-mtime"]; ok {
		if timestamp, err := strconv.ParseInt(strTimestamp, 0, 64); err == nil {
			attrs.Mtime = time.Unix(timestamp, 0)
		}
	}

	// Otherwise, if its been synced with gcsfuse before, we'll have gcsfuse_mtime
	if formatted, ok := f.src.Metadata["gcsfuse_mtime"]; ok {
		attrs.Mtime, err = time.Parse(time.RFC3339Nano, formatted)
		if err != nil {
			err = fmt.Errorf("time.Parse(%q): %w", formatted, err)
			return
		}
	}

	// If we've got local content, its size and (maybe) mtime take precedence.
	if f.content != nil {
		var sr gcsx.StatResult
		sr, err = f.content.Stat()
		if err != nil {
			err = fmt.Errorf("Stat: %w", err)
			return
		}

		attrs.Size = uint64(sr.Size)
		if sr.Mtime != nil {
			attrs.Mtime = *sr.Mtime
		}
	}

	// If the object has been clobbered, we reflect that as the inode being
	// unlinked.
	_, clobbered, err := f.clobbered(ctx, false)
	if err != nil {
		err = fmt.Errorf("clobbered: %w", err)
		return
	}

	if !clobbered {
		attrs.Nlink = 1
	}

	return
}

func (f *FileInode) Bucket() gcsx.SyncerBucket {
	return f.bucket
}

// Serve a read for this file with semantics matching io.ReaderAt.
//
// The caller may be better off reading directly from GCS when
// f.SourceGenerationIsAuthoritative() is true.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Read(
	ctx context.Context,
	dst []byte,
	offset int64) (n int, err error) {
	// Make sure f.content != nil.
	err = f.ensureContent(ctx)
	if err != nil {
		err = fmt.Errorf("ensureContent: %w", err)
		return
	}

	// Read from the local content, propagating io.EOF.
	n, err = f.content.ReadAt(dst, offset)
	switch {
	case err == io.EOF:
		return

	case err != nil:
		err = fmt.Errorf("content.ReadAt: %w", err)
		return
	}

	return
}

// Serve a write for this file with semantics matching fuseops.WriteFileOp.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Write(
	ctx context.Context,
	data []byte,
	offset int64) (err error) {
	// Make sure f.content != nil.
	err = f.ensureContent(ctx)
	if err != nil {
		err = fmt.Errorf("ensureContent: %w", err)
		return
	}

	// Write to the mutable content. Note that io.WriterAt guarantees it returns
	// an error for short writes.
	_, err = f.content.WriteAt(data, offset)

	return
}

// Set the mtime for this file. May involve a round trip to GCS.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) SetMtime(
	ctx context.Context,
	mtime time.Time) (err error) {
	// If we have a local temp file, stat it.
	var sr gcsx.StatResult
	if f.content != nil {
		sr, err = f.content.Stat()
		if err != nil {
			err = fmt.Errorf("Stat: %w", err)
			return
		}
	}

	// If the local content is dirty, simply update its mtime and return. This
	// will cause the object in the bucket to be updated once we sync. If we lose
	// power or something the mtime update will be lost, but so will the file
	// data modifications so this doesn't seem so bad. It's worth saving the
	// round trip to GCS for the common case of Linux writeback caching, where we
	// always receive a setattr request just before a flush of a dirty file.
	if sr.Mtime != nil {
		f.content.SetMtime(mtime)
		return
	}

	// Otherwise, update the backing object's metadata.
	formatted := mtime.UTC().Format(time.RFC3339Nano)
	srcGen := f.SourceGeneration()

	req := &gcs.UpdateObjectRequest{
		Name:                       f.src.Name,
		Generation:                 srcGen.Object,
		MetaGenerationPrecondition: &srcGen.Metadata,
		Metadata: map[string]*string{
			FileMtimeMetadataKey: &formatted,
		},
	}

	o, err := f.bucket.UpdateObject(ctx, req)
	if err == nil {
		f.src = convertObjToMinObject(o)
		return
	}

	var notFoundErr *gcs.NotFoundError
	if errors.As(err, &notFoundErr) {
		// Special case: silently ignore not found errors, which mean the file has
		// been unlinked.
		err = nil
		return
	}

	var preconditionErr *gcs.PreconditionError
	if errors.As(err, &preconditionErr) {
		// Special case: silently ignore precondition errors, which we also take to
		// mean the file has been unlinked.
		err = nil
		return
	}

	err = fmt.Errorf("UpdateObject: %w", err)
	return
}

// Sync writes out contents to GCS. If this fails due to the generation having been
// clobbered, treat it as a non-error (simulating the inode having been
// unlinked).
//
// After this method succeeds, SourceGeneration will return the new generation
// by which this inode should be known (which may be the same as before). If it
// fails, the generation will not change.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Sync(ctx context.Context) (err error) {
	// If we have not been dirtied, there is nothing to do.
	if f.content == nil {
		return
	}

	// When listObjects call is made, we fetch data with projection set as noAcl
	// which means acls and owner properties are not returned. So the f.src object
	// here will not have acl information even though there are acls present on
	// the gcsObject.
	// Hence, we are making an explicit gcs stat call to fetch the latest
	// properties and using that when object is synced below. StatObject by
	// default sets the projection to full, which fetches all the object
	// properties.
	latestGcsObj, isClobbered, err := f.clobbered(ctx, true)

	// Clobbered is treated as being unlinked. There's no reason to return an
	// error in that case. We simply return without syncing the object.
	if err != nil || isClobbered {
		return
	}

	// Write out the contents if they are dirty.
	// Object properties are also synced as part of content sync. Hence, passing
	// the latest object fetched from gcs which has all the properties populated.
	newObj, err := f.bucket.SyncObject(ctx, latestGcsObj, f.content)

	// Special case: a precondition error means we were clobbered, which we treat
	// as being unlinked. There's no reason to return an error in that case.
	var preconditionErr *gcs.PreconditionError
	if errors.As(err, &preconditionErr) {
		err = nil
		return
	}

	// Propagate other errors.
	if err != nil {
		err = fmt.Errorf("SyncObject: %w", err)
		return
	}

	// If we wrote out a new object, we need to update our state.
	if newObj != nil && !f.localFileCache {
		f.src = convertObjToMinObject(newObj)
		f.content.Destroy()
		f.content = nil
	}

	return
}

// Truncate the file to the specified size.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Truncate(
	ctx context.Context,
	size int64) (err error) {
	// Make sure f.content != nil.
	err = f.ensureContent(ctx)
	if err != nil {
		err = fmt.Errorf("ensureContent: %w", err)
		return
	}

	// Call through.
	err = f.content.Truncate(size)

	return
}

// Ensures cache content on read if content cache enabled
func (f *FileInode) CacheEnsureContent(ctx context.Context) (err error) {
	if f.localFileCache {
		err = f.ensureContent(ctx)
	}

	return
}

func convertObjToMinObject(o *gcs.Object) (mo storage.MinObject) {
	return storage.MinObject{
		Name:           o.Name,
		Size:           o.Size,
		Generation:     o.Generation,
		MetaGeneration: o.MetaGeneration,
		Updated:        o.Updated,
		Metadata:       o.Metadata,
	}
}
