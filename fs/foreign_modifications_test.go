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
	"encoding/hex"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/fs/inode"
	"github.com/jacobsa/fuse/fusetesting"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/ogletest"
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

func init() { ogletest.RegisterTestSuite(&ForeignModsTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *ForeignModsTest) StatRoot(t *ogletest.T) {
	fi, err := os.Stat(s.mfs.Dir())
	t.AssertEq(nil, err)

	t.ExpectEq(path.Base(s.mfs.Dir()), fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	t.ExpectTrue(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)
}

func (s *ForeignModsTest) ReadDir_EmptyRoot(t *ogletest.T) {
	// ReadDir
	entries, err := fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)

	t.ExpectThat(entries, ElementsAre())
}

func (s *ForeignModsTest) ReadDir_ContentsInRoot(t *ogletest.T) {
	// Set up contents.
	createTime := s.clock.Now()
	t.AssertEq(
		nil,
		s.createObjects(
			map[string]string{
				// File
				"foo": "taco",

				// Directory
				"bar/": "",

				// File
				"baz": "burrito",
			}))

	// Make sure the time below doesn's match.
	s.clock.AdvanceTime(time.Second)

	/////////////////////////
	// ReadDir
	/////////////////////////

	entries, err := fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)

	t.AssertEq(3, len(entries), "Names: %v", getFileNames(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	t.ExpectEq("bar", e.Name())
	t.ExpectEq(0, e.Size())
	t.ExpectEq(dirPerms|os.ModeDir, e.Mode())
	t.ExpectTrue(e.IsDir())
	t.ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)

	// baz
	e = entries[1]
	t.ExpectEq("baz", e.Name())
	t.ExpectEq(len("burrito"), e.Size())
	t.ExpectEq(filePerms, e.Mode())
	t.ExpectThat(e.ModTime(), timeutil.TimeEq(createTime))
	t.ExpectFalse(e.IsDir())
	t.ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)

	// foo
	e = entries[2]
	t.ExpectEq("foo", e.Name())
	t.ExpectEq(len("taco"), e.Size())
	t.ExpectEq(filePerms, e.Mode())
	t.ExpectThat(e.ModTime(), timeutil.TimeEq(createTime))
	t.ExpectFalse(e.IsDir())
	t.ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)
}

func (s *ForeignModsTest) ReadDir_EmptySubDirectory(t *ogletest.T) {
	// Set up an empty directory placeholder called 'bar'.
	t.AssertEq(nil, s.createEmptyObjects([]string{"bar/"}))

	// ReadDir
	entries, err := fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)
	t.AssertEq(1, len(entries))

	entries, err = fusetesting.ReadDirPicky(path.Join(s.mfs.Dir(), "bar"))
	t.AssertEq(nil, err)

	t.ExpectThat(entries, ElementsAre())
}

func (s *ForeignModsTest) ReadDir_ContentsInSubDirectory(t *ogletest.T) {
	// Set up contents.
	createTime := s.clock.Now()
	t.AssertEq(
		nil,
		s.createObjects(
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

	// Make sure the time below doesn's match.
	s.clock.AdvanceTime(time.Second)

	// Wait for the directory to show up in the file system.
	_, err := fusetesting.ReadDirPicky(path.Join(s.mfs.Dir()))
	t.AssertEq(nil, err)

	// ReadDir
	entries, err := fusetesting.ReadDirPicky(path.Join(s.mfs.Dir(), "dir"))
	t.AssertEq(nil, err)

	t.AssertEq(3, len(entries), "Names: %v", getFileNames(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	t.ExpectEq("bar", e.Name())
	t.ExpectEq(0, e.Size())
	t.ExpectEq(dirPerms|os.ModeDir, e.Mode())
	t.ExpectTrue(e.IsDir())
	t.ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)

	// baz
	e = entries[1]
	t.ExpectEq("baz", e.Name())
	t.ExpectEq(len("burrito"), e.Size())
	t.ExpectEq(filePerms, e.Mode())
	t.ExpectThat(e.ModTime(), timeutil.TimeEq(createTime))
	t.ExpectFalse(e.IsDir())
	t.ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)

	// foo
	e = entries[2]
	t.ExpectEq("foo", e.Name())
	t.ExpectEq(len("taco"), e.Size())
	t.ExpectEq(filePerms, e.Mode())
	t.ExpectThat(e.ModTime(), timeutil.TimeEq(createTime))
	t.ExpectFalse(e.IsDir())
	t.ExpectEq(1, e.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(), e.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(), e.Sys().(*syscall.Stat_t).Gid)
}

func (s *ForeignModsTest) UnreachableObjects(t *ogletest.T) {
	var fi os.FileInfo
	var err error

	// Set up objects that appear to be directory contents, but for which there
	// is no directory placeholder object. We don's have implicit directories
	// enabled, so these should be unreachable.
	err = gcsutil.CreateEmptyObjects(
		s.ctx,
		s.bucket,
		[]string{
			// Implicit directory contents, conflicting file name.
			"foo",
			"foo/0",
			"foo/1",

			// Implicit directory contents, no conflicting file name.
			"bar/0/",
		})

	t.AssertEq(nil, err)

	// Only the conflicitng file name should show up in the root.
	entries, err := fusetesting.ReadDirPicky(s.Dir)
	t.AssertEq(nil, err)
	t.AssertEq(1, len(entries))

	fi = entries[0]
	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(filePerms, fi.Mode())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting the conflicting name should give the file.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(filePerms, fi.Mode())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting the other name shouldn's work at all.
	_, err = os.Stat(path.Join(s.mfs.Dir(), "bar"))
	t.ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (s *ForeignModsTest) FileAndDirectoryWithConflictingName(t *ogletest.T) {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up an object named "foo" and one named "foo/", plus a child for the
	// latter.
	t.AssertEq(
		nil,
		s.createObjects(
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
	entries, err = fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)
	t.AssertEq(2, len(entries))

	fi = entries[0]
	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	t.ExpectTrue(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	fi = entries[1]
	t.ExpectEq("foo\n", fi.Name())
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectEq(filePerms, fi.Mode())
	t.ExpectFalse(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting "foo" should yield the directory.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the file.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo\n"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo\n", fi.Name())
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectFalse(fi.IsDir())

	// Listing the directory should yield the sole child file.
	entries, err = fusetesting.ReadDirPicky(path.Join(s.Dir, "foo"))
	t.AssertEq(nil, err)
	t.AssertEq(1, len(entries))

	fi = entries[0]
	t.ExpectEq("bar", fi.Name())
	t.ExpectEq(len("burrito"), fi.Size())
	t.ExpectEq(filePerms, fi.Mode())
	t.ExpectFalse(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (s *ForeignModsTest) SymlinkAndDirectoryWithConflictingName(t *ogletest.T) {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up an object named "foo" and one named "foo/", plus a child for the
	// latter.
	t.AssertEq(
		nil,
		s.createObjects(
			map[string]string{
				// Symlink
				"foo": "",

				// Directory
				"foo/": "",

				// Directory child
				"foo/bar": "burrito",
			}))

	err = setSymlinkTarget(s.ctx, s.bucket, "foo", "")
	t.AssertEq(nil, err)

	// A listing of the parent should contain a directory named "foo" and a
	// symlink named "foo\n".
	entries, err = fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)
	t.AssertEq(2, len(entries))

	fi = entries[0]
	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	t.ExpectTrue(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	fi = entries[1]
	t.ExpectEq("foo\n", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(filePerms|os.ModeSymlink, fi.Mode())
	t.ExpectFalse(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting "foo" should yield the directory.
	fi, err = os.Lstat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the symlink.
	fi, err = os.Lstat(path.Join(s.mfs.Dir(), "foo\n"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo\n", fi.Name())
	t.ExpectFalse(fi.IsDir())

	// Listing the directory should yield the sole child file.
	entries, err = fusetesting.ReadDirPicky(path.Join(s.Dir, "foo"))
	t.AssertEq(nil, err)
	t.AssertEq(1, len(entries))

	fi = entries[0]
	t.ExpectEq("bar", fi.Name())
	t.ExpectEq(len("burrito"), fi.Size())
	t.ExpectEq(filePerms, fi.Mode())
	t.ExpectFalse(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (s *ForeignModsTest) StatTrailingNewlineName_NoConflictingNames(t *ogletest.T) {
	var err error

	// Set up an object named "foo".
	t.AssertEq(
		nil,
		s.createObjects(
			map[string]string{
				"foo": "taco",
			}))

	// We shouldn's be able to stat "foo\n", because there is no conflicting
	// directory name.
	_, err = os.Stat(path.Join(s.mfs.Dir(), "foo\n"))
	t.ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (s *ForeignModsTest) Inodes(t *ogletest.T) {
	// Set up two files and a directory placeholder.
	t.AssertEq(
		nil,
		s.createEmptyObjects([]string{
			"foo",
			"bar/",
			"baz",
		}))

	// List.
	entries, err := fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)

	t.AssertEq(3, len(entries), "Names: %v", getFileNames(entries))

	// Confirm all of the inodes are distinct.
	inodesSeen := make(map[uint64]struct{})
	for _, fileInfo := range entries {
		stat := fileInfo.Sys().(*syscall.Stat_t)
		_, ok := inodesSeen[stat.Ino]
		t.AssertFalse(
			ok,
			"Duplicate inode (%v). File info: %v",
			stat.Ino,
			fileInfo)

		inodesSeen[stat.Ino] = struct{}{}
	}
}

func (s *ForeignModsTest) OpenNonExistentFile(t *ogletest.T) {
	_, err := os.Open(path.Join(s.mfs.Dir(), "foo"))

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("foo")))
	t.ExpectThat(err, Error(HasSubstr("no such file")))
}

func (s *ForeignModsTest) ReadFromFile_Small(t *ogletest.T) {
	const contents = "tacoburritoenchilada"
	const contentLen = len(contents)

	// Create an object.
	t.AssertEq(nil, s.createWithContents("foo", contents))

	// Wait for it to show up in the file system.
	_, err := fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)

	// Attempt to open it.
	f, err := os.Open(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)
	defer func() { t.AssertEq(nil, f.Close()) }()

	// Read its entire contents.
	slice, err := ioutil.ReadAll(f)
	t.AssertEq(nil, err)
	t.ExpectEq("tacoburritoenchilada", string(slice))

	// Read various ranges of it.
	var c string

	c, err = readRange(f, int64(len("taco")), len("burrito"))
	t.AssertEq(nil, err)
	t.ExpectEq("burrito", c)

	c, err = readRange(f, 0, len("taco"))
	t.AssertEq(nil, err)
	t.ExpectEq("taco", c)

	c, err = readRange(f, int64(len("tacoburrito")), len("enchilada"))
	t.AssertEq(nil, err)
	t.ExpectEq("enchilada", c)
}

func (s *ForeignModsTest) ReadFromFile_Large(t *ogletest.T) {
	const contentLen = 1 << 20
	contents := randString(contentLen)

	// Create an object.
	t.AssertEq(nil, s.createWithContents("foo", contents))

	// Wait for it to show up in the file system.
	_, err := fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)

	// Attempt to open it.
	f, err := os.Open(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)
	defer func() { t.AssertEq(nil, f.Close()) }()

	// Read its entire contents.
	slice, err := ioutil.ReadAll(f)
	t.AssertEq(nil, err)
	if contents != string(slice) {
		t.ExpectTrue(
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
		t.AssertEq(nil, err)

		actual, err := readRange(f, offset, size)
		t.AssertEq(nil, err)

		if expected != actual {
			t.AssertTrue(
				expected == actual,
				"Expected:\n%s\nActual:\n%s",
				hex.Dump([]byte(expected)),
				hex.Dump([]byte(actual)))
		}
	}
}

func (s *ForeignModsTest) ReadBeyondEndOfFile(t *ogletest.T) {
	const contents = "tacoburritoenchilada"
	const contentLen = len(contents)

	// Create an object.
	t.AssertEq(nil, s.createWithContents("foo", contents))

	// Wait for it to show up in the file system.
	_, err := fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)

	// Attempt to open it.
	f, err := os.Open(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)
	defer func() { t.AssertEq(nil, f.Close()) }()

	// Attempt to read beyond the end of the file.
	_, err = f.Seek(int64(contentLen-1), 0)
	t.AssertEq(nil, err)

	buf := make([]byte, 2)
	n, err := f.Read(buf)
	t.AssertEq(1, n, "err: %v", err)
	t.AssertEq(contents[contentLen-1], buf[0])

	if err == nil {
		n, err = f.Read(buf)
		t.AssertEq(0, n)
	}
}

func (s *ForeignModsTest) ObjectIsOverwritten_File(t *ogletest.T) {
	// Create an object.
	t.AssertEq(nil, s.createWithContents("foo", "taco"))

	// Open the corresponding file for reading.
	f1, err := os.OpenFile(path.Join(s.mfs.Dir(), "foo"), os.O_RDONLY, 0)
	t.AssertEq(nil, err)
	defer func() {
		t.ExpectEq(nil, f1.Close())
	}()

	// Make sure that the contents are cached locally.
	_, err = f1.ReadAt(make([]byte, 1), 0)
	t.AssertEq(nil, err)

	// Overwrite the object.
	t.AssertEq(nil, s.createWithContents("foo", "burrito"))

	// The file should appear to be unlinked, but with the previous contents.
	fi, err := f1.Stat()

	t.AssertEq(nil, err)
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)

	// Opening again should yield the new version.
	//
	// NOTE(jacobsa): We must open with a different mode here than above to work
	// around the fact that osxfuse will re-use file handles. See the notes on
	// fuse.FileSystem.OpenFile for more.
	f2, err := os.OpenFile(path.Join(s.mfs.Dir(), "foo"), os.O_RDWR, 0)
	t.AssertEq(nil, err)
	defer func() {
		t.ExpectEq(nil, f2.Close())
	}()

	fi, err = f2.Stat()
	t.AssertEq(nil, err)
	t.ExpectEq(len("burrito"), fi.Size())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Reading from the old file handle should give the old data.
	contents, err := ioutil.ReadAll(f1)
	t.AssertEq(nil, err)
	t.ExpectEq("taco", string(contents))

	contents, err = ioutil.ReadAll(f2)
	t.AssertEq(nil, err)
	t.ExpectEq("burrito", string(contents))
}

func (s *ForeignModsTest) ObjectIsOverwritten_Directory(t *ogletest.T) {
	var err error

	// Create a directory placeholder.
	t.AssertEq(nil, s.createWithContents("dir/", ""))

	// Open the corresponding inode.
	s.f1, err = os.Open(path.Join(s.mfs.Dir(), "dir"))
	t.AssertEq(nil, err)

	// Overwrite the object.
	t.AssertEq(nil, s.createWithContents("dir/", ""))

	// The inode should still be accessible.
	fi, err := s.f1.Stat()

	t.AssertEq(nil, err)
	t.ExpectEq("dir", fi.Name())
	t.ExpectTrue(fi.IsDir())
}

func (s *ForeignModsTest) ObjectIsDeleted_File(t *ogletest.T) {
	// Create an object.
	t.AssertEq(nil, s.createWithContents("foo", "taco"))

	// Open the corresponding file for reading.
	f1, err := os.Open(path.Join(s.mfs.Dir(), "foo"))
	defer func() {
		if f1 != nil {
			t.ExpectEq(nil, f1.Close())
		}
	}()

	t.AssertEq(nil, err)

	// Delete the object.
	t.AssertEq(nil, s.bucket.DeleteObject(s.ctx, "foo"))

	// The file should appear to be unlinked, but with the previous contents.
	fi, err := f1.Stat()

	t.AssertEq(nil, err)
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)

	// Opening again should not work.
	f2, err := os.Open(path.Join(s.mfs.Dir(), "foo"))
	defer func() {
		if f2 != nil {
			t.ExpectEq(nil, f2.Close())
		}
	}()

	t.AssertNe(nil, err)
	t.ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (s *ForeignModsTest) ObjectIsDeleted_Directory(t *ogletest.T) {
	var err error

	// Create a directory placeholder.
	t.AssertEq(nil, s.createWithContents("dir/", ""))

	// Open the corresponding inode.
	s.f1, err = os.Open(path.Join(s.mfs.Dir(), "dir"))
	t.AssertEq(nil, err)

	// Delete the object.
	t.AssertEq(nil, s.bucket.DeleteObject(s.ctx, "dir/"))

	// The inode should still be fstat'able.
	fi, err := s.f1.Stat()

	t.AssertEq(nil, err)
	t.ExpectEq("dir", fi.Name())
	t.ExpectTrue(fi.IsDir())

	// Opening again should not work.
	s.f2, err = os.Open(path.Join(s.mfs.Dir(), "dir"))
	t.ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (s *ForeignModsTest) Symlink(t *ogletest.T) {
	var err error

	// Create an object that looks like a symlink.
	req := &gcs.CreateObjectRequest{
		Name: "foo",
		Metadata: map[string]string{
			"gcsfuse_symlink_target": "bar/baz",
		},
		Contents: ioutil.NopCloser(strings.NewReader("")),
	}

	_, err = s.bucket.CreateObject(s.ctx, req)
	t.AssertEq(nil, err)

	// Stat the link.
	fi, err := os.Lstat(path.Join(s.Dir, "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(filePerms|os.ModeSymlink, fi.Mode())

	// Read the link.
	target, err := os.Readlink(path.Join(s.Dir, "foo"))
	t.AssertEq(nil, err)
	t.ExpectEq("bar/baz", target)
}
