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

// Provides integration tests for create local file.
package local_file_test

import (
	"path"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/tools/integration_tests/local_file/helpers"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestNewFileShouldNotGetSyncedToGCSTillClose(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.PreTestSetup(LocalFileTestDirInBucket)

	// Validate.
	NewFileShouldGetSyncedToGCSAtClose(FileName1, t)
}

func TestNewFileUnderExplicitDirectoryShouldNotGetSyncedToGCSTillClose(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.PreTestSetup(LocalFileTestDirInBucket)
	// Make explicit directory.
	CreateExplicitDirShouldNotThrowError(t)

	// Validate.
	NewFileShouldGetSyncedToGCSAtClose(path.Join(ExplicitDirName, ExplicitFileName1), t)
}

func TestCreateNewFileWhenSameFileExistsOnGCS(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.PreTestSetup(LocalFileTestDirInBucket)
	// Create a local file.
	_, fh := CreateLocalFile(FileName1, t)

	// Create a file on GCS with the same name.
	err := CreateObjectOnGCS(path.Join(LocalFileTestDirInBucket, FileName1), GCSFileContent)
	if err != nil {
		t.Fatalf("Create Object on GCS: %v.", err)
	}

	// Write to local file.
	WritingToLocalFileSHouldNotThrowError(fh, FileContents, t)
	// Close the local file and ensure that the content on GCS is not overwritten.
	CloseFileAndValidateObjectContents(fh, FileName1, GCSFileContent, t)
}
