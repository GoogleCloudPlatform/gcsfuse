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

package stale_handle

import (
	"log"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////
type staleFileHandleLocalFile struct {
	staleFileHandleCommon
}

type staleFileHandleEmptyGcsFile struct {
	staleFileHandleCommon
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (s *staleFileHandleLocalFile) SetupTest() {
	// Create a local file.
	s.fileName = path.Base(s.T().Name()) + setup.GenerateRandomString(5)
	s.f1 = operations.OpenFileWithODirect(s.T(), path.Join(testEnv.testDirPath, s.fileName))
	s.isLocal = true
}
func (s *staleFileHandleLocalFile) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *staleFileHandleEmptyGcsFile) SetupTest() {
	s.fileName = path.Base(s.T().Name()) + setup.GenerateRandomString(5)
	err := CreateObjectOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testDirName, s.fileName), "")
	assert.NoError(s.T(), err)
	s.f1 = operations.OpenFileWithODirect(s.T(), path.Join(testEnv.testDirPath, s.fileName))
}
func (s *staleFileHandleEmptyGcsFile) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *staleFileHandleEmptyGcsFile) TestClobberedFileReadThrowsStaleFileHandleError() {
	// TODO(b/410698332): Remove skip condition once takeover support is available.
	if s.isStreamingWritesEnabled && setup.IsZonalBucketRun() {
		s.T().Skip("Skip test due to takeover support not available.")
	}
	// Dirty the file by giving it some contents.
	_, err := s.f1.WriteAt([]byte(s.data), 0)
	assert.NoError(s.T(), err)
	operations.SyncFile(s.f1, s.T())

	// Replace the underlying object with a new generation.
	err = WriteToObject(testEnv.ctx, testEnv.storageClient, path.Join(testDirName, s.fileName), FileContents, storage.Conditions{})

	assert.NoError(s.T(), err)
	buffer := make([]byte, len(s.data))
	_, err = s.f1.Read(buffer)
	operations.ValidateESTALEError(s.T(), err)
}

func (s *staleFileHandleEmptyGcsFile) TestClobberedFileFirstWriteThrowsStaleFileHandleError() {
	// TODO(b/410698332): Remove skip condition once takeover support is available.
	if s.isStreamingWritesEnabled && setup.IsZonalBucketRun() {
		s.T().Skip("Skip test due to takeover support not available.")
	}
	// Clobber file by replacing the underlying object with a new generation.
	err := WriteToObject(testEnv.ctx, testEnv.storageClient, path.Join(testDirName, s.fileName), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	// Attempt first write to the file should give stale NFS file handle error.
	_, err = s.f1.Write([]byte(s.data))

	assert.NoError(s.T(), err)
	operations.ValidateSyncGivenThatFileIsClobbered(s.T(), s.f1, s.isStreamingWritesEnabled)
	err = s.f1.Close()
	operations.ValidateESTALEError(s.T(), err)
	ValidateObjectContentsFromGCS(testEnv.ctx, testEnv.storageClient, testDirName, s.fileName, FileContents, s.T())
}

func (s *staleFileHandleEmptyGcsFile) TestFileDeletedRemotelySyncAndCloseThrowsStaleFileHandleError() {
	// TODO(mohitkyadav): Enable test once fix in b/415713332 is released
	if s.isStreamingWritesEnabled && setup.IsZonalBucketRun() {
		s.T().Skip("Skip test due to bug (b/415713332) in client.")
	}
	// Dirty the file by giving it some contents.
	operations.WriteWithoutClose(s.f1, s.data, s.T())
	// Delete the file remotely.
	err := DeleteObjectOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testDirName, s.fileName))
	assert.NoError(s.T(), err)
	// Verify unlink operation succeeds.
	ValidateObjectNotFoundErrOnGCS(testEnv.ctx, testEnv.storageClient, testDirName, s.fileName, s.T())
	// Attempt to write to file should not give any error.
	operations.WriteWithoutClose(s.f1, s.data, s.T())

	operations.ValidateSyncGivenThatFileIsClobbered(s.T(), s.f1, s.isStreamingWritesEnabled)

	err = s.f1.Close()
	operations.ValidateESTALEError(s.T(), err)
	ValidateObjectNotFoundErrOnGCS(testEnv.ctx, testEnv.storageClient, testDirName, s.fileName, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestStaleHandleStreamingWritesEnabled(t *testing.T) {
	// Run tests for mounted directory if the flag is set and return.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		// Run tests for local file.
		suite.Run(t, &staleFileHandleLocalFile{staleFileHandleCommon{isStreamingWritesEnabled: true}})

		// Run tests for empty gcs file.
		suite.Run(t, &staleFileHandleEmptyGcsFile{staleFileHandleCommon{isStreamingWritesEnabled: true}})

		return
	}

	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		// Run local file tests
		sLocal := new(staleFileHandleLocalFile)
		sLocal.flags = flags
		log.Printf("Running local file tests with flags: %s", sLocal.flags)
		sLocal.isStreamingWritesEnabled = true
		suite.Run(t, sLocal)

		// Run empty GCS file tests
		sEmptyGCS := new(staleFileHandleEmptyGcsFile)
		sEmptyGCS.flags = flags
		log.Printf("Running empty GCS file tests with flags: %s", sEmptyGCS.flags)
		sEmptyGCS.isStreamingWritesEnabled = true
		suite.Run(t, sEmptyGCS)
	}
}

func TestStaleHandleStreamingWritesDisabled(t *testing.T) {
	// Run tests for mounted directory if the flag is set and return.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		// Run tests for local file.
		suite.Run(t, &staleFileHandleLocalFile{staleFileHandleCommon{isStreamingWritesEnabled: false}})

		// Run tests for empty gcs file.
		suite.Run(t, &staleFileHandleEmptyGcsFile{staleFileHandleCommon{isStreamingWritesEnabled: false}})

		return
	}

	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		// Run local file tests
		sLocal := new(staleFileHandleLocalFile)
		sLocal.flags = flags
		log.Printf("Running local file tests with flags: %s", sLocal.flags)
		sLocal.isStreamingWritesEnabled = false
		suite.Run(t, sLocal)

		// Run empty GCS file tests
		sEmptyGCS := new(staleFileHandleEmptyGcsFile)
		sEmptyGCS.flags = flags
		log.Printf("Running empty GCS file tests with flags: %s", sEmptyGCS.flags)
		sEmptyGCS.isStreamingWritesEnabled = false
		suite.Run(t, sEmptyGCS)
	}
}
