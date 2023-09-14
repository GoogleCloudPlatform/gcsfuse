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

package implicit_dir

import (
	"io/fs"
	"path"
	"path/filepath"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/tools/integration_tests/local_file/helpers"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestNewFileUnderImplicitDirectoryShouldNotGetSyncedToGCSTillClose(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create file in implicit directory.
	err := CreateObject(path.Join(ImplicitDirName, ImplicitFileName1), FileContents)
	if err != nil {
		t.Errorf("Error while creating implicit directory, err: %v", err)
	}

	// Validate.
	NewFileShouldGetSyncedToGCSAtClose(path.Join(ImplicitDirName, FileName1), t)
}

func TestReadDirForImplicitDirWithLocalFile(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create implicit dir with 2 local files and 1 synced file.
	err := CreateObject(path.Join(ImplicitDirName, ImplicitFileName1), "")
	if err != nil {
		t.Errorf("Error while creating implicit directory, err: %v", err)
	}
	_, fh1 := CreateLocalFile(path.Join(ImplicitDirName, FileName1), t)
	_, fh2 := CreateLocalFile(path.Join(ImplicitDirName, FileName2), t)

	// Attempt to list implicit directory.
	entries := ReadDirectory(path.Join(setup.MntDir(), ImplicitDirName), t)

	// Verify entries received successfully.
	VerifyCountOfEntries(3, len(entries), t)
	VerifyLocalFileEntry(entries[0], FileName1, 0, t)
	VerifyLocalFileEntry(entries[1], FileName2, 0, t)
	VerifyLocalFileEntry(entries[2], ImplicitFileName1, 0, t)
	// Close the local files.
	CloseFileAndValidateObjectContents(fh1, path.Join(ImplicitDirName, FileName1), "", t)
	CloseFileAndValidateObjectContents(fh2, path.Join(ImplicitDirName, FileName2), "", t)
}

func TestRecursiveListingWithLocalFiles(t *testing.T) {
	// Structure
	// mntDir/
	//	   - foo1 						--- file
	//     - explicit/		    --- directory
	//		   - explicitFile1  --- file
	//	   - implicit/ 				--- directory
	//		   - foo2  					--- file
	//		   - implicitFile1	--- file

	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create local file in mnt/ dir.
	_, fh1 := CreateLocalFile(FileName1, t)
	// Create explicit dir with 1 local file.
	CreateExplicitDirShouldNotThrowError(t)
	_, fh2 := CreateLocalFile(path.Join(ExplicitDirName, ExplicitFileName1), t)
	// Create implicit dir with 1 local file1 and 1 synced file.
	err := CreateObject(path.Join(ImplicitDirName, ImplicitFileName1), "")
	if err != nil {
		t.Errorf("Error while creating implicit directory, err: %v", err)
	}
	_, fh3 := CreateLocalFile(path.Join(ImplicitDirName, FileName2), t)

	// Recursively list mntDir/ directory.
	err = filepath.WalkDir(setup.MntDir(), func(walkPath string, dir fs.DirEntry, err error) error {
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
			// numberOfObjects = 3
			VerifyCountOfEntries(3, len(objs), t)
			VerifyDirectoryEntry(objs[0], ExplicitDirName, t)
			VerifyLocalFileEntry(objs[1], FileName1, 0, t)
			VerifyDirectoryEntry(objs[2], ImplicitDirName, t)
		}

		// Check if mntDir/explicitFoo/ has correct objects.
		if walkPath == path.Join(setup.MntDir(), ExplicitDirName) {
			// numberOfObjects = 1
			VerifyCountOfEntries(1, len(objs), t)
			VerifyLocalFileEntry(objs[0], ExplicitFileName1, 0, t)
		}

		// Check if mntDir/implicitFoo/ has correct objects.
		if walkPath == path.Join(setup.MntDir(), ImplicitDirName) {
			// numberOfObjects = 2
			VerifyCountOfEntries(2, len(objs), t)
			VerifyLocalFileEntry(objs[0], FileName2, 0, t)
			VerifyLocalFileEntry(objs[1], ImplicitFileName1, 0, t)
		}
		return nil
	})

	// Validate and close the files.
	if err != nil {
		t.Errorf("filepath.WalkDir() err: %v", err)
	}
	CloseFileAndValidateObjectContents(fh1, FileName1, "", t)
	CloseFileAndValidateObjectContents(fh2, path.Join(ExplicitDirName, ExplicitFileName1), "", t)
	CloseFileAndValidateObjectContents(fh3, path.Join(ImplicitDirName, FileName2), "", t)
}
