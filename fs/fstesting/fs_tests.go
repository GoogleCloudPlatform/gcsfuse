// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)
//
// Tests registered by RegisterFSTests.

package fstesting

import (
	"io/ioutil"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcsfuse/fs"
	"github.com/jacobsa/gcsfuse/fuseutil"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

////////////////////////////////////////////////////////////////////////
// Common
////////////////////////////////////////////////////////////////////////

type fsTest struct {
	ctx               context.Context
	bucket            gcs.Bucket
	mountedFileSystem *fuseutil.MountedFileSystem
}

var _ fsTestInterface = &fsTest{}

func (t *fsTest) setUpFsTest(b gcs.Bucket) {
	var err error

	// Record bucket and context information.
	t.bucket = b
	t.ctx = context.Background()

	// Set up a temporary directory for mounting.
	mountPoint, err := ioutil.TempDir("", "fs_test")
	if err != nil {
		panic("ioutil.TempDir: " + err.Error())
	}

	// Mount a file system.
	fileSystem, err := fs.NewFuseFS(b)
	if err != nil {
		panic("NewFuseFS: " + err.Error())
	}

	t.mountedFileSystem = fuseutil.MountFileSystem(mountPoint, fileSystem)
	if err := t.mountedFileSystem.WaitForReady(t.ctx); err != nil {
		panic("MountedFileSystem.WaitForReady: " + err.Error())
	}
}

func (t *fsTest) tearDownFsTest() {
	// Unmount the file system.
	if err := t.mountedFileSystem.Unmount(); err != nil {
		panic("MountedFileSystem.Unmount: " + err.Error())
	}

	if err := t.mountedFileSystem.Join(t.ctx); err != nil {
		panic("MountedFileSystem.Join: " + err.Error())
	}
}

////////////////////////////////////////////////////////////////////////
// Read-only interaction
////////////////////////////////////////////////////////////////////////

type readOnlyTest struct {
	fsTest
}

func (t *readOnlyTest) DoesFoo() {
	AssertTrue(false, "TODO")
}
