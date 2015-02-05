// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"log"
	"os"

	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

const dirSeparator = '/'

// A "directory" in GCS, defined by an object name prefix. All prefixes end
// with dirSeparator except for the special case of the root directory, where
// the prefix is the empty string.
type dir struct {
	authContext  context.Context
	bucketName   string
	objectPrefix string
}

func (d *dir) Attr() fuse.Attr {
	return fuse.Attr{
		// TODO(jacobsa): Expose ACLs from GCS?
		Mode: os.ModeDir | 0700,
	}
}

// A version of readDir that is context-aware. The context must contain auth
// information.
func (d *dir) readDirWithContext(ctx context.Context) (
	ents []fuse.Dirent, fuseErr fuse.Error) {
	// List repeatedly until there is no more to list.
	query := &storage.Query{
		Delimiter: string(dirSeparator),
		Prefix:    d.objectPrefix,
	}

	for query != nil {
		// Grab one set of results.
		objects, err := storage.ListObjects(ctx, d.bucketName, query)
		if err != nil {
			fuse.Debug("storage.ListObjects: " + err.Error())
			return nil, fuse.EIO
		}

		// Extract objects as files.
		log.Fatal("TODO")

		// Extract prefixes as directories.
		log.Fatal("TODO")

		// Move on to the next set of results.
		query = objects.Next
	}

	return
}

func (d *dir) ReadDir(intr fs.Intr) ([]fuse.Dirent, fuse.Error) {
	ctx, cancel := withIntr(d.authContext, intr)
	defer cancel()

	return d.readDirWithContext(ctx)
}
