// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/jacobsa/gcsfs/gcs"
)

type fileSystem struct {
	bucket gcs.Bucket
}

func (fs *fileSystem) Root() (fs.Node, fuse.Error) {
	d := &dir{
		bucket: fs.bucket,
	}

	return d, nil
}
