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
	"fmt"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
)

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
		fmt.Println("zonal bucket is enabled")
		dirName := path.Join(DirForImplicitDirTests, testDirName)
		fmt.Printf("parent dirName = %q\n", dirName)
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructureUsingStorageClient(testEnv.ctx, t, testEnv.storageClient, dirName)
		fmt.Printf("Successfully created %q\n", dirName)
	} else {
		fmt.Println("zonal bucket is disabled")
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructure(path.Join(DirForImplicitDirTests, testDirName))
		fmt.Println("Successfully created")
	}
	explicitDirPath := path.Join(testDirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, implicit_and_explicit_dir_setup.ImplicitSubDirectory, ExplicitDirInImplicitSubDir)
	fmt.Printf("explicitDirPath = %q\n", explicitDirPath)

	operations.CreateDirectoryWithNFiles(NumberOfFilesInExplicitDirInImplicitSubDir, explicitDirPath, PrefixFileInExplicitDirInImplicitSubDir, t)

	dirPath := path.Join(testDirPath, implicit_and_explicit_dir_setup.ImplicitDirectory)

	implicit_and_explicit_dir_setup.RemoveAndCheckIfDirIsDeleted(dirPath, implicit_and_explicit_dir_setup.ImplicitDirectory, t)
}
