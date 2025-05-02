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
	s.testDirPath = setup.SetupTestDirectory(s.T().Name())
	// Create an empty object on GCS.
	err := CreateObjectOnGCS(ctx, storageClient, path.Join(s.T().Name(), FileName1), "")
	assert.NoError(s.T(), err)
	s.f1 = operations.OpenFileWithODirect(s.T(), path.Join(s.testDirPath, FileName1))
	s.data = setup.GenerateRandomString(operations.MiB * 5)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *staleFileHandleEmptyGcsFile) TestClobberedFileReadThrowsStaleFileHandleError() {
	if streamingWrites && setup.IsZonalBucketRun() {
		s.T().Skip("Skip test due to takeover support not available.")
	}
	// Dirty the file by giving it some contents.
	_, err := s.f1.WriteAt([]byte(s.data), 0)
	assert.NoError(s.T(), err)
	operations.SyncFile(s.f1, s.T())
	// Replace the underlying object with a new generation.
	err = WriteToObject(ctx, storageClient, path.Join(s.T().Name(), FileName1), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	operations.ValidateReadGivenThatFileIsClobbered(s.T(), s.f1, streamingWrites, s.data)
}

func (s *staleFileHandleEmptyGcsFile) TestClobberedFileFirstWriteThrowsStaleFileHandleError() {
	if streamingWrites && setup.IsZonalBucketRun() {
		s.T().Skip("Skip test due to takeover support not available.")
	}
	// Clobber file by replacing the underlying object with a new generation.
	err := WriteToObject(ctx, storageClient, path.Join(s.T().Name(), FileName1), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	// Attempt first write to the file should give stale NFS file handle error.
	_, err = s.f1.Write([]byte(s.data))

	assert.NoError(s.T(), err)
	operations.ValidateSyncGivenThatFileIsClobbered(s.T(), s.f1, streamingWrites)
	err = s.f1.Close()
	operations.ValidateESTALEError(s.T(), err)
}

func (s *staleFileHandleEmptyGcsFile) TestFileDeletedRemotelySyncAndCloseThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	operations.WriteWithoutClose(s.f1, s.data, s.T())
	// Delete the file remotely.
	err := DeleteObjectOnGCS(ctx, storageClient, path.Join(s.T().Name(), FileName1))
	assert.NoError(s.T(), err)
	// Verify unlink operation succeeds.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, s.T().Name(), FileName1, s.T())
	// Attempt to write to file should not give any error.
	operations.WriteWithoutClose(s.f1, s.data, s.T())

	operations.ValidateSyncGivenThatFileIsClobbered(s.T(), s.f1, streamingWrites)

	err = s.f1.Close()
	operations.ValidateESTALEError(s.T(), err)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestStaleFileHandleEmptyGcsFileTest(t *testing.T) {
	suite.Run(t, new(staleFileHandleEmptyGcsFile))
}
