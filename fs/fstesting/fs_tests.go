// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)
//
// Tests registered by RegisterFSTests.

package fstesting

import (
	"io/ioutil"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	"github.com/jacobsa/gcsfuse/fs"
	"github.com/jacobsa/gcsfuse/fuseutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

////////////////////////////////////////////////////////////////////////
// Common
////////////////////////////////////////////////////////////////////////

type fsTest struct {
	ctx    context.Context
	bucket gcs.Bucket
	mfs    *fuseutil.MountedFileSystem
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

	t.mfs = fuseutil.MountFileSystem(mountPoint, fileSystem)
	if err := t.mfs.WaitForReady(t.ctx); err != nil {
		panic("MountedFileSystem.WaitForReady: " + err.Error())
	}
}

func (t *fsTest) tearDownFsTest() {
	// Unmount the file system.
	if err := t.mfs.Unmount(); err != nil {
		panic("MountedFileSystem.Unmount: " + err.Error())
	}

	if err := t.mfs.Join(t.ctx); err != nil {
		panic("MountedFileSystem.Join: " + err.Error())
	}
}

func (t *fsTest) createObjects(objects []*gcsutil.ObjectInfo) error {
	_, err := gcsutil.CreateObjects(t.ctx, t.bucket, objects)
	return err
}

////////////////////////////////////////////////////////////////////////
// Read-only interaction
////////////////////////////////////////////////////////////////////////

type readOnlyTest struct {
	fsTest
}

func (t *readOnlyTest) EmptyRoot() {
	// ReadDir
	entries, err := ioutil.ReadDir(t.mfs.Dir())
	AssertEq(nil, err)

	ExpectThat(entries, ElementsAre())
}

func (t *readOnlyTest) ContentsInRoot() {
	AssertEq(
		nil,
		t.createObjects(
			[]*gcsutil.ObjectInfo{
				// File
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "foo",
					},
					Contents: "taco",
				},

				// Directory
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "bar/",
					},
				},

				// File
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "baz",
					},
					Contents: "burrito",
				},
			}))

	// ReadDir
	_, err := ioutil.ReadDir(t.mfs.Dir())
	AssertEq(nil, err)

	AssertTrue(false, "TODO")
}

func (t *readOnlyTest) EmptySubDirectory() {
	AssertTrue(false, "TODO")
}

func (t *readOnlyTest) ContentsInSubDirectory() {
	AssertTrue(false, "TODO")
}

func (t *readOnlyTest) ContentsInLeafDirectory() {
	AssertTrue(false, "TODO")
}

// TODO(jacobsa): Error conditions
func (t *readOnlyTest) DoesFoo() {
	AssertTrue(false, "TODO")
}
