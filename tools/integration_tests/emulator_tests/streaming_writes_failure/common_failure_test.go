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

	"cloud.google.com/go/storage"
	emulator_tests "github.com/vipnydav/gcsfuse/v3/tools/integration_tests/emulator_tests/util"
	. "github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type commonFailureTestSuite struct {
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
	port               int
	proxyProcessId     int
	proxyServerLogFile string
	gcsObjectValidator
}

type gcsObjectValidator interface {
	// Validate file from GCS for empty gcs file and new local files.
	validateGcsObject()
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (t *commonFailureTestSuite) SetupSuite() {
	t.flags = []string{"--write-block-size-mb=1", "--write-max-blocks-per-file=1"}
	// Generate 5 MB random data.
	var err error
	t.data, err = operations.GenerateRandomData(5 * operations.MiB)
	require.NoError(t.T(), err)
	log.Printf("Running tests with flags: %v", t.flags)
}

func (t *commonFailureTestSuite) setupTest() {
	t.T().Helper()
	// Start proxy server for each test to ensure the config is initialized per test.
	t.proxyServerLogFile = setup.CreateProxyServerLogFile(t.T())
	var err error
	t.port, t.proxyProcessId, err = emulator_tests.StartProxyServer(t.configPath, t.proxyServerLogFile)
	require.NoError(t.T(), err)
	setup.AppendProxyEndpointToFlagSet(&t.flags, t.port)
	// Create storage client before running tests.
	t.ctx = context.Background()
	t.closeStorageClient = CreateStorageClientWithCancel(&t.ctx, &t.storageClient)
	setup.MountGCSFuseWithGivenMountFunc(t.flags, mountFunc)
	// Setup random testDirName.
	testDirName = testDirNamePrefix + setup.GenerateRandomString(5)
	// Setup test directory for testing.
	t.testDirPath = setup.SetupTestDirectory(testDirName)
}

func (t *commonFailureTestSuite) TearDownTest() {
	setup.UnmountGCSFuse(rootDir)
	assert.NoError(t.T(), t.closeStorageClient())
	assert.NoError(t.T(), emulator_tests.KillProxyServerProcess(t.proxyProcessId))
	setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
	setup.SaveProxyServerLogFileInCaseOfFailure(t.proxyServerLogFile, t.T())
}

func (t *commonFailureTestSuite) writingWithNewFileHandleAlsoFails(data []byte, off int64) {
	t.T().Helper()
	// Opening a new file handle succeeds.
	fh := operations.OpenFile(t.filePath, t.T())
	// Writes with this file handle fails.
	_, err := fh.WriteAt(data, off)
	assert.Error(t.T(), err)
	// Closing the file handle returns error.
	operations.CloseFileShouldThrowError(t.T(), fh)
}

func (t *commonFailureTestSuite) writingAfterBwhReinitializationSucceeds() {
	t.T().Helper()
	// Verify that expectation from GCS matches.
	t.validateGcsObject()
	// Opening new file handle and writing to file succeeds.
	t.fh1 = operations.CreateFile(t.filePath, FilePerms, t.T())
	_, err := t.fh1.WriteAt(t.data, 0)
	assert.NoError(t.T(), err)
	// Sync succeeds.
	err = t.fh1.Sync()
	assert.NoError(t.T(), err)
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func (t *commonFailureTestSuite) TestStreamingWritesFailsOnSecondChunkUploadFailure() {
	// Write first 2 MB (say A,B) block to file succeeds but async upload of block B will result in error.
	// Fuse:[B] -> Go-SDK:[A] -> GCS[]
	_, err := t.fh1.WriteAt(t.data[:2*operations.MiB], 0)
	assert.NoError(t.T(), err)
	// Write again 2MB (C, D) will trigger B upload.
	// Fuse:[D] -> Go-SDK:[C] -> GCS[A, B -> upload fails]
	_, _ = t.fh1.WriteAt(t.data[2*operations.MiB:4*operations.MiB], 2*operations.MiB)

	// Write 5th 1MB results in errors.
	_, err = t.fh1.WriteAt(t.data[4*operations.MiB:5*operations.MiB], 4*operations.MiB)

	require.Error(t.T(), err)
	t.writingWithNewFileHandleAlsoFails(t.data[4*operations.MiB:5*operations.MiB], 4*operations.MiB)
	// Close file handle to reinitialize bwh.
	operations.CloseFileShouldThrowError(t.T(), t.fh1)
	// Opening new file handle and writing to file succeeds.
	t.writingAfterBwhReinitializationSucceeds()
	// Close and validate object content found on GCS.
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, t.fh1, testDirName, FileName1, string(t.data), t.T())
}

func (t *commonFailureTestSuite) TestStreamingWritesTruncateSmallerFailsOnSecondChunkUploadFailure() {
	// Write first 2 MB (say A,B) block to file succeeds but async upload of block B will result in error.
	// Fuse:[B] -> Go-SDK:[A] -> GCS[]
	_, err := t.fh1.WriteAt(t.data[:2*operations.MiB], 0)
	assert.NoError(t.T(), err)
	// Write again 2MB (C, D) will trigger B upload.
	// Fuse:[D] -> Go-SDK:[C] -> GCS[A, B -> upload fails]
	_, _ = t.fh1.WriteAt(t.data[2*operations.MiB:4*operations.MiB], 2*operations.MiB)

	// Write 5th 1MB results in errors.
	_, err = t.fh1.WriteAt(t.data[4*operations.MiB:5*operations.MiB], 4*operations.MiB)

	require.Error(t.T(), err)
	// Truncate to smaller size fails.
	err = t.fh1.Truncate(1 * operations.MiB)
	assert.Error(t.T(), err)
	t.writingWithNewFileHandleAlsoFails(t.data[4*operations.MiB:5*operations.MiB], 4*operations.MiB)
	// Close file handle to reinitialize bwh.
	operations.CloseFileShouldThrowError(t.T(), t.fh1)
	// Opening new file handle and writing to file succeeds.
	t.writingAfterBwhReinitializationSucceeds()
	// Close and validate object content found on GCS.
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, t.fh1, testDirName, FileName1, string(t.data), t.T())
}

func (t *commonFailureTestSuite) TestStreamingWritesTruncateBiggerSucceedsOnSecondChunkUploadFailure() {
	// Write first 2 MB (say A,B) block to file succeeds but async upload of block B will result in error.
	// Fuse:[B] -> Go-SDK:[A] -> GCS[]
	_, err := t.fh1.WriteAt(t.data[:2*operations.MiB], 0)
	assert.NoError(t.T(), err)
	// Write again 2MB (C, D) will trigger B upload.
	// Fuse:[D] -> Go-SDK:[C] -> GCS[A, B -> upload fails]
	_, _ = t.fh1.WriteAt(t.data[2*operations.MiB:4*operations.MiB], 2*operations.MiB)

	// Write 5th 1MB results in errors.
	_, err = t.fh1.WriteAt(t.data[4*operations.MiB:5*operations.MiB], 4*operations.MiB)

	require.Error(t.T(), err)
	// Opening new file handle succeeds.
	fh2 := operations.OpenFile(t.filePath, t.T())
	// Truncate to bigger size succeeds.
	err = fh2.Truncate(5 * operations.MiB)
	assert.NoError(t.T(), err)
	// Closing all file handles to reinitialize bwh.
	operations.CloseFileShouldThrowError(t.T(), fh2)
	operations.CloseFileShouldThrowError(t.T(), t.fh1)
	// Opening new file handle and writing to file succeeds.
	t.writingAfterBwhReinitializationSucceeds()
	// Truncate to bigger size succeeds.
	err = t.fh1.Truncate(6 * operations.MiB)
	assert.NoError(t.T(), err)
	// Close and validate object content found on GCS.
	emptyBytes := make([]byte, operations.MiB)
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, t.fh1, testDirName, FileName1, string(t.data)+string(emptyBytes), t.T())
}

func (t *commonFailureTestSuite) TestStreamingWritesSyncFailsOnSecondChunkUploadFailure() {
	// Write first 2 MB (say A,B) block to file succeeds but async upload of block B will result in error.
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
	operations.SyncFileShouldThrowError(t.T(), t.fh1)
	// Close file handle to reinitialize bwh.
	operations.CloseFileShouldThrowError(t.T(), t.fh1)
	// Opening new file handle and writing to file succeeds.
	t.writingAfterBwhReinitializationSucceeds()
	// Close and validate object content found on GCS.
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, t.fh1, testDirName, FileName1, string(t.data), t.T())
}

func (t *commonFailureTestSuite) TestStreamingWritesCloseFailsOnSecondChunkUploadFailure() {
	// Write first 2 MB (say A,B) block to file succeeds but async upload of block B will result in error.
	// Fuse:[B] -> Go-SDK:[A] -> GCS[]
	_, err := t.fh1.WriteAt(t.data[:2*operations.MiB], 0)
	assert.NoError(t.T(), err)

	// Close fails as it sees error from B block upload.
	err = t.fh1.Close()

	require.Error(t.T(), err)
	// Opening new file handle and writing to file succeeds.
	t.writingAfterBwhReinitializationSucceeds()
	// Close and validate object content found on GCS.
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, t.fh1, testDirName, FileName1, string(t.data), t.T())
}

func (t *commonFailureTestSuite) TestStreamingWritesWhenFinalizeObjectFailure() {
	// Write 1 MB data to file succeeds and async upload of block will also succeed.
	_, err := t.fh1.WriteAt(t.data[:operations.MiB], 0)
	assert.NoError(t.T(), err)

	// Close fails as it sees error on the finalize.
	err = t.fh1.Close()

	require.Error(t.T(), err)
	// Opening new file handle and writing to file succeeds.
	t.writingAfterBwhReinitializationSucceeds()
	// Close and validate object content found on GCS.
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, t.fh1, testDirName, FileName1, string(t.data), t.T())
}

func (t *commonFailureTestSuite) TestStreamingWritesBwhResetsWhenFileHandlesAreOpenInReadMode() {
	// Write first 2 MB (say A,B) block to file succeeds but async upload of block B will result in error.
	// Fuse:[B] -> Go-SDK:[A] -> GCS[]
	_, err := t.fh1.WriteAt(t.data[:2*operations.MiB], 0)
	assert.NoError(t.T(), err)
	// Write again 2MB (C, D) will trigger B upload.
	// Fuse:[D] -> Go-SDK:[C] -> GCS[A, B -> upload fails]
	_, _ = t.fh1.WriteAt(t.data[2*operations.MiB:4*operations.MiB], 2*operations.MiB)

	// Write 5th 1MB results in errors.
	_, err = t.fh1.WriteAt(t.data[4*operations.MiB:5*operations.MiB], 4*operations.MiB)

	require.Error(t.T(), err)
	fh2, err := operations.OpenFileAsReadonly(t.filePath)
	assert.NoError(t.T(), err)
	// Closing only file handle in write mode.
	operations.CloseFileShouldThrowError(t.T(), t.fh1)
	// Opening new file handle and writing to file succeeds when file handles in O_RDONLY mode are open.
	t.writingAfterBwhReinitializationSucceeds()
	operations.CloseFileShouldNotThrowError(t.T(), fh2)
	// Close and validate object content found on GCS.
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, t.fh1, testDirName, FileName1, string(t.data), t.T())
}
