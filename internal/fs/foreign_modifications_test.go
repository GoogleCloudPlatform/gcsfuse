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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/inode"
	"github.com/jacobsa/fuse/fusetesting"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestForeignModsTestSuite(t *testing.T) {
	suite.Run(t, new(ForeignModsTest))
}

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
	suite.Suite
	suite.SetupAllSuite
	suite.TearDownAllSuite
	suite.TearDownTestSuite
	fsTest
}

func (t *ForeignModsTest) SetupSuite() {
	t.fsTest.SetupSuite()
}

//func (t *ForeignModsTest) TearDownSuite() {
//t.fsTest.TearDownSuite()
//}

// func (t *ForeignModsTest) TearDownTest() {
// 	t.fsTest.TearDownTest()
// }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ForeignModsTest) TestStatRoot() {
	fi, err := os.Stat(mntDir)
	assert.Nil(t.T(), err)

	assert.Equal(t.T(), path.Base(mntDir), fi.Name())
	assert.Equal(t.T(), 0, fi.Size())
	assert.Equal(t.T(), dirPerms|os.ModeDir, fi.Mode())
	assert.True(t.T(), fi.IsDir())
	assert.Equal(t.T(), 1, fi.Sys().(*syscall.Stat_t).Nlink)
	assert.Equal(t.T(), currentUid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Uid)
	assert.Equal(t.T(), currentGid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *ForeignModsTest) TestReadDir_EmptyRoot() {
	// ReadDir
	entries, err := fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)

	assert.ElementsMatch(t.T(), entries, []os.FileInfo{})
}

func (t *ForeignModsTest) TestReadDir_ContentsInRoot() {
	// Set up contents.
	createTime := mtimeClock.Now()
	assert.Equal(t.T(),
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
	assert.Nil(t.T(), err)

	assert.Equal(t.T(), 3, len(entries), "Names: %v", getFileNames(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	assert.Equal(t.T(), "bar", e.Name())
	assert.Equal(t.T(), 0, e.Size())
	assert.Equal(t.T(), dirPerms|os.ModeDir, e.Mode())
	assert.True(t.T(), e.IsDir())
	assert.Equal(t.T(), 1, e.Sys().(*syscall.Stat_t).Nlink)
	assert.Equal(t.T(), currentUid(&t.fsTest), e.Sys().(*syscall.Stat_t).Uid)
	assert.Equal(t.T(), currentGid(&t.fsTest), e.Sys().(*syscall.Stat_t).Gid)

	// baz
	e = entries[1]
	assert.Equal(t.T(), "baz", e.Name())
	assert.Equal(t.T(), len("burrito"), e.Size())
	assert.Equal(t.T(), filePerms, e.Mode())
	ExpectThat(e, fusetesting.MtimeIsWithin(createTime, timeSlop))
	assert.False(t.T(), e.IsDir())
	assert.Equal(t.T(), 1, e.Sys().(*syscall.Stat_t).Nlink)
	assert.Equal(t.T(), currentUid(&t.fsTest), e.Sys().(*syscall.Stat_t).Uid)
	assert.Equal(t.T(), currentGid(&t.fsTest), e.Sys().(*syscall.Stat_t).Gid)

	// foo
	e = entries[2]
	assert.Equal(t.T(), "foo", e.Name())
	assert.Equal(t.T(), len("taco"), e.Size())
	assert.Equal(t.T(), filePerms, e.Mode())
	ExpectThat(e, fusetesting.MtimeIsWithin(createTime, timeSlop))
	assert.False(t.T(), e.IsDir())
	assert.Equal(t.T(), 1, e.Sys().(*syscall.Stat_t).Nlink)
	assert.Equal(t.T(), currentUid(&t.fsTest), e.Sys().(*syscall.Stat_t).Uid)
	assert.Equal(t.T(), currentGid(&t.fsTest), e.Sys().(*syscall.Stat_t).Gid)
}

func (t *ForeignModsTest) TestReadDir_EmptySubDirectory() {
	// Set up an empty directory placeholder called 'bar'.
	assert.Equal(t.T(), nil, t.createEmptyObjects([]string{"bar/"}))

	// ReadDir
	entries, err := fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 1, len(entries))

	entries, err = fusetesting.ReadDirPicky(path.Join(mntDir, "bar"))
	assert.Nil(t.T(), err)

	ExpectThat(entries, ElementsAre())
}

func (t *ForeignModsTest) TestReadDir_ContentsInSubDirectory() {
	// Set up contents.
	createTime := mtimeClock.Now()
	assert.Equal(t.T(),
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
	assert.Nil(t.T(), err)

	// ReadDir
	entries, err := fusetesting.ReadDirPicky(path.Join(mntDir, "dir"))
	assert.Nil(t.T(), err)

	assert.Equal(t.T(), 3, len(entries), "Names: %v", getFileNames(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	assert.Equal(t.T(), "bar", e.Name())
	assert.Equal(t.T(), 0, e.Size())
	assert.Equal(t.T(), dirPerms|os.ModeDir, e.Mode())
	assert.True(t.T(), e.IsDir())
	assert.Equal(t.T(), 1, e.Sys().(*syscall.Stat_t).Nlink)
	assert.Equal(t.T(), currentUid(&t.fsTest), e.Sys().(*syscall.Stat_t).Uid)
	assert.Equal(t.T(), currentGid(&t.fsTest), e.Sys().(*syscall.Stat_t).Gid)

	// baz
	e = entries[1]
	assert.Equal(t.T(), "baz", e.Name())
	assert.Equal(t.T(), len("burrito"), e.Size())
	assert.Equal(t.T(), filePerms, e.Mode())
	ExpectThat(e, fusetesting.MtimeIsWithin(createTime, timeSlop))
	assert.False(t.T(), e.IsDir())
	assert.Equal(t.T(), 1, e.Sys().(*syscall.Stat_t).Nlink)
	assert.Equal(t.T(), currentUid(&t.fsTest), e.Sys().(*syscall.Stat_t).Uid)
	assert.Equal(t.T(), currentGid(&t.fsTest), e.Sys().(*syscall.Stat_t).Gid)

	// foo
	e = entries[2]
	assert.Equal(t.T(), "foo", e.Name())
	assert.Equal(t.T(), len("taco"), e.Size())
	assert.Equal(t.T(), filePerms, e.Mode())
	ExpectThat(e, fusetesting.MtimeIsWithin(createTime, timeSlop))
	assert.False(t.T(), e.IsDir())
	assert.Equal(t.T(), 1, e.Sys().(*syscall.Stat_t).Nlink)
	assert.Equal(t.T(), currentUid(&t.fsTest), e.Sys().(*syscall.Stat_t).Uid)
	assert.Equal(t.T(), currentGid(&t.fsTest), e.Sys().(*syscall.Stat_t).Gid)
}

func (t *ForeignModsTest) TestUnreachableObjects() {
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

	assert.Nil(t.T(), err)

	// Only the conflicitng file name should show up in the root.
	entries, err := fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 1, len(entries))

	fi = entries[0]
	assert.Equal(t.T(), "test", fi.Name())
	assert.Equal(t.T(), filePerms, fi.Mode())
	assert.Equal(t.T(), 1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting the conflicting name should give the file.
	fi, err = os.Stat(path.Join(mntDir, "test"))
	assert.Nil(t.T(), err)

	assert.Equal(t.T(), "test", fi.Name())
	assert.Equal(t.T(), filePerms, fi.Mode())
	assert.Equal(t.T(), 1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting the other name shouldn't work at all.
	_, err = os.Stat(path.Join(mntDir, "bar"))
	assert.True(t.T(), os.IsNotExist(err), "err: %v", err)

	// These unreachable objects (test/0, test/1, bar/0) start showing up in
	// other tests as soon as directory with similar name is created. Hence
	// cleaning them.
	err = storageutil.DeleteAllObjects(ctx, bucket)
	assert.Nil(t.T(), err)
}

func (t *ForeignModsTest) TestFileAndDirectoryWithConflictingName() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up an object named "foo" and one named "foo/", plus a child for the
	// latter.
	assert.Equal(t.T(),
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
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, len(entries))

	fi = entries[0]
	assert.Equal(t.T(), "foo", fi.Name())
	assert.Equal(t.T(), 0, fi.Size())
	assert.Equal(t.T(), dirPerms|os.ModeDir, fi.Mode())
	assert.True(t.T(), fi.IsDir())
	assert.Equal(t.T(), 1, fi.Sys().(*syscall.Stat_t).Nlink)

	fi = entries[1]
	assert.Equal(t.T(), "foo\n", fi.Name())
	assert.Equal(t.T(), len("taco"), fi.Size())
	assert.Equal(t.T(), filePerms, fi.Mode())
	assert.False(t.T(), fi.IsDir())
	assert.Equal(t.T(), 1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting "foo" should yield the directory.
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	assert.Equal(t.T(), "foo", fi.Name())
	assert.True(t.T(), fi.IsDir())

	// Statting "foo\n" should yield the file.
	fi, err = os.Stat(path.Join(mntDir, "foo\n"))
	assert.Nil(t.T(), err)

	assert.Equal(t.T(), "foo\n", fi.Name())
	assert.Equal(t.T(), len("taco"), fi.Size())
	assert.False(t.T(), fi.IsDir())

	// Listing the directory should yield the sole child file.
	entries, err = fusetesting.ReadDirPicky(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 1, len(entries))

	fi = entries[0]
	assert.Equal(t.T(), "bar", fi.Name())
	assert.Equal(t.T(), len("burrito"), fi.Size())
	assert.Equal(t.T(), filePerms, fi.Mode())
	assert.False(t.T(), fi.IsDir())
	assert.Equal(t.T(), 1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *ForeignModsTest) TestSymlinkAndDirectoryWithConflictingName() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up an object named "foo" and one named "foo/", plus a child for the
	// latter.
	assert.Equal(t.T(),
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
	assert.Nil(t.T(), err)

	// A listing of the parent should contain a directory named "foo" and a
	// symlink named "foo\n".
	entries, err = fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, len(entries))

	fi = entries[0]
	assert.Equal(t.T(), "foo", fi.Name())
	assert.Equal(t.T(), 0, fi.Size())
	assert.Equal(t.T(), dirPerms|os.ModeDir, fi.Mode())
	assert.True(t.T(), fi.IsDir())
	assert.Equal(t.T(), 1, fi.Sys().(*syscall.Stat_t).Nlink)

	fi = entries[1]
	assert.Equal(t.T(), "foo\n", fi.Name())
	assert.Equal(t.T(), 0, fi.Size())
	assert.Equal(t.T(), filePerms|os.ModeSymlink, fi.Mode())
	assert.False(t.T(), fi.IsDir())
	assert.Equal(t.T(), 1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting "foo" should yield the directory.
	fi, err = os.Lstat(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	assert.Equal(t.T(), "foo", fi.Name())
	assert.True(t.T(), fi.IsDir())

	// Statting "foo\n" should yield the symlink.
	fi, err = os.Lstat(path.Join(mntDir, "foo\n"))
	assert.Nil(t.T(), err)

	assert.Equal(t.T(), "foo\n", fi.Name())
	assert.False(t.T(), fi.IsDir())

	// Listing the directory should yield the sole child file.
	entries, err = fusetesting.ReadDirPicky(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 1, len(entries))

	fi = entries[0]
	assert.Equal(t.T(), "bar", fi.Name())
	assert.Equal(t.T(), len("burrito"), fi.Size())
	assert.Equal(t.T(), filePerms, fi.Mode())
	assert.False(t.T(), fi.IsDir())
	assert.Equal(t.T(), 1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *ForeignModsTest) TestStatTrailingNewlineName_NoConflictingNames() {
	var err error

	// Set up an object named "foo".
	assert.Equal(t.T(),
		nil,
		t.createObjects(
			map[string]string{
				"foo": "taco",
			}))

	// We shouldn't be able to stat "foo\n", because there is no conflicting
	// directory name.
	_, err = os.Stat(path.Join(mntDir, "foo\n"))
	assert.True(t.T(), os.IsNotExist(err), "err: %v", err)
}

func (t *ForeignModsTest) TestInodes() {
	// Set up two files and a directory placeholder.
	assert.Equal(t.T(),
		nil,
		t.createEmptyObjects([]string{
			"foo",
			"bar/",
			"baz",
		}))

	// List.
	entries, err := fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)

	assert.Equal(t.T(), 3, len(entries), "Names: %v", getFileNames(entries))

	// Confirm all of the inodes are distinct.
	inodesSeen := make(map[uint64]struct{})
	for _, fileInfo := range entries {
		stat := fileInfo.Sys().(*syscall.Stat_t)
		_, ok := inodesSeen[stat.Ino]
		assert.False(t.T(),
			ok,
			"Duplicate inode (%v). File info: %v",
			stat.Ino,
			fileInfo)

		inodesSeen[stat.Ino] = struct{}{}
	}
}

func (t *ForeignModsTest) TestOpenNonExistentFile() {
	_, err := os.Open(path.Join(mntDir, "foo"))

	assert.NotNil(t.T(), err)
	assert.ErrorContains(t.T(), err, "foo")
	assert.ErrorContains(t.T(), err, "no such file")
}

func (t *ForeignModsTest) TestReadFromFile_Small() {
	const contents = "tacoburritoenchilada"

	// Create an object.
	assert.Equal(t.T(), nil, t.createWithContents("foo", contents))

	// Wait for it to show up in the file system.
	_, err := fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)

	// Attempt to open it.
	f, err := os.Open(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)
	defer func() { assert.Equal(t.T(), nil, f.Close()) }()

	// Read its entire contents.
	slice, err := ioutil.ReadAll(f)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "tacoburritoenchilada", string(slice))

	// Read various ranges of it.
	var s string

	s, err = readRange(f, int64(len("taco")), len("burrito"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "burrito", s)

	s, err = readRange(f, 0, len("taco"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "taco", s)

	s, err = readRange(f, int64(len("tacoburrito")), len("enchilada"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "enchilada", s)
}

func (t *ForeignModsTest) TestReadFromFile_Large() {
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

		assert.Nil(t.T(), err)

		// Attempt to open it.
		f, err := os.Open(path.Join(mntDir, "foo"))
		assert.Nil(t.T(), err)
		defer func() { assert.Equal(t.T(), nil, f.Close()) }()

		// Read part of it.
		offset := randSrc.Int63n(contentLen + 1)
		size := randSrc.Intn(int(contentLen - offset))

		n, err := f.ReadAt(buf[:size], offset)
		if offset+int64(size) == contentLen && err == io.EOF {
			err = nil
		}

		assert.Nil(t.T(), err)
		assert.Equal(t.T(), size, n)
		assert.True(t.T(),
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

func (t *ForeignModsTest) TestReadBeyondEndOfFile() {
	const contents = "tacoburritoenchilada"
	const contentLen = len(contents)

	// Create an object.
	assert.Equal(t.T(), nil, t.createWithContents("foo", contents))

	// Wait for it to show up in the file system.
	_, err := fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)

	// Attempt to open it.
	f, err := os.Open(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)
	defer func() { assert.Equal(t.T(), nil, f.Close()) }()

	// Attempt to read beyond the end of the file.
	_, err = f.Seek(int64(contentLen-1), 0)
	assert.Nil(t.T(), err)

	buf := make([]byte, 2)
	n, err := f.Read(buf)
	assert.Equal(t.T(), 1, n, "err: %v", err)
	assert.Equal(t.T(), contents[contentLen-1], buf[0])

	if err == nil {
		n, err = f.Read(buf)
		assert.Equal(t.T(), 0, n)
	}
}

func (t *ForeignModsTest) TestObjectIsOverwritten_File() {
	// Create an object.
	assert.Equal(t.T(), nil, t.createWithContents("foo", "taco"))

	// Open the corresponding file for reading.
	f1, err := os.OpenFile(path.Join(mntDir, "foo"), os.O_RDONLY, 0)
	assert.Nil(t.T(), err)
	defer func() {
		assert.Equal(t.T(), nil, f1.Close())
	}()

	// Make sure that the contents are cached locally.
	_, err = f1.ReadAt(make([]byte, 1), 0)
	assert.Nil(t.T(), err)

	// Overwrite the object.
	assert.Equal(t.T(), nil, t.createWithContents("foo", "burrito"))

	// The file should appear to be unlinked, but with the previous contents.
	fi, err := f1.Stat()

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), len("taco"), fi.Size())
	assert.Equal(t.T(), 0, fi.Sys().(*syscall.Stat_t).Nlink)

	// Opening again should yield the new version.
	//
	// NOTE: We must open with a different mode here than above to work
	// around the fact that osxfuse will re-use file handles. See the notes on
	// fuse.FileSystem.OpenFile for more.
	f2, err := os.OpenFile(path.Join(mntDir, "foo"), os.O_RDWR, 0)
	assert.Nil(t.T(), err)
	defer func() {
		assert.Equal(t.T(), nil, f2.Close())
	}()

	fi, err = f2.Stat()
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), len("burrito"), fi.Size())
	assert.Equal(t.T(), 1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Reading from the old file handle should give the old data.
	contents, err := ioutil.ReadAll(f1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "taco", string(contents))

	contents, err = ioutil.ReadAll(f2)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "burrito", string(contents))
}

func (t *ForeignModsTest) TestObjectIsOverwritten_Directory() {
	var err error

	// Create a directory placeholder.
	assert.Equal(t.T(), nil, t.createWithContents("dir/", ""))

	// Open the corresponding inode.
	t.f1, err = os.Open(path.Join(mntDir, "dir"))
	assert.Nil(t.T(), err)

	// Overwrite the object.
	assert.Equal(t.T(), nil, t.createWithContents("dir/", ""))

	// The inode should still be accessible.
	fi, err := t.f1.Stat()

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "dir", fi.Name())
	assert.True(t.T(), fi.IsDir())
}

func (t *ForeignModsTest) TestObjectMetadataChanged_File() {
	// Create an object.
	assert.Equal(t.T(), nil, t.createWithContents("foo", "taco"))

	// Open the corresponding file for reading.
	f1, err := os.OpenFile(path.Join(mntDir, "foo"), os.O_RDONLY, 0)
	assert.Nil(t.T(), err)
	defer func() {
		assert.Equal(t.T(), nil, f1.Close())
	}()

	// Make sure that the contents are cached locally.
	_, err = f1.ReadAt(make([]byte, 1), 0)
	assert.Nil(t.T(), err)

	// Change the object's metadata, causing a new generation.
	lang := "fr"
	_, err = bucket.UpdateObject(
		ctx,
		&gcs.UpdateObjectRequest{
			Name:            "foo",
			ContentLanguage: &lang,
		})

	assert.Nil(t.T(), err)

	// The file should appear to be unlinked, but with the previous contents.
	fi, err := f1.Stat()

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), len("taco"), fi.Size())
	assert.Equal(t.T(), 0, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *ForeignModsTest) TestObjectMetadataChanged_Directory() {
	var err error

	// Create a directory placeholder.
	assert.Equal(t.T(), nil, t.createWithContents("dir/", ""))

	// Open the corresponding inode.
	t.f1, err = os.Open(path.Join(mntDir, "dir"))
	assert.Nil(t.T(), err)

	// Change the object's metadata, causing a new generation.
	lang := "fr"
	_, err = bucket.UpdateObject(
		ctx,
		&gcs.UpdateObjectRequest{
			Name:            "dir/",
			ContentLanguage: &lang,
		})

	assert.Nil(t.T(), err)

	// The inode should still be accessible.
	fi, err := t.f1.Stat()

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "dir", fi.Name())
	assert.True(t.T(), fi.IsDir())
}

func (t *ForeignModsTest) TestObjectIsDeleted_File() {
	// Create an object.
	assert.Equal(t.T(), nil, t.createWithContents("foo", "taco"))

	// Open the corresponding file for reading.
	f1, err := os.Open(path.Join(mntDir, "foo"))
	defer func() {
		if f1 != nil {
			assert.Equal(t.T(), nil, f1.Close())
		}
	}()

	assert.Nil(t.T(), err)

	// Delete the object.
	assert.Equal(t.T(),
		nil,
		bucket.DeleteObject(
			ctx,
			&gcs.DeleteObjectRequest{Name: "foo"}))

	// The file should appear to be unlinked, but with the previous contents.
	fi, err := f1.Stat()

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), len("taco"), fi.Size())
	assert.Equal(t.T(), 0, fi.Sys().(*syscall.Stat_t).Nlink)

	// Opening again should not work.
	f2, err := os.Open(path.Join(mntDir, "foo"))
	defer func() {
		if f2 != nil {
			assert.Equal(t.T(), nil, f2.Close())
		}
	}()

	assert.NotNil(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err), "err: %v", err)
}

func (t *ForeignModsTest) TestObjectIsDeleted_Directory() {
	var err error

	// Create a directory placeholder.
	assert.Equal(t.T(), nil, t.createWithContents("dir/", ""))

	// Open the corresponding inode.
	t.f1, err = os.Open(path.Join(mntDir, "dir"))
	assert.Nil(t.T(), err)

	// Delete the object.
	assert.Equal(t.T(),
		nil,
		bucket.DeleteObject(
			ctx,
			&gcs.DeleteObjectRequest{Name: "dir/"}))

	// The inode should still be fstat'able.
	fi, err := t.f1.Stat()

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "dir", fi.Name())
	assert.True(t.T(), fi.IsDir())

	// Opening again should not work.
	t.f2, err = os.Open(path.Join(mntDir, "dir"))
	assert.True(t.T(), os.IsNotExist(err), "err: %v", err)
}

func (t *ForeignModsTest) TestMtime() {
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
	assert.Nil(t.T(), err)

	// Stat the file.
	fi, err := os.Stat(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	ExpectThat(fi.ModTime(), timeutil.TimeEq(expected))
}

func (t *ForeignModsTest) TestRemoteMtimeChange() {
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

	assert.Nil(t.T(), err)

	// Stat the object so that the file system assigns it an inode.
	_, err = os.Stat(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

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

	assert.Nil(t.T(), err)

	// Stat the file again. We should see the new mtime.
	fi, err := os.Stat(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	//ExpectThat(fi.ModTime(), timeutil.TimeEq(expected))
	assert.Exactly(t.T(), expected, fi.ModTime())
}

func (t *ForeignModsTest) TestSymlink() {
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
	assert.Nil(t.T(), err)

	// Stat the link.
	fi, err := os.Lstat(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	assert.Equal(t.T(), "foo", fi.Name())
	assert.Equal(t.T(), 0, fi.Size())
	assert.Equal(t.T(), filePerms|os.ModeSymlink, fi.Mode())

	// Read the link.
	target, err := os.Readlink(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "bar/baz", target)
}
