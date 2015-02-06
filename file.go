// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
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
func (f *file) ensureTempFile(ctx context.Context) error

// Throw away the local temporary file, if any.
func (f *file) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Is there a file to close?
	if f.tempFile == nil {
		return nil
	}

	// Close it.
	if err := f.tempFile.Close(); err != nil {
		log.Println("Error closing temp file:", err)
	}

	f.tempFile = nil
	return nil
}

// Ensure that the local temporary file is initialized, then read from it.
func (f *file) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error
