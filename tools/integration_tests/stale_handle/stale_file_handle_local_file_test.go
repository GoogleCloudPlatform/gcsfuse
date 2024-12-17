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
	"log"
	"path"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"

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
type staleFileHandleLocalFile struct {
	flags []string
}

func (s *staleFileHandleLocalFile) Setup(t *testing.T) {
	setup.MountGCSFuseWithGivenMountFunc(s.flags, mountFunc)
}

func (s *staleFileHandleLocalFile) Teardown(t *testing.T) {
	setup.UnmountGCSFuse(setup.MntDir())
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *staleFileHandleLocalFile) TestLocalInodeClobberedRemotelySyncAndCloseThrowsStaleFileHandleError(t *testing.T) {
	t.Parallel() // Mark the test parallelizable.
	testCaseDir := "TestLocalInodeClobberedRemotelySyncAndCloseThrowsStaleFileHandleError" + setup.GenerateRandomString(3)
	testDirPath = setup.SetupTestDirectory(testCaseDir)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
	// Dirty the file by giving it some contents.
	operations.WriteWithoutClose(fh, FileContents, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testCaseDir, FileName1, t)
	// Replace the underlying object with a new generation.
	CreateObjectInGCSTestDir(ctx, storageClient, testCaseDir, FileName1, GCSFileContent, t)

	operations.SyncFileShouldThrowStaleHandleError(fh, t)
	operations.CloseFileShouldThrowStaleHandleError(fh, t)

	ValidateObjectContentsFromGCS(ctx, storageClient, testCaseDir, FileName1, GCSFileContent, t)
}

func (s *staleFileHandleLocalFile) TestUnlinkedLocalInodeSyncAndCloseThrowsStaleFileHandleError(t *testing.T) {
	t.Parallel() // Mark the test parallelizable.
	testCaseDir := "TestUnlinkedLocalInodeSyncAndCloseThrowsStaleFileHandleError" + setup.GenerateRandomString(3)
	testDirPath = setup.SetupTestDirectory(testCaseDir)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
	// Unlink the local file.
	operations.RemoveFile(fh.Name())
	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(t, fh.Name())
	// Write to unlinked local file.
	operations.WriteWithoutClose(fh, FileContents, t)

	operations.SyncFileShouldThrowStaleHandleError(fh, t)
	operations.CloseFileShouldThrowStaleHandleError(fh, t)

	// Verify unlinked file is not present on GCS.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testCaseDir, FileName1, t)
}

func (s *staleFileHandleLocalFile) TestUnlinkedDirectoryContainingSyncedAndLocalFilesCloseThrowsStaleFileHandleError(t *testing.T) {
	t.Parallel() // Mark the test parallelizable.
	testCaseDir := "TestUnlinkedDirectoryContainingSyncedAndLocalFilesCloseThrowsStaleFileHandleError" + setup.GenerateRandomString(3)
	testDirPath = setup.SetupTestDirectory(testCaseDir)
	targetDir := path.Join(testDirPath, ExplicitDirName)
	fileName1 := path.Join(ExplicitDirName, ExplicitFileName1)
	fileName2 := path.Join(ExplicitDirName, ExplicitLocalFileName1)
	// Create explicit directory with one synced and one local file.
	operations.CreateDirectory(targetDir, t)
	CreateObjectInGCSTestDir(ctx, storageClient, testCaseDir, fileName1, "", t)
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName2, t)
	// Attempt to remove explicit directory.
	err := DeleteObjectOnGCS(ctx, storageClient, path.Join(testCaseDir, ExplicitDirName))
	// Verify rmDir operation succeeds.
	assert.Equal(t, nil, err)
	operations.ValidateNoFileOrDirError(t, targetDir)
	operations.ValidateNoFileOrDirError(t, path.Join(targetDir, ExplicitFileName1))
	operations.ValidateNoFileOrDirError(t, path.Join(targetDir, ExplicitLocalFileName1))
	// Validate writing content to unlinked local file does not throw error.
	operations.WriteWithoutClose(fh, FileContents, t)

	err = operations.CloseLocalFile(t, &fh)

	operations.ValidateStaleNFSFileHandleError(t, err)
	// Validate both local and synced files are deleted.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testCaseDir, ExplicitDirName, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testCaseDir, path.Join(ExplicitDirName, ExplicitFileName1), t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testCaseDir, path.Join(ExplicitDirName, ExplicitLocalFileName1), t)
}

func TestStaleFileHandleLocalFile(t *testing.T) {
	testCases := []struct {
		name  string
		flags []string
	}{
		{
			name:  "StaleFileHandleLocalFile",
			flags: []string{"--precondition-errors=true", "--metadata-cache-ttl-secs=0", "implicit-dirs=true"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts := &staleFileHandleLocalFile{
				flags: tc.flags,
			}
			log.Printf("Running tests with flags: %s", ts.flags)
			test_setup.RunTests(t, ts)
		})
	}
}
