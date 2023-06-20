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
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
)

// Directory Structure
// testBucket/implicitDirectory                                                  -- Dir
// testBucket/implicitDirectory/fileInImplicitDir1                               -- File
// testBucket/implicitDirectory/implicitSubDirectory                             -- Dir
// testBucket/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File
func TestDeleteNonEmptyImplicitDir(t *testing.T) {
	implicit_and_explicit_dir_setup.CreateImplicitDirectory()

	dirPath := path.Join(setup.MntDir(), implicit_and_explicit_dir_setup.ImplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, t)

	// Cleaning the bucket.
	os.RemoveAll(setup.MntDir())
}

// Directory Structure
// testBucket/implicitDirectory                                                  -- Dir
// testBucket/implicitDirectory/fileInImplicitDir1                               -- File
// testBucket/implicitDirectory/implicitSubDirectory                             -- Dir
// testBucket/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File
func TestDeleteNonEmptyImplicitSubDir(t *testing.T) {
	implicit_and_explicit_dir_setup.CreateImplicitDirectory()

	subDirPath := path.Join(setup.MntDir(), implicit_and_explicit_dir_setup.ImplicitDirectory, implicit_and_explicit_dir_setup.ImplicitSubDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(subDirPath, implicit_and_explicit_dir_setup.ImplicitSubDirectory, t)

	// Cleaning the bucket.
	os.RemoveAll(setup.MntDir())
}

// Directory Structure
// testBucket/implicitDirectory                                                                    -- Dir
// testBucket/implicitDirectory/explicitDirInImplicitDir                                           -- Dir
// testBucket/implicitDirectory/explicitDirInImplicitDir/fileInExplicitDirInImplicitDir            -- File
// testBucket/implicitDirectory/fileInImplicitDir1                                                 -- File
// testBucket/implicitDirectory/implicitSubDirectory                                               -- Dir
// testBucket/implicitDirectory/implicitSubDirectory/fileInImplicitDir2                            -- File
func TestDeleteImplicitDirWithExplicitSubDir(t *testing.T) {
	implicit_and_explicit_dir_setup.CreateImplicitDirectory()

	explicitDirPath := path.Join(setup.MntDir(), implicit_and_explicit_dir_setup.ImplicitDirectory, ExplicitDirInImplicitDir)

	operations.CreateDirectoryWithNFiles(1, explicitDirPath, FileInExplicitDirInImplicitDir, t)

	dirPath := path.Join(setup.MntDir(), implicit_and_explicit_dir_setup.ImplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, t)

	// Cleaning the bucket.
	os.RemoveAll(setup.MntDir())
}

// Directory Structure
// testBucket/implicitDirectory                                                                                         -- Dir
// testBucket/implicitDirectory/fileInImplicitDir1                                                                      -- File
// testBucket/implicitDirectory/implicitSubDirectory                                                                    -- Dir
// testBucket/implicitDirectory/implicitSubDirectory/fileInImplicitDir2                                                 -- File
// testBucket/implicitDirectory/implicitSubDirectory/explicitDirInImplicitDir                                           -- Dir
// testBucket/implicitDirectory/implicitSubDirectory/explicitDirInImplicitDir/fileInExplicitDirInImplicitDir            -- File
func TestDeleteImplicitDirWithExplicitSubDirInImplicitSubDir(t *testing.T) {
	implicit_and_explicit_dir_setup.CreateImplicitDirectory()

	explicitDirPath := path.Join(setup.MntDir(), implicit_and_explicit_dir_setup.ImplicitDirectory, implicit_and_explicit_dir_setup.ImplicitSubDirectory, ExplicitDirInImplicitSubDir)

	//time.Sleep(1 * time.Minute)

	err := os.Mkdir(explicitDirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in creating directory: %v", err)
	}

	//time.Sleep(1 * time.Minute)

	//filePath := path.Join(explicitDirPath, FileInExplicitDirInImplicitSubDir)
	//_, err = os.Create(filePath)
	//if err != nil {
	//	t.Errorf("Create file at %q: %v", explicitDirPath, err)
	//}

	dirPath := path.Join(setup.MntDir(), implicit_and_explicit_dir_setup.ImplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, t)

	// Cleaning the bucket.
	os.RemoveAll(setup.MntDir())
}

// Directory Structure
// testBucket/explicitDirectory                                                                   -- Dir
// testBucket/explictFile                                                                         -- File
// testBucket/explicitDirectory/fileInExplicitDir1                                                -- File
// testBucket/explicitDirectory/fileInExplicitDir2                                                -- File
// testBucket/explicitDirectory/implicitDirectory                                                 -- Dir
// testBucket/explicitDirectory/implicitDirectory/fileInImplicitDir1                              -- File
// testBucket/explicitDirectory/implicitDirectory/implicitSubDirectory                            -- Dir
// testBucket/explicitDirectory/implicitDirectory/implicitSubDirectory/fileInImplicitDir2         -- File
func TestDeleteImplicitDirInExplicitDir(t *testing.T) {
	implicit_and_explicit_dir_setup.CreateImplicitDirectoryInExplicitDirectory(t)

	dirPath := path.Join(setup.MntDir(), implicit_and_explicit_dir_setup.ExplicitDirectory, implicit_and_explicit_dir_setup.ImplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, t)

	// Cleaning the bucket.
	os.RemoveAll(setup.MntDir())
}

// Directory Structure
// testBucket/explicitDirectory                                                                   -- Dir
// testBucket/explictFile                                                                         -- File
// testBucket/explicitDirectory/fileInExplicitDir1                                                -- File
// testBucket/explicitDirectory/fileInExplicitDir2                                                -- File
// testBucket/explicitDirectory/implicitDirectory                                                 -- Dir
// testBucket/explicitDirectory/implicitDirectory/fileInImplicitDir1                              -- File
// testBucket/explicitDirectory/implicitDirectory/implicitSubDirectory                            -- Dir
// testBucket/explicitDirectory/implicitDirectory/implicitSubDirectory/fileInImplicitDir2         -- File
func TestDeleteExplicitDirWithImplicitSubDir(t *testing.T) {
	implicit_and_explicit_dir_setup.CreateImplicitDirectoryInExplicitDirectory(t)

	dirPath := path.Join(setup.MntDir(), implicit_and_explicit_dir_setup.ExplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ExplicitDirectory, t)

	// Cleaning the bucket.
	os.RemoveAll(setup.MntDir())
}
