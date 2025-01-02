// Copyright 2024 Google LLC
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
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type staleFileHandleCommon struct {
	flags []string
	f1    *os.File
	f2    *os.File
	suite.Suite
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *staleFileHandleCommon) TestClobberedFileSyncAndCloseThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	operations.WriteWithoutClose(s.f1, Content, s.T())
	// Replace the underlying object with a new generation.
	err := WriteToObject(ctx, storageClient, path.Join(s.T().Name(), FileName1), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	err = s.f1.Sync()

	operations.ValidateStaleNFSFileHandleError(s.T(), err)
	err = s.f1.Close()
	operations.ValidateStaleNFSFileHandleError(s.T(), err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	s.f1 = nil
	ValidateObjectContentsFromGCS(ctx, storageClient, s.T().Name(), FileName1, FileContents, s.T())
}

func (s *staleFileHandleCommon) TestDeletedFileSyncAndCloseThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	_, err := s.f1.WriteString(Content)
	assert.NoError(s.T(), err)
	// Delete the file.
	operations.RemoveFile(s.f1.Name())
	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(s.T(), s.f1.Name())
	// Attempt to write to file should not give any error.
	operations.WriteWithoutClose(s.f1, Content2, s.T())

	err = s.f1.Sync()

	operations.ValidateStaleNFSFileHandleError(s.T(), err)
	err = s.f1.Close()
	operations.ValidateStaleNFSFileHandleError(s.T(), err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	s.f1 = nil
	// Verify unlinked file is not present on GCS.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, s.T().Name(), FileName1, s.T())
}
