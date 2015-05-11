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

package gcsproxy

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

var fTempDir = flag.String(
	"gcsproxy.temp_dir", "",
	"The temporary directory in which to store local copies of GCS objects. "+
		"If empty, the system default (probably /tmp) will be used.")

// A view on a particular generation of an object in GCS that allows random
// access reads and writes.
//
// Reads may involve reading from a local cache. Writes are buffered locally
// until the Sync method is called, at which time a new generation of the
// object is created.
//
// This type is not safe for concurrent access. The user must provide external
// synchronization around the methods where it is not otherwise noted.
type MutableObject struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	bucket gcs.Bucket
	clock  timeutil.Clock

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A record for the specific generation of the object from which our local
	// state is branched.
	src gcs.Object

	// The current generation number. Must be accessed using sync/atomic.
	//
	// INVARIANT: atomic.LoadInt64(&sourceGeneration) == src.Generation
	sourceGeneration int64

	// When clean, a read proxy around src. When dirty, nil.
	readProxy *ReadProxy

	// When dirty, a local temporary file containing our current contents. When
	// clean, nil.
	//
	// TODO(jacobsa): Use a read/write lease here, and make it possible to
	// "downgrade" directly to a read proxy by adding a variant of
	// gcsproxy.NewReadProxy that takes an existing read/write lease, then ditto
	// with lease.NewReadProxy  in addition to the refresh function.
	//
	// INVARIANT: (readProxy == nil) != (localFile == nil)
	localFile *os.File

	// The time at which a method that modifies our contents was last called, or
	// nil if never.
	mtime *time.Time
}

type StatResult struct {
	// The current size in bytes of the content, including any local
	// modifications that have not been Sync'd.
	Size int64

	// The time at which the contents were last updated, or the creation time of
	// the source object if they never have been.
	Mtime time.Time

	// Has the object changed out from under us in GCS? If so, Sync will fail.
	Clobbered bool
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// Create a view on the given GCS object generation.
//
// REQUIRES: o != nil
func NewMutableObject(
	clock timeutil.Clock,
	bucket gcs.Bucket,
	o *gcs.Object) (mo *MutableObject) {
	// Set up the basic struct.
	mo = &MutableObject{
		clock:            clock,
		bucket:           bucket,
		src:              *o,
		sourceGeneration: o.Generation,
	}

	return
}

// Return the name of the proxied object. This may or may not be an object that
// currently exists in the bucket, depending on whether the backing object has
// been deleted.
//
// May be called concurrently with any method.
//
// TODO(jacobsa): I think there is a race on reading src.Name with the write in
// Sync. Do we actually need this guarantee?
func (mo *MutableObject) Name() string {
	return mo.src.Name
}

// Return the generation of the object from which the current contents of this
// proxy were branched. If Sync has been successfully called, this is the
// generation most recently returned by Sync. Otherwise it is the generation
// from which the proxy was created.
//
// May be called concurrently with any method, but note that without excluding
// concurrent calls to Sync this may change spontaneously.
func (mo *MutableObject) SourceGeneration() int64 {
	return atomic.LoadInt64(&mo.sourceGeneration)
}

// Panic if any internal invariants are violated. Careful users can call this
// at appropriate times to help debug weirdness. Consider using
// syncutil.InvariantMutex to automate the process.
func (mo *MutableObject) CheckInvariants() {
	// INVARIANT: atomic.LoadInt64(&sourceGeneration) == src.Generation
	{
		g := atomic.LoadInt64(&mo.sourceGeneration)
		if g != mo.src.Generation {
			panic(fmt.Sprintf("Generation mismatch: %v vs. %v", g, mo.src.Generation))
		}
	}

	// INVARIANT: If dirty, then localFile != nil
	if mo.dirty && mo.localFile == nil {
		panic("Expected non-nil localFile.")
	}

	// INVARIANT: If dirty, then mtime != nil
	if mo.dirty && mo.mtime == nil {
		panic("Expected non-nil mtime.")
	}
}

// Destroy any local file caches, putting the proxy into an indeterminate
// state. Should be used before dropping the final reference to the proxy.
func (mo *MutableObject) Destroy() (err error) {
	// Make sure that when we exit no invariants are violated.
	defer func() {
		mo.localFile = nil
		mo.dirty = false
	}()

	// If we have no local file, there's nothing to do.
	if mo.localFile == nil {
		return
	}

	// Close the local file.
	if err = mo.localFile.Close(); err != nil {
		err = fmt.Errorf("Close: %v", err)
		return
	}

	return
}

// Return the current size in bytes of the content and an indication of whether
// the proxied object has changed out from under us (in which case Sync will
// fail).
//
// sr.Clobbered will be set only if needClobbered is true. Otherwise a round
// trip to GCS can be saved.
func (mo *MutableObject) Stat(
	ctx context.Context,
	needClobbered bool) (sr StatResult, err error) {
	// If we have ever been modified, our mtime field is authoritative (even if
	// we've been Sync'd, because Sync is not supposed to affect the mtime).
	// Otherwise our source object's creation time is our mtime.
	if mo.mtime != nil {
		sr.Mtime = *mo.mtime
	} else {
		sr.Mtime = mo.src.Updated
	}

	// If we have a file, it is authoritative for our size. Otherwise our source
	// size is authoritative.
	if mo.localFile != nil {
		var fi os.FileInfo
		if fi, err = mo.localFile.Stat(); err != nil {
			err = fmt.Errorf("Stat: %v", err)
			return
		}

		sr.Size = fi.Size()
	} else {
		sr.Size = int64(mo.src.Size)
	}

	// Figure out whether we were clobbered iff the user asked us to.
	if needClobbered {
		sr.Clobbered, err = mo.clobbered(ctx)
		if err != nil {
			err = fmt.Errorf("clobbered: %v", err)
			return
		}
	}

	return
}

// Make a random access read into our view of the content. May block for
// network access.
//
// Guarantees that err != nil if n < len(buf)
func (mo *MutableObject) ReadAt(
	ctx context.Context,
	buf []byte,
	offset int64) (n int, err error) {
	// Make sure we have a local file.
	if err = mo.ensureLocalFile(ctx); err != nil {
		err = fmt.Errorf("ensureLocalFile: %v", err)
		return
	}

	// Serve the read from the file.
	n, err = mo.localFile.ReadAt(buf, offset)

	return
}

// Make a random access write into our view of the content. May block for
// network access. Not guaranteed to be reflected remotely until after Sync is
// called successfully.
//
// Guarantees that err != nil if n < len(buf)
func (mo *MutableObject) WriteAt(
	ctx context.Context,
	buf []byte,
	offset int64) (n int, err error) {
	// Make sure we have a local file.
	if err = mo.ensureLocalFile(ctx); err != nil {
		err = fmt.Errorf("ensureLocalFile: %v", err)
		return
	}

	newMtime := mo.clock.Now()

	mo.dirty = true
	mo.mtime = &newMtime
	n, err = mo.localFile.WriteAt(buf, offset)

	return
}

// Truncate our view of the content to the given number of bytes, extending if
// n is greater than the current size. May block for network access. Not
// guaranteed to be reflected remotely until after Sync is called successfully.
func (mo *MutableObject) Truncate(ctx context.Context, n int64) (err error) {
	// Make sure we have a local file.
	if err = mo.ensureLocalFile(ctx); err != nil {
		err = fmt.Errorf("ensureLocalFile: %v", err)
		return
	}

	// Convert to signed, which is what os.File wants.
	if n > math.MaxInt64 {
		err = fmt.Errorf("Illegal offset: %v", n)
		return
	}

	newMtime := mo.clock.Now()

	mo.dirty = true
	mo.mtime = &newMtime
	err = mo.localFile.Truncate(int64(n))

	return
}

// If the proxy is dirty due to having been modified, save its current contents
// to GCS, creating a generation with exactly those contents. Do so with a
// precondition such that the creation will fail if the source generation is
// not current. In that case, return an error of type *gcs.PreconditionError.
// If the proxy is not dirty, simply return nil.
//
// After this method successfully returns, SourceGeneration returns the
// generation at which the contents are current.
func (mo *MutableObject) Sync(ctx context.Context) (err error) {
	// Do we need to do anything?
	if !mo.dirty {
		return
	}

	// Seek the file to the start so that it can be used as a reader for its full
	// contents below.
	_, err = mo.localFile.Seek(0, 0)
	if err != nil {
		err = fmt.Errorf("Seek: %v", err)
		return
	}

	// Write a new generation of the object with the appropriate contents, using
	// an appropriate precondition.
	req := &gcs.CreateObjectRequest{
		Name:                   mo.src.Name,
		Contents:               mo.localFile,
		GenerationPrecondition: &mo.src.Generation,
	}

	o, err := mo.bucket.CreateObject(ctx, req)

	// Special case: handle precondition errors.
	if _, ok := err.(*gcs.PreconditionError); ok {
		err = &gcs.PreconditionError{
			Err: fmt.Errorf("CreateObject: %v", err),
		}

		return
	}

	// Propagate other errors more directly.
	if err != nil {
		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	// Update our state.
	mo.src = *o
	mo.dirty = false
	atomic.StoreInt64(&mo.sourceGeneration, mo.src.Generation)

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Set up an unlinked local temporary file for the given generation of the
// given object.
func makeLocalFile(
	ctx context.Context,
	bucket gcs.Bucket,
	name string,
	generation int64) (f *os.File, err error) {
	// Create the file.
	f, err = fsutil.AnonymousFile(*fTempDir)
	if err != nil {
		err = fmt.Errorf("AnonymousFile: %v", err)
		return
	}

	// Ensure that we clean up the file if we return in error from this method.
	defer func() {
		if err != nil {
			f.Close()
			f = nil
		}
	}()

	// Open the object for reading.
	req := &gcs.ReadObjectRequest{
		Name:       name,
		Generation: generation,
	}

	var rc io.ReadCloser
	if rc, err = bucket.NewReader(ctx, req); err != nil {
		err = fmt.Errorf("NewReader: %v", err)
		return
	}

	// Copy to the file.
	if _, err = io.Copy(f, rc); err != nil {
		err = fmt.Errorf("Copy: %v", err)
		return
	}

	// Close.
	if err = rc.Close(); err != nil {
		err = fmt.Errorf("Close: %v", err)
		return
	}

	return
}

// Ensure that mo.localFile is non-nil with an authoritative view of mo's
// contents.
func (mo *MutableObject) ensureLocalFile(ctx context.Context) (err error) {
	// Is there anything to do?
	if mo.localFile != nil {
		return
	}

	// Set up the file.
	f, err := makeLocalFile(ctx, mo.bucket, mo.Name(), mo.src.Generation)
	if err != nil {
		err = fmt.Errorf("makeLocalFile: %v", err)
		return
	}

	mo.localFile = f
	return
}

func (mo *MutableObject) clobbered(
	ctx context.Context) (clobbered bool, err error) {
	// Stat the object in GCS.
	req := &gcs.StatObjectRequest{Name: mo.Name()}
	o, err := mo.bucket.StatObject(ctx, req)

	// Special case: "not found" means we have been clobbered.
	if _, ok := err.(*gcs.NotFoundError); ok {
		err = nil
		clobbered = true
		return
	}

	// Propagate other errors.
	if err != nil {
		err = fmt.Errorf("StatObject: %v", err)
		return
	}

	// We are clobbered iff the generation doesn't match our source generation.
	clobbered = (o.Generation != mo.src.Generation)

	return
}
