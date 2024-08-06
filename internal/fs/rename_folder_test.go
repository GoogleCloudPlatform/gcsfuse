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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RenameFolderTests struct {
	suite.Suite
	fsTest
}

func TestRenameFolderTests(t *testing.T) { suite.Run(t, new(RenameFolderTests)) }

func (t *RenameFolderTests) SetupSuite() {
	t.serverCfg.MountConfig = &config.MountConfig{EnableHNS: true}
	bucketType = gcs.Hierarchical
	t.fsTest.SetUpTestSuite()
}

func (t *RenameFolderTests) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *RenameFolderTests) SetupTest() {
	err = t.createFolders([]string{"foo/", "bar/", "foo/test/", "foo/test2/", "foo/test/subDir/", "rename/"})
	require.NoError(t.T(), err)
	err = t.createObjects(
		map[string]string{
			"foo/file1.txt":             "abcdef",
			"foo/file2.txt":             "xyz",
			"foo/test/file3.txt":        "xyz",
			"foo/test/file4.txt":        "xyz",
			"foo/test/subDir/file4.txt": "xyz",
			"bar/file1.txt":             "-1234556789",
		})
	require.NoError(t.T(), err)
}

func (t *RenameFolderTests) TearDown() {
	t.fsTest.TearDown()
}

func (t *RenameFolderTests) TestRenameFolderWithSrcDoesNotExist() {
	oldDirPath := path.Join(mntDir, "foo_not_exist")
	newDirPath := path.Join(mntDir, "foo_rename_2")

	err = os.Rename(oldDirPath, newDirPath)

	assert.NotNil(t.T(), err)
	_, err = os.Stat(newDirPath)
	assert.NotNil(t.T(), err)
}

func (t *RenameFolderTests) TestRenameFolderWithDstDirectoryIsNotEmpty() {
	oldDirPath := path.Join(mntDir, "foo")
	_, err = os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "bar")
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)

	err = os.Rename(oldDirPath, newDirPath)

	assert.NotNil(t.T(), err)
}

func (t *RenameFolderTests) TestRenameFolderWithSameParent() {
	oldDirPath := path.Join(mntDir, "foo")
	_, err = os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "foo_rename")

	err = os.Rename(oldDirPath, newDirPath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldDirPath)
	assert.NotNil(t.T(), err)
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)
}

func (t *RenameFolderTests) TestRenameFolderWithDifferentParent() {
	oldDirPath := path.Join(mntDir, "foo", "test")
	_, err = os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "rename", "test_rename")

	err = os.Rename(oldDirPath, newDirPath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldDirPath)
	assert.NotNil(t.T(), err)
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)
}