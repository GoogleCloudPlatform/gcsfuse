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

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/internal/fs/inode"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/ogletest"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////
const FileName = "foo"
const FileName2 = "foo2"
const implicitLocalFileName = "implicitLocalFile"
const explicitLocalFileName = "explicitLocalFile"
const FileContents = "teststring"

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
	_, err := gcsutil.ReadObject(ctx, bucket, fileName)

	ExpectTrue(errors.As(err, &notFoundErr))
}

func (t *LocalFileTest) closeFileAndValidateObjectContents(f **os.File, fileName string, contents string) {
	err := (*f).Close()
	AssertEq(nil, err)
	*f = nil

	t.validateObjectContents(fileName, contents)
}

func (t *LocalFileTest) validateObjectContents(fileName string, contents string) {
	contentBytes, err := gcsutil.ReadObject(ctx, bucket, fileName)
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
	/* Structure
	mntDir/
		- baseLocalFile 			--- file
	  - explicitFoo/				--- directory
			- explicitLocalFile --- file
		- implicitFoo/ 				--- directory
			- bar								--- file
			- implicitLocalFile --- file
	*/

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
