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
	CopyFolder                    = "copyFolderAdminPerm"
	MoveFolder                    = "moveFolderAdminPerm"
	CopyFile                      = "copyFileAdminPerm"
	MoveFile                      = "moveFileAdminPerm"
	MoveDestFile                  = "moveDestFileAdminPerm"
	TestFile                      = "testFileAdminPerm"
	ManagedFolderCreateDeleteTest = "managedFolderCreateDeleteTest"
	ManagedFolderMoveTest         = "managedFolderMoveTest"
	ManagedFolderCopyTest         = "managedFolderCopyTest"
)

var (
	bucket  string
	testDir string
)

type managedFoldersAdminPermission struct {
}

func (s *managedFoldersAdminPermission) Setup(t *testing.T) {
}

func (s *managedFoldersAdminPermission) Teardown(t *testing.T) {
}

func (s *managedFoldersAdminPermission) TestCreateDeleteObjectInFolderAndDeleteNonEmptyFolder(t *testing.T) {
	bucket, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(testDirNameForNonEmptyManagedFolder)
	folderPath := path.Join(testDir, ManagedFolderCreateDeleteTest)
	operations.CreateManagedFoldersInBucket(folderPath, bucket, t)
	defer operations.DeleteManagedFoldersInBucket(folderPath, bucket, t)
	f := operations.CreateFile(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), setup.FilePermission_0600, t)
	defer operations.CloseFile(f)
	operations.CopyFileInBucket(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), path.Join(testDir, ManagedFolderCreateDeleteTest), bucket, t)

	filePath := path.Join(setup.MntDir(), testDirNameForNonEmptyManagedFolder, ManagedFolderCreateDeleteTest, TestFile)
	// Creating object in managed folder.
	file, err := os.Create(filePath)
	err = file.Close()

	if err != nil {
		t.Errorf("Error in creating file in managed folder.")
	}

	// Deleting object in managed folder.
	err = os.Remove(filePath)
	if err != nil {
		t.Errorf("Error in deleting file in managed folder.")
	}

	err = os.RemoveAll(folderPath)
	if err != nil {
		t.Errorf("Error in deleting managed folder.")
	}
	_, err = os.Stat(folderPath)
	if err == nil {
		t.Errorf("Managed folder exist after deletion...")
	}
}

func (s *managedFoldersAdminPermission) TestMoveFileAndMoveNonEmptyManagedFolder(t *testing.T) {
	bucket, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(testDirNameForNonEmptyManagedFolder)
	folderPath := path.Join(testDir, ManagedFolderMoveTest)
	operations.CreateManagedFoldersInBucket(folderPath, bucket, t)
	defer operations.DeleteManagedFoldersInBucket(folderPath, bucket, t)
	f := operations.CreateFile(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), setup.FilePermission_0600, t)
	defer operations.CloseFile(f)
	operations.CopyFileInBucket(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), path.Join(testDir, ManagedFolderMoveTest), bucket, t)

	srcFile := path.Join(setup.MntDir(), testDirNameForNonEmptyManagedFolder, ManagedFolderMoveTest, MoveFile)
	destFile := path.Join(setup.MntDir(), testDirNameForNonEmptyManagedFolder, ManagedFolderMoveTest, MoveDestFile)

	err := operations.MoveFile(srcFile, destFile)
	if err != nil {
		t.Errorf("Error in moving file managed folder.")
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

	flags := []string{"--implicit-dirs", "--key-file=" + localKeyFilePath}

	if setup.OnlyDirMounted() != "" {
		operations.CreateManagedFoldersInBucket(onlyDirMounted, setup.TestBucket(), t)
		defer operations.DeleteManagedFoldersInBucket(onlyDirMounted, setup.TestBucket(), t)
	}
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	defer setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	setup.SetMntDir(mountDir)
	bucket, testDir = setup.GetBucketAndObjectBasedOnTypeOfMount(testDirNameForNonEmptyManagedFolder)
	createDirectoryStructureForNonEmptyManagedFolders(t)

	// Run tests.
	log.Printf("Running tests with flags, bucket have admin permission and managed folder have nil permissions: %s", flags)
	test_setup.RunTests(t, ts)

	// Provide storage.objectViewer role to managed folders.
	log.Printf("Running tests with flags, bucket have admin permission and managed folder have view permissions: %s", flags)
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, IAMRoleForViewPermission, t)
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, IAMRoleForViewPermission, t)
	test_setup.RunTests(t, ts)
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, IAMRoleForViewPermission, t)
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, IAMRoleForViewPermission, t)

	// Provide storage.objectViewer role to managed folders.
	log.Printf("Running tests with flags, bucket have admin permission and managed folder have admin permissions: %s", flags)
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, IAMRoleForAdminPermission, t)
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, IAMRoleForAdminPermission, t)
	test_setup.RunTests(t, ts)

	// Revoke admin permission on bucket.
	log.Printf("Running tests with flags, bucket have view permission and managed folder have admin permissions: %s", flags)
	creds_tests.RevokePermission(serviceAccount, AdminPermission, setup.TestBucket())
	creds_tests.ApplyPermissionToServiceAccount(serviceAccount, ViewPermission)
	defer creds_tests.RevokePermission(serviceAccount, ViewPermission, setup.TestBucket())
	test_setup.RunTests(t, ts)
	cleanup(bucket, testDir, serviceAccount, IAMRoleForAdminPermission, t)
}
