// Copyright 2026 Google LLC
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
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

type managedFoldersRestrictedPermission struct {
	flags []string
	suite.Suite
}

func (s *managedFoldersRestrictedPermission) SetupSuite() {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, s.flags, testEnv.mountFunc)
	setup.SetMntDir(testEnv.mountDir)
}

func (s *managedFoldersRestrictedPermission) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *managedFoldersRestrictedPermission) TestCreateAndReadObject() {
	// We should be able to create and read objects in the mount directory
	// because the mount directory is mapped to the managed folder where we have admin permission.
	filePath := path.Join(setup.MntDir(), "restricted_test_file.txt")

	// Create file
	file, err := os.Create(filePath)
	s.Require().NoError(err)
	_, err = file.Write([]byte("hello restricted world"))
	s.Require().NoError(err)
	err = file.Close()
	s.Require().NoError(err)

	// Read file
	content, err := os.ReadFile(filePath)
	s.Require().NoError(err)
	s.Assert().Equal("hello restricted world", string(content))

	// Clean up file
	err = os.Remove(filePath)
	s.Require().NoError(err)
}

func TestManagedFolders_RestrictedPermission(t *testing.T) {
	if setup.OnlyDirMounted() == "" {
		t.Skip("Skipping restricted permissions test for non-only-dir mounting")
	}

	ts := &managedFoldersRestrictedPermission{}

	// We need to ensure the SA has NO bucket level permissions.
	// Revoke them just in case they were left over from previous tests.
	creds_tests.RevokePermission(testEnv.ctx, testEnv.storageClient, testEnv.serviceAccount, AdminPermission, setup.TestBucket())
	creds_tests.RevokePermission(testEnv.ctx, testEnv.storageClient, testEnv.serviceAccount, ViewPermission, setup.TestBucket())
	// Wait for revocation to propagate.
	time.Sleep(60 * time.Second)

	// Create the managed folder that we will mount.
	// The only-dir mounted name is "TestManagedFolderOnlyDir/" (configured in TestMain).
	folderPath := "TestManagedFolderOnlyDir"

	// We use the testEnv.controlClient (which has admin permissions) to create the folder.
	client.CreateManagedFoldersInBucket(testEnv.ctx, testEnv.controlClient, folderPath, setup.TestBucket())
	defer client.DeleteManagedFoldersInBucket(testEnv.ctx, testEnv.controlClient, folderPath, setup.TestBucket())

	// Grant Admin permission to the restricted SA ONLY on this managed folder.
	providePermissionToManagedFolder(setup.TestBucket(), folderPath, testEnv.serviceAccount, IAMRoleForAdminPermission, t)
	defer revokePermissionToManagedFolder(setup.TestBucket(), folderPath, testEnv.serviceAccount, IAMRoleForAdminPermission, t)

	// Wait for policy propagation.
	time.Sleep(60 * time.Second)

	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, ts.flags = range flagsSet {
		log.Printf("Running restricted permissions test with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
