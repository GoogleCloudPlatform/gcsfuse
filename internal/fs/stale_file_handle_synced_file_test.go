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

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type staleFileHandleSyncedFile struct {
	staleFileHandleCommon
}

func TestStaleFileHandleSyncedFile(t *testing.T) {
	suite.Run(t, new(staleFileHandleSyncedFile))
}

func (t *staleFileHandleSyncedFile) SetupSuite() {
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

func (t *staleFileHandleSyncedFile) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}
func (t *staleFileHandleSyncedFile) SetupTest() {
	// Create an object on bucket.
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	assert.NoError(t.T(), err)
	// Open file handle to read or write.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_RDWR|syscall.O_DIRECT, filePerms)
	assert.NoError(t.T(), err)
}

func (t *staleFileHandleSyncedFile) TearDownTest() {
	// fsTest Cleanups to clean up mntDir and close t.f1 and t.f2.
	t.fsTest.TearDown()
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func (t *staleFileHandleSyncedFile) TestClobberedFileReadThrowsStaleFileHandleError() {
	// Replace the underlying object with a new generation.
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("foobar"))
	assert.NoError(t.T(), err)

	buffer := make([]byte, 6)
	_, err = t.f1.Read(buffer)

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Validate that object is updated with new content.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), "foobar", string(contents))
}

func (t *staleFileHandleSyncedFile) TestClobberedFileFirstWriteThrowsStaleFileHandleError() {
	// Replace the underlying object with a new generation.
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("foobar"))
	assert.NoError(t.T(), err)

	_, err = t.f1.Write([]byte("taco"))

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Attempt to sync to file should not result in error as we first check if the
	// content has been dirtied before clobbered check in Sync flow.
	err = t.f1.Sync()
	assert.NoError(t.T(), err)
	// Validate that object is not updated with new content as write failed.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), "foobar", string(contents))
}

func (t *staleFileHandleSyncedFile) TestRenamedFileSyncThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	n, err := t.f1.Write([]byte("foobar"))
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 6, n)
	// Rename the object.
	err = os.Rename(t.f1.Name(), path.Join(mntDir, "bar"))
	assert.NoError(t.T(), err)
	// Attempt to write to file should not give any error.
	n, err = t.f1.Write([]byte("taco"))
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 4, n)

	err = t.f1.Sync()

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	err = t.f1.Close()
	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
}

func (t *staleFileHandleSyncedFile) TestFileDeletedRemotelySyncAndCloseThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	n, err := t.f1.Write([]byte("foobar"))
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 6, n)
	// Unlink the file.
	err = storageutil.DeleteObject(ctx, bucket, "foo")
	assert.NoError(t.T(), err)
	// Verify unlink operation succeeds.
	assert.NoError(t.T(), err)
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, "foo")
	// Attempt to write to file should not give any error.
	n, err = t.f1.Write([]byte("taco"))
	assert.Equal(t.T(), 4, n)
	assert.NoError(t.T(), err)

	err = t.f1.Sync()

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	err = t.f1.Close()
	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
}
