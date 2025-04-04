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
	"os"
	"path"
	"syscall"
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

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (s *staleFileHandleEmptyGcsFile) SetupTest() {
	// Create an empty object on GCS.
	err := CreateObjectOnGCS(ctx, storageClient, path.Join(s.testDirName, FileName1), "")
	assert.NoError(s.T(), err)
	s.f1, err = os.OpenFile(path.Join(s.testDirPath, FileName1), os.O_RDWR|syscall.O_DIRECT, operations.FilePermission_0600)
	assert.NoError(s.T(), err)
	s.data = setup.GenerateRandomString(operations.MiB * 5)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *staleFileHandleEmptyGcsFile) TestClobberedFileReadThrowsStaleFileHandleError() {
	if s.streamingWritesEnabled() {
		s.T().Skip("Skipping test as reads aren't supported with streaming writes.")
	}
	// Dirty the file by giving it some contents.
	_, err := s.f1.WriteAt([]byte(s.data), 0)
	operations.SyncFile(s.f1, s.T())
	assert.NoError(s.T(), err)
	// Replace the underlying object with a new generation.
	err = WriteToObject(ctx, storageClient, path.Join(s.testDirName, FileName1), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	buffer := make([]byte, len(s.data))
	_, err = s.f1.Read(buffer)

	operations.ValidateStaleNFSFileHandleError(s.T(), err)
}

func (s *staleFileHandleEmptyGcsFile) TestClobberedFileFirstWriteThrowsStaleFileHandleError() {
	// Clobber file by replacing the underlying object with a new generation.
	err := WriteToObject(ctx, storageClient, path.Join(s.testDirName, FileName1), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	// Attempt first write to the file should give stale NFS file handle error.
	_, err = s.f1.WriteAt([]byte(s.data), 0)

	operations.ValidateStaleNFSFileHandleError(s.T(), err)
	// Attempt to sync to file should not result in error as we first check if the
	// content has been dirtied before clobbered check in Sync flow.
	operations.SyncFile(s.f1, s.T())
	operations.CloseFileShouldNotThrowError(s.f1, s.T())
}

func (s *staleFileHandleEmptyGcsFile) TestRenamedFileSyncAndCloseThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	n, err := s.f1.WriteAt([]byte(s.data), 0)
	assert.NoError(s.T(), err)
	newFile := "new" + FileName1
	err = operations.RenameFile(s.f1.Name(), path.Join(s.testDirPath, newFile))
	assert.NoError(s.T(), err)

	// Attempt to write to file should give error iff streaming writes are enabled.
	_, err = s.f1.WriteAt([]byte(s.data), int64(n))

	if s.streamingWritesEnabled() {
		operations.ValidateStaleNFSFileHandleError(s.T(), err)
	} else {
		assert.NoError(s.T(), err)
	}
	err = s.f1.Sync()
	s.validateStaleNFSFileHandleErrorIfStreamingWritesDisabled(err)
	err = s.f1.Close()
	s.validateStaleNFSFileHandleErrorIfStreamingWritesDisabled(err)
}

func (s *staleFileHandleEmptyGcsFile) TestFileDeletedRemotelySyncAndCloseThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	n, err := s.f1.WriteAt([]byte(s.data), 0)
	assert.NoError(s.T(), err)
	// Delete the file remotely.
	err = DeleteObjectOnGCS(ctx, storageClient, path.Join(s.testDirName, FileName1))
	assert.NoError(s.T(), err)
	// Verify unlink operation succeeds.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, s.testDirName, FileName1, s.T())
	// Attempt to write to file should not give any error.
	_, err = s.f1.WriteAt([]byte(s.data), int64(n))
	assert.NoError(s.T(), err)

	err = s.f1.Sync()

	s.validateStaleNFSFileHandleErrorIfStreamingWritesDisabled(err)
	err = s.f1.Close()
	operations.ValidateStaleNFSFileHandleError(s.T(), err)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, s.testDirName, FileName1, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestStaleFileHandleEmptyGcsFileTest(t *testing.T) {
	t.Parallel()
	for scenario, flags := range flagsSetMap {
		ts := new(staleFileHandleEmptyGcsFile)
		ts.flags = flags
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			suite.Run(t, ts)
		})
	}
}
