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

package managed_folders

import (
	"fmt"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/test_setup"
	"log"
	"path"
	"testing"
)

const (
	ViewPermission = "objectViewer"
	testDirName2 = "ManagedFolderTest2"
	ManagedFolder1             = "managedFolder1"
	ManagedFolder2             = "managedFolder2"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type managedFoldersBucketViewPermissionFolderNil struct {
	flags []string
}

func (s *managedFoldersBucketViewPermissionFolderNil) Setup(t *testing.T) {
	setup.SetupTestDirectory(testDirName2)
}

func  (s *managedFoldersBucketViewPermissionFolderNil) TestListNonEmptyManagedFolders(t *testing.T) {
	bucket, testDir := setup.GetBucketAndTestDir(testDirName)
	operations.CreateManagedFoldersInTestDir(ManagedFolder1, bucket, testDir, t)
	operations.CreateManagedFoldersInTestDir(ManagedFolder2, bucket, testDir, t)
}

func (s *managedFoldersBucketViewPermissionFolderNil) Teardown(t *testing.T) {
	// Clean up test directory.
	bucket, testDir := setup.GetBucketAndTestDir(testDirName)
	setup.CleanupDirectoryOnGCS(path.Join(bucket, testDir))
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestManagedFolders_BucketViewPermissionFolderNil(t *testing.T) {
	ts := &managedFoldersBucketViewPermissionFolderNil{}

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// Run tests for mountedDirectory only if --mountedDirectory  and --testBucket flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	configFile := setup.YAMLConfigFile(
		getMountConfigForEmptyManagedFolders(),
		"config.yaml")

	serviceAccount, localKeyFilePath := creds_tests.CreateCredentials()
	creds_tests.ApplyPermissionToServiceAccount(serviceAccount, ViewPermission)
	defer creds_tests.RevokePermission(fmt.Sprintf("iam ch -d serviceAccount:%s:%s gs://%s", serviceAccount, ViewPermission, setup.TestBucket()))

	flagSet := [][]string{{"--implicit-dirs", "--config-file=" + configFile, "--key-file=" + localKeyFilePath}}

	// Run tests.
	for _, flags := range flagSet {
		ts.flags = flags
		setup.MountGCSFuseWithGivenMountFunc(ts.flags, mountFunc)
		setup.SetMntDir(mountDir)
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
		setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	}
}
