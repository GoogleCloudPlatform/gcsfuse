// Copyright 2015 Google LLC
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
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse/fusetesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////
const FileName = "foo"
const FileName2 = "foo2"
const implicitLocalFileName = "implicitLocalFile"
const explicitLocalFileName = "explicitLocalFile"
const FileContents = "teststring"
const FileContentsSize = 10
const Delta = 30 * time.Minute

type LocalFileTest struct {
	// fsTest has f1 *osFile and f2 *osFile which we will reuse here.
	f3 *os.File
	fsTest
	suite.Suite
}

func TestLocalFileTest(t *testing.T) {
	suite.Run(t, &LocalFileTest{})
}

func (t *LocalFileTest) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.NewConfig = &cfg.Config{
		Write: cfg.WriteConfig{
			CreateEmptyFile: false,
		}}
	t.fsTest.SetUpTestSuite()
}

func (t *LocalFileTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *LocalFileTest) TearDownTest() {
	// Close t.f3 in case of test failure.
	if t.f3 != nil {
		assert.NoError(t.T(), t.f3.Close())
		t.f3 = nil
	}

	// fsTest Cleanups to clean up mntDir and close t.f1 and t.f2.
	t.fsTest.TearDown()
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////
func (t *LocalFileTest) createLocalFile(fileName string) (filePath string, f *os.File) {
	t.T().Helper()
	// Creating a file shouldn't create file on GCS.
	filePath = path.Join(mntDir, fileName)
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC|syscall.O_DIRECT, 0655)

	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr(fileName)

	return
}

func (t *LocalFileTest) verifyLocalFileEntry(entry os.DirEntry, fileName string, size int) {
	t.T().Helper()
	assert.False(t.T(), entry.IsDir())
	assert.Equal(t.T(), fileName, entry.Name())

	fileInfo, err := entry.Info()
	require.NoError(t.T(), err)
	assert.EqualValues(t.T(), size, fileInfo.Size())
}

func (t *LocalFileTest) verifyDirectoryEntry(entry os.DirEntry, dirName string) {
	t.T().Helper()
	assert.True(t.T(), entry.IsDir())
	assert.Equal(t.T(), dirName, entry.Name())
}

func (t *LocalFileTest) readDirectory(dirPath string) (entries []os.DirEntry) {
	t.T().Helper()
	entries, err := os.ReadDir(dirPath)
	require.NoError(t.T(), err)
	return
}

func (t *LocalFileTest) validateObjectNotFoundErr(fileName string) {
	t.T().Helper()
	var notFoundErr *gcs.NotFoundError
	_, err := storageutil.ReadObject(ctx, bucket, fileName)

	assert.True(t.T(), errors.As(err, &notFoundErr))
}

func (t *LocalFileTest) closeLocalFile(f **os.File) error {
	err := (*f).Close()
	*f = nil
	return err
}

func (t *LocalFileTest) closeFileAndValidateObjectContents(f **os.File, fileName string, contents string) {
	t.T().Helper()
	err := t.closeLocalFile(f)
	require.NoError(t.T(), err)
	t.validateObjectContents(fileName, contents)
}

func (t *LocalFileTest) validateObjectContents(fileName string, contents string) {
	t.T().Helper()
	contentBytes, err := storageutil.ReadObject(ctx, bucket, fileName)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), contents, string(contentBytes))
}

func (t *LocalFileTest) newFileShouldGetSyncedToGCSAtClose(fileName string) {
	t.T().Helper()
	// Create a local file.
	_, t.f1 = t.createLocalFile(fileName)
	// Writing contents to local file shouldn't create file on GCS.
	_, err := t.f1.Write([]byte(FileContents))
	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr(fileName)

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(&t.f1, fileName, FileContents)

	// Validate object attributes non-nil and non-empty.
	minObject, extendedAttr, err := bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: fileName, ForceFetchFromGcs: true, ReturnExtendedObjectAttributes: true})
	require.NoError(t.T(), err)
	require.NotNil(t.T(), extendedAttr)
	require.NotNil(t.T(), minObject)
	assert.False(t.T(), reflect.DeepEqual(*extendedAttr, gcs.ExtendedObjectAttributes{}))
	assert.False(t.T(), reflect.DeepEqual(*minObject, gcs.MinObject{}))
}

func (t *LocalFileTest) validateNoFileOrDirError(filename string) {
	_, err := os.Stat(path.Join(mntDir, filename))
	require.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *LocalFileTest) TestNewFileShouldNotGetSyncedToGCSTillClose() {
	t.newFileShouldGetSyncedToGCSAtClose(FileName)
}

func (t *LocalFileTest) TestNewFileUnderExplicitDirectoryShouldNotGetSyncedToGCSTillClose() {
	err := os.Mkdir(path.Join(mntDir, "explicit"), dirPerms)
	require.NoError(t.T(), err)

	t.newFileShouldGetSyncedToGCSAtClose("explicit/foo")
}

func (t *LocalFileTest) TestNewFileUnderImplicitDirectoryShouldNotGetSyncedToGCSTillClose() {
	require.NoError(
		t.T(),
		t.createObjects(
			map[string]string{
				// File
				"implicitFoo/bar": "",
			}))

	t.newFileShouldGetSyncedToGCSAtClose("implicitFoo/foo")
}

func (t *LocalFileTest) TestStatOnLocalFile() {
	// Create a local file.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)

	// Stat the local file.
	fi, err := os.Stat(filePath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), path.Base(filePath), fi.Name())
	assert.EqualValues(t.T(), 0, fi.Size())
	assert.Equal(t.T(), filePerms, fi.Mode())

	// Writing contents to local file shouldn't create file on GCS.
	_, err = t.f1.Write([]byte(FileContents))
	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr(FileName)

	// Stat the local file again to check if new contents are written.
	fi, err = os.Stat(filePath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), path.Base(filePath), fi.Name())
	assert.EqualValues(t.T(), 10, fi.Size())
	assert.Equal(t.T(), filePerms, fi.Mode())

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, FileContents)
}

func (t *LocalFileTest) TestStatOnLocalFileWithConflictingFileNameSuffix() {
	// Create a local file.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)
	// Stat the local file.
	fi, err := os.Stat(filePath + inode.ConflictingFileNameSuffix)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), path.Base(filePath)+inode.ConflictingFileNameSuffix, fi.Name())
	assert.EqualValues(t.T(), 0, fi.Size())
	assert.Equal(t.T(), filePerms, fi.Mode())

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, "")
}

func (t *LocalFileTest) TestStatOnUnlinkedLocalFile() {
	// Create a local file.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)
	// unlink the local file.
	err := os.Remove(filePath)
	require.NoError(t.T(), err)

	// Stat the local file and validate error.
	t.validateNoFileOrDirError(FileName)

	// Close the file and validate that file is not created on GCS.
	err = t.closeLocalFile(&t.f1)
	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr(FileName)
}

func (t *LocalFileTest) TestTruncateLocalFile() {
	// Create a local file.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)
	// Writing contents to local file .
	_, err := t.f1.Write([]byte(FileContents))
	require.NoError(t.T(), err)

	// Stat the file to validate if new contents are written.
	fi, err := os.Stat(filePath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), path.Base(filePath), fi.Name())
	assert.EqualValues(t.T(), 10, fi.Size())
	assert.Equal(t.T(), filePerms, fi.Mode())

	// Truncate the file to update the file size.
	err = os.Truncate(filePath, 5)
	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr(FileName)

	// Stat the file to validate if file is truncated correctly.
	fi, err = os.Stat(filePath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), path.Base(filePath), fi.Name())
	assert.EqualValues(t.T(), 5, fi.Size())
	assert.Equal(t.T(), filePerms, fi.Mode())

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, "tests")
}

func (t *LocalFileTest) TestMultipleWritesToLocalFile() {
	// Create a local file.
	_, t.f1 = t.createLocalFile(FileName)

	// Write some contents to file sequentially.
	_, err := t.f1.Write([]byte("string1"))
	require.NoError(t.T(), err)
	_, err = t.f1.Write([]byte("string2"))
	require.NoError(t.T(), err)
	_, err = t.f1.Write([]byte("string3"))
	require.NoError(t.T(), err)
	// File shouldn't get created on GCS.
	t.validateObjectNotFoundErr(FileName)

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, "string1string2string3")
}

func (t *LocalFileTest) TestRandomWritesToLocalFile() {
	// Create a local file.
	_, t.f1 = t.createLocalFile(FileName)

	// Write some contents to file randomly.
	_, err := t.f1.WriteAt([]byte("string1"), 0)
	require.NoError(t.T(), err)
	_, err = t.f1.WriteAt([]byte("string2"), 2)
	require.NoError(t.T(), err)
	_, err = t.f1.WriteAt([]byte("string3"), 3)
	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr(FileName)

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, "stsstring3")
}

func (t *LocalFileTest) TestTestReadDirWithEmptyLocalFiles() {
	// Create local files.
	_, t.f1 = t.createLocalFile(FileName)
	_, t.f2 = t.createLocalFile(FileName2)

	// Attempt to list mntDir.
	entries := t.readDirectory(mntDir)

	// Verify entries received successfully.
	require.Equal(t.T(), 2, len(entries))
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
	require.NoError(t.T(), err)

	// Attempt to list mntDir.
	entries := t.readDirectory(mntDir)

	// Verify entries received successfully.
	require.Equal(t.T(), 1, len(entries))
	t.verifyLocalFileEntry(entries[0], FileName, 10)
	// Close the local files.
	t.closeFileAndValidateObjectContents(&t.f1, FileName, FileContents)
}

func (t *LocalFileTest) TestReadDirForExplicitDirWithLocalFile() {
	// Create explicit dir with 2 local files.
	assert.Nil(t.T(),
		t.createObjects(
			map[string]string{
				"explicitFoo/": "",
			}))
	_, t.f1 = t.createLocalFile("explicitFoo/" + FileName)
	_, t.f2 = t.createLocalFile("explicitFoo/" + FileName2)

	// Attempt to list explicit directory.
	entries := t.readDirectory(path.Join(mntDir, "explicitFoo/"))

	// Verify entries received successfully.
	assert.Equal(t.T(), 2, len(entries))
	t.verifyLocalFileEntry(entries[0], FileName, 0)
	t.verifyLocalFileEntry(entries[1], FileName2, 0)
	// Close the local files.
	t.closeFileAndValidateObjectContents(&t.f1, "explicitFoo/"+FileName, "")
	t.closeFileAndValidateObjectContents(&t.f2, "explicitFoo/"+FileName2, "")
}

func (t *LocalFileTest) TestReadDirForImplicitDirWithLocalFile() {
	// Create implicit dir with 2 local files and 1 synced file.
	require.NoError(t.T(),
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
	require.Equal(t.T(), 3, len(entries))
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
	require.NoError(t.T(),
		t.createObjects(
			map[string]string{
				// File
				"implicitFoo/bar": "",
			}))
	_, t.f1 = t.createLocalFile("implicitFoo/" + implicitLocalFileName)
	// Create explicit dir with 1 local file.
	require.NoError(t.T(),
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
		require.NoError(t.T(), err)

		// Check if mntDir has correct objects.
		if path == mntDir {
			// numberOfObjects = 3
			require.Equal(t.T(), 3, len(objs))
			t.verifyDirectoryEntry(objs[0], "explicitFoo")
			t.verifyLocalFileEntry(objs[1], FileName, 0)
			t.verifyDirectoryEntry(objs[2], "implicitFoo")
		}

		// Check if mntDir/explicitFoo/ has correct objects.
		if path == mntDir+"/explicitFoo" {
			// numberOfObjects = 1
			require.Equal(t.T(), 1, len(objs))
			t.verifyLocalFileEntry(objs[0], explicitLocalFileName, 0)
		}

		// Check if mntDir/implicitFoo/ has correct objects.
		if path == mntDir+"/implicitFoo" {
			// numberOfObjects = 2
			require.Equal(t.T(), 2, len(objs))
			t.verifyLocalFileEntry(objs[0], "bar", 0)
			t.verifyLocalFileEntry(objs[1], implicitLocalFileName, 0)
		}
		return nil
	})

	// Validate and close the files.
	require.NoError(t.T(), err)
	t.closeFileAndValidateObjectContents(&t.f1, "implicitFoo/"+implicitLocalFileName, "")
	t.closeFileAndValidateObjectContents(&t.f2, "explicitFoo/"+explicitLocalFileName, "")
	t.closeFileAndValidateObjectContents(&t.f3, ""+FileName, "")
}

func (t *LocalFileTest) TestRenameOfLocalFile() {
	// Create local file with some content.
	_, t.f1 = t.createLocalFile(FileName)
	_, err := t.f1.WriteString(FileContents)
	require.NoError(t.T(), err)
	newName := "newName"

	// Attempt to rename local file.
	err = os.Rename(path.Join(mntDir, FileName), path.Join(mntDir, newName))

	// Verify rename operation fails.
	require.NoError(t.T(), err)
	// Close the local file.
	t.validateObjectContents(newName, FileContents)
	t.validateObjectNotFoundErr(FileName)
}

func (t *LocalFileTest) TestRenameOfDirectoryWithLocalFileFails() {
	// Create directory foo.
	require.NoError(t.T(),
		t.createObjects(
			map[string]string{
				"foo/":        "",
				"foo/gcsFile": "",
			}))
	// Create local file with some content.
	_, t.f1 = t.createLocalFile("foo/" + FileName)
	_, err := t.f1.WriteString(FileContents)
	require.NoError(t.T(), err)

	// Attempt to rename directory containing local file.
	err = os.Rename(path.Join(mntDir, "foo/"), path.Join(mntDir, "bar/"))

	// Verify rename operation fails.
	require.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "operation not supported"))
	// write more content to local file.
	_, err = t.f1.WriteString(FileContents)
	require.NoError(t.T(), err)
	// Close the local file.
	t.closeFileAndValidateObjectContents(&t.f1, "foo/"+FileName, FileContents+FileContents)
}

func (t *LocalFileTest) TestRenameOfDirectoryWithLocalFileSucceedsAfterSync() {
	t.TestRenameOfDirectoryWithLocalFileFails()

	// Attempt to rename directory again after sync.
	err := os.Rename(path.Join(mntDir, "foo/"), path.Join(mntDir, "bar/"))

	// Validate.
	require.NoError(t.T(), err)
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
	require.NoError(t.T(), err)

	// File shouldn't get created on GCS.
	t.validateObjectNotFoundErr(FileName)
	// Read the local file contents.
	buf := make([]byte, len(contents))
	n, err := t.f1.ReadAt(buf, 0)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), len(contents), n)
	assert.Equal(t.T(), contents, string(buf))

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
	require.NoError(t.T(), err)

	// Attempt to list mntDir.
	entries := t.readDirectory(mntDir)

	// Verify unlinked entries are not listed.
	require.Equal(t.T(), 2, len(entries))
	t.verifyLocalFileEntry(entries[0], FileName+"1", 0)
	t.verifyLocalFileEntry(entries[1], FileName+"2", 0)
	// Close the local files.
	t.closeFileAndValidateObjectContents(&t.f1, FileName+"1", "")
	t.closeFileAndValidateObjectContents(&t.f2, FileName+"2", "")
	// Verify unlinked file is not written to GCS
	err = t.closeLocalFile(&t.f3)
	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr(FileName + "3")
}

func (t *LocalFileTest) TestUnlinkOfLocalFile() {
	// Create empty local file.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)

	// Attempt to unlink local file.
	err := os.Remove(filePath)

	// Verify unlink operation succeeds.
	require.NoError(t.T(), err)
	t.validateNoFileOrDirError(FileName)
	err = t.closeLocalFile(&t.f1)
	require.NoError(t.T(), err)
	// Validate file it is not present on GCS.
	t.validateObjectNotFoundErr(FileName)
}

func (t *LocalFileTest) TestWriteOnUnlinkedLocalFileSucceeds() {
	// Create local file and unlink.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)
	err := os.Remove(filePath)
	// Verify unlink operation succeeds.
	require.NoError(t.T(), err)
	t.validateNoFileOrDirError(FileName)

	// Write to unlinked local file.
	_, err = t.f1.WriteString(FileContents)
	require.NoError(t.T(), err)
	err = t.closeLocalFile(&t.f1)

	// Validate flush file does not throw error.
	require.NoError(t.T(), err)
	// Validate unlinked file is not written to GCS
	t.validateObjectNotFoundErr(FileName)
}

func (t *LocalFileTest) TestSyncOnUnlinkedLocalFile() {
	// Create local file.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)

	// Attempt to unlink local file.
	err := os.Remove(filePath)

	// Verify unlink operation succeeds.
	require.NoError(t.T(), err)
	t.validateNoFileOrDirError(FileName)
	// Validate sync operation does not write to GCS after unlink.
	err = t.f1.Sync()
	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr(FileName)
	// Close the local file and validate it is not present on GCS.
	err = t.closeLocalFile(&t.f1)
	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr(FileName)
}

func (t *LocalFileTest) TestUnlinkOfSyncedLocalFile() {
	// Create local file and sync to GCS.
	var filePath string
	filePath, t.f1 = t.createLocalFile(FileName)
	t.closeFileAndValidateObjectContents(&t.f1, FileName, "")

	// Attempt to unlink synced file.
	err := os.Remove(filePath)

	// Verify unlink operation succeeds.
	require.NoError(t.T(), err)
	t.validateNoFileOrDirError(FileName)
	t.validateObjectNotFoundErr(FileName)
}

func (t *LocalFileTest) TestRmDirOfDirectoryContainingGCSAndLocalFiles() {
	// Create explicit directory with one synced and one local file.
	require.NoError(t.T(),
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
	require.NoError(t.T(), err)
	t.validateNoFileOrDirError("explicit/" + explicitLocalFileName)
	t.validateNoFileOrDirError("explicit/foo")
	t.validateNoFileOrDirError("explicit")
	// Validate writing content to unlinked local file does not throw error
	_, err = t.f1.WriteString(FileContents)
	require.NoError(t.T(), err)
	// Validate flush file throws IO error and does not create object on GCS
	err = t.closeLocalFile(&t.f1)
	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr("explicit/" + explicitLocalFileName)
	// Validate synced files are also deleted.
	t.validateObjectNotFoundErr("explicit/foo")
	t.validateObjectNotFoundErr("explicit/")
}

func (t *LocalFileTest) TestRmDirOfDirectoryContainingOnlyLocalFiles() {
	// Create a directory with two local files.
	err := os.Mkdir(path.Join(mntDir, "explicit"), dirPerms)
	require.NoError(t.T(), err)
	_, t.f1 = t.createLocalFile("explicit/" + explicitLocalFileName)
	_, t.f2 = t.createLocalFile("explicit/" + FileName)

	// Attempt to remove explicit directory.
	err = os.RemoveAll(path.Join(mntDir, "explicit"))

	// Verify rmDir operation succeeds.
	require.NoError(t.T(), err)
	t.validateNoFileOrDirError("explicit/" + explicitLocalFileName)
	t.validateNoFileOrDirError("explicit/" + FileName)
	t.validateNoFileOrDirError("explicit")
	// Close the local files and validate they are not present on GCS.
	err = t.closeLocalFile(&t.f1)
	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr("explicit/" + explicitLocalFileName)
	err = t.closeLocalFile(&t.f2)
	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr("explicit/" + FileName)
	// Validate directory is also deleted.
	t.validateObjectNotFoundErr("explicit/")
}

func (t *LocalFileTest) TestRmDirOfDirectoryContainingOnlyGCSFiles() {
	// Create explicit directory with one synced and one local file.
	require.NoError(t.T(),
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
	require.NoError(t.T(), err)
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
	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr(FileName)

	// Create the symlink.
	symlinkName := path.Join(mntDir, "bar")
	err = os.Symlink(filePath, symlinkName)
	require.NoError(t.T(), err)

	// Read the link.
	target, err := os.Readlink(symlinkName)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), filePath, target)
	contents, err := os.ReadFile(symlinkName)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), FileContents, string(contents))
	t.closeFileAndValidateObjectContents(&t.f1, FileName, FileContents)
}

func (t *LocalFileTest) TestReadSymlinkForDeletedLocalFile() {
	var filePath string
	// Create a local file.
	filePath, t.f1 = t.createLocalFile(FileName)
	// Writing contents to local file shouldn't create file on GCS.
	_, err := t.f1.Write([]byte(FileContents))
	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr(FileName)
	// Create the symlink.
	symlinkName := path.Join(mntDir, "bar")
	err = os.Symlink(filePath, symlinkName)
	require.NoError(t.T(), err)
	// Read the link.
	target, err := os.Readlink(symlinkName)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), filePath, target)

	// Remove filePath and then close the fileHandle to avoid syncing to GCS.
	err = os.Remove(filePath)
	require.NoError(t.T(), err)
	err = t.closeLocalFile(&t.f1)
	require.NoError(t.T(), err)
	t.validateObjectNotFoundErr(FileName)

	// Reading symlink should fail.
	_, err = os.Stat(symlinkName)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
}

func (t *LocalFileTest) TestAtimeMtimeAndCtime() {
	createTime := mtimeClock.Now()
	var filePath string
	// Create a local file.
	filePath, t.f1 = t.createLocalFile(FileName)
	var err error
	fi, err := os.Stat(filePath)
	require.NoError(t.T(), err)

	// Check if mtime is returned correctly for unsynced file.
	_, _, mtime := fusetesting.GetTimes(fi)

	assert.WithinDuration(t.T(), createTime, mtime, Delta)

	// Write some contents.
	_, err = t.f1.Write([]byte("test contents"))
	require.NoError(t.T(), err)

	// Stat it.
	fi, err = os.Stat(filePath)
	require.NoError(t.T(), err)

	// We require only that atime and ctime be "reasonable".
	atime, ctime, mtime := fusetesting.GetTimes(fi)
	assert.WithinDuration(t.T(), createTime, mtime, Delta)
	assert.WithinDuration(t.T(), createTime, atime, Delta)
	assert.WithinDuration(t.T(), createTime, ctime, Delta)
}

// Create local file inside - test.txt
// Stat that local file.
// Remove the local file.
// Create local file with the same name - test.txt
// Stat that local file.
func (t *LocalFileTest) TestStatLocalFileAfterRecreatingItWithSameName() {
	filePath := path.Join(mntDir, "test.txt")
	f1, err := os.Create(filePath)
	defer require.NoError(t.T(), f1.Close())
	require.NoError(t.T(), err)
	_, err = os.Stat(filePath)
	require.NoError(t.T(), err)
	err = os.Remove(filePath)
	require.NoError(t.T(), err)
	f2, err := os.Create(filePath)
	require.NoError(t.T(), err)
	defer require.NoError(t.T(), f2.Close())

	f, err := os.Stat(filePath)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), "test.txt", f.Name())
	assert.False(t.T(), f.IsDir())
}

func (t *LocalFileTest) TestStatFailsOnNewFileAfterDeletion() {
	t.serverCfg.NewConfig = &cfg.Config{
		ImplicitDirs: true,
		MetadataCache: cfg.MetadataCacheConfig{
			TtlSecs:            -1,
			TypeCacheMaxSizeMb: -1,
			StatCacheMaxSizeMb: -1,
		},
		Logging: cfg.DefaultLoggingConfig(),
	}
	t.serverCfg.MetricHandle = metrics.NewNoopMetrics()
	filePath := path.Join(mntDir, "test.txt")
	f1, err := os.Create(filePath)
	require.NoError(t.T(), err)
	defer assert.Equal(t.T(), nil, f1.Close())
	assert.Equal(t.T(), nil, os.Remove(filePath))

	_, err = os.Stat(filePath)

	require.Error(t.T(), err)
}
