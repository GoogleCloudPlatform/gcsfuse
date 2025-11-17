// Copyright 2025 Google LLC
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

package requester_pays_bucket

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type operationTests struct {
	suite.Suite
}

func (t *operationTests) SetupSuite() {
	testEnv.testDirPath = filepath.Join(setup.MntDir(), testDirName)
	err := os.MkdirAll(testEnv.testDirPath, 0755)
	if err != nil {
		t.T().Fatalf("Failed to create directory: %v", err)
	}

}

func (t *operationTests) TearDownSuite() {
	err := os.RemoveAll(testEnv.testDirPath)
	assert.NoError(t.T(), err)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

// Test all the folder operations in the mount.
func (t *operationTests) TestDirOperations() {
	var fi fs.FileInfo
	var err error
	dirName := "dir" + setup.GenerateRandomString(5)
	mountedDirPath := filepath.Join(testEnv.testDirPath, dirName)

	// Check that the directory does not exist initially.
	_, err = os.Stat(mountedDirPath)
	require.Error(t.T(), err)
	require.True(t.T(), os.IsNotExist(err), "Directory should not exist before creation")

	// Create the directory.
	err = os.Mkdir(mountedDirPath, setup.FilePermission_0600)
	require.NoError(t.T(), err)

	// Check that the directory now exists.
	fi, err = os.Stat(mountedDirPath)
	require.NoError(t.T(), err)
	require.True(t.T(), fi.IsDir(), "%q should be a directory", mountedDirPath)

	// Rename the directory.
	renamedDirName := "dir" + setup.GenerateRandomString(5)
	mountedRenamedDirPath := filepath.Join(testEnv.testDirPath, renamedDirName)
	err = os.Rename(mountedDirPath, mountedRenamedDirPath)
	require.NoError(t.T(), err)

	// Check that the old path no longer exists and the new one does.
	_, err = os.Stat(mountedDirPath)
	require.Error(t.T(), err)
	require.True(t.T(), os.IsNotExist(err), "Old directory path should not exist after rename")
	fi, err = os.Stat(mountedRenamedDirPath)
	require.NoError(t.T(), err, "New directory path should exist after rename")
	require.True(t.T(), fi.IsDir(), "%q should be a directory", mountedRenamedDirPath)

	// Remove the directory.
	err = os.RemoveAll(mountedRenamedDirPath)
	require.NoError(t.T(), err)

	// Check that the directory is removed.
	_, err = os.Stat(mountedRenamedDirPath)
	require.Error(t.T(), err)
	require.True(t.T(), os.IsNotExist(err), "Directory should not exist after removal")
}

// Test all the file operations in the mount.
func (t *operationTests) TestFileOperations() {
	var fi fs.FileInfo
	var err error
	contentLength := 5
	content := setup.GenerateRandomString(contentLength)
	fileName := "file" + setup.GenerateRandomString(5)
	mountedFilePath := filepath.Join(testEnv.testDirPath, fileName)

	// Check that the file does not exist initially.
	_, err = os.Stat(mountedFilePath)
	require.Error(t.T(), err)
	require.True(t.T(), os.IsNotExist(err), "File should not exist before creation")

	// Create the file.
	operations.CreateFileWithContent(mountedFilePath, setup.FilePermission_0600, content, t.T())

	// Check that the file now exists.
	fi, err = os.Stat(mountedFilePath)
	require.NoError(t.T(), err)
	require.False(t.T(), fi.IsDir(), "%q should be a file", mountedFilePath)

	// Rename the file.
	renamedFileName := "rename-output-file-" + setup.GenerateRandomString(5)
	mountedRenamedFilePath := filepath.Join(testEnv.testDirPath, renamedFileName)
	err = os.Rename(mountedFilePath, mountedRenamedFilePath)
	require.NoError(t.T(), err)

	// Check that the old path no longer exists and the new one does.
	_, err = os.Stat(mountedFilePath)
	require.Error(t.T(), err)
	require.True(t.T(), os.IsNotExist(err), "Old file path should not exist after rename")
	fi, err = os.Stat(mountedRenamedFilePath)
	require.NoError(t.T(), err, "New file path should exist after rename")
	require.False(t.T(), fi.IsDir(), "%q should be a file", mountedRenamedFilePath)

	// Remove the renamed file.
	err = os.RemoveAll(mountedRenamedFilePath)
	require.NoError(t.T(), err)

	// Check that the renamed file is removed.
	_, err = os.Stat(mountedRenamedFilePath)
	require.Error(t.T(), err)
	require.True(t.T(), os.IsNotExist(err), "file should not exist after removal")
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestOperations(t *testing.T) {
	suite.Run(t, new(operationTests))
}
