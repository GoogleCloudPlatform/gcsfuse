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
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type StaleFileHandleLocalFile struct {
	// fsTest has f1 *osFile and f2 *osFile which we will reuse here.
	fsTest
	suite.Suite
}

func TestStaleFileHandleLocalFile(t *testing.T) {
	suite.Run(t, new(StaleFileHandleLocalFile))
}

func (t *StaleFileHandleLocalFile) SetupSuite() {
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

func (t *StaleFileHandleLocalFile) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *StaleFileHandleLocalFile) SetupTest() {
	// Create a local file.
	_, t.f1 = operations.CreateLocalFile(ctx, t.T(), mntDir, bucket, "foo")
}

func (t *StaleFileHandleLocalFile) TearDownTest() {
	filePath := filepath.Join(mntDir, "foo")
	if _, err := os.Stat(filePath); err == nil {
		err = os.Remove(filePath)
		assert.Equal(t.T(), nil, err)
	}

	// fsTest Cleanups to clean up mntDir and close t.f1 and t.f2.
	t.fsTest.TearDown()
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func (t *StaleFileHandleLocalFile) TestLocalInodeClobberedRemotely_SyncAndClose_ThrowsStaleFileHandleError() {
	// Dirty the local file by giving it some contents.
	n, err := t.f1.Write([]byte("taco"))
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), 4, n)
	// Replace the underlying object with a new generation.
	_, err = storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("foobar"))
	assert.Equal(t.T(), nil, err)

	err = t.f1.Sync()

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	err = operations.CloseLocalFile(t.T(), &t.f1)
	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Validate that local file content is not synced to GCS.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), "foobar", string(contents))
}

func (t *StaleFileHandleLocalFile) TestUnlinkedLocalInode_SyncAndClose_ThrowsStaleFileHandleError() {
	// Unlink the local file.
	err := os.Remove(t.f1.Name())
	// Verify unlink operation succeeds.
	assert.Equal(t.T(), nil, err)
	operations.ValidateNoFileOrDirError(t.T(), path.Join(mntDir, FileName))
	// Write to unlinked local file.
	_, err = t.f1.WriteString(FileContents)
	assert.Equal(t.T(), nil, err)

	err = t.f1.Sync()

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	err = operations.CloseLocalFile(t.T(), &t.f1)
	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Verify unlinked file is not present on GCS.
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, FileName)
}

func (t *StaleFileHandleLocalFile) TestUnlinkedDirectoryContainingSyncedAndLocalFiles_Close_ThrowsStaleFileHandleError() {
	// Create explicit directory with one synced and one local file.
	assert.Equal(t.T(),
		nil,
		t.createObjects(
			map[string]string{
				// File
				"explicit/":    "",
				"explicit/foo": "",
			}))
	_, t.f1 = operations.CreateLocalFile(ctx, t.T(), mntDir, bucket, "explicit/"+explicitLocalFileName)
	// Attempt to remove explicit directory.
	err := os.RemoveAll(path.Join(mntDir, "explicit"))
	// Verify rmDir operation succeeds.
	assert.Equal(t.T(), nil, err)
	operations.ValidateNoFileOrDirError(t.T(), path.Join(mntDir, "explicit/"+explicitLocalFileName))
	operations.ValidateNoFileOrDirError(t.T(), path.Join(mntDir, "explicit/foo"))
	operations.ValidateNoFileOrDirError(t.T(), path.Join(mntDir, "explicit"))
	// Validate writing content to unlinked local file does not throw error.
	_, err = t.f1.WriteString(FileContents)
	assert.Equal(t.T(), nil, err)

	err = operations.CloseLocalFile(t.T(), &t.f1)

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Validate both local and synced files are deleted.
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, "explicit/"+explicitLocalFileName)
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, "explicit/foo")
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, "explicit/")
}
