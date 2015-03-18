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

////////////////////////////////////////////////////////////////////////
// Read/write interaction
////////////////////////////////////////////////////////////////////////

type readWriteTest struct {
	fsTest
}

func (t *readWriteTest) Mkdir_OneLevel() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) Mkdir_TwoLevels() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) Mkdir_AlreadyExists() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) Mkdir_IntermediateIsFile() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) Mkdir_IntermediateIsNonExistent() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) WritePastEndOfFile() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) Seek() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) Stat() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) Sync() {
	// Make sure to test that the content shows up in the bucket.
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) Close() {
	// Make sure to test that the content shows up in the bucket.
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) Truncate() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) CreateEmptyDirectory() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) CreateWithinSubDirectory() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenWithinSubDirectory() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) TruncateWithinSubDirectory() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) ReadFromWriteOnlyFile() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) WriteToReadOnlyFile() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) StatUnopenedFile() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) LstatUnopenedFile() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) BufferedWritesFlushedOnUnmount() {
	AssertTrue(false, "TODO")
}
