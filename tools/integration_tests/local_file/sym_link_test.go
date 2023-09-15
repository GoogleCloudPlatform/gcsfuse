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
// Provides integration tests for symlink operation on local files.
package local_file_test

import (
	"os"
	"path"
	"strings"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/tools/integration_tests/local_file/helpers"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestCreateSymlinkForLocalFile(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(LocalFileTestDirInBucket)
	// Create a local file.
	filePath, fh := CreateLocalFileInTestDir(testDirPath, FileName1, t)
	WritingToLocalFileShouldNotWriteToGCS(fh, FileName1, t)

	// Create the symlink.
	symlinkName := path.Join(testDirPath, "bar")
	SymLinkShouldNotThrowError(filePath, symlinkName, t)

	// Read the link.
	VerifyReadLink(filePath, symlinkName, t)
	VerifyReadFile(symlinkName, t)
	CloseFileAndValidateObjectContents(fh, FileName1, FileContents, t)
}

func TestReadSymlinkForDeletedLocalFile(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(LocalFileTestDirInBucket)
	// Create a local file.
	filePath, fh := CreateLocalFileInTestDir(testDirPath, FileName1, t)
	WritingToLocalFileShouldNotWriteToGCS(fh, FileName1, t)

	// Create the symlink.
	symlinkName := path.Join(testDirPath, "bar")
	SymLinkShouldNotThrowError(filePath, symlinkName, t)

	// Read the link.
	VerifyReadLink(filePath, symlinkName, t)
	// Remove filePath and then close the fileHandle to avoid syncing to GCS.
	UnlinkShouldNotThrowError(filePath, t)
	CloseFile(fh, FileName1, t)
	ValidateObjectNotFoundErrOnGCS(FileName1, t)

	// Reading symlink should fail.
	_, err := os.Stat(symlinkName)
	if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("Reading symlink for deleted local file did not fail.")
	}
}
