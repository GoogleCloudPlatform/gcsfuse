// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"golang.org/x/net/context"

	"bazil.org/fuse"
)

type file struct {
	authContext context.Context
	bucketName  string
	objectName  string
}

func (f *file) Attr() fuse.Attr {
	return fuse.Attr{
		// TODO(jacobsa): Expose ACLs from GCS?
		Mode: 0400,
	}
}
