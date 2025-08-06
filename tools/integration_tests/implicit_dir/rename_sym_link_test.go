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

package implicit_dir_test

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenameSymlinkToImplicitDir(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForImplicitDirTests)

	// Create an object that defines an implicit directory.
	objectNameInGCS := path.Join(DirForImplicitDirTests, "implicit_dir/file.txt")
	implicitDirFileContent := "taco"
	err := client.CreateObjectOnGCS(testEnv.ctx, testEnv.storageClient, objectNameInGCS, implicitDirFileContent)
	require.NoError(t, err)

	// Create and rename the symlink.
	targetPath := path.Join("implicit_dir", "file.txt")
	oldSymlinkPath := path.Join(testDir, "symlink_old")
	err = os.Symlink(targetPath, oldSymlinkPath)
	require.NoError(t, err)
	newSymlinkPath := path.Join(testDir, "symlink_new")
	err = os.Rename(oldSymlinkPath, newSymlinkPath)
	require.NoError(t, err)

	// Assertions
	_, err = os.Lstat(oldSymlinkPath)
	assert.True(t, os.IsNotExist(err))
	fi, err := os.Lstat(newSymlinkPath)
	assert.NoError(t, err)
	assert.Equal(t, os.ModeSymlink, fi.Mode()&os.ModeType)
	targetRead, err := os.Readlink(newSymlinkPath)
	assert.NoError(t, err)
	assert.Equal(t, targetPath, targetRead)
	content, err := operations.ReadFile(newSymlinkPath)
	assert.NoError(t, err)
	assert.Equal(t, implicitDirFileContent, string(content))
}
