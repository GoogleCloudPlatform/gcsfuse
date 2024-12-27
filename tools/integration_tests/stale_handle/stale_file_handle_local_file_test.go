// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stale_handle

import (
	"log"
	"path"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/suite"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"

	"github.com/stretchr/testify/assert"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type staleFileHandleLocalFile struct {
	flags []string
	suite.Suite
}

func (s *staleFileHandleLocalFile) Setup(t *testing.T) {
	mountGCSFuseAndSetupTestDir(s.flags, ctx, storageClient, testDirName)
}

func (s *staleFileHandleLocalFile) Teardown(t *testing.T) {
	setup.UnmountGCSFuse(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *staleFileHandleLocalFile) TestLocalInodeClobberedRemotelySyncAndCloseThrowsStaleFileHandleError(t *testing.T) {
	testCaseDir := "TestLocalInodeClobberedRemotelySyncAndCloseThrowsStaleFileHandleError"
	targetDir := path.Join(testDirPath, testCaseDir)
	operations.CreateDirectory(targetDir, t)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, targetDir, FileName1, t)
	// Dirty the file by giving it some contents.
	operations.WriteWithoutClose(fh, FileContents, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, path.Join(testDirName, testCaseDir), FileName1, t)
	// Replace the underlying object with a new generation.
	CreateObjectInGCSTestDir(ctx, storageClient, path.Join(testDirName, testCaseDir), FileName1, GCSFileContent, t)

	operations.SyncFileShouldThrowStaleHandleError(fh, t)
	operations.CloseFileShouldThrowStaleHandleError(fh, t)

	ValidateObjectContentsFromGCS(ctx, storageClient, path.Join(testDirName, testCaseDir), FileName1, GCSFileContent, t)
}

func (s *staleFileHandleLocalFile) TestUnlinkedLocalInodeSyncAndCloseThrowsStaleFileHandleError(t *testing.T) {
	testCaseDir := "TestUnlinkedLocalInodeSyncAndCloseThrowsStaleFileHandleError"
	targetDir := path.Join(testDirPath, testCaseDir)
	operations.CreateDirectory(targetDir, t)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, targetDir, FileName1, t)
	// Unlink the local file.
	operations.RemoveFile(fh.Name())
	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(t, fh.Name())
	// Write to unlinked local file.
	operations.WriteWithoutClose(fh, FileContents, t)

	operations.SyncFileShouldThrowStaleHandleError(fh, t)
	operations.CloseFileShouldThrowStaleHandleError(fh, t)

	// Verify unlinked file is not present on GCS.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, path.Join(testDirName, testCaseDir), FileName1, t)
}

func (s *staleFileHandleLocalFile) TestUnlinkedDirectoryContainingSyncedAndLocalFilesCloseThrowsStaleFileHandleError(t *testing.T) {
	testCaseDir := "TestUnlinkedDirectoryContainingSyncedAndLocalFilesCloseThrowsStaleFileHandleError"
	targetDir := path.Join(testDirPath, testCaseDir)
	operations.CreateDirectory(targetDir, t)
	explicitDir := path.Join(targetDir, ExplicitDirName)
	// Create explicit directory with one synced and one local file.
	operations.CreateDirectory(explicitDir, t)
	CreateObjectInGCSTestDir(ctx, storageClient, path.Join(testDirName, testCaseDir, ExplicitDirName), ExplicitFileName1, "", t)
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, explicitDir, ExplicitLocalFileName1, t)
	err := DeleteObjectOnGCS(ctx, storageClient, path.Join(testDirName, testCaseDir, ExplicitDirName)+"/")
	assert.NoError(t, err)
	operations.ValidateNoFileOrDirError(t, explicitDir+"/")
	operations.ValidateNoFileOrDirError(t, path.Join(explicitDir, ExplicitFileName1))
	operations.ValidateNoFileOrDirError(t, path.Join(explicitDir, ExplicitLocalFileName1))
	// Validate writing content to unlinked local file does not throw error.
	operations.WriteWithoutClose(fh, FileContents, t)

	err = operations.CloseLocalFile(t, &fh)

	operations.ValidateStaleNFSFileHandleError(t, err)
	// Validate both local and synced files are deleted.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testCaseDir, ExplicitDirName, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testCaseDir, path.Join(ExplicitDirName, ExplicitFileName1), t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testCaseDir, path.Join(ExplicitDirName, ExplicitLocalFileName1), t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestStaleFileHandleLocalFileTest(t *testing.T) {
	ts := &staleFileHandleLocalFile{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--metadata-cache-ttl-secs=0", "--precondition-errors=true", "--implicit-dirs"},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
