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

package fs_test

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type FlatBucketTests struct {
	fsTest
	RenameFileTests
}

var expectedFooDirEntriesFlatBucket = []dirEntry{
	{name: "test", isDir: true},
	{name: "file1.txt", isDir: false},
	{name: "file2.txt", isDir: false},
	{name: "implicit_dir", isDir: true},
}

func TestFlatBucketTests(t *testing.T) { suite.Run(t, new(FlatBucketTests)) }

func (t *FlatBucketTests) SetupSuite() {
	t.serverCfg.RenameDirLimit = 20
	t.serverCfg.ImplicitDirectories = true
	t.fsTest.SetUpTestSuite()
}

func (t *FlatBucketTests) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *FlatBucketTests) SetupTest() {
	err := t.createObjects(
		map[string]string{
			"foo/file1.txt":              file1Content,
			"foo/file2.txt":              file2Content,
			"foo/test/file3.txt":         "xyz",
			"foo/implicit_dir/file3.txt": "xxw",
			"bar/file1.txt":              "-1234556789",
		})
	require.NoError(t.T(), err)
}

func (t *FlatBucketTests) TearDownTest() {
	t.fsTest.TearDown()
}

func (t *FlatBucketTests) TestRenameFolderWithSourceDirectoryHaveLocalFiles() {
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

func (t *FlatBucketTests) TestRenameFolderWithSameParent() {
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
	assert.Equal(t.T(), 4, len(dirEntries))
	actualDirEntries := []dirEntry{}
	for _, d := range dirEntries {
		actualDirEntries = append(actualDirEntries, dirEntry{
			name:  d.Name(),
			isDir: d.IsDir(),
		})
	}
	assert.ElementsMatch(t.T(), actualDirEntries, expectedFooDirEntriesFlatBucket)
}

func (t *FlatBucketTests) TestRenameFolderWithDifferentParents() {
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
	assert.Equal(t.T(), 4, len(dirEntries))
	actualDirEntries := []dirEntry{}
	for _, d := range dirEntries {
		actualDirEntries = append(actualDirEntries, dirEntry{
			name:  d.Name(),
			isDir: d.IsDir(),
		})
	}
	assert.ElementsMatch(t.T(), actualDirEntries, expectedFooDirEntriesFlatBucket)
}

func (t *FlatBucketTests) TestRenameFolderWithOpenGCSFile() {
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
func (t *FlatBucketTests) TestCreateDirectoryWithSameNameAfterRename() {
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
	//require.Equal(t.T(), 5, len(dirEntries))
	actualDirEntries := []dirEntry{}
	for _, d := range dirEntries {
		actualDirEntries = append(actualDirEntries, dirEntry{
			name:  d.Name(),
			isDir: d.IsDir(),
		})
	}
	require.ElementsMatch(t.T(), actualDirEntries, expectedFooDirEntriesFlatBucket)

	// Create old directory again.
	err = os.Mkdir(oldDirPath, dirPerms)

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	dirEntries, err = os.ReadDir(oldDirPath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 0, len(dirEntries))
}
