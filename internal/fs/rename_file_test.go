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

package fs_test

import (
	"os"
	"path"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RenameFileTests struct {
	suite.Suite
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *RenameFileTests) TestRenameFileWithSrcFileDoesNotExist() {
	oldFilePath := path.Join(mntDir, "file")
	newFilePath := path.Join(mntDir, "file_rename")

	err := os.Rename(oldFilePath, newFilePath)

	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newFilePath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
}

func (t *RenameFileTests) TestRenameFileWithDstDestFileExist() {
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

func (t *RenameFileTests) TestRenameFile() {
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

func (t *RenameFileTests) TestRenameSymlinkToFile() {
	// Create a target file for the symlink to point to.
	targetPath := path.Join(mntDir, "target")
	err := os.WriteFile(targetPath, []byte("taco"), filePerms)
	require.NoError(t.T(), err)
	// Create the symbolic link that we will rename.
	oldPath := path.Join(mntDir, "symlink_old")
	err = os.Symlink(targetPath, oldPath)
	require.NoError(t.T(), err)
	newPath := path.Join(mntDir, "symlink_new")

	err = os.Rename(oldPath, newPath)

	require.NoError(t.T(), err)
	// The old path should no longer exist.
	_, err = os.Lstat(oldPath)
	require.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err), "err: %v", err)
	// The new path should now be a symlink, having replaced the original file.
	fi, err := os.Lstat(newPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), os.ModeSymlink, fi.Mode()&os.ModeSymlink)
	// The new symlink should point to the correct target.
	targetRead, err := os.Readlink(newPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), targetPath, targetRead)
}
