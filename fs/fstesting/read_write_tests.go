// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)
//
// A collection of tests for a file system backed by a GCS bucket, where in
// most cases we interact with the file system directly for creating and
// mofiying files (rather than through the side channel of the GCS bucket
// itself).
//
// These tests are registered by RegisterFSTests.

package fstesting

import (
	"io/ioutil"
	"math"
	"os"
	"path"
	"time"

	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Read-write interaction
////////////////////////////////////////////////////////////////////////

type readWriteTest struct {
	fsTest
}

func (t *readWriteTest) OpenNonExistent_CreateFlagNotSet() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenNonExistent_ReadOnly() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenNonExistent_WriteOnly() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenNonExistent_ReadWrite() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenExistingFile_ReadOnly() {
	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file for reading.
	f, err := os.Open(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	defer func() {
		ExpectEq(nil, f.Close())
	}()

	// Check its vitals.
	ExpectEq(path.Join(t.mfs.Dir(), "foo"), f.Name())

	fi, err := f.Stat()
	ExpectEq("foo", fi.Name())
	ExpectEq(len(contents), fi.Size())
	ExpectEq(os.FileMode(0), fi.Mode() & ^os.ModePerm)
	ExpectLt(math.Abs(time.Since(fi.ModTime()).Seconds()), 10)
	ExpectFalse(fi.IsDir())

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

func (t *readWriteTest) OpenExistingFile_WriteOnly() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenExistingFile_ReadWrite() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) TruncateExistingFile_ReadOnly() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) TruncateExistingFile_WriteOnly() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) TruncateExistingFile_ReadWrite() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenAlreadyOpenedFile_ReadOnly() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenAlreadyOpenedFile_WriteOnly() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenAlreadyOpenedFile_ReadWrite() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenReadOnlyFileForWrite() {
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
