// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fs

import (
	"bazil.org/fuse/fs"
	"github.com/jacobsa/gcloud/gcs"
)

type fileSystem struct {
	bucket gcs.Bucket
}

func (fs *fileSystem) Root() (fs.Node, error) {
	d := &dir{
		bucket: fs.bucket,
	}

	return d, nil
}

// Create a fuse file system whose root directory is the root of the supplied
// bucket.
func NewFuseFS(bucket gcs.Bucket) (fs.FS, error) {
	return &fileSystem{bucket}, nil
}
