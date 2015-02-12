// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fuseutil

import (
	"errors"
	"log"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

// A struct representing the status of a mount operation, with methods for
// waiting on the mount to complete, waiting for unmounting, and causing
// unmounting.
type MountedFileSystem struct {
	dir string

	// The result to return from WaitForReady. Not valid until the channel is
	// closed.
	readyStatus          error
	readyStatusAvailable chan struct{}

	// The result to return from Join. Not valid until the channel is closed.
	joinStatus          error
	joinStatusAvailable chan struct{}
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

// Block until a mounted file system has been unmounted. The return value will
// be non-nil if anything unexpected happened while serving. May be called
// multiple times. Must not be called unless WaitForReady has returned nil.
func (mfs *MountedFileSystem) Join(ctx context.Context) error {
	select {
	case <-mfs.joinStatusAvailable:
		return mfs.joinStatus
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Attempt to unmount the file system. Use Join to wait for it to actually be
// unmounted. You must first call WaitForReady to ensure there is no race with
// mounting.
func (mfs *MountedFileSystem) Unmount() error {
	return fuse.Unmount(mfs.dir)
}

// Runs in the background.
func (mfs *MountedFileSystem) mountAndServe(
	fs fusefs.FS,
	options []fuse.MountOption) {
	// Open a FUSE connection.
	log.Println("Opening a FUSE connection.")
	c, err := fuse.Mount(mfs.dir, options...)
	if err != nil {
		mfs.readyStatus = errors.New("fuse.Mount: " + err.Error())
		close(mfs.readyStatusAvailable)
		return
	}

	defer c.Close()

	// Start a goroutine that will notify the MountedFileSystem object when the
	// connection says it is ready (or it fails to become ready).
	go func() {
		<-c.Ready
		mfs.readyStatus = c.MountError
		close(mfs.readyStatusAvailable)
	}()

	// Serve the connection using the file system object.
	if err := fusefs.Serve(c, fs); err != nil {
		mfs.joinStatus = errors.New("fusefs.Serve: " + err.Error())
		close(mfs.joinStatusAvailable)
		return
	}

	// Signal that everything is okay.
	close(mfs.joinStatusAvailable)
}

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
		joinStatusAvailable:  make(chan struct{}),
	}

	// Mount in the background.
	go mfs.mountAndServe(fs, options)

	return mfs
}
