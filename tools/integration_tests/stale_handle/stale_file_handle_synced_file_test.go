// Copyright 2024 Google LLC
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

const Content = "foobar"
const Content2 = "foobar2"

type staleFileHandleSyncedFile struct {
	staleFileHandleCommon
}

func (s *staleFileHandleSyncedFile) SetupTest() {
	testDirPath := setup.SetupTestDirectory(s.T().Name())
	// Create an object on bucket
	err := CreateObjectOnGCS(ctx, storageClient, path.Join(s.T().Name(), FileName1), GCSFileContent)
	assert.NoError(s.T(), err)
	s.f1, err = os.OpenFile(path.Join(testDirPath, FileName1), os.O_RDWR|syscall.O_DIRECT, operations.FilePermission_0600)
	assert.NoError(s.T(), err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *staleFileHandleSyncedFile) TestClobberedFileReadThrowsStaleFileHandleError() {
	// Replace the underlying object with a new generation.
	err := WriteToObject(ctx, storageClient, path.Join(s.T().Name(), FileName1), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	buffer := make([]byte, GCSFileSize)
	_, err = s.f1.Read(buffer)

	operations.ValidateStaleNFSFileHandleError(s.T(), err)
	ValidateObjectContentsFromGCS(ctx, storageClient, s.T().Name(), FileName1, FileContents, s.T())
}

func (s *staleFileHandleSyncedFile) TestClobberedFileFirstWriteThrowsStaleFileHandleError() {
	// Replace the underlying object with a new generation.
	err := WriteToObject(ctx, storageClient, path.Join(s.T().Name(), FileName1), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	_, err = s.f1.WriteString(Content)

	operations.ValidateStaleNFSFileHandleError(s.T(), err)
	// Attempt to sync to file should not result in error as we first check if the
	// content has been dirtied before clobbered check in Sync flow.
	operations.SyncFile(s.f1, s.T())
	// Validate that object is not updated with new content as write failed.
	ValidateObjectContentsFromGCS(ctx, storageClient, s.T().Name(), FileName1, FileContents, s.T())
}

func (s *staleFileHandleSyncedFile) TestRenamedFileSyncAndCloseThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	_, err := s.f1.WriteString(Content)
	assert.NoError(s.T(), err)
	err = operations.RenameFile(s.f1.Name(), path.Join(setup.MntDir(), s.T().Name(), FileName2))
	assert.NoError(s.T(), err)
	// Attempt to write to file should not give any error.
	_, err = s.f1.WriteString(Content2)
	assert.NoError(s.T(), err)

	err = s.f1.Sync()

	operations.ValidateStaleNFSFileHandleError(s.T(), err)
	err = s.f1.Close()
	operations.ValidateStaleNFSFileHandleError(s.T(), err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	s.f1 = nil
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestStaleFileHandleSyncedFileTest(t *testing.T) {
	suite.Run(t, new(staleFileHandleSyncedFile))
}
