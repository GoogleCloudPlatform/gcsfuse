// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fs

import (
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"github.com/jacobsa/gcsfuse/gcsproxy"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
)

// A thin wrapper around an object proxy for an object in GCS, implementing the
// interfaces necessary for the fusefs package.
//
// TODO(jacobsa): After becoming comfortable with the representation of dir and
// its concurrency protection, audit this file and make sure it is up to par.
type file struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	logger *log.Logger

	/////////////////////////
	// Mutable state
	/////////////////////////

	mu syncutil.InvariantMutex

	// A proxy for the file contents in GCS.
	//
	// INVARIANT: objectProxy.CheckInvariants() doesn't panic
	objectProxy *gcsproxy.ObjectProxy // PT_GUARDED_BY(mu)
}

// Make sure file implements the interfaces we think it does.
var (
	_ fusefs.Node          = &file{}
	_ fusefs.NodeGetattrer = &file{}

	_ fusefs.Handle         = &file{}
	_ fusefs.HandleFlusher  = &file{}
	_ fusefs.HandleReader   = &file{}
	_ fusefs.HandleReleaser = &file{}
	_ fusefs.HandleWriter   = &file{}
)

func newFile(
	logger *log.Logger,
	bucket gcs.Bucket,
	remoteObject *storage.Object) (f *file, err error) {
	// Create the struct.
	f = &file{
		logger: logger,
	}

	// Set up the object proxy.
	f.objectProxy, err = gcsproxy.NewObjectProxy(bucket, remoteObject.Name)
	if err != nil {
		err = fmt.Errorf("NewObjectProxy: %v", err)
		return
	}

	// Set up the mutex.
	f.mu = syncutil.NewInvariantMutex(f.checkInvariants)

	return
}

// SHARED_LOCKS_REQUIRED(f.mu)
func (f *file) checkInvariants() {
	f.objectProxy.CheckInvariants()
}

func generateInodeNumber(objectName string) uint64 {
	h := fnv.New64()
	if _, err := io.WriteString(h, objectName); err != nil {
		panic(err)
	}

	return h.Sum64()
}

// TODO(jacobsa): Share code with Getattr below.
//
// LOCKS_EXCLUDED(f.mu)
func (f *file) Attr() fuse.Attr {
	f.mu.RLock()
	defer f.mu.RUnlock()

	f.logger.Printf("Attr: [%s]/%s", f.bucket.Name(), f.objectName)

	// Find the current size.
	size := f.remoteSize
	if f.tempFile != nil {
		// See if the scenario in the TODO below can ever even come to pass. If so,
		// file a bug with the fuse package to see what we're intended to do with
		// errors.
		panic("Received an Attr call with a temp file active.")

		fi, err := f.tempFile.Stat()
		if err != nil {
			// TODO(jacobsa): What do we do about this? The fuse package gives us no
			// opportunity for returning an error here.
			panic("tempFile.Stat: " + err.Error())
		}

		size = uint64(fi.Size())
	}

	return fuse.Attr{
		Inode: generateInodeNumber(f.objectName),
		Mode:  0700,
		Size:  size,
	}
}

// LOCKS_EXCLUDED(f.mu)
func (f *file) Getattr(
	ctx context.Context,
	req *fuse.GetattrRequest,
	resp *fuse.GetattrResponse) (err error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	f.logger.Printf("Getattr: [%s]/%s", f.bucket.Name(), f.objectName)

	// Find the current size.
	size := f.remoteSize
	if f.tempFile != nil {
		var fi os.FileInfo
		if fi, err = f.tempFile.Stat(); err != nil {
			err = fmt.Errorf("tempFile.Stat: %v", err)
			return
		}

		size = uint64(fi.Size())
	}

	resp.Attr = fuse.Attr{
		Inode: generateInodeNumber(f.objectName),
		Mode:  0700,
		Size:  size,

		// TODO(jacobsa): Do something more useful here.
		Mtime: time.Now(),
	}

	return
}

// If the file contents have not yet been fetched to a temporary file, fetch
// them.
//
// EXCLUSIVE_LOCKS_REQUIRED(f.mu)
func (f *file) ensureTempFile(ctx context.Context) error {
	// Do we already have a file?
	if f.tempFile != nil {
		return nil
	}

	// Create a temporary file.
	tempFile, err := ioutil.TempFile("", "gcsfuse")
	if err != nil {
		return fmt.Errorf("ioutil.TempFile: %v", err)
	}

	// Create a reader for the object.
	readCloser, err := f.bucket.NewReader(ctx, f.objectName)
	if err != nil {
		return fmt.Errorf("bucket.NewReader: %v", err)
	}

	defer readCloser.Close()

	// Copy the object contents into the file.
	if _, err := io.Copy(tempFile, readCloser); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}

	// Save the file for later.
	f.tempFile = tempFile

	// remoteSize is no longer authoritative.
	f.remoteSize = 0

	return nil
}

// Throw away the local temporary file, if any.
//
// TODO(jacobsa): There are a few bugs here.
//
// 1. This is called when a file descriptor is closed (actually the last clone
// of a file descriptor? -- test this), not when the last file descriptor
// referring to an inode is closed. We don't want to throw away the temp file
// just yet. Add tests for this.
//
// 2. When we do throw out the temp file (probably when the inode is being
// forgotten?, we need to write it back if it's dirty.
//
// Add tests for all of these bugs before fixing them.
//
// LOCKS_EXCLUDED(f.mu)
func (f *file) Release(
	ctx context.Context,
	req *fuse.ReleaseRequest) (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.logger.Printf("Release: [%s]/%s", f.bucket.Name(), f.objectName)

	// Is there a file to close?
	if f.tempFile == nil {
		return nil
	}

	// Stat it, for use below.
	fileInfo, err := f.tempFile.Stat()
	if err != nil {
		err = fmt.Errorf("Statting temp file: %v", err)
		return
	}

	// Close it, after grabbing its path.
	path := f.tempFile.Name()
	if err = f.tempFile.Close(); err != nil {
		err = fmt.Errorf("Closing temp file: %v", err)
		return
	}

	// Attempt to delete it.
	if err = os.Remove(path); err != nil {
		err = fmt.Errorf("Deleting temp file: %v", err)
		return
	}

	f.tempFile = nil
	f.tempFileDirty = false
	f.remoteSize = uint64(fileInfo.Size())

	return nil
}

// Ensure that the local temporary file is initialized, then read from it.
//
// LOCKS_EXCLUDED(f.mu)
func (f *file) Read(
	ctx context.Context,
	req *fuse.ReadRequest,
	resp *fuse.ReadResponse) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.logger.Printf("Read: [%s]/%s", f.bucket.Name(), f.objectName)

	// Ensure the temp file is present.
	if err := f.ensureTempFile(ctx); err != nil {
		return err
	}

	// Allocate a response buffer.
	resp.Data = make([]byte, req.Size)

	// Read the data.
	n, err := f.tempFile.ReadAt(resp.Data, req.Offset)
	resp.Data = resp.Data[:n]

	// Special case: read(2) doesn't return EOF errors.
	if err == io.EOF {
		err = nil
	}

	return err
}

// Ensure that the local temporary file is initialized, then write to it.
//
// LOCKS_EXCLUDED(f.mu)
func (f *file) Write(
	ctx context.Context,
	req *fuse.WriteRequest,
	resp *fuse.WriteResponse) (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.logger.Printf("Write: [%s]/%s", f.bucket.Name(), f.objectName)

	// Ensure the temp file is present. If it's not, grab the current contents
	// from GCS.
	if err = f.ensureTempFile(ctx); err != nil {
		err = fmt.Errorf("ensureTempFile: %v", err)
		return
	}

	// Mark us dirty.
	f.tempFileDirty = true

	// Write to the temp file.
	resp.Size, err = f.tempFile.WriteAt(req.Data, req.Offset)

	return
}

// Put the temporary file back in the bucket if it's dirty.
//
// TODO(jacobsa): This probably isn't the write place to do this. ext2 doesn't
// appear to do anything at all for i_op->flush, for example, and the fuse
// documentation pretty much says as much (http://goo.gl/KkBJM3 "Filesystems
// shouldn't assume that flush will always be called after some writes, or that
// if will be called at all"). Instead:
//
//  1. We should definitely do it on fsync, because the user asked.
//  2. We should definitely do it when the kernel is forgetting the inode,
//     because we won't get another chance.
//  3. Maybe we should do it after some timeout after the file is closed (the
//     file handle is released).
//
// Avoid doing #3 for now, because the kernel may already forget the inode
// after some timeout after it is unused, to avoid data loss due to power loss.
// Do #3 only if it becomes clear that #2 is not sufficient for real users.
//
// I wonder if we also need to do it when destroying. Will the kernel send us
// forgets before destroying? Again, add only if needed.
//
// LOCKS_EXCLUDED(f.mu)
func (f *file) Flush(
	ctx context.Context,
	req *fuse.FlushRequest) (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.logger.Printf("Flush: [%s]/%s", f.bucket.Name(), f.objectName)

	// Is there anything interesting for us to do?
	if !f.tempFileDirty {
		return
	}

	// Flush the temp file to GCS.
	createReq := &gcs.CreateObjectRequest{
		Attrs: storage.ObjectAttrs{
			Name: f.objectName,
		},
		Contents: f.tempFile,
	}

	if _, err = f.bucket.CreateObject(ctx, createReq); err != nil {
		err = fmt.Errorf("bucket.CreateObject: %v", err)
		return
	}

	// We are no longer dirty.
	//
	// TODO(jacobsa): Add a test for this. Cause a flush to happen, then
	// overwrite object contents out of band, then make sure they don't restore
	// next time flush happens.
	f.tempFileDirty = false

	return
}
