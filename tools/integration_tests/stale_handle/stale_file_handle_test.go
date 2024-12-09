// Copyright 2024 Google LLC
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

package stale_handle

import (
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

// This test-suite contains parallelizable test-case. Use "-parallel n" to limit
// the degree of parallelism. By default it uses GOMAXPROCS.
// Ref: https://stackoverflow.com/questions/24375966/does-go-test-run-unit-tests-concurrently
type staleFileHandleTest struct{}

func (s *staleFileHandleTest) Setup(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func (s *staleFileHandleTest) Teardown(t *testing.T) {}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func Test_SyncingClobberedLocalInodeFailsWithStaleHandle(t *testing.T) {
	t.Parallel() // Mark the test parallelizable.
	testCaseDir := "Test_SyncingClobberedLocalInodeFailsWithStaleHandle"
	targetDir := path.Join(testDirPath, testCaseDir)
	targetDirPath := setup.SetupTestDirectory(targetDir)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, targetDirPath, FileName1, t)
	// Dirty the file by giving it some contents.
	operations.WriteWithoutClose(fh, FileContents, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t)

	// Replace the underlying object with a new generation.
	CreateObjectInGCSTestDir(ctx, storageClient, testDirName, FileName1, GCSFileContent, t)

	operations.SyncFileShouldThrowStaleHandleError(fh, t)
	// Closing the file should also throw error
	operations.CloseFileShouldThrowStaleHandleError(fh, t)
	ValidateObjectContentsFromGCS(ctx, storageClient, targetDir, FileName1, GCSFileContent, t)
}

func Test_ReadingFileAfterObjectClobberedRemotelyFailsWithStaleHandle(t *testing.T) {
	// Create an object on bucket
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	AssertEq(nil, err)

	// Open the read handle
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_RDONLY|syscall.O_DIRECT, filePerms)
	AssertEq(nil, err)
	// Replace the underlying object with a new generation.
	_, err = storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("foobar"))
	AssertEq(nil, err)
	// Attempt to read the file should result in stale NFS file handle error.
	buffer := make([]byte, 6)
	_, err = t.f1.Read(buffer)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("foobar", string(contents))
}

func Test_WritingToFileAfterObjectClobberedRemotelyFailsWithStaleHandle(t *testing.T) {
	// Create an object on bucket
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	AssertEq(nil, err)

	// Open file handle to write
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_WRONLY|syscall.O_DIRECT, filePerms)
	AssertEq(nil, err)
	// Replace the underlying object with a new generation.
	_, err = storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("foobar"))
	AssertEq(nil, err)
	// Attempt to write to file should result in stale NFS file handle error.
	_, err = t.f1.Write([]byte("taco"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Attempt to sync to file should not result in error as content written is
	// nil.
	err = t.f1.Sync()
	AssertEq(nil, err)
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("foobar", string(contents))
}

func Test_SyncingFileAfterObjectClobberedRemotelyFailsWithStaleHandle(t *testing.T) {
	// Create an object on bucket
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	AssertEq(nil, err)

	// Open file handle to write
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_WRONLY|syscall.O_DIRECT, filePerms)
	AssertEq(nil, err)
	// Dirty the file by giving it some contents.
	n, err := t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)
	// Replace the underlying object with a new generation.
	_, err = storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("foobar"))
	AssertEq(nil, err)
	// Attempt to sync the file should result in clobbered error.
	err = t.f1.Sync()

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Validate closing the file also throws stale NFS file handle error
	err = t.f1.Close()
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file
	t.f1 = nil
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("foobar", string(contents))
}

func Test_SyncingFileAfterObjectDeletedFailsWithStaleHandle(t *testing.T) {
	// Create an object on bucket
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	AssertEq(nil, err)

	// Open file handle to write
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_WRONLY|syscall.O_DIRECT, filePerms)
	AssertEq(nil, err)
	// Dirty the file by giving it some contents.
	n, err := t.f1.Write([]byte("foobar"))
	AssertEq(nil, err)
	AssertEq(6, n)
	// Delete the object.
	err = os.Remove(t.f1.Name())
	AssertEq(nil, err)
	// Attempt to write to file should not give any error.
	n, err = t.f1.Write([]byte("taco"))
	AssertEq(4, n)
	AssertEq(nil, err)
	// Attempt to sync the file should result in clobbered error.
	err = t.f1.Sync()

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Closing file should also give error
	err = t.f1.Close()
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file
	t.f1 = nil
}

func Test_WritingToFileAfterObjectDeletedFailsWithStaleHandle(t *testing.T) {
	// Create an object on bucket
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	AssertEq(nil, err)

	// Open file handle to write
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_WRONLY|syscall.O_DIRECT, filePerms)
	AssertEq(nil, err)
	// Delete the object.
	err = os.Remove(t.f1.Name())
	AssertEq(nil, err)
	// Attempt to write to file should result in stale NFS file handle error.
	_, err = t.f1.Write([]byte("taco"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Attempt to sync to file should not result in error as content written is
	// nil.
	err = t.f1.Sync()
	AssertEq(nil, err)
}

func Test_SyncingLocalInodeAfterObjectDeletedFailsWithStaleHandle(t *testing.T) {
	// Create a local file.
	_, t.f1 = t.createLocalFile("foo")

	// Delete the object.
	err := os.Remove(t.f1.Name())
	AssertEq(nil, err)
	// Attempt to write to file should not give any error as for local inode data
	// is written to buffer, and we don't check for object on GCS.
	n, err := t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)
	// Attempt to sync the file should result in clobbered error.
	err = t.f1.Sync()

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Closing file should also give error
	err = t.f1.Close()
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file
	t.f1 = nil
}

func Test_SyncingFileAfterObjectRenamedFailsWithStaleHandle(t *testing.T) {
	// Create an object on bucket
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	AssertEq(nil, err)

	// Open file handle to write
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_WRONLY|syscall.O_DIRECT, filePerms)
	AssertEq(nil, err)
	// Dirty the file by giving it some contents.
	n, err := t.f1.Write([]byte("foobar"))
	AssertEq(nil, err)
	AssertEq(6, n)
	// Rename the object.
	err = os.Rename(t.f1.Name(), path.Join(mntDir, "bar"))
	AssertEq(nil, err)
	// Attempt to write to file should not give any error.
	n, err = t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)
	// Attempt to sync the file should result in clobbered error.
	err = t.f1.Sync()

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Closing file should also give error
	err = t.f1.Close()
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file
	t.f1 = nil
}

func Test_WritingToFileAfterObjectRenamedFailsWithStaleHandle(t *testing.T) {
	// Create an object on bucket
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	AssertEq(nil, err)

	// Open file handle to write
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_WRONLY|syscall.O_DIRECT, filePerms)
	AssertEq(nil, err)
	// Rename the object.
	err = os.Rename(t.f1.Name(), path.Join(mntDir, "bar"))
	AssertEq(nil, err)
	// Attempt to write to file should result in stale NFS file handle error.
	_, err = t.f1.Write([]byte("taco"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Attempt to sync to file should not result in error as content written is
	// nil.
	err = t.f1.Sync()
	AssertEq(nil, err)
}
