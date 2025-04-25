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
	"log"
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
	flags       []string
	f1          *os.File
	data        string
	testDirPath string
	suite.Suite
	validator
}

// validator validates the error from sync/close/write operation after the file has been clobbered.
type validator interface {
	validate(err error)
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (s *staleFileHandleCommon) validate(err error) {
	operations.ValidateESTALEError(s.T(), err)
}

func (s *staleFileHandleCommon) SetupSuite() {
	log.Printf("Running tests with flag: %v", s.flags)
	setup.MountGCSFuseWithGivenMountFunc(s.flags, mountFunc)
}

func (s *staleFileHandleCommon) TearDownSuite() {
	setup.UnmountGCSFuse(rootDir)
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *staleFileHandleCommon) streamingWritesEnabled() bool {
	s.T().Helper()
	return slices.Contains(s.flags, "--enable-streaming-writes=true")
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *staleFileHandleCommon) TestClobberedFileSyncAndCloseThrowsStaleFileHandleError() {
	// TODO(b/410698332): Remove skip condition once takeover support is ready.
	if s.streamingWritesEnabled() && setup.IsZonalBucketRun() {
		s.T().Skip("Skip the test until unfinalized object overwrite is supported.")
	}
	// Dirty the file by giving it some contents.
	operations.WriteWithoutClose(s.f1, s.data, s.T())
	// Clobber file by replacing the underlying object with a new generation.
	err := WriteToObject(ctx, storageClient, path.Join(s.T().Name(), FileName1), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	err = s.f1.Sync()
	s.validator.validate(err)

	err = s.f1.Close()
	operations.ValidateESTALEError(s.T(), err)
}

func (s *staleFileHandleCommon) TestFileDeletedLocallySyncAndCloseDoNotThrowError() {
	// TODO(b/410698332): Remove skip condition once generation issue is fixed for ZB.
	if s.streamingWritesEnabled() && setup.IsZonalBucketRun() {
		s.T().Skip("Skip the test due to generation issue in ZB.")
	}
	// Dirty the file by giving it some contents.
	operations.WriteWithoutClose(s.f1, s.data, s.T())
	operations.SyncFile(s.f1, s.T())
	// Delete the file.
	operations.RemoveFile(s.f1.Name())
	// Verify unlink operation succeeds.

	operations.ValidateNoFileOrDirError(s.T(), s.f1.Name())
	_, err := s.f1.Write([]byte(s.data))
	s.validator.validate(err)

	operations.SyncFile(s.f1, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), s.f1)
}
