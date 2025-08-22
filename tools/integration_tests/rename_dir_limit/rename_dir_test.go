// Copyright 2023 Google LLC
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

// Provides integration tests when --rename-dir-limit flag is set.
package rename_dir_limit_test

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

// As --rename-directory-limit = 3, and the number of objects in the directory is three,
// which is equal to the limit, the operation should get successful.
func TestRenameDirectoryWithThreeFiles(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForRenameDirLimitTests)
	// Create directory structure
	// testBucket/dirForRenameDirLimitTests/directoryWithThreeFiles               -- Dir
	// testBucket/dirForRenameDirLimitTests/directoryWithThreeFiles/temp1.txt     -- File
	// testBucket/dirForRenameDirLimitTests/directoryWithThreeFiles/temp2.txt     -- File
	// testBucket/dirForRenameDirLimitTests/directoryWithThreeFiles/temp3.txt     -- File
	dirPath := path.Join(testDir, DirectoryWithThreeFiles)
	operations.CreateDirectoryWithNFiles(3, dirPath, PrefixTempFile, t)

	oldDirPath := path.Join(testDir, DirectoryWithThreeFiles)
	newDirPath := path.Join(testDir, RenamedDirectory)

	//  Cleaning the directory before renaming.
	operations.RemoveDir(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err != nil {
		t.Errorf("Error in renaming directory: %v", err)
	}
}

// As --rename-directory-limit = 3, and the number of objects in the directory is two,
// which is less than the limit, the operation should get successful.
func TestRenameDirectoryWithTwoFiles(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForRenameDirLimitTests)
	// Create directory structure
	// testBucket/dirForRenameDirLimitTests/directoryWithTwoFiles              -- Dir
	// testBucket/dirForRenameDirLimitTests/directoryWithTwoFiles/temp1.txt    -- File
	// testBucket/dirForRenameDirLimitTests/directoryWithTwoFiles/temp2.txt    -- File
	dirPath := path.Join(testDir, DirectoryWithTwoFiles)

	operations.CreateDirectoryWithNFiles(2, dirPath, PrefixTempFile, t)

	oldDirPath := path.Join(testDir, DirectoryWithTwoFiles)
	newDirPath := path.Join(testDir, RenamedDirectory)

	//  Cleaning the directory before renaming.
	operations.RemoveDir(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err != nil {
		t.Errorf("Error in renaming directory: %v", err)
	}
}

// As --rename-directory-limit = 3, and the number of objects in the directory is two,
// which is greater than the limit, the operation should get fail.
func TestRenameDirectoryWithFourFiles(t *testing.T) {
	if setup.ResolveIsHierarchicalBucket(ctx, setup.TestBucket(), storageClient) {
		t.SkipNow()
	}
	testDir := setup.SetupTestDirectory(DirForRenameDirLimitTests)
	// Creating directory structure
	// testBucket/dirForRenameDirLimitTests/directoryWithFourFiles              -- Dir
	// testBucket/dirForRenameDirLimitTests/directoryWithFourFiles/temp1.txt    -- File
	// testBucket/dirForRenameDirLimitTests/directoryWithFourFiles/temp2.txt    -- File
	// testBucket/dirForRenameDirLimitTests/directoryWithFourFiles/temp3.txt    -- File
	// testBucket/dirForRenameDirLimitTests/directoryWithFourFiles/temp4.txt    -- File
	oldDirPath := path.Join(testDir, DirectoryWithFourFiles)
	operations.CreateDirectoryWithNFiles(4, oldDirPath, PrefixTempFile, t)
	newDirPath := path.Join(testDir, RenamedDirectory)

	//  Cleaning the directory before renaming.
	operations.RemoveDir(newDirPath)
	err := os.Rename(oldDirPath, newDirPath)

	if err == nil {
		t.Errorf("Renaming directory succeeded with objects greater than rename-dir-limit.")
	}
}

// As --rename-directory-limit = 3, and the number of objects in the directory is three,
// which is equal to limit, the operation should get successful.
func TestRenameDirectoryWithTwoFilesAndOneEmptyDirectory(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForRenameDirLimitTests)
	// Creating directory structure
	// testBucket/dirForRenameDirLimitTests/directoryWithTwoFilesOneEmptyDirectory                       -- Dir
	// testBucket/dirForRenameDirLimitTests/directoryWithTwoFilesOneEmptyDirectory/a.txt                 -- File
	// testBucket/dirForRenameDirLimitTests/directoryWithTwoFilesOneEmptyDirectory/b.txt                 -- File
	// testBucket/dirForRenameDirLimitTests/directoryWithTwoFilesOneEmptyDirectory/emptySubDirectory     -- Dir
	dirPath := path.Join(testDir, DirectoryWithTwoFilesOneEmptyDirectory)
	subDirPath := path.Join(testDir, DirectoryWithTwoFilesOneEmptyDirectory, EmptySubDirectory)

	operations.CreateDirectoryWithNFiles(2, dirPath, PrefixTempFile, t)
	operations.CreateDirectoryWithNFiles(0, subDirPath, PrefixTempFile, t)

	oldDirPath := path.Join(testDir, DirectoryWithTwoFilesOneEmptyDirectory)
	newDirPath := path.Join(testDir, RenamedDirectory)

	//  Cleaning the directory before renaming.
	operations.RemoveDir(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err != nil {
		t.Errorf("Error in renaming directory: %v", err)
	}
}

// As --rename-directory-limit = 3, and the number of objects in the directory is Four,
// which is greater than the limit, the operation should get fail.
func TestRenameDirectoryWithTwoFilesAndOneNonEmptyDirectory(t *testing.T) {
	if setup.ResolveIsHierarchicalBucket(ctx, setup.TestBucket(), storageClient) {
		t.SkipNow()
	}
	testDir := setup.SetupTestDirectory(DirForRenameDirLimitTests)
	// Creating directory structure
	// testBucket/dirForRenameDirLimitTests/directoryWithTwoFilesOneNonEmptyDirectory                                      -- Dir
	// testBucket/dirForRenameDirLimitTests/directoryWithTwoFilesOneNonEmptyDirectory/temp1.txt                            -- File
	// testBucket/dirForRenameDirLimitTests/directoryWithTwoFilesOneNonEmptyDirectory/temp2.txt                            -- File
	// testBucket/dirForRenameDirLimitTests/directoryWithTwoFilesOneNonEmptyDirectory/NonEmptySubDirectory                 -- Dir
	// testBucket/dirForRenameDirLimitTests/directoryWithTwoFilesOneNonEmptyDirectory/NonEmptySubDirectory/temp3.txt   		 -- File

	dirPath := path.Join(testDir, DirectoryWithTwoFilesOneNonEmptyDirectory)
	subDirPath := path.Join(dirPath, NonEmptySubDirectory)

	operations.CreateDirectoryWithNFiles(2, dirPath, PrefixTempFile, t)
	operations.CreateDirectoryWithNFiles(1, subDirPath, PrefixTempFile, t)

	oldDirPath := path.Join(testDir, DirectoryWithTwoFilesOneNonEmptyDirectory)
	newDirPath := path.Join(testDir, RenamedDirectory)

	//  Cleaning the directory before renaming.
	operations.RemoveDir(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err == nil {
		t.Errorf("Renaming directory succeeded with objects greater than rename-dir-limit.")
	}
}

func TestRenameDirectoryWithExistingEmptyDestDirectory(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForRenameDirLimitTests)
	// Creating directory structure
	// testBucket/dirForRenameDirLimitTests/srcDirectory                                      -- Dir
	// testBucket/dirForRenameDirLimitTests/srcDirectory/temp1.txt                            -- File
	// testBucket/dirForRenameDirLimitTests/srcDirectory/NonEmptySubDirectory                 -- Dir
	// testBucket/dirForRenameDirLimitTests/srcDirectory/NonEmptySubDirectory/temp3.txt   		-- File
	// testBucket/dirForRenameDirLimitTests/emptyDestDirectory   		 													-- Dir
	oldDirPath := path.Join(testDir, SrcDirectory)
	subDirPath := path.Join(oldDirPath, NonEmptySubDirectory)
	operations.CreateDirectoryWithNFiles(1, oldDirPath, PrefixTempFile, t)
	operations.CreateDirectoryWithNFiles(1, subDirPath, PrefixTempFile, t)
	newDirPath := path.Join(testDir, EmptyDestDirectory)
	operations.CreateDirectory(newDirPath, t)

	// Go's Rename function does not support renaming a directory into an existing empty directory.
	// To achieve this, we call a Python rename function as a workaround.
	cmd := exec.Command("python3", "-c", fmt.Sprintf("import os; os.rename('%s', '%s')", oldDirPath, newDirPath))
	_, err := cmd.CombinedOutput()

	assert.NoError(t, err)
	_, err = os.Stat(oldDirPath)
	assert.ErrorContains(t, err, "no such file or directory")
	_, err = os.Stat(newDirPath)
	assert.NoError(t, err)
	dirEntries, err := os.ReadDir(newDirPath)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(dirEntries))
	assert.Equal(t, NonEmptySubDirectory, dirEntries[0].Name())
	assert.True(t, dirEntries[0].IsDir())
	assert.Equal(t, "temp1", dirEntries[1].Name())
	assert.False(t, dirEntries[1].IsDir())
}
