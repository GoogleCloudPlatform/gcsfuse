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
	"os"
	"path"

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

	AssertTrue(false, "TODO")
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
