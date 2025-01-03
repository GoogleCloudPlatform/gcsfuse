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

package fs_test

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type staleFileHandleLocalFile struct {
	staleFileHandleCommon
}

func TestStaleFileHandleLocalFile(t *testing.T) {
	suite.Run(t, new(staleFileHandleLocalFile))
}

func (t *staleFileHandleLocalFile) SetupSuite() {
	t.serverCfg.NewConfig = &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			PreconditionErrors: true,
		},
		MetadataCache: cfg.MetadataCacheConfig{
			TtlSecs: 0,
		},
	}
	t.fsTest.SetUpTestSuite()
}

func (t *staleFileHandleLocalFile) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *staleFileHandleLocalFile) SetupTest() {
	// Create a local file.
	_, t.f1 = operations.CreateLocalFile(ctx, t.T(), mntDir, bucket, "foo")
}

func (t *staleFileHandleLocalFile) TearDownTest() {
	// fsTest Cleanups to clean up mntDir and close t.f1 and t.f2.
	t.fsTest.TearDown()
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func (t *staleFileHandleLocalFile) TestUnlinkedDirectoryContainingSyncedAndLocalFilesCloseThrowsStaleFileHandleError() {
	// Create explicit directory with one synced and one local file.
	assert.Equal(t.T(),
		nil,
		t.createObjects(
			map[string]string{
				// File
				"explicit/":    "",
				"explicit/foo": "",
			}))
	_, t.f2 = operations.CreateLocalFile(ctx, t.T(), mntDir, bucket, "explicit/"+explicitLocalFileName)
	// Attempt to remove explicit directory.
	err := os.RemoveAll(path.Join(mntDir, "explicit"))
	// Verify rmDir operation succeeds.
	assert.NoError(t.T(), err)
	operations.ValidateNoFileOrDirError(t.T(), path.Join(mntDir, "explicit/"+explicitLocalFileName))
	operations.ValidateNoFileOrDirError(t.T(), path.Join(mntDir, "explicit/foo"))
	operations.ValidateNoFileOrDirError(t.T(), path.Join(mntDir, "explicit"))
	// Validate writing content to unlinked local file does not throw error.
	_, err = t.f2.WriteString(FileContents)
	assert.NoError(t.T(), err)

	err = operations.CloseLocalFile(t.T(), &t.f2)

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Validate both local and synced files are deleted.
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, "explicit/"+explicitLocalFileName)
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, "explicit/foo")
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, "explicit/")
}
