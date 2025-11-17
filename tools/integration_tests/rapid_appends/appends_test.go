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

package rapid_appends

import (
	"os"
	"path"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Tests for the DualMountAppendsTestSuite
////////////////////////////////////////////////////////////////////////

func (t *DualMountAppendsTestSuite) TestAppendSessionInvalidatedByAnotherClientUponTakeover() {
	const initialContent = "dummy content"
	const appendContent = "appended content"

	t.createUnfinalizedObject()
	defer t.deleteUnfinalizedObject()

	// Initiate an append session using the primary file handle.
	appendFileHandle := operations.OpenFileInMode(t.T(), path.Join(t.primaryMount.testDirPath, t.fileName), fileOpenModeAppend|syscall.O_DIRECT)
	n, err := appendFileHandle.WriteString(initialContent)
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(initialContent), n)

	// Open a new file handle from the secondary mount to the same file.
	newAppendFileHandle := operations.OpenFileInMode(t.T(), path.Join(t.secondaryMount.testDirPath, t.fileName), fileOpenModeAppend|syscall.O_DIRECT)
	defer operations.CloseFileShouldNotThrowError(t.T(), newAppendFileHandle)

	// This append should succeed, confirming the takeover.
	n, err = newAppendFileHandle.WriteString(appendContent)
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(appendContent), n)

	// This should now fail, as its append session has been invalidated.
	_, _ = appendFileHandle.WriteString(appendContent)
	err = appendFileHandle.Sync()
	operations.ValidateESTALEError(t.T(), err)

	// Syncing from the new handle must succeed.
	err = newAppendFileHandle.Sync()
	require.NoError(t.T(), err)

	// Read directly using storage client to validate the contents which has persisted in
	// GCS after takeover from the secondary mount.
	// Close the open append handle before issuing read on the file as Sync() triggered on
	// ReadFile() due to BWH still being initialized, is expected to error out with stale NFS file handle.
	operations.CloseFileShouldThrowError(t.T(), appendFileHandle)
	expectedContent := t.fileContent + appendContent
	content, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), expectedContent, string(content))
}

////////////////////////////////////////////////////////////////////////
// Tests for the SingleMountAppendsTestSuite
////////////////////////////////////////////////////////////////////////

func (t *SingleMountAppendsTestSuite) TestContentAppendedInNonAppendModeNotVisibleTillClose() {
	// Initially create an unfinalized object.
	t.createUnfinalizedObject()
	defer t.deleteUnfinalizedObject()

	initialContent := t.fileContent
	wh, err := os.OpenFile(path.Join(t.primaryMount.testDirPath, t.fileName), fileOpenModeRPlus|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(t.T(), err)

	// Write sufficient data to the end of file.
	data := setup.GenerateRandomString(contentSizeForBW * operations.OneMiB)
	n, err := wh.WriteAt([]byte(data), int64(len(initialContent)))
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(data), n)

	// Read from GCS to validate that appended content is not yet visible.
	contentBeforeClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), initialContent, string(contentBeforeClose))

	// Close the file handle to persist the data.
	err = wh.Close()
	require.NoError(t.T(), err)

	// Validate that the appended content is now visible in GCS.
	expectedContent := initialContent + data
	contentAfterClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), expectedContent, string(contentAfterClose))
}

func (t *SingleMountAppendsTestSuite) TestAppendsToFinalizedObjectNotVisibleUntilClose() {
	const initialContent = "dummy content"

	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	// Create Finalized Object in the GCS bucket.
	client.CreateFinalizedObjectInGCSTestDir(
		ctx, storageClient, testDirName, t.fileName, initialContent, t.T())

	// Append to the finalized object from the primary mount.
	data := setup.GenerateRandomString(contentSizeForBW * operations.OneMiB)
	filePath := path.Join(t.primaryMount.testDirPath, t.fileName)
	fh, err := os.OpenFile(filePath, fileOpenModeAppend|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(t.T(), err)
	n, err := fh.Write([]byte(data))
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(data), n)

	// Read from GCS to validate appended content is not yet visible.
	contentBeforeClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), initialContent, string(contentBeforeClose))

	// Close the file handle and verify appended content is now visible.
	require.NoError(t.T(), fh.Close())
	expectedContent := initialContent + data
	contentAfterClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), expectedContent, string(contentAfterClose))
}

func (t *SingleMountAppendsTestSuite) TestAppendsVisibleInRealTimeWithConcurrentRPlusHandle() {
	const initialContent = "dummy content"

	// Initially create an unfinalized object.
	t.createUnfinalizedObject()
	defer t.deleteUnfinalizedObject()

	primaryPath := path.Join(t.primaryMount.testDirPath, t.fileName)
	// Open first handle in append mode.
	appendFileHandle := operations.OpenFileInMode(t.T(), primaryPath, fileOpenModeAppend|syscall.O_DIRECT)
	defer appendFileHandle.Close()
	readHandle := operations.OpenFileInMode(t.T(), primaryPath, fileOpenModeRPlus|syscall.O_DIRECT)
	defer readHandle.Close()

	// Write initial content with append handle to trigger buffered write workflow.
	n, err := appendFileHandle.Write([]byte(initialContent))
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(initialContent), n)

	// Append additional content with the "r+" handle.
	data := setup.GenerateRandomString(contentSizeForBW * blockSize)
	appendOffset := int64(unfinalizedObjectSize + len(initialContent))
	n, err = readHandle.WriteAt([]byte(data), appendOffset)
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(data), n)

	// Read from back-door to validate visibility on GCS.
	// The first 1MiB block is guaranteed to be flushed due to implicit behavior.
	// That block includes both the initial content (written via "a" file handle )
	// and some part of data written by the "r+" file handle.
	dataInBlockOffset := blockSize - len(initialContent)
	expectedContent := t.fileContent + initialContent + data[0:dataInBlockOffset]
	contentRead, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
	require.NoError(t.T(), err)
	require.GreaterOrEqual(t.T(), len(contentRead), len(expectedContent))
	assert.Equal(t.T(), expectedContent, string(contentRead[0:len(expectedContent)]))
}

func (t *SingleMountAppendsTestSuite) TestRandomWritesVisibleAfterCloseWithConcurrentRPlusHandle() {
	const initialContent = "dummy content"
	t.createUnfinalizedObject()
	defer t.deleteUnfinalizedObject()

	primaryPath := path.Join(t.primaryMount.testDirPath, t.fileName)
	appendFileHandle := operations.OpenFileInMode(t.T(), primaryPath, fileOpenModeAppend|syscall.O_DIRECT)
	defer appendFileHandle.Close()
	readHandle := operations.OpenFileInMode(t.T(), primaryPath, fileOpenModeRPlus|syscall.O_DIRECT)

	n, err := appendFileHandle.Write([]byte(initialContent))
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(initialContent), n)
	t.fileContent = t.fileContent + initialContent

	// Random write at an incorrect offset.
	data := setup.GenerateRandomString(contentSizeForBW * blockSize)
	n, err = readHandle.WriteAt([]byte(data), int64(len(t.fileContent))+1)
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(data), n)

	// Validate content is not yet visible.
	contentBeforeClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), t.fileContent, string(contentBeforeClose))

	// Close handle and validate final content (with null byte for the gap).
	readHandle.Close()
	expectedContent := t.fileContent + "\x00" + data
	contentAfterClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), expectedContent, string(contentAfterClose))
}

func (t *SingleMountAppendsTestSuite) TestFallbackHappensWhenNonAppendHandleDoesFirstWrite() {
	// Initially create an unfinalized object.
	t.createUnfinalizedObject()
	defer t.deleteUnfinalizedObject()

	primaryPath := path.Join(t.primaryMount.testDirPath, t.fileName)
	appendFileHandle := operations.OpenFileInMode(t.T(), primaryPath, fileOpenModeAppend|syscall.O_DIRECT)
	defer appendFileHandle.Close()
	readHandle := operations.OpenFileInMode(t.T(), primaryPath, fileOpenModeRPlus|syscall.O_DIRECT)

	// Append content using the "r+" handle first.
	data := setup.GenerateRandomString(contentSizeForBW * blockSize)
	n, err := readHandle.WriteAt([]byte(data), int64(len(t.fileContent)))
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(data), n)

	// Validate content is not yet visible.
	contentBeforeClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), t.fileContent, string(contentBeforeClose))

	// Close handle and validate final content.
	readHandle.Close()
	expectedContent := t.fileContent + data
	contentAfterClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), expectedContent, string(contentAfterClose))
}

func (t *SingleMountAppendsTestSuite) TestKernelShouldSeeUpdatedSizeOnAppends_ValidStatCache() {
	const initialContent = "dummy content"

	t.createUnfinalizedObject()
	defer t.deleteUnfinalizedObject()
	filePath := path.Join(t.primaryMount.testDirPath, t.fileName)

	// Append to the object and close the file handle.
	appendFileHandle := operations.OpenFileInMode(t.T(), filePath, fileOpenModeAppend|syscall.O_DIRECT)
	n, err := appendFileHandle.Write([]byte(initialContent))
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(initialContent), n)
	appendFileHandle.Close()

	// Since we don't wait for the cache to expire, the stat should reflect the new size immediately.
	expectedFileSize := int64(unfinalizedObjectSize + len(initialContent))
	fileInfo, err := operations.StatFile(filePath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), expectedFileSize, (*fileInfo).Size())
}

func (t *SingleMountAppendsTestSuite) TestKernelShouldSeeUpdatedSizeOnAppends_ExpiredStatCache() {
	const initialContent = "dummy content"

	t.createUnfinalizedObject()
	defer t.deleteUnfinalizedObject()
	filePath := path.Join(t.primaryMount.testDirPath, t.fileName)

	// Append to the object and close the file handle.
	appendFileHandle := operations.OpenFileInMode(t.T(), filePath, fileOpenModeAppend|syscall.O_DIRECT)
	n, err := appendFileHandle.Write([]byte(initialContent))
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(initialContent), n)
	appendFileHandle.Close()

	// Expire stat cache. By default, stat cache ttl is 60 seconds.
	time.Sleep(defaultMetadataCacheTTL)

	// The stat should now fetch the latest size from the source, reflecting the new size.
	expectedFileSize := int64(unfinalizedObjectSize + len(initialContent))
	fileInfo, err := operations.StatFile(filePath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), expectedFileSize, (*fileInfo).Size())
}

////////////////////////////////////////////////////////////////////////
// Test Runner
////////////////////////////////////////////////////////////////////////

// appendTestConfigs defines the matrix of configurations for the AppendsTestSuite.
var appendTestConfigs = []*testConfig{
	{
		name:              "SingleMount",
		isDualMount:       false,
		primaryMountFlags: []string{"--write-block-size-mb=1"},
	},
	{
		name:                "DualMount",
		isDualMount:         true,
		primaryMountFlags:   []string{"--write-block-size-mb=1"},
		secondaryMountFlags: []string{"--write-block-size-mb=1"},
	},
}

// TestAppendsSuiteRunner executes all general append tests against the appendTestConfigs matrix.
func TestAppendsSuiteRunner(t *testing.T) {
	for _, cfg := range appendTestConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			if cfg.isDualMount {
				suite.Run(t, &DualMountAppendsTestSuite{BaseSuite{cfg: cfg}})
			} else {
				suite.Run(t, &SingleMountAppendsTestSuite{BaseSuite{cfg: cfg}})
			}
		})
	}
}
