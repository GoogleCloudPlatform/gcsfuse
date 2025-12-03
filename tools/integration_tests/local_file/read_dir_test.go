// Copyright 2023 Google LLC
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

// Provides integration tests for readDir call containing local files.
package local_file

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////
func creatingNLocalFilesShouldNotThrowError(n int, wg *sync.WaitGroup, t *testing.T) {
	defer wg.Done()
	operations.CreateDirectory(path.Join(testDirPath, ExplicitDirName), t)
	for i := range n {
		filePath := path.Join(testDirPath, ExplicitDirName, FileName1+strconv.FormatInt(int64(i), 10))
		operations.CreateFile(filePath, FilePerms, t)
	}
}

func readingDirNTimesShouldNotThrowError(n int, wg *sync.WaitGroup, t *testing.T) {
	defer wg.Done()
	for i := range n {
		_, err := os.ReadDir(setup.MntDir())
		if err != nil {
			t.Errorf("Error while reading directory %dth time: %v", i, err)
		}
	}
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func (t *LocalFileTestSuite) TestReadDir() {
	// Structure
	// mntDir/
	// mntDir/explicit/		    				--- directory
	// mntDir/explicit/explicitFile1  --- file
	// mntDir/foo1 										--- empty local file
	// mntDir/foo2  									--- non empty local file
	// mntDir/foo3										--- gcs synced file

	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create explicit dir with 1 local file.
	operations.CreateDirectory(path.Join(testDirPath, ExplicitDirName), t.T())
	_, fh1 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath,
		path.Join(ExplicitDirName, ExplicitFileName1), t.T())
	// Create empty local file.
	_, fh2 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())
	// Create non-empty local file.
	_, fh3 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName2, t.T())
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh3, testDirName, FileName2, t.T())
	// Create GCS synced file.
	CreateObjectInGCSTestDir(ctx, storageClient, testDirName, FileName3, GCSFileContent, t.T())

	// Attempt to list mnt and explicit directory.
	entriesMnt := operations.ReadDirectory(testDirPath, t.T())
	entriesDir := operations.ReadDirectory(path.Join(testDirPath, ExplicitDirName), t.T())

	// Verify entriesMnt received successfully.
	operations.VerifyCountOfDirectoryEntries(4, len(entriesMnt), t.T())
	operations.VerifyDirectoryEntry(entriesMnt[0], ExplicitDirName, t.T())
	operations.VerifyFileEntry(entriesMnt[1], FileName1, 0, t.T())
	operations.VerifyFileEntry(entriesMnt[2], FileName2, SizeOfFileContents, t.T())
	operations.VerifyFileEntry(entriesMnt[3], FileName3, GCSFileSize, t.T())
	// Verify entriesDir received successfully.
	operations.VerifyCountOfDirectoryEntries(1, len(entriesDir), t.T())
	operations.VerifyFileEntry(entriesDir[0], ExplicitFileName1, 0, t.T())
	// Close the local files.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh1, testDirName,
		path.Join(ExplicitDirName, ExplicitFileName1), "", t.T())
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh2, testDirName,
		FileName1, "", t.T())
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh3, testDirName,
		FileName2, FileContents, t.T())
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, FileName3,
		GCSFileContent, t.T())
}

func (t *LocalFileTestSuite) TestRecursiveListingWithLocalFiles() {
	// Structure
	// mntDir/
	// mntDir/foo1 										--- file
	// mntDir/explicit/		    				--- directory
	// mntDir/explicit/explicitFile1  --- file

	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local file in mnt/ dir.
	_, fh1 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())
	// Create explicit dir with 1 local file.
	operations.CreateDirectory(path.Join(testDirPath, ExplicitDirName), t.T())
	_, fh2 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath,
		path.Join(ExplicitDirName, ExplicitFileName1), t.T())

	// Recursively list mntDir/ directory.
	err := filepath.WalkDir(testDirPath, func(walkPath string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// The object type is not directory.
		if !dir.IsDir() {
			return nil
		}

		objs := operations.ReadDirectory(walkPath, t.T())

		// Check if mntDir has correct objects.
		if walkPath == testDirPath {
			// numberOfObjects = 2
			operations.VerifyCountOfDirectoryEntries(2, len(objs), t.T())
			operations.VerifyDirectoryEntry(objs[0], ExplicitDirName, t.T())
			operations.VerifyFileEntry(objs[1], FileName1, 0, t.T())
		}

		// Check if mntDir/explicit/ has correct objects.
		if walkPath == path.Join(setup.MntDir(), ExplicitDirName) {
			// numberOfObjects = 1
			operations.VerifyCountOfDirectoryEntries(1, len(objs), t.T())
			operations.VerifyFileEntry(objs[0], ExplicitFileName1, 0, t.T())
		}

		return nil
	})

	// Validate and close the files.
	if err != nil {
		t.T().Fatalf("filepath.WalkDir() err: %v", err)
	}
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh1, testDirName,
		FileName1, "", t.T())
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh2, testDirName,
		path.Join(ExplicitDirName, ExplicitFileName1), "", t.T())
}

func (t *LocalFileTestSuite) TestReadDirWithSameNameLocalAndGCSFile() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local file.
	_, fh1 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())
	// Create same name gcs file.
	time.Sleep(2 * time.Second)
	CreateObjectInGCSTestDir(ctx, storageClient, testDirName, FileName1, GCSFileContent, t.T())

	// Attempt to list testDir.
	_, err := os.ReadDir(testDirPath)
	if err != nil {
		t.T().Fatalf("ReadDir err: %v", err)
	}

	// Validate closing local file throws error.
	err = fh1.Close()
	operations.ValidateESTALEError(t.T(), err)
}

func (t *LocalFileTestSuite) TestStatLocalFileAfterRecreatingItWithSameName() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	filePath := path.Join(testDirPath, FileName1)
	operations.CreateFile(filePath, FilePerms, t.T())
	_, err := os.Stat(filePath)
	require.NoError(t.T(), err)
	err = os.Remove(filePath)
	require.NoError(t.T(), err)
	operations.CreateFile(filePath, FilePerms, t.T())

	f, err := os.Stat(filePath)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), FileName1, f.Name())
	assert.False(t.T(), f.IsDir())
}
