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
	"fmt"
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

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const (
	CreateTestFile = "createTestFile"
)

// The permission granted by roles at project, bucket, and managed folder
// levels apply additively (union) throughout the resource hierarchy.
// Hence, here managed folder will have admin permission throughout all the tests.
type managedFoldersAdminPermission struct {
	bucketPermission         string
	managedFoldersPermission string
	flags                    []string
	suite.Suite
}

func (s *managedFoldersAdminPermission) SetupSuite() {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, s.flags, testEnv.mountFunc)
	setup.SetMntDir(testEnv.mountDir)

	if s.managedFoldersPermission != "nil" {
		bucket, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(TestDirForManagedFolderTest)
		client.CreateManagedFoldersInBucket(testEnv.ctx, testEnv.controlClient, path.Join(testDir, ManagedFolder1), bucket)
		client.CreateManagedFoldersInBucket(testEnv.ctx, testEnv.controlClient, path.Join(testDir, ManagedFolder2), bucket)

		providePermissionToManagedFolder(testEnv.bucket, path.Join(testEnv.testDir, ManagedFolder1), testEnv.serviceAccount, s.managedFoldersPermission, s.T())
		providePermissionToManagedFolder(testEnv.bucket, path.Join(testEnv.testDir, ManagedFolder2), testEnv.serviceAccount, s.managedFoldersPermission, s.T())
		// Waiting for 60 seconds for policy changes to propagate. This values we kept based on our experiments.
		time.Sleep(60 * time.Second)
	}
}

func (s *managedFoldersAdminPermission) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)

	if s.managedFoldersPermission != "nil" {
		revokePermissionToManagedFolder(testEnv.bucket, path.Join(testEnv.testDir, ManagedFolder1), testEnv.serviceAccount, s.managedFoldersPermission, s.T())
		revokePermissionToManagedFolder(testEnv.bucket, path.Join(testEnv.testDir, ManagedFolder2), testEnv.serviceAccount, s.managedFoldersPermission, s.T())
	}
}

func (s *managedFoldersAdminPermission) SetupTest() {
	testEnv.testDirPath = setup.SetupTestDirectory(TestDirForManagedFolderTest)
	createDirectoryStructureForNonEmptyManagedFolders(testEnv.ctx, testEnv.storageClient, testEnv.controlClient, TestDirForManagedFolderTest, s.T())
}

func (s *managedFoldersAdminPermission) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	// Due to bucket view permissions, it prevents cleaning resources outside managed folders. So we are cleaning managed folders resources only.
	if s.bucketPermission == ViewPermission {
		setup.CleanUpDir(path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1))
		setup.CleanUpDir(path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder2))
		return
	}
	setup.CleanUpDir(path.Join(setup.MntDir(), TestDirForManagedFolderTest))
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *managedFoldersAdminPermission) TestCreateObjectInManagedFolder() {
	testDirPath := path.Join(testEnv.testDirPath, ManagedFolder1)
	file := path.Join(testDirPath, CreateTestFile)

	createFileForTest(file, s.T())
}

func (s *managedFoldersAdminPermission) TestDeleteObjectInManagedFolder() {
	filePath := path.Join(testEnv.testDirPath, ManagedFolder1, FileInNonEmptyManagedFoldersTest)

	err := os.Remove(filePath)
	if err != nil {
		s.T().Errorf("Error in removing file from managed folder: %v", err)
	}

	operations.RetryUntil(testEnv.ctx, s.T(), 5*time.Second, 60*time.Second, func() (any, error) {
		_, err := operations.StatFile(filePath)
		if err == nil {
			return nil, fmt.Errorf("file still exists")
		}
		return nil, nil
	})
}

// Managed folders will not be deleted, but they will become empty. Default empty managed folders will be hidden.
func (s *managedFoldersAdminPermission) TestDeleteManagedFolder() {
	dirPath := path.Join(testEnv.testDirPath, ManagedFolder1)

	err := os.RemoveAll(dirPath)
	if err != nil {
		s.T().Errorf("Error in removing managed folder: %v", err)
	}

	operations.RetryUntil(testEnv.ctx, s.T(), 5*time.Second, 60*time.Second, func() (any, error) {
		_, err := os.Stat(dirPath)
		if err == nil {
			return nil, fmt.Errorf("directory still exists")
		}
		return nil, nil
	})
}

func (s *managedFoldersAdminPermission) TestCopyObjectWithInManagedFolder() {
	testDirPath := path.Join(testEnv.testDirPath, ManagedFolder1)
	srcCopyFile := path.Join(testDirPath, FileInNonEmptyManagedFoldersTest)
	destCopyFile := path.Join(testDirPath, DestFile)

	err := operations.CopyFile(srcCopyFile, destCopyFile)
	if err != nil {
		s.T().Errorf("Error in copying file managed folder from src: %s to dest %s: %v", srcCopyFile, destCopyFile, err)
	}

	operations.RetryUntil(testEnv.ctx, s.T(), 5*time.Second, 60*time.Second, func() (any, error) {
		_, err := operations.StatFile(destCopyFile)
		return nil, err
	})
}

func (s *managedFoldersAdminPermission) TestCopyManagedFolder() {
	srcDirPath := path.Join(testEnv.testDirPath, ManagedFolder1)
	destDirPath := path.Join(testEnv.testDirPath, DestFolder)

	err := operations.CopyDir(srcDirPath, destDirPath)

	if s.bucketPermission == ViewPermission {
		operations.CheckErrorForReadOnlyFileSystem(s.T(), err)
	} else {
		_, err = os.Stat(destDirPath)
		if err != nil {
			s.T().Errorf("Error in stating destination dir: %v", err)
		}
	}
}

func (s *managedFoldersAdminPermission) TestMoveObjectWithInManagedFolder() {
	testDirPath := path.Join(testEnv.testDirPath, ManagedFolder1)
	srcMoveFile := path.Join(testDirPath, FileInNonEmptyManagedFoldersTest)
	destMoveFile := path.Join(testDirPath, DestFile)

	err := operations.Move(srcMoveFile, destMoveFile)
	if err != nil {
		s.T().Errorf("Error in moving file managed folder from src: %s to dest %s: %v", srcMoveFile, destMoveFile, err)
	}

	operations.RetryUntil(testEnv.ctx, s.T(), 5*time.Second, 60*time.Second, func() (any, error) {
		_, err := operations.StatFile(destMoveFile)
		return nil, err
	})
	operations.RetryUntil(testEnv.ctx, s.T(), 5*time.Second, 60*time.Second, func() (any, error) {
		_, err := operations.StatFile(srcMoveFile)
		if err == nil {
			return nil, fmt.Errorf("srcFile still exists after move")
		}
		return nil, nil
	})
}

func (s *managedFoldersAdminPermission) TestMoveManagedFolder() {
	srcDirPath := path.Join(testEnv.testDirPath, ManagedFolder1)
	destDirPath := path.Join(testEnv.testDirPath, DestFolder)

	err := operations.Move(srcDirPath, destDirPath)

	if s.bucketPermission == ViewPermission {
		operations.CheckErrorForReadOnlyFileSystem(s.T(), err)
	} else {
		_, err = os.Stat(destDirPath)
		if err != nil {
			s.T().Errorf("Error in stating destination dir: %v", err)
		}
		_, err = os.Stat(srcDirPath)
		if err == nil {
			s.T().Errorf("SrcDir is not removed after move.")
		}
	}
}

func (s *managedFoldersAdminPermission) TestListNonEmptyManagedFoldersWithAdminPermission() {
	listNonEmptyManagedFolders(setup.MntDir(), TestDirForManagedFolderTest, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestManagedFolders_FolderAdminPermission(t *testing.T) {
	ts := &managedFoldersAdminPermission{}
	setup.RunTestsOnlyForStaticMount(testEnv.mountDir, t)

	// Run tests for GCE environment otherwise.
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, ts.flags = range flagsSet {
		creds_tests.ApplyPermissionToServiceAccount(testEnv.ctx, testEnv.storageClient, testEnv.serviceAccount, AdminPermission, setup.TestBucket())
		// Run tests on given {Bucket permission, Managed folder permission}.
		permissions := [][]string{{AdminPermission, "nil"}, {AdminPermission, IAMRoleForViewPermission}, {AdminPermission, IAMRoleForAdminPermission}, {ViewPermission, IAMRoleForAdminPermission}}
		for i := range permissions {
			testEnv.bucket, testEnv.testDir = setup.GetBucketAndObjectBasedOnTypeOfMount(TestDirForManagedFolderTest)
			ts.bucketPermission = permissions[i][0]
			if ts.bucketPermission == ViewPermission {
				creds_tests.RevokePermission(testEnv.ctx, testEnv.storageClient, testEnv.serviceAccount, AdminPermission, setup.TestBucket())
				creds_tests.ApplyPermissionToServiceAccount(testEnv.ctx, testEnv.storageClient, testEnv.serviceAccount, ViewPermission, setup.TestBucket())
			}
			ts.managedFoldersPermission = permissions[i][1]
			suite.Run(t, ts)
			if ts.bucketPermission == ViewPermission {
				creds_tests.RevokePermission(testEnv.ctx, testEnv.storageClient, testEnv.serviceAccount, ViewPermission, setup.TestBucket())
			}
		}
		t.Cleanup(func() {
			client.DeleteManagedFoldersInBucket(testEnv.ctx, testEnv.controlClient, path.Join(testEnv.testDir, ManagedFolder1), setup.TestBucket())
			client.DeleteManagedFoldersInBucket(testEnv.ctx, testEnv.controlClient, path.Join(testEnv.testDir, ManagedFolder2), setup.TestBucket())
		})
	}
}
