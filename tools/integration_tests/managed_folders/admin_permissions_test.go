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
	MoveFile     = "moveFileAdminPerm"
	MoveDestFile = "moveDestFileAdminPerm"
	CopyFile     = "copyFileAdminPerm"
	CopyDestFile = "copyDestFileAdminPerm"
	TestFile     = "testFileAdminPerm"
)

var (
	bucket   string
	testDir  string
	testDir2 string
)

type managedFoldersAdminPermission struct {
}

func (s *managedFoldersAdminPermission) Setup(t *testing.T) {
}

func (s *managedFoldersAdminPermission) Teardown(t *testing.T) {
}

func (s *managedFoldersAdminPermission) TestCreateMoveCopyAndDeleteObjectInFolder(t *testing.T) {
	// Create Object test
	srcMoveFile := path.Join(setup.MntDir(), testDirNameForNonEmptyManagedFolder2, ManagedFolder3, MoveFile)
	// Creating object in managed folder.
	file, err := os.Create(srcMoveFile)
	err = file.Close()
	if err != nil {
		t.Errorf("Error in creating file in managed folder: %v", err)
	}
	srcCopyFile := path.Join(setup.MntDir(), testDirNameForNonEmptyManagedFolder2, ManagedFolder3, CopyFile)
	// Creating object in managed folder.
	file, err = os.Create(srcCopyFile)
	err = file.Close()
	if err != nil {
		t.Errorf("Error in creating file in managed folder: %v", err)
	}

	// Move Object test
	destMoveFile := path.Join(setup.MntDir(), testDirNameForNonEmptyManagedFolder2, ManagedFolder3, MoveDestFile)
	err = operations.MoveDir(srcMoveFile, destMoveFile)
	if err != nil {
		t.Errorf("Error in moving file managed folder from src: %s to dest %s: %v", srcMoveFile, destMoveFile, err)
	}
	_, err = operations.StatFile(srcMoveFile)
	if err == nil {
		t.Errorf("Src file does not get deleted.")
	}
	_, err = operations.StatFile(destMoveFile)
	if err != nil {
		t.Errorf("Dest file does not get created: %v", err)
	}

	// Copy Object test
	destCopyFile := path.Join(setup.MntDir(), testDirNameForNonEmptyManagedFolder2, ManagedFolder3, CopyDestFile)
	err = operations.CopyDir(srcCopyFile, destCopyFile)
	if err != nil {
		t.Errorf("Error in moving file managed folder from src: %s to dest %s: %v", srcCopyFile, destCopyFile, err)
	}
	_, err = operations.StatFile(srcCopyFile)
	if err != nil {
		t.Errorf("Src file gets deleted: %v", err)
	}
	_, err = operations.StatFile(destCopyFile)
	if err != nil {
		t.Errorf("Dest file does not gets created: %v", err)
	}

	// Delete tests.
	err = os.RemoveAll(path.Join(setup.MntDir(), testDirNameForNonEmptyManagedFolder2, ManagedFolder3))
	if err != nil {
		t.Errorf("Error in deleting file in managed folder: %v", err)
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

	if setup.MountedDirectory() != "" {
		t.Logf("These tests will not run with mounted directory..")
		return
	}

	// Fetch credentials and apply permission on bucket.
	serviceAccount, localKeyFilePath := creds_tests.CreateCredentials()
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
	bucket, testDir = setup.GetBucketAndObjectBasedOnTypeOfMount(testDirNameForNonEmptyManagedFolder)
	createDirectoryStructureForNonEmptyManagedFolders(t)

	// For create, delete, move, copy tests.
	bucket, testDir2 = setup.GetBucketAndObjectBasedOnTypeOfMount(testDirNameForNonEmptyManagedFolder2)
	operations.CreateManagedFoldersInBucket(path.Join(testDir2, ManagedFolder3), bucket, t)
	f := operations.CreateFile(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), setup.FilePermission_0600, t)
	defer operations.CloseFile(f)
	operations.CopyFileInBucket(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), path.Join(testDir2, ManagedFolder3), bucket, t)

	// Run tests.
	log.Printf("Running tests with flags, bucket have admin permission and managed folder have nil permissions: %s", flags)
	test_setup.RunTests(t, ts)

	// Provide storage.objectViewer role to managed folders.
	log.Printf("Running tests with flags, bucket have admin permission and managed folder have view permissions: %s", flags)
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, IAMRoleForViewPermission, t)
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, IAMRoleForViewPermission, t)
	providePermissionToManagedFolder(bucket, path.Join(testDir2, ManagedFolder3), serviceAccount, IAMRoleForViewPermission, t)
	operations.CopyFileInBucket(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), path.Join(testDir2, ManagedFolder3), bucket, t)
	test_setup.RunTests(t, ts)
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, IAMRoleForViewPermission, t)
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, IAMRoleForViewPermission, t)
	revokePermissionToManagedFolder(bucket, path.Join(testDir2, ManagedFolder3), serviceAccount, IAMRoleForViewPermission, t)

	// Provide storage.objectViewer role to managed folders.
	log.Printf("Running tests with flags, bucket have admin permission and managed folder have admin permissions: %s", flags)
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, IAMRoleForAdminPermission, t)
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, IAMRoleForAdminPermission, t)
	providePermissionToManagedFolder(bucket, path.Join(testDir2, ManagedFolder3), serviceAccount, IAMRoleForAdminPermission, t)
	operations.CopyFileInBucket(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), path.Join(testDir2, ManagedFolder3), bucket, t)
	test_setup.RunTests(t, ts)

	// Revoke admin permission on bucket.
	log.Printf("Running tests with flags, bucket have view permission and managed folder have admin permissions: %s", flags)
	creds_tests.RevokePermission(serviceAccount, AdminPermission, setup.TestBucket())
	creds_tests.ApplyPermissionToServiceAccount(serviceAccount, ViewPermission)
	defer creds_tests.RevokePermission(serviceAccount, ViewPermission, setup.TestBucket())
	operations.CopyFileInBucket(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), path.Join(testDir2, ManagedFolder3), bucket, t)
	test_setup.RunTests(t, ts)
	cleanup(bucket, testDir, serviceAccount, IAMRoleForAdminPermission, t)
	revokePermissionToManagedFolder(bucket, path.Join(testDir2, ManagedFolder3), serviceAccount, IAMRoleForAdminPermission, t)
	operations.DeleteManagedFoldersInBucket(path.Join(testDir2, ManagedFolder3), setup.TestBucket(), t)
}
