// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"os"

	"golang.org/x/net/context"

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
func (d *dir) readDirWithContext(ctx context.Context) ([]fuse.Dirent, fuse.Error)

func (d *dir) ReadDir(intr fs.Intr) ([]fuse.Dirent, fuse.Error) {
	ctx, cancel := withIntr(d.authContext, intr)
	defer cancel()

	return d.readDirWithContext(ctx)
}
