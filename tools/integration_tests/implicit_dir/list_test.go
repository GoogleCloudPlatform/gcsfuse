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

package implicit_dir_test

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListImplicitObjectsFromBucket(t *testing.T) {
	testDirName := "testListImplicitObjectsFromBucket" + setup.GenerateRandomString(4)
	testDirPath := setupTestDir(testDirName)
	// Directory Structure
	// testBucket/dirForImplicitDirTests/testDir/implicitDirectory                                                  -- Dir
	// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/fileInImplicitDir1                               -- File
	// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/implicitSubDirectory                             -- Dir
	// testBucket/dirForImplicitDirTests/testDir/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File
	// testBucket/dirForImplicitDirTests/testDir/explicitDirectory                                                  -- Dir
	// testBucket/dirForImplicitDirTests/testDir/explicitFile                                                       -- File
	// testBucket/dirForImplicitDirTests/testDir/explicitDirectory/fileInExplicitDir1                               -- File
	// testBucket/dirForImplicitDirTests/testDir/explicitDirectory/fileInExplicitDir2                               -- File

	// TODO: Remove the condition and keep the storage-client flow for non-ZB too.
	if setup.IsZonalBucketRun() {
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructureUsingStorageClient(testEnv.ctx, t, testEnv.storageClient, path.Join(DirForImplicitDirTests, testDirName))
	} else {
		implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructure(path.Join(DirForImplicitDirTests, testDirName))
	}
	implicit_and_explicit_dir_setup.CreateExplicitDirectoryStructure(path.Join(DirForImplicitDirTests, testDirName), t)

	err := filepath.WalkDir(testDirPath, func(path string, dir fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		// The object type is not directory.
		if !dir.IsDir() {
			return nil
		}

		objs, err := os.ReadDir(path)
		if err != nil {
			log.Fatal(err)
		}

		// Check if mntDir has correct objects.
		if path == testDirPath {
			// numberOfObjects - 3
			require.Equalf(t, implicit_and_explicit_dir_setup.NumberOfTotalObjects, len(objs), "Got Objects: %v", objs)
			// testBucket/dirForImplicitDirTests/testDir/explicitDir     -- Dir
			assert.Equal(t, implicit_and_explicit_dir_setup.ExplicitDirectory, objs[0].Name())
			assert.True(t, objs[0].IsDir())
			// testBucket/dirForImplicitDirTests/testDir/explicitFile    -- File
			assert.Equal(t, implicit_and_explicit_dir_setup.ExplicitFile, objs[1].Name())
			assert.False(t, objs[1].IsDir())
			// testBucket/dirForImplicitDirTests/testDir/implicitDir     -- Dir
			assert.Equal(t, implicit_and_explicit_dir_setup.ImplicitDirectory, objs[2].Name())
			assert.True(t, objs[2].IsDir())
		}

		// Check if explictDir directory has correct data.
		if dir.IsDir() && dir.Name() == implicit_and_explicit_dir_setup.ExplicitDirectory {
			// numberOfObjects - 2
			require.Equalf(t, implicit_and_explicit_dir_setup.NumberOfFilesInExplicitDirectory, len(objs), "Got Objects: %v", objs)
			// testBucket/dirForImplicitDirTests/testDir/explicitDir/fileInExplicitDir1   -- File
			assert.Equal(t, implicit_and_explicit_dir_setup.FirstFileInExplicitDirectory, objs[0].Name())
			assert.False(t, objs[0].IsDir())
			// testBucket/dirForImplicitDirTests/testDir/explicitDir/fileInExplicitDir2    -- File
			assert.Equal(t, implicit_and_explicit_dir_setup.SecondFileInExplicitDirectory, objs[1].Name())
			assert.False(t, objs[1].IsDir())
			return nil
		}

		// Check if implicitDir directory has correct data.
		if dir.IsDir() && dir.Name() == implicit_and_explicit_dir_setup.ImplicitDirectory {
			// numberOfObjects - 2
			require.Equalf(t, implicit_and_explicit_dir_setup.NumberOfFilesInImplicitDirectory, len(objs), "Got Objects: %v", objs)
			// testBucket/dirForImplicitDirTests/testDir/implicitDir/fileInImplicitDir1  -- File
			assert.Equal(t, implicit_and_explicit_dir_setup.FileInImplicitDirectory, objs[0].Name())
			assert.False(t, objs[0].IsDir())
			// testBucket/dirForImplicitDirTests/testDir/implicitDir/implicitSubDirectory  -- Dir
			assert.Equal(t, implicit_and_explicit_dir_setup.ImplicitSubDirectory, objs[1].Name())
			assert.True(t, objs[1].IsDir())
			return nil
		}

		// Check if implicit subdirectory has correct data.
		if dir.IsDir() && dir.Name() == implicit_and_explicit_dir_setup.ImplicitSubDirectory {
			// numberOfObjects - 1
			require.Equalf(t, implicit_and_explicit_dir_setup.NumberOfFilesInImplicitSubDirectory, len(objs), "Got Objects: %v", objs)
			// testBucket/dirForImplicitDirTests/testDir/implicitDir/implicitSubDir/fileInImplicitDir2   -- File
			assert.Equal(t, implicit_and_explicit_dir_setup.FileInImplicitSubDirectory, objs[0].Name())
			assert.False(t, objs[0].IsDir())
			return nil
		}
		return nil
	})
	require.NoError(t, err, "error walking the path")
}
