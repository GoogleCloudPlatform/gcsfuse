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

	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
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
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR, 0700)
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("no such file")))
}

func (t *openTest) NonExistent_CreateFlagSet() {
	// Open the file.
	f, err := os.OpenFile(
		path.Join(t.mfs.Dir(), "foo"),
		os.O_RDWR|os.O_CREATE,
		0700)

	AssertEq(nil, err)
	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Write some contents.
	_, err = f.Write([]byte("012"))
	AssertEq(nil, err)

	// Read some contents with Seek and Read.
	_, err = f.Seek(1, 0)
	AssertEq(nil, err)

	buf := make([]byte, 2)
	_, err = io.ReadFull(f, buf)

	AssertEq(nil, err)
	ExpectEq("12", string(buf))

	// Close the file.
	AssertEq(nil, f.Close())
	f = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("012", string(fileContents))
}

func (t *openTest) ExistingFile() {
	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR, 0)
	AssertEq(nil, err)

	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Write to the start of the file using File.Write.
	_, err = f.Write([]byte("012"))
	AssertEq(nil, err)

	// Read some contents with Seek and Read.
	_, err = f.Seek(2, 0)
	AssertEq(nil, err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(f, buf)

	AssertEq(nil, err)
	ExpectEq("2obu", string(buf))

	// Close the file.
	AssertEq(nil, f.Close())
	f = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("012oburritoenchilada", string(fileContents))
}

func (t *openTest) ExistingFile_Truncate() {
	// Create a file.
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte("blahblahblah"),
			os.FileMode(0644)))

	// Open the file.
	f, err := os.OpenFile(
		path.Join(t.mfs.Dir(), "foo"),
		os.O_RDWR|os.O_TRUNC,
		0)

	AssertEq(nil, err)

	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// The file should be empty.
	fi, err := f.Stat()
	AssertEq(nil, err)
	ExpectEq(0, fi.Size())

	// Write to the start of the file using File.Write.
	_, err = f.Write([]byte("012"))
	AssertEq(nil, err)

	// Read the contents.
	_, err = f.Seek(0, 0)
	AssertEq(nil, err)

	contentsSlice, err := ioutil.ReadAll(f)
	AssertEq(nil, err)
	ExpectEq("012", string(contentsSlice))

	// Close the file.
	AssertEq(nil, f.Close())
	f = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("012", string(fileContents))
}

func (t *openTest) AlreadyOpenedFile() {
	AssertTrue(false, "TODO")
}

func (t *openTest) OpenReadOnlyFileForWrite() {
	AssertTrue(false, "TODO")
}

func (t *openTest) CreateWithinSubDirectory() {
	AssertTrue(false, "TODO")
}

func (t *openTest) OpenWithinSubDirectory() {
	AssertTrue(false, "TODO")
}

func (t *openTest) TruncateWithinSubDirectory() {
	AssertTrue(false, "TODO")
}

////////////////////////////////////////////////////////////////////////
// Modes
////////////////////////////////////////////////////////////////////////

type modesTest struct {
	fsTest
}

func (t *modesTest) ReadOnlyMode() {
	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDONLY, 0)
	AssertEq(nil, err)

	defer func() {
		ExpectEq(nil, f.Close())
	}()

	// Read its contents.
	fileContents, err := ioutil.ReadAll(f)
	AssertEq(nil, err)
	ExpectEq(contents, string(fileContents))

	// Attempt to write.
	n, err := f.Write([]byte("taco"))

	AssertEq(0, n)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))
}

func (t *modesTest) WriteOnlyMode() {
	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_WRONLY, 0)
	AssertEq(nil, err)

	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Reading should fail.
	_, err = ioutil.ReadAll(f)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))

	// Write to the start of the file using File.Write.
	_, err = f.Write([]byte("000"))
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = f.WriteAt([]byte("111"), 4)
	AssertEq(nil, err)

	// Seek and write past the end of the file.
	_, err = f.Seek(int64(len(contents)), 0)
	AssertEq(nil, err)

	_, err = f.Write([]byte("222"))
	AssertEq(nil, err)

	// Check the size now.
	fi, err := f.Stat()
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Close the file.
	AssertEq(nil, f.Close())
	f = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(fileContents))
}

func (t *modesTest) ReadWriteMode() {
	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR, 0)
	AssertEq(nil, err)

	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Write to the start of the file using File.Write.
	_, err = f.Write([]byte("000"))
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = f.WriteAt([]byte("111"), 4)
	AssertEq(nil, err)

	// Seek and write past the end of the file.
	_, err = f.Seek(int64(len(contents)), 0)
	AssertEq(nil, err)

	_, err = f.Write([]byte("222"))
	AssertEq(nil, err)

	// Check the size now.
	fi, err := f.Stat()
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Read some contents with Seek and Read.
	_, err = f.Seek(4, 0)
	AssertEq(nil, err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(f, buf)

	AssertEq(nil, err)
	ExpectEq("111r", string(buf))

	// Read the full contents with ReadAt.
	buf = make([]byte, len(contents)+len("222"))
	_, err = f.ReadAt(buf, 0)

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(buf))

	// Close the file.
	AssertEq(nil, f.Close())
	f = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(fileContents))
}

func (t *modesTest) AppendMode_SeekAndWrite() {
	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR|os.O_APPEND, 0)
	AssertEq(nil, err)

	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Write using File.Write. This should go to the end of the file regardless
	// of whether we Seek somewhere else first.
	_, err = f.Seek(1, 0)
	AssertEq(nil, err)

	_, err = f.Write([]byte("222"))
	AssertEq(nil, err)

	// The seek position should have been updated.
	off, err := getFileOffset(f)
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222"), off)

	// Check the size now.
	fi, err := f.Stat()
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Read the full contents with ReadAt.
	buf := make([]byte, 1024)
	n, err := f.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq(contents+"222", string(buf[:n]))

	// Read the full contents with another file handle.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq(contents+"222", string(fileContents))
}

func (t *modesTest) AppendMode_WriteAt() {
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
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR|os.O_APPEND, 0)
	AssertEq(nil, err)

	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Seek somewhere in the file.
	_, err = f.Seek(1, 0)
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = f.WriteAt([]byte("111"), 4)
	AssertEq(nil, err)

	// The seek position should have been unaffected.
	off, err := getFileOffset(f)
	AssertEq(nil, err)
	ExpectEq(1, off)

	// Check the size now.
	fi, err := f.Stat()
	AssertEq(nil, err)

	if isLinux {
		ExpectEq(len(contents+"111"), fi.Size())
	} else {
		ExpectEq(len(contents), fi.Size())
	}

	// Read the full contents with ReadAt.
	buf := make([]byte, 1024)
	n, err := f.ReadAt(buf, 0)

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
	// Linux's support for pwrite is buggy; the pwrite(2) man page says this:
	//
	//     POSIX requires that opening a file with the O_APPEND flag should have
	//     no affect on the location at which pwrite() writes data.  However, on
	//     Linux,  if  a  file  is opened with O_APPEND, pwrite() appends data to
	//     the end of the file, regardless of the value of offset.
	//
	isLinux := (runtime.GOOS == "linux")

	// Open a file.
	f, err := os.OpenFile(
		path.Join(t.mfs.Dir(), "foo"),
		os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0600)

	AssertEq(nil, err)

	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Write three bytes.
	n, err := f.Write([]byte("111"))
	AssertEq(nil, err)
	AssertEq(3, n)

	// Write at offset six.
	n, err = f.WriteAt([]byte("222"), 6)
	AssertEq(nil, err)
	AssertEq(3, n)

	// The seek position should have been unaffected.
	off, err := getFileOffset(f)
	AssertEq(nil, err)
	ExpectEq(3, off)

	// Read the full contents of the file.
	contents, err := ioutil.ReadFile(f.Name())
	AssertEq(nil, err)

	if isLinux {
		ExpectEq("111222", string(contents))
	} else {
		ExpectEq("111\x00\x00\x00222", string(contents))
	}
}

func (t *modesTest) ReadFromWriteOnlyFile() {
	AssertTrue(false, "TODO")
}

func (t *modesTest) WriteToReadOnlyFile() {
	AssertTrue(false, "TODO")
}

////////////////////////////////////////////////////////////////////////
// Directory interaction
////////////////////////////////////////////////////////////////////////

type directoryTest struct {
	fsTest
}

func (t *directoryTest) Stat() {
	AssertTrue(false, "TODO")
}

func (t *directoryTest) ReadDir() {
	AssertTrue(false, "TODO")
}

func (t *directoryTest) Mkdir_OneLevel() {
	AssertTrue(false, "TODO")
}

func (t *directoryTest) Mkdir_TwoLevels() {
	AssertTrue(false, "TODO")
}

func (t *directoryTest) Mkdir_AlreadyExists() {
	AssertTrue(false, "TODO")
}

func (t *directoryTest) Mkdir_IntermediateIsFile() {
	AssertTrue(false, "TODO")
}

func (t *directoryTest) Mkdir_IntermediateIsNonExistent() {
	AssertTrue(false, "TODO")
}

func (t *directoryTest) Rmdir_NonEmpty() {
	AssertTrue(false, "TODO")
}

func (t *directoryTest) Rmdir_Empty() {
	AssertTrue(false, "TODO")
}

func (t *directoryTest) Rmdir_OpenedForReading() {
	AssertTrue(false, "TODO")
}

func (t *directoryTest) CreateHardLink() {
	AssertTrue(false, "TODO")
}

func (t *directoryTest) CreateSymlink() {
	AssertTrue(false, "TODO")
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
	f, err := os.Create(path.Join(t.mfs.Dir(), "foo"))
	t.toClose = append(t.toClose, f)
	AssertEq(nil, err)

	// Make it 4 bytes long.
	err = f.Truncate(4)
	AssertEq(nil, err)

	// Write the range [2, 6).
	n, err = f.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(f)
	AssertEq(nil, err)
	ExpectEq("\x00\x00taco", string(contents))
}

func (t *fileTest) WriteStartsAtEndOfFile() {
	var err error
	var n int

	// Create a file.
	f, err := os.Create(path.Join(t.mfs.Dir(), "foo"))
	t.toClose = append(t.toClose, f)
	AssertEq(nil, err)

	// Make it 2 bytes long.
	err = f.Truncate(2)
	AssertEq(nil, err)

	// Write the range [2, 6).
	n, err = f.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(f)
	AssertEq(nil, err)
	ExpectEq("\x00\x00taco", string(contents))
}

func (t *fileTest) WriteStartsPastEndOfFile() {
	var err error
	var n int

	// Create a file.
	f, err := os.Create(path.Join(t.mfs.Dir(), "foo"))
	t.toClose = append(t.toClose, f)
	AssertEq(nil, err)

	// Write the range [2, 6).
	n, err = f.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(f)
	AssertEq(nil, err)
	ExpectEq("\x00\x00taco", string(contents))
}

func (t *fileTest) WriteAtDoesntChangeOffset_NotAppendMode() {
	var err error
	var n int

	// Create a file.
	f, err := os.Create(path.Join(t.mfs.Dir(), "foo"))
	t.toClose = append(t.toClose, f)
	AssertEq(nil, err)

	// Make it 16 bytes long.
	err = f.Truncate(16)
	AssertEq(nil, err)

	// Seek to offset 4.
	_, err = f.Seek(4, 0)
	AssertEq(nil, err)

	// Write the range [10, 14).
	n, err = f.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// We should still be at offset 4.
	offset, err := getFileOffset(f)
	AssertEq(nil, err)
	ExpectEq(4, offset)
}

func (t *fileTest) WriteAtDoesntChangeOffset_AppendMode() {
	var err error
	var n int

	// Create a file in append mode.
	f, err := os.OpenFile(
		path.Join(t.mfs.Dir(), "foo"),
		os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0600)

	t.toClose = append(t.toClose, f)
	AssertEq(nil, err)

	// Make it 16 bytes long.
	err = f.Truncate(16)
	AssertEq(nil, err)

	// Seek to offset 4.
	_, err = f.Seek(4, 0)
	AssertEq(nil, err)

	// Write the range [10, 14).
	n, err = f.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// We should still be at offset 4.
	offset, err := getFileOffset(f)
	AssertEq(nil, err)
	ExpectEq(4, offset)
}

func (t *fileTest) ReadsPastEndOfFile() {
	var err error
	var n int
	buf := make([]byte, 1024)

	// Create a file.
	f, err := os.Create(path.Join(t.mfs.Dir(), "foo"))
	t.toClose = append(t.toClose, f)
	AssertEq(nil, err)

	// Give it some contents.
	n, err = f.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Read a range overlapping EOF.
	n, err = f.ReadAt(buf[:4], 2)
	AssertEq(io.EOF, err)
	ExpectEq(2, n)
	ExpectEq("co", string(buf[:n]))

	// Read a range starting at EOF.
	n, err = f.ReadAt(buf[:4], 4)
	AssertEq(io.EOF, err)
	ExpectEq(0, n)
	ExpectEq("", string(buf[:n]))

	// Read a range starting past EOF.
	n, err = f.ReadAt(buf[:4], 100)
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
	f, err := os.OpenFile(fileName, os.O_RDWR, 0)
	t.toClose = append(t.toClose, f)
	AssertEq(nil, err)

	// Truncate it.
	err = f.Truncate(2)
	AssertEq(nil, err)

	// Stat it.
	fi, err := f.Stat()
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
	f, err := os.OpenFile(fileName, os.O_RDWR, 0)
	t.toClose = append(t.toClose, f)
	AssertEq(nil, err)

	// Truncate it.
	err = f.Truncate(4)
	AssertEq(nil, err)

	// Stat it.
	fi, err := f.Stat()
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
	f, err := os.OpenFile(fileName, os.O_RDWR, 0)
	t.toClose = append(t.toClose, f)
	AssertEq(nil, err)

	// Truncate it.
	err = f.Truncate(6)
	AssertEq(nil, err)

	// Stat it.
	fi, err := f.Stat()
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
	f, err := os.Create(path.Join(t.mfs.Dir(), "foo"))
	t.toClose = append(t.toClose, f)
	AssertEq(nil, err)

	// Give it some contents.
	n, err = f.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Seek and overwrite.
	off, err := f.Seek(1, 0)
	AssertEq(nil, err)
	AssertEq(1, off)

	n, err = f.Write([]byte("xx"))
	AssertEq(nil, err)
	AssertEq(2, n)

	// Read full the contents of the file.
	n, err = f.ReadAt(buf, 0)
	AssertEq(io.EOF, err)
	ExpectEq("txxo", string(buf[:n]))
}

func (t *fileTest) Stat() {
	var err error
	var n int

	// Create a file.
	f, err := os.Create(path.Join(t.mfs.Dir(), "foo"))
	t.toClose = append(t.toClose, f)
	AssertEq(nil, err)

	// Give it some contents.
	t.advanceTime()
	writeTime := t.clock.Now()

	n, err = f.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	t.advanceTime()

	// Stat it.
	fi, err := f.Stat()
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
	fileName := path.Join(t.mfs.Dir(), "foo")

	// Create and open a file.
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0600)
	t.toClose = append(t.toClose, f)
	AssertEq(nil, err)

	// Write some data into it.
	n, err := f.Write([]byte("taco"))
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
	fi, err := f.Stat()

	AssertEq(nil, err)
	ExpectEq(4, fi.Size())
	ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)

	// The contents should still be available.
	buf := make([]byte, 1024)
	n, err = f.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	AssertEq(4, n)
	ExpectEq("taco", string(buf[:4]))

	// Writing should still work, too.
	n, err = f.Write([]byte("burrito"))
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
	AssertTrue(false, "TODO")
}

func (t *fileTest) Chmod() {
	AssertTrue(false, "TODO")
}

func (t *fileTest) Chtimes() {
	AssertTrue(false, "TODO")
}

func (t *fileTest) Sync() {
	// Make sure to test that the content shows up in the bucket.
	AssertTrue(false, "TODO")
}

func (t *fileTest) Close() {
	// Make sure to test that the content shows up in the bucket.
	AssertTrue(false, "TODO")
}

func (t *fileTest) BufferedWritesFlushedOnUnmount() {
	AssertTrue(false, "TODO")
}
