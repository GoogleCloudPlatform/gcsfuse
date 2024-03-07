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
	testDirNameForEmptyManagedFolder = "NonEmptyManagedFoldersTest"
	ViewPermission                   = "objectViewer"
	ManagedFolder1                   = "managedFolder1"
	ManagedFolder2                   = "managedFolder2"
	CopyFolder                       = "copyFolder"
	CopyFile                         = "copyFile"
	IAMRole                          = "roles/storage.objectViewer"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type managedFoldersBucketViewPermissionFolderNil struct {
}

func (s *managedFoldersBucketViewPermissionFolderNil) Setup(t *testing.T) {
}

func (s *managedFoldersBucketViewPermissionFolderNil) Teardown(t *testing.T) {
}

var (
	bucket  string
	testDir string
)

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
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, IAMRole, t)
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, IAMRole, t)
	operations.DeleteManagedFoldersInBucket(path.Join(testDir, ManagedFolder1), setup.TestBucket(), t)
	operations.DeleteManagedFoldersInBucket(path.Join(testDir, ManagedFolder2), setup.TestBucket(), t)
	setup.CleanupDirectoryOnGCS(path.Join(bucket, testDir))
}

func (s *managedFoldersBucketViewPermissionFolderNil) TestListNonEmptyManagedFolders(t *testing.T) {
	// Recursively walk into directory and test.
	err := filepath.WalkDir(path.Join(setup.MntDir(), testDirNameForEmptyManagedFolder), func(path string, dir fs.DirEntry, err error) error {
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
		if dir.Name() == testDirNameForEmptyManagedFolder {
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

func (s *managedFoldersBucketViewPermissionFolderNil) TestDeleteNonEmptyManagedFolder(t *testing.T) {
	err := os.RemoveAll(path.Join(testDir, ManagedFolder1))

	if err == nil {
		t.Errorf("Managed folder deleted with view only permission.")
	}

	setup.CheckErrorForReadOnlyFileSystem(err, t)
}

func (s *managedFoldersBucketViewPermissionFolderNil) TestDeleteObjectFromManagedFolder(t *testing.T) {
	err := os.Remove(path.Join(testDir, ManagedFolder1, File))

	if err == nil {
		t.Errorf("File from managed folder get deleted with view only permission.")
	}

	setup.CheckErrorForReadOnlyFileSystem(err, t)
}

func (s *managedFoldersBucketViewPermissionFolderNil) TestCreateObjectInManagedFolder(t *testing.T) {
	filePath := path.Join(testDir, ManagedFolder2, "test.txt")
	file, err := os.OpenFile(filePath, os.O_CREATE, setup.FilePermission_0600)

	if err == nil {
		t.Errorf("File is created in read-only file system.")
		operations.CloseFile(file)
	}

	setup.CheckErrorForReadOnlyFileSystem(err, t)
}

func (s *managedFoldersBucketViewPermissionFolderNil) TestCopyNonEmptyManagedFolder(t *testing.T) {
	srcDir := path.Join(testDir, ManagedFolder1)
	destDir := path.Join(testDir, CopyFolder)

	err := operations.CopyDir(srcDir, destDir)
	if err == nil {
		t.Errorf("Managed folder get copied with view only permission.")
	}

	setup.CheckErrorForReadOnlyFileSystem(err, t)
}

func (s *managedFoldersBucketViewPermissionFolderNil) TestCopyObjectInManagedFolder(t *testing.T) {
	srcFile := path.Join(testDir, ManagedFolder1, File)
	destFile := path.Join(testDir, CopyFolder, CopyFile)

	err := operations.CopyDir(srcFile, destFile)
	if err == nil {
		t.Errorf("Managed folder get copied with view only permission.")
	}

	setup.CheckErrorForReadOnlyFileSystem(err, t)
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestManagedFolders_BucketViewPermissionFolderNil(t *testing.T) {
	ts := &managedFoldersBucketViewPermissionFolderNil{}

	// Run tests for mountedDirectory only if --mountedDirectory  and --testBucket flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	bucket, testDir = setup.GetBucketAndObjectBasedOnTypeOfMount(testDirNameForEmptyManagedFolder)

	configFile := setup.YAMLConfigFile(
		getMountConfigForEmptyManagedFolders(),
		"config.yaml")

	// Fetch credentials and apply permission on bucket.
	serviceAccount, localKeyFilePath := creds_tests.CreateCredentials()
	//creds_tests.ApplyPermissionToServiceAccount(serviceAccount, ViewPermission)

	flags := []string{"--implicit-dirs", "--config-file=" + configFile, "--key-file=" + localKeyFilePath}

	if setup.OnlyDirMounted() != "" {
		operations.CreateManagedFoldersInBucket(onlyDirMounted, setup.TestBucket(), t)
		defer operations.DeleteManagedFoldersInBucket(onlyDirMounted, setup.TestBucket(), t)
	}
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	defer setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	setup.SetMntDir(mountDir)
	// Create directory structure for testing.
	createDirectoryStructureForNonEmptyManagedFolders(t)
	// Clean up....
	defer cleanup(bucket, testDir, serviceAccount, t)
	// Revoke permission on bucket after unmounting and cleanup.
	defer creds_tests.RevokePermission(fmt.Sprintf("iam ch -d serviceAccount:%s:%s gs://%s", serviceAccount, ViewPermission, setup.TestBucket()))

	// Run tests.
	log.Printf("Running tests with flags and managed folder have nil permissions: %s", flags)
	test_setup.RunTests(t, ts)

	// Provide storage.objectViewer role to managed folders.
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, IAMRole, t)
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, IAMRole, t)

	log.Printf("Running tests with flags and managed folder have view permissions: %s", flags)
	test_setup.RunTests(t, ts)
}
