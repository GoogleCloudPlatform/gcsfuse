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
	"os"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type folderOperationTests struct {
	suite.Suite
}

func (t *folderOperationTests) SetupSuite() {
	testEnv.testDirPath = filepath.Join(setup.MntDir(), testDirName)
	err := os.MkdirAll(testEnv.testDirPath, 0755)
	if err != nil {
		t.T().Fatalf("Failed to create directory: %v", err)
	}

}

func (t *folderOperationTests) TearDownSuite() {
	err := os.RemoveAll(testEnv.testDirPath)
	assert.NoError(t.T(), err)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

// Test all the folder operations in the mount.
func (t *folderOperationTests) TestDirOperations() {
	dirName := "dir" + setup.GenerateRandomString(5)
	mountedDirPath := filepath.Join(testEnv.testDirPath, dirName)

	// Check that the directory does not exist initially.
	_, err := os.Stat(mountedDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err), "Directory should not exist before creation")

	// Create the directory.
	err = os.Mkdir(mountedDirPath, setup.FilePermission_0600)
	assert.NoError(t.T(), err)

	// Check that the directory now exists.
	_, err = os.Stat(mountedDirPath)
	assert.NoError(t.T(), err)

	// Rename the directory.
	renamedDirName := "dir" + setup.GenerateRandomString(5)
	mountedRenamedDirPath := filepath.Join(testEnv.testDirPath, renamedDirName)
	err = os.Rename(mountedDirPath, mountedRenamedDirPath)
	assert.NoError(t.T(), err)

	// Check that the old path no longer exists and the new one does.
	_, err = os.Stat(mountedDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err), "Old directory path should not exist after rename")
	_, err = os.Stat(mountedRenamedDirPath)
	assert.NoError(t.T(), err, "New directory path should exist after rename")

	// Remove the directory.
	err = os.RemoveAll(mountedRenamedDirPath)
	assert.NoError(t.T(), err)

	// Check that the directory is removed.
	_, err = os.Stat(mountedRenamedDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err), "Directory should not exist after removal")
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestFolderOperations(t *testing.T) {
	suite.Run(t, new(folderOperationTests))
}
