// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy

import (
	"io"

	"github.com/jacobsa/gcloud/gcs"
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
}

var _ io.ReaderAt = &ProxyObject{}
var _ io.WriterAt = &ProxyObject{}

// Create a new view on the GCS object with the given name. The object is
// assumed to be non-existent, so that all non-empty reads return an error. Use
// NoteLatest to change that if necessary.
func NewProxyObject(
	bucket gcs.Bucket,
	name string) (*ProxyObject, error)

// Inform the proxy object of the most recently observed generation of the
// object of interest in GCS.
//
// If this is no newer than the newest generation that has previously been
// observed, it is ignored. Otherwise, it becomes the definitive source of data
// for the object. Any local caches are thrown out, including local
// modifications.
func (po *ProxyObject) NoteLatest(o storage.Object) error

func (po *ProxyObject) Size() uint64

func (po *ProxyObject) ReadAt(buf []byte, offset int64) (int, error)

func (po *ProxyObject) WriteAt(buf []byte, offset int64) (int, error)

func (po *ProxyObject) Truncate(n uint64) error

// Ensure that the remote object reflects the local state, returning a record
// for a generation that does. Clobbers the remote version.
func (po *ProxyObject) Sync() (storage.Object, error)
