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
	dir string

	// The result to return from WaitForReady. Not valid until the channel is closed.
	readyStatus          error
	readyStatusAvailable chan struct{}
}

// Wait until the mount point is ready to be used. After a successful return
// from this function, the contents of the mounted file system should be
// visible in the directory supplied to NewMountPoint. May be called multiple
// times.
func (mfs *MountedFileSystem) WaitForReady(ctx context.Context) error {
	select {
	case <-mfs.readyStatusAvailable:
		return mfs.readyStatus
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Block until the file system has been unmounted. The return value will be
// non-nil if anything unexpected happened while mounting or serving. May be
// called multiple times.
func (mfs *MountedFileSystem) Join() error

// Attempt to unmount the file system. Use Join to wait for it to actually be
// unmounted. You must first call WaitForReady to ensure there is no race with
// mounting.
func (mfs *MountedFileSystem) Unmount() error

// Runs in the background.
func (mfs *MountedFileSystem) mountAndServe(
	fs fusefs.FS,
	options []fuse.MountOption)

// Attempt to mount the supplied file system on the given directory.
// mfs.WaitForReady() must be called to find out whether the mount was
// successful.
func MountFileSystem(
	dir string,
	fs fusefs.FS,
	options ...fuse.MountOption) (mfs *MountedFileSystem) {
	// Initialize the struct.
	mfs = &MountedFileSystem{
		dir:                  dir,
		readyStatusAvailable: make(chan struct{}),
	}

	// Mount in the background.
	go mfs.mountAndServe(fs, options)

	return mfs
}
