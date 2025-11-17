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

package fs_test

import (
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/cfg"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/storageutil"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type staleFileHandleCommon struct {
	// fsTest has f1 *osFile and f2 *osFile which we will reuse here.
	fsTest
	suite.Suite
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func commonServerConfig() *cfg.Config {
	return &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			PreconditionErrors: true,
		},
		MetadataCache: cfg.MetadataCacheConfig{
			TtlSecs: 0,
		},
	}

}

func clobberFile(t *testing.T, content string) {
	t.Helper()
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		fileName,
		[]byte(content))
	assert.NoError(t, err)
}

func createGCSObject(t *testing.T, content string) *os.File {
	t.Helper()
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		fileName,
		[]byte(content))
	assert.NoError(t, err)
	// Open file handle to read or write.
	fh, err := os.OpenFile(path.Join(mntDir, fileName), os.O_RDWR|syscall.O_DIRECT, filePerms)
	assert.NoError(t, err)
	return fh
}

func (t *staleFileHandleCommon) SetupSuite() {
	t.serverCfg.NewConfig = commonServerConfig()
	t.fsTest.SetUpTestSuite()
}

func (t *staleFileHandleCommon) TearDownTest() {
	// fsTest Cleanups to clean up mntDir and close t.f1 and t.f2.
	t.fsTest.TearDown()
}

func (t *staleFileHandleCommon) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func (t *staleFileHandleCommon) TestClobberedFileSyncAndCloseThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	n, err := t.f1.Write([]byte("taco"))
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 4, n)
	// Replace the underlying object with a new generation.
	clobberFile(t.T(), "foobar")

	err = t.f1.Sync()

	operations.ValidateESTALEError(t.T(), err)
	err = t.f1.Close()
	operations.ValidateESTALEError(t.T(), err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
	// Validate that object is not updated with un-synced content.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), "foobar", string(contents))
}

func (t *staleFileHandleCommon) TestFileDeletedLocallySyncAndCloseDoNotThrowError() {
	// Dirty the file by giving it some contents.
	n, err := t.f1.Write([]byte("foobar"))
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 6, n)
	// Unlink the file.
	err = os.Remove(t.f1.Name())
	// Verify unlink operation succeeds.
	assert.NoError(t.T(), err)
	operations.ValidateNoFileOrDirError(t.T(), path.Join(mntDir, "foo"))
	// Attempt to write to file should not give any error.
	n, err = t.f1.Write([]byte("taco"))
	assert.Equal(t.T(), 4, n)
	assert.NoError(t.T(), err)

	operations.SyncFile(t.f1, t.T())
	operations.CloseFileShouldNotThrowError(t.T(), t.f1)

	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, "foo")
}
