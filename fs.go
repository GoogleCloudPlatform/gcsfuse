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
	bucketName  string
}

func (fs *fileSystem) Root() (fs.Node, fuse.Error) {
	d := &dir{
		authContext: fs.authContext,
		bucketName:  fs.bucketName,
	}

	return d, nil
}
