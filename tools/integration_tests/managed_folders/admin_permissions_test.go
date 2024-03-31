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
	MoveFile       = "moveFileAdminPerm"
	MoveDestFile   = "moveDestFileAdminPerm"
	CopyFile       = "copyFileAdminPerm"
	CopyDestFile   = "copyDestFileAdminPerm"
	CreateTestFile = "createTestFile"
)

var (
	bucket           string
	testDir          string
	serviceAccount   string
	localKeyFilePath string
)

type managedFoldersAdminPermission struct {
	iamPermission    string
	bucketPermission string
}

func (s *managedFoldersAdminPermission) Setup(t *testing.T) {
	bucket, testDir = setup.GetBucketAndObjectBasedOnTypeOfMount(TestDirForManagedFolderTest)
	createDirectoryStructureForNonEmptyManagedFolders(t)
	if s.iamPermission != "" {
		providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, s.iamPermission, t)
		providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, s.iamPermission, t)
	}
}

func (s *managedFoldersAdminPermission) Teardown(t *testing.T) {
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, s.iamPermission, t)
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, s.iamPermission, t)
	cleanup(bucket, testDir, serviceAccount, s.iamPermission, t)
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
}

// Managed folders will not be deleted, but they will become empty. Default empty managed folders will be hidden.
func (s *managedFoldersAdminPermission) TestDeleteManagedFolder(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)

	err := os.RemoveAll(dirPath)
	if err != nil {
		t.Errorf("Error in removing managed folder: %v", err)
	}
}

func (s *managedFoldersAdminPermission) TestCopyObjectInManagedFolder(t *testing.T) {
	testDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)
	srcCopyFile := path.Join(testDirPath, CopyFile)
	// Creating object in managed folder.
	createFileForTest(srcCopyFile, t)

	destCopyFile := path.Join(testDirPath, DestFolder)

	err := operations.CopyFile(srcCopyFile, destCopyFile)
	if err != nil {
		t.Errorf("Error in copying file managed folder from src: %s to dest %s: %v", srcCopyFile, destCopyFile, err)
	}

	_, err = operations.StatFile(destCopyFile)
	if err != nil {
		t.Errorf("Error in stating destination copy file: %v", err)
	}
}

func (s *managedFoldersAdminPermission) TestCopyManagedFolder(t *testing.T) {
	if s.bucketPermission == ViewPermission {
		t.Logf("This test will run only for bucket with admin permission.")
		t.SkipNow()
	}

	srcDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)
	destDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, DestFolder)

	err := operations.CopyDir(srcDirPath, destDirPath)
	if err != nil {
		t.Errorf("Error in copying directory: %v", err)
	}

	_, err = os.Stat(destDirPath)
	if err != nil {
		t.Errorf("Error in stating destination copy dir: %v", err)
	}
}

func (s *managedFoldersAdminPermission) TestMoveObjectInManagedFolder(t *testing.T) {
	testDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)
	srcMoveFile := path.Join(testDirPath, MoveFile)
	// Creating object in managed folder.
	createFileForTest(srcMoveFile, t)

	destMoveFile := path.Join(testDirPath, DestFile)

	err := operations.Move(srcMoveFile, destMoveFile)
	if err != nil {
		t.Errorf("Error in moving file managed folder from src: %s to dest %s: %v", srcMoveFile, destMoveFile, err)
	}

	_, err = operations.StatFile(destMoveFile)
	if err != nil {
		t.Errorf("Error in stating destination move file: %v", err)
	}
	_, err = operations.StatFile(srcMoveFile)
	if err == nil {
		t.Errorf("SrcFile is not removed after move.")
	}
}

func (s *managedFoldersAdminPermission) TestMoveManagedFolder(t *testing.T) {
	if s.bucketPermission == ViewPermission {
		log.Printf("This test will run only for bucket with admin permission.")
		t.SkipNow()
	}

	srcDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)
	destDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, DestFolder)

	err := operations.Move(srcDirPath, destDirPath)
	if err != nil {
		t.Errorf("Error in moving directory: %v", err)
	}

	_, err = os.Stat(destDirPath)
	if err != nil {
		t.Errorf("Error in stating destination copy dir: %v", err)
	}
	_, err = os.Stat(srcDirPath)
	if err == nil {
		t.Errorf("SrcDir is not removed after move.")
	}
}

func (s *managedFoldersAdminPermission) TestMoveManagedFolderWithViewBucketPermission(t *testing.T) {
	if s.bucketPermission == ViewPermission {
		srcDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)
		destDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, DestFolder)

		moveAndCheckErrForViewPermission(srcDirPath, destDirPath, t)
	}
}

func (s *managedFoldersAdminPermission) TestCopyManagedFolderWithViewBucketPermission(t *testing.T) {
	if s.bucketPermission == ViewPermission {
		srcDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)
		destDirPath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, DestFolder)

		copyDirAndCheckErrForViewPermission(srcDirPath, destDirPath, t)
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

	setup.RunTestsOnlyForStaticMount(mountDir, t)

	// Fetch credentials and apply permission on bucket.
	serviceAccount, localKeyFilePath = creds_tests.CreateCredentials()
	creds_tests.ApplyPermissionToServiceAccount(serviceAccount, AdminPermission)
	// Revoke permission on bucket.
	defer creds_tests.RevokePermission(serviceAccount, AdminPermission, setup.TestBucket())
	ts.bucketPermission = AdminPermission

	flags := []string{"--implicit-dirs", "--key-file=" + localKeyFilePath, "--rename-dir-limit=5"}

	if setup.OnlyDirMounted() != "" {
		operations.CreateManagedFoldersInBucket(onlyDirMounted, setup.TestBucket())
		defer operations.DeleteManagedFoldersInBucket(onlyDirMounted, setup.TestBucket())
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
	ts.bucketPermission = ViewPermission
	defer creds_tests.RevokePermission(serviceAccount, ViewPermission, setup.TestBucket())
	test_setup.RunTests(t, ts)
}
