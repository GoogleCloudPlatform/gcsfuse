// Copyright 2024 Google LLC
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

// Provides integration tests for rename dir.
package operations_test

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

func TestRenameDirToNonEmptyDestDirectory(t *testing.T) {
	// Set up the test directory.
	testDir := setup.SetupTestDirectory(DirForOperationTests)

	// Create source directory.
	srcDirPath := path.Join(testDir, "srcDir")
	err := os.Mkdir(srcDirPath, 0700)
	assert.NoError(t, err)

	// Create destination directory and put a file in it.
	destDirPath := path.Join(testDir, "destDir")
	err = os.Mkdir(destDirPath, 0700)
	assert.NoError(t, err)

	destFilePath := path.Join(destDirPath, "file.txt")
	operations.CreateFileWithContent(destFilePath, setup.FilePermission_0600, Content, t)

	// Attempt to rename the source directory to the destination directory.
	err = os.Rename(srcDirPath, destDirPath)

	// Assert that an error occurred (because destination is not empty).
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "file exists") || strings.Contains(err.Error(), "directory not empty") || strings.Contains(err.Error(), "not empty"), "Error message should mention file exists or not empty, but got: %v", err)

	// If the issue was not fixed, this process could hang on further operations due to leaked locks.
	// We do some operations to confirm things are healthy.
	_, err = os.Stat(srcDirPath)
	assert.NoError(t, err)

	_, err = os.Stat(destDirPath)
	assert.NoError(t, err)

	// Delete both directories successfully
	err = os.Remove(destFilePath)
	assert.NoError(t, err)
	err = os.Remove(destDirPath)
	assert.NoError(t, err)
	err = os.Remove(srcDirPath)
	assert.NoError(t, err)
}
