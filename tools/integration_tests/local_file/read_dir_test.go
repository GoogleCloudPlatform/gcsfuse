// Copyright 2023 Google Inc. All Rights Reserved.
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
	"strings"
	"testing"
	"time"

	. "github.com/googlecloudplatform/gcsfuse/tools/integration_tests/local_file/helpers"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestReadDir(t *testing.T) {
	// Structure
	// mntDir/
	// mntDir/explicit/		    				--- directory
	// mntDir/explicit/explicitFile1  --- file
	// mntDir/foo1 										--- empty local file
	// mntDir/foo2  									--- non empty local file
	// mntDir/foo3										--- gcs synced file

	setup.SetupTestDirectory(testDirPath)
	// Create explicit dir with 1 local file.
	CreateExplicitDirInTestDir(testDirPath, t)
	_, fh1 := CreateLocalFileInTestDir(testDirPath, path.Join(ExplicitDirName, ExplicitFileName1), t)
	// Create empty local file.
	_, fh2 := CreateLocalFileInTestDir(testDirPath, FileName1, t)
	// Create non-empty local file.
	_, fh3 := CreateLocalFileInTestDir(testDirPath, FileName2, t)
	WritingToLocalFileShouldNotWriteToGCS(fh3, FileName2, t)
	// Create GCS synced file.
	CreateObjectInGCSTestDir(FileName3, GCSFileContent, t)

	// Attempt to list mnt and explicit directory.
	entriesMnt := ReadDirectory(testDirPath, t)
	entriesDir := ReadDirectory(path.Join(testDirPath, ExplicitDirName), t)

	// Verify entriesMnt received successfully.
	VerifyCountOfEntries(4, len(entriesMnt), t)
	VerifyDirectoryEntry(entriesMnt[0], ExplicitDirName, t)
	VerifyLocalFileEntry(entriesMnt[1], FileName1, 0, t)
	VerifyLocalFileEntry(entriesMnt[2], FileName2, SizeOfFileContents, t)
	VerifyLocalFileEntry(entriesMnt[3], FileName3, SizeOfGCSContent, t)
	// Verify entriesDir received successfully.
	VerifyCountOfEntries(1, len(entriesDir), t)
	VerifyLocalFileEntry(entriesDir[0], ExplicitFileName1, 0, t)
	// Close the local files.
	CloseFileAndValidateObjectContents(fh1, path.Join(ExplicitDirName, ExplicitFileName1), "", t)
	CloseFileAndValidateObjectContents(fh2, FileName1, "", t)
	CloseFileAndValidateObjectContents(fh3, FileName2, FileContents, t)
	ValidateObjectContents(FileName3, GCSFileContent, t)
}

func TestRecursiveListingWithLocalFiles(t *testing.T) {
	// Structure
	// mntDir/
	// mntDir/foo1 										--- file
	// mntDir/explicit/		    				--- directory
	// mntDir/explicit/explicitFile1  --- file

	setup.SetupTestDirectory(testDirPath)
	// Create local file in mnt/ dir.
	_, fh1 := CreateLocalFileInTestDir(testDirPath, FileName1, t)
	// Create explicit dir with 1 local file.
	CreateExplicitDirInTestDir(testDirPath, t)
	_, fh2 := CreateLocalFileInTestDir(testDirPath, path.Join(ExplicitDirName, ExplicitFileName1), t)

	// Recursively list mntDir/ directory.
	err := filepath.WalkDir(testDirPath, func(walkPath string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// The object type is not directory.
		if !dir.IsDir() {
			return nil
		}

		objs := ReadDirectory(walkPath, t)

		// Check if mntDir has correct objects.
		if walkPath == testDirPath {
			// numberOfObjects = 2
			VerifyCountOfEntries(2, len(objs), t)
			VerifyDirectoryEntry(objs[0], ExplicitDirName, t)
			VerifyLocalFileEntry(objs[1], FileName1, 0, t)
		}

		// Check if mntDir/explicit/ has correct objects.
		if walkPath == path.Join(setup.MntDir(), ExplicitDirName) {
			// numberOfObjects = 1
			VerifyCountOfEntries(1, len(objs), t)
			VerifyLocalFileEntry(objs[0], ExplicitFileName1, 0, t)
		}

		return nil
	})

	// Validate and close the files.
	if err != nil {
		t.Fatalf("filepath.WalkDir() err: %v", err)
	}
	CloseFileAndValidateObjectContents(fh1, FileName1, "", t)
	CloseFileAndValidateObjectContents(fh2, path.Join(ExplicitDirName, ExplicitFileName1), "", t)
}

func TestReadDirWithSameNameLocalAndGCSFile(t *testing.T) {
	setup.SetupTestDirectory(testDirPath)
	// Create local file.
	_, fh1 := CreateLocalFileInTestDir(testDirPath, FileName1, t)
	// Create same name gcs file.
	time.Sleep(2 * time.Second)
	CreateObjectInGCSTestDir(FileName1, GCSFileContent, t)

	// Attempt to list mntDir.
	_, err := os.ReadDir(testDirPath)
	if err == nil || !strings.Contains(err.Error(), "input/output error") {
		t.Fatalf("Expected error: %s, Got error: %v", "input/output error", err)
	}

	// Close the local file.
	CloseFile(fh1, FileName1, t)
}
