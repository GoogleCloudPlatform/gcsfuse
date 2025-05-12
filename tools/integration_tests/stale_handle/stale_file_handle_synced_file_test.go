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
	"path"
	"slices"
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

type staleFileHandleEmptyGcsFile struct {
	staleFileHandleCommon
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (s *staleFileHandleEmptyGcsFile) SetupTest() {
	// Create an empty object on GCS.
	s.fileName = path.Base(s.T().Name()) + setup.GenerateRandomString(5)
	err := CreateObjectOnGCS(ctx, storageClient, path.Join(testDirName, s.fileName), "")
	assert.NoError(s.T(), err)
	s.f1 = operations.OpenFileWithODirect(s.T(), path.Join(s.testDirPath, s.fileName))
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
	err = WriteToObject(ctx, storageClient, path.Join(testDirName, s.fileName), FileContents, storage.Conditions{})

	assert.NoError(s.T(), err)
	operations.ValidateReadGivenThatFileIsClobbered(s.T(), s.f1, s.isStreamingWritesEnabled, s.data)
}

func (s *staleFileHandleEmptyGcsFile) TestClobberedFileFirstWriteThrowsStaleFileHandleError() {
	// TODO(b/410698332): Remove skip condition once takeover support is available.
	if s.isStreamingWritesEnabled && setup.IsZonalBucketRun() {
		s.T().Skip("Skip test due to takeover support not available.")
	}
	// Clobber file by replacing the underlying object with a new generation.
	err := WriteToObject(ctx, storageClient, path.Join(testDirName, s.fileName), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	// Attempt first write to the file should give stale NFS file handle error.
	_, err = s.f1.Write([]byte(s.data))

	assert.NoError(s.T(), err)
	operations.ValidateSyncGivenThatFileIsClobbered(s.T(), s.f1, s.isStreamingWritesEnabled)
	err = s.f1.Close()
	operations.ValidateESTALEError(s.T(), err)
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, s.fileName, FileContents, s.T())
}

func (s *staleFileHandleEmptyGcsFile) TestFileDeletedRemotelySyncAndCloseThrowsStaleFileHandleError() {
	// TODO(mohitkyadav): Enable test once fix in b/415713332 is released
	if s.isStreamingWritesEnabled && setup.IsZonalBucketRun() {
		s.T().Skip("Skip test due to bug (b/415713332) in client.")
	}
	// Dirty the file by giving it some contents.
	operations.WriteWithoutClose(s.f1, s.data, s.T())
	// Delete the file remotely.
	err := DeleteObjectOnGCS(ctx, storageClient, path.Join(testDirName, s.fileName))
	assert.NoError(s.T(), err)
	// Verify unlink operation succeeds.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, s.fileName, s.T())
	// Attempt to write to file should not give any error.
	operations.WriteWithoutClose(s.f1, s.data, s.T())

	operations.ValidateSyncGivenThatFileIsClobbered(s.T(), s.f1, s.isStreamingWritesEnabled)

	err = s.f1.Close()
	operations.ValidateESTALEError(s.T(), err)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, s.fileName, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestStaleFileHandleEmptyGcsFileTest(t *testing.T) {
	// Run tests for mounted directory if the flag is set and return.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, new(staleFileHandleEmptyGcsFile))
		return
	}
	for _, flags := range flagsSet {
		s := new(staleFileHandleEmptyGcsFile)
		s.flags = flags
		s.isStreamingWritesEnabled = slices.Contains(s.flags, "--enable-streaming-writes=true")
		suite.Run(t, s)
	}
}
