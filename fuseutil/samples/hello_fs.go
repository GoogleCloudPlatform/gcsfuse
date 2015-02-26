// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package samples

import (
	"github.com/jacobsa/gcsfuse/fuseutil"
	"github.com/jacobsa/gcsfuse/timeutil"
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
