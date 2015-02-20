// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fs

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
)

// A remote object's name and metadata, along with a local temporary file that
// contains its contents (when initialized).
//
// TODO(jacobsa): After becoming comfortable with the representation of dir and
// its concurrency protection, audit this file and make sure it is up to par.
type file struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	logger *log.Logger
	bucket gcs.Bucket

	/////////////////////////
	// Constant data
	/////////////////////////

	objectName string

	/////////////////////////
	// Mutable state
	/////////////////////////

	mu syncutil.InvariantMutex

	// A local temporary file containing the current contents of the logical
	// file. Lazily created. When non-ni, this is authoritative.
	tempFile *os.File // GUARDED_BY(mu)

	// Set to true when we need to flush tempFile to GCS before allowing the user
	// to successfully close the file. false implies that the GCS object is up to
	// date (or has been modified only by a foreign machine).
	//
	// INVARIANT: If true, then tempFile != nil
	tempFileDirty bool // GUARDED_BY(mu)

	// When tempFile == nil, the current size of the object named objectName on
	// GCS, as far as we are aware.
	//
	// INVARIANT: If tempFile != nil, then remoteSize == 0
	remoteSize uint64 // GUARDED_BY(mu)
}

// Make sure file implements the interfaces we think it does.
var (
	_ fusefs.Node = &file{}

	_ fusefs.Handle         = &file{}
	_ fusefs.HandleFlusher  = &file{}
	_ fusefs.HandleReader   = &file{}
	_ fusefs.HandleReleaser = &file{}
	_ fusefs.HandleWriter   = &file{}
)

func newFile(
	logger *log.Logger,
	bucket gcs.Bucket,
	objectName string,
	remoteSize uint64) *file {
	f := &file{
		logger:     logger,
		bucket:     bucket,
		objectName: objectName,
		remoteSize: remoteSize,
	}

	f.mu = syncutil.NewInvariantMutex(func() { f.checkInvariants() })

	return f
}

func (f *file) checkInvariants()

func (f *file) Attr() fuse.Attr {
	return fuse.Attr{
		// TODO(jacobsa): Expose ACLs from GCS?
		Mode: 0400,
		// TODO(jacobsa): Catch the bug here (that this may be wrong when
		// f.tempFile != nil) with a test, then fix it.
		Size: f.remoteSize,
	}
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

	return nil
}

// Throw away the local temporary file, if any.
//
// LOCKS_EXCLUDED(f.mu)
func (f *file) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Is there a file to close?
	if f.tempFile == nil {
		return nil
	}

	// Close it, after grabbing its path.
	path := f.tempFile.Name()
	if err := f.tempFile.Close(); err != nil {
		f.logger.Println("Error closing temp file:", err)
	}

	// Attempt to delete it.
	if err := os.Remove(path); err != nil {
		f.logger.Println("Error deleting temp file:", err)
	}

	f.tempFile = nil
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

	// Ensure the temp file is present. If it's not, grab the current contents
	// from GCS.
	if err = f.ensureTempFile(ctx); err != nil {
		err = fmt.Errorf("ensureTempFile: %v", err)
		return
	}

	// Write to the temp file.
	resp.Size, err = f.tempFile.WriteAt(req.Data, req.Offset)
	return
}

// Put the temporary file back in the bucket if it's dirty.
//
// LOCKS_EXCLUDED(f.mu)
func (f *file) Flush(
	ctx context.Context,
	req *fuse.FlushRequest) (err error) {
	err = errors.New("TODO(jacobsa): file.Flush.")
	return
}
