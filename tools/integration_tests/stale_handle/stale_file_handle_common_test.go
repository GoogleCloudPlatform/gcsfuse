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

package stale_handle

import (
	"os"
	"path"
	"slices"

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

type staleFileHandleCommon struct {
	f1          *os.File
	data        string
	testDirPath string
	suite.Suite
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (s *staleFileHandleCommon) SetupSuite() {
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
}

func (s *staleFileHandleCommon) TearDownSuite() {
	setup.UnmountGCSFuse(rootDir)
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *staleFileHandleCommon) streamingWritesEnabled() bool {
	s.T().Helper()
	return slices.Contains(flags, "--enable-streaming-writes=true")
}

// Used to validate stale handle error from sync/close when streaming writes are disabled.
func (s *staleFileHandleCommon) validateStaleNFSFileHandleErrorIfStreamingWritesDisabled(err error) {
	s.T().Helper()
	if !s.streamingWritesEnabled() {
		operations.ValidateStaleNFSFileHandleError(s.T(), err)
	} else {
		assert.NoError(s.T(), err)
	}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *staleFileHandleCommon) TestClobberedFileSyncAndCloseThrowsStaleFileHandleError() {
	if s.streamingWritesEnabled() && setup.IsZonalBucketRun() {
		s.T().Skip("Skip the test")
	}
	// Dirty the file by giving it some contents.
	_, err := s.f1.WriteAt([]byte(s.data), 0)
	assert.NoError(s.T(), err)
	// Clobber file by replacing the underlying object with a new generation.
	err = WriteToObject(ctx, storageClient, path.Join(s.T().Name(), FileName1), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	err = s.f1.Sync()

	operations.ValidateESTALEError(s.T(), err)
	err = s.f1.Close()
	operations.ValidateESTALEError(s.T(), err)
}

func (s *staleFileHandleCommon) TestFileDeletedLocallySyncAndCloseDoNotThrowError() {
	// Dirty the file by giving it some contents.
	bytesWrote, err := s.f1.WriteAt([]byte(s.data), 0)
	assert.NoError(s.T(), err)
	// Delete the file.
	operations.RemoveFile(s.f1.Name())
	// Verify unlink operation succeeds.

	operations.ValidateNoFileOrDirError(s.T(), s.f1.Name())
	// Attempt to write to file should not give any error.
	_, err = s.f1.WriteAt([]byte(s.data), int64(bytesWrote))

	assert.NoError(s.T(), err)
	operations.SyncFile(s.f1, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), s.f1)
}
