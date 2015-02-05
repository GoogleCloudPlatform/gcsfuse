// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type fileSystem struct {
	authContext context.Context
}

func (fs *fileSystem) Root() (fs.Node, fuse.Error) {
	d := &dir{
		authContext: fs.authContext,
		bucketName:  "TODO(jacobsa): Accept a bucket name in a flag.",
	}

	return d, nil
}
