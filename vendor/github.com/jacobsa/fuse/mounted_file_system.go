// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fuse

import (
	"context"
	"fmt"
)

// MountedFileSystem represents the status of a mount operation, with a method
// that waits for unmounting.
type MountedFileSystem struct {
	dir string

	// The result to return from Join. Not valid until the channel is closed.
	joinStatus          error
	joinStatusAvailable chan struct{}
}

// Dir returns the directory on which the file system is mounted (or where we
// attempted to mount it.)
func (mfs *MountedFileSystem) Dir() string {
	return mfs.dir
}

// Join blocks until a mounted file system has been unmounted. It does not
// return successfully until all ops read from the connection have been
// responded to (i.e. the file system server has finished processing all
// in-flight ops).
//
// The return value will be non-nil if anything unexpected happened while
// serving. May be called multiple times.
func (mfs *MountedFileSystem) Join(ctx context.Context) error {
	select {
	case <-mfs.joinStatusAvailable:
		return mfs.joinStatus
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetFuseContext implements the equiv. of FUSE-C fuse_get_context() and thus
// returns the UID / GID / PID associated with all FUSE requests send by the kernel.
// ctx parameter must be one of the context from the fuseops handlers (e.g.: CreateFile)
func (mfs *MountedFileSystem) GetFuseContext(ctx context.Context) (uid, gid, pid uint32, err error) {
	foo := ctx.Value(contextKey)
	state, ok := foo.(opState)
	if !ok {
		return 0, 0, 0, fmt.Errorf("GetFuseContext called with invalid context: %#v", ctx)
	}
	inMsg := state.inMsg
	header := inMsg.Header()
	return header.Uid, header.Gid, header.Pid, nil
}
