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

	"github.com/stretchr/testify/require"
	. "github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	testDirName = "ImplicitDirTest"
)

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func TestNewFileUnderImplicitDirectoryShouldNotGetSyncedToGCSTillClose(t *testing.T) {
	testBaseDirName := path.Join(testDirName, operations.GetRandomName(t))
	testEnv.testDirPath = setup.SetupTestDirectoryRecursive(testBaseDirName)
	CreateImplicitDir(testEnv.ctx, testEnv.storageClient, testBaseDirName, t)
	fileName := path.Join(ImplicitDirName, FileName1)

	_, fh := CreateLocalFileInTestDir(testEnv.ctx, testEnv.storageClient, testEnv.testDirPath, fileName, t)
	operations.WriteWithoutClose(fh, FileContents, t)
	if !setup.IsZonalBucketRun() {
		// For non-zonal buckets, the object is not visible until the file is closed.
		ValidateObjectNotFoundErrOnGCS(testEnv.ctx, testEnv.storageClient, testBaseDirName, fileName, t)
	} else {
		// For zonal buckets, the object is unfinalized, but visible.
		// A zonal bucket object written without sync would be recognized as having zero-size.
		ValidateObjectContentsFromGCS(testEnv.ctx, testEnv.storageClient, testBaseDirName, fileName, "", t)

		// A zonal bucket object written with sync can be fully read.
		err := fh.Sync()
		require.NoError(t, err)
		ValidateObjectContentsFromGCS(testEnv.ctx, testEnv.storageClient, testBaseDirName, fileName, FileContents, t)
	}

	// Validate.
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, fh, testBaseDirName, fileName, FileContents, t)
}

func TestReadDirForImplicitDirWithLocalFile(t *testing.T) {
	testBaseDirName := path.Join(testDirName, operations.GetRandomName(t))
	testEnv.testDirPath = setup.SetupTestDirectoryRecursive(testBaseDirName)
	CreateImplicitDir(testEnv.ctx, testEnv.storageClient, testBaseDirName, t)
	fileName1 := path.Join(ImplicitDirName, FileName1)
	fileName2 := path.Join(ImplicitDirName, FileName2)
	_, fh1 := CreateLocalFileInTestDir(testEnv.ctx, testEnv.storageClient, testEnv.testDirPath, fileName1, t)
	_, fh2 := CreateLocalFileInTestDir(testEnv.ctx, testEnv.storageClient, testEnv.testDirPath, fileName2, t)

	// Attempt to list implicit directory.
	entries := operations.ReadDirectory(path.Join(testEnv.testDirPath, ImplicitDirName), t)

	// Verify entries received successfully.
	operations.VerifyCountOfDirectoryEntries(3, len(entries), t)
	operations.VerifyFileEntry(entries[0], FileName1, 0, t)
	operations.VerifyFileEntry(entries[1], FileName2, 0, t)
	operations.VerifyFileEntry(entries[2], ImplicitFileName1, GCSFileSize, t)
	// Close the local files.
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, fh1, testBaseDirName, fileName1, "", t)
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, fh2, testBaseDirName, fileName2, "", t)
}

func TestRecursiveListingWithLocalFiles(t *testing.T) {
	// Structure
	// mntDir/
	// mntDir/foo1 										--- file
	// mntDir/explicit/		    				--- directory
	// mntDir/explicit/explicitFile1  --- file
	// mntDir/implicit/ 							--- directory
	// mntDir/implicit/foo2  					--- file
	// mntDir/implicit/implicitFile1	--- file

	testBaseDirName := path.Join(testDirName, operations.GetRandomName(t))
	testEnv.testDirPath = setup.SetupTestDirectoryRecursive(testBaseDirName)
	fileName2 := path.Join(ExplicitDirName, ExplicitFileName1)
	fileName3 := path.Join(ImplicitDirName, FileName2)
	// Create local file in mnt/ dir.
	_, fh1 := CreateLocalFileInTestDir(testEnv.ctx, testEnv.storageClient, testEnv.testDirPath, FileName1, t)
	// Create explicit dir with 1 local file.
	operations.CreateDirectory(path.Join(testEnv.testDirPath, ExplicitDirName), t)
	_, fh2 := CreateLocalFileInTestDir(testEnv.ctx, testEnv.storageClient, testEnv.testDirPath, fileName2, t)
	// Create implicit dir with 1 local file1 and 1 synced file.
	CreateImplicitDir(testEnv.ctx, testEnv.storageClient, testBaseDirName, t)
	_, fh3 := CreateLocalFileInTestDir(testEnv.ctx, testEnv.storageClient, testEnv.testDirPath, fileName3, t)

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

			objs := operations.ReadDirectory(walkPath, t)

			// Check if mntDir has correct objects.
			if walkPath == setup.MntDir() {
				// numberOfObjects = 3
				operations.VerifyCountOfDirectoryEntries(3, len(objs), t)
				operations.VerifyDirectoryEntry(objs[0], ExplicitDirName, t)
				operations.VerifyFileEntry(objs[1], FileName1, 0, t)
				operations.VerifyDirectoryEntry(objs[2], ImplicitDirName, t)
			}

			// Check if mntDir/explicitFoo/ has correct objects.
			if walkPath == path.Join(testEnv.testDirPath, ExplicitDirName) {
				// numberOfObjects = 1
				operations.VerifyCountOfDirectoryEntries(1, len(objs), t)
				operations.VerifyFileEntry(objs[0], ExplicitFileName1, 0, t)
			}

			// Check if mntDir/implicitFoo/ has correct objects.
			if walkPath == path.Join(testEnv.testDirPath, ImplicitDirName) {
				// numberOfObjects = 2
				operations.VerifyCountOfDirectoryEntries(2, len(objs), t)
				operations.VerifyFileEntry(objs[0], FileName2, 0, t)
				operations.VerifyFileEntry(objs[1], ImplicitFileName1, GCSFileSize, t)
			}
			return nil
		})

	// Validate and close the files.
	if err != nil {
		t.Errorf("filepath.WalkDir() err: %v", err)
	}
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, fh1, testBaseDirName, FileName1, "", t)
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, fh2, testBaseDirName, fileName2, "", t)
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, fh3, testBaseDirName, fileName3, "", t)
}
