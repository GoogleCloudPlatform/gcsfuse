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

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

type FileInode struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	bucket gcs.Bucket
	syncer gcsx.Syncer
	clock  timeutil.Clock

	/////////////////////////
	// Constant data
	/////////////////////////

	id    fuseops.InodeID
	name  string
	attrs fuseops.InodeAttributes

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
// REQUIRES: len(o.Name) > 0
// REQUIRES: o.Name[len(o.Name)-1] != '/'
func NewFileInode(
	id fuseops.InodeID,
	o *gcs.Object,
	attrs fuseops.InodeAttributes,
	bucket gcs.Bucket,
	syncer gcsx.Syncer,
	clock timeutil.Clock) (f *FileInode) {
	// Set up the basic struct.
	f = &FileInode{
		bucket: bucket,
		syncer: syncer,
		clock:  clock,
		id:     id,
		name:   o.Name,
		attrs:  attrs,
		src:    *o,
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
	b = (o.Generation != f.src.Generation)

	return
}

// Ensure that f.content != nil
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) ensureContent(ctx context.Context) (err error) {
	// Is there anything to do?
	if f.content != nil {
		return
	}

	// Open a reader for the generation we care about.
	rc, err := f.bucket.NewReader(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       f.src.Name,
			Generation: f.src.Generation,
		})

	if err != nil {
		err = fmt.Errorf("NewReader: %v", err)
		return
	}

	defer rc.Close()

	// Create a temporary file with its contents.
	tf, err := gcsx.NewTempFile(rc, f.clock)
	if err != nil {
		err = fmt.Errorf("NewTempFile: %v", err)
		return
	}

	// Update state.
	f.content = tf

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

func (f *FileInode) Name() string {
	return f.name
}

// Return a record for the GCS object generation from which this inode is
// branched. The record is guaranteed not to be modified, and users must not
// modify it.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Source() *gcs.Object {
	// Make a copy, since we modify f.src.
	o := f.src
	return &o
}

// If true, it is safe to serve reads directly from the object generation given
// by f.Source(), rather than calling f.ReadAt. Doing so may be more efficient,
// because f.ReadAt may cause the entire object to be faulted in and requires
// the inode to be locked during the read.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) SourceGenerationIsAuthoritative() bool {
	return f.content == nil
}

// Equivalent to f.Source().Generation.
//
// LOCKS_REQUIRED(f)
func (f *FileInode) SourceGeneration() int64 {
	return f.src.Generation
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

	// If GCS is no longer authoritative, stat our local content to obtain size
	// and mtime.
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
	// Make sure f.content != nil.
	err = f.ensureContent(ctx)
	if err != nil {
		err = fmt.Errorf("ensureContent: %v", err)
		return
	}

	// Read from the local content, propagating io.EOF.
	n, err = f.content.ReadAt(dst, offset)
	switch {
	case err == io.EOF:
		return

	case err != nil:
		err = fmt.Errorf("content.ReadAt: %v", err)
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
		err = fmt.Errorf("ensureContent: %v", err)
		return
	}

	// Write to the mutable content. Note that io.WriterAt guarantees it returns
	// an error for short writes.
	_, err = f.content.WriteAt(data, offset)

	return
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
		return
	}

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
		return
	}

	// If we wrote out a new object, we need to update our state.
	if newObj != nil {
		f.src = *newObj
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
		err = fmt.Errorf("ensureContent: %v", err)
		return
	}

	// Call through.
	err = f.content.Truncate(size)

	return
}
