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

// Provides integration tests for write on local files.
package local_file

import (
	. "github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

func (t *LocalFileTestSuite) TestMultipleWritesToLocalFile() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())

	// Write some contents to file sequentially.
	for range 3 {
		operations.WriteWithoutClose(fh, FileContents, t.T())
	}
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t.T())

	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName,
		FileName1, FileContents+FileContents+FileContents, t.T())
}

func (t *LocalFileTestSuite) TestRandomWritesToLocalFile() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())

	// Write some contents to file randomly.
	operations.WriteAt("string1", 0, fh, t.T())
	operations.WriteAt("string2", 2, fh, t.T())
	operations.WriteAt("string3", 3, fh, t.T())

	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, FileName1, "stsstring3", t.T())
}

func (t *LocalFileTestSuite) TestOutOfOrderWritesToNewFile() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())

	// Write some contents to file sequentially.
	for range 2 {
		operations.WriteWithoutClose(fh, FileContents, t.T())
	}
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t.T())

	// Write at previous offset.
	operations.WriteAt("hello", 0, fh, t.T())

	expectedString := "hellotringtestString"
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName,
		FileName1, expectedString, t.T())
}

func (t *LocalFileTestSuite) TestMultipleOutOfOrderWritesToNewFile() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())

	// Write some contents to file sequentially.
	for range 2 {
		operations.WriteWithoutClose(fh, FileContents, t.T())
	}
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t.T())

	// Write at previous offset.
	operations.WriteAt("hello", 15, fh, t.T())
	// Write at new offset.
	operations.WriteAt("hey", 30, fh, t.T())

	emptyBytes := [10]byte{}
	expectedString := "testStringtestShello" + string(emptyBytes[:]) + "hey"
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName,
		FileName1, expectedString, t.T())
}

func (t *LocalFileTestSuite) TestWritesToNewFileStartingAtNonZeroOffset() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())
	// Write at future offset.
	operations.WriteAt("hello", 15, fh, t.T())
	// Write at zero offset now.
	operations.WriteAt("hey", 0, fh, t.T())

	emptyBytes := [12]byte{}
	expectedString := "hey" + string(emptyBytes[:]) + "hello"
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName,
		FileName1, expectedString, t.T())
}
