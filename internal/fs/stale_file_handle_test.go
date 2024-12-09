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

package fs_test

import (
	"errors"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type StaleHandleTest struct {
	// fsTest has f1 *osFile and f2 *osFile which we will reuse here.
	f3 *os.File
	fsTest
}

func init() {
	RegisterTestSuite(&StaleHandleTest{})
}

func (t *StaleHandleTest) SetUpTestSuite() {
	t.serverCfg.NewConfig = &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			PreconditionErrors: true,
		},
		MetadataCache: cfg.MetadataCacheConfig{
			TtlSecs: 0,
		},
	}
	t.fsTest.SetUpTestSuite()
}

func (t *StaleHandleTest) TearDown() {
	// Close t.f3 in case of test failure.
	if t.f3 != nil {
		AssertEq(nil, t.f3.Close())
		t.f3 = nil
	}

	// fsTest Cleanups to clean up mntDir and close t.f1 and t.f2.
	t.fsTest.TearDown()
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (t *StaleHandleTest) createLocalFile(fileName string) (filePath string, f *os.File) {
	// Creating a file shouldn't create file on GCS.
	filePath = path.Join(mntDir, fileName)
	_, err := os.Stat(mntDir)
	AssertEq(nil, err)

	f, err = os.Create(filePath)

	AssertEq(nil, err)
	t.validateObjectNotFoundErr(fileName)
	return
}

func (t *StaleHandleTest) validateObjectNotFoundErr(fileName string) {
	var notFoundErr *gcs.NotFoundError
	_, err := storageutil.ReadObject(ctx, bucket, fileName)

	ExpectTrue(errors.As(err, &notFoundErr))
}

func (t *StaleHandleTest) validateNoFileOrDirError(filename string) {
	_, err := os.Stat(path.Join(mntDir, filename))

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
}

func (t *StaleHandleTest) closeLocalFile(f **os.File) error {
	err := (*f).Close()
	*f = nil
	return err
}

func (t *StaleHandleTest) readDirectory(dirPath string) (entries []os.DirEntry) {
	entries, err := os.ReadDir(dirPath)

	AssertEq(nil, err)
	return
}

func (t *StaleHandleTest) verifyLocalFileEntry(entry os.DirEntry, fileName string, size int) {
	AssertEq(false, entry.IsDir())
	AssertEq(fileName, entry.Name())

	fileInfo, err := entry.Info()

	AssertEq(nil, err)
	AssertEq(size, fileInfo.Size())
}

func (t *StaleHandleTest) closeFileAndValidateObjectContents(f **os.File, fileName string, contents string) {
	err := t.closeLocalFile(f)
	AssertEq(nil, err)
	t.validateObjectContents(fileName, contents)
}

func (t *StaleHandleTest) validateObjectContents(fileName string, contents string) {
	contentBytes, err := storageutil.ReadObject(ctx, bucket, fileName)
	AssertEq(nil, err)
	ExpectEq(contents, string(contentBytes))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *StaleHandleTest) StatOnUnlinkedLocalFile() {
	// Create a local file.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)
	// unlink the local file.
	err := os.Remove(filePath)
	AssertEq(nil, err)

	// Stat the local file and validate error.
	t.validateNoFileOrDirError(FileName)

	// Validate that closing local unlinked file throws stale NFS file handle
	// error and the object is not created on GCS.
	err = t.closeLocalFile(&t.f1)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	t.validateObjectNotFoundErr(FileName)
}

func (t *StaleHandleTest) TestReadDirContainingUnlinkedLocalFiles() {
	// Create local files.
	_, t.f1 = t.createLocalFile(FileName + "1")
	_, t.f2 = t.createLocalFile(FileName + "2")
	var filePath3 string
	filePath3, t.f3 = t.createLocalFile(FileName + "3")
	// Unlink local file 3
	err := os.Remove(filePath3)
	AssertEq(nil, err)

	// Attempt to list mntDir.
	entries := t.readDirectory(mntDir)

	// Verify unlinked entries are not listed.
	AssertEq(2, len(entries))
	t.verifyLocalFileEntry(entries[0], FileName+"1", 0)
	t.verifyLocalFileEntry(entries[1], FileName+"2", 0)
	// Close the local files.
	t.closeFileAndValidateObjectContents(&t.f1, FileName+"1", "")
	t.closeFileAndValidateObjectContents(&t.f2, FileName+"2", "")
	// Validate closing unlinked local file throws stale NFS file handle error.
	err = t.closeLocalFile(&t.f3)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Verify unlinked file is not written to GCS
	t.validateObjectNotFoundErr(FileName + "3")
}

func (t *StaleHandleTest) TestUnlinkOfLocalFile() {
	// Create empty local file.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)

	// Attempt to unlink local file.
	err := os.Remove(filePath)

	// Verify unlink operation succeeds.
	AssertEq(nil, err)
	t.validateNoFileOrDirError(FileName)
	// Validate closing unlinked local file throws stale NFS file handle error.
	err = t.closeLocalFile(&t.f1)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Verify unlinked object is not present on GCS.
	t.validateObjectNotFoundErr(FileName)
}

func (t *StaleHandleTest) TestWriteOnUnlinkedLocalFileSucceeds() {
	// Create local file and unlink.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)
	err := os.Remove(filePath)
	// Verify unlink operation succeeds.
	AssertEq(nil, err)
	t.validateNoFileOrDirError(FileName)

	// Write to unlinked local file.
	_, err = t.f1.WriteString(FileContents)

	AssertEq(nil, err)
	// Validate closing unlinked local file throws stale NFS file handle error.
	err = t.closeLocalFile(&t.f1)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Verify unlinked file is not written to GCS
	t.validateObjectNotFoundErr(FileName)
}

func (t *StaleHandleTest) TestSyncOnUnlinkedLocalFile() {
	// Create local file.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)

	// Attempt to unlink local file.
	err := os.Remove(filePath)

	// Verify unlink operation succeeds.
	AssertEq(nil, err)
	t.validateNoFileOrDirError(FileName)
	// Validate sync operation throws stale NFS file handle error
	// and does not write to GCS after unlink.
	err = t.f1.Sync()
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	t.validateObjectNotFoundErr(FileName)
	// Validate closing unlinked local file also throws stale NFS file handle error.
	err = t.closeLocalFile(&t.f1)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Verify unlinked file is not present on GCS.
	t.validateObjectNotFoundErr(FileName)
}

func (t *StaleHandleTest) TestRmDirOfDirectoryContainingGCSAndLocalFiles() {
	// Create explicit directory with one synced and one local file.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"explicit/":    "",
				"explicit/foo": "",
			}))
	_, t.f1 = t.createLocalFile("explicit/" + explicitLocalFileName)

	// Attempt to remove explicit directory.
	err := os.RemoveAll(path.Join(mntDir, "explicit"))

	// Verify rmDir operation succeeds.
	AssertEq(nil, err)
	t.validateNoFileOrDirError("explicit/" + explicitLocalFileName)
	t.validateNoFileOrDirError("explicit/foo")
	t.validateNoFileOrDirError("explicit")
	// Validate writing content to unlinked local file does not throw error
	_, err = t.f1.WriteString(FileContents)
	AssertEq(nil, err)
	// Validate close file throws stale NFS file handle error and does not create
	// object on GCS.
	err = t.closeLocalFile(&t.f1)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	t.validateObjectNotFoundErr("explicit/" + explicitLocalFileName)
	// Validate synced files are also deleted.
	t.validateObjectNotFoundErr("explicit/foo")
	t.validateObjectNotFoundErr("explicit/")
}

func (t *StaleHandleTest) TestRmDirOfDirectoryContainingOnlyLocalFiles() {
	// Create a directory with two local files.
	err := os.Mkdir(path.Join(mntDir, "explicit"), dirPerms)
	AssertEq(nil, err)
	_, t.f1 = t.createLocalFile("explicit/" + explicitLocalFileName)
	_, t.f2 = t.createLocalFile("explicit/" + FileName)

	// Attempt to remove explicit directory.
	err = os.RemoveAll(path.Join(mntDir, "explicit"))

	// Verify rmDir operation succeeds.
	AssertEq(nil, err)
	t.validateNoFileOrDirError("explicit/" + explicitLocalFileName)
	t.validateNoFileOrDirError("explicit/" + FileName)
	t.validateNoFileOrDirError("explicit")
	// Validate closing local unlinked files throw stale NFS file handle errors
	// and do not create objects on GCS.
	err = t.closeLocalFile(&t.f1)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	t.validateObjectNotFoundErr("explicit/" + explicitLocalFileName)
	err = t.closeLocalFile(&t.f2)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	t.validateObjectNotFoundErr("explicit/" + FileName)
	// Validate directory is also deleted.
	t.validateObjectNotFoundErr("explicit/")
}

func (t *StaleHandleTest) TestReadSymlinkForDeletedLocalFile() {
	// Create a local file.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)
	// Writing contents to local file shouldn't create file on GCS.
	_, err := t.f1.Write([]byte(FileContents))
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)
	// Create the symlink.
	symlinkName := path.Join(mntDir, "bar")
	err = os.Symlink(filePath, symlinkName)
	AssertEq(nil, err)
	// Read the link.
	target, err := os.Readlink(symlinkName)
	AssertEq(nil, err)
	ExpectEq(filePath, target)
	// Attempt to unlink local file.
	err = os.Remove(filePath)
	// Verify unlink operation succeeds.
	AssertEq(nil, err)
	// Validate flushing local unlinked file throws stale NFS file handle error
	// and does not create object on GCS.
	err = t.closeLocalFile(&t.f1)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	t.validateObjectNotFoundErr(FileName)

	// Reading symlink should fail.
	_, err = os.Stat(symlinkName)

	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
}

func (t *StaleHandleTest) SyncClobberedLocalInode() {
	// Create a local file.
	_, t.f1 = t.createLocalFile("foo")
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
	// Validate closing the file also throws error
	err = t.closeLocalFile(&t.f1)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("foobar", string(contents))
}

func (t *StaleHandleTest) ReadingFileAfterObjectClobberedRemotelyFailsWithStaleHandle() {
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

func (t *StaleHandleTest) WritingToFileAfterObjectClobberedRemotelyFailsWithStaleHandle() {
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

func (t *StaleHandleTest) SyncingFileAfterObjectClobberedRemotelyFailsWithStaleHandle() {
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

	err = t.f1.Sync()

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Validate closing the file also throws error
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

func (t *StaleHandleTest) SyncingFileAfterObjectDeletedFailsWithStaleHandle() {
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

func (t *StaleHandleTest) WritingToFileAfterObjectDeletedFailsWithStaleHandle() {
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

	_, err = t.f1.Write([]byte("taco"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Attempt to sync to file should not result in error as content written is
	// nil.
	err = t.f1.Sync()
	AssertEq(nil, err)
}

func (t *StaleHandleTest) SyncingLocalInodeAfterObjectDeletedFailsWithStaleHandle() {
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

func (t *StaleHandleTest) SyncingFileAfterObjectRenamedFailsWithStaleHandle() {
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

func (t *StaleHandleTest) WritingToFileAfterObjectRenamedFailsWithStaleHandle() {
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

	_, err = t.f1.Write([]byte("taco"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("stale NFS file handle")))
	// Attempt to sync to file should not result in error as content written is
	// nil.
	err = t.f1.Sync()
	AssertEq(nil, err)
}
