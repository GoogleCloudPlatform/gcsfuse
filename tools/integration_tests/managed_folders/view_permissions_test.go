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

package managed_folders

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/test_setup"
)

const (
	testDirNameForNonEmptyManagedFolder = "NonEmptyManagedFoldersTest"
	ViewPermission                      = "objectViewer"
	ManagedFolder1                      = "managedFolder1"
	ManagedFolder2                      = "managedFolder2"
	IAMRoleForViewPermission            = "roles/storage.objectViewer"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type managedFoldersViewPermission struct {
}

func (s *managedFoldersViewPermission) Setup(t *testing.T) {
}

func (s *managedFoldersViewPermission) Teardown(t *testing.T) {
}

////////////////////////////////////////////////////////////////////////
// Helper functions
////////////////////////////////////////////////////////////////////////

func createDirectoryStructureForNonEmptyManagedFolders(t *testing.T) {
	// testBucket/NonEmptyManagedFoldersTest/managedFolder1
	// testBucket/NonEmptyManagedFoldersTest/managedFolder1/testFile
	// testBucket/NonEmptyManagedFoldersTest/managedFolder2
	// testBucket/NonEmptyManagedFoldersTest/managedFolder2/testFile
	// testBucket/NonEmptyManagedFoldersTest/simulatedFolder
	// testBucket/NonEmptyManagedFoldersTest/simulatedFolder/testFile
	// testBucket/NonEmptyManagedFoldersTest/testFile
	bucket, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(testDirNameForNonEmptyManagedFolder)
	operations.CreateManagedFoldersInBucket(path.Join(testDir, ManagedFolder1), bucket, t)
	f := operations.CreateFile(path.Join("/tmp", File), setup.FilePermission_0600, t)
	defer operations.CloseFile(f)
	operations.CopyFileInBucket(path.Join("/tmp", File), path.Join(testDir, ManagedFolder1), bucket, t)
	operations.CreateManagedFoldersInBucket(path.Join(testDir, ManagedFolder2), bucket, t)
	operations.CopyFileInBucket(path.Join("/tmp", File), path.Join(testDir, ManagedFolder2), bucket, t)
	operations.CopyFileInBucket(path.Join("/tmp", File), path.Join(testDir, SimulatedFolder), bucket, t)
	operations.CopyFileInBucket(path.Join("/tmp", File), testDir, bucket, t)
}

func cleanup(bucket, testDir, serviceAccount string, t *testing.T) {
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, IAMRoleForViewPermission, t)
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, IAMRoleForViewPermission, t)
	operations.DeleteManagedFoldersInBucket(path.Join(testDir, ManagedFolder1), setup.TestBucket(), t)
	operations.DeleteManagedFoldersInBucket(path.Join(testDir, ManagedFolder2), setup.TestBucket(), t)
	setup.CleanupDirectoryOnGCS(path.Join(bucket, testDir))
}

func (s *managedFoldersViewPermission) TestListNonEmptyManagedFolders(t *testing.T) {
	// Recursively walk into directory and test.
	err := filepath.WalkDir(path.Join(setup.MntDir(), testDirNameForNonEmptyManagedFolder), func(path string, dir fs.DirEntry, err error) error {
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
		if dir.Name() == testDirNameForNonEmptyManagedFolder {
			// numberOfObjects - 4
			if len(objs) != NumberOfObjectsInDirForListTest {
				t.Errorf("Incorrect number of objects in the directory %s expectected %d: got %d: ", dir.Name(), NumberOfObjectsInDirForListTest, len(objs))
			}

			// testBucket/NonEmptyManagedFoldersTest/managedFolder1  -- ManagedFolder1
			if objs[0].Name() != ManagedFolder1 || objs[0].IsDir() != true {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", EmptyManagedFolder1, objs[0].Name())
			}

			// testBucket/NonEmptyManagedFoldersTest/managedFolder2     -- ManagedFolder2
			if objs[1].Name() != ManagedFolder2 || objs[1].IsDir() != true {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", EmptyManagedFolder2, objs[1].Name())
			}

			// testBucket/NonEmptyManagedFoldersTest/simulatedFolder   -- SimulatedFolder
			if objs[2].Name() != SimulatedFolder || objs[2].IsDir() != true {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", SimulatedFolder, objs[2].Name())
			}

			// testBucket/NonEmptyManagedFoldersTest/testFile  -- File
			if objs[3].Name() != File || objs[3].IsDir() != false {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", File, objs[3].Name())
			}
			return nil
		}
		// Check if subDirectory is empty.
		if dir.Name() == ManagedFolder1 {
			// numberOfObjects - 1
			if len(objs) != 1 {
				t.Errorf("Incorrect number of objects in the directory %s expectected %d: got %d: ", dir.Name(), 1, len(objs))
			}
			// testBucket/NonEmptyManagedFoldersTest/managedFolder1/testFile  -- File
			if objs[0].Name() != File || objs[0].IsDir() != false {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", File, objs[3].Name())
			}
		}
		// Check if subDirectory is empty.
		if dir.Name() == ManagedFolder2 {
			// numberOfObjects - 1
			if len(objs) != 1 {
				t.Errorf("Incorrect number of objects in the directory %s expectected %d: got %d: ", dir.Name(), 1, len(objs))
			}
			// testBucket/NonEmptyManagedFoldersTest/managedFolder2/testFile  -- File
			if objs[0].Name() != File || objs[0].IsDir() != false {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", File, objs[3].Name())
			}
		}
		// Check if subDirectory is empty.
		if dir.Name() == SimulatedFolder {
			// numberOfObjects - 1
			if len(objs) != 1 {
				t.Errorf("Incorrect number of objects in the directory %s expectected %d: got %d: ", dir.Name(), 1, len(objs))
			}

			// testBucket/NonEmptyManagedFoldersTest/simulatedFolder/testFile  -- File
			if objs[0].Name() != File || objs[0].IsDir() != false {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", File, objs[3].Name())
			}
		}
		return nil
	})
	if err != nil {
		t.Errorf("error walking the path : %v\n", err)
		return
	}

}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestManagedFolders_FolderViewPermission(t *testing.T) {
	ts := &managedFoldersViewPermission{}

	// Run tests for mountedDirectory only if --mountedDirectory  and --testBucket flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	// Fetch credentials and apply permission on bucket.
	serviceAccount, localKeyFilePath := creds_tests.CreateCredentials()
	creds_tests.ApplyPermissionToServiceAccount(serviceAccount, ViewPermission)

	flags := []string{"--implicit-dirs", "--key-file=" + localKeyFilePath}

	if setup.OnlyDirMounted() != "" {
		operations.CreateManagedFoldersInBucket(onlyDirMounted, setup.TestBucket(), t)
		defer operations.DeleteManagedFoldersInBucket(onlyDirMounted, setup.TestBucket(), t)
	}
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	defer setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	setup.SetMntDir(mountDir)
	bucket, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(testDirNameForNonEmptyManagedFolder)
	// Create directory structure for testing.
	createDirectoryStructureForNonEmptyManagedFolders(t)
	defer func() {
		// Revoke permission on bucket after unmounting and cleanup.
		creds_tests.RevokePermission(fmt.Sprintf("iam ch -d serviceAccount:%s:%s gs://%s", serviceAccount, ViewPermission, setup.TestBucket()))
		// Clean up....
		cleanup(bucket, testDir, serviceAccount, t)
	}()

	// Run tests.
	log.Printf("Running tests with flags and managed folder have nil permissions: %s", flags)
	test_setup.RunTests(t, ts)

	// Provide storage.objectViewer role to managed folders.
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, IAMRoleForViewPermission, t)
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, IAMRoleForViewPermission, t)

	log.Printf("Running tests with flags and managed folder have view permissions: %s", flags)
	test_setup.RunTests(t, ts)
}
