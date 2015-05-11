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
	"fmt"
	"io"
	"math"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

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
	leaser lease.FileLeaser
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

	// When dirty, a read/write lease containing our current contents. When
	// clean, nil.
	//
	// INVARIANT: (readProxy == nil) != (readWriteLease == nil)
	readWriteLease lease.ReadWriteLease

	// The time at which a method that modifies our contents was last called, or
	// nil if never.
	//
	// INVARIANT: If dirty(), then mtime != nil
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

// Create a view on the given GCS object generation, using the supplied leaser
// to mediate temporary space usage.
//
// REQUIRES: o != nil
func NewMutableObject(
	o *gcs.Object,
	bucket gcs.Bucket,
	leaser lease.FileLeaser,
	clock timeutil.Clock) (mo *MutableObject) {
	// Set up the basic struct.
	mo = &MutableObject{
		bucket:           bucket,
		leaser:           leaser,
		clock:            clock,
		src:              *o,
		sourceGeneration: o.Generation,
		readProxy:        NewReadProxy(leaser, bucket, o, nil),
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

	// INVARIANT: (readProxy == nil) != (readWriteLease == nil)
	if mo.readProxy == nil && mo.readWriteLease == nil {
		panic("Both readProxy and readWriteLease are nil")
	}

	if mo.readProxy != nil && mo.readWriteLease != nil {
		panic("Both readProxy and readWriteLease are non-nil")
	}

	// INVARIANT: If dirty(), then mtime != nil
	if mo.dirty() && mo.mtime == nil {
		panic("Expected non-nil mtime.")
	}
}

// Destroy any local file caches, putting the proxy into an indeterminate
// state. The MutableObject must not be used after calling this method,
// regardless of outcome.
func (mo *MutableObject) Destroy() (err error) {
	// If we have no read/write lease, there's nothing to do.
	if mo.readWriteLease == nil {
		return
	}

	// Downgrade to a read lease.
	rl, err := mo.readWriteLease.Downgrade()
	if err != nil {
		err = fmt.Errorf("Downgrade: %v", err)
		return
	}

	mo.readWriteLease = nil

	// Revoke the read lease.
	rl.Revoke()

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

	// If we have a read/write lease, it is authoritative for our size. Otherwise
	// the read proxy is authoritative.
	if mo.readWriteLease != nil {
		sr.Size, err = mo.readWriteLease.Size()
		if err != nil {
			err = fmt.Errorf("Size: %v", err)
			return
		}
	} else {
		sr.Size = mo.readProxy.Size()
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
	// Serve from the read proxy or the read/write lease.
	if mo.dirty() {
		n, err = mo.readWriteLease.ReadAt(buf, offset)
	} else {
		n, err = mo.readProxy.ReadAt(ctx, buf, offset)
	}

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
	// Make sure we have a read/write lease.
	if err = mo.ensureReadWriteLease(ctx); err != nil {
		err = fmt.Errorf("ensureReadWriteLease: %v", err)
		return
	}

	newMtime := mo.clock.Now()
	mo.mtime = &newMtime
	n, err = mo.readWriteLease.WriteAt(buf, offset)

	return
}

// Truncate our view of the content to the given number of bytes, extending if
// n is greater than the current size. May block for network access. Not
// guaranteed to be reflected remotely until after Sync is called successfully.
func (mo *MutableObject) Truncate(ctx context.Context, n int64) (err error) {
	// Make sure we have a read/write lease.
	if err = mo.ensureReadWriteLease(ctx); err != nil {
		err = fmt.Errorf("ensureReadWriteLease: %v", err)
		return
	}

	// Convert to signed, which is what lease.ReadWriteLease wants.
	if n > math.MaxInt64 {
		err = fmt.Errorf("Illegal offset: %v", n)
		return
	}

	newMtime := mo.clock.Now()
	mo.mtime = &newMtime
	err = mo.readWriteLease.Truncate(int64(n))

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
	if !mo.dirty() {
		return
	}

	// Seek the read/write lease to the start so that it can be used as a reader
	// for its full contents below.
	_, err = mo.readWriteLease.Seek(0, 0)
	if err != nil {
		err = fmt.Errorf("Seek: %v", err)
		return
	}

	// Write a new generation of the object with the appropriate contents, using
	// an appropriate precondition.
	req := &gcs.CreateObjectRequest{
		Name:                   mo.src.Name,
		Contents:               mo.readWriteLease,
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

	mo.src = *o
	atomic.StoreInt64(&mo.sourceGeneration, mo.src.Generation)

	// Attempt to downgrade the read/write lease to a read lease, and use that to
	// prime the new read proxy. But whether or not that pans out, ensure that a
	// read proxy is set up.
	defer func() {
		if mo.readProxy == nil {
			mo.readProxy = NewReadProxy(mo.leaser, mo.bucket, o, nil)
		}
	}()

	rwl := mo.readWriteLease
	mo.readWriteLease = nil

	rl, err := rwl.Downgrade()
	if err != nil {
		err = fmt.Errorf("Downgrade: %v", err)
		return
	}

	mo.readProxy = NewReadProxy(mo.leaser, mo.bucket, o, rl)

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Set up a read/write lease containing the given generation of the given
// object.
func makeReadWriteLease(
	ctx context.Context,
	bucket gcs.Bucket,
	leaser lease.FileLeaser,
	name string,
	generation int64) (rwl lease.ReadWriteLease, err error) {
	// Create the read/write lease.
	rwl, err = leaser.NewFile()
	if err != nil {
		err = fmt.Errorf("NewFile: %v", err)
		return
	}

	// Ensure that we clean up the lease if we return in error from this method.
	defer func() {
		if err != nil {
			rwl.Downgrade()
			rwl = nil
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

	// Copy to the read/write lease.
	if _, err = io.Copy(rwl, rc); err != nil {
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

func (mo *MutableObject) dirty() bool {
	return mo.readWriteLease != nil
}

// Ensure that mo.readWriteLease is non-nil with an authoritative view of mo's
// contents.
func (mo *MutableObject) ensureReadWriteLease(ctx context.Context) (err error) {
	// Is there anything to do?
	if mo.readWriteLease != nil {
		return
	}

	// Set up the read/write lease.
	rwl, err := makeReadWriteLease(
		ctx,
		mo.bucket,
		mo.leaser,
		mo.Name(),
		mo.src.Generation)

	if err != nil {
		err = fmt.Errorf("makeReadWriteLease: %v", err)
		return
	}

	mo.readWriteLease = rwl

	// Throw away the read proxy.
	rp := mo.readProxy
	mo.readProxy = nil

	err = rp.Destroy()
	if err != nil {
		err = fmt.Errorf("Destroy: %v", err)
		return
	}

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
