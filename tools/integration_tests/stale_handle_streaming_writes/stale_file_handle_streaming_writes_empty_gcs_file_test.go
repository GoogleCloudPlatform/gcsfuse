// Copyright 2025 Google LLC
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

package stale_handle_streaming_writes

import (
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

type staleFileHandleStreamingWritesEmptyGcsFile struct {
	staleFileHandleStreamingWritesCommon
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (t *staleFileHandleStreamingWritesEmptyGcsFile) SetupTest() {
	t.fileName = setup.GenerateRandomString(5)
	// Create an empty object on GCS
	CreateObjectInGCSTestDir(ctx, storageClient, testDirName, t.fileName, "", t.T())
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, t.fileName, "", t.T())
	t.filePath = path.Join(t.testDirPath, t.fileName)
	t.f1 = operations.OpenFile(t.filePath, t.T())
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *staleFileHandleStreamingWritesEmptyGcsFile) TestFirstWriteToClobberedFileThrowsStaleFileHandleError() {
	// Clobber file by replacing the underlying object with a new generation.
	err := WriteToObject(ctx, storageClient, path.Join(testDirName, t.fileName), FileContents, storage.Conditions{})
	assert.NoError(t.T(), err)

	// Attempt first write to the file.
	_, err = t.f1.WriteAt([]byte(t.data), 0)

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	operations.SyncFile(t.f1, t.T())
	operations.CloseFileShouldNotThrowError(t.f1, t.T())
}

func (t *staleFileHandleStreamingWritesEmptyGcsFile) TestWriteOnRenamedFileThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	n, err := t.f1.WriteAt([]byte(t.data), 0)
	assert.NoError(t.T(), err)
	newFile := "new" + t.fileName
	err = operations.RenameFile(t.filePath, path.Join(t.testDirPath, newFile))
	assert.NoError(t.T(), err)

	// Attempt to write to file should give stale NFS file handle erorr.
	_, err = t.f1.WriteAt([]byte(t.data), int64(n))

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Sync and Close succeeds.
	operations.SyncFile(t.f1, t.T())
	operations.CloseFileShouldNotThrowError(t.f1, t.T())
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, newFile, t.data, t.T())
}

func (t *staleFileHandleStreamingWritesEmptyGcsFile) TestFileDeletedRemotelyWriteAndSyncDoNoThrowStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	n, err := t.f1.WriteAt([]byte(t.data), 0)
	assert.NoError(t.T(), err)
	// Delete the file remotely.
	err = DeleteObjectOnGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
	assert.NoError(t.T(), err)
	// Verify unlink operation succeeds.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, t.fileName, t.T())
	_, err = t.f1.WriteAt([]byte(t.data), int64(n))
	assert.NoError(t.T(), err)
	operations.SyncFile(t.f1, t.T())

	// Closing the file/writer returns stale NFS file handle error.
	err = t.f1.Close()

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, t.fileName, t.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestStaleFileHandleStreamingWritesEmptyGcsFileTest(t *testing.T) {
	suite.Run(t, new(staleFileHandleStreamingWritesEmptyGcsFile))
}
