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
	"os"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HNSBucketTests struct {
	suite.Suite
	fsTest
}

func TestHNSBucketTests(t *testing.T) { suite.Run(t, new(HNSBucketTests)) }

func (t *HNSBucketTests) SetupSuite() {
	t.serverCfg.NewConfig = &cfg.Config{EnableHns: true}
	t.serverCfg.ImplicitDirectories = false
	bucketType = gcs.Hierarchical
	t.fsTest.SetUpTestSuite()
}

func (t *HNSBucketTests) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *HNSBucketTests) SetupTest() {
	err = t.createFolders([]string{"foo/", "bar/", "foo/test2/", "foo/test/"})
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
	for i := 0; i < 5; i++ {
		switch dirEntries[i].Name() {
		case "test":
			assert.Equal(t.T(), "test", dirEntries[i].Name())
			assert.True(t.T(), dirEntries[i].IsDir())
		case "test2":
			assert.Equal(t.T(), "test2", dirEntries[i].Name())
			assert.True(t.T(), dirEntries[i].IsDir())
		case "file1.txt":
			assert.Equal(t.T(), "file1.txt", dirEntries[i].Name())
			assert.False(t.T(), dirEntries[i].IsDir())
		case "file2.txt":
			assert.Equal(t.T(), "file2.txt", dirEntries[i].Name())
			assert.False(t.T(), dirEntries[i].IsDir())
		case "implicit_dir":
			assert.Equal(t.T(), "implicit_dir", dirEntries[i].Name())
			assert.True(t.T(), dirEntries[i].IsDir())
		}
	}
}

func (t *HNSBucketTests) TestDeleteFolder() {
	dirPath := path.Join(mntDir, "foo")

	err = os.RemoveAll(dirPath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(dirPath)
	assert.Error(t.T(), err)
}

func (t *HNSBucketTests) TestRenameFolderWithSrcDirectoryDoesNotExist() {
	oldDirPath := path.Join(mntDir, "foo_not_exist")
	newDirPath := path.Join(mntDir, "foo_rename")

	err = os.Rename(oldDirPath, newDirPath)

	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newDirPath)
	assert.Error(t.T(), err)
}

func (t *HNSBucketTests) TestRenameFolderWithDstDirectoryNotEmpty() {
	oldDirPath := path.Join(mntDir, "foo")
	_, err = os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "bar")
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)

	err = os.Rename(oldDirPath, newDirPath)

	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "file exists"))
}

func (t *HNSBucketTests) TestRenameFolderWithSourceDirectoryHaveOpenFiles() {
	oldDirPath := path.Join(mntDir, "foo", "test")
	_, err = os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	file, err := os.OpenFile(path.Join(oldDirPath, "file3.txt"), os.O_RDWR, 0600)
	assert.NoError(t.T(), err)
	defer file.Close()
	newDirPath := path.Join(mntDir, "bar", "foo_rename")

	err = os.Rename(oldDirPath, newDirPath)

	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "file exists"))
}

type DirEntry struct {
	name  string
	isDir bool
}

func (t *HNSBucketTests) TestRenameFolderWithSameParent() {
	expectedDirEntries := []DirEntry{
		{name: "test", isDir: true},
		{name: "test2", isDir: true},
		{name: "file1.txt", isDir: false},
		{name: "file2.txt", isDir: false},
		{name: "implicit_dir", isDir: true},
	}
	oldDirPath := path.Join(mntDir, "foo")
	_, err = os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "foo_rename")

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
	actualDirEntries := []DirEntry{}
	for _, dirEntry := range dirEntries {
		actualDirEntries = append(actualDirEntries, DirEntry{
			name:  dirEntry.Name(),
			isDir: dirEntry.IsDir(),
		})
	}
	assert.ElementsMatch(t.T(), actualDirEntries, expectedDirEntries)
}

func (t *HNSBucketTests) TestRenameFolderWithDifferentParents() {
	expectedDirEntries := []DirEntry{
		{name: "test", isDir: true},
		{name: "test2", isDir: true},
		{name: "file1.txt", isDir: false},
		{name: "file2.txt", isDir: false},
		{name: "implicit_dir", isDir: true},
	}
	oldDirPath := path.Join(mntDir, "foo")
	_, err = os.Stat(oldDirPath)
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
	actualDirEntries := []DirEntry{}
	for _, dirEntry := range dirEntries {
		actualDirEntries = append(actualDirEntries, DirEntry{
			name:  dirEntry.Name(),
			isDir: dirEntry.IsDir(),
		})
	}
	assert.ElementsMatch(t.T(), actualDirEntries, expectedDirEntries)
}
