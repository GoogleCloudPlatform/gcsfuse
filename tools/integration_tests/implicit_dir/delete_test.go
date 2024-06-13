// Copyright 2023 Google Inc. All Rights Reserved.
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

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
)

// Directory Structure
// testBucket/dirForImplicitDirTests/implicitDirectory                                                  -- Dir
// testBucket/dirForImplicitDirTests/implicitDirectory/fileInImplicitDir1                               -- File
// testBucket/dirForImplicitDirTests/implicitDirectory/implicitSubDirectory                             -- Dir
// testBucket/dirForImplicitDirTests/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File
func TestDeleteNonEmptyImplicitDir(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForImplicitDirTests)
	implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructure(DirForImplicitDirTests, t)

	dirPath := path.Join(testDir, implicit_and_explicit_dir_setup.ImplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, t)
}

// Directory Structure
// testBucket/dirForImplicitDirTests/implicitDirectory                                                  -- Dir
// testBucket/dirForImplicitDirTests/implicitDirectory/fileInImplicitDir1                               -- File
// testBucket/dirForImplicitDirTests/implicitDirectory/implicitSubDirectory                             -- Dir
// testBucket/dirForImplicitDirTests/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File
func TestDeleteNonEmptyImplicitSubDir(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForImplicitDirTests)
	implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructure(DirForImplicitDirTests, t)

	subDirPath := path.Join(testDir, implicit_and_explicit_dir_setup.ImplicitDirectory, implicit_and_explicit_dir_setup.ImplicitSubDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(subDirPath, implicit_and_explicit_dir_setup.ImplicitSubDirectory, t)
}

// Directory Structure
// testBucket/dirForImplicitDirTests/implicitDirectory                                                                    -- Dir
// testBucket/dirForImplicitDirTests/implicitDirectory/explicitDirInImplicitDir                                           -- Dir
// testBucket/dirForImplicitDirTests/implicitDirectory/explicitDirInImplicitDir/fileInExplicitDirInImplicitDir            -- File
// testBucket/dirForImplicitDirTests/implicitDirectory/fileInImplicitDir1                                                 -- File
// testBucket/dirForImplicitDirTests/implicitDirectory/implicitSubDirectory                                               -- Dir
// testBucket/dirForImplicitDirTests/implicitDirectory/implicitSubDirectory/fileInImplicitDir2                            -- File
func TestDeleteImplicitDirWithExplicitSubDir(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForImplicitDirTests)
	implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructure(DirForImplicitDirTests, t)

	explicitDirPath := path.Join(testDir, implicit_and_explicit_dir_setup.ImplicitDirectory, ExplicitDirInImplicitDir)

	operations.CreateDirectoryWithNFiles(NumberOfFilesInExplicitDirInImplicitDir, explicitDirPath, PrefixFileInExplicitDirInImplicitDir, t)

	dirPath := path.Join(testDir, implicit_and_explicit_dir_setup.ImplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, t)
}

// Directory Structure
// testBucket/dirForImplicitDirTests/implicitDirectory                                                                                         -- Dir
// testBucket/dirForImplicitDirTests/implicitDirectory/fileInImplicitDir1                                                                      -- File
// testBucket/dirForImplicitDirTests/implicitDirectory/implicitSubDirectory                                                                    -- Dir
// testBucket/dirForImplicitDirTests/implicitDirectory/implicitSubDirectory/fileInImplicitDir2                                                 -- File
// testBucket/dirForImplicitDirTests/implicitDirectory/implicitSubDirectory/explicitDirInImplicitDir                                           -- Dir
// testBucket/dirForImplicitDirTests/implicitDirectory/implicitSubDirectory/explicitDirInImplicitDir/fileInExplicitDirInImplicitDir            -- File
func TestDeleteImplicitDirWithImplicitSubDirContainingExplicitDir(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForImplicitDirTests)
	implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructure(DirForImplicitDirTests, t)
	explicitDirPath := path.Join(testDir, implicit_and_explicit_dir_setup.ImplicitDirectory, implicit_and_explicit_dir_setup.ImplicitSubDirectory, ExplicitDirInImplicitSubDir)

	operations.CreateDirectoryWithNFiles(NumberOfFilesInExplicitDirInImplicitSubDir, explicitDirPath, PrefixFileInExplicitDirInImplicitSubDir, t)

	dirPath := path.Join(testDir, implicit_and_explicit_dir_setup.ImplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, t)
}

// Directory Structure
// testBucket/dirForImplicitDirTests/explicitDirectory                                                                   -- Dir
// testBucket/dirForImplicitDirTests/explictFile                                                                         -- File
// testBucket/dirForImplicitDirTests/explicitDirectory/fileInExplicitDir1                                                -- File
// testBucket/dirForImplicitDirTests/explicitDirectory/fileInExplicitDir2                                                -- File
// testBucket/dirForImplicitDirTests/explicitDirectory/implicitDirectory                                                 -- Dir
// testBucket/dirForImplicitDirTests/explicitDirectory/implicitDirectory/fileInImplicitDir1                              -- File
// testBucket/dirForImplicitDirTests/explicitDirectory/implicitDirectory/implicitSubDirectory                            -- Dir
// testBucket/dirForImplicitDirTests/explicitDirectory/implicitDirectory/implicitSubDirectory/fileInImplicitDir2         -- File
func TestDeleteImplicitDirInExplicitDir(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForImplicitDirTests)
	implicit_and_explicit_dir_setup.CreateImplicitDirectoryInExplicitDirectoryStructure(DirForImplicitDirTests, t)

	dirPath := path.Join(testDir, implicit_and_explicit_dir_setup.ExplicitDirectory, implicit_and_explicit_dir_setup.ImplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, t)
}

// Directory Structure
// testBucket/dirForImplicitDirTests/explicitDirectory                                                                   -- Dir
// testBucket/dirForImplicitDirTests/explictFile                                                                         -- File
// testBucket/dirForImplicitDirTests/explicitDirectory/fileInExplicitDir1                                                -- File
// testBucket/dirForImplicitDirTests/explicitDirectory/fileInExplicitDir2                                                -- File
// testBucket/dirForImplicitDirTests/explicitDirectory/implicitDirectory                                                 -- Dir
// testBucket/dirForImplicitDirTests/explicitDirectory/implicitDirectory/fileInImplicitDir1                              -- File
// testBucket/dirForImplicitDirTests/explicitDirectory/implicitDirectory/implicitSubDirectory                            -- Dir
// testBucket/dirForImplicitDirTests/explicitDirectory/implicitDirectory/implicitSubDirectory/fileInImplicitDir2         -- File
func TestDeleteExplicitDirContainingImplicitSubDir(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForImplicitDirTests)
	implicit_and_explicit_dir_setup.CreateImplicitDirectoryInExplicitDirectoryStructure(DirForImplicitDirTests, t)

	dirPath := path.Join(testDir, implicit_and_explicit_dir_setup.ExplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ExplicitDirectory, t)
}
