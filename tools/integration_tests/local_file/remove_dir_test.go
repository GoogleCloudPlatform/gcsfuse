// Copyright 2023 Google Inc. All Rights Reserved.
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
//
// Provides integration tests for removeDir operation on directories containing local files.
package local_file_test

import (
	"path"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/tools/integration_tests/local_file/helpers"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestRmDirOfDirectoryContainingGCSAndLocalFiles(t *testing.T) {
	setup.SetupTestDirectory(testDirPath)
	// Create explicit directory with one synced and one local file.
	CreateExplicitDirInTestDir(testDirPath, t)
	syncedFile := path.Join(ExplicitDirName, FileName1)
	localFile := path.Join(ExplicitDirName, FileName2)
	_, fh1 := CreateLocalFileInTestDir(testDirPath, syncedFile, t)
	CloseFileAndValidateObjectContents(fh1, syncedFile, "", t)
	_, fh2 := CreateLocalFileInTestDir(testDirPath, localFile, t)

	// Attempt to remove explicit directory.
	RemoveDirShouldNotThrowError(path.Join(testDirPath, ExplicitDirName), t)

	// Verify that directory is removed.
	ValidateNoFileOrDirError(path.Join(testDirPath, ExplicitDirName), t)
	// Validate writing content to unlinked local file does not throw error.
	WritingToFileSHouldNotThrowError(fh2, FileContents, t)
	// Validate flush file does not throw error and does not create object on GCS.
	CloseFile(fh2, localFile, t)
	ValidateObjectNotFoundErrOnGCS(localFile, t)
	// Validate synced files are also deleted.
	ValidateObjectNotFoundErrOnGCS(syncedFile, t)
	ValidateObjectNotFoundErrOnGCS(ExplicitDirName, t)
}

func TestRmDirOfDirectoryContainingOnlyLocalFiles(t *testing.T) {
	setup.SetupTestDirectory(testDirPath)
	// Create a directory with two local files.
	CreateExplicitDirInTestDir(testDirPath, t)
	localFile1 := path.Join(ExplicitDirName, FileName1)
	localFile2 := path.Join(ExplicitDirName, FileName2)
	_, fh1 := CreateLocalFileInTestDir(testDirPath, localFile1, t)
	_, fh2 := CreateLocalFileInTestDir(testDirPath, localFile2, t)

	// Attempt to remove explicit directory.
	RemoveDirShouldNotThrowError(path.Join(testDirPath, ExplicitDirName), t)

	// Verify rmDir operation succeeds.
	ValidateNoFileOrDirError(path.Join(testDirPath, ExplicitDirName), t)
	// Close the local files and validate they are not present on GCS.
	CloseFile(fh1, localFile1, t)
	ValidateObjectNotFoundErrOnGCS(localFile1, t)
	CloseFile(fh2, localFile2, t)
	ValidateObjectNotFoundErrOnGCS(localFile2, t)
	// Validate directory is also deleted.
	ValidateObjectNotFoundErrOnGCS(ExplicitDirName, t)
}
