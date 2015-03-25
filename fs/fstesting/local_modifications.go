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

// A collection of tests for a file system backed by a GCS bucket, where we
// interact with the file system directly for creating and modifying files
// (rather than through the side channel of the GCS bucket itself).
//
// These tests are registered by RegisterFSTests.

package fstesting

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"syscall"
	"time"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"google.golang.org/cloud/storage"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func getFileOffset(f *os.File) (offset int64, err error) {
	const relativeToCurrent = 1
	offset, err = f.Seek(0, relativeToCurrent)
	return
}

////////////////////////////////////////////////////////////////////////
// Open
////////////////////////////////////////////////////////////////////////

type openTest struct {
	fsTest
}

func (t *openTest) NonExistent_CreateFlagNotSet() {
	var err error
	t.f1, err = os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR, 0700)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("no such file")))

	// No object should have been created.
	_, err = gcsutil.ReadObject(t.ctx, t.bucket, "foo")
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *openTest) NonExistent_CreateFlagSet() {
	var err error

	// Open the file.
	t.f1, err = os.OpenFile(
		path.Join(t.mfs.Dir(), "foo"),
		os.O_RDWR|os.O_CREATE,
		0700)

	AssertEq(nil, err)

	// The object should now be present in the bucket, with empty contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("", contents)

	// Write some contents.
	_, err = t.f1.Write([]byte("012"))
	AssertEq(nil, err)

	// Read some contents with Seek and Read.
	_, err = t.f1.Seek(1, 0)
	AssertEq(nil, err)

	buf := make([]byte, 2)
	_, err = io.ReadFull(t.f1, buf)

	AssertEq(nil, err)
	ExpectEq("12", string(buf))

	// Close the file.
	AssertEq(nil, t.f1.Close())
	t.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("012", string(fileContents))
}

func (t *openTest) ExistingFile() {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR, 0)
	AssertEq(nil, err)

	// Write to the start of the file using File.Write.
	_, err = t.f1.Write([]byte("012"))
	AssertEq(nil, err)

	// Read some contents with Seek and Read.
	_, err = t.f1.Seek(2, 0)
	AssertEq(nil, err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(t.f1, buf)

	AssertEq(nil, err)
	ExpectEq("2obu", string(buf))

	// Close the file.
	AssertEq(nil, t.f1.Close())
	t.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("012oburritoenchilada", string(fileContents))
}

func (t *openTest) ExistingFile_Truncate() {
	var err error

	// Create a file.
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte("blahblahblah"),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(
		path.Join(t.mfs.Dir(), "foo"),
		os.O_RDWR|os.O_TRUNC,
		0)

	AssertEq(nil, err)

	// The file should be empty.
	fi, err := t.f1.Stat()
	AssertEq(nil, err)
	ExpectEq(0, fi.Size())

	// Write to the start of the file using File.Write.
	_, err = t.f1.Write([]byte("012"))
	AssertEq(nil, err)

	// Read the contents.
	_, err = t.f1.Seek(0, 0)
	AssertEq(nil, err)

	contentsSlice, err := ioutil.ReadAll(t.f1)
	AssertEq(nil, err)
	ExpectEq("012", string(contentsSlice))

	// Close the file.
	AssertEq(nil, t.f1.Close())
	t.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("012", string(fileContents))
}

func (t *openTest) AlreadyOpenedFile() {
	var err error
	var n int
	buf := make([]byte, 1024)

	// Create and open a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Write some data into it.
	n, err = t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Open another handle for reading and writing.
	t.f2, err = os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR, 0)
	AssertEq(nil, err)

	// The contents written through the first handle should be available to the
	// second handle..
	n, err = t.f2.Read(buf[:2])
	AssertEq(nil, err)
	AssertEq(2, n)
	ExpectEq("ta", string(buf[:n]))

	// Write some contents with the second handle, which should now be at offset
	// 2.
	n, err = t.f2.Write([]byte("nk"))
	AssertEq(nil, err)
	AssertEq(2, n)

	// Check the overall contents now.
	contents, err := ioutil.ReadFile(t.f2.Name())
	AssertEq(nil, err)
	ExpectEq("tank", string(contents))
}

////////////////////////////////////////////////////////////////////////
// Modes
////////////////////////////////////////////////////////////////////////

type modesTest struct {
	fsTest
}

func (t *modesTest) ReadOnlyMode() {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDONLY, 0)
	AssertEq(nil, err)

	// Read its contents.
	fileContents, err := ioutil.ReadAll(t.f1)
	AssertEq(nil, err)
	ExpectEq(contents, string(fileContents))

	// Attempt to write.
	n, err := t.f1.Write([]byte("taco"))

	AssertEq(0, n)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))
}

func (t *modesTest) WriteOnlyMode() {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_WRONLY, 0)
	AssertEq(nil, err)

	// Reading should fail.
	_, err = ioutil.ReadAll(t.f1)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))

	// Write to the start of the file using File.Write.
	_, err = t.f1.Write([]byte("000"))
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = t.f1.WriteAt([]byte("111"), 4)
	AssertEq(nil, err)

	// Seek and write past the end of the file.
	_, err = t.f1.Seek(int64(len(contents)), 0)
	AssertEq(nil, err)

	_, err = t.f1.Write([]byte("222"))
	AssertEq(nil, err)

	// Check the size now.
	fi, err := t.f1.Stat()
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Close the file.
	AssertEq(nil, t.f1.Close())
	t.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(fileContents))
}

func (t *modesTest) ReadWriteMode() {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR, 0)
	AssertEq(nil, err)

	// Write to the start of the file using File.Write.
	_, err = t.f1.Write([]byte("000"))
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = t.f1.WriteAt([]byte("111"), 4)
	AssertEq(nil, err)

	// Seek and write past the end of the file.
	_, err = t.f1.Seek(int64(len(contents)), 0)
	AssertEq(nil, err)

	_, err = t.f1.Write([]byte("222"))
	AssertEq(nil, err)

	// Check the size now.
	fi, err := t.f1.Stat()
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Read some contents with Seek and Read.
	_, err = t.f1.Seek(4, 0)
	AssertEq(nil, err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(t.f1, buf)

	AssertEq(nil, err)
	ExpectEq("111r", string(buf))

	// Read the full contents with ReadAt.
	buf = make([]byte, len(contents)+len("222"))
	_, err = t.f1.ReadAt(buf, 0)

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(buf))

	// Close the file.
	AssertEq(nil, t.f1.Close())
	t.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(fileContents))
}

func (t *modesTest) AppendMode_SeekAndWrite() {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR|os.O_APPEND, 0)
	AssertEq(nil, err)

	// Write using File.Write. This should go to the end of the file regardless
	// of whether we Seek somewhere else first.
	_, err = t.f1.Seek(1, 0)
	AssertEq(nil, err)

	_, err = t.f1.Write([]byte("222"))
	AssertEq(nil, err)

	// The seek position should have been updated.
	off, err := getFileOffset(t.f1)
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222"), off)

	// Check the size now.
	fi, err := t.f1.Stat()
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Read the full contents with ReadAt.
	buf := make([]byte, 1024)
	n, err := t.f1.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq(contents+"222", string(buf[:n]))

	// Read the full contents with another file handle.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq(contents+"222", string(fileContents))
}

func (t *modesTest) AppendMode_WriteAt() {
	var err error

	// Linux's support for pwrite is buggy; the pwrite(2) man page says this:
	//
	//     POSIX requires that opening a file with the O_APPEND flag should have
	//     no affect on the location at which pwrite() writes data.  However, on
	//     Linux,  if  a  file  is opened with O_APPEND, pwrite() appends data to
	//     the end of the file, regardless of the value of offset.
	//
	isLinux := (runtime.GOOS == "linux")

	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR|os.O_APPEND, 0)
	AssertEq(nil, err)

	// Seek somewhere in the file.
	_, err = t.f1.Seek(1, 0)
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = t.f1.WriteAt([]byte("111"), 4)
	AssertEq(nil, err)

	// The seek position should have been unaffected.
	off, err := getFileOffset(t.f1)
	AssertEq(nil, err)
	ExpectEq(1, off)

	// Check the size now.
	fi, err := t.f1.Stat()
	AssertEq(nil, err)

	if isLinux {
		ExpectEq(len(contents+"111"), fi.Size())
	} else {
		ExpectEq(len(contents), fi.Size())
	}

	// Read the full contents with ReadAt.
	buf := make([]byte, 1024)
	n, err := t.f1.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	if isLinux {
		ExpectEq("tacoburritoenchilada111", string(buf[:n]))
	} else {
		ExpectEq("taco111ritoenchilada", string(buf[:n]))
	}

	// Read the full contents with another file handle.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	if isLinux {
		ExpectEq("tacoburritoenchilada111", string(fileContents))
	} else {
		ExpectEq("taco111ritoenchilada", string(fileContents))
	}
}

func (t *modesTest) AppendMode_WriteAt_PastEOF() {
	var err error

	// Linux's support for pwrite is buggy; the pwrite(2) man page says this:
	//
	//     POSIX requires that opening a file with the O_APPEND flag should have
	//     no affect on the location at which pwrite() writes data.  However, on
	//     Linux,  if  a  file  is opened with O_APPEND, pwrite() appends data to
	//     the end of the file, regardless of the value of offset.
	//
	isLinux := (runtime.GOOS == "linux")

	// Open a file.
	t.f1, err = os.OpenFile(
		path.Join(t.mfs.Dir(), "foo"),
		os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0600)

	AssertEq(nil, err)

	// Write three bytes.
	n, err := t.f1.Write([]byte("111"))
	AssertEq(nil, err)
	AssertEq(3, n)

	// Write at offset six.
	n, err = t.f1.WriteAt([]byte("222"), 6)
	AssertEq(nil, err)
	AssertEq(3, n)

	// The seek position should have been unaffected.
	off, err := getFileOffset(t.f1)
	AssertEq(nil, err)
	ExpectEq(3, off)

	// Read the full contents of the file.
	contents, err := ioutil.ReadFile(t.f1.Name())
	AssertEq(nil, err)

	if isLinux {
		ExpectEq("111222", string(contents))
	} else {
		ExpectEq("111\x00\x00\x00222", string(contents))
	}
}

func (t *modesTest) ReadFromWriteOnlyFile() {
	var err error

	// Create and open a file for writing.
	t.f1, err = os.OpenFile(
		path.Join(t.mfs.Dir(), "foo"),
		os.O_WRONLY|os.O_CREATE,
		0700)

	AssertEq(nil, err)

	// Attempt to read from it.
	_, err = t.f1.Read(make([]byte, 1024))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))
}

func (t *modesTest) WriteToReadOnlyFile() {
	var err error

	// Create and open a file for reading.
	t.f1, err = os.OpenFile(
		path.Join(t.mfs.Dir(), "foo"),
		os.O_RDONLY|os.O_CREATE,
		0700)

	AssertEq(nil, err)

	// Attempt to write t it.
	_, err = t.f1.Write([]byte("taco"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))
}

////////////////////////////////////////////////////////////////////////
// Directory interaction
////////////////////////////////////////////////////////////////////////

type directoryTest struct {
	fsTest
}

func (t *directoryTest) Mkdir_OneLevel() {
	var err error
	var fi os.FileInfo
	var entries []os.FileInfo

	dirName := path.Join(t.mfs.Dir(), "dir")

	// Create a directory within the root.
	err = os.Mkdir(dirName, 0754)
	AssertEq(nil, err)

	// Stat the directory.
	fi, err = os.Stat(dirName)

	AssertEq(nil, err)
	ExpectEq("dir", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(os.ModeDir|0700, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)

	// Read the directory.
	entries, err = ioutil.ReadDir(dirName)

	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())

	// Read the root.
	entries, err = ioutil.ReadDir(t.mfs.Dir())

	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("dir", fi.Name())
	ExpectEq(os.ModeDir|0700, fi.Mode())
}

func (t *directoryTest) Mkdir_TwoLevels() {
	var err error
	var fi os.FileInfo
	var entries []os.FileInfo

	// Create a directory within the root.
	err = os.Mkdir(path.Join(t.mfs.Dir(), "parent"), 0700)
	AssertEq(nil, err)

	// Create a child of that directory.
	err = os.Mkdir(path.Join(t.mfs.Dir(), "parent/dir"), 0754)
	AssertEq(nil, err)

	// Stat the directory.
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "parent/dir"))

	AssertEq(nil, err)
	ExpectEq("dir", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(os.ModeDir|0700, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)

	// Read the directory.
	entries, err = ioutil.ReadDir(path.Join(t.mfs.Dir(), "parent/dir"))

	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())

	// Read the parent.
	entries, err = ioutil.ReadDir(path.Join(t.mfs.Dir(), "parent"))

	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("dir", fi.Name())
	ExpectEq(os.ModeDir|0700, fi.Mode())
}

func (t *directoryTest) Mkdir_AlreadyExists() {
	var err error
	dirName := path.Join(t.mfs.Dir(), "dir")

	// Create the directory once.
	err = os.Mkdir(dirName, 0754)
	AssertEq(nil, err)

	// Attempt to create it again.
	err = os.Mkdir(dirName, 0754)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("exists")))
}

func (t *directoryTest) Mkdir_IntermediateIsFile() {
	var err error

	// Create a file.
	fileName := path.Join(t.mfs.Dir(), "foo")
	err = ioutil.WriteFile(fileName, []byte{}, 0700)
	AssertEq(nil, err)

	// Attempt to create a directory within the file.
	dirName := path.Join(fileName, "dir")
	err = os.Mkdir(dirName, 0754)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("not a directory")))
}

func (t *directoryTest) Mkdir_IntermediateIsNonExistent() {
	var err error

	// Attempt to create a sub-directory of a non-existent sub-directory.
	dirName := path.Join(t.mfs.Dir(), "foo/dir")
	err = os.Mkdir(dirName, 0754)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("no such file or directory")))
}

func (t *directoryTest) Stat_Root() {
	fi, err := os.Stat(t.mfs.Dir())
	AssertEq(nil, err)

	ExpectEq(path.Base(t.mfs.Dir()), fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(os.ModeDir|0700, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *directoryTest) Stat_FirstLevelDirectory() {
	var err error

	// Create a sub-directory.
	err = os.Mkdir(path.Join(t.mfs.Dir(), "dir"), 0700)
	AssertEq(nil, err)

	// Stat it.
	fi, err := os.Stat(path.Join(t.mfs.Dir(), "dir"))
	AssertEq(nil, err)

	ExpectEq("dir", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(os.ModeDir|0700, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *directoryTest) Stat_SecondLevelDirectory() {
	var err error

	// Create two levels of directories.
	err = os.MkdirAll(path.Join(t.mfs.Dir(), "parent/dir"), 0700)
	AssertEq(nil, err)

	// Stat it.
	fi, err := os.Stat(path.Join(t.mfs.Dir(), "parent/dir"))
	AssertEq(nil, err)

	ExpectEq("dir", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(os.ModeDir|0700, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *directoryTest) ReadDir_Root() {
	var err error
	var fi os.FileInfo

	// Create a file and a directory.
	t.advanceTime()
	createTime := t.clock.Now()

	err = ioutil.WriteFile(path.Join(t.mfs.Dir(), "bar"), []byte("taco"), 0700)
	AssertEq(nil, err)

	err = os.Mkdir(path.Join(t.mfs.Dir(), "foo"), 0700)
	AssertEq(nil, err)

	t.advanceTime()

	// ReadDir
	entries, err := ioutil.ReadDir(t.mfs.Dir())
	AssertEq(nil, err)
	AssertEq(2, len(entries))

	// bar
	fi = entries[0]
	ExpectEq("bar", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(os.FileMode(0700), fi.Mode())
	ExpectThat(fi.ModTime(), t.matchesStartTime(createTime))
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)

	// foo
	fi = entries[1]
	ExpectEq("foo", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(os.ModeDir|0700, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *directoryTest) ReadDir_SubDirectory() {
	var err error
	var fi os.FileInfo

	// Create a directory.
	parent := path.Join(t.mfs.Dir(), "parent")
	err = os.Mkdir(parent, 0700)
	AssertEq(nil, err)

	// Create a file and a directory within it.
	t.advanceTime()
	createTime := t.clock.Now()

	err = ioutil.WriteFile(path.Join(parent, "bar"), []byte("taco"), 0700)
	AssertEq(nil, err)

	err = os.Mkdir(path.Join(parent, "foo"), 0700)
	AssertEq(nil, err)

	t.advanceTime()

	// ReadDir
	entries, err := ioutil.ReadDir(parent)
	AssertEq(nil, err)
	AssertEq(2, len(entries))

	// bar
	fi = entries[0]
	ExpectEq("bar", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(os.FileMode(0700), fi.Mode())
	ExpectThat(fi.ModTime(), t.matchesStartTime(createTime))
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)

	// foo
	fi = entries[1]
	ExpectEq("foo", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(os.ModeDir|0700, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *directoryTest) Rmdir_Empty() {
	var err error
	var entries []os.FileInfo

	// Create two levels of directories.
	err = os.MkdirAll(path.Join(t.mfs.Dir(), "foo/bar"), 0754)
	AssertEq(nil, err)

	// Remove the leaf.
	err = os.Remove(path.Join(t.mfs.Dir(), "foo/bar"))
	AssertEq(nil, err)

	// There should be nothing left in the parent.
	entries, err = ioutil.ReadDir(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())

	// Remove the parent.
	err = os.Remove(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Now the root directory should be empty, too.
	entries, err = ioutil.ReadDir(t.mfs.Dir())

	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())
}

func (t *directoryTest) Rmdir_OpenedForReading() {
	// Cf. https://github.com/GoogleCloudPlatform/gcsfuse/issues/8
	AddFailure("Waiting for issue #8 to be resolved.")
	AbortTest()

	var err error

	// Create a directory.
	err = os.Mkdir(path.Join(t.mfs.Dir(), "dir"), 0700)
	AssertEq(nil, err)

	// Open the directory for reading.
	t.f1, err = os.Open(path.Join(t.mfs.Dir(), "dir"))
	AssertEq(nil, err)

	// Remove the directory.
	err = os.Remove(path.Join(t.mfs.Dir(), "dir"))
	AssertEq(nil, err)

	// Create a new directory, with the same name even, and add some contents
	// within it.
	err = os.MkdirAll(path.Join(t.mfs.Dir(), "dir/foo"), 0700)
	AssertEq(nil, err)

	err = os.MkdirAll(path.Join(t.mfs.Dir(), "dir/bar"), 0700)
	AssertEq(nil, err)

	err = os.MkdirAll(path.Join(t.mfs.Dir(), "dir/baz"), 0700)
	AssertEq(nil, err)

	// We should still be able to stat the open file handle. It should show up as
	// unlinked.
	fi, err := t.f1.Stat()

	ExpectEq("dir", fi.Name())
	ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)

	// Attempt to read from the directory. This shouldn't see any junk from the
	// new directory. It should either succeed with an empty result or should
	// return ENOENT.
	entries, err := t.f1.Readdir(0)

	if err != nil {
		ExpectThat(err, Error(HasSubstr("no such file")))
	} else {
		ExpectThat(entries, ElementsAre())
	}
}

func (t *directoryTest) CreateHardLink() {
	var err error

	// Write a file.
	err = ioutil.WriteFile(path.Join(t.mfs.Dir(), "foo"), []byte(""), 0700)
	AssertEq(nil, err)

	// Attempt to hard link it. We don't support doing so.
	err = os.Link(
		path.Join(t.mfs.Dir(), "foo"),
		path.Join(t.mfs.Dir(), "bar"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("not implemented")))
}

func (t *directoryTest) CreateSymlink() {
	var err error

	// Write a file.
	err = ioutil.WriteFile(path.Join(t.mfs.Dir(), "foo"), []byte(""), 0700)
	AssertEq(nil, err)

	// Attempt to symlink it. We don't support doing so.
	err = os.Symlink(
		path.Join(t.mfs.Dir(), "foo"),
		path.Join(t.mfs.Dir(), "bar"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("not implemented")))
}

////////////////////////////////////////////////////////////////////////
// File interaction
////////////////////////////////////////////////////////////////////////

type fileTest struct {
	fsTest
}

func (t *fileTest) WriteOverlapsEndOfFile() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Make it 4 bytes long.
	err = t.f1.Truncate(4)
	AssertEq(nil, err)

	// Write the range [2, 6).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(t.f1)
	AssertEq(nil, err)
	ExpectEq("\x00\x00taco", string(contents))
}

func (t *fileTest) WriteStartsAtEndOfFile() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Make it 2 bytes long.
	err = t.f1.Truncate(2)
	AssertEq(nil, err)

	// Write the range [2, 6).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(t.f1)
	AssertEq(nil, err)
	ExpectEq("\x00\x00taco", string(contents))
}

func (t *fileTest) WriteStartsPastEndOfFile() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Write the range [2, 6).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(t.f1)
	AssertEq(nil, err)
	ExpectEq("\x00\x00taco", string(contents))
}

func (t *fileTest) WriteAtDoesntChangeOffset_NotAppendMode() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Make it 16 bytes long.
	err = t.f1.Truncate(16)
	AssertEq(nil, err)

	// Seek to offset 4.
	_, err = t.f1.Seek(4, 0)
	AssertEq(nil, err)

	// Write the range [10, 14).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// We should still be at offset 4.
	offset, err := getFileOffset(t.f1)
	AssertEq(nil, err)
	ExpectEq(4, offset)
}

func (t *fileTest) WriteAtDoesntChangeOffset_AppendMode() {
	var err error
	var n int

	// Create a file in append mode.
	t.f1, err = os.OpenFile(
		path.Join(t.mfs.Dir(), "foo"),
		os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0600)

	AssertEq(nil, err)

	// Make it 16 bytes long.
	err = t.f1.Truncate(16)
	AssertEq(nil, err)

	// Seek to offset 4.
	_, err = t.f1.Seek(4, 0)
	AssertEq(nil, err)

	// Write the range [10, 14).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// We should still be at offset 4.
	offset, err := getFileOffset(t.f1)
	AssertEq(nil, err)
	ExpectEq(4, offset)
}

func (t *fileTest) ReadsPastEndOfFile() {
	var err error
	var n int
	buf := make([]byte, 1024)

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Give it some contents.
	n, err = t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Read a range overlapping EOF.
	n, err = t.f1.ReadAt(buf[:4], 2)
	AssertEq(io.EOF, err)
	ExpectEq(2, n)
	ExpectEq("co", string(buf[:n]))

	// Read a range starting at EOF.
	n, err = t.f1.ReadAt(buf[:4], 4)
	AssertEq(io.EOF, err)
	ExpectEq(0, n)
	ExpectEq("", string(buf[:n]))

	// Read a range starting past EOF.
	n, err = t.f1.ReadAt(buf[:4], 100)
	AssertEq(io.EOF, err)
	ExpectEq(0, n)
	ExpectEq("", string(buf[:n]))
}

func (t *fileTest) Truncate_Smaller() {
	var err error
	fileName := path.Join(t.mfs.Dir(), "foo")

	// Create a file.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	AssertEq(nil, err)

	// Open it for modification.
	t.f1, err = os.OpenFile(fileName, os.O_RDWR, 0)
	AssertEq(nil, err)

	// Truncate it.
	err = t.f1.Truncate(2)
	AssertEq(nil, err)

	// Stat it.
	fi, err := t.f1.Stat()
	AssertEq(nil, err)
	ExpectEq(2, fi.Size())

	// Read the contents.
	contents, err := ioutil.ReadFile(fileName)
	AssertEq(nil, err)
	ExpectEq("ta", string(contents))
}

func (t *fileTest) Truncate_SameSize() {
	var err error
	fileName := path.Join(t.mfs.Dir(), "foo")

	// Create a file.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	AssertEq(nil, err)

	// Open it for modification.
	t.f1, err = os.OpenFile(fileName, os.O_RDWR, 0)
	AssertEq(nil, err)

	// Truncate it.
	err = t.f1.Truncate(4)
	AssertEq(nil, err)

	// Stat it.
	fi, err := t.f1.Stat()
	AssertEq(nil, err)
	ExpectEq(4, fi.Size())

	// Read the contents.
	contents, err := ioutil.ReadFile(fileName)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *fileTest) Truncate_Larger() {
	var err error
	fileName := path.Join(t.mfs.Dir(), "foo")

	// Create a file.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	AssertEq(nil, err)

	// Open it for modification.
	t.f1, err = os.OpenFile(fileName, os.O_RDWR, 0)
	AssertEq(nil, err)

	// Truncate it.
	err = t.f1.Truncate(6)
	AssertEq(nil, err)

	// Stat it.
	fi, err := t.f1.Stat()
	AssertEq(nil, err)
	ExpectEq(6, fi.Size())

	// Read the contents.
	contents, err := ioutil.ReadFile(fileName)
	AssertEq(nil, err)
	ExpectEq("taco\x00\x00", string(contents))
}

func (t *fileTest) Seek() {
	var err error
	var n int
	buf := make([]byte, 1024)

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Give it some contents.
	n, err = t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Seek and overwrite.
	off, err := t.f1.Seek(1, 0)
	AssertEq(nil, err)
	AssertEq(1, off)

	n, err = t.f1.Write([]byte("xx"))
	AssertEq(nil, err)
	AssertEq(2, n)

	// Read full the contents of the file.
	n, err = t.f1.ReadAt(buf, 0)
	AssertEq(io.EOF, err)
	ExpectEq("txxo", string(buf[:n]))
}

func (t *fileTest) Stat() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Give it some contents.
	t.advanceTime()
	writeTime := t.clock.Now()

	n, err = t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	t.advanceTime()

	// Stat it.
	fi, err := t.f1.Stat()
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(os.FileMode(0700), fi.Mode())
	ExpectThat(fi.ModTime(), t.matchesStartTime(writeTime))
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *fileTest) StatUnopenedFile() {
	var err error

	// Create and close a file.
	t.advanceTime()
	createTime := t.clock.Now()

	err = ioutil.WriteFile(path.Join(t.mfs.Dir(), "foo"), []byte("taco"), 0700)
	AssertEq(nil, err)

	t.advanceTime()

	// Stat it.
	fi, err := os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(os.FileMode(0700), fi.Mode())
	ExpectThat(fi.ModTime(), t.matchesStartTime(createTime))
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *fileTest) LstatUnopenedFile() {
	var err error

	// Create and close a file.
	t.advanceTime()
	createTime := t.clock.Now()

	err = ioutil.WriteFile(path.Join(t.mfs.Dir(), "foo"), []byte("taco"), 0700)
	AssertEq(nil, err)

	t.advanceTime()

	// Lstat it.
	fi, err := os.Lstat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(os.FileMode(0700), fi.Mode())
	ExpectThat(fi.ModTime(), t.matchesStartTime(createTime))
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *fileTest) UnlinkFile_Exists() {
	var err error

	// Write a file.
	fileName := path.Join(t.mfs.Dir(), "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	AssertEq(nil, err)

	// Unlink it.
	err = os.Remove(fileName)
	AssertEq(nil, err)

	// Statting it should fail.
	_, err = os.Stat(fileName)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("no such file")))

	// Nothing should be in the directory.
	entries, err := ioutil.ReadDir(t.mfs.Dir())
	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())
}

func (t *fileTest) UnlinkFile_NonExistent() {
	err := os.Remove(path.Join(t.mfs.Dir(), "foo"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("no such file")))
}

func (t *fileTest) UnlinkFile_StillOpen() {
	var err error

	fileName := path.Join(t.mfs.Dir(), "foo")

	// Create and open a file.
	t.f1, err = os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0600)
	AssertEq(nil, err)

	// Write some data into it.
	n, err := t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Unlink it.
	err = os.Remove(fileName)
	AssertEq(nil, err)

	// The directory should no longer contain it.
	entries, err := ioutil.ReadDir(t.mfs.Dir())
	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())

	// We should be able to stat the file. It should still show as having
	// contents, but with no links.
	fi, err := t.f1.Stat()

	AssertEq(nil, err)
	ExpectEq(4, fi.Size())
	ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)

	// The contents should still be available.
	buf := make([]byte, 1024)
	n, err = t.f1.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	AssertEq(4, n)
	ExpectEq("taco", string(buf[:4]))

	// Writing should still work, too.
	n, err = t.f1.Write([]byte("burrito"))
	AssertEq(nil, err)
	AssertEq(len("burrito"), n)
}

func (t *fileTest) UnlinkFile_NoLongerInBucket() {
	var err error

	// Write a file.
	fileName := path.Join(t.mfs.Dir(), "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	AssertEq(nil, err)

	// Delete it from the bucket through the back door.
	err = t.bucket.DeleteObject(t.ctx, "foo")
	AssertEq(nil, err)

	// Attempt to unlink it.
	err = os.Remove(fileName)

	AssertNe(nil, err)
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *fileTest) UnlinkFile_FromSubDirectory() {
	var err error

	// Create a sub-directory.
	dirName := path.Join(t.mfs.Dir(), "dir")
	err = os.Mkdir(dirName, 0700)
	AssertEq(nil, err)

	// Write a file to that directory.
	fileName := path.Join(dirName, "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	AssertEq(nil, err)

	// Unlink it.
	err = os.Remove(fileName)
	AssertEq(nil, err)

	// Statting it should fail.
	_, err = os.Stat(fileName)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("no such file")))

	// Nothing should be in the directory.
	entries, err := ioutil.ReadDir(dirName)
	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())
}

func (t *fileTest) Chmod() {
	var err error

	// Write a file.
	fileName := path.Join(t.mfs.Dir(), "foo")
	err = ioutil.WriteFile(fileName, []byte(""), 0700)
	AssertEq(nil, err)

	// Attempt to chmod it. We don't support doing so.
	err = os.Chmod(fileName, 0777)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("not implemented")))
}

func (t *fileTest) Chtimes() {
	var err error

	// Write a file.
	fileName := path.Join(t.mfs.Dir(), "foo")
	err = ioutil.WriteFile(fileName, []byte(""), 0700)
	AssertEq(nil, err)

	// Attempt to change its atime and mtime. We don't support doing so.
	err = os.Chtimes(fileName, time.Now(), time.Now())

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("not implemented")))
}

func (t *fileTest) Sync_Dirty() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Give it some contents.
	n, err = t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Sync it.
	err = t.f1.Sync()
	AssertEq(nil, err)

	// The contents should now be in the bucket, even though we haven't closed
	// the file.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("taco", contents)
}

func (t *fileTest) Sync_NotDirty() {
	var err error

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// The above should have created a generation for the object. Grab a record
	// for it.
	statReq := &gcs.StatObjectRequest{
		Name: "foo",
	}

	o1, err := t.bucket.StatObject(t.ctx, statReq)
	AssertEq(nil, err)

	// Sync the file.
	err = t.f1.Sync()
	AssertEq(nil, err)

	// A new generation need not have been written.
	o2, err := t.bucket.StatObject(t.ctx, statReq)
	AssertEq(nil, err)

	ExpectEq(o1.Generation, o2.Generation)
}

func (t *fileTest) Sync_Clobbered() {
	AssertTrue(false, "TODO")
}

func (t *fileTest) Close_Dirty() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Give it some contents.
	n, err = t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Close it.
	err = t.f1.Close()
	t.f1 = nil
	AssertEq(nil, err)

	// The contents should now be in the bucket.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("taco", contents)
}

func (t *fileTest) Close_NotDirty() {
	var err error

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// The above should have created a generation for the object. Grab a record
	// for it.
	statReq := &gcs.StatObjectRequest{
		Name: "foo",
	}

	o1, err := t.bucket.StatObject(t.ctx, statReq)
	AssertEq(nil, err)

	// Close the file.
	err = t.f1.Close()
	t.f1 = nil
	AssertEq(nil, err)

	// A new generation need not have been written.
	o2, err := t.bucket.StatObject(t.ctx, statReq)
	AssertEq(nil, err)

	ExpectEq(o1.Generation, o2.Generation)
}

func (t *fileTest) Close_Clobbered() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Dirty the file by giving it some contents.
	n, err = t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Replace the underyling object with a new generation.
	_, err = gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		&storage.ObjectAttrs{
			Name: "foo",
		},
		"foobar")

	// Close the file. This should not result in an error, but the new generation
	// should not be replaced.
	err = t.f1.Close()
	t.f1 = nil
	AssertEq(nil, err)

	// Check that the new generation was not replaced.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("foobar", contents)
}
