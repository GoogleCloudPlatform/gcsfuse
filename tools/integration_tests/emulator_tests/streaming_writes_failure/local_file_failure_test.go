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

package streaming_writes_failure

import (
	"context"
	"log"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	emulator_tests "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/emulator_tests/util"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type defaultFailureTestSuite struct {
	suite.Suite
	configPath         string
	flags              []string
	testDirPath        string
	filePath           string
	fh1                *os.File
	storageClient      *storage.Client // Storage Client based on proxy server.
	closeStorageClient func() error
	ctx                context.Context
	data               []byte
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (t *defaultFailureTestSuite) SetupSuite() {
	t.configPath = "../proxy_server/configs/second_chunk_upload_returns412.yaml"
	t.flags = []string{"--enable-streaming-writes=true", "--write-block-size-mb=1", "--write-max-blocks-per-file=1", "--custom-endpoint=" + proxyEndpoint}
	// Generate 5 MB random data.
	data, err := operations.GenerateRandomData(5 * operations.MiB)
	t.data = data
	if err != nil {
		t.T().Fatalf("Error in generating data: %v", err)
	}
	log.Printf("Running tests with flags: %v", t.flags)
}

func (t *defaultFailureTestSuite) SetupTest() {
	// Start proxy server for each test to ensure the config is initialized per test.
	emulator_tests.StartProxyServer(t.configPath)
	// Create storage client before running tests.
	t.ctx = context.Background()
	t.closeStorageClient = client.CreateStorageClientWithCancel(&t.ctx, &t.storageClient)
	setup.MountGCSFuseWithGivenMountFunc(t.flags, mountFunc)
	// Setup test directory for testing.
	t.testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	t.filePath, t.fh1 = CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, FileName1, t.T())
}

func (t *defaultFailureTestSuite) TearDownTest() {
	// CleanUp MntDir before unmounting GCSFuse.
	setup.CleanUpDir(rootDir)
	setup.UnmountGCSFuse(rootDir)
	err := t.closeStorageClient()
	if err != nil {
		log.Fatalf("closeStorageClient failed: %v", err)
	}
	assert.NoError(t.T(), emulator_tests.KillProxyServerProcess(port))
}

func (t *defaultFailureTestSuite) WritingWithNewFileHandleAlsoFails(data []byte, off int64) {
	// Opening a new file handle succeeds.
	fh := operations.OpenFile(t.filePath, t.T())
	// Writes with this file handle fails.
	_, err := fh.WriteAt(data, off)
	assert.Error(t.T(), err)
	// Closing the file handle returns error.
	operations.CloseFileShouldThrowError(fh, t.T())
}

func (t *defaultFailureTestSuite) WritingAfterBwhReinitializationSucceeds() {
	// Opening new file handle and writing to file succeeds.
	fh := operations.CreateFile(t.filePath, FilePerms, t.T())
	_, err := fh.WriteAt(t.data, 0)
	assert.NoError(t.T(), err)
	// Sync succeeds.
	err = fh.Sync()
	assert.NoError(t.T(), err)
	// Close and validate object content found on GCS.
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, fh, testDirName, FileName1, string(t.data), t.T())
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func (t *defaultFailureTestSuite) TestStreamingWritesFailsOnSecondChunkUploadFailure() {
	// Write first 2 MB (say A,B) block to file succeeds but upload of block B will result in error.
	// Fuse:[B] -> Go-SDK:[A] -> GCS[]
	_, err := t.fh1.WriteAt(t.data[:2*operations.MiB], 0)
	assert.NoError(t.T(), err)
	// Write again 2MB (C, D) may or may not fail based on the status of block B upload but it ensures the block B
	// upload attempt is done and the error is propagated.
	// Fuse:[D] -> Go-SDK:[C] -> GCS[A, B -> upload fails]
	_, _ = t.fh1.WriteAt(t.data[2*operations.MiB:4*operations.MiB], 2*operations.MiB)

	// Write 5th 1MB results in errors as it sees the error propagated from block B upload failure.
	_, err = t.fh1.WriteAt(t.data[4*operations.MiB:5*operations.MiB], 4*operations.MiB)

	assert.Error(t.T(), err)
	// Writing from new file handles also fails.
	t.WritingWithNewFileHandleAlsoFails(t.data[4*operations.MiB:5*operations.MiB], 4*operations.MiB)
	// Close file handle to reinitialize bwh.
	operations.CloseFileShouldThrowError(t.fh1, t.T())
	// Verify that Object is not found on GCS.
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, testDirName, FileName1, t.T())
	// Writing after bwh reinitialization succeeds.
	t.WritingAfterBwhReinitializationSucceeds()
}

func (t *defaultFailureTestSuite) TestStreamingWritesTruncateSmallerFailsOnSecondChunkUploadFailure() {
	// Write first 2 MB (say A,B) block to file succeeds but upload of block B will result in error.
	// Fuse:[B] -> Go-SDK:[A] -> GCS[]
	_, err := t.fh1.WriteAt(t.data[:2*operations.MiB], 0)
	assert.NoError(t.T(), err)
	// Write again 2MB (C, D) may or may not fail based on the status of block B upload but it ensures the block B
	// upload attempt is done and the error is propagated.
	// Fuse:[D] -> Go-SDK:[C] -> GCS[A, B -> upload fails]
	_, _ = t.fh1.WriteAt(t.data[2*operations.MiB:4*operations.MiB], 2*operations.MiB)

	// Write 5th 1MB results in errors as it sees the error propagated from block B upload failure.
	_, err = t.fh1.WriteAt(t.data[4*operations.MiB:5*operations.MiB], 4*operations.MiB)

	assert.Error(t.T(), err)
	// Truncate to smaller size fails.
	err = t.fh1.Truncate(1 * operations.MiB)
	assert.Error(t.T(), err)
	// Writing from new file handles also fails.
	t.WritingWithNewFileHandleAlsoFails(t.data[4*operations.MiB:5*operations.MiB], 4*operations.MiB)
	// Close file handle to reinitialize bwh.
	operations.CloseFileShouldThrowError(t.fh1, t.T())
	// Verify that Object is not found on GCS.
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, testDirName, FileName1, t.T())
	// Writing after bwh reinitialization succeeds.
	t.WritingAfterBwhReinitializationSucceeds()
}

func (t *defaultFailureTestSuite) TestStreamingWritesTruncateBiggerSucceedsOnSecondChunkUploadFailure() {
	// Write first 2 MB (say A,B) block to file succeeds but upload of block B will result in error.
	// Fuse:[B] -> Go-SDK:[A] -> GCS[]
	_, err := t.fh1.WriteAt(t.data[:2*operations.MiB], 0)
	assert.NoError(t.T(), err)
	// Write again 2MB (C, D) may or may not fail based on the status of block B upload but it ensures the block B
	// upload attempt is done and the error is propagated.
	// Fuse:[D] -> Go-SDK:[C] -> GCS[A, B -> upload fails]
	_, _ = t.fh1.WriteAt(t.data[2*operations.MiB:4*operations.MiB], 2*operations.MiB)

	// Write 5th 1MB results in errors as it sees the error propagated via B upload failure.
	_, err = t.fh1.WriteAt(t.data[4*operations.MiB:5*operations.MiB], 4*operations.MiB)

	assert.Error(t.T(), err)
	// Opening new file handle succeeds.
	fh2 := operations.OpenFile(t.filePath, t.T())
	// Truncate to bigger size succeeds.
	err = fh2.Truncate(5 * operations.MiB)
	assert.NoError(t.T(), err)
	// Closing all file handles to reinitialize bwh.
	operations.CloseFileShouldThrowError(fh2, t.T())
	operations.CloseFileShouldThrowError(t.fh1, t.T())
	// Verify that Object is not found on GCS.
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, testDirName, FileName1, t.T())
	// Opening new file handle and writing to file succeeds.
	fh3 := operations.CreateFile(t.filePath, FilePerms, t.T())
	_, err = fh3.WriteAt(t.data, 0)
	assert.NoError(t.T(), err)
	// Truncate to bigger succeeds.
	err = fh3.Truncate(6 * operations.MiB)
	assert.NoError(t.T(), err)
	// Close and validate object content found on GCS.
	emptyBytes := make([]byte, operations.MiB)
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, fh3, testDirName, FileName1, string(t.data)+string(emptyBytes), t.T())
}

func (t *defaultFailureTestSuite) TestStreamingWritesSyncFailsOnSecondChunkUploadFailure() {
	// Write first 2 MB (say A,B) block to file succeeds but upload of block B will result in error.
	// Fuse:[B] -> Go-SDK:[A] -> GCS[]
	_, err := t.fh1.WriteAt(t.data[:2*operations.MiB], 0)
	assert.NoError(t.T(), err)
	// Sync file succeeds as the block B is only passed to Go-SDK for upload.
	// Fuse:[] -> Go-SDK:[B]-> GCS[A]
	operations.SyncFile(t.fh1, t.T())

	// Write next 1 MB block C may succeed based on the status of block B.
	// Fuse:[C] -> Go-SDK:[B]-> GCS[A]
	_, _ = t.fh1.WriteAt(t.data[2*operations.MiB:3*operations.MiB], 2*operations.MiB)

	// Sync now reports failure from B block upload.
	// Fuse:[] -> Go-SDK:[C]-> GCS[A, B -> upload fails]
	operations.SyncFileShouldThrowError(t.fh1, t.T())
	// Opening new file handle fails.
	_, err = os.OpenFile(t.filePath, os.O_RDWR, FilePerms)
	assert.Error(t.T(), err)
	// Close file handle to reinitialize bwh.
	operations.CloseFileShouldThrowError(t.fh1, t.T())
	// Verify that Object is not found on GCS.
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, testDirName, FileName1, t.T())
	// Opening new file handle and writing to file succeeds.
	t.WritingAfterBwhReinitializationSucceeds()
}

func (t *defaultFailureTestSuite) TestStreamingWritesCloseFailsOnSecondChunkUploadFailure() {
	// Write first 2 MB (say A,B) block to file succeeds but upload of block B will result in error.
	// Fuse:[B] -> Go-SDK:[A] -> GCS[]
	_, err := t.fh1.WriteAt(t.data[:2*operations.MiB], 0)
	assert.NoError(t.T(), err)

	// Close fails as it sees error from B block upload.
	err = t.fh1.Close()

	assert.NotNil(t.T(), err)
	// Opening new file handle fails.
	_, err = os.OpenFile(t.filePath, os.O_RDWR, FilePerms)
	assert.Error(t.T(), err)
	// Close file handle to reinitialize bwh.
	operations.CloseFileShouldThrowError(t.fh1, t.T())
	// Verify that Object is not found on GCS.
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, testDirName, FileName1, t.T())
	// Opening new file handle and writing to file succeeds.
	t.WritingAfterBwhReinitializationSucceeds()
}

func (t *defaultFailureTestSuite) TestStreamingWritesWhenFinalizeObjectFailure() {
	// Write 1 MB data to file succeeds and upload of block will also succeed.
	_, err := t.fh1.WriteAt(t.data[:operations.MiB], 0)
	assert.NoError(t.T(), err)

	// Close fails as it sees error on the finalize.
	err = t.fh1.Close()

	assert.NotNil(t.T(), err)
	// Verify that Object is not found on GCS.
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, testDirName, FileName1, t.T())
	// Opening new file handle and writing to file succeeds.
	t.WritingAfterBwhReinitializationSucceeds()
}

func TestUploadFailureTestSuite(t *testing.T) {
	suite.Run(t, new(defaultFailureTestSuite))
}
