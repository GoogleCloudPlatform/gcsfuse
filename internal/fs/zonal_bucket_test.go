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

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ZonalBucketTests struct {
	suite.Suite
	fsTest
}

func TestZonalBucketTests(t *testing.T) { suite.Run(t, new(ZonalBucketTests)) }

func (t *ZonalBucketTests) SetupSuite() {
	t.serverCfg.ImplicitDirectories = false
	t.serverCfg.MetricHandle = common.NewNoopMetrics()
	bucketType = gcs.BucketType{Zonal: true}
	t.fsTest.SetUpTestSuite()
}

func (t *ZonalBucketTests) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *ZonalBucketTests) SetupTest() {
	err := t.createFolders([]string{"foo/", "bar/", "foo/test2/", "foo/test/"})
	require.NoError(t.T(), err)

	err = t.createObjects(
		map[string]string{
			"foo/file1.txt":              file1Content,
			"foo/file2.txt":              file2Content,
			"foo/test/file3.txt":         "xyz",
			"foo/implicit_dir/file3.txt": "xxw",
			"bar/file1.txt":              "-1234556789",
		})
	require.NoError(t.T(), err)
}

func (t *ZonalBucketTests) TearDownTest() {
	t.fsTest.TearDown()
}

func (t *ZonalBucketTests) TestRenameFileWithSrcFileDoesNotExist() {
	oldFilePath := path.Join(mntDir, "file")
	newFilePath := path.Join(mntDir, "file_rename")

	err := os.Rename(oldFilePath, newFilePath)

	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newFilePath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
}

func (t *ZonalBucketTests) TestRenameFileWithDstDestFileExist() {
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
	content, err := os.ReadFile(newFilePath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), file1Content, string(content))
}

func (t *ZonalBucketTests) TestRenameFile() {
	testCases := []struct {
		name        string
		oldFilePath string
		newFilePath string
		wantContent string
	}{
		{
			name:        "DifferentParent",
			oldFilePath: path.Join(mntDir, "foo", "file1.txt"),
			newFilePath: path.Join(mntDir, "bar", "file3.txt"),
			wantContent: file1Content,
		},
		{
			name:        "SameParent",
			oldFilePath: path.Join(mntDir, "foo", "file2.txt"),
			newFilePath: path.Join(mntDir, "foo", "file3.txt"),
			wantContent: file2Content,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			// Ensure file exists before renaming.
			_, err := os.Stat(tc.oldFilePath)
			require.NoError(t.T(), err)

			// Rename the file.
			err = os.Rename(tc.oldFilePath, tc.newFilePath)
			assert.NoError(t.T(), err)

			// Verify the old file no longer exists.
			_, err = os.Stat(tc.oldFilePath)
			assert.Error(t.T(), err)
			assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
			// Verify the new file exists and has the correct content.
			f, err := os.Stat(tc.newFilePath)
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), path.Base(tc.newFilePath), f.Name())
			content, err := os.ReadFile(tc.newFilePath)
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), tc.wantContent, string(content))
		})
	}
}
