// Copyright 2015 Google LLC
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
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/bufferedwrites"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
	"golang.org/x/sync/semaphore"
)

// A GCS object metadata key for file mtimes. mtimes are UTC, and are stored in
// the format defined by time.RFC3339Nano.
const (
	FileMtimeMetadataKey = gcs.MtimeMetadataKey
	// TODO(b/447991081): Update streaming writes semantic message once semantics for ZB are updated on semantics doc.
	StreamingWritesSemantics = "Streaming writes is supported for sequential writes to new/empty files. " +
			"For more details, see: https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/semantics.md#writes"
)

type FileInode struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	bucket     *gcsx.SyncerBucket
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
	// INVARIANT: for non local files,  src.Name == name.GcsObjectName()
	//
	// GUARDED_BY(mu)
	src gcs.MinObject

	// The current content of this inode, or nil if the source object is still
	// authoritative.
	content gcsx.TempFile

	// Has Destroy been called?
	//
	// GUARDED_BY(mu)
	destroyed bool

	// Represents a local file which is not yet synced to GCS.
	local bool

	// Represents if local file has been unlinked.
	unlinked bool

	// Wrapper object for multi range downloader. Needed as we will create the MRD in
	// random reader and we can't pass fileInode object to random reader as it
	// creates a cyclic dependency.
	// Todo: Investigate if cyclic dependency can be removed by removing some unused
	// code.
	MRDWrapper *gcsx.MultiRangeDownloaderWrapper

	bwh    bufferedwrites.BufferedWriteHandler
	config *cfg.Config

	// Once write is started on the file i.e, bwh is initialized, any fileHandles
	// opened in write mode before or after this and not yet closed are considered
	// as writing to the file even though they are not writing.
	// In case of successful flush, we will set bwh to nil. But in case of error,
	// we will keep returning that error to all the fileHandles open during that time
	// and set bwh to nil after all fileHandlers are closed.
	// writeHandleCount tracks the count of open fileHandles in write mode.
	writeHandleCount int32

	// Limits the max number of blocks that can be created across file system when
	// streaming writes are enabled.
	globalMaxWriteBlocksSem *semaphore.Weighted

	mrdInstance gcsx.MrdInstance
}

var _ Inode = &FileInode{}

// Create a file inode for the given min object in GCS. The initial lookup count is
// zero.
//
// REQUIRES: m != nil
// REQUIRES: m.Generation > 0
// REQUIRES: m.MetaGeneration > 0
// REQUIRES: len(m.Name) > 0
// REQUIRES: m.Name[len(m.Name)-1] != '/'
func NewFileInode(
		id fuseops.InodeID,
		name Name,
		m *gcs.MinObject,
		attrs fuseops.InodeAttributes,
		bucket *gcsx.SyncerBucket,
		localFileCache bool,
		contentCache *contentcache.ContentCache,
		mtimeClock timeutil.Clock,
		localFile bool,
		cfg *cfg.Config,
		globalMaxBlocksSem *semaphore.Weighted,
		mrdCache *lru.Cache) (f *FileInode) {
	// Set up the basic struct.
	var minObj gcs.MinObject
	if m != nil {
		minObj = *m
	}
	f = &FileInode{
		bucket:                  bucket,
		mtimeClock:              mtimeClock,
		id:                      id,
		name:                    name,
		attrs:                   attrs,
		localFileCache:          localFileCache,
		contentCache:            contentCache,
		src:                     minObj,
		local:                   localFile,
		unlinked:                false,
		config:                  cfg,
		globalMaxWriteBlocksSem: globalMaxBlocksSem,
		mrdInstance:             gcsx.NewMrdInstance(),
	}
	var err error
	f.MRDWrapper, err = gcsx.NewMultiRangeDownloaderWrapper(bucket, &minObj, cfg, mrdCache)
	if err != nil {
		logger.Errorf("NewFileInode: Error in creating MRDWrapper %v", err)
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

	// INVARIANT: For non-local inodes, src.Name == name
	if !f.IsLocal() && f.src.Name != name.GcsObjectName() {
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
func (f *FileInode) clobbered(ctx context.Context, forceFetchFromGcs bool, includeExtendedObjectAttributes bool) (o *gcs.Object, b bool, err error) {
	// Stat the object in GCS. ForceFetchFromGcs ensures object is fetched from
	// gcs and not cache.
	req := &gcs.StatObjectRequest{
		Name:                           f.name.GcsObjectName(),
		ForceFetchFromGcs:              forceFetchFromGcs,
		ReturnExtendedObjectAttributes: includeExtendedObjectAttributes,
	}
	m, e, err := f.bucket.StatObject(ctx, req)
	if includeExtendedObjectAttributes {
		o = storageutil.ConvertMinObjectAndExtendedObjectAttributesToObject(m, e)
	} else {
		o = storageutil.ConvertMinObjectToObject(m)
	}
	// Special case: "not found" means we have been clobbered.
	var notFoundErr *gcs.NotFoundError
	if errors.As(err, &notFoundErr) {
		err = nil
		if f.IsLocal() {
			// For localFile, it is expected that object doesn't exist in GCS.
			return
		}

		b = true
		return
	}

	// Propagate other errors.
	if err != nil {
		err = fmt.Errorf("StatObject: %w", err)
		return
	}

	// We are clobbered iff the generation doesn't match our source generation.
	oGen := Generation{o.Generation, o.MetaGeneration, o.Size}
	b = oGen.Compare(f.SourceGeneration()) != 0

	return
}

// Open a reader for the generation of object we care about.
func (f *FileInode) openReader(ctx context.Context) (io.ReadCloser, error) {
	rc, err := f.bucket.NewReaderWithReadHandle(
		ctx,
		&gcs.ReadObjectRequest{
			Name:           f.src.Name,
			Generation:     f.src.Generation,
			ReadCompressed: f.src.HasContentEncodingGzip(),
		})
	// If the object with requested generation doesn't exist in GCS, it indicates
	// a file clobbering scenario. This likely occurred because the file was
	// modified/deleted leading to different generation number.
	var notFoundError *gcs.NotFoundError
	if errors.As(err, &notFoundError) {
		err = &gcsfuse_errors.FileClobberedError{
			Err:        fmt.Errorf("NewReader: %w", err),
			ObjectName: f.src.Name,
		}
	}
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

func (f *FileInode) IsLocal() bool {
	return f.local
}

func (f *FileInode) IsUnlinked() bool {
	return f.unlinked
}

func (f *FileInode) Unlink() {
	f.unlinked = true

	if f.bwh != nil {
		f.bwh.Unlink()
	}
}

// Returns true if the fileInode is using Buffered Write Handler.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) IsUsingBWH() bool {
	return f.bwh != nil
}

// Source returns a record for the GCS object from which this inode is branched. The
// record is guaranteed not to be modified, and users must not modify it.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Source() *gcs.MinObject {
	// Make a copy, since we modify f.src.
	o := f.src
	return &o
}

// If true, it is safe to serve reads directly from the object given by
// f.Source(), rather than calling f.ReadAt. Doing so may be more efficient,
// because f.ReadAt may cause the entire object to be faulted in and requires
// the inode to be locked during the read. SourceGenerationAuthoritative requires
// SyncPendingBufferedWrites method has been called on f within same inode lock for
// streaming writes with zonal bucket.
// TODO(b/406160290): Check if this can be improved.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) SourceGenerationIsAuthoritative() bool {
	// Source generation is authoritative if:
	//   1.  No pending writes exists on the inode (both content and bwh are nil).
	//   2.  The bucket is zonal and there are no pending writes in the temporary file.
	return (f.content == nil && f.bwh == nil) || (f.bucket.BucketType().Zonal && f.content == nil)
}

// Equivalent to the generation returned by f.Source().
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) SourceGeneration() (g Generation) {
	g.Size = f.src.Size
	g.Object = f.src.Generation
	g.Metadata = f.src.MetaGeneration
	// If bwh is not nil, it's size takes precedence as that is being actively
	// written to GCS.
	// Since temporary file does not write to GCS on the go, it's size is not
	// used as source object's size.
	if f.bwh != nil {
		writeFileInfo := f.bwh.WriteFileInfo()
		g.Size = uint64(writeFileInfo.TotalSize)
	}
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
func (f *FileInode) RegisterFileHandle(readOnly bool) {
	if !readOnly {
		f.writeHandleCount++
	}
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) DeRegisterFileHandle(readOnly bool) {
	if readOnly {
		return
	}

	if f.writeHandleCount <= 0 {
		logger.Errorf("Mismatch in number of write file handles for inode :%d", f.id)
	}

	f.writeHandleCount--

	// All write fileHandles associated with bwh are closed. So safe to set bwh to nil.
	if f.writeHandleCount == 0 && f.bwh != nil {
		err := f.bwh.Destroy()
		if err != nil {
			logger.Warnf("Error while destroying the bufferedWritesHandler: %v", err)
		}
		f.bwh = nil
	}
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
		ctx context.Context, clobberedCheck bool) (attrs fuseops.InodeAttributes, err error) {
	attrs = f.attrs
	// Obtain default information from the source object.
	attrs.Mtime = f.src.Updated
	attrs.Size = f.src.Size

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
			err = fmt.Errorf("stat: %w", err)
			return
		}

		attrs.Size = uint64(sr.Size)
		if sr.Mtime != nil {
			attrs.Mtime = *sr.Mtime
		}
	}

	if f.bwh != nil {
		writeFileInfo := f.bwh.WriteFileInfo()
		attrs.Mtime = writeFileInfo.Mtime
		attrs.Size = uint64(writeFileInfo.TotalSize)
	}

	// We require only that atime and ctime be "reasonable".
	attrs.Atime = attrs.Mtime
	attrs.Ctime = attrs.Mtime

	if clobberedCheck {
		// If the object has been clobbered, we reflect that as the inode being
		// unlinked.
		var clobbered bool
		_, clobbered, err = f.clobbered(ctx, false, false)
		if err != nil {
			err = fmt.Errorf("clobbered: %w", err)
			return
		}
		if clobbered {
			attrs.Nlink = 0
			return
		}
	}

	attrs.Nlink = 1

	// For local files, also checking if file is unlinked locally.
	if f.IsLocal() && f.IsUnlinked() {
		attrs.Nlink = 0
	}

	return
}

func (f *FileInode) Bucket() *gcsx.SyncerBucket {
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
	if f.bwh != nil {
		err = fmt.Errorf("unexpected read call for %q when streaming write is in progress for it", f.Name().LocalName())
		return
	}

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
// It returns true if the file is successfully synced during the write operation.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Write(
		ctx context.Context,
		data []byte,
		offset int64,
		openMode util.OpenMode) (bool, error) {
	if f.bwh != nil {
		return f.writeUsingBufferedWrites(ctx, data, offset)
	}

	return false, f.writeUsingTempFile(ctx, data, offset)
}

// Helper function to serve write for file using temp file.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) writeUsingTempFile(ctx context.Context, data []byte, offset int64) (err error) {
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

// Helper function to serve write for file using buffered writes handler.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) writeUsingBufferedWrites(ctx context.Context, data []byte, offset int64) (bool, error) {
	err := f.bwh.Write(data, offset)
	var preconditionErr *gcs.PreconditionError
	if errors.As(err, &preconditionErr) {
		return false, &gcsfuse_errors.FileClobberedError{
			Err:        fmt.Errorf("f.bwh.Write(): %w", err),
			ObjectName: f.src.Name,
		}
	}
	// Fall back to temp file for Out-Of-Order Writes.
	if errors.Is(err, bufferedwrites.ErrOutOfOrderWrite) {
		logger.Infof("Out of order write detected. File %s will now use legacy staged writes. "+StreamingWritesSemantics, f.name.String())
		// Finalize the object.
		err = f.flushUsingBufferedWriteHandler()
		if err != nil {
			return false, fmt.Errorf("could not finalize what has been written so far: %w", err)
		}
		return true, f.writeUsingTempFile(ctx, data, offset)
	}
	if err != nil {
		return false, fmt.Errorf("write to buffered write handler failed: %w", err)
	}
	return false, nil
}

// Helper function to flush buffered writes handler and update inode state with
// new object.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) flushUsingBufferedWriteHandler() error {
	obj, err := f.bwh.Flush()
	var preconditionErr *gcs.PreconditionError
	if errors.As(err, &preconditionErr) {
		return &gcsfuse_errors.FileClobberedError{
			Err:        fmt.Errorf("f.bwh.Flush(): %w", err),
			ObjectName: f.src.Name,
		}
	}
	if err != nil {
		return fmt.Errorf("f.bwh.Flush(): %w", err)
	}
	// If we finalized the object, we need to update our state.
	f.updateInodeStateAfterFlush(obj)
	return nil
}

// SyncPendingBufferedWrites flushes any pending writes on the bwh to GCS.
// It is a no-op when bwh is nil.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) SyncPendingBufferedWrites() (gcsSynced bool, err error) {
	if f.bwh == nil {
		return
	}
	minObj, err := f.bwh.Sync()
	var preconditionErr *gcs.PreconditionError
	if errors.As(err, &preconditionErr) {
		err = &gcsfuse_errors.FileClobberedError{
			Err:        fmt.Errorf("f.bwh.Sync(): %w", err),
			ObjectName: f.src.Name,
		}
		return
	}
	if err != nil {
		err = fmt.Errorf("f.bwh.Sync(): %w", err)
		return
	}
	// We return gcsSynced as true when we get minObject from Sync() for Zonal Buckets.
	// For Non-Zonal Buckets minObj is always nil.
	gcsSynced = minObj != nil
	// If we flushed out object, we need to update our state.
	f.updateInodeStateAfterSync(minObj)
	return
}

// Set the mtime for this file. May involve a round trip to GCS.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) SetMtime(
		ctx context.Context,
		mtime time.Time) (err error) {
	if f.IsUnlinked() {
		// No need to update mtime on GCS for unlinked file.
		return
	}

	// When bufferedWritesHandler instance is not nil, set time on bwh.
	// It will not be nil in 2 cases when bufferedWrites are enabled:
	// 1. local files
	// 2. After first write on empty GCS files.
	if f.bwh != nil {
		f.bwh.SetMtime(mtime)
		return
	}

	// If we have a local temp file, stat it.
	var sr gcsx.StatResult
	if f.content != nil {
		sr, err = f.content.Stat()
		if err != nil {
			err = fmt.Errorf("stat: %w", err)
			return
		}
	}

	// 1. If the local content is dirty, simply update its mtime and return. This
	// will cause the object in the bucket to be updated once we sync. If we lose
	// power or something the mtime update will be lost, but so will the file
	// data modifications so this doesn't seem so bad. It's worth saving the
	// round trip to GCS for the common case of Linux writeback caching, where we
	// always receive a setattr request just before a flush of a dirty file.
	//
	// 2. If the file is local, that means its not yet synced to GCS. Just update
	// the mtime locally, it will be synced when the object is created on GCS.
	if sr.Mtime != nil || f.IsLocal() {
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
		var minObj gcs.MinObject
		minObjPtr := storageutil.ConvertObjToMinObject(o)
		if minObjPtr != nil {
			minObj = *minObjPtr
		}
		f.src = minObj
		f.updateMRDWrapper()
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

func (f *FileInode) fetchLatestGcsObject(ctx context.Context) (*gcs.Object, error) {
	// When listObjects call is made, we fetch data with projection set as noAcl
	// which means acls and owner properties are not returned. So the f.src object
	// here will not have acl information even though there are acls present on
	// the gcsObject.
	// Hence, we are making an explicit gcs stat call to fetch the latest
	// properties and using that when object is synced below. StatObject by
	// default sets the projection to full, which fetches all the object
	// properties.
	latestGcsObj, isClobbered, err := f.clobbered(ctx, true, true)
	if err != nil {
		return nil, err
	}
	if isClobbered {
		return nil, &gcsfuse_errors.FileClobberedError{
			Err:        fmt.Errorf("file was clobbered"),
			ObjectName: f.src.Name,
		}
	}
	return latestGcsObj, nil
}

// Sync writes out contents to GCS.  If this fails due to the generation
// having been clobbered, failure is propagated back to the calling
// function as an error.
//
// For buffered writes, this method only waits for any partial buffers to be
// uploaded to GCS. It does not guarantee that the entire contents of the file
// have been persisted.
//
// For non-buffered writes, this method writes the entire contents to GCS.
// If this method succeeds, SourceGeneration will return the new generation by
// which this inode should be known (which may be the same as before). If it
// fails, the generation will not change.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Sync(ctx context.Context) (gcsSynced bool, err error) {
	// If we have not been dirtied, there is nothing to do.
	if f.content == nil && f.bwh == nil {
		return
	}

	if f.bwh != nil {
		gcsSynced, err = f.SyncPendingBufferedWrites()
		if err != nil {
			err = fmt.Errorf("could not sync what has been written so far: %w", err)
		}
		return
	}
	err = f.syncUsingContent(ctx)
	if err != nil {
		return false, err
	}
	return true, nil
}

// syncUsingContent syncs the inode content to GCS. It fetches the latest GCS
// object, syncs the content and updates the inode state.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) syncUsingContent(ctx context.Context) error {
	var latestGcsObj *gcs.Object
	if !f.local {
		var err error
		latestGcsObj, err = f.fetchLatestGcsObject(ctx)
		if err != nil {
			return err
		}
	}

	// Write out the contents if they are dirty.
	// Object properties are also synced as part of content sync. Hence, passing
	// the latest object fetched from gcs which has all the properties populated.
	newObj, err := f.bucket.SyncObject(ctx, f.Name().GcsObjectName(), latestGcsObj, f.content)

	var preconditionErr *gcs.PreconditionError
	if errors.As(err, &preconditionErr) {
		return &gcsfuse_errors.FileClobberedError{
			Err:        fmt.Errorf("SyncObject: %w", err),
			ObjectName: f.src.Name,
		}
	}

	// Propagate other errors.
	if err != nil {
		return fmt.Errorf("SyncObject: %w", err)
	}
	minObj := storageutil.ConvertObjToMinObject(newObj)
	// If we wrote out a new object, we need to update our state.
	f.updateInodeStateAfterFlush(minObj)
	return nil
}

// Flush writes out contents to GCS. If this fails due to the generation
// having been clobbered, failure is propagated back to the calling
// function as an error.
//
// After this method succeeds, SourceGeneration will return the new generation
// by which this inode should be known (which may be the same as before). If it
// fails, the generation will not change.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Flush(ctx context.Context) (err error) {
	// If we have not been dirtied, there is nothing to do.
	if f.content == nil && f.bwh == nil {
		return
	}

	// Flush using the appropriate method based on whether we're using a
	// buffered write handler.
	if f.bwh != nil {
		return f.flushUsingBufferedWriteHandler()
	}
	return f.syncUsingContent(ctx)
}

func (f *FileInode) updateInodeStateAfterFlush(minObj *gcs.MinObject) {
	if minObj != nil && !f.localFileCache {
		// Set BWH to nil as as object has been finalized.
		if f.bwh != nil {
			f.bwh = nil
		}
		f.updateInodeStateAfterSync(minObj)
	}
}

func (f *FileInode) updateInodeStateAfterSync(minObj *gcs.MinObject) {
	if minObj != nil && !f.localFileCache {
		f.src = *minObj
		// Update MRDWrapper
		f.updateMRDWrapper()
		// Convert localFile to nonLocalFile after it is synced to GCS.
		if f.IsLocal() {
			f.local = false
		}
		if f.content != nil {
			f.content.Destroy()
			f.content = nil
		}
	}
}

// Updates the min object stored in MRDWrapper corresponding to the inode.
// Should be called when minObject associated with inode is updated.
func (f *FileInode) updateMRDWrapper() {
	err := f.MRDWrapper.SetMinObject(f.Source())
	if err != nil {
		logger.Errorf("FileInode::updateMRDWrapper Error in setting minObject %v", err)
	}
}

// truncateUsingBufferedWriteHandler attempts to truncate the file using the buffered
// write handler. If the requested size is smaller than the file's current size,
// it finalizes the existing writes, falls back to using a temporary file for
// the truncation, and returns true to indicate the file has been synced.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) truncateUsingBufferedWriteHandler(ctx context.Context, size int64) (bool, error) {
	err := f.bwh.Truncate(size)
	// If truncate size is less than the total file size resulting in OutOfOrder write, finalize and fall back to temp file.
	if errors.Is(err, bufferedwrites.ErrOutOfOrderWrite) {
		logger.Infof("Out of order write detected. File %s will now use legacy staged writes. "+StreamingWritesSemantics, f.name.String())
		// Finalize the object.
		err = f.flushUsingBufferedWriteHandler()
		if err != nil {
			return false, fmt.Errorf("could not finalize what has been written so far: %w", err)
		}
		return true, f.truncateUsingTempFile(ctx, size)
	}
	if err != nil {
		return false, fmt.Errorf("got unexpected error from bwh.Truncate(): %w", err)
	}
	return false, nil
}

// truncateUsingTempFile truncates the file by first ensuring a temporary file
// is available and then truncating that temporary file to the specified size.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) truncateUsingTempFile(ctx context.Context, size int64) error {
	// Make sure f.content != nil.
	err := f.ensureContent(ctx)
	if err != nil {
		return fmt.Errorf("ensureContent: %w", err)
	}
	// Truncate temp file.
	return f.content.Truncate(size)
}

// Truncate the file to the specified size.
// It returns true if the file has been successfully synced to GCS.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Truncate(
		ctx context.Context,
		size int64) (bool, error) {
	if f.IsUsingBWH() {
		return f.truncateUsingBufferedWriteHandler(ctx, size)
	}
	return false, f.truncateUsingTempFile(ctx, size)
}

// Ensures cache content on read if content cache enabled
func (f *FileInode) CacheEnsureContent(ctx context.Context) (err error) {
	if f.localFileCache {
		err = f.ensureContent(ctx)
	}

	return
}

// CreateEmptyTempFile creates an empty file with no contents when
// streaming writes are not in enabled.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) CreateEmptyTempFile(ctx context.Context) (err error) {
	// Skip creating empty temp file when streaming writes are enabled
	// or temp file is already created.
	if f.bwh != nil || f.content != nil {
		return
	}

	// Creating a file with no contents. The contents will be updated with
	// writeFile operations.
	f.content, err = f.contentCache.NewTempFile(io.NopCloser(strings.NewReader("")))
	// Setting the initial mtime to creation time.
	f.content.SetMtime(f.mtimeClock.Now())
	return
}

// Initializes Buffered Write Handler if the file inode is eligible and returns
// initialized as true when the new instance of buffered writer handler is created.
func (f *FileInode) InitBufferedWriteHandlerIfEligible(ctx context.Context, openMode util.OpenMode) (bool, error) {
	// bwh already initialized, do nothing.
	if f.bwh != nil {
		return false, nil
	}

	tempFileInUse := f.content != nil
	if !f.config.Write.EnableStreamingWrites || tempFileInUse {
		// bwh should not be initialized under these conditions.
		return false, nil
	}

	var latestGcsObj *gcs.Object
	var err error
	if !f.local {
		latestGcsObj, err = f.fetchLatestGcsObject(ctx)
		if err != nil {
			return false, err
		}
	}

	if !f.areBufferedWritesSupported(openMode, latestGcsObj) {
		return false, nil
	}

	if f.bwh == nil {
		f.bwh, err = bufferedwrites.NewBWHandler(&bufferedwrites.CreateBWHandlerRequest{
			Object:                   latestGcsObj,
			ObjectName:               f.name.GcsObjectName(),
			Bucket:                   f.bucket,
			BlockSize:                f.config.Write.BlockSizeMb * util.MiB,
			MaxBlocksPerFile:         f.config.Write.MaxBlocksPerFile,
			GlobalMaxBlocksSem:       f.globalMaxWriteBlocksSem,
			ChunkTransferTimeoutSecs: f.config.GcsRetries.ChunkTransferTimeoutSecs,
		})
		if errors.Is(err, block.CantAllocateAnyBlockError) {
			logger.Warnf("File %s will use legacy staged writes because concurrent streaming write "+
					"limit (set by --write-global-max-blocks) has been reached. To allow more concurrent files "+
					"to use streaming writes, consider increasing this limit if sufficient memory is available. "+
					"For more details on memory usage, see: https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/semantics.md#writes", f.name.String())
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("failed to create bufferedWriteHandler: %w", err)
		}
		f.bwh.SetMtime(f.mtimeClock.Now())
		return true, nil
	}
	return false, nil
}

func (f *FileInode) areBufferedWritesSupported(openMode util.OpenMode, obj *gcs.Object) bool {
	// For new files and existing files of size 0, buffered writes are always supported.
	if f.local || obj.Size == 0 {
		return true
	}
	if f.config.Write.EnableRapidAppends && openMode.IsAppend() && f.bucket.BucketType().Zonal && obj.Finalized.IsZero() {
		return true
	}
	logger.Infof("Existing file %s of size %d bytes (non-zero) will use legacy staged writes. "+StreamingWritesSemantics, f.name.String(), obj.Size)
	return false
}
