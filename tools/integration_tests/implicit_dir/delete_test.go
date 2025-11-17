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

// Provide test for deleting implicit directory.
package implicit_dir_test

import (
	"path"
	"testing"

	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
)

// Directory Structure
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory                                                  -- Dir
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/fileInImplicitDir1                               -- File
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/implicitSubDirectory                             -- Dir
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File
func TestDeleteNonEmptyImplicitDir(t *testing.T) {
	testDirName := "testDeleteNonEmptyImplicitDir"
	testDirPath := setupTestDir(testDirName)
	// TODO: Remove the condition and keep the storage-client flow for non-ZB too.
	if setup.IsZonalBucketRun() {
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructureUsingStorageClient(testEnv.ctx, t, testEnv.storageClient, path.Join(DirForImplicitDirTests, testDirName))
	} else {
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructure(path.Join(DirForImplicitDirTests, testDirName))
	}

	dirPath := path.Join(testDirPath, implicit_and_explicit_dir_setup.ImplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, t)
}

// Directory Structure
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory                                                  -- Dir
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/fileInImplicitDir1                               -- File
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/implicitSubDirectory                             -- Dir
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File
func TestDeleteNonEmptyImplicitSubDir(t *testing.T) {
	testDirName := "testDeleteNonEmptyImplicitSubDir"
	testDirPath := setupTestDir(testDirName)
	// TODO: Remove the condition and keep the storage-client flow for non-ZB too.
	if setup.IsZonalBucketRun() {
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructureUsingStorageClient(testEnv.ctx, t, testEnv.storageClient, path.Join(DirForImplicitDirTests, testDirName))
	} else {
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructure(path.Join(DirForImplicitDirTests, testDirName))
	}

	subDirPath := path.Join(testDirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, implicit_and_explicit_dir_setup.ImplicitSubDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(subDirPath, implicit_and_explicit_dir_setup.ImplicitSubDirectory, t)
}

// Directory Structure
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory                                                                    -- Dir
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/explicitDirInImplicitDir                                           -- Dir
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/explicitDirInImplicitDir/fileInExplicitDirInImplicitDir            -- File
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/fileInImplicitDir1                                                 -- File
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/implicitSubDirectory                                               -- Dir
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/implicitSubDirectory/fileInImplicitDir2                            -- File
func TestDeleteImplicitDirWithExplicitSubDir(t *testing.T) {
	testDirName := "testDeleteImplicitDirWithExplicitSubDir"
	testDirPath := setupTestDir(testDirName)
	// TODO: Remove the condition and keep the storage-client flow for non-ZB too.
	if setup.IsZonalBucketRun() {
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructureUsingStorageClient(testEnv.ctx, t, testEnv.storageClient, path.Join(DirForImplicitDirTests, testDirName))
	} else {
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructure(path.Join(DirForImplicitDirTests, testDirName))
	}

	explicitDirPath := path.Join(testDirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, ExplicitDirInImplicitDir)

	operations.CreateDirectoryWithNFiles(NumberOfFilesInExplicitDirInImplicitDir, explicitDirPath, PrefixFileInExplicitDirInImplicitDir, t)

	dirPath := path.Join(testDirPath, implicit_and_explicit_dir_setup.ImplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, t)
}

// Directory Structure
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory                                                                                         -- Dir
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/fileInImplicitDir1                                                                      -- File
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/implicitSubDirectory                                                                    -- Dir
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/implicitSubDirectory/fileInImplicitDir2                                                 -- File
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/implicitSubDirectory/explicitDirInImplicitDir                                           -- Dir
// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/implicitSubDirectory/explicitDirInImplicitDir/fileInExplicitDirInImplicitDir            -- File
func TestDeleteImplicitDirWithImplicitSubDirContainingExplicitDir(t *testing.T) {
	testDirName := "testDeleteImplicitDirWithImplicitSubDirContainingExplicitDir"
	testDirPath := setupTestDir(testDirName)
	// TODO: Remove the condition and keep the storage-client flow for non-ZB too.
	if setup.IsZonalBucketRun() {
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructureUsingStorageClient(testEnv.ctx, t, testEnv.storageClient, path.Join(DirForImplicitDirTests, testDirName))
	} else {
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructure(path.Join(DirForImplicitDirTests, testDirName))
	}
	explicitDirPath := path.Join(testDirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, implicit_and_explicit_dir_setup.ImplicitSubDirectory, ExplicitDirInImplicitSubDir)

	operations.CreateDirectoryWithNFiles(NumberOfFilesInExplicitDirInImplicitSubDir, explicitDirPath, PrefixFileInExplicitDirInImplicitSubDir, t)

	dirPath := path.Join(testDirPath, implicit_and_explicit_dir_setup.ImplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, t)
}

// Directory Structure
// testBucket/dirForImplicitDirTests/testDir/explicitDirectory                                                                   -- Dir
// testBucket/dirForImplicitDirTests/testDir/explictFile                                                                         -- File
// testBucket/dirForImplicitDirTests/testDir/explicitDirectory/fileInExplicitDir1                                                -- File
// testBucket/dirForImplicitDirTests/testDir/explicitDirectory/fileInExplicitDir2                                                -- File
// testBucket/dirForImplicitDirTests/testDir/explicitDirectory/implicitDirectory                                                 -- Dir
// testBucket/dirForImplicitDirTests/testDir/explicitDirectory/implicitDirectory/fileInImplicitDir1                              -- File
// testBucket/dirForImplicitDirTests/testDir/explicitDirectory/implicitDirectory/implicitSubDirectory                            -- Dir
// testBucket/dirForImplicitDirTests/testDir/explicitDirectory/implicitDirectory/implicitSubDirectory/fileInImplicitDir2         -- File
func TestDeleteImplicitDirInExplicitDir(t *testing.T) {
	testDirName := "testDeleteImplicitDirInExplicitDir"
	testDirPath := setupTestDir(testDirName)
	// TODO: Remove the condition and keep the storage-client flow for non-ZB too.
	if setup.IsZonalBucketRun() {
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryInExplicitDirectoryStructureUsingStorageClient(testEnv.ctx, t, testEnv.storageClient, path.Join(DirForImplicitDirTests, testDirName))
	} else {
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryInExplicitDirectoryStructure(path.Join(DirForImplicitDirTests, testDirName), t)
	}

	dirPath := path.Join(testDirPath, implicit_and_explicit_dir_setup.ExplicitDirectory, implicit_and_explicit_dir_setup.ImplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, t)
}

// Directory Structure
// testBucket/dirForImplicitDirTests/testDir/explicitDirectory                                                                   -- Dir
// testBucket/dirForImplicitDirTests/testDir/explictFile                                                                         -- File
// testBucket/dirForImplicitDirTests/testDir/explicitDirectory/fileInExplicitDir1                                                -- File
// testBucket/dirForImplicitDirTests/testDir/explicitDirectory/fileInExplicitDir2                                                -- File
// testBucket/dirForImplicitDirTests/testDir/explicitDirectory/implicitDirectory                                                 -- Dir
// testBucket/dirForImplicitDirTests/testDir/explicitDirectory/implicitDirectory/fileInImplicitDir1                              -- File
// testBucket/dirForImplicitDirTests/testDir/explicitDirectory/implicitDirectory/implicitSubDirectory                            -- Dir
// testBucket/dirForImplicitDirTests/testDir/explicitDirectory/implicitDirectory/implicitSubDirectory/fileInImplicitDir2         -- File
func TestDeleteExplicitDirContainingImplicitSubDir(t *testing.T) {
	testDirName := "testDeleteExplicitDirContainingImplicitSubDir"
	testDirPath := setupTestDir(testDirName)
	// TODO: Remove the condition and keep the storage-client flow for non-ZB too.
	if setup.IsZonalBucketRun() {
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryInExplicitDirectoryStructureUsingStorageClient(testEnv.ctx, t, testEnv.storageClient, path.Join(DirForImplicitDirTests, testDirName))
	} else {
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryInExplicitDirectoryStructure(path.Join(DirForImplicitDirTests, testDirName), t)
	}

	dirPath := path.Join(testDirPath, implicit_and_explicit_dir_setup.ExplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ExplicitDirectory, t)
}
