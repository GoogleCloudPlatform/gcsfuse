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

package stale_handle

import (
	"log"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

const Content = "foobar"
const Content2 = "foobar2"

type staleFileHandleSyncedFile struct {
	flags []string
	suite.Suite
}

func (s *staleFileHandleSyncedFile) SetupSuite() {
	mountGCSFuseAndSetupTestDir(s.flags, ctx, storageClient, testDirName)
}

func (s *staleFileHandleSyncedFile) TearDownSuite() {
	setup.UnmountGCSFuse(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *staleFileHandleSyncedFile) TestSyncedObjectClobberedRemotelyReadThrowsStaleFileHandleError() {
	testCaseDir := "TestSyncedObjectClobberedRemotelyReadThrowsStaleFileHandleError"
	targetDir := path.Join(testDirPath, testCaseDir)
	operations.CreateDirectory(targetDir, s.T())
	// Create an object on bucket
	err := CreateObjectOnGCS(ctx, storageClient, path.Join(testDirName, testCaseDir, FileName1), GCSFileContent)
	assert.NoError(s.T(), err)
	filePath := path.Join(targetDir, FileName1)
	// Open the read handle
	fh, err := operations.OpenFileAsReadonly(filePath)
	assert.NoError(s.T(), err)
	// Replace the underlying object with a new generation.
	err = WriteToObject(ctx, storageClient, path.Join(testDirName, testCaseDir, FileName1), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	buffer := make([]byte, GCSFileSize)
	err = operations.ReadBytesFromFile(fh, GCSFileSize, buffer)

	operations.ValidateStaleNFSFileHandleError(s.T(), err)
	ValidateObjectContentsFromGCS(ctx, storageClient, path.Join(testDirName, testCaseDir), FileName1, FileContents, s.T())
}

func (s *staleFileHandleSyncedFile) TestSyncedObjectClobberedRemotelyFirstWriteThrowsStaleFileHandleError() {
	testCaseDir := "TestSyncedObjectClobberedRemotelyFirstWriteThrowsStaleFileHandleError"
	targetDir := path.Join(testDirPath, testCaseDir)
	operations.CreateDirectory(targetDir, s.T())
	err := CreateObjectOnGCS(ctx, storageClient, path.Join(testDirName, testCaseDir, FileName1), GCSFileContent)
	assert.NoError(s.T(), err)
	filePath := path.Join(targetDir, FileName1)
	fh, err := operations.OpenFileAsWriteOnly(filePath)
	assert.NoError(s.T(), err)
	// Replace the underlying object with a new generation.
	err = WriteToObject(ctx, storageClient, path.Join(testDirName, testCaseDir, FileName1), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	_, err = fh.WriteString(Content)

	operations.ValidateStaleNFSFileHandleError(s.T(), err)
	// Attempt to sync to file should not result in error as we first check if the
	// content has been dirtied before clobbered check in Sync flow.
	operations.SyncFile(fh, s.T())
	// Validate that object is not updated with new content as write failed.
	ValidateObjectContentsFromGCS(ctx, storageClient, path.Join(testDirName, testCaseDir), FileName1, FileContents, s.T())
}

func (s *staleFileHandleSyncedFile) TestSyncedObjectClobberedRemotelySyncAndCloseThrowsStaleFileHandleError() {
	testCaseDir := "TestSyncedObjectClobberedRemotelySyncAndCloseThrowsStaleFileHandleError"
	targetDir := path.Join(testDirPath, testCaseDir)
	operations.CreateDirectory(targetDir, s.T())
	err := CreateObjectOnGCS(ctx, storageClient, path.Join(testDirName, testCaseDir, FileName1), GCSFileContent)
	assert.NoError(s.T(), err)
	filePath := path.Join(targetDir, FileName1)
	fh, err := operations.OpenFileAsWriteOnly(filePath)
	assert.NoError(s.T(), err)
	// Dirty the file by giving it some contents.
	fh.WriteString(Content)
	// Replace the underlying object with a new generation.
	err = WriteToObject(ctx, storageClient, path.Join(testDirName, testCaseDir, FileName1), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	operations.SyncFileShouldThrowStaleHandleError(fh, s.T())
	operations.CloseFileShouldThrowStaleHandleError(fh, s.T())

	// Make fh nil, so that another attempt is not taken in TearDown to close the
	// file.
	fh = nil
	// Validate that object is not updated with un-synced content.
	ValidateObjectContentsFromGCS(ctx, storageClient, path.Join(testDirName, testCaseDir), FileName1, FileContents, s.T())
}

func (s *staleFileHandleSyncedFile) TestSyncedObjectDeletedRemotelySyncAndCloseThrowsStaleFileHandleError() {
	testCaseDir := "TestSyncedObjectDeletedRemotelySyncAndCloseThrowsStaleFileHandleError"
	targetDir := path.Join(testDirPath, testCaseDir)
	operations.CreateDirectory(targetDir, s.T())
	err := CreateObjectOnGCS(ctx, storageClient, path.Join(testDirName, testCaseDir, FileName1), GCSFileContent)
	assert.NoError(s.T(), err)
	filePath := path.Join(targetDir, FileName1)
	fh, err := operations.OpenFileAsWriteOnly(filePath)
	assert.NoError(s.T(), err)
	// Dirty the file by giving it some contents.
	fh.WriteString(Content)
	// Delete the object remotely.
	err = DeleteObjectOnGCS(ctx, storageClient, path.Join(testDirName, testCaseDir, FileName1))
	assert.NoError(s.T(), err)
	// Attempt to write to file should not give any error.
	fh.WriteString(Content2)

	operations.SyncFileShouldThrowStaleHandleError(fh, s.T())
	operations.CloseFileShouldThrowStaleHandleError(fh, s.T())

	// Make fh nil, so that another attempt is not taken in TearDown to close the
	// file.
	fh = nil
}

func (s *staleFileHandleSyncedFile) TestSyncedObjectDeletedLocallySyncAndCloseThrowsStaleFileHandleError() {
	testCaseDir := "TestSyncedObjectDeletedLocallySyncAndCloseThrowsStaleFileHandleError"
	targetDir := path.Join(testDirPath, testCaseDir)
	operations.CreateDirectory(targetDir, s.T())
	err := CreateObjectOnGCS(ctx, storageClient, path.Join(testDirName, testCaseDir, FileName1), GCSFileContent)
	assert.NoError(s.T(), err)
	filePath := path.Join(targetDir, FileName1)
	fh, err := operations.OpenFileAsWriteOnly(filePath)
	assert.NoError(s.T(), err)
	// Dirty the file by giving it some contents.
	fh.WriteString(Content)
	// Delete the object locally.
	operations.RemoveFile(filePath)
	// Attempt to write to file should not give any error.
	fh.WriteString(Content2)

	operations.SyncFileShouldThrowStaleHandleError(fh, s.T())
	operations.CloseFileShouldThrowStaleHandleError(fh, s.T())

	// Make fh nil, so that another attempt is not taken in TearDown to close the
	// file.
	fh = nil
}

func (s *staleFileHandleSyncedFile) TestRenamedSyncedObjectSyncAndCloseThrowsStaleFileHandleError() {
	testCaseDir := "TestRenamedSyncedObjectSyncAndCloseThrowsStaleFileHandleError"
	targetDir := path.Join(testDirPath, testCaseDir)
	operations.CreateDirectory(targetDir, s.T())
	err := CreateObjectOnGCS(ctx, storageClient, path.Join(testDirName, testCaseDir, FileName1), GCSFileContent)
	assert.NoError(s.T(), err)
	filePath := path.Join(targetDir, FileName1)
	newFilePath := path.Join(targetDir, FileName2)
	fh, err := operations.OpenFileAsWriteOnly(filePath)
	assert.NoError(s.T(), err)
	// Dirty the file by giving it some contents.
	fh.WriteString(Content)
	operations.RenameFile(filePath, newFilePath)
	// Attempt to write to file should not give any error.
	fh.WriteString(Content2)

	operations.SyncFileShouldThrowStaleHandleError(fh, s.T())
	operations.CloseFileShouldThrowStaleHandleError(fh, s.T())

	// Make fh nil, so that another attempt is not taken in TearDown to close the
	// file.
	fh = nil
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestStaleFileHandleSyncedFileTest(t *testing.T) {
	ts := &staleFileHandleSyncedFile{}

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
