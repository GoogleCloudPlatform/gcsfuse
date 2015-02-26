// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fuseutil

import "bazil.org/fuse"

const (
	// Errors corresponding to kernel error numbers. These may be treated
	// specially when returned by a FileSystem method.
	ENOSYS = fuse.ENOSYS
)
