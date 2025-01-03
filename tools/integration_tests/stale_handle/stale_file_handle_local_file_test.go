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
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/stretchr/testify/suite"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"

	"github.com/stretchr/testify/assert"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type staleFileHandleLocalFile struct {
	staleFileHandleCommon
}

func (s *staleFileHandleLocalFile) SetupTest() {
	testDirPath := setup.SetupTestDirectory(s.T().Name())
	// Create a local file.
	_, s.f1 = CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, s.T())
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *staleFileHandleLocalFile) TestUnlinkedDirectoryContainingSyncedAndLocalFilesCloseThrowsStaleFileHandleError() {
	explicitDir := path.Join(setup.MntDir(), s.T().Name(), ExplicitDirName)
	// Create explicit directory with one synced and one local file.
	operations.CreateDirectory(explicitDir, s.T())
	CreateObjectInGCSTestDir(ctx, storageClient, path.Join(s.T().Name(), ExplicitDirName), ExplicitFileName1, "", s.T())
	_, f2 := CreateLocalFileInTestDir(ctx, storageClient, explicitDir, ExplicitLocalFileName1, s.T())
	err := os.RemoveAll(explicitDir)
	assert.NoError(s.T(), err)
	operations.ValidateNoFileOrDirError(s.T(), explicitDir+"/")
	operations.ValidateNoFileOrDirError(s.T(), path.Join(explicitDir, ExplicitFileName1))
	operations.ValidateNoFileOrDirError(s.T(), path.Join(explicitDir, ExplicitLocalFileName1))
	// Validate writing content to unlinked local file does not throw error.
	operations.WriteWithoutClose(f2, FileContents, s.T())

	err = f2.Close()

	operations.ValidateStaleNFSFileHandleError(s.T(), err)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestStaleFileHandleLocalFileTest(t *testing.T) {
	suite.Run(t, new(staleFileHandleLocalFile))
}
