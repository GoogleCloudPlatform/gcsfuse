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
	"os"
	"path"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
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

func (s *staleFileHandleLocalFile) SetupSuite() {
	mountGCSFuseAndSetupTestDir(s.flags, ctx, storageClient, testDirName)
}

func (s *staleFileHandleLocalFile) TearDownSuite() {
	setup.UnmountGCSFuse(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *staleFileHandleLocalFile) TestLocalInodeClobberedRemotelySyncAndCloseThrowsStaleFileHandleError() {
	testCaseDir := "TestLocalInodeClobberedRemotelySyncAndCloseThrowsStaleFileHandleError"
	targetDir := path.Join(testDirPath, testCaseDir)
	operations.CreateDirectory(targetDir, s.T())
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, targetDir, FileName1, s.T())
	// Dirty the file by giving it some contents.
	operations.WriteWithoutClose(fh, FileContents, s.T())
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, path.Join(testDirName, testCaseDir), FileName1, s.T())
	// Replace the underlying object with a new generation.
	CreateObjectInGCSTestDir(ctx, storageClient, path.Join(testDirName, testCaseDir), FileName1, GCSFileContent, s.T())

	operations.SyncFileShouldThrowStaleHandleError(fh, s.T())
	operations.CloseFileShouldThrowStaleHandleError(fh, s.T())

	ValidateObjectContentsFromGCS(ctx, storageClient, path.Join(testDirName, testCaseDir), FileName1, GCSFileContent, s.T())
}

func (s *staleFileHandleLocalFile) TestUnlinkedLocalInodeSyncAndCloseThrowsStaleFileHandleError() {
	testCaseDir := "TestUnlinkedLocalInodeSyncAndCloseThrowsStaleFileHandleError"
	targetDir := path.Join(testDirPath, testCaseDir)
	operations.CreateDirectory(targetDir, s.T())
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, targetDir, FileName1, s.T())
	// Unlink the local file.
	operations.RemoveFile(fh.Name())
	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(s.T(), fh.Name())
	// Write to unlinked local file.
	operations.WriteWithoutClose(fh, FileContents, s.T())

	operations.SyncFileShouldThrowStaleHandleError(fh, s.T())
	operations.CloseFileShouldThrowStaleHandleError(fh, s.T())

	// Verify unlinked file is not present on GCS.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, path.Join(testDirName, testCaseDir), FileName1, s.T())
}

func (s *staleFileHandleLocalFile) TestUnlinkedDirectoryContainingSyncedAndLocalFilesCloseThrowsStaleFileHandleError() {
	testCaseDir := "TestUnlinkedDirectoryContainingSyncedAndLocalFilesCloseThrowsStaleFileHandleError"
	targetDir := path.Join(testDirPath, testCaseDir)
	operations.CreateDirectory(targetDir, s.T())
	explicitDir := path.Join(targetDir, ExplicitDirName)
	// Create explicit directory with one synced and one local file.
	operations.CreateDirectory(explicitDir, s.T())
	CreateObjectInGCSTestDir(ctx, storageClient, path.Join(testDirName, testCaseDir, ExplicitDirName), ExplicitFileName1, "", s.T())
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, explicitDir, ExplicitLocalFileName1, s.T())
	err := os.RemoveAll(explicitDir)
	assert.NoError(s.T(), err)
	operations.ValidateNoFileOrDirError(s.T(), explicitDir+"/")
	operations.ValidateNoFileOrDirError(s.T(), path.Join(explicitDir, ExplicitFileName1))
	operations.ValidateNoFileOrDirError(s.T(), path.Join(explicitDir, ExplicitLocalFileName1))
	// Validate writing content to unlinked local file does not throw error.
	operations.WriteWithoutClose(fh, FileContents, s.T())

	operations.CloseFileShouldThrowStaleHandleError(fh, s.T())

	// Validate both local and synced files are deleted.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testCaseDir, ExplicitDirName, s.T())
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testCaseDir, path.Join(ExplicitDirName, ExplicitFileName1), s.T())
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testCaseDir, path.Join(ExplicitDirName, ExplicitLocalFileName1), s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestStaleFileHandleLocalFileTest(t *testing.T) {
	ts := &staleFileHandleLocalFile{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, ts)
		return
	}

	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--metadata-cache-ttl-secs=0", "--precondition-errors=true"},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
