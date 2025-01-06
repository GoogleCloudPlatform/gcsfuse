// Copyright 2024 Google Inc. All Rights Reserved.
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

package fs_test

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HNSBucketTests struct {
	suite.Suite
	fsTest
}

type dirEntry struct {
	name  string
	isDir bool
}

var expectedFooDirEntries = []dirEntry{
	{name: "test", isDir: true},
	{name: "test2", isDir: true},
	{name: "file1.txt", isDir: false},
	{name: "file2.txt", isDir: false},
	{name: "implicit_dir", isDir: true},
}

func TestHNSBucketTests(t *testing.T) { suite.Run(t, new(HNSBucketTests)) }

func (t *HNSBucketTests) SetupSuite() {
	t.serverCfg.ImplicitDirectories = false
	t.serverCfg.NewConfig = &cfg.Config{
		EnableHns: true,
	}
	t.serverCfg.MetricHandle = common.NewNoopMetrics()
	bucketType = gcs.Hierarchical
	t.fsTest.SetUpTestSuite()
}

func (t *HNSBucketTests) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *HNSBucketTests) SetupTest() {
	err := t.createFolders([]string{"foo/", "bar/", "foo/test2/", "foo/test/"})
	require.NoError(t.T(), err)

	err = t.createObjects(
		map[string]string{
			"foo/file1.txt":              "abcdef",
			"foo/file2.txt":              "xyz",
			"foo/test/file3.txt":         "xyz",
			"foo/implicit_dir/file3.txt": "xxw",
			"bar/file1.txt":              "-1234556789",
		})
	require.NoError(t.T(), err)
}

func (t *HNSBucketTests) TearDownTest() {
	t.fsTest.TearDown()
}

func (t *HNSBucketTests) TestReadDir() {
	dirPath := path.Join(mntDir, "foo")

	dirEntries, err := os.ReadDir(dirPath)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, len(dirEntries))
	actualDirEntries := []dirEntry{}
	for _, d := range dirEntries {
		actualDirEntries = append(actualDirEntries, dirEntry{
			name:  d.Name(),
			isDir: d.IsDir(),
		})
	}
	assert.ElementsMatch(t.T(), actualDirEntries, expectedFooDirEntries)
}

func (t *HNSBucketTests) TestDeleteFolder() {
	dirPath := path.Join(mntDir, "foo")

	err := os.RemoveAll(dirPath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(dirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
}

func (t *HNSBucketTests) TestDeleteImplicitDir() {
	dirPath := path.Join(mntDir, "foo", "implicit_dir")

	err := os.RemoveAll(dirPath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(dirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
}

func (t *HNSBucketTests) TestRenameFolderWithSrcDirectoryDoesNotExist() {
	oldDirPath := path.Join(mntDir, "foo_not_exist")
	newDirPath := path.Join(mntDir, "foo_rename")

	err := os.Rename(oldDirPath, newDirPath)

	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
}

func (t *HNSBucketTests) TestRenameFolderWithDstDirectoryNotEmpty() {
	oldDirPath := path.Join(mntDir, "foo")
	_, err := os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	// In the setup phase, we created file1.txt within the bar directory.
	newDirPath := path.Join(mntDir, "bar")
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)

	err = os.Rename(oldDirPath, newDirPath)

	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "file exists"))
}

func (t *HNSBucketTests) TestRenameFolderWithEmptySourceDirectory() {
	oldDirPath := path.Join(mntDir, "foo", "test2")
	_, err := os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "foo_rename")
	_, err = os.Stat(newDirPath)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))

	err = os.Rename(oldDirPath, newDirPath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)
	dirEntries, err := os.ReadDir(newDirPath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 0, len(dirEntries))
}

func (t *HNSBucketTests) TestRenameFolderWithSourceDirectoryHaveLocalFiles() {
	oldDirPath := path.Join(mntDir, "foo", "test")
	_, err := os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	file, err := os.OpenFile(path.Join(oldDirPath, "file4.txt"), os.O_RDWR|os.O_CREATE, filePerms)
	assert.NoError(t.T(), err)
	defer file.Close()
	newDirPath := path.Join(mntDir, "bar", "foo_rename")

	err = os.Rename(oldDirPath, newDirPath)

	assert.Error(t.T(), err)
	// In the logs, we encountered the following error:
	// "Rename: operation not supported, can't rename directory 'test' with open files: operation not supported."
	// This was translated to an "operation not supported" error at the kernel level.
	assert.True(t.T(), strings.Contains(err.Error(), "operation not supported"))
}

func (t *HNSBucketTests) TestRenameFolderWithSameParent() {
	oldDirPath := path.Join(mntDir, "foo")
	_, err := os.Stat(oldDirPath)
	require.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "foo_rename")
	_, err = os.Stat(newDirPath)
	require.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))

	err = os.Rename(oldDirPath, newDirPath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)
	dirEntries, err := os.ReadDir(newDirPath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, len(dirEntries))
	actualDirEntries := []dirEntry{}
	for _, d := range dirEntries {
		actualDirEntries = append(actualDirEntries, dirEntry{
			name:  d.Name(),
			isDir: d.IsDir(),
		})
	}
	assert.ElementsMatch(t.T(), actualDirEntries, expectedFooDirEntries)
}

func (t *HNSBucketTests) TestRenameFolderWithExistingEmptyDestDirectory() {
	oldDirPath := path.Join(mntDir, "foo", "test")
	_, err := os.Stat(oldDirPath)
	require.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "foo", "test2")
	_, err = os.Stat(newDirPath)
	require.NoError(t.T(), err)

	// Go's Rename function does not support renaming a directory into an existing empty directory.
	// To achieve this, we call a Python rename function as a workaround.
	cmd := exec.Command("python3", "-c", fmt.Sprintf("import os; os.rename('%s', '%s')", oldDirPath, newDirPath))
	_, err = cmd.CombinedOutput()

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)
	dirEntries, err := os.ReadDir(newDirPath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 1, len(dirEntries))
	assert.Equal(t.T(), "file3.txt", dirEntries[0].Name())
	assert.False(t.T(), dirEntries[0].IsDir())
}

func (t *HNSBucketTests) TestRenameFolderWithDifferentParents() {
	oldDirPath := path.Join(mntDir, "foo")
	_, err := os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "bar", "foo_rename")

	err = os.Rename(oldDirPath, newDirPath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)
	dirEntries, err := os.ReadDir(newDirPath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, len(dirEntries))
	actualDirEntries := []dirEntry{}
	for _, d := range dirEntries {
		actualDirEntries = append(actualDirEntries, dirEntry{
			name:  d.Name(),
			isDir: d.IsDir(),
		})
	}
	assert.ElementsMatch(t.T(), actualDirEntries, expectedFooDirEntries)
}

func (t *HNSBucketTests) TestRenameFolderWithOpenGCSFile() {
	oldDirPath := path.Join(mntDir, "bar")
	_, err := os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "bar_rename")
	filePath := path.Join(oldDirPath, "file1.txt")
	f, err := os.Open(filePath)
	require.NoError(t.T(), err)

	err = os.Rename(oldDirPath, newDirPath)

	require.NoError(t.T(), err)
	_, err = f.WriteString("test")
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "bad file descriptor"))
	assert.NoError(t.T(), f.Close())
	_, err = os.Stat(oldDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)
	dirEntries, err := os.ReadDir(newDirPath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 1, len(dirEntries))
	assert.Equal(t.T(), "file1.txt", dirEntries[0].Name())
	assert.False(t.T(), dirEntries[0].IsDir())
}

// Create directory foo.
// Stat the directory foo.
// Rename directory foo --> foo_rename
// Stat the old directory.
// Stat the new directory.
// Read new directory and validate.
// Create old directory again with same name - foo
// Stat the directory - foo
// Read directory again and validate it is empty.
func (t *HNSBucketTests) TestCreateDirectoryWithSameNameAfterRename() {
	oldDirPath := path.Join(mntDir, "foo")
	_, err := os.Stat(oldDirPath)
	require.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "foo_rename")
	// Rename directory foo --> foo_rename
	err = os.Rename(oldDirPath, newDirPath)
	require.NoError(t.T(), err)
	// Stat old directory.
	_, err = os.Stat(oldDirPath)
	require.Error(t.T(), err)
	require.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	// Stat new directory.
	_, err = os.Stat(newDirPath)
	require.NoError(t.T(), err)
	// Read new directory and validate.
	dirEntries, err := os.ReadDir(newDirPath)
	require.NoError(t.T(), err)
	require.Equal(t.T(), 5, len(dirEntries))
	actualDirEntries := []dirEntry{}
	for _, d := range dirEntries {
		actualDirEntries = append(actualDirEntries, dirEntry{
			name:  d.Name(),
			isDir: d.IsDir(),
		})
	}
	require.ElementsMatch(t.T(), actualDirEntries, expectedFooDirEntries)

	// Create old directory again.
	err = os.Mkdir(oldDirPath, dirPerms)

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	dirEntries, err = os.ReadDir(oldDirPath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 0, len(dirEntries))
}

// Create directory - foo/test2
// Create local file in directory - foo/test2/test.txt
// Stat the local file - foo/test2/test.txt
// Delete directory - rm -r foo/test2
// Create directory again - foo/test2
// Create local file with the same name in directory - foo/test2/test.txt
// Stat the local file - foo/test2/test.txt
func (t *HNSBucketTests) TestCreateLocalFileInSamePathAfterDeletingParentDirectory() {
	dirPath := path.Join(mntDir, "foo", "test2")
	filePath := path.Join(dirPath, "test.txt")
	// Create local file in side it.
	f1, err := os.Create(filePath)
	defer require.NoError(t.T(), f1.Close())
	require.NoError(t.T(), err)
	_, err = os.Stat(filePath)
	require.NoError(t.T(), err)
	// Delete directory rm -r foo/test2
	err = os.RemoveAll(dirPath)
	assert.NoError(t.T(), err)
	// Create directory again foo/test2
	err = os.Mkdir(dirPath, dirPerms)
	assert.NoError(t.T(), err)

	// Create local file again.
	f2, err := os.Create(filePath)
	defer require.NoError(t.T(), f2.Close())

	assert.NoError(t.T(), err)
	_, err = os.Stat(filePath)
	assert.NoError(t.T(), err)
}

func (t *HNSBucketTests) TestRenameFileWithSrcFileDoesNotExist() {
	oldFilePath := path.Join(mntDir, "file")
	newFilePath := path.Join(mntDir, "file_rename")

	err := os.Rename(oldFilePath, newFilePath)

	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newFilePath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
}

func (t *HNSBucketTests) TestRenameFileWithDstDestFileExist() {
	oldFilePath := path.Join(mntDir, "foo", "file1.txt")
	_, err := os.Stat(oldFilePath)
	assert.NoError(t.T(), err)
	newFilePath := path.Join(mntDir, "foo", "file2.txt")
	_, err = os.Stat(newFilePath)
	assert.NoError(t.T(), err)

	err = os.Rename(oldFilePath, newFilePath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldFilePath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
}

func (t *HNSBucketTests) TestRenameFileWithDifferentParent() {
	oldFilePath := path.Join(mntDir, "foo", "file1.txt")
	_, err := os.Stat(oldFilePath)
	assert.NoError(t.T(), err)
	newFilePath := path.Join(mntDir, "bar", "file3.txt")
	_, err = os.Stat(newFilePath)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))

	err = os.Rename(oldFilePath, newFilePath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldFilePath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newFilePath)
	assert.NoError(t.T(), err)
}

func (t *HNSBucketTests) TestRenameFileWithSameParent() {
	oldFilePath := path.Join(mntDir, "foo", "file1.txt")
	_, err := os.Stat(oldFilePath)
	assert.NoError(t.T(), err)
	newFilePath := path.Join(mntDir, "foo", "file3.txt")
	_, err = os.Stat(newFilePath)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))

	err = os.Rename(oldFilePath, newFilePath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldFilePath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	f, err := os.Stat(newFilePath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), "file3.txt", f.Name())
}
