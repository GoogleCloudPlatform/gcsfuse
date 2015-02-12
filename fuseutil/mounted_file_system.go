// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fuseutil

import (
	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

// A struct representing the status of a mount operation, with methods for
// waiting on the mount to complete, waiting for unmounting, and causing
// unmounting.
type MountedFileSystem struct {
}

// Wait until the mount point is ready to be used. After a successful return
// from this function, the contents of the mounted file system should be
// visible in the directory supplied to NewMountPoint. May be called multiple
// times.
func (mfs *MountedFileSystem) WaitForReady(ctx context.Context) error

// Block until the file system has been unmounted. The return value will be
// non-nil if anything unexpected happened while mounting or serving. May be
// called multiple times.
func (mfs *MountedFileSystem) Join() error

// Attempt to unmount the file system. Use Join to wait for it to actually be
// unmounted.
func (mfs *MountedFileSystem) Unmount() error

// Attempt to mount the supplied file system on the given directory.
// mfs.WaitForReady() must be called to find out whether the mount was
// successful.
func MountFileSystem(
	dir string,
	fs fusefs.FS,
	options ...fuse.MountOption) (mfs *MountedFileSystem)
