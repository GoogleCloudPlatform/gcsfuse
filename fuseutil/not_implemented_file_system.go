// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fuseutil

import "golang.org/x/net/context"

// Embed this within your file system type to inherit default implementations
// of all methods that return ENOSYS.
type NotImplementedFileSystem struct {
}

var _ FileSystem = &NotImplementedFileSystem{}

func (fs *NotImplementedFileSystem) Open(
	ctx context.Context,
	req *OpenRequest) (*OpenResponse, error) {
	return nil, ENOSYS
}

func (fs *NotImplementedFileSystem) Lookup(
	ctx context.Context,
	req *LookupRequest) (*LookupResponse, error) {
	return nil, ENOSYS
}

func (fs *NotImplementedFileSystem) Forget(
	ctx context.Context,
	req *ForgetRequest) (*ForgetResponse, error) {
	return nil, ENOSYS
}
