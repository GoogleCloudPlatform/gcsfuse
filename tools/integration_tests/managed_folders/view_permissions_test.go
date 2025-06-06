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
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_setup"
)

const (
	DestFile   = "destFile"
	DestFolder = "destFolder"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

// The permission granted by roles at project, bucket, and managed folder
// levels apply additively (union) throughout the resource hierarchy.
// Hence here managed folder will have view permission throughout all the tests.
type managedFoldersViewPermission struct {
}

func (s *managedFoldersViewPermission) Setup(t *testing.T) {
}

func (s *managedFoldersViewPermission) Teardown(t *testing.T) {
}

func (s *managedFoldersViewPermission) TestListNonEmptyManagedFolders(t *testing.T) {
	listNonEmptyManagedFolders(t)
}

func (s *managedFoldersViewPermission) TestCreateObjectInManagedFolder(t *testing.T) {
	filePath := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder2, DestFile)

	// The error must happen either at file creation or file handle close.
	file, err := os.Create(filePath)
	if file != nil {
		err = file.Close()
	}

	operations.CheckErrorForReadOnlyFileSystem(t, err)
}

func (s *managedFoldersViewPermission) TestDeleteObjectFromManagedFolder(t *testing.T) {
	err := os.Remove(path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1, FileInNonEmptyManagedFoldersTest))

	if err == nil {
		t.Errorf("File from managed folder gets deleted with view only permission.")
	}

	operations.CheckErrorForReadOnlyFileSystem(t, err)
}

func (s *managedFoldersViewPermission) TestDeleteNonEmptyManagedFolder(t *testing.T) {
	err := os.RemoveAll(path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1))

	if err == nil {
		t.Errorf("Managed folder deleted with view only permission.")
	}

	operations.CheckErrorForReadOnlyFileSystem(t, err)
}

func (s *managedFoldersViewPermission) TestMoveManagedFolder(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)
	destDir := path.Join(setup.MntDir(), TestDirForManagedFolderTest, DestFolder)

	moveAndCheckErrForViewPermission(srcDir, destDir, t)
}

func (s *managedFoldersViewPermission) TestMoveObjectWithInManagedFolder(t *testing.T) {
	srcFile := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1, FileInNonEmptyManagedFoldersTest)
	destFile := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1, DestFile)

	moveAndCheckErrForViewPermission(srcFile, destFile, t)
}

func (s *managedFoldersViewPermission) TestMoveObjectOutOfManagedFolder(t *testing.T) {
	srcFile := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1, FileInNonEmptyManagedFoldersTest)
	destFile := path.Join(setup.MntDir(), TestDirForManagedFolderTest, DestFile)

	moveAndCheckErrForViewPermission(srcFile, destFile, t)
}

func (s *managedFoldersViewPermission) TestCopyNonEmptyManagedFolder(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1)
	destDir := path.Join(setup.MntDir(), TestDirForManagedFolderTest, DestFolder)

	copyDirAndCheckErrForViewPermission(srcDir, destDir, t)
}

func (s *managedFoldersViewPermission) TestCopyObjectWithInManagedFolder(t *testing.T) {
	srcFile := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1, FileInNonEmptyManagedFoldersTest)
	destFile := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1, DestFile)

	copyObjectAndCheckErrForViewPermission(srcFile, destFile, t)
}

func (s *managedFoldersViewPermission) TestCopyObjectOutOfManagedFolder(t *testing.T) {
	srcFile := path.Join(setup.MntDir(), TestDirForManagedFolderTest, ManagedFolder1, FileInNonEmptyManagedFoldersTest)
	destFile := path.Join(setup.MntDir(), TestDirForManagedFolderTest, DestFile)

	copyObjectAndCheckErrForViewPermission(srcFile, destFile, t)
}

// //////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
// //////////////////////////////////////////////////////////////////////
func TestManagedFolders_FolderViewPermission(t *testing.T) {
	ts := &managedFoldersViewPermission{}

	// Fetch credentials and apply permission on bucket.
	serviceAccount, localKeyFilePath := creds_tests.CreateCredentials(ctx)
	creds_tests.ApplyPermissionToServiceAccount(ctx, storageClient, serviceAccount, ViewPermission, setup.TestBucket())
	defer creds_tests.RevokePermission(ctx, storageClient, serviceAccount, ViewPermission, setup.TestBucket())

	flags := []string{"--implicit-dirs", "--key-file=" + localKeyFilePath, "--rename-dir-limit=3"}
	if hnsFlagSet, err := setup.AddHNSFlagForHierarchicalBucket(ctx, storageClient); err == nil {
		flags = hnsFlagSet
		flags = append(flags, "--key-file="+localKeyFilePath)
	}
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	defer setup.UnmountGCSFuse(rootDir)
	setup.SetMntDir(mountDir)

	bucket, testDir = setup.GetBucketAndObjectBasedOnTypeOfMount(TestDirForManagedFolderTest)
	// Create directory structure for testing.
	createDirectoryStructureForNonEmptyManagedFolders(ctx, storageClient, controlClient, t)
	defer cleanup(ctx, storageClient, controlClient, bucket, testDir, serviceAccount, IAMRoleForViewPermission, t)

	// Run tests.
	log.Printf("Running tests with flags and managed folder have nil permissions: %s", flags)
	test_setup.RunTests(t, ts)

	// Provide storage.objectViewer role to managed folders.
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, IAMRoleForViewPermission, t)
	providePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, IAMRoleForViewPermission, t)
	// Waiting for 60 seconds for policy changes to propagate. This values we kept based on our experiments.
	time.Sleep(60 * time.Second)

	log.Printf("Running tests with flags and managed folder have view permissions: %s", flags)
	test_setup.RunTests(t, ts)
}
