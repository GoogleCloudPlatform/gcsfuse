// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package samples

import (
	"github.com/jacobsa/gcsfuse/fuseutil"
	"github.com/jacobsa/gcsfuse/timeutil"
	"golang.org/x/net/context"
)

// A file system with a fixed structure that looks like this:
//
//     hello
//     dir/
//         world
//
// Each file contains the string "Hello, world!".
type HelloFS struct {
	fuseutil.NotImplementedFileSystem
	Clock timeutil.Clock
}

var _ fuseutil.FileSystem = &HelloFS{}

func (fs *HelloFS) Open(
	ctx context.Context,
	req *fuseutil.OpenRequest) (resp *fuseutil.OpenResponse, err error) {
	// We always allow opening the root directory.
	if req.Inode == fuseutil.RootInodeID {
		return
	}

	// TODO(jacobsa): Handle others.
	err = fuseutil.ENOSYS
	return
}
