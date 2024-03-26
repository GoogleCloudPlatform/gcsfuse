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

// Test list, delete, move, copy, and create operations on managed folders with the following permissions:
// 1. Bucket with Admin permission and folders with nil permission
// 2. Bucket with Admin permission and folders with admin permission
// 3. Bucket with View permission and folders with admin permission
// 4. Bucket with Admin permission and folders with view permission
package managed_folders

import (
	"fmt"
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

const (
	MoveFile       = "moveFileAdminPerm"
	MoveDestFile   = "moveDestFileAdminPerm"
	CopyFile       = "copyFileAdminPerm"
	CopyDestFile   = "copyDestFileAdminPerm"
	TestFile       = "testFileAdminPerm"
	CreateTestFile = "createTestFile"
)

var (
	bucket           string
	testDir          string
	serviceAccount   string
	localKeyFilePath string
)

type managedFoldersAdminPermission struct {
	iamPermission string
}

func (s *managedFoldersAdminPermission) Setup(t *testing.T) {
	fmt.Println("In setup")
	bucket, testDir = setup.GetBucketAndObjectBasedOnTypeOfMount(testDirNameForNonEmptyManagedFolder)
	createDirectoryStructureForNonEmptyManagedFolders(t)
	if s.iamPermission != "" {
		providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, s.iamPermission, t)
		providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, s.iamPermission, t)
	}
}

func (s *managedFoldersAdminPermission) Teardown(t *testing.T) {
	fmt.Println("In teardown")
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, s.iamPermission, t)
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, s.iamPermission, t)
	cleanup(bucket, testDir, serviceAccount, s.iamPermission, t)
}

func (s *managedFoldersAdminPermission) TestCreateObjectInManagedFolder(t *testing.T) {
	testDirPath := path.Join(setup.MntDir(), testDirNameForNonEmptyManagedFolder, ManagedFolder1,FileInNonEmptyManagedFoldersTest)
	file := path.Join(testDirPath, FileInNonEmptyManagedFoldersTest)

	createFileForTest(file, t)
}

func (s *managedFoldersAdminPermission) TestDeleteObjectInManagedFolder(t *testing.T) {
	filePath := path.Join(setup.MntDir(), testDirNameForNonEmptyManagedFolder, ManagedFolder1, FileInNonEmptyManagedFoldersTest)

	err := os.Remove(filePath)
	if err != nil {
		t.Errorf("Error in removing file from managed folder: %v", err)
	}
}

// Managed folder will not get deleted but it will become empty and default empty managed folder will get hide.
func (s *managedFoldersAdminPermission) TestDeleteManagedFolder(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), testDirNameForNonEmptyManagedFolder, ManagedFolder1)

	err := os.RemoveAll(dirPath)
	if err != nil {
		t.Errorf("Error in removing managed folder: %v", err)
	}
}

//func (s *managedFoldersAdminPermission) TestCreateMoveCopyAndDeleteObjectInFolder(t *testing.T) {
//	testDirPath := path.Join(setup.MntDir(), testDirNameForNonEmptyManagedFolder, ManagedFolder1)
//	// Create Object test
//	srcMoveFile := path.Join(testDirPath, MoveFile)
//	createFileForTest(srcMoveFile, t)
//
//	srcCopyFile := path.Join(testDirPath, CopyFile)
//	// Creating object in managed folder.
//	createFileForTest(srcCopyFile, t)
//
//	// Move/Rename Object test
//	destMoveFile := path.Join(testDirPath, MoveDestFile)
//	err := operations.RenameFile(srcMoveFile, destMoveFile)
//	if err != nil {
//		t.Errorf("Error in moving file managed folder from src: %s to dest %s: %v", srcMoveFile, destMoveFile, err)
//	}
//	_, err = operations.StatFile(srcMoveFile)
//	if err == nil {
//		t.Errorf("Src file does not get deleted.")
//	}
//	_, err = operations.StatFile(destMoveFile)
//	if err != nil {
//		t.Errorf("Dest file does not get created: %v", err)
//	}
//
//	// Copy Object test
//	destCopyFile := path.Join(testDirPath, CopyDestFile)
//	err = operations.CopyFile(srcCopyFile, destCopyFile)
//	if err != nil {
//		t.Errorf("Error in moving file managed folder from src: %s to dest %s: %v", srcCopyFile, destCopyFile, err)
//	}
//
//	// Delete tests.
//	err = os.RemoveAll(testDirPath)
//	if err != nil {
//		t.Errorf("Error in deleting file in managed folder: %v", err)
//	}
//}

func (s *managedFoldersAdminPermission) TestListNonEmptyManagedFoldersWithAdminPermission(t *testing.T) {
	listNonEmptyManagedFolders(t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestManagedFolders_FolderAdminPermission(t *testing.T) {
	ts := &managedFoldersAdminPermission{}

	if setup.MountedDirectory() != "" {
		t.Logf("These tests will not run with mounted directory..")
		return
	}

	// Fetch credentials and apply permission on bucket.
	serviceAccount, localKeyFilePath = creds_tests.CreateCredentials()
	creds_tests.ApplyPermissionToServiceAccount(serviceAccount, AdminPermission)
	// Revoke permission on bucket.
	defer creds_tests.RevokePermission(serviceAccount, AdminPermission, setup.TestBucket())

	flags := []string{"--implicit-dirs", "--key-file=" + localKeyFilePath, "--rename-dir-limit=5"}

	if setup.OnlyDirMounted() != "" {
		operations.CreateManagedFoldersInBucket(onlyDirMounted, setup.TestBucket(), t)
		defer operations.DeleteManagedFoldersInBucket(onlyDirMounted, setup.TestBucket(), t)
	}
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	defer setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	setup.SetMntDir(mountDir)

	// Run tests.
	log.Printf("Running tests with flags, bucket have admin permission and managed folder have nil permissions: %s", flags)
	test_setup.RunTests(t, ts)

	// Provide storage.objectViewer role to managed folders.
	log.Printf("Running tests with flags, bucket have admin permission and managed folder have view permissions: %s", flags)
	ts.iamPermission = IAMRoleForViewPermission

	test_setup.RunTests(t, ts)

	// Provide storage.objectViewer role to managed folders.
	log.Printf("Running tests with flags, bucket have admin permission and managed folder have admin permissions: %s", flags)
	ts.iamPermission = IAMRoleForAdminPermission
	test_setup.RunTests(t, ts)

	// Revoke admin permission on bucket.
	log.Printf("Running tests with flags, bucket have view permission and managed folder have admin permissions: %s", flags)
	creds_tests.RevokePermission(serviceAccount, AdminPermission, setup.TestBucket())
	creds_tests.ApplyPermissionToServiceAccount(serviceAccount, ViewPermission)
	defer creds_tests.RevokePermission(serviceAccount, ViewPermission, setup.TestBucket())
	test_setup.RunTests(t, ts)
}
