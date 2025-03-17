// Copyright 2025 Google LLC
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

package stale_handle_streaming_writes

import (
	"os"
	"path"

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

type staleFileHandleStreamingWritesCommon struct {
	f1          *os.File
	data        string
	testDirPath string
	fileName    string
	filePath    string
	suite.Suite
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (t *staleFileHandleStreamingWritesCommon) SetupSuite() {
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	t.testDirPath = setup.SetupTestDirectory(testDirName)
	t.data = setup.GenerateRandomString(operations.MiB * 5)
}

func (t *staleFileHandleStreamingWritesCommon) TearDownSuite() {
	setup.UnmountGCSFuse(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *staleFileHandleStreamingWritesCommon) TestFileDeletedLocallySyncAndCloseDoNotThrowError() {
	// Dirty the file by giving it some contents.
	bytesWrote, err := t.f1.WriteAt([]byte(t.data), 0)
	assert.NoError(t.T(), err)
	// Delete the file.
	operations.RemoveFile(t.f1.Name())
	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(t.T(), t.f1.Name())

	// Attempt to write to file should give stale NFS file handle erorr.
	_, err = t.f1.WriteAt([]byte(t.data), int64(bytesWrote))

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	operations.SyncFile(t.f1, t.T())
	operations.CloseFileShouldNotThrowError(t.f1, t.T())
}

func (t *staleFileHandleStreamingWritesCommon) TestClosingFileHandleForClobberedFileReturnsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	_, err := t.f1.WriteAt([]byte(t.data), 0)
	assert.NoError(t.T(), err)
	err = t.f1.Sync()
	assert.NoError(t.T(), err)
	// Replace the underlying object with a new generation.
	err = WriteToObject(ctx, storageClient, path.Join(testDirName, t.fileName), FileContents, storage.Conditions{})
	assert.NoError(t.T(), err)

	// Closing the file/writer returns stale NFS file handle error.
	err = t.f1.Close()

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, t.fileName, FileContents, t.T())
}
