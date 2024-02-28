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

package empty_managed_folders_list

import (
	"fmt"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func deleteManagedFoldersInTestDir(managedFolder, bucket, testDir string) {
	gcloudDeleteManagedFolderCmd := fmt.Sprintf("alpha storage rm -r gs://%s/%s/%s", bucket, testDir, managedFolder)
	_, err := operations.ExecuteGcloudCommandf(gcloudDeleteManagedFolderCmd)
	if err != nil && !strings.Contains(err.Error(), "The following URLs matched no objects or files") {
		setup.LogAndExit(fmt.Sprintf("Error while deleting managed folder: %v", err))
	}
}

func createManagedFoldersInTestDir(managedFolder string) {
	bucket := setup.TestBucket()
	testDir := testDirName
	client.SetBucketAndObjectBasedOnTypeOfMount(&bucket, &testDir)

	// Delete if already exist.
	deleteManagedFoldersInTestDir(managedFolder, bucket, testDir)
	gcloudCreateManagedFolderCmd := fmt.Sprintf("alpha storage managed-folders create gs://%s/%s/%s", bucket, testDir, managedFolder)
	_, err := operations.ExecuteGcloudCommandf(gcloudCreateManagedFolderCmd)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error while creating managed folder: %v", err))
	}
}

func createDirectoryStructureForTest(t *testing.T) {
	createManagedFoldersInTestDir(EmptyManagedFolder1)
	createManagedFoldersInTestDir(EmptyManagedFolder2)
	operations.CreateDirectory(path.Join(setup.MntDir(), testDirName, SimulatedFolder), t)
	f := operations.CreateFile(path.Join(setup.MntDir(), testDirName, File), setup.FilePermission_0600, t)
	operations.CloseFile(f)
}

func TestListDirectoryForEmptyManagedFolders(t *testing.T) {
	setup.SetupTestDirectory(testDirName)

	// Create directory structure for testing.
	createDirectoryStructureForTest(t)

	// Recursively walk into directory and test.
	err := filepath.WalkDir(path.Join(setup.MntDir(), testDirName), func(path string, dir fs.DirEntry, err error) error {
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

		// Check if managedFolderTest directory has correct data.
		if dir.IsDir() && dir.Name() == testDirName {
			// numberOfObjects - 4
			if len(objs) != NumberOfObjectsInDirForListTest {
				t.Errorf("Incorrect number of objects in the directory expectected %d: got %d: ", NumberOfObjectsInDirForListTest, len(objs))
			}

			// testBucket/managedFolderTest/emptyManagedFolder1   -- ManagedFolder1
			if objs[0].Name() != EmptyManagedFolder1 || objs[0].IsDir() != true {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", EmptyManagedFolder1, objs[0].Name())
			}

			// testBucket/managedFolderTest/emptyManagedFolder2     -- ManagedFolder2
			if objs[1].Name() != EmptyManagedFolder2 || objs[1].IsDir() != true {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", EmptyManagedFolder2, objs[1].Name())
			}

			// testBucket/managedFolderTest/simulatedFolder   -- SimulatedFolder
			if objs[2].Name() != SimulatedFolder || objs[2].IsDir() != true {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", SimulatedFolder, objs[2].Name())
			}

			// testBucket/managedFolderTest/testFile  -- File
			if objs[3].Name() != File || objs[3].IsDir() != false {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", File, objs[3].Name())
			}

			return nil
		}

		// Check if subDirectory is empty.
		if dir.IsDir() && (dir.Name() == EmptyManagedFolder1 || dir.Name() == EmptyManagedFolder2 || dir.Name() == SimulatedFolder) {
			// numberOfObjects - 0
			if len(objs) != 0 {
				t.Errorf("Incorrect number of objects in the directory expectected %d: got %d: ", 0, len(objs))
			}
		}

		return nil
	})
	if err != nil {
		t.Errorf("error walking the path : %v\n", err)
		return
	}
}
