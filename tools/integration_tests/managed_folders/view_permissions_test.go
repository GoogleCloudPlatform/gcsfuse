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
// In both the scenarios bucket have view permission.
// 1. Folders with nil permission
// 2. Folders with view only permission
package managed_folders

import (
	"log"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type managedFoldersViewPermission struct {
}

func (s *managedFoldersViewPermission) Setup(t *testing.T) {
}

func (s *managedFoldersViewPermission) Teardown(t *testing.T) {
}

func (s *managedFoldersViewPermission) TestListNonEmptyManagedFolders(t *testing.T) {
	listNonEmptyManagedFolders(t)
}

// //////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
// //////////////////////////////////////////////////////////////////////
func TestManagedFolders_FolderViewPermission(t *testing.T) {
	ts := &managedFoldersViewPermission{}

	if setup.MountedDirectory() != "" {
		t.Logf("These tests will not run with mounted directory..")
		return
	}

	// Fetch credentials and apply permission on bucket.
	serviceAccount, localKeyFilePath := creds_tests.CreateCredentials()
	creds_tests.ApplyPermissionToServiceAccount(serviceAccount, ViewPermission)

	flags := []string{"--implicit-dirs", "--key-file=" + localKeyFilePath}

	if setup.OnlyDirMounted() != "" {
		operations.CreateManagedFoldersInBucket(onlyDirMounted, setup.TestBucket(), t)
		defer operations.DeleteManagedFoldersInBucket(onlyDirMounted, setup.TestBucket(), t)
	}
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	defer setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	setup.SetMntDir(mountDir)
	bucket, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(testDirNameForNonEmptyManagedFolder)
	// Create directory structure for testing.
	createDirectoryStructureForNonEmptyManagedFolders(t)
	defer func() {
		// Revoke permission on bucket after unmounting and cleanup.
		creds_tests.RevokePermission(serviceAccount, ViewPermission, setup.TestBucket())
		// Clean up....
		cleanup(bucket, testDir, serviceAccount, IAMRoleForViewPermission, t)
	}()

	// Run tests.
	log.Printf("Running tests with flags and managed folder have nil permissions: %s", flags)
	test_setup.RunTests(t, ts)

	// Provide storage.objectViewer role to managed folders.
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, IAMRoleForViewPermission, t)
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, IAMRoleForViewPermission, t)

	log.Printf("Running tests with flags and managed folder have view permissions: %s", flags)
	test_setup.RunTests(t, ts)
}
