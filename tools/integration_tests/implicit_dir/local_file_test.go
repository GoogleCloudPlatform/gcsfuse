// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package implicit_dir_test

import (
	"io/fs"
	"path"
	"path/filepath"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	testDirName = "ImplicitDirTest"
)

type implicitDirTest struct {
	isRapidWritesEnabled bool
	suite.Suite
}

func TestImplicitDirBase(t *testing.T) {
	suite.Run(t, &implicitDirTest{isRapidWritesEnabled: false})
}

func TestImplicitDirRapidWritesEnabled(t *testing.T) {
	if !setup.IsPirloBucketRun() {
		t.Skip("Rapid writes tests are only applicable to Pirlo buckets")
	}
	suite.Run(t, &implicitDirTest{isRapidWritesEnabled: true})
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func (i *implicitDirTest) TestNewFileUnderImplicitDirectoryShouldNotGetSyncedToGCSTillClose() {
	testBaseDirName := path.Join(testDirName, operations.GetRandomName(i.T()))
	testEnv.testDirPath = setup.SetupTestDirectoryRecursive(testBaseDirName)
	CreateImplicitDir(testEnv.ctx, testEnv.storageClient, testBaseDirName, i.T())
	fileName := path.Join(ImplicitDirName, FileName1)

	_, fh := CreateLocalFileInTestDir(testEnv.ctx, testEnv.storageClient, testEnv.testDirPath, fileName, i.T())
	operations.WriteWithoutClose(fh, FileContents, i.T())
	if setup.IsZonalBucketRun() || (setup.IsPirloBucketRun() && i.isRapidWritesEnabled) {
		// For zonal buckets and pirlo rapid writes, the object is unfinalized, but visible.
		// An object written without sync would be recognized as having zero-size.
		ValidateObjectContentsFromGCS(testEnv.ctx, testEnv.storageClient, testBaseDirName, fileName, "", i.T())

		// An object written with sync can be fully read.
		err := fh.Sync()
		require.NoError(i.T(), err)
		ValidateObjectContentsFromGCS(testEnv.ctx, testEnv.storageClient, testBaseDirName, fileName, FileContents, i.T())
	} else {
		// For non-zonal/non-rapid buckets, the object is not visible until the file is closed.
		ValidateObjectNotFoundErrOnGCS(testEnv.ctx, testEnv.storageClient, testBaseDirName, fileName, i.T())
	}

	// Validate.
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, fh, testBaseDirName, fileName, FileContents, i.T())
}

func (i *implicitDirTest) TestReadDirForImplicitDirWithLocalFile() {
	testBaseDirName := path.Join(testDirName, operations.GetRandomName(i.T()))
	testEnv.testDirPath = setup.SetupTestDirectoryRecursive(testBaseDirName)
	CreateImplicitDir(testEnv.ctx, testEnv.storageClient, testBaseDirName, i.T())
	fileName1 := path.Join(ImplicitDirName, FileName1)
	fileName2 := path.Join(ImplicitDirName, FileName2)
	_, fh1 := CreateLocalFileInTestDir(testEnv.ctx, testEnv.storageClient, testEnv.testDirPath, fileName1, i.T())
	_, fh2 := CreateLocalFileInTestDir(testEnv.ctx, testEnv.storageClient, testEnv.testDirPath, fileName2, i.T())

	// Attempt to list implicit directory.
	entries := operations.ReadDirectory(path.Join(testEnv.testDirPath, ImplicitDirName), i.T())

	// Verify entries received successfully.
	operations.VerifyCountOfDirectoryEntries(3, len(entries), i.T())
	operations.VerifyFileEntry(entries[0], FileName1, 0, i.T())
	operations.VerifyFileEntry(entries[1], FileName2, 0, i.T())
	operations.VerifyFileEntry(entries[2], ImplicitFileName1, GCSFileSize, i.T())
	// Close the local files.
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, fh1, testBaseDirName, fileName1, "", i.T())
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, fh2, testBaseDirName, fileName2, "", i.T())
}

func (i *implicitDirTest) TestRecursiveListingWithLocalFiles() {
	// Structure
	// mntDir/
	// mntDir/foo1 										--- file
	// mntDir/explicit/		    				--- directory
	// mntDir/explicit/explicitFile1  --- file
	// mntDir/implicit/ 							--- directory
	// mntDir/implicit/foo2  					--- file
	// mntDir/implicit/implicitFile1	--- file

	testBaseDirName := path.Join(testDirName, operations.GetRandomName(i.T()))
	testEnv.testDirPath = setup.SetupTestDirectoryRecursive(testBaseDirName)
	fileName2 := path.Join(ExplicitDirName, ExplicitFileName1)
	fileName3 := path.Join(ImplicitDirName, FileName2)
	// Create local file in mnt/ dir.
	_, fh1 := CreateLocalFileInTestDir(testEnv.ctx, testEnv.storageClient, testEnv.testDirPath, FileName1, i.T())
	// Create explicit dir with 1 local file.
	operations.CreateDirectory(path.Join(testEnv.testDirPath, ExplicitDirName), i.T())
	_, fh2 := CreateLocalFileInTestDir(testEnv.ctx, testEnv.storageClient, testEnv.testDirPath, fileName2, i.T())
	// Create implicit dir with 1 local file1 and 1 synced file.
	CreateImplicitDir(testEnv.ctx, testEnv.storageClient, testBaseDirName, i.T())
	_, fh3 := CreateLocalFileInTestDir(testEnv.ctx, testEnv.storageClient, testEnv.testDirPath, fileName3, i.T())

	// Recursively list mntDir/ directory.
	err := filepath.WalkDir(testEnv.testDirPath,
		func(walkPath string, dir fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// The object type is not directory.
			if !dir.IsDir() {
				return nil
			}

			objs := operations.ReadDirectory(walkPath, i.T())

			// Check if mntDir has correct objects.
			if walkPath == setup.MntDir() {
				// numberOfObjects = 3
				operations.VerifyCountOfDirectoryEntries(3, len(objs), i.T())
				operations.VerifyDirectoryEntry(objs[0], ExplicitDirName, i.T())
				operations.VerifyFileEntry(objs[1], FileName1, 0, i.T())
				operations.VerifyDirectoryEntry(objs[2], ImplicitDirName, i.T())
			}

			// Check if mntDir/explicitFoo/ has correct objects.
			if walkPath == path.Join(testEnv.testDirPath, ExplicitDirName) {
				// numberOfObjects = 1
				operations.VerifyCountOfDirectoryEntries(1, len(objs), i.T())
				operations.VerifyFileEntry(objs[0], ExplicitFileName1, 0, i.T())
			}

			// Check if mntDir/implicitFoo/ has correct objects.
			if walkPath == path.Join(testEnv.testDirPath, ImplicitDirName) {
				// numberOfObjects = 2
				operations.VerifyCountOfDirectoryEntries(2, len(objs), i.T())
				operations.VerifyFileEntry(objs[0], FileName2, 0, i.T())
				operations.VerifyFileEntry(objs[1], ImplicitFileName1, GCSFileSize, i.T())
			}
			return nil
		})

	// Validate and close the files.
	assert.NoError(i.T(), err, "filepath.WalkDir failed")
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, fh1, testBaseDirName, FileName1, "", i.T())
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, fh2, testBaseDirName, fileName2, "", i.T())
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, fh3, testBaseDirName, fileName3, "", i.T())
}
