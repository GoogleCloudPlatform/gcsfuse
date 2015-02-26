// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fuseutil

import "golang.org/x/net/context"

type NotImplementedFileSystem struct {
}

var _ FileSystem = &NotImplementedFileSystem{}

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
