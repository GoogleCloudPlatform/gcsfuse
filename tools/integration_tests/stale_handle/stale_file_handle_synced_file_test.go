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
	"path"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

// This test-suite contains parallelizable test-case. Use "-parallel n" to limit
// the degree of parallelism. By default it uses GOMAXPROCS.
// Ref: https://stackoverflow.com/questions/24375966/does-go-test-run-unit-tests-concurrently
type staleFileHandleSyncedFile struct{}

func (s *staleFileHandleSyncedFile) Setup(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func (s *staleFileHandleSyncedFile) Teardown(t *testing.T) {}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func TestSyncedObjectClobberedRemotelyReadThrowsStaleFileHandleError(t *testing.T) {
	t.Parallel() // Mark the test parallelizable.
	testCaseDir := "TestSyncedObjectClobberedRemotelyReadThrowsStaleFileHandleError"
	gcsDirPath := path.Join(testDirName, testCaseDir)
	// Create an object on bucket
	CreateObjectInGCSTestDir(ctx, storageClient, gcsDirPath, FileName1, GCSFileContent, t)
	targetDir := path.Join(testDirPath, testCaseDir)
	filePath := path.Join(targetDir, FileName1)
	// Open the read handle
	fh, err := operations.OpenFileAsReadonly(filePath)
	assert.Equal(t, nil, err)
	// Replace the underlying object with a new generation.
	CreateObjectInGCSTestDir(ctx, storageClient, gcsDirPath, FileName1, FileContents, t)

	buffer := make([]byte, GCSFileSize)
	err = operations.ReadBytesFromFile(fh, GCSFileSize, buffer)

	assert.NotEqual(t, nil, err)
	operations.ValidateStaleNFSFileHandleError(t, err)
	ValidateObjectContentsFromGCS(ctx, storageClient, targetDir, FileName1, FileContents, t)
}

//func TestSyncedObjectClobberedRemotelyFirstWriteThrowsStaleFileHandleError(t *testing.T) {
//	// Replace the underlying object with a new generation.
//	_, err := storageutil.CreateObject(
//		ctx,
//		bucket,
//		"foo",
//		[]byte("foobar"))
//	assert.Equal(t.T(), nil, err)
//
//	_, err = t.f1.Write([]byte("taco"))
//
//	operations.ValidateStaleNFSFileHandleError(t.T(), err)
//	// Attempt to sync to file should not result in error as we first check if the
//	// content has been dirtied before clobbered check in Sync flow.
//	err = t.f1.Sync()
//	assert.Equal(t.T(), nil, err)
//	// Validate that object is not updated with new content as write failed.
//	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
//	assert.Equal(t.T(), nil, err)
//	assert.Equal(t.T(), "foobar", string(contents))
//}
//
//func TestSyncedObjectClobberedRemotely_SyncAndClose_ThrowsStaleFileHandleError(t *testing.T) {
//	// Dirty the file by giving it some contents.
//	n, err := t.f1.Write([]byte("taco"))
//	assert.Equal(t.T(), nil, err)
//	assert.Equal(t.T(), 4, n)
//	// Replace the underlying object with a new generation.
//	_, err = storageutil.CreateObject(
//		ctx,
//		bucket,
//		"foo",
//		[]byte("foobar"))
//	assert.Equal(t.T(), nil, err)
//
//	err = t.f1.Sync()
//
//	operations.ValidateStaleNFSFileHandleError(t.T(), err)
//	err = t.f1.Close()
//	operations.ValidateStaleNFSFileHandleError(t.T(), err)
//	// Make f1 nil, so that another attempt is not taken in TearDown to close the
//	// file.
//	t.f1 = nil
//	// Validate that object is not updated with un-synced content.
//	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
//	assert.Equal(t.T(), nil, err)
//	assert.Equal(t.T(), "foobar", string(contents))
//}
//
//func TestSyncedObjectDeletedRemotelySyncAndCloseThrowsStaleFileHandleError(t *testing.T) {
//	// Dirty the file by giving it some contents.
//	n, err := t.f1.Write([]byte("foobar"))
//	assert.Equal(t.T(), nil, err)
//	assert.Equal(t.T(), 6, n)
//	// Delete the object remotely.
//	err = storageutil.DeleteObject(ctx, bucket, "foo")
//	assert.Equal(t.T(), nil, err)
//	// Attempt to write to file should not give any error.
//	n, err = t.f1.Write([]byte("taco"))
//	assert.Equal(t.T(), 4, n)
//	assert.Equal(t.T(), nil, err)
//
//	err = t.f1.Sync()
//
//	operations.ValidateStaleNFSFileHandleError(t.T(), err)
//	err = t.f1.Close()
//	operations.ValidateStaleNFSFileHandleError(t.T(), err)
//	// Make f1 nil, so that another attempt is not taken in TearDown to close the
//	// file.
//	t.f1 = nil
//}
//
//func TestSyncedObjectDeletedLocallySyncAndCloseThrowsStaleFileHandleError(t *testing.T) {
//	// Dirty the file by giving it some contents.
//	n, err := t.f1.Write([]byte("foobar"))
//	assert.Equal(t.T(), nil, err)
//	assert.Equal(t.T(), 6, n)
//	// Delete the object locally.
//	err = os.Remove(t.f1.Name())
//	assert.Equal(t.T(), nil, err)
//	// Attempt to write to file should not give any error.
//	n, err = t.f1.Write([]byte("taco"))
//	assert.Equal(t.T(), 4, n)
//	assert.Equal(t.T(), nil, err)
//
//	err = t.f1.Sync()
//
//	operations.ValidateStaleNFSFileHandleError(t.T(), err)
//	err = t.f1.Close()
//	operations.ValidateStaleNFSFileHandleError(t.T(), err)
//	// Make f1 nil, so that another attempt is not taken in TearDown to close the
//	// file.
//	t.f1 = nil
//}
//
//func TestRenamedSyncedObject_Sync_ThrowsStaleFileHandleError(t *testing.T) {
//	// Dirty the file by giving it some contents.
//	n, err := t.f1.Write([]byte("foobar"))
//	assert.Equal(t.T(), nil, err)
//	assert.Equal(t.T(), 6, n)
//	// Rename the object.
//	err = os.Rename(t.f1.Name(), path.Join(mntDir, "bar"))
//	assert.Equal(t.T(), nil, err)
//	// Attempt to write to file should not give any error.
//	n, err = t.f1.Write([]byte("taco"))
//	assert.Equal(t.T(), nil, err)
//	assert.Equal(t.T(), 4, n)
//
//	err = t.f1.Sync()
//
//	operations.ValidateStaleNFSFileHandleError(t.T(), err)
//	err = t.f1.Close()
//	operations.ValidateStaleNFSFileHandleError(t.T(), err)
//	// Make f1 nil, so that another attempt is not taken in TearDown to close the
//	// file.
//	t.f1 = nil
//}
