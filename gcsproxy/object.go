// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy

import (
	"io"
	"log"
	"os"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"google.golang.org/cloud/storage"
)

// A view on an object in GCS that allows random access reads and writes.
//
// Reads may involve reading from a local cache. Writes are buffered locally
// until the Sync method is called, at which time a new generation of the
// object is created.
//
// All methods are safe for concurrent access. Concurrent readers and writers
// within process receive the same guarantees as with POSIX files.
type ProxyObject struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	logger *log.Logger
	bucket gcs.Bucket

	/////////////////////////
	// Constant data
	/////////////////////////

	// The name of the GCS object for which we are a proxy. Might not exist in
	// the bucket.
	name string

	/////////////////////////
	// Mutable state
	/////////////////////////

	mu syncutil.InvariantMutex

	// The specific generation of the object from which our local state is
	// branched. If we have no local state, the contents of this object are
	// exactly our contents. May be nil if NoteLatest was never called.
	source *storage.Object // GUARDED_BY(mu)

	// A local temporary file containing the contents of our source (or the empty
	// string if no source) along with any local modifications. When nil, to be
	// regarded as the empty file.
	localFile *os.File // GUARDED_BY(mu)

	// false iff source is non-nil and is authoritative for our view of the
	// contents. Sync needs to do work iff this is true.
	dirty bool // GUARDED_BY(mu)
}

var _ io.ReaderAt = &ProxyObject{}
var _ io.WriterAt = &ProxyObject{}

// Create a new view on the GCS object with the given name. The remote object
// is assumed to be non-existent, so that the local contents are empty. Use
// NoteLatest to change that if necessary.
func NewProxyObject(
	bucket gcs.Bucket,
	name string) (po *ProxyObject, err error) {
	po = &ProxyObject{
		logger: getLogger(),
		bucket: bucket,
		name:   name,

		// Initial state: empty contents, dirty. (The remote object needs to be
		// truncated.)
		source:    nil,
		localFile: nil,
		dirty:     true,
	}

	po.mu = syncutil.NewInvariantMutex(po.checkInvariants)
	return
}

// SHARED_LOCKS_REQUIRED(po.mu)
func (po *ProxyObject) checkInvariants()

// Inform the proxy object of the most recently observed generation of the
// object of interest in GCS.
//
// If this is no newer than the newest generation that has previously been
// observed, it is ignored. Otherwise, it becomes the definitive source of data
// for the object. Any local-only state is clobbered, including local
// modifications.
func (po *ProxyObject) NoteLatest(o storage.Object) error

// Return the current size in bytes of our view of the content.
func (po *ProxyObject) Size() uint64

// Make a random access read into our view of the content. May block for
// network access.
func (po *ProxyObject) ReadAt(buf []byte, offset int64) (int, error)

// Make a random access write into our view of the content. May block for
// network access. Not guaranteed to be reflected remotely until after Sync is
// called successfully.
func (po *ProxyObject) WriteAt(buf []byte, offset int64) (int, error)

// Truncate our view of the content to the given number of bytes, extending if
// n is greater than Size(). May block for network access. Not guaranteed to be
// reflected remotely until after Sync is called successfully.
func (po *ProxyObject) Truncate(n uint64) error

// Ensure that the remote object reflects the local state, returning a record
// for a generation that does. Clobbers the remote version. Does no work if the
// remote version is already up to date.
func (po *ProxyObject) Sync() (storage.Object, error)
