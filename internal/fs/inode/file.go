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
	"fmt"
	"io"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/internal/util"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
	"log"
	"sync"
)

// A GCS object metadata key for file mtimes. mtimes are UTC, and are stored in
// the format defined by time.RFC3339Nano.
const FileMtimeMetadataKey = gcsx.MtimeMetadataKey

type FileInode struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	bucket     gcs.Bucket
	syncer     gcsx.Syncer
	mtimeClock timeutil.Clock

	/////////////////////////
	// Constant data
	/////////////////////////

	id                fuseops.InodeID
	name              string
	attrs             fuseops.InodeAttributes
	tempDir           string
	cacheRemovalDelay time.Duration

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
	// INVARIANT: src.Name == name
	//
	// GUARDED_BY(mu)
	src gcs.Object

	// The current content of this inode, or nil if the source object is still
	// authoritative.
	content gcsx.TempFile

	rmu sync.Mutex
	sourceReader io.ReadSeeker

	// Has Destroy been called?
	//
	// GUARDED_BY(mu)
	destroyed bool

	sc               *util.Schedule
	syncRequired     bool
	cleanupScheduled bool
	cleanupFunc      func(in Inode)
	syncing          bool
	syncReceived     bool
	tempFileState    *gcsx.TempFileSate
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
	o *gcs.Object,
	attrs fuseops.InodeAttributes,
	bucket gcs.Bucket,
	syncer gcsx.Syncer,
	tempDir string,
	mtimeClock timeutil.Clock, cleanupFunc func(Inode), p *gcsx.TempFileSate, cacheRemovalDelay time.Duration,) (f *FileInode) {
	// Set up the basic struct.
	f = &FileInode{
		bucket:        bucket,
		syncer:        syncer,
		mtimeClock:    mtimeClock,
		id:            id,
		name:          o.Name,
		attrs:         attrs,
		tempDir:       tempDir,
		cacheRemovalDelay: cacheRemovalDelay,
		src:           *o,
		cleanupFunc:   cleanupFunc,
		tempFileState: p,
	}
	f.sc = util.NewSchedule(f.cacheRemovalDelay, 0, nil, func(i interface{}) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.Cleanup()
	})

	f.lc.Init(id)

	// Set up invariant checking.
	f.mu = syncutil.NewInvariantMutex(f.checkInvariants)
	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Cleanup() {
	log.Println("fuse: removing cache for inode", f.id, f.name)
	if f.content != nil {
		name := f.GetTmpFileName()
		if er := f.tempFileState.DeleteFileStatus(name); er != nil {
			log.Println("fuse: failed to delete cache status", name, er)
		}
		f.content.Destroy()
		f.content = nil
	}
	f.cleanupScheduled = false

	if f.lc.count == 0 && f.syncRequired == false {
		log.Println("fuse: cleanup inode", f.id, f.name)
		f.Destroy()
		f.cleanupFunc(f)
	}
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) checkInvariants() {
	if f.destroyed {
		return
	}

	// Make sure the name is legal.
	name := f.Name()
	if len(name) == 0 || name[len(name)-1] == '/' {
		panic("Illegal file name: " + name)
	}

	// INVARIANT: src.Name == name
	if f.src.Name != name {
		panic(fmt.Sprintf("Name mismatch: %q vs. %q", f.src.Name, name))
	}

	// INVARIANT: content.CheckInvariants() does not panic
	if f.content != nil {
		f.content.CheckInvariants()
	}
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) clobbered(ctx context.Context) (b bool, err error) {
	// Stat the object in GCS.
	req := &gcs.StatObjectRequest{Name: f.name}
	o, err := f.bucket.StatObject(ctx, req)

	// Special case: "not found" means we have been clobbered.
	if _, ok := err.(*gcs.NotFoundError); ok {
		err = nil
		b = true
		return
	}

	// Propagate other errors.
	if err != nil {
		err = fmt.Errorf("StatObject: %v", err)
		return
	}

	// We are clobbered iff the generation doesn't match our source generation.
	oGen := Generation{o.Generation, o.MetaGeneration}
	b = f.SourceGeneration().Compare(oGen) != 0

	return
}

// Ensure that f.content != nil
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) ensureContent(ctx context.Context, readMode bool) (err error) {
	f.cancelCleanupSchedule()
	// Is there anything to do?
	if f.content != nil || readMode && f.sourceReader != nil {
		return
	}
	log.Println("fuse: ensureContent", f.name)
	defer log.Println("fuse: ensureContent done", f.name)

	// Open a reader for the generation we care about.
	rc, err := f.bucket.NewReader(
		context.Background(),
		&gcs.ReadObjectRequest{
			Name:       f.src.Name,
			Generation: f.src.Generation,
		})

	if err != nil {
		err = fmt.Errorf("NewReader: %v", err)
		return
	}

	//defer rc.Close()
	if readMode {
		f.sourceReader = rc
		return
	}

	// Create a temporary file with its contents.
	tf, err := gcsx.NewTempFile(rc, f.tempDir, f.mtimeClock, readMode, rc.Close)
	if err != nil {
		err = fmt.Errorf("NewTempFile: %v", err)
		return
	}

	// Update state.
	f.content = tf

	return
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) CanDestroy() bool {
	return !f.syncRequired && !f.cleanupScheduled && !f.syncing
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) scheduleCleanUp() {
	f.sc.Schedule(f.name)
	f.cleanupScheduled = true
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) cancelCleanupSchedule() {
	f.sc.Cancel(f.name)
	f.cleanupScheduled = false
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) SyncLocal() error {
	if f.content == nil {
		return nil
	}
	return f.content.SyncLocal()
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) IsSyncRequired() bool {
	return f.syncRequired
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) SetSyncRequired(s bool) {
	f.syncRequired = s
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) IsSyncReceived() bool {
	return f.syncReceived
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) SyncReceived() {
	f.syncReceived = true
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) GetTmpFileName() string {
	return f.content.GetFileRO().Name()
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) HasContent() bool {
	return f.content != nil
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

func (f *FileInode) RLock() {
	f.mu.RLock()
}

func (f *FileInode) RUnlock() {
	f.mu.RUnlock()
}

func (f *FileInode) ID() fuseops.InodeID {
	return f.id
}

func (f *FileInode) Name() string {
	return f.name
}

// Return a record for the GCS object from which this inode is branched. The
// record is guaranteed not to be modified, and users must not modify it.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Source() *gcs.Object {
	// Make a copy, since we modify f.src.
	o := f.src
	return &o
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) UpdateName(name string) {
	f.src.Name = name
	f.name = name
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
	log.Println("fuse: destroying file inode", f.name, f.id)
	f.destroyed = true

	if f.content != nil {
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
	if formatted, ok := f.src.Metadata["gcsfuse_mtime"]; ok {
		attrs.Mtime, err = time.Parse(time.RFC3339Nano, formatted)
		if err != nil {
			err = fmt.Errorf("time.Parse(%q): %v", formatted, err)
			return
		}
	}

	// If we've got local content, its size and (maybe) mtime take precedence.
	if f.content != nil {
		var sr gcsx.StatResult
		sr, err = f.content.Stat()
		if err != nil {
			err = fmt.Errorf("Stat: %v", err)
			return
		}

		attrs.Size = uint64(sr.Size)
		if sr.Mtime != nil {
			attrs.Mtime = *sr.Mtime
		}
	}

	// If the object has been clobbered, we reflect that as the inode being
	// unlinked.
	clobbered, err := f.clobbered(ctx)
	if err != nil {
		err = fmt.Errorf("clobbered: %v", err)
		return
	}

	if !clobbered {
		attrs.Nlink = 1
	}

	return
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
	defer func() {
		if !f.syncing {
			f.scheduleCleanUp()
		}
	}()
	// Make sure f.content != nil.
	err = f.ensureContent(ctx, true)
	if err != nil {
		err = fmt.Errorf("ensureContent: %v", err)
		return
	}

	// Read from the local content, propagating io.EOF.
	if f.content != nil {
		n, err = f.content.ReadAt(dst, offset)
	} else {
		_, err = f.sourceReader.Seek(offset, io.SeekStart)
		if err != nil {
			return
		}
		f.rmu.Lock()
		n, err = f.sourceReader.Read(dst)
		f.rmu.Unlock()
	}

	switch {
	case err == io.EOF:
		return

	case err != nil:
		err = fmt.Errorf("content.ReadAt: %v", err)
		f.cancelCleanupSchedule()
		f.Cleanup()
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
	f.syncRequired = true
	f.syncReceived = false
	// Make sure f.content != nil.
	err = f.ensureContent(ctx, false)
	if err != nil {
		err = fmt.Errorf("ensureContent: %v", err)
		f.cancelCleanupSchedule()
		f.Cleanup()
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
			err = fmt.Errorf("Stat: %v", err)
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
	switch err.(type) {
	case nil:
		f.src = *o
		return

	case *gcs.NotFoundError:
		// Special case: silently ignore not found errors, which mean the file has
		// been unlinked.
		err = nil
		return

	case *gcs.PreconditionError:
		// Special case: silently ignore precondition errors, which we also take to
		// mean the file has been unlinked.
		err = nil
		return

	default:
		err = fmt.Errorf("UpdateObject: %v", err)
		return
	}
}

// Write out contents to GCS. If this fails due to the generation having been
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
		log.Println("fuse: sync canceled. nil content", f.name)
		return
	}

	if !f.syncRequired {
		log.Println("fuse: sync canceled. sync is not required", f.name)
		return
	}
	f.syncReceived = true
	f.syncing = true
	f.cancelCleanupSchedule()
	defer func() {
		f.syncing = false
		f.scheduleCleanUp()
	}()

	// Write out the contents if they are dirty.
	newObj, err := f.syncer.SyncObject(ctx, &f.src, f.content)

	// Special case: a precondition error means we were clobbered, which we treat
	// as being unlinked. There's no reason to return an error in that case.
	if _, ok := err.(*gcs.PreconditionError); ok {
		err = nil
	}

	// Propagate other errors.
	if err != nil {
		err = fmt.Errorf("SyncObject: %v", err)
		f.syncRequired = true
		return
	}

	// If we wrote out a new object, we need to update our state.
	if newObj != nil {
		f.src = *newObj
		//f.content = nil
	}
	f.syncRequired = false

	return
}

// Truncate the file to the specified size.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Truncate(
	ctx context.Context,
	size int64) (err error) {
	// Make sure f.content != nil.
	err = f.ensureContent(ctx, false)
	if err != nil {
		err = fmt.Errorf("ensureContent: %v", err)
		f.cancelCleanupSchedule()
		f.Cleanup()
		return
	}

	// Call through.
	err = f.content.Truncate(size)

	return
}
