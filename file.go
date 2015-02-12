// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// A remote object's name and metadata, along with a local temporary file that
// contains its contents (when initialized).
type file struct {
	bucket     gcs.Bucket
	objectName string
	size       uint64

	mu       sync.RWMutex
	tempFile *os.File // GUARDED_BY(mu)
}

// Make sure file implements the interfaces we think it does.
var (
	_ fs.Node           = &file{}
	_ fs.Handle         = &file{}
	_ fs.HandleReader   = &file{}
	_ fs.HandleReleaser = &file{}
)

func (f *file) Attr() fuse.Attr {
	return fuse.Attr{
		// TODO(jacobsa): Expose ACLs from GCS?
		Mode: 0400,
		Size: f.size,
	}
}

// If the file contents have not yet been fetched to a temporary file, fetch
// them.
func (f *file) ensureTempFile(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

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
		log.Println("Error closing temp file:", err)
	}

	// Attempt to delete it.
	if err := os.Remove(path); err != nil {
		log.Println("Error deleting temp file:", err)
	}

	f.tempFile = nil
	return nil
}

// Ensure that the local temporary file is initialized, then read from it.
func (f *file) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	// Ensure the temp file is present.
	if err := f.ensureTempFile(ctx); err != nil {
		return err
	}

	// Lock to read the temp file. If it went away in the meantime, that means
	// the kernel (erroneously) released us while reading from us.
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Allocate a response buffer.
	resp.Data = make([]byte, req.Size)

	// Read the data.
	_, err := f.tempFile.ReadAt(resp.Data, req.Offset)

	return err
}
