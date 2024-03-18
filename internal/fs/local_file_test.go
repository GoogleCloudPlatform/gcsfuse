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
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/jacobsa/fuse/fusetesting"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////
const FileName = "foo"
const FileName2 = "foo2"
const implicitLocalFileName = "implicitLocalFile"
const explicitLocalFileName = "explicitLocalFile"
const FileContents = "teststring"
const Delta = 30 * time.Minute

type LocalFileTest struct {
	// fsTest has f1 *osFile and f2 *osFile which we will reuse here.
	f3 *os.File
	fsTest
}

func init() {
	RegisterTestSuite(&LocalFileTest{})
}

func (t *LocalFileTest) SetUpTestSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		WriteConfig: config.WriteConfig{
			CreateEmptyFile: false,
		}}
	t.fsTest.SetUpTestSuite()
}

func (t *LocalFileTest) TearDown() {
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
func (t *LocalFileTest) createLocalFile(fileName string) (filePath string, f *os.File) {
	// Creating a file shouldn't create file on GCS.
	filePath = path.Join(mntDir, fileName)
	f, err := os.Create(filePath)

	AssertEq(nil, err)
	t.validateObjectNotFoundErr(fileName)

	return
}

func (t *LocalFileTest) verifyLocalFileEntry(entry os.DirEntry, fileName string, size int) {
	AssertEq(false, entry.IsDir())
	AssertEq(fileName, entry.Name())

	fileInfo, err := entry.Info()
	AssertEq(nil, err)
	AssertEq(size, fileInfo.Size())
}

func (t *LocalFileTest) verifyDirectoryEntry(entry os.DirEntry, dirName string) {
	AssertEq(true, entry.IsDir())
	AssertEq(dirName, entry.Name())
}

func (t *LocalFileTest) readDirectory(dirPath string) (entries []os.DirEntry) {
	entries, err := os.ReadDir(dirPath)
	AssertEq(nil, err)
	return
}

func (t *LocalFileTest) validateObjectNotFoundErr(fileName string) {
	var notFoundErr *gcs.NotFoundError
	_, err := storageutil.ReadObject(ctx, bucket, fileName)

	ExpectTrue(errors.As(err, &notFoundErr))
}

func (t *LocalFileTest) closeLocalFile(f **os.File) error {
	err := (*f).Close()
	*f = nil
	return err
}

func (t *LocalFileTest) closeFileAndValidateObjectContents(f **os.File, fileName string, contents string) {
	err := t.closeLocalFile(f)
	AssertEq(nil, err)
	t.validateObjectContents(fileName, contents)
}

func (t *LocalFileTest) validateObjectContents(fileName string, contents string) {
	contentBytes, err := storageutil.ReadObject(ctx, bucket, fileName)
	AssertEq(nil, err)
	ExpectEq(contents, string(contentBytes))
}

func (t *LocalFileTest) newFileShouldGetSyncedToGCSAtClose(fileName string) {
	// Create a local file.
	_, t.f1 = t.createLocalFile(fileName)
	// Writing contents to local file shouldn't create file on GCS.
	_, err := t.f1.Write([]byte(FileContents))
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(fileName)

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(&t.f1, fileName, FileContents)
}

func (t *LocalFileTest) validateNoFileOrDirError(filename string) {
	_, err := os.Stat(path.Join(mntDir, filename))
	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *LocalFileTest) NewFileShouldNotGetSyncedToGCSTillClose() {
	t.newFileShouldGetSyncedToGCSAtClose(FileName)
}

func (t *LocalFileTest) NewFileUnderExplicitDirectoryShouldNotGetSyncedToGCSTillClose() {
	err := os.Mkdir(path.Join(mntDir, "explicit"), dirPerms)
	AssertEq(nil, err)

	t.newFileShouldGetSyncedToGCSAtClose("explicit/foo")
}

func (t *LocalFileTest) NewFileUnderImplicitDirectoryShouldNotGetSyncedToGCSTillClose() {
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"implicitFoo/bar": "",
			}))

	t.newFileShouldGetSyncedToGCSAtClose("implicitFoo/foo")
}

func (t *LocalFileTest) StatOnLocalFile() {
	// Create a local file.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)

	// Stat the local file.
	fi, err := os.Stat(filePath)
	AssertEq(nil, err)
	ExpectEq(path.Base(filePath), fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(filePerms, fi.Mode())

	// Writing contents to local file shouldn't create file on GCS.
	_, err = t.f1.Write([]byte(FileContents))
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)

	// Stat the local file again to check if new contents are written.
	fi, err = os.Stat(filePath)
	AssertEq(nil, err)
	ExpectEq(path.Base(filePath), fi.Name())
	ExpectEq(10, fi.Size())
	ExpectEq(filePerms, fi.Mode())

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, FileContents)
}

func (t *LocalFileTest) StatOnLocalFileWithConflictingFileNameSuffix() {
	// Create a local file.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)
	// Stat the local file.
	fi, err := os.Stat(filePath + inode.ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(path.Base(filePath)+inode.ConflictingFileNameSuffix, fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(filePerms, fi.Mode())

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, "")
}

func (t *LocalFileTest) StatOnUnlinkedLocalFile() {
	// Create a local file.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)
	// unlink the local file.
	err := os.Remove(filePath)
	AssertEq(nil, err)

	// Stat the local file and validate error.
	t.validateNoFileOrDirError(FileName)

	// Close the file and validate that file is not created on GCS.
	err = t.closeLocalFile(&t.f1)
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)
}

func (t *LocalFileTest) TruncateLocalFile() {
	// Create a local file.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)
	// Writing contents to local file .
	_, err := t.f1.Write([]byte(FileContents))
	AssertEq(nil, err)

	// Stat the file to validate if new contents are written.
	fi, err := os.Stat(filePath)
	AssertEq(nil, err)
	ExpectEq(path.Base(filePath), fi.Name())
	ExpectEq(10, fi.Size())
	ExpectEq(filePerms, fi.Mode())

	// Truncate the file to update the file size.
	err = os.Truncate(filePath, 5)
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)

	// Stat the file to validate if file is truncated correctly.
	fi, err = os.Stat(filePath)
	AssertEq(nil, err)
	ExpectEq(path.Base(filePath), fi.Name())
	ExpectEq(5, fi.Size())
	ExpectEq(filePerms, fi.Mode())

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, "tests")
}

func (t *LocalFileTest) MultipleWritesToLocalFile() {
	// Create a local file.
	_, t.f1 = t.createLocalFile(FileName)

	// Write some contents to file sequentially.
	_, err := t.f1.Write([]byte("string1"))
	AssertEq(nil, err)
	_, err = t.f1.Write([]byte("string2"))
	AssertEq(nil, err)
	_, err = t.f1.Write([]byte("string3"))
	AssertEq(nil, err)
	// File shouldn't get created on GCS.
	t.validateObjectNotFoundErr(FileName)

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, "string1string2string3")
}

func (t *LocalFileTest) RandomWritesToLocalFile() {
	// Create a local file.
	_, t.f1 = t.createLocalFile(FileName)

	// Write some contents to file randomly.
	_, err := t.f1.WriteAt([]byte("string1"), 0)
	AssertEq(nil, err)
	_, err = t.f1.WriteAt([]byte("string2"), 2)
	AssertEq(nil, err)
	_, err = t.f1.WriteAt([]byte("string3"), 3)
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, "stsstring3")
}

func (t *LocalFileTest) TestReadDirWithEmptyLocalFiles() {
	// Create local files.
	_, t.f1 = t.createLocalFile(FileName)
	_, t.f2 = t.createLocalFile(FileName2)

	// Attempt to list mntDir.
	entries := t.readDirectory(mntDir)

	// Verify entries received successfully.
	AssertEq(2, len(entries))
	t.verifyLocalFileEntry(entries[0], FileName, 0)
	t.verifyLocalFileEntry(entries[1], FileName2, 0)
	// Close the local files.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, "")
	t.closeFileAndValidateObjectContents(&t.f2, FileName2, "")
}

func (t *LocalFileTest) TestReadDirWithNonEmptyLocalFile() {
	// Create local files.
	_, t.f1 = t.createLocalFile(FileName)
	_, err := t.f1.WriteString(FileContents)
	AssertEq(nil, err)

	// Attempt to list mntDir.
	entries := t.readDirectory(mntDir)

	// Verify entries received successfully.
	AssertEq(1, len(entries))
	t.verifyLocalFileEntry(entries[0], FileName, 10)
	// Close the local files.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, FileContents)
}

func (t *LocalFileTest) TestReadDirForExplicitDirWithLocalFile() {
	// Create explicit dir with 2 local files.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"explicitFoo/": "",
			}))
	_, t.f1 = t.createLocalFile("explicitFoo/" + FileName)
	_, t.f2 = t.createLocalFile("explicitFoo/" + FileName2)

	// Attempt to list explicit directory.
	entries := t.readDirectory(path.Join(mntDir, "explicitFoo/"))

	// Verify entries received successfully.
	AssertEq(2, len(entries))
	t.verifyLocalFileEntry(entries[0], FileName, 0)
	t.verifyLocalFileEntry(entries[1], FileName2, 0)
	// Close the local files.
	t.closeFileAndValidateObjectContents(&t.f1, "explicitFoo/"+FileName, "")
	t.closeFileAndValidateObjectContents(&t.f2, "explicitFoo/"+FileName2, "")
}

func (t *LocalFileTest) TestReadDirForImplicitDirWithLocalFile() {
	// Create implicit dir with 2 local files and 1 synced file.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"implicitFoo/bar": "",
			}))
	_, t.f1 = t.createLocalFile("implicitFoo/" + FileName)
	_, t.f2 = t.createLocalFile("implicitFoo/" + FileName2)

	// Attempt to list implicit directory.
	entries := t.readDirectory(path.Join(mntDir, "implicitFoo/"))

	// Verify entries received successfully.
	AssertEq(3, len(entries))
	t.verifyLocalFileEntry(entries[0], "bar", 0)
	t.verifyLocalFileEntry(entries[1], FileName, 0)
	t.verifyLocalFileEntry(entries[2], FileName2, 0)
	// Close the local files.
	t.closeFileAndValidateObjectContents(&t.f1, "implicitFoo/"+FileName, "")
	t.closeFileAndValidateObjectContents(&t.f2, "implicitFoo/"+FileName2, "")
}

func (t *LocalFileTest) TestRecursiveListingWithLocalFiles() {
	// Structure
	// mntDir/
	//	   - baseLocalFile 			--- file
	//     - explicitFoo/		    --- directory
	//		   - explicitLocalFile  --- file
	//	   - implicitFoo/ 			--- directory
	//		   - bar				--- file
	//		   - implicitLocalFile  --- file

	// Create implicit dir with 1 local file1 and 1 synced file.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"implicitFoo/bar": "",
			}))
	_, t.f1 = t.createLocalFile("implicitFoo/" + implicitLocalFileName)
	// Create explicit dir with 1 local file.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"explicitFoo/": "",
			}))
	_, t.f2 = t.createLocalFile("explicitFoo/" + explicitLocalFileName)
	// Create local file in mnt/ dir.
	_, t.f3 = t.createLocalFile(FileName)

	// Recursively list mntDir/ directory.
	err := filepath.WalkDir(mntDir, func(path string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// The object type is not directory.
		if !dir.IsDir() {
			return nil
		}

		objs, err := os.ReadDir(path)
		AssertEq(nil, err)

		// Check if mntDir has correct objects.
		if path == mntDir {
			// numberOfObjects = 3
			AssertEq(3, len(objs))
			t.verifyDirectoryEntry(objs[0], "explicitFoo")
			t.verifyLocalFileEntry(objs[1], FileName, 0)
			t.verifyDirectoryEntry(objs[2], "implicitFoo")
		}

		// Check if mntDir/explicitFoo/ has correct objects.
		if path == mntDir+"/explicitFoo" {
			// numberOfObjects = 1
			AssertEq(1, len(objs))
			t.verifyLocalFileEntry(objs[0], explicitLocalFileName, 0)
		}

		// Check if mntDir/implicitFoo/ has correct objects.
		if path == mntDir+"/implicitFoo" {
			// numberOfObjects = 2
			AssertEq(2, len(objs))
			t.verifyLocalFileEntry(objs[0], "bar", 0)
			t.verifyLocalFileEntry(objs[1], implicitLocalFileName, 0)
		}
		return nil
	})

	// Validate and close the files.
	AssertEq(nil, err)
	t.closeFileAndValidateObjectContents(&t.f1, "implicitFoo/"+implicitLocalFileName, "")
	t.closeFileAndValidateObjectContents(&t.f2, "explicitFoo/"+explicitLocalFileName, "")
	t.closeFileAndValidateObjectContents(&t.f3, ""+FileName, "")
}

func (t *LocalFileTest) TestRenameOfLocalFileFails() {
	// Create local file with some content.
	_, t.f1 = t.createLocalFile(FileName)
	_, err := t.f1.WriteString(FileContents)
	AssertEq(nil, err)

	// Attempt to rename local file.
	err = os.Rename(path.Join(mntDir, FileName), path.Join(mntDir, "newName"))

	// Verify rename operation fails.
	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "operation not supported"))
	// write more content to local file.
	_, err = t.f1.WriteString(FileContents)
	AssertEq(nil, err)
	// Close the local file.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, FileContents+FileContents)
}

func (t *LocalFileTest) TestRenameOfDirectoryWithLocalFileFails() {
	// Create directory foo.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"foo/":        "",
				"foo/gcsFile": "",
			}))
	// Create local file with some content.
	_, t.f1 = t.createLocalFile("foo/" + FileName)
	_, err := t.f1.WriteString(FileContents)
	AssertEq(nil, err)

	// Attempt to rename directory containing local file.
	err = os.Rename(path.Join(mntDir, "foo/"), path.Join(mntDir, "bar/"))

	// Verify rename operation fails.
	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "operation not supported"))
	// write more content to local file.
	_, err = t.f1.WriteString(FileContents)
	AssertEq(nil, err)
	// Close the local file.
	t.closeFileAndValidateObjectContents(&t.f1, "foo/"+FileName, FileContents+FileContents)
}

func (t *LocalFileTest) TestRenameOfLocalFileSucceedsAfterSync() {
	t.TestRenameOfLocalFileFails()

	// Attempt to Rename synced file.
	err := os.Rename(path.Join(mntDir, FileName), path.Join(mntDir, "newName"))

	// Validate.
	AssertEq(nil, err)
	t.validateObjectContents("newName", FileContents+FileContents)
	t.validateObjectNotFoundErr(FileName)
}

func (t *LocalFileTest) TestRenameOfDirectoryWithLocalFileSucceedsAfterSync() {
	t.TestRenameOfDirectoryWithLocalFileFails()

	// Attempt to rename directory again after sync.
	err := os.Rename(path.Join(mntDir, "foo/"), path.Join(mntDir, "bar/"))

	// Validate.
	AssertEq(nil, err)
	t.validateObjectContents("bar/"+FileName, FileContents+FileContents)
	t.validateObjectNotFoundErr("foo/" + FileName)
	t.validateObjectContents("bar/gcsFile", "")
	t.validateObjectNotFoundErr("foo/gcsFile")
}

func (t *LocalFileTest) ReadLocalFile() {
	// Create a local file.
	_, t.f1 = t.createLocalFile(FileName)

	// Write some contents to file.
	contents := "string1string2string3"
	_, err := t.f1.Write([]byte(contents))
	AssertEq(nil, err)

	// File shouldn't get created on GCS.
	t.validateObjectNotFoundErr(FileName)
	// Read the local file contents.
	buf := make([]byte, len(contents))
	n, err := t.f1.ReadAt(buf, 0)
	AssertEq(nil, err)
	AssertEq(len(contents), n)
	AssertEq(contents, string(buf))

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, contents)
}

func (t *LocalFileTest) TestReadDirContainingUnlinkedLocalFiles() {
	// Create local files.
	var filepath3 string
	_, t.f1 = t.createLocalFile(FileName + "1")
	_, t.f2 = t.createLocalFile(FileName + "2")
	filepath3, t.f3 = t.createLocalFile(FileName + "3")
	// Unlink local file 3
	err := os.Remove(filepath3)
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
	// Verify unlinked file is not written to GCS
	err = t.closeLocalFile(&t.f3)
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName + "3")
}

func (t *LocalFileTest) TestUnlinkOfLocalFile() {
	// Create empty local file.
	var filepath string
	filepath, t.f1 = t.createLocalFile(FileName)

	// Attempt to unlink local file.
	err := os.Remove(filepath)

	// Verify unlink operation succeeds.
	AssertEq(nil, err)
	t.validateNoFileOrDirError(FileName)
	err = t.closeLocalFile(&t.f1)
	AssertEq(nil, err)
	// Validate file it is not present on GCS.
	t.validateObjectNotFoundErr(FileName)
}

func (t *LocalFileTest) TestWriteOnUnlinkedLocalFileSucceeds() {
	// Create local file and unlink.
	var filepath string
	filepath, t.f1 = t.createLocalFile(FileName)
	err := os.Remove(filepath)
	// Verify unlink operation succeeds.
	AssertEq(nil, err)
	t.validateNoFileOrDirError(FileName)

	// Write to unlinked local file.
	_, err = t.f1.WriteString(FileContents)
	AssertEq(nil, err)
	err = t.closeLocalFile(&t.f1)

	// Validate flush file does not throw error.
	AssertEq(nil, err)
	// Validate unlinked file is not written to GCS
	t.validateObjectNotFoundErr(FileName)
}

func (t *LocalFileTest) TestSyncOnUnlinkedLocalFile() {
	// Create local file.
	var filepath string
	filepath, t.f1 = t.createLocalFile(FileName)

	// Attempt to unlink local file.
	err := os.Remove(filepath)

	// Verify unlink operation succeeds.
	AssertEq(nil, err)
	t.validateNoFileOrDirError(FileName)
	// Validate sync operation does not write to GCS after unlink.
	err = t.f1.Sync()
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)
	// Close the local file and validate it is not present on GCS.
	err = t.closeLocalFile(&t.f1)
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)
}

func (t *LocalFileTest) TestUnlinkOfSyncedLocalFile() {
	// Create local file and sync to GCS.
	var filepath string
	filepath, t.f1 = t.createLocalFile(FileName)
	t.closeFileAndValidateObjectContents(&t.f1, FileName, "")

	// Attempt to unlink synced file.
	err := os.Remove(filepath)

	// Verify unlink operation succeeds.
	AssertEq(nil, err)
	t.validateNoFileOrDirError(FileName)
	t.validateObjectNotFoundErr(FileName)
}

func (t *LocalFileTest) TestRmDirOfDirectoryContainingGCSAndLocalFiles() {
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
	// Validate flush file throws IO error and does not create object on GCS
	err = t.closeLocalFile(&t.f1)
	AssertEq(nil, err)
	t.validateObjectNotFoundErr("explicit/" + explicitLocalFileName)
	// Validate synced files are also deleted.
	t.validateObjectNotFoundErr("explicit/foo")
	t.validateObjectNotFoundErr("explicit/")
}

func (t *LocalFileTest) TestRmDirOfDirectoryContainingOnlyLocalFiles() {
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
	// Close the local files and validate they are not present on GCS.
	err = t.closeLocalFile(&t.f1)
	AssertEq(nil, err)
	t.validateObjectNotFoundErr("explicit/" + explicitLocalFileName)
	err = t.closeLocalFile(&t.f2)
	AssertEq(nil, err)
	t.validateObjectNotFoundErr("explicit/" + FileName)
	// Validate directory is also deleted.
	t.validateObjectNotFoundErr("explicit/")
}

func (t *LocalFileTest) TestRmDirOfDirectoryContainingOnlyGCSFiles() {
	// Create explicit directory with one synced and one local file.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"explicit/":    "",
				"explicit/foo": "",
				"explicit/bar": "",
			}))

	// Attempt to remove explicit directory.
	err := os.RemoveAll(path.Join(mntDir, "explicit"))

	// Verify rmDir operation succeeds.
	AssertEq(nil, err)
	t.validateNoFileOrDirError("explicit")
	t.validateNoFileOrDirError("explicit/foo")
	t.validateNoFileOrDirError("explicit/bar")
	// Validate files are also deleted from GCS.
	t.validateObjectNotFoundErr("explicit/")
	t.validateObjectNotFoundErr("explicit/foo")
	t.validateObjectNotFoundErr("explicit/bar")
}

func (t *LocalFileTest) TestCreateSymlinkForLocalFile() {
	var filePath string
	// Create a local file.
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
	contents, err := os.ReadFile(symlinkName)
	AssertEq(nil, err)
	ExpectEq(FileContents, string(contents))
	t.closeFileAndValidateObjectContents(&t.f1, FileName, FileContents)
}

func (t *LocalFileTest) TestReadSymlinkForDeletedLocalFile() {
	var filePath string
	// Create a local file.
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

	// Remove filePath and then close the fileHandle to avoid syncing to GCS.
	err = os.Remove(filePath)
	AssertEq(nil, err)
	err = t.closeLocalFile(&t.f1)
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)

	// Reading symlink should fail.
	_, err = os.Stat(symlinkName)
	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
}

func (t *LocalFileTest) AtimeMtimeAndCtime() {
	createTime := mtimeClock.Now()
	var filePath string
	// Create a local file.
	filePath, t.f1 = t.createLocalFile(FileName)
	var err error
	fi, err := os.Stat(filePath)
	AssertEq(nil, err)

	// Check if mtime is returned correctly for unsynced file.
	_, _, mtime := fusetesting.GetTimes(fi)

	ExpectThat(mtime, timeutil.TimeNear(createTime, Delta))

	// Write some contents.
	_, err = t.f1.Write([]byte("test contents"))
	AssertEq(nil, err)

	// Stat it.
	fi, err = os.Stat(filePath)
	AssertEq(nil, err)

	// We require only that atime and ctime be "reasonable".
	atime, ctime, mtime := fusetesting.GetTimes(fi)
	ExpectThat(mtime, timeutil.TimeNear(createTime, Delta))
	ExpectThat(atime, timeutil.TimeNear(createTime, Delta))
	ExpectThat(ctime, timeutil.TimeNear(createTime, Delta))
}
