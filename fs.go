// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

type fileSystem struct {
}

func (fs *fileSystem) Root() (fs.Node, fuse.Error) {
	return nil, fuse.ENOSYS
}
