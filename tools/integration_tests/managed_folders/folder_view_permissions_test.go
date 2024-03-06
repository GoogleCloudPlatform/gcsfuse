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
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/test_setup"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"
)

const (
	ViewPermission = "objectViewer"
	testDirName2   = "ManagedFolderTest2"
	ManagedFolder1 = "managedFolder1"
	ManagedFolder2 = "managedFolder2"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type managedFoldersBucketViewPermissionFolderNil struct {
	flags []string
}

func (s *managedFoldersBucketViewPermissionFolderNil) Setup(t *testing.T) {
	setup.SetupTestDirectory(testDirName2)
}

func (s *managedFoldersBucketViewPermissionFolderNil) Teardown(t *testing.T) {
	// Clean up test directory.
	bucket, testDir := setup.GetBucketAndTestDir(testDirName2)
	operations.DeleteManagedFoldersInBucket(path.Join(testDir, EmptyManagedFolder1), setup.TestBucket(), t)
	operations.DeleteManagedFoldersInBucket(path.Join(testDir, EmptyManagedFolder2), setup.TestBucket(), t)
	setup.CleanupDirectoryOnGCS(path.Join(bucket, testDir))
}

func createDirectoryStructureForListNonEmptyManagedFolders(t *testing.T) {
	bucket, testDir := setup.GetBucketAndTestDir(testDirName2)
	operations.CreateManagedFoldersInBucket(path.Join(testDir, ManagedFolder1), bucket, t)
	f := operations.CreateFile(path.Join("/tmp", File), setup.FilePermission_0600, t)
	defer operations.CloseFile(f)
	operations.CopyFileInFolder(path.Join("/tmp", File), bucket, path.Join(testDir, ManagedFolder1), t)
	operations.CreateManagedFoldersInBucket(path.Join(testDir, ManagedFolder2), bucket, t)
	operations.CopyFileInFolder(path.Join("/tmp", File), bucket, path.Join(testDir, ManagedFolder2), t)
	operations.CreateDirectory(path.Join(setup.MntDir(), testDirName2, SimulatedFolder), t)
	f = operations.CreateFile(path.Join(setup.MntDir(), testDirName2, File), setup.FilePermission_0600, t)
	operations.CloseFile(f)
}

func (s *managedFoldersBucketViewPermissionFolderNil) TestListNonEmptyManagedFolders(t *testing.T) {
	// Create directory structure for testing.
	createDirectoryStructureForListNonEmptyManagedFolders(t)

	// Recursively walk into directory and test.
	err := filepath.WalkDir(path.Join(setup.MntDir(), testDirName2), func(path string, dir fs.DirEntry, err error) error {
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
		if dir.Name() == testDirName2 {
			// numberOfObjects - 4
			if len(objs) != NumberOfObjectsInDirForListTest {
				t.Errorf("Incorrect number of objects in the directory %s expectected %d: got %d: ", dir.Name(), NumberOfObjectsInDirForListTest, len(objs))
			}

			// testBucket/managedFolderTest/emptyManagedFolder1   -- ManagedFolder1
			if objs[0].Name() != ManagedFolder1 || objs[0].IsDir() != true {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", EmptyManagedFolder1, objs[0].Name())
			}

			// testBucket/managedFolderTest/emptyManagedFolder2     -- ManagedFolder2
			if objs[1].Name() != ManagedFolder2 || objs[1].IsDir() != true {
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
		if dir.Name() == ManagedFolder1 {
			// numberOfObjects - 1
			if len(objs) != 1 {
				t.Errorf("Incorrect number of objects in the directory %s expectected %d: got %d: ", dir.Name(), 0, len(objs))
			}
			// testBucket/managedFolderTest/testFile  -- File
			if objs[0].Name() != File || objs[0].IsDir() != false {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", File, objs[3].Name())
			}
		}
		// Check if subDirectory is empty.
		if dir.Name() == ManagedFolder2 {
			// numberOfObjects - 1
			if len(objs) != 1 {
				t.Errorf("Incorrect number of objects in the directory %s expectected %d: got %d: ", dir.Name(), 0, len(objs))
			}
			// testBucket/managedFolderTest/testFile  -- File
			if objs[0].Name() != File || objs[0].IsDir() != false {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", File, objs[3].Name())
			}
		}
		// Check if subDirectory is empty.
		if dir.Name() == SimulatedFolder {
			// numberOfObjects - 0
			if len(objs) != 0 {
				t.Errorf("Incorrect number of objects in the directory %s expectected %d: got %d: ", dir.Name(), 0, len(objs))
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
// TestMain
////////////////////////////////////////////////////////////////////////

func TestManagedFolders_BucketViewPermissionFolderNil(t *testing.T) {
	ts := &managedFoldersBucketViewPermissionFolderNil{}

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// Run tests for mountedDirectory only if --mountedDirectory  and --testBucket flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	configFile := setup.YAMLConfigFile(
		getMountConfigForEmptyManagedFolders(),
		"config.yaml")

	serviceAccount, localKeyFilePath := creds_tests.CreateCredentials()
	creds_tests.ApplyPermissionToServiceAccount(serviceAccount, ViewPermission)
	defer creds_tests.RevokePermission(fmt.Sprintf("iam ch -d serviceAccount:%s:%s gs://%s", serviceAccount, ViewPermission, setup.TestBucket()))

	flagSet := [][]string{{"--implicit-dirs", "--config-file=" + configFile, "--key-file=" + localKeyFilePath}}

	// Run tests.
	for _, flags := range flagSet {
		ts.flags = flags
		setup.MountGCSFuseWithGivenMountFunc(ts.flags, mountFunc)
		setup.SetMntDir(mountDir)
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
		setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	}
}
