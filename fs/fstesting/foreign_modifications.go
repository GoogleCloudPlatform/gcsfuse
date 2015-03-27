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

// A collection of tests for a file system where we do not attempt to write to
// the file system at all. Rather we set up contents in a GCS bucket out of
// band, wait for them to be available, and then read them via the file system.
//
// These tests are registered by RegisterFSTests.

package fstesting

import (
	"encoding/hex"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"google.golang.org/cloud/storage"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type foreignModsTest struct {
	fsTest
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *foreignModsTest) StatRoot() {
	fi, err := os.Stat(t.mfs.Dir())
	AssertEq(nil, err)

	ExpectEq(path.Base(t.mfs.Dir()), fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(0700|os.ModeDir, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *foreignModsTest) ReadDir_EmptyRoot() {
	// ReadDir
	entries, err := t.readDirUntil(0, t.mfs.Dir())
	AssertEq(nil, err)

	ExpectThat(entries, ElementsAre())
}

func (t *foreignModsTest) ReadDir_ContentsInRoot() {
	// Set up contents.
	createTime := t.clock.Now()
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

	// Make sure the time below doesn't match.
	t.advanceTime()

	/////////////////////////
	// ReadDir
	/////////////////////////

	entries, err := t.readDirUntil(3, t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(3, len(entries), "Names: %v", getFileNames(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	ExpectEq("bar", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(0700|os.ModeDir, e.Mode())
	ExpectTrue(e.IsDir())
	ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)

	// baz
	e = entries[1]
	ExpectEq("baz", e.Name())
	ExpectEq(len("burrito"), e.Size())
	ExpectEq(os.FileMode(0700), e.Mode())
	ExpectThat(e.ModTime(), t.matchesStartTime(createTime))
	ExpectFalse(e.IsDir())
	ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)

	// foo
	e = entries[2]
	ExpectEq("foo", e.Name())
	ExpectEq(len("taco"), e.Size())
	ExpectEq(os.FileMode(0700), e.Mode())
	ExpectThat(e.ModTime(), t.matchesStartTime(createTime))
	ExpectFalse(e.IsDir())
	ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)
}

func (t *foreignModsTest) ReadDir_EmptySubDirectory() {
	// Set up an empty directory placeholder called 'bar'.
	AssertEq(nil, t.createEmptyObjects([]string{"bar/"}))

	// ReadDir
	entries, err := t.readDirUntil(1, t.mfs.Dir())
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	entries, err = t.readDirUntil(0, path.Join(t.mfs.Dir(), "bar"))
	AssertEq(nil, err)

	ExpectThat(entries, ElementsAre())
}

func (t *foreignModsTest) ReadDir_ContentsInSubDirectory() {
	// Set up contents.
	createTime := t.clock.Now()
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
			}))

	// Make sure the time below doesn't match.
	t.advanceTime()

	// Wait for the directory to show up in the file system.
	_, err := t.readDirUntil(1, path.Join(t.mfs.Dir()))
	AssertEq(nil, err)

	// ReadDir
	entries, err := t.readDirUntil(3, path.Join(t.mfs.Dir(), "dir"))
	AssertEq(nil, err)

	AssertEq(3, len(entries), "Names: %v", getFileNames(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	ExpectEq("bar", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(0700|os.ModeDir, e.Mode())
	ExpectTrue(e.IsDir())
	ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)

	// baz
	e = entries[1]
	ExpectEq("baz", e.Name())
	ExpectEq(len("burrito"), e.Size())
	ExpectEq(os.FileMode(0700), e.Mode())
	ExpectThat(e.ModTime(), t.matchesStartTime(createTime))
	ExpectFalse(e.IsDir())
	ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)

	// foo
	e = entries[2]
	ExpectEq("foo", e.Name())
	ExpectEq(len("taco"), e.Size())
	ExpectEq(os.FileMode(0700), e.Mode())
	ExpectThat(e.ModTime(), t.matchesStartTime(createTime))
	ExpectFalse(e.IsDir())
	ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)
}

func (t *foreignModsTest) UnreachableObjects() {
	// Set up objects that appear to be directory contents, but for which there
	// is no directory placeholder object. We don't have implicit directories
	// enabled, so these should be unreachable.
	_, err := gcsutil.CreateEmptyObjects(
		t.ctx,
		t.bucket,
		[]string{
			"foo/0",
			"foo/1",
			"bar/0/",
		})

	AssertEq(nil, err)

	// Nothing should show up in the root.
	_, err = t.readDirUntil(0, path.Join(t.mfs.Dir()))
	AssertEq(nil, err)

	// Statting the directories shouldn't work.
	_, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))

	AssertNe(nil, err)
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *foreignModsTest) FileAndDirectoryWithConflictingName() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up an object named "foo" and one named "foo/", plus a child for the
	// latter.
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
						Name: "foo/",
					},
				},

				// Directory child
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "foo/bar",
					},
					Contents: "",
				},
			}))

	// Statting "foo" should yield the directory.
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Listing the directory will result in two copies of the directory.
	//
	// This behavior is a bug.
	// Cf. https://github.com/GoogleCloudPlatform/gcsfuse/issues/28
	entries, err = ioutil.ReadDir(t.mfs.Dir())
	AssertEq(nil, err)
	AssertEq(2, len(entries))

	ExpectEq("foo", entries[0].Name())
	ExpectEq("foo", entries[1].Name())

	ExpectEq(0, entries[0].Size())
	ExpectEq(0, entries[1].Size())

	ExpectTrue(entries[0].IsDir())
	ExpectTrue(entries[1].IsDir())
}

func (t *foreignModsTest) Inodes() {
	// Set up two files and a directory placeholder.
	AssertEq(
		nil,
		t.createEmptyObjects([]string{
			"foo",
			"bar/",
			"baz",
		}))

	// List.
	entries, err := t.readDirUntil(3, t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(3, len(entries), "Names: %v", getFileNames(entries))

	// Confirm all of the inodes are distinct.
	inodesSeen := make(map[uint64]struct{})
	for _, fileInfo := range entries {
		stat := fileInfo.Sys().(*syscall.Stat_t)
		_, ok := inodesSeen[stat.Ino]
		AssertFalse(
			ok,
			"Duplicate inode (%v). File info: %v",
			stat.Ino,
			fileInfo)

		inodesSeen[stat.Ino] = struct{}{}
	}
}

func (t *foreignModsTest) OpenNonExistentFile() {
	_, err := os.Open(path.Join(t.mfs.Dir(), "foo"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("foo")))
	ExpectThat(err, Error(HasSubstr("no such file")))
}

func (t *foreignModsTest) ReadFromFile_Small() {
	const contents = "tacoburritoenchilada"
	const contentLen = len(contents)

	// Create an object.
	AssertEq(nil, t.createWithContents("foo", contents))

	// Wait for it to show up in the file system.
	_, err := t.readDirUntil(1, t.mfs.Dir())
	AssertEq(nil, err)

	// Attempt to open it.
	f, err := os.Open(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)
	defer func() { AssertEq(nil, f.Close()) }()

	// Read its entire contents.
	slice, err := ioutil.ReadAll(f)
	AssertEq(nil, err)
	ExpectEq("tacoburritoenchilada", string(slice))

	// Read various ranges of it.
	var s string

	s, err = readRange(f, int64(len("taco")), len("burrito"))
	AssertEq(nil, err)
	ExpectEq("burrito", s)

	s, err = readRange(f, 0, len("taco"))
	AssertEq(nil, err)
	ExpectEq("taco", s)

	s, err = readRange(f, int64(len("tacoburrito")), len("enchilada"))
	AssertEq(nil, err)
	ExpectEq("enchilada", s)
}

func (t *foreignModsTest) ReadFromFile_Large() {
	const contentLen = 1 << 20
	contents := randString(contentLen)

	// Create an object.
	AssertEq(nil, t.createWithContents("foo", contents))

	// Wait for it to show up in the file system.
	_, err := t.readDirUntil(1, t.mfs.Dir())
	AssertEq(nil, err)

	// Attempt to open it.
	f, err := os.Open(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)
	defer func() { AssertEq(nil, f.Close()) }()

	// Read its entire contents.
	slice, err := ioutil.ReadAll(f)
	AssertEq(nil, err)
	if contents != string(slice) {
		ExpectTrue(
			false,
			"Expected:\n%v\n\nActual:\n%v",
			hex.Dump([]byte(contents)),
			hex.Dump(slice))
	}

	// Read from parts of it.
	referenceReader := strings.NewReader(contents)
	for trial := 0; trial < 1000; trial++ {
		offset := rand.Int63n(contentLen + 1)
		size := rand.Intn(int(contentLen - offset))

		expected, err := readRange(referenceReader, offset, size)
		AssertEq(nil, err)

		actual, err := readRange(f, offset, size)
		AssertEq(nil, err)

		if expected != actual {
			AssertTrue(
				expected == actual,
				"Expected:\n%s\nActual:\n%s",
				hex.Dump([]byte(expected)),
				hex.Dump([]byte(actual)))
		}
	}
}

func (t *foreignModsTest) ReadBeyondEndOfFile() {
	const contents = "tacoburritoenchilada"
	const contentLen = len(contents)

	// Create an object.
	AssertEq(nil, t.createWithContents("foo", contents))

	// Wait for it to show up in the file system.
	_, err := t.readDirUntil(1, t.mfs.Dir())
	AssertEq(nil, err)

	// Attempt to open it.
	f, err := os.Open(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)
	defer func() { AssertEq(nil, f.Close()) }()

	// Attempt to read beyond the end of the file.
	_, err = f.Seek(int64(contentLen-1), 0)
	AssertEq(nil, err)

	buf := make([]byte, 2)
	n, err := f.Read(buf)
	AssertEq(1, n, "err: %v", err)
	AssertEq(contents[contentLen-1], buf[0])

	if err == nil {
		n, err = f.Read(buf)
		AssertEq(0, n)
	}
}

func (t *foreignModsTest) ObjectIsOverwritten_File() {
	// Create an object.
	AssertEq(nil, t.createWithContents("foo", "taco"))

	// Open the corresponding file for reading.
	f1, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDONLY, 0)
	AssertEq(nil, err)
	defer func() {
		ExpectEq(nil, f1.Close())
	}()

	// Make sure that the contents are cached locally.
	_, err = f1.ReadAt(make([]byte, 1), 0)
	AssertEq(nil, err)

	// Overwrite the object.
	AssertEq(nil, t.createWithContents("foo", "burrito"))

	// The file should appear to be unlinked, but with the previous contents.
	fi, err := f1.Stat()

	AssertEq(nil, err)
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)

	// Opening again should yield the new version.
	//
	// NOTE(jacobsa): We must open with a different mode here than above to work
	// around the fact that osxfuse will re-use file handles. See the notes on
	// fuse.FileSystem.OpenFile for more.
	f2, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR, 0)
	AssertEq(nil, err)
	defer func() {
		ExpectEq(nil, f2.Close())
	}()

	fi, err = f2.Stat()
	AssertEq(nil, err)
	ExpectEq(len("burrito"), fi.Size())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Reading from the old file handle should give the old data.
	contents, err := ioutil.ReadAll(f1)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	contents, err = ioutil.ReadAll(f2)
	AssertEq(nil, err)
	ExpectEq("burrito", string(contents))
}

func (t *foreignModsTest) ObjectIsOverwritten_Directory() {
	var err error

	// Create a directory placeholder.
	AssertEq(nil, t.createWithContents("dir/", ""))

	// Open the corresponding inode.
	t.f1, err = os.Open(path.Join(t.mfs.Dir(), "dir"))
	AssertEq(nil, err)

	// Overwrite the object.
	AssertEq(nil, t.createWithContents("dir/", ""))

	// The inode should appear to be unlinked, but otherwise be okay.
	fi, err := t.f1.Stat()

	AssertEq(nil, err)
	ExpectTrue(fi.IsDir())
	ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *foreignModsTest) ObjectIsDeleted_File() {
	// Create an object.
	AssertEq(nil, t.createWithContents("foo", "taco"))

	// Open the corresponding file for reading.
	f1, err := os.Open(path.Join(t.mfs.Dir(), "foo"))
	defer func() {
		if f1 != nil {
			ExpectEq(nil, f1.Close())
		}
	}()

	AssertEq(nil, err)

	// Delete the object.
	AssertEq(nil, t.bucket.DeleteObject(t.ctx, "foo"))

	// The file should appear to be unlinked, but with the previous contents.
	fi, err := f1.Stat()

	AssertEq(nil, err)
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)

	// Opening again should not work.
	f2, err := os.Open(path.Join(t.mfs.Dir(), "foo"))
	defer func() {
		if f2 != nil {
			ExpectEq(nil, f2.Close())
		}
	}()

	AssertNe(nil, err)
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *foreignModsTest) ObjectIsDeleted_Directory() {
	var err error

	// Create a directory placeholder.
	AssertEq(nil, t.createWithContents("dir/", ""))

	// Open the corresponding inode.
	t.f1, err = os.Open(path.Join(t.mfs.Dir(), "dir"))
	AssertEq(nil, err)

	// Delete the object.
	AssertEq(nil, t.bucket.DeleteObject(t.ctx, "dir/"))

	// The inode should appear to be unlinked, but otherwise be okay.
	fi, err := t.f1.Stat()

	AssertEq(nil, err)
	ExpectTrue(fi.IsDir())
	ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)

	// Opening again should not work.
	t.f2, err = os.Open(path.Join(t.mfs.Dir(), "dir"))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

////////////////////////////////////////////////////////////////////////
// Implicit directories
////////////////////////////////////////////////////////////////////////

type implicitDirsTest struct {
	fsTest
}

func (t *implicitDirsTest) setUpFSTest(cfg FSTestConfig) {
	cfg.ServerConfig.ImplicitDirectories = true
	t.fsTest.setUpFSTest(cfg)
}

func (t *implicitDirsTest) NothingPresent() {
	// ReadDir
	entries, err := t.readDirUntil(0, t.mfs.Dir())
	AssertEq(nil, err)

	ExpectThat(entries, ElementsAre())
}

func (t *implicitDirsTest) FileObjectPresent() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

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
			}))

	// Statting the name should return an entry for the file.
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectEq(4, fi.Size())
	ExpectFalse(fi.IsDir())

	// ReadDir should show the file.
	entries, err = t.readDirUntil(1, t.mfs.Dir())
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectEq(4, fi.Size())
	ExpectFalse(fi.IsDir())
}

func (t *implicitDirsTest) DirectoryObjectPresent() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			[]*gcsutil.ObjectInfo{
				// Directory
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "foo/",
					},
				},
			}))

	// Statting the name should return an entry for the directory.
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = t.readDirUntil(1, t.mfs.Dir())
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *implicitDirsTest) ImplicitDirectory_DefinedByFile() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			[]*gcsutil.ObjectInfo{
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "foo/bar",
					},
				},
			}))

	// Statting the name should return an entry for the directory.
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = t.readDirUntil(1, t.mfs.Dir())
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *implicitDirsTest) ImplicitDirectory_DefinedByDirectory() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			[]*gcsutil.ObjectInfo{
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "foo/bar/",
					},
				},
			}))

	// Statting the name should return an entry for the directory.
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = t.readDirUntil(1, t.mfs.Dir())
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *implicitDirsTest) ConflictingNames_PlaceholderPresent() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

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
						Name: "foo/",
					},
				},
			}))

	// Statting the name should return an entry for the directory.
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// ReadDir shows two copies of the directory.
	//
	// This behavior is a bug.
	// Cf. https://github.com/GoogleCloudPlatform/gcsfuse/issues/28
	entries, err = t.readDirUntil(2, t.mfs.Dir())
	AssertEq(nil, err)
	AssertEq(2, len(entries))

	ExpectEq("foo", entries[0].Name())
	ExpectEq("foo", entries[1].Name())

	ExpectEq(0, entries[0].Size())
	ExpectEq(0, entries[1].Size())

	ExpectTrue(entries[0].IsDir())
	ExpectTrue(entries[1].IsDir())
}

func (t *implicitDirsTest) ConflictingNames_PlaceholderNotPresent() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

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
						Name: "foo/bar",
					},
				},
			}))

	// Statting the name should return an entry for the directory.
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// ReadDir shows two copies of the directory.
	//
	// This behavior is a bug.
	// Cf. https://github.com/GoogleCloudPlatform/gcsfuse/issues/28
	entries, err = t.readDirUntil(2, t.mfs.Dir())
	AssertEq(nil, err)
	AssertEq(2, len(entries))

	ExpectEq("foo", entries[0].Name())
	ExpectEq("foo", entries[1].Name())

	ExpectEq(0, entries[0].Size())
	ExpectEq(0, entries[1].Size())

	ExpectTrue(entries[0].IsDir())
	ExpectTrue(entries[1].IsDir())
}

func (t *implicitDirsTest) StatUnknownName_NoOtherContents() {
	var err error

	// Stat an unknown name.
	_, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *implicitDirsTest) StatUnknownName_UnrelatedContents() {
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			[]*gcsutil.ObjectInfo{
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "bar",
					},
				},

				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "baz",
					},
				},
			}))

	// Stat an unknown name.
	_, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *implicitDirsTest) StatUnknownName_PrefixOfActualNames() {
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			[]*gcsutil.ObjectInfo{
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "foop",
					},
				},

				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "fooq/",
					},
				},
			}))

	// Stat an unknown name.
	_, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}
