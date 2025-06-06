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

// Provides integration tests for read operation on local files.
package local_file

import (
	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *CommonLocalFileTestSuite) TestReadLocalFile() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())

	// Write FileContents twice to local file.
	content := FileContents + FileContents
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, FileName1, t.T())
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, FileName1, t.T())

	// Read the local file contents.
	buf := make([]byte, len(content))
	n, err := fh.ReadAt(buf, 0)
	if err != nil || len(content) != n || content != string(buf) {
		t.T().Fatalf("Read file operation failed on local file: %v "+
			"Expected content: %s, Got Content: %s", err, content, string(buf))
	}

	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, FileName1, content, t.T())
}
