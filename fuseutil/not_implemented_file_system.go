// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fuseutil

type NotImplementedFileSystem struct {
}

var _ FileSystem = NotImplementedFileSystem{}
