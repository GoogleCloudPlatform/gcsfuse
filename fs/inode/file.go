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

	"github.com/googlecloudplatform/gcsfuse/gcsproxy"
	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
)

type FileInode struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	bucket gcs.Bucket

	/////////////////////////
	// Constant data
	/////////////////////////

	id           fuseops.InodeID
	supportNlink bool

	// Redundant with proxy.Name(), but stored separately so that it can be
	// accessed without a lock.
	name string

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A mutex that must be held when calling certain methods. See documentation
	// for each method.
	mu syncutil.InvariantMutex

	// GUARDED_BY(mu)
	lc lookupCount

	// A proxy for the backing object in GCS.
	//
	// INVARIANT: proxy.CheckInvariants() does not panic
	// INVARIANT: proxy.Name() == name
	//
	// GUARDED_BY(mu)
	proxy *gcsproxy.MutableObject

	// Has Destroy been called?
	//
	// GUARDED_BY(mu)
	destroyed bool
}

var _ Inode = &FileInode{}

// Create a file inode for the given object in GCS. The initial lookup count is
// zero.
//
// If supportNlink is set, Attributes will use bucket.StatObject to find out
// whether the backing objet has been clobbered. Otherwise, Attributes will
// always show Nlink == 1.
//
// REQUIRES: o != nil
// REQUIRES: o.Generation > 0
// REQUIRES: len(o.Name) > 0
// REQUIRES: o.Name[len(o.Name)-1] != '/'
func NewFileInode(
	id fuseops.InodeID,
	o *gcs.Object,
	supportNlink bool,
	bucket gcs.Bucket,
	leaser lease.FileLeaser,
	clock timeutil.Clock) (f *FileInode) {
	// Set up the basic struct.
	f = &FileInode{
		bucket:       bucket,
		id:           id,
		name:         o.Name,
		supportNlink: supportNlink,
		proxy:        gcsproxy.NewMutableObject(o, bucket, leaser, clock),
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

	// INVARIANT: proxy.CheckInvariants() does not panic
	f.proxy.CheckInvariants()

	// INVARIANT: proxy.Name() == name
	if f.proxy.Name() != f.name {
		panic(fmt.Sprintf("Name mismatch: %q %q", f.proxy.Name(), f.name))
	}
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

// Return the object generation number from which this inode was branched.
//
// Does not require the lock to be held, but may spontaneously increase if the
// lock is not held.
func (f *FileInode) SourceGeneration() int64 {
	return f.proxy.SourceGeneration()
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

	err = f.proxy.Destroy()
	return
}

// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Attributes(
	ctx context.Context) (attrs fuseops.InodeAttributes, err error) {
	// Stat the object.
	sr, err := f.proxy.Stat(ctx, f.supportNlink)
	if err != nil {
		err = fmt.Errorf("Stat: %v", err)
		return
	}

	// Fill out the struct.
	attrs = fuseops.InodeAttributes{
		Nlink: 1,
		Size:  uint64(sr.Size),
		Mode:  0700,
		Mtime: sr.Mtime,
	}

	// If the object has been clobbered, we reflect that as the inode being
	// unlinked.
	if sr.Clobbered {
		attrs.Nlink = 0
	}

	return
}

// Serve a read op for this file, without responding.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Read(
	op *fuseops.ReadFileOp) (err error) {
	// Read from the proxy.
	buf := make([]byte, op.Size)
	n, err := f.proxy.ReadAt(op.Context(), buf, op.Offset)

	// We don't return errors for EOF. Otherwise, propagate errors.
	if err == io.EOF {
		err = nil
	} else if err != nil {
		err = fmt.Errorf("ReadAt: %v", err)
		return
	}

	// Fill in the response.
	op.Data = buf[:n]

	return
}

// Serve a write op for this file, without responding.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Write(
	op *fuseops.WriteFileOp) (err error) {
	// Write to the proxy. Note that the proxy guarantees that it returns an
	// error for short writes.
	_, err = f.proxy.WriteAt(op.Context(), op.Data, op.Offset)

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
	// Write out the proxy's contents if it is dirty.
	err = f.proxy.Sync(ctx)

	// Special case: a precondition error means we were clobbered, which we treat
	// as being unlinked. There's no reason to return an error in that case.
	if _, ok := err.(*gcs.PreconditionError); ok {
		err = nil
	}

	// Propagate other errors.
	if err != nil {
		err = fmt.Errorf("ObjectProxy.Sync: %v", err)
		return
	}

	return
}

// Truncate the file to the specified size.
//
// LOCKS_REQUIRED(f.mu)
func (f *FileInode) Truncate(
	ctx context.Context,
	size int64) (err error) {
	err = f.proxy.Truncate(ctx, size)
	return
}
