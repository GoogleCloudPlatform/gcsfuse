// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"github.com/jacobsa/gcloud/gcs"

	"bazil.org/fuse"
)

type file struct {
	bucket     gcs.Bucket
	objectName string
	size       uint64
}

func (f *file) Attr() fuse.Attr {
	return fuse.Attr{
		// TODO(jacobsa): Expose ACLs from GCS?
		Mode: 0400,
		Size: f.size,
	}
}
