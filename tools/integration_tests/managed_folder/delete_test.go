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

// Provides integration tests for delete directory.
package managed_folder

import (
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"path"
	"testing"
)

func createTestDirectoryStructure(t *testing.T) {
	bucket := setup.TestBucket()
	testDir := testDirName
	client.SetBucketAndObjectBasedOnTypeOfMount(&bucket, &testDir)

	operations.CreateManagedFoldersInTestDir(ManagedFolder, bucket, testDirName, t)
	filePath := path.Join("/tmp", TestFileInManagedFolder)
	f := operations.CreateFile(filePath, setup.FilePermission_0600, t)
	defer operations.CloseFile(f)
	operations.MoveFileInFolder(filePath, bucket, path.Join(testDirName, ManagedFolder), t)
}

func TestDeleteManagedFolder_BucketAndFolderViewPermissions(t *testing.T) {
	SkipTestIfNotViewerPermission(t)

	setup.SetupTestDirectory(testDirName)
	createTestDirectoryStructure(t)

}

func TestDeleteObjectInManagedFolder_BucketAndFolderViewPermissions(t *testing.T) {
	SkipTestIfNotViewerPermission(t)

	setup.SetupTestDirectory(testDirName)
	createTestDirectoryStructure(t)
}
