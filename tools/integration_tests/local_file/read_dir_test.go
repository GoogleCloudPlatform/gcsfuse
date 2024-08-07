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
package local_file_test

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////
func creatingNLocalFilesShouldNotThrowError(n int, wg *sync.WaitGroup, t *testing.T) {
	defer wg.Done()
	operations.CreateDirectory(path.Join(testDirPath, ExplicitDirName), t)
	for i := 0; i < n; i++ {
		filePath := path.Join(testDirPath, ExplicitDirName, FileName1+strconv.FormatInt(int64(i), 10))
		operations.CreateFile(filePath, FilePerms, t)
	}
}

func readingDirNTimesShouldNotThrowError(n int, wg *sync.WaitGroup, t *testing.T) {
	defer wg.Done()
	for i := 0; i < n; i++ {
		_, err := os.ReadDir(setup.MntDir())
		if err != nil {
			t.Errorf("Error while reading directory %dth time: %v", i, err)
		}
	}
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func TestReadDir(t *testing.T) {
	// Structure
	// mntDir/
	// mntDir/explicit/		    				--- directory
	// mntDir/explicit/explicitFile1  --- file
	// mntDir/foo1 										--- empty local file
	// mntDir/foo2  									--- non empty local file
	// mntDir/foo3										--- gcs synced file

	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create explicit dir with 1 local file.
	operations.CreateDirectory(path.Join(testDirPath, ExplicitDirName), t)
	_, fh1 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath,
		path.Join(ExplicitDirName, ExplicitFileName1), t)
	// Create empty local file.
	_, fh2 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
	// Create non-empty local file.
	_, fh3 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName2, t)
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh3, testDirName, FileName2, t)
	// Create GCS synced file.
	CreateObjectInGCSTestDir(ctx, storageClient, testDirName, FileName3, GCSFileContent, t)

	// Attempt to list mnt and explicit directory.
	entriesMnt := operations.ReadDirectory(testDirPath, t)
	entriesDir := operations.ReadDirectory(path.Join(testDirPath, ExplicitDirName), t)

	// Verify entriesMnt received successfully.
	operations.VerifyCountOfDirectoryEntries(4, len(entriesMnt), t)
	operations.VerifyDirectoryEntry(entriesMnt[0], ExplicitDirName, t)
	operations.VerifyFileEntry(entriesMnt[1], FileName1, 0, t)
	operations.VerifyFileEntry(entriesMnt[2], FileName2, SizeOfFileContents, t)
	operations.VerifyFileEntry(entriesMnt[3], FileName3, GCSFileSize, t)
	// Verify entriesDir received successfully.
	operations.VerifyCountOfDirectoryEntries(1, len(entriesDir), t)
	operations.VerifyFileEntry(entriesDir[0], ExplicitFileName1, 0, t)
	// Close the local files.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh1, testDirName,
		path.Join(ExplicitDirName, ExplicitFileName1), "", t)
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh2, testDirName,
		FileName1, "", t)
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh3, testDirName,
		FileName2, FileContents, t)
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, FileName3,
		GCSFileContent, t)
}

func TestRecursiveListingWithLocalFiles(t *testing.T) {
	// Structure
	// mntDir/
	// mntDir/foo1 										--- file
	// mntDir/explicit/		    				--- directory
	// mntDir/explicit/explicitFile1  --- file

	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local file in mnt/ dir.
	_, fh1 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
	// Create explicit dir with 1 local file.
	operations.CreateDirectory(path.Join(testDirPath, ExplicitDirName), t)
	_, fh2 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath,
		path.Join(ExplicitDirName, ExplicitFileName1), t)

	// Recursively list mntDir/ directory.
	err := filepath.WalkDir(testDirPath, func(walkPath string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// The object type is not directory.
		if !dir.IsDir() {
			return nil
		}

		objs := operations.ReadDirectory(walkPath, t)

		// Check if mntDir has correct objects.
		if walkPath == testDirPath {
			// numberOfObjects = 2
			operations.VerifyCountOfDirectoryEntries(2, len(objs), t)
			operations.VerifyDirectoryEntry(objs[0], ExplicitDirName, t)
			operations.VerifyFileEntry(objs[1], FileName1, 0, t)
		}

		// Check if mntDir/explicit/ has correct objects.
		if walkPath == path.Join(setup.MntDir(), ExplicitDirName) {
			// numberOfObjects = 1
			operations.VerifyCountOfDirectoryEntries(1, len(objs), t)
			operations.VerifyFileEntry(objs[0], ExplicitFileName1, 0, t)
		}

		return nil
	})

	// Validate and close the files.
	if err != nil {
		t.Fatalf("filepath.WalkDir() err: %v", err)
	}
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh1, testDirName,
		FileName1, "", t)
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh2, testDirName,
		path.Join(ExplicitDirName, ExplicitFileName1), "", t)
}

func TestReadDirWithSameNameLocalAndGCSFile(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local file.
	_, fh1 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
	// Create same name gcs file.
	time.Sleep(2 * time.Second)
	CreateObjectInGCSTestDir(ctx, storageClient, testDirName, FileName1, GCSFileContent, t)

	// Attempt to list testDir.
	_, err := os.ReadDir(testDirPath)
	if err != nil {
		t.Fatalf("ReadDir err: %v", err)
	}

	// Close the local file.
	operations.CloseFileShouldNotThrowError(fh1, t)
}

func TestConcurrentReadDirAndCreationOfLocalFiles_DoesNotThrowError(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	var wg sync.WaitGroup
	wg.Add(2)

	// Concurrently create 100 local files and read directory 200 times.
	go creatingNLocalFilesShouldNotThrowError(100, &wg, t)
	go readingDirNTimesShouldNotThrowError(200, &wg, t)

	wg.Wait()
}
