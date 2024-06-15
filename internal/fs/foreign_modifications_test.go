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

package fs_test

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/inode"
	"github.com/jacobsa/fuse/fusetesting"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func setSymlinkTarget(
	ctx context.Context,
	bucket gcs.Bucket,
	objName string,
	target string) (err error) {
	_, err = bucket.UpdateObject(
		ctx,
		&gcs.UpdateObjectRequest{
			Name: objName,
			Metadata: map[string]*string{
				inode.SymlinkMetadataKey: &target,
			},
		})

	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ForeignModsTest struct {
	fsTest
}

func init() {
	RegisterTestSuite(&ForeignModsTest{})
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ForeignModsTest) StatRoot() {
	fi, err := os.Stat(mntDir)
	AssertEq(nil, err)

	ExpectEq(path.Base(mntDir), fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *ForeignModsTest) ReadDir_EmptyRoot() {
	// ReadDir
	entries, err := fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)

	ExpectThat(entries, ElementsAre())
}

func (t *ForeignModsTest) ReadDir_ContentsInRoot() {
	// Set up contents.
	createTime := mtimeClock.Now()
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"foo": "taco",

				// Directory
				"bar/": "",

				// File
				"baz": "burrito",
			}))

	/////////////////////////
	// ReadDir
	/////////////////////////

	entries, err := fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)

	AssertEq(3, len(entries), "Names: %v", getFileNames(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	ExpectEq("bar", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(dirPerms|os.ModeDir, e.Mode())
	ExpectTrue(e.IsDir())
	ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)

	// baz
	e = entries[1]
	ExpectEq("baz", e.Name())
	ExpectEq(len("burrito"), e.Size())
	ExpectEq(filePerms, e.Mode())
	ExpectThat(e, fusetesting.MtimeIsWithin(createTime, timeSlop))
	ExpectFalse(e.IsDir())
	ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)

	// foo
	e = entries[2]
	ExpectEq("foo", e.Name())
	ExpectEq(len("taco"), e.Size())
	ExpectEq(filePerms, e.Mode())
	ExpectThat(e, fusetesting.MtimeIsWithin(createTime, timeSlop))
	ExpectFalse(e.IsDir())
	ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)
}

func (t *ForeignModsTest) ReadDir_EmptySubDirectory() {
	// Set up an empty directory placeholder called 'bar'.
	AssertEq(nil, t.createEmptyObjects([]string{"bar/"}))

	// ReadDir
	entries, err := fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	entries, err = fusetesting.ReadDirPicky(path.Join(mntDir, "bar"))
	AssertEq(nil, err)

	ExpectThat(entries, ElementsAre())
}

func (t *ForeignModsTest) ReadDir_ContentsInSubDirectory() {
	// Set up contents.
	createTime := mtimeClock.Now()
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// Placeholder
				"dir/": "",

				// File
				"dir/foo": "taco",

				// Directory
				"dir/bar/": "",

				// File
				"dir/baz": "burrito",
			}))

	// Wait for the directory to show up in the file system.
	_, err := fusetesting.ReadDirPicky(path.Join(mntDir))
	AssertEq(nil, err)

	// ReadDir
	entries, err := fusetesting.ReadDirPicky(path.Join(mntDir, "dir"))
	AssertEq(nil, err)

	AssertEq(3, len(entries), "Names: %v", getFileNames(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	ExpectEq("bar", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(dirPerms|os.ModeDir, e.Mode())
	ExpectTrue(e.IsDir())
	ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)

	// baz
	e = entries[1]
	ExpectEq("baz", e.Name())
	ExpectEq(len("burrito"), e.Size())
	ExpectEq(filePerms, e.Mode())
	ExpectThat(e, fusetesting.MtimeIsWithin(createTime, timeSlop))
	ExpectFalse(e.IsDir())
	ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)

	// foo
	e = entries[2]
	ExpectEq("foo", e.Name())
	ExpectEq(len("taco"), e.Size())
	ExpectEq(filePerms, e.Mode())
	ExpectThat(e, fusetesting.MtimeIsWithin(createTime, timeSlop))
	ExpectFalse(e.IsDir())
	ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)
}

func (t *ForeignModsTest) UnreachableObjects() {
	var fi os.FileInfo
	var err error

	// Set up objects that appear to be directory contents, but for which there
	// is no directory placeholder object. We don't have implicit directories
	// enabled, so these should be unreachable.
	err = storageutil.CreateEmptyObjects(
		ctx,
		bucket,
		[]string{
			// Implicit directory contents, conflicting file name.
			"test",
			"test/0",
			"test/1",

			// Implicit directory contents, no conflicting file name.
			"bar/0/",
		})

	AssertEq(nil, err)

	// Only the conflicitng file name should show up in the root.
	entries, err := fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("test", fi.Name())
	ExpectEq(filePerms, fi.Mode())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting the conflicting name should give the file.
	fi, err = os.Stat(path.Join(mntDir, "test"))
	AssertEq(nil, err)

	ExpectEq("test", fi.Name())
	ExpectEq(filePerms, fi.Mode())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting the other name shouldn't work at all.
	_, err = os.Stat(path.Join(mntDir, "bar"))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)

	// These unreachable objects (test/0, test/1, bar/0) start showing up in
	// other tests as soon as directory with similar name is created. Hence
	// cleaning them.
	err = storageutil.DeleteAllObjects(ctx, bucket)
	AssertEq(nil, err)
}

func (t *ForeignModsTest) FileAndDirectoryWithConflictingName() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up an object named "foo" and one named "foo/", plus a child for the
	// latter.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"foo": "taco",

				// Directory
				"foo/": "",

				// Directory child
				"foo/bar": "burrito",
			}))

	// A listing of the parent should contain a directory named "foo" and a
	// file named "foo\n".
	entries, err = fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)
	AssertEq(2, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	fi = entries[1]
	ExpectEq("foo\n", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(filePerms, fi.Mode())
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting "foo" should yield the directory.
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the file.
	fi, err = os.Stat(path.Join(mntDir, "foo\n"))
	AssertEq(nil, err)

	ExpectEq("foo\n", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectFalse(fi.IsDir())

	// Listing the directory should yield the sole child file.
	entries, err = fusetesting.ReadDirPicky(path.Join(mntDir, "foo"))
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("bar", fi.Name())
	ExpectEq(len("burrito"), fi.Size())
	ExpectEq(filePerms, fi.Mode())
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *ForeignModsTest) SymlinkAndDirectoryWithConflictingName() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up an object named "foo" and one named "foo/", plus a child for the
	// latter.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// Symlink
				"foo": "",

				// Directory
				"foo/": "",

				// Directory child
				"foo/bar": "burrito",
			}))

	err = setSymlinkTarget(ctx, bucket, "foo", "")
	AssertEq(nil, err)

	// A listing of the parent should contain a directory named "foo" and a
	// symlink named "foo\n".
	entries, err = fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)
	AssertEq(2, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	fi = entries[1]
	ExpectEq("foo\n", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(filePerms|os.ModeSymlink, fi.Mode())
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting "foo" should yield the directory.
	fi, err = os.Lstat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the symlink.
	fi, err = os.Lstat(path.Join(mntDir, "foo\n"))
	AssertEq(nil, err)

	ExpectEq("foo\n", fi.Name())
	ExpectFalse(fi.IsDir())

	// Listing the directory should yield the sole child file.
	entries, err = fusetesting.ReadDirPicky(path.Join(mntDir, "foo"))
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("bar", fi.Name())
	ExpectEq(len("burrito"), fi.Size())
	ExpectEq(filePerms, fi.Mode())
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *ForeignModsTest) StatTrailingNewlineName_NoConflictingNames() {
	var err error

	// Set up an object named "foo".
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"foo": "taco",
			}))

	// We shouldn't be able to stat "foo\n", because there is no conflicting
	// directory name.
	_, err = os.Stat(path.Join(mntDir, "foo\n"))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *ForeignModsTest) Inodes() {
	// Set up two files and a directory placeholder.
	AssertEq(
		nil,
		t.createEmptyObjects([]string{
			"foo",
			"bar/",
			"baz",
		}))

	// List.
	entries, err := fusetesting.ReadDirPicky(mntDir)
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

func (t *ForeignModsTest) OpenNonExistentFile() {
	_, err := os.Open(path.Join(mntDir, "foo"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("foo")))
	ExpectThat(err, Error(HasSubstr("no such file")))
}

func (t *ForeignModsTest) ReadFromFile_Small() {
	const contents = "tacoburritoenchilada"

	// Create an object.
	AssertEq(nil, t.createWithContents("foo", contents))

	// Wait for it to show up in the file system.
	_, err := fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)

	// Attempt to open it.
	f, err := os.Open(path.Join(mntDir, "foo"))
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

func (t *ForeignModsTest) ReadFromFile_Large() {
	randSrc := rand.New(rand.NewSource(0xdeadbeef))

	// Create some random contents.
	const contentLen = 1 << 22
	contents := randBytes(contentLen)

	// Repeatedly:
	//
	//  *  Create an object with the random contents.
	//  *  Read a random range of it.
	//  *  Verify the result.
	//
	var buf [contentLen]byte
	runOnce := func() {
		// Create an object.
		_, err := storageutil.CreateObject(
			ctx,
			bucket,
			"foo",
			contents)

		AssertEq(nil, err)

		// Attempt to open it.
		f, err := os.Open(path.Join(mntDir, "foo"))
		AssertEq(nil, err)
		defer func() { AssertEq(nil, f.Close()) }()

		// Read part of it.
		offset := randSrc.Int63n(contentLen + 1)
		size := randSrc.Intn(int(contentLen - offset))

		n, err := f.ReadAt(buf[:size], offset)
		if offset+int64(size) == contentLen && err == io.EOF {
			err = nil
		}

		AssertEq(nil, err)
		AssertEq(size, n)
		AssertTrue(
			bytes.Equal(contents[offset:offset+int64(size)], buf[:n]),
			"offset: %d\n"+
				"size:   %d\n"+
				"n:      %d",
			offset,
			size,
			n)
	}

	start := time.Now()
	for time.Since(start) < 2*time.Second {
		runOnce()
	}
}

func (t *ForeignModsTest) ReadBeyondEndOfFile() {
	const contents = "tacoburritoenchilada"
	const contentLen = len(contents)

	// Create an object.
	AssertEq(nil, t.createWithContents("foo", contents))

	// Wait for it to show up in the file system.
	_, err := fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)

	// Attempt to open it.
	f, err := os.Open(path.Join(mntDir, "foo"))
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

func (t *ForeignModsTest) ObjectIsOverwritten_File() {
	// Create an object.
	AssertEq(nil, t.createWithContents("foo", "taco"))

	// Open the corresponding file for reading.
	f1, err := os.OpenFile(path.Join(mntDir, "foo"), os.O_RDONLY, 0)
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
	// NOTE: We must open with a different mode here than above to work
	// around the fact that osxfuse will re-use file handles. See the notes on
	// fuse.FileSystem.OpenFile for more.
	f2, err := os.OpenFile(path.Join(mntDir, "foo"), os.O_RDWR, 0)
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

func (t *ForeignModsTest) ObjectIsOverwritten_Directory() {
	var err error

	// Create a directory placeholder.
	AssertEq(nil, t.createWithContents("dir/", ""))

	// Open the corresponding inode.
	t.f1, err = os.Open(path.Join(mntDir, "dir"))
	AssertEq(nil, err)

	// Overwrite the object.
	AssertEq(nil, t.createWithContents("dir/", ""))

	// The inode should still be accessible.
	fi, err := t.f1.Stat()

	AssertEq(nil, err)
	ExpectEq("dir", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ForeignModsTest) ObjectMetadataChanged_File() {
	// Create an object.
	AssertEq(nil, t.createWithContents("foo", "taco"))

	// Open the corresponding file for reading.
	f1, err := os.OpenFile(path.Join(mntDir, "foo"), os.O_RDONLY, 0)
	AssertEq(nil, err)
	defer func() {
		ExpectEq(nil, f1.Close())
	}()

	// Make sure that the contents are cached locally.
	_, err = f1.ReadAt(make([]byte, 1), 0)
	AssertEq(nil, err)

	// Change the object's metadata, causing a new generation.
	lang := "fr"
	_, err = bucket.UpdateObject(
		ctx,
		&gcs.UpdateObjectRequest{
			Name:            "foo",
			ContentLanguage: &lang,
		})

	AssertEq(nil, err)

	// The file should appear to be unlinked, but with the previous contents.
	fi, err := f1.Stat()

	AssertEq(nil, err)
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *ForeignModsTest) ObjectMetadataChanged_Directory() {
	var err error

	// Create a directory placeholder.
	AssertEq(nil, t.createWithContents("dir/", ""))

	// Open the corresponding inode.
	t.f1, err = os.Open(path.Join(mntDir, "dir"))
	AssertEq(nil, err)

	// Change the object's metadata, causing a new generation.
	lang := "fr"
	_, err = bucket.UpdateObject(
		ctx,
		&gcs.UpdateObjectRequest{
			Name:            "dir/",
			ContentLanguage: &lang,
		})

	AssertEq(nil, err)

	// The inode should still be accessible.
	fi, err := t.f1.Stat()

	AssertEq(nil, err)
	ExpectEq("dir", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ForeignModsTest) ObjectIsDeleted_File() {
	// Create an object.
	AssertEq(nil, t.createWithContents("foo", "taco"))

	// Open the corresponding file for reading.
	f1, err := os.Open(path.Join(mntDir, "foo"))
	defer func() {
		if f1 != nil {
			ExpectEq(nil, f1.Close())
		}
	}()

	AssertEq(nil, err)

	// Delete the object.
	AssertEq(
		nil,
		bucket.DeleteObject(
			ctx,
			&gcs.DeleteObjectRequest{Name: "foo"}))

	// The file should appear to be unlinked, but with the previous contents.
	fi, err := f1.Stat()

	AssertEq(nil, err)
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)

	// Opening again should not work.
	f2, err := os.Open(path.Join(mntDir, "foo"))
	defer func() {
		if f2 != nil {
			ExpectEq(nil, f2.Close())
		}
	}()

	AssertNe(nil, err)
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *ForeignModsTest) ObjectIsDeleted_Directory() {
	var err error

	// Create a directory placeholder.
	AssertEq(nil, t.createWithContents("dir/", ""))

	// Open the corresponding inode.
	t.f1, err = os.Open(path.Join(mntDir, "dir"))
	AssertEq(nil, err)

	// Delete the object.
	AssertEq(
		nil,
		bucket.DeleteObject(
			ctx,
			&gcs.DeleteObjectRequest{Name: "dir/"}))

	// The inode should still be fstat'able.
	fi, err := t.f1.Stat()

	AssertEq(nil, err)
	ExpectEq("dir", fi.Name())
	ExpectTrue(fi.IsDir())

	// Opening again should not work.
	t.f2, err = os.Open(path.Join(mntDir, "dir"))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *ForeignModsTest) Mtime() {
	var err error

	// Create an object that has an mtime set.
	expected := time.Date(2001, 2, 3, 4, 5, 6, 7, time.Local)
	req := &gcs.CreateObjectRequest{
		Name: "foo",
		Metadata: map[string]string{
			"gcsfuse_mtime": expected.UTC().Format(time.RFC3339Nano),
		},
		Contents: ioutil.NopCloser(strings.NewReader("")),
	}

	_, err = bucket.CreateObject(ctx, req)
	AssertEq(nil, err)

	// Stat the file.
	fi, err := os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectThat(fi.ModTime(), timeutil.TimeEq(expected))
}

func (t *ForeignModsTest) RemoteMtimeChange() {
	var err error

	// Create an object that has an mtime set.
	_, err = bucket.CreateObject(
		ctx,
		&gcs.CreateObjectRequest{
			Name: "foo",
			Metadata: map[string]string{
				"gcsfuse_mtime": time.Now().UTC().Format(time.RFC3339Nano),
			},
			Contents: ioutil.NopCloser(strings.NewReader("")),
		})

	AssertEq(nil, err)

	// Stat the object so that the file system assigns it an inode.
	_, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	// Update the mtime.
	expected := time.Date(2001, 2, 3, 4, 5, 6, 7, time.Local)
	formatted := expected.UTC().Format(time.RFC3339Nano)

	_, err = bucket.UpdateObject(
		ctx,
		&gcs.UpdateObjectRequest{
			Name: "foo",
			Metadata: map[string]*string{
				"gcsfuse_mtime": &formatted,
			},
		})

	AssertEq(nil, err)

	// Stat the file again. We should see the new mtime.
	fi, err := os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectThat(fi.ModTime(), timeutil.TimeEq(expected))
}

func (t *ForeignModsTest) Symlink() {
	var err error

	// Create an object that looks like a symlink.
	req := &gcs.CreateObjectRequest{
		Name: "foo",
		Metadata: map[string]string{
			"gcsfuse_symlink_target": "bar/baz",
		},
		Contents: ioutil.NopCloser(strings.NewReader("")),
	}

	_, err = bucket.CreateObject(ctx, req)
	AssertEq(nil, err)

	// Stat the link.
	fi, err := os.Lstat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(filePerms|os.ModeSymlink, fi.Mode())

	// Read the link.
	target, err := os.Readlink(path.Join(mntDir, "foo"))
	AssertEq(nil, err)
	ExpectEq("bar/baz", target)
}
