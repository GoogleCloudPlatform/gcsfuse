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
type staleFileHandleLocalFileTest struct{}

func (s *staleFileHandleLocalFileTest) Setup(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func (s *staleFileHandleLocalFileTest) Teardown(t *testing.T) {}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func TestLocalInodeClobberedRemotelySyncAndCloseThrowsStaleFileHandleError(t *testing.T) {
	t.Parallel() // Mark the test parallelizable.
	testCaseDir := "TestLocalInodeClobberedRemotelySyncAndCloseThrowsStaleFileHandleError"
	targetDir := path.Join(testDirPath, testCaseDir)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, targetDir, FileName1, t)
	// Dirty the file by giving it some contents.
	operations.WriteWithoutClose(fh, FileContents, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t)
	gcsDirPath := path.Join(testDirName, testCaseDir)
	// Replace the underlying object with a new generation.
	CreateObjectInGCSTestDir(ctx, storageClient, gcsDirPath, FileName1, GCSFileContent, t)

	operations.SyncFileShouldThrowStaleHandleError(fh, t)
	operations.CloseFileShouldThrowStaleHandleError(fh, t)

	ValidateObjectContentsFromGCS(ctx, storageClient, targetDir, FileName1, GCSFileContent, t)
}

func TestUnlinkedLocalInodeSyncAndCloseThrowsStaleFileHandleError(t *testing.T) {
	t.Parallel() // Mark the test parallelizable.
	testCaseDir := "TestLocalInodeClobberedRemotelySyncAndCloseThrowsStaleFileHandleError"
	targetDir := path.Join(testDirPath, testCaseDir)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, targetDir, FileName1, t)
	// Unlink the local file.
	operations.RemoveFile(fh.Name())
	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(t, fh.Name())
	// Write to unlinked local file.
	err := operations.WriteFile(fh.Name(), FileContents)
	assert.Equal(t, nil, err)

	operations.SyncFileShouldThrowStaleHandleError(fh, t)
	operations.CloseFileShouldThrowStaleHandleError(fh, t)

	// Verify unlinked file is not present on GCS.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, path.Join(testCaseDir, FileName1), t)
}

func TestUnlinkedDirectoryContainingSyncedAndLocalFiles_Close_ThrowsStaleFileHandleError(t *testing.T) {
	// Create explicit directory with one synced and one local file.
	operations.CreateDirectory(path.Join(testDirPath, ExplicitDirName), t)
	assert.Equal(t,
		nil,
		t.createObjects(
			map[string]string{
				// File
				"explicit/":    "",
				"explicit/foo": "",
			}))
	fh := operations.CreateLocalFile(ctx, t, mntDir, bucket, "explicit/"+explicitLocalFileName)
	// Attempt to remove explicit directory.
	err := os.RemoveAll(path.Join(mntDir, "explicit"))
	// Verify rmDir operation succeeds.
	assert.Equal(t, nil, err)
	operations.ValidateNoFileOrDirError(t, path.Join(mntDir, "explicit/"+explicitLocalFileName))
	operations.ValidateNoFileOrDirError(t, path.Join(mntDir, "explicit/foo"))
	operations.ValidateNoFileOrDirError(t, path.Join(mntDir, "explicit"))
	// Validate writing content to unlinked local file does not throw error.
	_, err = t.f1.WriteString(FileContents)
	assert.Equal(t, nil, err)

	err = operations.CloseLocalFile(t, &t.f1)

	operations.ValidateStaleNFSFileHandleError(t, err)
	// Validate both local and synced files are deleted.
	operations.ValidateObjectNotFoundErr(ctx, t, bucket, "explicit/"+explicitLocalFileName)
	operations.ValidateObjectNotFoundErr(ctx, t, bucket, "explicit/foo")
	operations.ValidateObjectNotFoundErr(ctx, t, bucket, "explicit/")
}
