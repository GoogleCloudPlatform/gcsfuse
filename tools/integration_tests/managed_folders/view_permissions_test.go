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
// In both the scenarios bucket have view permission.
// 1. Folders with nil permission
// 2. Folders with view only permission
package managed_folders

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

const (
	DestFile   = "destFile"
	DestFolder = "destFolder"
)

type managedFoldersViewPermission struct {
	managedFoldersPermission string
	flags                    []string
	suite.Suite
}

func (s *managedFoldersViewPermission) SetupSuite() {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, s.flags, testEnv.mountFunc)
	setup.SetMntDir(testEnv.mountDir)

	if s.managedFoldersPermission != "nil" {
		bucket, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(TestDirForManagedFolderTest)
		client.CreateManagedFoldersInBucket(testEnv.ctx, testEnv.controlClient, path.Join(testDir, ManagedFolder1), bucket)
		client.CreateManagedFoldersInBucket(testEnv.ctx, testEnv.controlClient, path.Join(testDir, ManagedFolder2), bucket)

		providePermissionToManagedFolder(testEnv.bucket, path.Join(testEnv.testDir, ManagedFolder1), testEnv.serviceAccount, IAMRoleForViewPermission, s.T())
		providePermissionToManagedFolder(testEnv.bucket, path.Join(testEnv.testDir, ManagedFolder2), testEnv.serviceAccount, IAMRoleForViewPermission, s.T())
		// Waiting for 60 seconds for policy changes to propagate. This values we kept based on our experiments.
		time.Sleep(60 * time.Second)
	}
}

func (s *managedFoldersViewPermission) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)

	if s.managedFoldersPermission != "nil" {
		revokePermissionToManagedFolder(testEnv.bucket, path.Join(testEnv.testDir, ManagedFolder1), testEnv.serviceAccount, IAMRoleForViewPermission, s.T())
		revokePermissionToManagedFolder(testEnv.bucket, path.Join(testEnv.testDir, ManagedFolder2), testEnv.serviceAccount, IAMRoleForViewPermission, s.T())
	}
}

func (s *managedFoldersViewPermission) SetupTest() {
	testEnv.testDirPath = setup.SetupTestDirectory(TestDirForManagedFolderTest)
	createDirectoryStructureForNonEmptyManagedFolders(testEnv.ctx, testEnv.storageClient, testEnv.controlClient, TestDirForManagedFolderTest, s.T())
}

func (s *managedFoldersViewPermission) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	// Due to bucket view permissions, it prevents cleaning resources outside managed folders. So we are cleaning managed folders resources only.
	if s.managedFoldersPermission != "nil" {
		setup.CleanUpDir(path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1))
		setup.CleanUpDir(path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder2))
		return
	}
	setup.CleanUpDir(path.Join(setup.MntDir(), TestDirForManagedFolderTest))
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *managedFoldersViewPermission) TestListNonEmptyManagedFolders() {
	listNonEmptyManagedFolders(setup.MntDir(), TestDirForManagedFolderTest, s.T())
}

func (s *managedFoldersViewPermission) TestCreateObjectInManagedFolder() {
	filePath := path.Join(testEnv.testDirPath, ManagedFolder2, DestFile)

	// The error must happen either at file creation or file handle close.
	file, err := os.Create(filePath)
	if file != nil {
		err = file.Close()
	}

	operations.CheckErrorForReadOnlyFileSystem(s.T(), err)
}

func (s *managedFoldersViewPermission) TestDeleteObjectFromManagedFolder() {
	err := os.Remove(path.Join(testEnv.testDirPath, ManagedFolder1, FileInNonEmptyManagedFoldersTest))

	if err == nil {
		s.T().Errorf("File from managed folder gets deleted with view only permission.")
	}

	operations.CheckErrorForReadOnlyFileSystem(s.T(), err)
}

func (s *managedFoldersViewPermission) TestDeleteNonEmptyManagedFolder() {
	err := os.RemoveAll(path.Join(testEnv.testDirPath, ManagedFolder1))

	if err == nil {
		s.T().Errorf("Managed folder deleted with view only permission.")
	}

	operations.CheckErrorForReadOnlyFileSystem(s.T(), err)
}

func (s *managedFoldersViewPermission) TestMoveManagedFolder() {
	srcDir := path.Join(testEnv.testDirPath, ManagedFolder1)
	destDir := path.Join(testEnv.testDirPath, DestFolder)

	moveAndCheckErrForViewPermission(srcDir, destDir, s.T())
}

func (s *managedFoldersViewPermission) TestMoveObjectWithInManagedFolder() {
	srcFile := path.Join(testEnv.testDirPath, ManagedFolder1, FileInNonEmptyManagedFoldersTest)
	destFile := path.Join(testEnv.testDirPath, ManagedFolder1, DestFile)

	moveAndCheckErrForViewPermission(srcFile, destFile, s.T())
}

func (s *managedFoldersViewPermission) TestMoveObjectOutOfManagedFolder() {
	srcFile := path.Join(testEnv.testDirPath, ManagedFolder1, FileInNonEmptyManagedFoldersTest)
	destFile := path.Join(testEnv.testDirPath, DestFile)

	moveAndCheckErrForViewPermission(srcFile, destFile, s.T())
}

func (s *managedFoldersViewPermission) TestCopyNonEmptyManagedFolder() {
	srcDir := path.Join(testEnv.testDirPath, ManagedFolder1)
	destDir := path.Join(testEnv.testDirPath, DestFolder)

	copyDirAndCheckErrForViewPermission(srcDir, destDir, s.T())
}

func (s *managedFoldersViewPermission) TestCopyObjectWithInManagedFolder() {
	srcFile := path.Join(testEnv.testDirPath, ManagedFolder1, FileInNonEmptyManagedFoldersTest)
	destFile := path.Join(testEnv.testDirPath, ManagedFolder1, DestFile)

	copyObjectAndCheckErrForViewPermission(srcFile, destFile, s.T())
}

func (s *managedFoldersViewPermission) TestCopyObjectOutOfManagedFolder() {
	srcFile := path.Join(testEnv.testDirPath, ManagedFolder1, FileInNonEmptyManagedFoldersTest)
	destFile := path.Join(testEnv.testDirPath, DestFile)

	copyObjectAndCheckErrForViewPermission(srcFile, destFile, s.T())
}

// //////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
// //////////////////////////////////////////////////////////////////////
func TestManagedFolders_FolderViewPermission(t *testing.T) {
	ts := &managedFoldersViewPermission{}

	creds_tests.ApplyPermissionToServiceAccount(testEnv.ctx, testEnv.storageClient, testEnv.serviceAccount, ViewPermission, setup.TestBucket())
	defer creds_tests.RevokePermission(testEnv.ctx, testEnv.storageClient, testEnv.serviceAccount, ViewPermission, setup.TestBucket())

	// Run tests for GCE environment otherwise.
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, ts.flags = range flagsSet {
		permissions := []string{"nil", IAMRoleForViewPermission}

		for i := range permissions {
			testEnv.bucket, testEnv.testDir = setup.GetBucketAndObjectBasedOnTypeOfMount(TestDirForManagedFolderTest)
			ts.managedFoldersPermission = permissions[i]
			suite.Run(t, ts)
		}
	}
}
