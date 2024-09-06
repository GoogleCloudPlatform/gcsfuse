// Copyright 2024 Google LLC
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

// Test list, delete, move, copy, and create operations on managed folders with the following permissions:
// 1. Bucket with Admin permission and folders with nil permission
// 2. Bucket with Admin permission and folders with admin permission
// 3. Bucket with View permission and folders with admin permission
// 4. Bucket with Admin permission and folders with view permission
package managed_folders

import (
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

const (
	CreateTestFile = "createTestFile"
)

var (
	bucket           string
	testDir          string
	serviceAccount   string
	localKeyFilePath string
)

// The permission granted by roles at project, bucket, and managed folder
// levels apply additively (union) throughout the resource hierarchy.
// Hence here managed folder will have admin permission throughout all the tests.
type managedFoldersAdminPermission struct {
	bucketPermission         string
	managedFoldersPermission string
}

func (s *managedFoldersAdminPermission) Setup(t *testing.T) {
	createDirectoryStructureForNonEmptyManagedFolders(ctx, storageClient, t)
	if s.managedFoldersPermission != "nil" {
		providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, s.managedFoldersPermission, t)
		providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, s.managedFoldersPermission, t)
		// Waiting for 15 seconds as it usually takes 10 seconds for policy changes to propagate.
		time.Sleep(15 * time.Second)
	}
}

func (s *managedFoldersAdminPermission) Teardown(t *testing.T) {
	// Due to bucket view permissions, it prevents cleaning resources outside of managed folders. So we are cleaning managed folders resources only.
	if s.bucketPermission == ViewPermission {
		revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, s.managedFoldersPermission, t)
		setup.CleanUpDir(path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1))
		revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, s.managedFoldersPermission, t)
		setup.CleanUpDir(path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder2))
		return
	}
	setup.CleanUpDir(path.Join(setup.MntDir(), TestDirForManagedFolderTest))
}

func (s *managedFoldersAdminPermission) TestCreateObjectInManagedFolder(t *testing.T) {
	testDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)
	file := path.Join(testDirPath, CreateTestFile)

	createFileForTest(file, t)
}

func (s *managedFoldersAdminPermission) TestDeleteObjectInManagedFolder(t *testing.T) {
	filePath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1, FileInNonEmptyManagedFoldersTest)

	err := os.Remove(filePath)
	if err != nil {
		t.Errorf("Error in removing file from managed folder: %v", err)
	}

	_, err = operations.StatFile(filePath)
	if err == nil {
		t.Errorf("file is not removed.")
	}
}

// Managed folders will not be deleted, but they will become empty. Default empty managed folders will be hidden.
func (s *managedFoldersAdminPermission) TestDeleteManagedFolder(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)

	err := os.RemoveAll(dirPath)
	if err != nil {
		t.Errorf("Error in removing managed folder: %v", err)
	}

	_, err = os.Stat(dirPath)
	if err == nil {
		t.Errorf("Directory is not removed.")
	}
}

func (s *managedFoldersAdminPermission) TestCopyObjectWithInManagedFolder(t *testing.T) {
	testDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)
	srcCopyFile := path.Join(testDirPath, FileInNonEmptyManagedFoldersTest)
	destCopyFile := path.Join(testDirPath, DestFile)

	err := operations.CopyFile(srcCopyFile, destCopyFile)
	if err != nil {
		t.Errorf("Error in copying file managed folder from src: %s to dest %s: %v", srcCopyFile, destCopyFile, err)
	}

	_, err = operations.StatFile(destCopyFile)
	if err != nil {
		t.Errorf("Error in stating destination file: %v", err)
	}
}

func (s *managedFoldersAdminPermission) TestCopyManagedFolder(t *testing.T) {
	srcDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)
	destDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, DestFolder)

	err := operations.CopyDir(srcDirPath, destDirPath)

	if s.bucketPermission == ViewPermission {
		operations.CheckErrorForReadOnlyFileSystem(err, t)
	} else {
		_, err = os.Stat(destDirPath)
		if err != nil {
			t.Errorf("Error in stating destination dir: %v", err)
		}
	}
}

func (s *managedFoldersAdminPermission) TestMoveObjectWithInManagedFolder(t *testing.T) {
	testDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)
	srcMoveFile := path.Join(testDirPath, FileInNonEmptyManagedFoldersTest)
	destMoveFile := path.Join(testDirPath, DestFile)

	err := operations.Move(srcMoveFile, destMoveFile)
	if err != nil {
		t.Errorf("Error in moving file managed folder from src: %s to dest %s: %v", srcMoveFile, destMoveFile, err)
	}

	_, err = operations.StatFile(destMoveFile)
	if err != nil {
		t.Errorf("Error in stating destination file: %v", err)
	}
	_, err = operations.StatFile(srcMoveFile)
	if err == nil {
		t.Errorf("SrcFile is not removed after move.")
	}
}

func (s *managedFoldersAdminPermission) TestMoveManagedFolder(t *testing.T) {
	srcDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)
	destDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, DestFolder)

	err := operations.Move(srcDirPath, destDirPath)

	if s.bucketPermission == ViewPermission {
		operations.CheckErrorForReadOnlyFileSystem(err, t)
	} else {
		_, err = os.Stat(destDirPath)
		if err != nil {
			t.Errorf("Error in stating destination dir: %v", err)
		}
		_, err = os.Stat(srcDirPath)
		if err == nil {
			t.Errorf("SrcDir is not removed after move.")
		}
	}
}

func (s *managedFoldersAdminPermission) TestListNonEmptyManagedFoldersWithAdminPermission(t *testing.T) {
	listNonEmptyManagedFolders(t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestManagedFolders_FolderAdminPermission(t *testing.T) {
	ts := &managedFoldersAdminPermission{}

	setup.RunTestsOnlyForStaticMount(mountDir, t)

	// Fetch credentials and apply permission on bucket.
	serviceAccount, localKeyFilePath = creds_tests.CreateCredentials()
	creds_tests.ApplyPermissionToServiceAccount(serviceAccount, AdminPermission, setup.TestBucket())

	flags := []string{"--implicit-dirs", "--key-file=" + localKeyFilePath, "--rename-dir-limit=5", "--stat-cache-ttl=0"}
	if hnsFlagSet, err := setup.AddHNSFlagForHierarchicalBucket(ctx, storageClient); err == nil {
		flags = hnsFlagSet
		flags = append(flags, "--key-file="+localKeyFilePath, "--stat-cache-ttl=0")
	}

	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	defer setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	setup.SetMntDir(mountDir)

	// Run tests on given {Bucket permission, Managed folder permission}.
	permissions := [][]string{{AdminPermission, "nil"}, {AdminPermission, IAMRoleForViewPermission}, {AdminPermission, IAMRoleForAdminPermission}, {ViewPermission, IAMRoleForAdminPermission}}

	for i := 0; i < len(permissions); i++ {
		log.Printf("Running tests with flags, bucket have %s permission and managed folder have %s permissions: %s", permissions[i][0], permissions[i][1], flags)
		bucket, testDir = setup.GetBucketAndObjectBasedOnTypeOfMount(TestDirForManagedFolderTest)
		ts.bucketPermission = permissions[i][0]
		if ts.bucketPermission == ViewPermission {
			creds_tests.RevokePermission(serviceAccount, AdminPermission, setup.TestBucket())
			creds_tests.ApplyPermissionToServiceAccount(serviceAccount, ViewPermission, setup.TestBucket())
			defer creds_tests.RevokePermission(serviceAccount, ViewPermission, setup.TestBucket())
		}
		ts.managedFoldersPermission = permissions[i][1]

		test_setup.RunTests(t, ts)
	}
	t.Cleanup(func() {
		operations.DeleteManagedFoldersInBucket(path.Join(testDir, ManagedFolder1), setup.TestBucket())
		operations.DeleteManagedFoldersInBucket(path.Join(testDir, ManagedFolder2), setup.TestBucket())
	})
}
