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

package rename_symlink

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/local_file"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type renameSymlinkTest struct {
	testDir  string
	filename string
	suite.Suite
}

func (t *renameSymlinkTest) SetupTest() {
	t.testDir = setup.SetupTestDirectory(testDirName)
	t.filename = "oldsymlink"
}

func (t *renameSymlinkTest) createAndVerifySymLink() (filePath, symlink string, fh *os.File) {
	t.testDir = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	filePath, fh = client.CreateLocalFileInTestDir(ctx, storageClient, t.testDir, t.filename, t.T())
	local_file.WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, t.filename, t.T())

	// Create the symlink.
	symlink = path.Join(t.testDir, "bar")
	operations.CreateSymLink(filePath, symlink, t.T())

	// Read the link.
	operations.VerifyReadLink(filePath, symlink, t.T())
	operations.VerifyReadFile(symlink, client.FileContents, t.T())
	return
}

func (t *renameSymlinkTest) TestRenameSymlinkToFile() {
	targetName := "target.txt"
	targetPath := path.Join(t.testDir, targetName)
	err := os.WriteFile(targetPath, []byte("taco"), setup.FilePermission_0600)
	require.NoError(t.T(), err)
	oldSymlinkPath := path.Join(t.testDir, "symlink_old")
	err = os.Symlink(targetPath, oldSymlinkPath)
	require.NoError(t.T(), err)
	newSymlinkPath := path.Join(t.testDir, "symlink_new")

	err = os.Rename(oldSymlinkPath, newSymlinkPath)

	require.NoError(t.T(), err)
	_, err = os.Lstat(oldSymlinkPath)
	require.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err))
	fi, err := os.Lstat(newSymlinkPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), os.ModeSymlink, fi.Mode()&os.ModeType)
	targetRead, err := os.Readlink(newSymlinkPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), targetPath, targetRead)
	content, err := operations.ReadFile(newSymlinkPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), "taco", string(content))
}

func (t *renameSymlinkTest) TestRenameSymlinkForLocalFile() {
	filePath, symlinkPath, fh := t.createAndVerifySymLink()
	newSymlinkPath := path.Join(t.testDir, "newSymlink")

	err := os.Rename(symlinkPath, newSymlinkPath)

	require.NoError(t.T(), err, "os.Rename failed for symlink")
	_, err = os.Lstat(symlinkPath)
	require.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err), "Old symlink should not exist after rename. err: %v", err)
	operations.VerifyReadLink(filePath, newSymlinkPath, t.T())
	operations.VerifyReadFile(newSymlinkPath, client.FileContents, t.T())
	client.CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, t.filename, client.FileContents, t.T())
}

func (t *renameSymlinkTest) TestRenameSymlinkToImplicitDir() {
	implicitDirName := "implicit_dir"
	// Create an object that defines an implicit directory. This creates `implicit_dir/`.
	objectNameInGCS := path.Join(testDirName, implicitDirName, "placeholder")
	err := client.CreateObjectOnGCS(ctx, storageClient, objectNameInGCS, "")
	require.NoError(t.T(), err)
	implicitDirPath := path.Join(t.testDir, implicitDirName)
	oldSymlinkPath := path.Join(t.testDir, "symlink_old")
	err = os.Symlink(implicitDirPath, oldSymlinkPath)
	require.NoError(t.T(), err)
	newSymlinkPath := path.Join(t.testDir, "symlink_new")

	err = os.Rename(oldSymlinkPath, newSymlinkPath)

	require.NoError(t.T(), err)
	_, err = os.Lstat(oldSymlinkPath)
	require.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err))
	fi, err := os.Lstat(newSymlinkPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), os.ModeSymlink, fi.Mode()&os.ModeType)
	targetRead, err := os.Readlink(newSymlinkPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), implicitDirPath, targetRead)
	targetFi, err := os.Stat(newSymlinkPath)
	require.NoError(t.T(), err)
	assert.True(t.T(), targetFi.IsDir())
}

func (t *renameSymlinkTest) TestRenameSymlinkToExplicitDir() {
	targetDirName := "target_dir"
	targetDirPath := path.Join(t.testDir, targetDirName)
	err := os.Mkdir(targetDirPath, setup.DirPermission_0755)
	require.NoError(t.T(), err)
	oldSymlinkPath := path.Join(t.testDir, "symlink_old")
	err = os.Symlink(targetDirPath, oldSymlinkPath)
	require.NoError(t.T(), err)
	newSymlinkPath := path.Join(t.testDir, "symlink_new")

	err = os.Rename(oldSymlinkPath, newSymlinkPath)

	require.NoError(t.T(), err)
	_, err = os.Lstat(oldSymlinkPath)
	require.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err))
	fi, err := os.Lstat(newSymlinkPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), os.ModeSymlink, fi.Mode()&os.ModeType)
	targetRead, err := os.Readlink(newSymlinkPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), targetDirPath, targetRead)
	targetFi, err := os.Stat(newSymlinkPath)
	require.NoError(t.T(), err)
	assert.True(t.T(), targetFi.IsDir())
}

func TestRenameSymlink(t *testing.T) {
	suite.Run(t, new(renameSymlinkTest))
}
