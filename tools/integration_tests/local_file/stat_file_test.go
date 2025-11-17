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

// Provides integration tests for stat operation on local files.
package local_file

import (
	"os"

	"github.com/vipnydav/gcsfuse/v3/internal/fs/inode"
	. "github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

func (t *LocalFileTestSuite) TestStatOnLocalFile() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	filePath, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())

	// Stat the local file.
	operations.VerifyStatFile(filePath, 0, FilePerms, t.T())

	// Writing contents to local file shouldn't create file on GCS.
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, FileName1, t.T())

	// Stat the local file again to check if new content is written.
	operations.VerifyStatFile(filePath, SizeOfFileContents, FilePerms, t.T())

	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName,
		FileName1, FileContents, t.T())
}

func (t *LocalFileTestSuite) TestStatOnLocalFileWithConflictingFileNameSuffix() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	filePath, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())

	// Stat the local file.
	operations.VerifyStatFile(filePath+inode.ConflictingFileNameSuffix, 0, FilePerms, t.T())

	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName,
		FileName1, "", t.T())
}

func (t *LocalFileTestSuite) TestTruncateLocalFileToSmallerSize() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	fileName := FileName1 + setup.GenerateRandomString(5)
	filePath, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName, t.T())
	// Writing contents to local file .
	operations.WriteWithoutClose(fh, FileContents, t.T())

	// Stat the file to validate if new contents are written.
	operations.VerifyStatFile(filePath, SizeOfFileContents, FilePerms, t.T())

	// Truncate the file to update file size to smaller file size.
	err := os.Truncate(filePath, SmallerSizeTruncate)
	if err != nil {
		t.T().Fatalf("os.Truncate err: %v", err)
	}

	// Stat the file to validate if file is truncated correctly.
	operations.VerifyStatFile(filePath, SmallerSizeTruncate, FilePerms, t.T())

	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName,
		fileName, FileContents[:SmallerSizeTruncate], t.T())
}
