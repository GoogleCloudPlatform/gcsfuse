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

type StaleHandleTest struct {
	// fsTest has f1 *osFile and f2 *osFile which we will reuse here.
	f3 *os.File
	fsTest
	suite.Suite
}

func TestFileTestSuite(t *testing.T) {
	suite.Run(t, new(StaleHandleTest))
}

func (t *StaleHandleTest) SetupTest() {
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

func (t *StaleHandleTest) TearDownTest() {
	// Close t.f3 in case of test failure.
	if t.f3 != nil {
		assert.Equal(t.T(), nil, t.f3.Close())
		t.f3 = nil
	}

	// fsTest Cleanups to clean up mntDir and close t.f1 and t.f2.
	t.fsTest.TearDown()
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////
func (t *StaleHandleTest) validateStaleNFSFileHandleError(err error) {
	assert.NotEqual(t.T(), nil, err)
	assert.Regexp(t.T(), "stale NFS file handle", err.Error())
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func (t *StaleHandleTest) TestSyncedObjectClobberedRemotely_Read_ThrowsStaleFileHandleError() {
	// Create an object on bucket.
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	assert.Equal(t.T(), nil, err)
	// Open file handle to read.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_RDONLY|syscall.O_DIRECT, filePerms)
	assert.NotEqual(t.T(), nil, err)
	// Replace the underlying object with a new generation.
	_, err = storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("foobar"))
	assert.Equal(t.T(), nil, err)

	buffer := make([]byte, 6)
	_, err = t.f1.Read(buffer)

	t.validateStaleNFSFileHandleError(err)
	// Validate that object is updated with new content.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), "foobar", string(contents))
}

func (t *StaleHandleTest) TestSyncedObjectClobberedRemotely_FirstWrite_ThrowsStaleFileHandleError() {
	// Create an object on bucket.
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	assert.Equal(t.T(), nil, err)
	// Open file handle to write.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_WRONLY|syscall.O_DIRECT, filePerms)
	assert.Equal(t.T(), nil, err)
	// Replace the underlying object with a new generation.
	_, err = storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("foobar"))
	assert.Equal(t.T(), nil, err)

	_, err = t.f1.Write([]byte("taco"))

	t.validateStaleNFSFileHandleError(err)
	// Attempt to sync to file should not result in error as we first check if the
	// content has been dirtied before clobbered check in Sync flow.
	err = t.f1.Sync()
	assert.Equal(t.T(), nil, err)
	// Validate that object is not updated with new content as write failed.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), "foobar", string(contents))
}

func (t *StaleHandleTest) TestLocalInodeClobberedRemotely_SyncAndClose_ThrowsStaleFileHandleError() {
	// Create a local file.
	_, t.f1 = operations.CreateLocalFile(ctx, mntDir, bucket, "foo", t.T())
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

	t.validateStaleNFSFileHandleError(err)
	err = operations.CloseLocalFile(&t.f1, t.T())
	t.validateStaleNFSFileHandleError(err)
	// Validate that local file content is not synced to GCS.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), "foobar", string(contents))
}

func (t *StaleHandleTest) TestSyncedObjectClobberedRemotely_SyncAndClose_ThrowsStaleFileHandleError() {
	// Create an object on bucket.
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	assert.Equal(t.T(), nil, err)
	// Open file handle to write.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_WRONLY|syscall.O_DIRECT, filePerms)
	assert.Equal(t.T(), nil, err)
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

	t.validateStaleNFSFileHandleError(err)
	err = t.f1.Close()
	t.validateStaleNFSFileHandleError(err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
	// Validate that object is not updated with un-synced content.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), "foobar", string(contents))
}

func (t *StaleHandleTest) TestUnlinkedLocalInode_SyncAndClose_ThrowsStaleFileHandleError() {
	// Create local file and unlink.
	var filePath string
	filePath, t.f1 = operations.CreateLocalFile(ctx, mntDir, bucket, FileName, t.T())
	err := os.Remove(filePath)
	// Verify unlink operation succeeds.
	assert.Equal(t.T(), nil, err)
	operations.ValidateNoFileOrDirError(path.Join(mntDir, FileName), t.T())
	// Write to unlinked local file.
	_, err = t.f1.WriteString(FileContents)
	assert.Equal(t.T(), nil, err)

	err = t.f1.Sync()

	t.validateStaleNFSFileHandleError(err)
	err = operations.CloseLocalFile(&t.f1, t.T())
	t.validateStaleNFSFileHandleError(err)
	// Verify unlinked file is not present on GCS.
	operations.ValidateObjectNotFoundErr(ctx, bucket, FileName, t.T())
}

func (t *StaleHandleTest) TestSyncedObjectDeletedRemotely_SyncAndClose_ThrowsStaleFileHandleError() {
	// Create an object on bucket.
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	assert.Equal(t.T(), nil, err)
	// Open file handle to write.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_WRONLY|syscall.O_DIRECT, filePerms)
	assert.Equal(t.T(), nil, err)
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

	t.validateStaleNFSFileHandleError(err)
	err = t.f1.Close()
	t.validateStaleNFSFileHandleError(err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
}

func (t *StaleHandleTest) TestSyncedObjectDeletedLocally_SyncAndClose_ThrowsStaleFileHandleError() {
	// Create an object on bucket.
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	assert.Equal(t.T(), nil, err)
	// Open file handle to write.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_WRONLY|syscall.O_DIRECT, filePerms)
	assert.Equal(t.T(), nil, err)
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

	t.validateStaleNFSFileHandleError(err)
	err = t.f1.Close()
	t.validateStaleNFSFileHandleError(err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
}

func (t *StaleHandleTest) TestUnlinkedDirectoryContainingGCSAndLocalFiles_Close_ThrowsStaleFileHandleError() {
	// Create explicit directory with one synced and one local file.
	assert.Equal(t.T(),
		nil,
		t.createObjects(
			map[string]string{
				// File
				"explicit/":    "",
				"explicit/foo": "",
			}))
	_, t.f1 = operations.CreateLocalFile(ctx, mntDir, bucket, "explicit/"+explicitLocalFileName, t.T())
	// Attempt to remove explicit directory.
	err := os.RemoveAll(path.Join(mntDir, "explicit"))
	// Verify rmDir operation succeeds.
	assert.Equal(t.T(), nil, err)
	operations.ValidateNoFileOrDirError(path.Join(mntDir, "explicit/"+explicitLocalFileName), t.T())
	operations.ValidateNoFileOrDirError(path.Join(mntDir, "explicit/foo"), t.T())
	operations.ValidateNoFileOrDirError(path.Join(mntDir, "explicit"), t.T())
	// Validate writing content to unlinked local file does not throw error.
	_, err = t.f1.WriteString(FileContents)
	assert.Equal(t.T(), nil, err)

	err = operations.CloseLocalFile(&t.f1, t.T())

	t.validateStaleNFSFileHandleError(err)
	// Validate both local and synced files are deleted.
	operations.ValidateObjectNotFoundErr(ctx, bucket, "explicit/"+explicitLocalFileName, t.T())
	operations.ValidateObjectNotFoundErr(ctx, bucket, "explicit/foo", t.T())
	operations.ValidateObjectNotFoundErr(ctx, bucket, "explicit/", t.T())
}

func (t *StaleHandleTest) TestRenamedSyncedObject_Sync_ThrowsStaleFileHandleError() {
	// Create an object on bucket.
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("bar"))
	assert.Equal(t.T(), nil, err)
	// Open file handle to write.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_WRONLY|syscall.O_DIRECT, filePerms)
	assert.Equal(t.T(), nil, err)
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

	t.validateStaleNFSFileHandleError(err)
	err = t.f1.Close()
	t.validateStaleNFSFileHandleError(err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
}
