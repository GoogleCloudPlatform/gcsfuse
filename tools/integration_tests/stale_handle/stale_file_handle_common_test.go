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

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
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
	flags                    []string
	f1                       *os.File
	fileName                 string
	data                     string
	testDirPath              string
	isStreamingWritesEnabled bool
	isLocal                  bool
	suite.Suite
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////
func (s *staleFileHandleCommon) SetupSuite() {
	setup.MountGCSFuseWithGivenMountFunc(s.flags, mountFunc)
	s.testDirPath = setup.SetupTestDirectory(testDirName)
	s.data = setup.GenerateRandomString(5 * util.MiB)
}

func (s *staleFileHandleCommon) TearDownSuite() {
	setup.UnmountGCSFuse(rootDir)
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
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
	err := WriteToObject(ctx, storageClient, path.Join(testDirName, s.fileName), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	operations.ValidateSyncGivenThatFileIsClobbered(s.T(), s.f1, s.isStreamingWritesEnabled)

	err = s.f1.Close()
	operations.ValidateESTALEError(s.T(), err)
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, s.fileName, FileContents, s.T())
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
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, s.fileName, s.T())
}

func (s *staleFileHandleCommon) TestRenamedFileSyncAndCloseThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	_, err := s.f1.WriteString(s.data)
	assert.NoError(s.T(), err)
	newFile := "new" + s.fileName

	err = operations.RenameFile(s.f1.Name(), path.Join(s.testDirPath, newFile))

	// TODO(b/402335988): Update this test once rename flow is fixed.
	if s.isLocal && !s.isStreamingWritesEnabled {
		// Rename operation not supported in this scenario.
		operations.ValidateEOPNOTSUPPError(s.T(), err)
		return
	}
	assert.NoError(s.T(), err)
	operations.ValidateWriteGivenThatFileIsRenamed(s.T(), s.f1, s.isStreamingWritesEnabled, s.data)
	operations.ValidateSyncGivenThatFileIsClobbered(s.T(), s.f1, s.isStreamingWritesEnabled)
	operations.ValidateCloseGivenThatFileIsRenamed(s.T(), s.f1, s.isStreamingWritesEnabled)
}
