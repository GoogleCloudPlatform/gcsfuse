// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strings"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

// A view on an object in GCS that allows random access reads and writes.
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

	// The name of the GCS object for which we are a proxy. Might not exist in
	// the bucket.
	name string

	/////////////////////////
	// Mutable state
	/////////////////////////

	// The specific generation of the object from which our local state is
	// branched. If we have no local state, the contents of this object are
	// exactly our contents. May be nil if NoteLatest was never called.
	//
	// INVARIANT: If source != nil, source.Size >= 0
	// INVARIANT: If source != nil, source.Name == name
	source *storage.Object

	// A local temporary file containing the contents of our source (or the empty
	// string if no source) along with any local modifications. The authority on
	// our view of the object when non-nil.
	//
	// A nil file is to be regarded as empty, but is not authoritative unless
	// source is also nil.
	localFile *os.File

	// false if the contents of localFile may be different from the contents of
	// the object referred to by source. Sync needs to do work iff this is true.
	//
	// INVARIANT: If false, then source != nil.
	dirty bool
}

// Create a new view on the GCS object with the given name. The remote object
// is assumed to be non-existent, so that the local contents are empty. Use
// NoteLatest to change that if necessary.
func NewObjectProxy(
	bucket gcs.Bucket,
	name string) (op *ObjectProxy, err error) {
	op = &ObjectProxy{
		bucket: bucket,
		name:   name,

		// Initial state: empty contents, dirty. (The remote object needs to be
		// truncated.)
		source:    nil,
		localFile: nil,
		dirty:     true,
	}

	return
}

// Return the name of the proxied object.
func (op *ObjectProxy) Name() string {
	return op.name
}

// Panic if any internal invariants are violated. Careful users can call this
// at appropriate times to help debug weirdness. Consider using
// syncutil.InvariantMutex to automate the process.
func (op *ObjectProxy) CheckInvariants() {
	if op.source != nil && op.source.Size <= 0 {
		if op.source.Size < 0 {
			panic(fmt.Sprintf("Non-sensical source size: %v", op.source.Size))
		}

		if op.source.Name != op.name {
			panic(fmt.Sprintf("Name mismatch: %s vs. %s", op.source.Name, op.name))
		}
	}

	if !op.dirty && op.source == nil {
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
func (op *ObjectProxy) NoteLatest(o *storage.Object) (err error) {
	// Sanity check the input.
	if o.Size < 0 {
		err = fmt.Errorf("Object contains negative size: %v", o.Size)
		return
	}

	if o.Name != op.name {
		err = fmt.Errorf("Object name mismatch: %s vs. %s", o.Name, op.name)
		return
	}

	// Do nothing if this is not newer than what we have.
	if op.source != nil && o.Generation <= op.source.Generation {
		return
	}

	// Throw away any local state.
	if err = op.Clean(); err != nil {
		err = fmt.Errorf("Clean: %v", err)
		return
	}

	// Update the source.
	op.source = o

	return
}

// Return the current size in bytes of our view of the content.
func (op *ObjectProxy) Size() (n uint64, err error) {
	// If we have a local file, it is authoritative.
	if op.localFile != nil {
		var fi os.FileInfo
		if fi, err = op.localFile.Stat(); err != nil {
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
	if op.source != nil {
		n = uint64(op.source.Size)
		return
	}

	// Otherwise, we are empty.
	return
}

// Make a random access read into our view of the content. May block for
// network access.
func (op *ObjectProxy) ReadAt(
	ctx context.Context,
	buf []byte,
	offset int64) (n int, err error) {
	if err = op.ensureLocalFile(ctx); err != nil {
		return
	}

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
	if err = op.ensureLocalFile(ctx); err != nil {
		return
	}

	op.dirty = true
	n, err = op.localFile.WriteAt(buf, offset)
	return
}

// Truncate our view of the content to the given number of bytes, extending if
// n is greater than Size(). May block for network access. Not guaranteed to be
// reflected remotely until after Sync is called successfully.
func (op *ObjectProxy) Truncate(ctx context.Context, n uint64) (err error) {
	if err = op.ensureLocalFile(ctx); err != nil {
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

// Ensure that the remote object reflects the local state, returning a record
// for a generation that does. Clobbers the remote version. Does no work if the
// remote version is already up to date.
func (op *ObjectProxy) Sync(ctx context.Context) (o *storage.Object, err error) {
	// Is there anything to do?
	if !op.dirty {
		o = op.source
		return
	}

	// Choose a reader.
	var contents io.Reader
	if op.localFile != nil {
		contents = op.localFile
	} else {
		contents = strings.NewReader("")
	}

	// Create a new generation of the object.
	req := &gcs.CreateObjectRequest{
		Attrs: storage.ObjectAttrs{
			Name: op.name,
		},
		Contents: contents,
	}

	if o, err = op.bucket.CreateObject(ctx, req); err != nil {
		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	// Update local state.
	op.source = o
	op.dirty = false

	return
}

// Ensure that op.localFile != nil and contains the correct contents.
func (op *ObjectProxy) ensureLocalFile(ctx context.Context) (err error) {
	// If we've already got a local file, we're done.
	if op.localFile != nil {
		return
	}

	// Create a temporary file.
	var f *os.File
	if f, err = ioutil.TempFile("", "gcsproxy"); err != nil {
		err = fmt.Errorf("ioutil.TempFile: %v", err)
		return
	}

	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	// If we have a source, then we must fetch its contents.
	//
	// TODO(jacobsa): We need to plumb in a particular generation here, or we may
	// consider ourselves to have branched from the wrong generation, causing
	// write clobbering. For example:
	//
	//  1. Initial state: object is at generation N.
	//  2. User lists directory, sees generation N.
	//  3. Other user writes generation N+1.
	//  4. User opens file, we read generation N+1 but still think we're at N.
	//  5. User makes local modifications.
	//  6. User lists directory again, sees generation N+1.
	//
	// At the end of this process, the local modifications are blown away even
	// though in actual fact they were based on generation N+1.
	if op.source != nil {
		var reader io.Reader
		if reader, err = op.bucket.NewReader(ctx, op.name); err != nil {
			err = fmt.Errorf("NewReader: %v", err)
			return
		}

		if _, err = io.Copy(f, reader); err != nil {
			err = fmt.Errorf("io.Copy: %v", err)
			return
		}
	}

	// Snarf the file.
	op.localFile, f = f, nil

	return
}

// Throw away any local modifications to the object, reverting to the latest
// version handed to NoteLatest (or the non-existent object if none). Watch
// out!
//
// Careful users should call this in order to clean up local state before
// dropping all references to the object proxy.
func (op *ObjectProxy) Clean() (err error) {
	// Throw out the local file, if any.
	if op.localFile != nil {
		path := op.localFile.Name()

		if err = op.localFile.Close(); err != nil {
			err = fmt.Errorf("Closing local file: %v", err)
			return
		}

		if err = os.Remove(path); err != nil {
			err = fmt.Errorf("Unlinking local file: %v", err)
			return
		}
	}

	op.localFile = nil

	// We are now dirty iff we have never seen a remote source.
	op.dirty = (op.source == nil)

	return
}
