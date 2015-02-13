// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)
//
// Tests registered by RegisterFSTests.

package fstesting

import (
	"io/ioutil"
	"math"
	"os"
	"path"
	"syscall"
	"time"

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

func (t *fsTest) createEmptyObjects(names []string) error {
	_, err := gcsutil.CreateEmptyObjects(t.ctx, t.bucket, names)
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
	// Set up contents.
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

				// File in sub-directory
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "qux/asdf",
					},
					Contents: "",
				},
			}))

	// ReadDir
	entries, err := ioutil.ReadDir(t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(4, len(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	ExpectEq("bar", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(os.ModeDir|os.FileMode(0500), e.Mode())
	ExpectLt(math.Abs(time.Since(e.ModTime()).Seconds()), 30)
	ExpectTrue(e.IsDir())

	// baz
	e = entries[1]
	ExpectEq("baz", e.Name())
	ExpectEq(len("burrito"), e.Size())
	ExpectEq(os.FileMode(0400), e.Mode())
	ExpectLt(math.Abs(time.Since(e.ModTime()).Seconds()), 30)
	ExpectFalse(e.IsDir())

	// foo
	e = entries[2]
	ExpectEq("foo", e.Name())
	ExpectEq(len("taco"), e.Size())
	ExpectEq(os.FileMode(0400), e.Mode())
	ExpectLt(math.Abs(time.Since(e.ModTime()).Seconds()), 30)
	ExpectFalse(e.IsDir())

	// qux
	e = entries[3]
	ExpectEq("qux", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(os.ModeDir|os.FileMode(0500), e.Mode())
	ExpectLt(math.Abs(time.Since(e.ModTime()).Seconds()), 30)
	ExpectTrue(e.IsDir())
}

func (t *readOnlyTest) EmptySubDirectory() {
	// Set up an empty directory placeholder called 'bar'.
	AssertEq(nil, t.createEmptyObjects([]string{"bar/"}))

	// ReadDir
	entries, err := ioutil.ReadDir(path.Join(t.mfs.Dir(), "bar"))
	AssertEq(nil, err)

	ExpectThat(entries, ElementsAre())
}

func (t *readOnlyTest) ContentsInSubDirectory_PlaceholderPresent() {
	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			[]*gcsutil.ObjectInfo{
				// Placeholder
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/",
					},
					Contents: "",
				},

				// File
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/foo",
					},
					Contents: "taco",
				},

				// Directory
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/bar/",
					},
				},

				// File
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/baz",
					},
					Contents: "burrito",
				},

				// File in sub-directory
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/qux/asdf",
					},
					Contents: "",
				},
			}))

	// ReadDir
	entries, err := ioutil.ReadDir(path.Join(t.mfs.Dir(), "dir"))
	AssertEq(nil, err)

	AssertEq(4, len(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	ExpectEq("bar", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(os.ModeDir|os.FileMode(0500), e.Mode())
	ExpectLt(math.Abs(time.Since(e.ModTime()).Seconds()), 30)
	ExpectTrue(e.IsDir())

	// baz
	e = entries[1]
	ExpectEq("baz", e.Name())
	ExpectEq(len("burrito"), e.Size())
	ExpectEq(os.FileMode(0400), e.Mode())
	ExpectLt(math.Abs(time.Since(e.ModTime()).Seconds()), 30)
	ExpectFalse(e.IsDir())

	// foo
	e = entries[2]
	ExpectEq("foo", e.Name())
	ExpectEq(len("taco"), e.Size())
	ExpectEq(os.FileMode(0400), e.Mode())
	ExpectLt(math.Abs(time.Since(e.ModTime()).Seconds()), 30)
	ExpectFalse(e.IsDir())

	// qux
	e = entries[3]
	ExpectEq("qux", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(os.ModeDir|os.FileMode(0500), e.Mode())
	ExpectLt(math.Abs(time.Since(e.ModTime()).Seconds()), 30)
	ExpectTrue(e.IsDir())
}

func (t *readOnlyTest) ContentsInSubDirectory_PlaceholderNotPresent() {
	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			[]*gcsutil.ObjectInfo{
				// File
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/foo",
					},
					Contents: "taco",
				},

				// Directory
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/bar/",
					},
				},

				// File
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/baz",
					},
					Contents: "burrito",
				},

				// File in sub-directory
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/qux/asdf",
					},
					Contents: "",
				},
			}))

	// ReadDir
	entries, err := ioutil.ReadDir(path.Join(t.mfs.Dir(), "dir"))
	AssertEq(nil, err)

	AssertEq(4, len(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	ExpectEq("bar", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(os.ModeDir|os.FileMode(0500), e.Mode())
	ExpectLt(math.Abs(time.Since(e.ModTime()).Seconds()), 30)
	ExpectTrue(e.IsDir())

	// baz
	e = entries[1]
	ExpectEq("baz", e.Name())
	ExpectEq(len("burrito"), e.Size())
	ExpectEq(os.FileMode(0400), e.Mode())
	ExpectLt(math.Abs(time.Since(e.ModTime()).Seconds()), 30)
	ExpectFalse(e.IsDir())

	// foo
	e = entries[2]
	ExpectEq("foo", e.Name())
	ExpectEq(len("taco"), e.Size())
	ExpectEq(os.FileMode(0400), e.Mode())
	ExpectLt(math.Abs(time.Since(e.ModTime()).Seconds()), 30)
	ExpectFalse(e.IsDir())

	// qux
	e = entries[3]
	ExpectEq("qux", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(os.ModeDir|os.FileMode(0500), e.Mode())
	ExpectLt(math.Abs(time.Since(e.ModTime()).Seconds()), 30)
	ExpectTrue(e.IsDir())
}

func (t *readOnlyTest) ListDirectoryTwice_NoChange() {
	// Set up initial contents.
	AssertEq(
		nil,
		t.createEmptyObjects([]string{
			"foo",
			"bar",
		}))

	// List once.
	entries, err := ioutil.ReadDir(t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(2, len(entries))
	ExpectEq("bar", entries[0].Name())
	ExpectEq("foo", entries[1].Name())

	// List again.
	entries, err = ioutil.ReadDir(t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(2, len(entries))
	ExpectEq("bar", entries[0].Name())
	ExpectEq("foo", entries[1].Name())
}

func (t *readOnlyTest) ListDirectoryTwice_Changed() {
	// Set up initial contents.
	AssertEq(
		nil,
		t.createEmptyObjects([]string{
			"foo",
			"bar",
		}))

	// List once.
	entries, err := ioutil.ReadDir(t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(2, len(entries))
	ExpectEq("bar", entries[0].Name())
	ExpectEq("foo", entries[1].Name())

	// Add "baz" and remove "bar".
	AssertEq(nil, t.bucket.DeleteObject(t.ctx, "bar"))
	AssertEq(nil, t.createEmptyObjects([]string{"baz"}))

	// List again.
	entries, err = ioutil.ReadDir(t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(2, len(entries))
	ExpectEq("baz", entries[0].Name())
	ExpectEq("foo", entries[1].Name())
}

func (t *readOnlyTest) Inodes() {
	// Set up two files and a directory placeholder.
	AssertEq(
		nil,
		t.createEmptyObjects([]string{
			"foo",
			"bar/",
			"baz",
		}))

	// List.
	entries, err := ioutil.ReadDir(t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(3, len(entries))

	// Confirm all of the inodes are distinct.
	inodesSeen := make(map[uint64]struct{})
	for _, fileInfo := range entries {
		stat := fileInfo.Sys().(*syscall.Stat_t)
		_, ok := inodesSeen[stat.Ino]
		AssertFalse(ok, "Duplicate inode: %v", fileInfo)

		inodesSeen[stat.Ino] = struct{}{}
	}
}

func (t *readOnlyTest) OpenNonExistentFile() {
	_, err := os.Open(path.Join(t.mfs.Dir(), "foo"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("foo")))
	ExpectThat(err, Error(HasSubstr("no such file")))
}
