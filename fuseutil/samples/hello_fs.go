// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package samples

import "github.com/jacobsa/gcsfuse/fuseutil"

type HelloFS struct {
	fuseutil.NotImplementedFileSystem
}

var _ fuseutil.FileSystem = &HelloFS{}
