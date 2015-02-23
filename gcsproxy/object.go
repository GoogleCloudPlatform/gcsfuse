// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy

import (
	"fmt"
	"io"
	"log"
	"math"
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
	//
	// INVARIANT: If source != nil, source.Size >= 0
	// INVARIANT: If source != nil, source.Name == name
	source *storage.Object // GUARDED_BY(mu)

	// A local temporary file containing the contents of our source (or the empty
	// string if no source) along with any local modifications. The authority on
	// our view of the object when non-nil.
	//
	// A nil file is to be regarded as empty, but is not authoritative unless
	// source is also nil.
	localFile *os.File // GUARDED_BY(mu)

	// false if the contents of localFile may be different from the contents of
	// the object referred to by source. Sync needs to do work iff this is true.
	//
	// INVARIANT: If false, then source != nil.
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
func (po *ProxyObject) checkInvariants() {
	if po.source != nil && po.source.Size <= 0 {
		if po.source.Size <= 0 {
			panic(fmt.Sprintf("Non-sensical source size: %v", po.source.Size))
		}

		if po.source.Name != po.name {
			panic(fmt.Sprintf("Name mismatch: %s vs. %s", po.source.Name, po.name))
		}
	}

	if !po.dirty && po.source == nil {
		panic("A clean proxy must have a source set.")
	}
}

// Inform the proxy object of the most recently observed generation of the
// object of interest in GCS.
//
// If this is no newer than the newest generation that has previously been
// observed, it is ignored. Otherwise, it becomes the definitive source of data
// for the object. Any local-only state is clobbered, including local
// modifications.
func (po *ProxyObject) NoteLatest(o storage.Object) (err error) {
	// Sanity check the input.
	if o.Size < 0 {
		err = fmt.Errorf("Object contains negative size: %v", o.Size)
		return
	}

	if o.Name != po.name {
		err = fmt.Errorf("Object name mismatch: %s vs. %s", o.Name, po.name)
		return
	}

	// Do nothing if nothing has changed.
	if po.source != nil && po.source.Generation == o.Generation {
		return
	}

	// Throw out the local file, if any.
	if po.localFile != nil {
		path := po.localFile.Name()

		if err = po.localFile.Close(); err != nil {
			err = fmt.Errorf("Closing local file: %v", err)
			return
		}

		if err = os.Remove(path); err != nil {
			err = fmt.Errorf("Unlinking local file: %v", err)
			return
		}
	}

	// Reset state.
	po.source = &o
	po.localFile = nil
	po.dirty = false

	return
}

// Return the current size in bytes of our view of the content.
func (po *ProxyObject) Size() (n uint64, err error) {
	// If we have a local file, it is authoritative.
	if po.localFile != nil {
		var fi os.FileInfo
		if fi, err = po.localFile.Stat(); err != nil {
			err = fmt.Errorf("localFile.Stat: %v", err)
			return
		}

		nSigned := fi.Size()
		if nSigned < 0 {
			err = fmt.Errorf("Stat returned nonsense size: %v", nSigned)
			return
		}

		n = uint64(nSigned)
		return
	}

	// Otherwise, if we have a source then it is authoritative.
	if po.source != nil {
		n = uint64(po.source.Size)
		return
	}

	// Otherwise, we are empty.
	return
}

// Make a random access read into our view of the content. May block for
// network access.
func (po *ProxyObject) ReadAt(buf []byte, offset int64) (n int, err error) {
	if err = po.ensureLocalFile(); err != nil {
		return
	}

	n, err = po.localFile.ReadAt(buf, offset)
	return
}

// Make a random access write into our view of the content. May block for
// network access. Not guaranteed to be reflected remotely until after Sync is
// called successfully.
func (po *ProxyObject) WriteAt(buf []byte, offset int64) (n int, err error) {
	if err = po.ensureLocalFile(); err != nil {
		return
	}

	po.dirty = true
	n, err = po.localFile.WriteAt(buf, offset)
	return
}

// Truncate our view of the content to the given number of bytes, extending if
// n is greater than Size(). May block for network access. Not guaranteed to be
// reflected remotely until after Sync is called successfully.
func (po *ProxyObject) Truncate(n uint64) (err error) {
	if err = po.ensureLocalFile(); err != nil {
		return
	}

	// Convert to signed, which is what os.File wants.
	if n > math.MaxInt64 {
		err = fmt.Errorf("Illegal offset: %v", n)
		return
	}

	po.dirty = true
	err = po.localFile.Truncate(int64(n))
	return
}

// Ensure that the remote object reflects the local state, returning a record
// for a generation that does. Clobbers the remote version. Does no work if the
// remote version is already up to date.
func (po *ProxyObject) Sync() (storage.Object, error)

// Ensure that po.localFile != nil and contains the correct contents.
func (po *ProxyObject) ensureLocalFile() (err error)
