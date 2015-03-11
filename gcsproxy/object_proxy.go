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
	"errors"
	"os"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// A sentinel error returned by ObjectProxy.Sync.
const ErrNotCurrent = errors.New("Source generation not current.")

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
	// Constant data
	/////////////////////////

	// The name of the GCS object for which we are a proxy. Might not currently
	// exist in the bucket.
	name string

	/////////////////////////
	// Mutable state
	/////////////////////////

	// The specific generation of the object from which our local state is
	// branched. If we have no local state, the contents of this object are
	// exactly our contents. May be zero if our source is a "doesn't exist"
	// generation.
	srcGeneration uint64

	// A local temporary file containing our current contents. When non-nil, this
	// is the authority on our contents. When nil, our contents are defined by
	// the generation identified by srcGeneration.
	localFile *os.File

	// false if localFile is present but its contents may be different from the
	// contents of our source generation. Sync needs to do work iff this is true.
	//
	// INVARIANT: If srcGeneration == 0, then dirty
	// INVARIANT: If dirty, then localFile != nil
	dirty bool
}

// Create a view on the given GCS object generation, or zero if branching from
// a non-existent object (in which case the initial contents are empty).
func NewObjectProxy(
	bucket gcs.Bucket,
	name string,
	srcGeneration uint64) (op *ObjectProxy, err error)

// Return the name of the proxied object. This may or may not be an object that
// currently exists in the bucket.
func (op *ObjectProxy) Name() string

// Panic if any internal invariants are violated. Careful users can call this
// at appropriate times to help debug weirdness. Consider using
// syncutil.InvariantMutex to automate the process.
func (op *ObjectProxy) CheckInvariants()

// Destroy any local file caches, putting the proxy into an indeterminate
// state. Should be used before dropping the final reference to the proxy.
func (op *ObjectProxy) Destroy()

// Return the current size in bytes of the content and an indication of whether
// the proxied object has changed out from under us (in which case Sync will
// fail).
func (op *ObjectProxy) Stat(
	ctx context.Context) (size uint64, clobbered bool, err error)

// Make a random access read into our view of the content. May block for
// network access.
//
// Guarantees that err != nil if n < len(buf)
func (op *ObjectProxy) ReadAt(
	ctx context.Context,
	buf []byte,
	offset int64) (n int, err error)

// Make a random access write into our view of the content. May block for
// network access. Not guaranteed to be reflected remotely until after Sync is
// called successfully.
//
// Guarantees that err != nil if n < len(buf)
func (op *ObjectProxy) WriteAt(
	ctx context.Context,
	buf []byte,
	offset int64) (n int, err error)

// Truncate our view of the content to the given number of bytes, extending if
// n is greater than the current size. May block for network access. Not
// guaranteed to be reflected remotely until after Sync is called successfully.
func (op *ObjectProxy) Truncate(ctx context.Context, n uint64) (err error)

// If the proxy is dirty due to having been written to or due to having a nil
// source, save its current contents to GCS and return a generation number for
// a generation with exactly those contents. Do so with a precondition such
// that the creation will fail if the source generation is not current. In that
// case, return ErrNotCurrent.
func (op *ObjectProxy) Sync(ctx context.Context) (gen uint64, err error)
