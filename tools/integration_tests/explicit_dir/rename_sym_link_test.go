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

package explicit_dir_test

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

func TestRenameSymlinkToExplicitDir(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForExplicitDirTests)
	targetDirName := "target_dir"
	targetDirPath := path.Join(testDir, targetDirName)
	err := os.Mkdir(targetDirPath, setup.DirPermission_0755)
	require.NoError(t, err)
	oldSymlinkPath := path.Join(testDir, "symlink_old")
	err = os.Symlink(targetDirPath, oldSymlinkPath)
	require.NoError(t, err)
	newSymlinkPath := path.Join(testDir, "symlink_new")

	err = os.Rename(oldSymlinkPath, newSymlinkPath)

	require.NoError(t, err)
	_, err = os.Lstat(oldSymlinkPath)
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
	fi, err := os.Lstat(newSymlinkPath)
	require.NoError(t, err)
	assert.Equal(t, os.ModeSymlink, fi.Mode()&os.ModeType)
	targetRead, err := os.Readlink(newSymlinkPath)
	require.NoError(t, err)
	assert.Equal(t, targetDirPath, targetRead)
	targetFi, err := os.Stat(newSymlinkPath)
	require.NoError(t, err)
	assert.True(t, targetFi.IsDir())
}
