// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"os"

	"bazil.org/fuse"
)

const dirSeparator = '/'

// A "directory" in GCS, defined by an object name prefix. All prefixes end
// with dirSeparator except for the special case of the root directory, where
// the prefix is the empty string.
type dir struct {
	prefix string
}

func (d *dir) Attr() fuse.Attr {
	return fuse.Attr{
		// TODO(jacobsa): Expose ACLs from GCS?
		Mode: os.ModeDir | 0700,
	}
}
