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

type StaleFileHandleSyncedFile struct {
	// fsTest has f1 *osFile and f2 *osFile which we will reuse here.
	fsTest
	suite.Suite
}

func TestStaleFileHandleSyncedFile(t *testing.T) {
	suite.Run(t, new(StaleFileHandleSyncedFile))
}

func (t *StaleFileHandleSyncedFile) SetupSuite() {
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

func (t *StaleFileHandleSyncedFile) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}
func (t *StaleFileHandleSyncedFile) SetupTest() {
	// Create an object on bucket.
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	assert.Equal(t.T(), nil, err)
	// Open file handle to read or write.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_RDWR|syscall.O_DIRECT, filePerms)
	assert.Equal(t.T(), nil, err)
}

func (t *StaleFileHandleSyncedFile) TearDownTest() {
	err := storageutil.DeleteObject(
		ctx,
		bucket,
		"foo")
	assert.Equal(t.T(), nil, err)

	// fsTest Cleanups to clean up mntDir and close t.f1 and t.f2.
	t.fsTest.TearDown()
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func (t *StaleFileHandleSyncedFile) TestSyncedObjectClobberedRemotely_Read_ThrowsStaleFileHandleError() {
	// Replace the underlying object with a new generation.
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("foobar"))
	assert.Equal(t.T(), nil, err)

	buffer := make([]byte, 6)
	_, err = t.f1.Read(buffer)

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Validate that object is updated with new content.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), "foobar", string(contents))
}

func (t *StaleFileHandleSyncedFile) TestSyncedObjectClobberedRemotely_FirstWrite_ThrowsStaleFileHandleError() {
	// Replace the underlying object with a new generation.
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("foobar"))
	assert.Equal(t.T(), nil, err)

	_, err = t.f1.Write([]byte("taco"))

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Attempt to sync to file should not result in error as we first check if the
	// content has been dirtied before clobbered check in Sync flow.
	err = t.f1.Sync()
	assert.Equal(t.T(), nil, err)
	// Validate that object is not updated with new content as write failed.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), "foobar", string(contents))
}

func (t *StaleFileHandleSyncedFile) TestSyncedObjectClobberedRemotely_SyncAndClose_ThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
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
	err = t.f1.Close()
	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
	// Validate that object is not updated with un-synced content.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), "foobar", string(contents))
}

func (t *StaleFileHandleSyncedFile) TestSyncedObjectDeletedRemotely_SyncAndClose_ThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	n, err := t.f1.Write([]byte("foobar"))
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), 6, n)
	// Delete the object remotely.
	err = storageutil.DeleteObject(ctx, bucket, "foo")
	assert.Equal(t.T(), nil, err)
	// Attempt to write to file should not give any error.
	n, err = t.f1.Write([]byte("taco"))
	assert.Equal(t.T(), 4, n)
	assert.Equal(t.T(), nil, err)

	err = t.f1.Sync()

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	err = t.f1.Close()
	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
}

func (t *StaleFileHandleSyncedFile) TestSyncedObjectDeletedLocally_SyncAndClose_ThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	n, err := t.f1.Write([]byte("foobar"))
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), 6, n)
	// Delete the object locally.
	err = os.Remove(t.f1.Name())
	assert.Equal(t.T(), nil, err)
	// Attempt to write to file should not give any error.
	n, err = t.f1.Write([]byte("taco"))
	assert.Equal(t.T(), 4, n)
	assert.Equal(t.T(), nil, err)

	err = t.f1.Sync()

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	err = t.f1.Close()
	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
}

func (t *StaleFileHandleSyncedFile) TestRenamedSyncedObject_SyncAndClose_ThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	n, err := t.f1.Write([]byte("foobar"))
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), 6, n)
	// Rename the object.
	err = os.Rename(t.f1.Name(), path.Join(mntDir, "bar"))
	assert.Equal(t.T(), nil, err)
	// Attempt to write to file should not give any error.
	n, err = t.f1.Write([]byte("taco"))
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), 4, n)

	err = t.f1.Sync()

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	err = t.f1.Close()
	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
}
