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

	. "github.com/googlecloudplatform/gcsfuse/tools/integration_tests/local_file/helpers"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestReadDirWithEmptyLocalFiles(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create local files.
	_, fh1 := CreateLocalFile(FileName1, t)
	_, fh2 := CreateLocalFile(FileName2, t)

	// Attempt to list mntDir.
	entries := ReadDirectory(setup.MntDir(), t)

	// Verify entries received successfully.
	VerifyCountOfEntries(2, len(entries), t)
	VerifyLocalFileEntry(entries[0], FileName1, 0, t)
	VerifyLocalFileEntry(entries[1], FileName2, 0, t)
	// Close the local files.
	CloseFileAndValidateObjectContents(fh1, FileName1, "", t)
	CloseFileAndValidateObjectContents(fh2, FileName2, "", t)
}

func TestReadDirWithNonEmptyLocalFile(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create local files.
	_, fh := CreateLocalFile(FileName1, t)
	WritingToLocalFileShouldNotWriteToGCS(fh, FileName1, t)

	// Attempt to list mntDir.
	entries := ReadDirectory(setup.MntDir(), t)

	// Verify entries received successfully.
	VerifyCountOfEntries(1, len(entries), t)
	VerifyLocalFileEntry(entries[0], FileName1, 10, t)
	// Close the local files.
	CloseFileAndValidateObjectContents(fh, FileName1, FileContents, t)
}

func TestReadDirForExplicitDirWithLocalFile(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create explicit dir with 2 local files.
	CreateExplicitDirShouldNotThrowError(t)
	_, fh1 := CreateLocalFile(path.Join(ExplicitDirName, FileName1), t)
	_, fh2 := CreateLocalFile(path.Join(ExplicitDirName, FileName2), t)

	// Attempt to list explicit directory.
	entries := ReadDirectory(path.Join(setup.MntDir(), ExplicitDirName), t)

	// Verify entries received successfully.
	VerifyCountOfEntries(2, len(entries), t)
	VerifyLocalFileEntry(entries[0], FileName1, 0, t)
	VerifyLocalFileEntry(entries[1], FileName2, 0, t)
	// Close the local files.
	CloseFileAndValidateObjectContents(fh1, path.Join(ExplicitDirName, FileName1), "", t)
	CloseFileAndValidateObjectContents(fh2, path.Join(ExplicitDirName, FileName2), "", t)
}

func TestRecursiveListingWithLocalFiles(t *testing.T) {
	// Structure
	// mntDir/
	//	   - foo1 						--- file
	//     - explicit/		    --- directory
	//		   - explicitFile1  --- file

	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create local file in mnt/ dir.
	_, fh1 := CreateLocalFile(FileName1, t)
	// Create explicit dir with 1 local file.
	CreateExplicitDirShouldNotThrowError(t)
	_, fh2 := CreateLocalFile(path.Join(ExplicitDirName, ExplicitFileName1), t)

	// Recursively list mntDir/ directory.
	err := filepath.WalkDir(setup.MntDir(), func(walkPath string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// The object type is not directory.
		if !dir.IsDir() {
			return nil
		}

		objs := ReadDirectory(walkPath, t)

		// Check if mntDir has correct objects.
		if walkPath == setup.MntDir() {
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
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create local file.
	_, fh1 := CreateLocalFile(FileName1, t)
	// Create same name gcs file.
	err := CreateObject(FileName1, GCSFileContent)
	if err != nil {
		t.Fatalf("Create Object on GCS: %v.", err)
	}

	// Attempt to list mntDir.
	_, err = os.ReadDir(setup.MntDir())
	if err == nil || !strings.Contains(err.Error(), "input/output error") {
		t.Fatalf("Expected error: %s, Got error: %v", "input/output error", err)
	}

	// Close the local file.
	CloseLocalFile(fh1, FileName1, t)
}
