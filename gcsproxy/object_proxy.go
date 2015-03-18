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
	"io/ioutil"
	"math"
	"os"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

// A view on a particular generation of an object in GCS that allows random
// access reads and writes.
//
// Reads may involve reading from a local cache. Writes are buffered locally
// until the Sync method is called, at which time a new generation of the
// object is created.
//
// This type is not safe for concurrent access. The user must provide external
// synchronization.
type ObjectProxy struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	bucket gcs.Bucket

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A record for the specific generation of the object from which our local
	// state is branched. If we have no local state, the contents of this
	// generation are exactly our contents.
	src storage.Object

	// A local temporary file containing our current contents. When non-nil, this
	// is the authority on our contents. When nil, our contents are defined by
	// 'src' above.
	localFile *os.File

	// true if localFile is present but its contents may be different from the
	// contents of our source generation. Sync needs to do work iff this is true.
	//
	// INVARIANT: If dirty, then localFile != nil
	dirty bool
}

type StatResult struct {
	// The current size in bytes of the content, including any local
	// modifications that have not been Sync'd.
	Size int64

	// Has the object changed out from under us in GCS? If so, Sync will fail.
	Clobbered bool
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// Create a view on the given GCS object generation.
//
// REQUIRES: o != nil
func NewObjectProxy(
	ctx context.Context,
	bucket gcs.Bucket,
	o *storage.Object) (op *ObjectProxy, err error) {
	// Set up the basic struct.
	op = &ObjectProxy{
		bucket: bucket,
		src:    *o,
	}

	return
}

// Return the name of the proxied object. This may or may not be an object that
// currently exists in the bucket, depending on whether the backing object has
// been deleted.
func (op *ObjectProxy) Name() string {
	return op.src.Name
}

// Return the generation of the object from which the current contents of this
// proxy were branched. If Sync has been successfully called, this is the
// generation most recently returned by Sync. Otherwise it is the generation
// from which the proxy was created.
func (op *ObjectProxy) SourceGeneration() int64 {
	return op.src.Generation
}

// Panic if any internal invariants are violated. Careful users can call this
// at appropriate times to help debug weirdness. Consider using
// syncutil.InvariantMutex to automate the process.
func (op *ObjectProxy) CheckInvariants() {
	// INVARIANT: If dirty, then localFile != nil
	if op.dirty && op.localFile == nil {
		panic("Expected non-nil localFile.")
	}
}

// Destroy any local file caches, putting the proxy into an indeterminate
// state. Should be used before dropping the final reference to the proxy.
func (op *ObjectProxy) Destroy() (err error) {
	// Make sure that when we exit no invariants are violated.
	defer func() {
		op.localFile = nil
		op.dirty = false
	}()

	// If we have no local file, there's nothing to do.
	if op.localFile == nil {
		return
	}

	// Close the local file.
	if err = op.localFile.Close(); err != nil {
		err = fmt.Errorf("Close: %v", err)
		return
	}

	return
}

// Return the current size in bytes of the content and an indication of whether
// the proxied object has changed out from under us (in which case Sync will
// fail).
func (op *ObjectProxy) Stat(
	ctx context.Context) (size int64, clobbered bool, err error) {
	// If we have a file, it is authoritative for our size. Otherwise our source
	// size is authoritative.
	if op.localFile != nil {
		var fi os.FileInfo
		if fi, err = op.localFile.Stat(); err != nil {
			err = fmt.Errorf("Stat: %v", err)
			return
		}

		size = fi.Size()
	} else {
		size = op.src.Size
	}

	// Stat the object in GCS.
	req := &gcs.StatObjectRequest{Name: op.Name()}
	o, err := op.bucket.StatObject(ctx, req)

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
	clobbered = (o.Generation != op.src.Generation)

	return
}

// Make a random access read into our view of the content. May block for
// network access.
//
// Guarantees that err != nil if n < len(buf)
func (op *ObjectProxy) ReadAt(
	ctx context.Context,
	buf []byte,
	offset int64) (n int, err error) {
	// Make sure we have a local file.
	if err = op.ensureLocalFile(ctx); err != nil {
		err = fmt.Errorf("ensureLocalFile: %v", err)
		return
	}

	// Serve the read from the file.
	n, err = op.localFile.ReadAt(buf, offset)

	return
}

// Make a random access write into our view of the content. May block for
// network access. Not guaranteed to be reflected remotely until after Sync is
// called successfully.
//
// Guarantees that err != nil if n < len(buf)
func (op *ObjectProxy) WriteAt(
	ctx context.Context,
	buf []byte,
	offset int64) (n int, err error) {
	// Make sure we have a local file.
	if err = op.ensureLocalFile(ctx); err != nil {
		err = fmt.Errorf("ensureLocalFile: %v", err)
		return
	}

	op.dirty = true
	n, err = op.localFile.WriteAt(buf, offset)

	return
}

// Truncate our view of the content to the given number of bytes, extending if
// n is greater than the current size. May block for network access. Not
// guaranteed to be reflected remotely until after Sync is called successfully.
func (op *ObjectProxy) Truncate(ctx context.Context, n int64) (err error) {
	// Make sure we have a local file.
	if err = op.ensureLocalFile(ctx); err != nil {
		err = fmt.Errorf("ensureLocalFile: %v", err)
		return
	}

	// Convert to signed, which is what os.File wants.
	if n > math.MaxInt64 {
		err = fmt.Errorf("Illegal offset: %v", n)
		return
	}

	op.dirty = true
	err = op.localFile.Truncate(int64(n))

	return
}

// If the proxy is dirty due to having been modified, save its current contents
// to GCS and return a generation number for a generation with exactly those
// contents. Do so with a precondition such that the creation will fail if the
// source generation is not current. In that case, return an error of type
// *gcs.PreconditionError.
func (op *ObjectProxy) Sync(ctx context.Context) (gen int64, err error) {
	// Do we need to do anything?
	if !op.dirty {
		gen = op.src.Generation
		return
	}

	// Seek the file to the start so that it can be used as a reader for its full
	// contents below.
	_, err = op.localFile.Seek(0, 0)
	if err != nil {
		err = fmt.Errorf("Seek: %v", err)
		return
	}

	// Write a new generation of the object with the appropriate contents, using
	// an appropriate precondition.
	req := &gcs.CreateObjectRequest{
		Attrs: storage.ObjectAttrs{
			Name: op.src.Name,
		},
		Contents:               op.localFile,
		GenerationPrecondition: &op.src.Generation,
	}

	o, err := op.bucket.CreateObject(ctx, req)

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
	op.src = *o
	op.dirty = false

	// Return the generation number.
	gen = op.src.Generation

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
	f, err = ioutil.TempFile("", "object_proxy")
	if err != nil {
		err = fmt.Errorf("TempFile: %v", err)
		return
	}

	// Ensure that we clean up the file if we return in error from this method.
	defer func() {
		if err != nil {
			f.Close()
			f = nil
		}
	}()

	// Unlink the file so that its inode will be garbage collected when the file
	// is closed.
	if err = os.Remove(f.Name()); err != nil {
		err = fmt.Errorf("Remove: %v", err)
		return
	}

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

// Ensure that op.localFile is non-nil with an authoritative view of op's
// contents.
func (op *ObjectProxy) ensureLocalFile(ctx context.Context) (err error) {
	// Is there anything to do?
	if op.localFile != nil {
		return
	}

	// Set up the file.
	f, err := makeLocalFile(ctx, op.bucket, op.Name(), op.src.Generation)
	if err != nil {
		err = fmt.Errorf("makeLocalFile: %v", err)
		return
	}

	op.localFile = f
	return
}
