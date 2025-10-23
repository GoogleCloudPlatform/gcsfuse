// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type staleFileHandleCommon struct {
	suite.Suite
	flags                    []string
	f1                       *os.File
	fileName                 string
	data                     string
	isStreamingWritesEnabled bool
	isLocal                  bool
}

func (s *staleFileHandleCommon) SetupSuite() {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, s.flags, mountFunc)
	testEnv.testDirPath = SetupTestDirectory(testEnv.ctx, testEnv.storageClient, testDirName)
	s.data = setup.GenerateRandomString(5 * util.MiB)
}

func (s *staleFileHandleCommon) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *staleFileHandleCommon) TestClobberedFileSyncAndCloseThrowsStaleFileHandleError() {
	// TODO(b/410698332): Remove skip condition once takeover support is available.
	if s.isStreamingWritesEnabled && setup.IsZonalBucketRun() {
		s.T().Skip("Skip test due to unable to overwrite the unfinalized zonal object.")
	}
	// Dirty the file by giving it some contents.
	operations.WriteWithoutClose(s.f1, s.data, s.T())
	// Clobber file by replacing the underlying object with a new generation.
	err := WriteToObject(testEnv.ctx, testEnv.storageClient, path.Join(testDirName, s.fileName), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	if s.isStreamingWritesEnabled && !s.isLocal {
		err = s.f1.Sync()
		operations.ValidateESTALEError(s.T(), err)
	} else {
		operations.ValidateSyncGivenThatFileIsClobbered(s.T(), s.f1, s.isStreamingWritesEnabled)
	}

	err = s.f1.Close()
	operations.ValidateESTALEError(s.T(), err)
	ValidateObjectContentsFromGCS(testEnv.ctx, testEnv.storageClient, testDirName, s.fileName, FileContents, s.T())
}

func (s *staleFileHandleCommon) TestFileDeletedLocallySyncAndCloseDoNotThrowError() {
	// Dirty the file by giving it some contents.
	operations.WriteWithoutClose(s.f1, s.data, s.T())

	// Delete the file.
	operations.RemoveFile(s.f1.Name())

	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(s.T(), s.f1.Name())
	// Attempt to write to file should not give any error.
	operations.WriteWithoutClose(s.f1, s.data, s.T())
	operations.SyncFile(s.f1, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), s.f1)
	ValidateObjectNotFoundErrOnGCS(testEnv.ctx, testEnv.storageClient, testDirName, s.fileName, s.T())
}

func (s *staleFileHandleCommon) TestRenamedFileSyncAndCloseThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	_, err := s.f1.WriteString(s.data)
	assert.NoError(s.T(), err)
	newFile := "new" + s.fileName

	err = operations.RenameFile(s.f1.Name(), path.Join(testEnv.testDirPath, newFile))

	assert.NoError(s.T(), err)
	_, err = s.f1.WriteString(s.data)
	operations.ValidateESTALEError(s.T(), err)
	// Sync/Flush call won't throw error as data couldn't be written after rename, so we don't have anything to upload.
	err = s.f1.Sync()
	require.NoError(s.T(), err)
	err = s.f1.Close()
	require.NoError(s.T(), err)
}
