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

// A collection of tests for a file system backed by a GCS bucket, where in
// most cases we interact with the file system directly for creating and
// mofiying files (rather than through the side channel of the GCS bucket
// itself).
//
// These tests are registered by RegisterFSTests.

package fstesting

import (
	"io"
	"io/ioutil"
	"os"
	"path"

	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

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

func (t *modesTest) AppendMode_ReadOnly() {
	AssertTrue(false, "TODO")
}

func (t *modesTest) AppendMode_WriteOnly() {
	AssertTrue(false, "TODO")
}

func (t *modesTest) AppendMode_ReadWrite() {
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

	// Write to the middle of the file using File.WriteAt.
	_, err = f.WriteAt([]byte("111"), 4)
	AssertEq(nil, err)

	// Write well past the end of the file using File.WriteAt.
	_, err = f.WriteAt([]byte("333"), 100)
	AssertEq(nil, err)

	// Check the size now.
	fi, err := f.Stat()
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222333"), fi.Size())

	// Read some contents with Seek and Read.
	_, err = f.Seek(4, 0)
	AssertEq(nil, err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(f, buf)

	AssertEq(nil, err)
	ExpectEq("111r", string(buf))

	// Read the full contents with ReadAt.
	buf = make([]byte, len(contents)+len("222333"))
	_, err = f.ReadAt(buf, 0)

	AssertEq(nil, err)
	ExpectEq("taco111ritoenchilada222333", string(buf))

	// Close the file.
	AssertEq(nil, f.Close())
	f = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("taco111ritoenchilada222333", string(fileContents))
}

////////////////////////////////////////////////////////////////////////
// Read/write interaction
////////////////////////////////////////////////////////////////////////

type readWriteTest struct {
	fsTest
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

func (t *readWriteTest) ListingsAreCached() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) AdditionsAreCached() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) RemovalsAreCached() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) BufferedWritesFlushedOnUnmount() {
	AssertTrue(false, "TODO")
}
