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
	"io"
	"math/rand"
	"os"
	"path"
	"strings"
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
	testDirName              string
	flags                    []string
	suite.Suite
}

func (s *managedFoldersAdminPermission) SetupSuite() {
	if s.managedFoldersPermission != "nil" {
		bucket, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(s.testDirName)
		s.T().Logf("Creating managed folders in bucket: %s, testDir: %s", bucket, testDir)
		client.CreateManagedFoldersInBucket(testEnv.ctx, testEnv.controlClient, path.Join(testDir, ManagedFolder1), bucket)
		client.CreateManagedFoldersInBucket(testEnv.ctx, testEnv.controlClient, path.Join(testDir, ManagedFolder2), bucket)

		s.T().Logf("Providing %s permission to managed folders...", s.managedFoldersPermission)
		providePermissionToManagedFolder(testEnv.bucket, path.Join(testEnv.testDir, ManagedFolder1), testEnv.serviceAccount, s.managedFoldersPermission, s.T())
		providePermissionToManagedFolder(testEnv.bucket, path.Join(testEnv.testDir, ManagedFolder2), testEnv.serviceAccount, s.managedFoldersPermission, s.T())
		// Waiting for 60 seconds for policy changes to propagate. This values we kept based on our experiments.
		// Adding random delay to avoid thundering herd problem.
		sleepDuration := 60*time.Second + time.Duration(rand.Intn(20))*time.Second
		s.T().Logf("Sleeping for %v to allow policy changes to propagate...", sleepDuration)
		time.Sleep(sleepDuration)
	}

	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, s.flags, testEnv.mountFunc)
	setup.SetMntDir(testEnv.mountDir)
}

func (s *managedFoldersAdminPermission) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)

	if s.managedFoldersPermission != "nil" {
		revokePermissionToManagedFolder(testEnv.bucket, path.Join(testEnv.testDir, ManagedFolder1), testEnv.serviceAccount, s.managedFoldersPermission, s.T())
		revokePermissionToManagedFolder(testEnv.bucket, path.Join(testEnv.testDir, ManagedFolder2), testEnv.serviceAccount, s.managedFoldersPermission, s.T())
	}
}

func (s *managedFoldersAdminPermission) SetupTest() {
	s.T().Logf("Setting up test directory: %s", s.testDirName)

	testEnv.testDirPath = path.Join(setup.MntDir(), s.testDirName)
	err := os.MkdirAll(testEnv.testDirPath, 0755)
	if err != nil && !strings.Contains(err.Error(), "file exists") {
		s.T().Logf("Error while setting up directory %s for testing: %v", testEnv.testDirPath, err)
	}
	s.T().Logf("Creating directory structure in: %s", s.testDirName)
	createDirectoryStructureForNonEmptyManagedFolders(testEnv.ctx, testEnv.storageClient, testEnv.controlClient, s.testDirName, s.T())
}

func (s *managedFoldersAdminPermission) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	// Use GCS API to clean up resources to avoid permission denied errors on mount.
	_, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(s.testDirName)
	err := client.DeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, testDir)
	if err != nil {
		s.T().Logf("Failed to clean up test directory via GCS client in TearDownTest: %v", err)
	}
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *managedFoldersAdminPermission) TestCreateObjectInManagedFolder() {
	testDirPath := path.Join(testEnv.testDirPath, ManagedFolder1)
	file := path.Join(testDirPath, CreateTestFile)

	s.T().Logf("Creating file: %s", file)
	createFileForTest(testEnv.ctx, file, s.T())
}

func (s *managedFoldersAdminPermission) TestDeleteObjectInManagedFolder() {
	filePath := path.Join(testEnv.testDirPath, ManagedFolder1, FileInNonEmptyManagedFoldersTest)

	s.T().Logf("Removing file: %s", filePath)
	operations.RetryUntil(testEnv.ctx, s.T(), 5*time.Second, 60*time.Second, func() (any, error) {
		err := os.Remove(filePath)
		if err != nil {
			return nil, err
		}
		_, err = operations.StatFile(filePath)
		if err == nil {
			return nil, fmt.Errorf("file still exists")
		}
		return nil, nil
	})
}

// Managed folders will not be deleted, but they will become empty. Default empty managed folders will be hidden.
func (s *managedFoldersAdminPermission) TestDeleteManagedFolder() {
	dirPath := path.Join(testEnv.testDirPath, ManagedFolder1)

	s.T().Logf("Removing directory: %s", dirPath)
	operations.RetryUntil(testEnv.ctx, s.T(), 5*time.Second, 60*time.Second, func() (any, error) {
		err := os.RemoveAll(dirPath)
		if err != nil {
			return nil, err
		}
		_, err = os.Stat(dirPath)
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

	s.T().Logf("Copying file from %s to %s", srcCopyFile, destCopyFile)
	operations.RetryUntil(testEnv.ctx, s.T(), 5*time.Second, 60*time.Second, func() (any, error) {
		source, err := os.Open(srcCopyFile)
		if err != nil {
			return nil, err
		}
		defer source.Close()

		destination, err := os.Create(destCopyFile)
		if err != nil {
			return nil, err
		}

		_, err = io.Copy(destination, source)
		if err != nil {
			destination.Close()
			return nil, err
		}

		err = destination.Close()
		if err != nil {
			return nil, err
		}

		_, err = operations.StatFile(destCopyFile)
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

	s.T().Logf("Moving file from %s to %s", srcMoveFile, destMoveFile)
	operations.RetryUntil(testEnv.ctx, s.T(), 5*time.Second, 60*time.Second, func() (any, error) {
		err := operations.Move(srcMoveFile, destMoveFile)
		if err != nil {
			return nil, err
		}
		_, err = operations.StatFile(destMoveFile)
		if err != nil {
			return nil, err
		}
		_, err = operations.StatFile(srcMoveFile)
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
	listNonEmptyManagedFolders(setup.MntDir(), s.testDirName, s.T())
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
		permissions := [][]string{{AdminPermission, "nil"}, {AdminPermission, IAMRoleForViewPermission}, {AdminPermission, IAMRoleForAdminPermission}}
		for i := range permissions {
			ts.testDirName = fmt.Sprintf("%s_%d", TestDirForManagedFolderTest, i)
			testEnv.bucket, testEnv.testDir = setup.GetBucketAndObjectBasedOnTypeOfMount(ts.testDirName)
			ts.bucketPermission = permissions[i][0]
			ts.managedFoldersPermission = permissions[i][1]
			suite.Run(t, ts)
		}
		t.Cleanup(func() {
			for i := range permissions {
				testDirName := fmt.Sprintf("%s_%d", TestDirForManagedFolderTest, i)
				_, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(testDirName)
				client.DeleteManagedFoldersInBucket(testEnv.ctx, testEnv.controlClient, path.Join(testDir, ManagedFolder1), setup.TestBucket())
				client.DeleteManagedFoldersInBucket(testEnv.ctx, testEnv.controlClient, path.Join(testDir, ManagedFolder2), setup.TestBucket())
			}
		})
	}
}

func TestManagedFolders_FolderAdminPermission_Restricted(t *testing.T) {
	ts := &managedFoldersAdminPermission{}
	setup.RunTestsOnlyForStaticMount(testEnv.mountDir, t)

	// Run tests for GCE environment otherwise.
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, "TestManagedFolders_FolderAdminPermission")
	for _, ts.flags = range flagsSet {
		// Revoke Admin permission from bucket if it was applied by other tests.
		creds_tests.RevokePermission(testEnv.ctx, testEnv.storageClient, testEnv.serviceAccount, AdminPermission, setup.TestBucket())
		// Apply View permission to bucket.
		creds_tests.ApplyPermissionToServiceAccount(testEnv.ctx, testEnv.storageClient, testEnv.serviceAccount, ViewPermission, setup.TestBucket())
		
		ts.testDirName = fmt.Sprintf("%s_%s", TestDirForManagedFolderTest, "restricted")
		testEnv.bucket, testEnv.testDir = setup.GetBucketAndObjectBasedOnTypeOfMount(ts.testDirName)
		ts.bucketPermission = ViewPermission
		ts.managedFoldersPermission = IAMRoleForAdminPermission
		
		suite.Run(t, ts)
		
		// Revoke View permission from bucket.
		creds_tests.RevokePermission(testEnv.ctx, testEnv.storageClient, testEnv.serviceAccount, ViewPermission, setup.TestBucket())
		
		t.Cleanup(func() {
			_, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(ts.testDirName)
			client.DeleteManagedFoldersInBucket(testEnv.ctx, testEnv.controlClient, path.Join(testDir, ManagedFolder1), setup.TestBucket())
			client.DeleteManagedFoldersInBucket(testEnv.ctx, testEnv.controlClient, path.Join(testDir, ManagedFolder2), setup.TestBucket())
		})
	}
}
