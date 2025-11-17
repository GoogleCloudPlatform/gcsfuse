// Copyright 2025 Google LLC
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

package local_file

import (
	"os"
	"path"

	"github.com/stretchr/testify/require"
	. "github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

func (t *LocalFileTestSuite) TestEditsToNewlyCreatedFile() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())
	// Write some contents to file sequentially.
	for range 3 {
		operations.WriteWithoutClose(fh, FileContents, t.T())
	}
	// Close the file and validate that the file is created on GCS.
	expectedContent := FileContents + FileContents + FileContents
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, FileName1, expectedContent, t.T())

	// Perform edit
	fhNew := operations.OpenFile(path.Join(testDirPath, FileName1), t.T())
	newContent := "newContent"
	_, err := fhNew.WriteAt([]byte(newContent), 0)

	require.Nil(t.T(), err)
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fhNew, testDirName, FileName1, newContent+FileContents+FileContents, t.T())
}

func (t *LocalFileTestSuite) TestAppendsToNewlyCreatedFile() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())
	// Write some contents to file sequentially.
	for range 3 {
		operations.WriteWithoutClose(fh, FileContents, t.T())
	}
	// Close the file and validate that the file is created on GCS.
	expectedContent := FileContents + FileContents + FileContents
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, FileName1, expectedContent, t.T())

	// Append to the file.
	fhNew, err := os.OpenFile(path.Join(testDirPath, FileName1), os.O_RDWR|os.O_APPEND, operations.FilePermission_0777)
	require.Nil(t.T(), err)
	appendedContent := "appendedContent"
	_, err = fhNew.Write([]byte(appendedContent))

	require.Nil(t.T(), err)
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fhNew, testDirName, FileName1, expectedContent+appendedContent, t.T())
}
