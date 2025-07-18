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

package unfinalized_appends

import (
	"log"
	"os"
	"path"
	"syscall"
	"testing"

	"cloud.google.com/go/storage"
	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type UnfinalizedAppendsSuite struct {
	appendFileHandle *os.File
	appendableWriter *storage.Writer
	fileName         string
	fileSize         int
	initialContent   string
	suite.Suite
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (t *UnfinalizedAppendsSuite) SetupSuite() {
	setup.MountGCSFuseWithGivenMountFunc(gFlags, gMountFunc)
	gRootDir = setup.MntDir()
	gLogFilePath = setup.LogFile()
	gTestDirPath = setup.SetupTestDirectory(testDirName)
}

func (t *UnfinalizedAppendsSuite) TearDownSuite() {
	setup.UnmountGCSFuse(gRootDir)
	setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
	if t.T().Failed() {
		log.Println("Secondary mount log file:")
		setup.SetLogFile(gOtherLogFilePath)
		setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
	}
}

func (t *UnfinalizedAppendsSuite) SetupTest() {
	t.fileName = FileName1 + setup.GenerateRandomString(5)
	var err error
	// Create unfinalized object.
	t.appendableWriter = CreateUnfinalizedObject(gCtx, t.T(), gStorageClient, path.Join(testDirName, t.fileName), SizeOfFileContents)
	t.fileSize = SizeOfFileContents

	t.appendFileHandle, err = os.OpenFile(path.Join(gTestDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(t.T(), err)
	initialContent, err := operations.ReadFile(path.Join(gTestDirPath, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), SizeOfFileContents, len(initialContent))
	t.initialContent = string(initialContent)
}

func (t *UnfinalizedAppendsSuite) TearDownTest() {
	t.appendableWriter.Close()
	t.appendFileHandle.Close()
}

// AppendToFile appends "appendContent" using
// existing appendFileHandle in the first mount.
func (t *UnfinalizedAppendsSuite) AppendToFile(appendContent string) {
	n, err := t.appendFileHandle.WriteString(appendContent)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), len(appendContent), n)
	t.fileSize += n
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *UnfinalizedAppendsSuite) TestAppendsFromSameMount() {
	expectedContent := t.initialContent
	for range 3 {
		t.AppendToFile(FileContents)
		expectedContent += FileContents

		// Read content of file from same mount.
		gotContent, err := operations.ReadFile(path.Join(gTestDirPath, t.fileName))

		assert.NoError(t.T(), err)
		assert.Equal(t.T(), t.fileSize, len(gotContent))
		assert.Equal(t.T(), expectedContent, string(gotContent))
	}
}

func (t *UnfinalizedAppendsSuite) TestAppendsFromDifferentMount() {
	expectedContent := t.initialContent
	for range 3 {
		t.AppendToFile(FileContents)
		expectedContent += FileContents
		operations.SyncFile(t.appendFileHandle, t.T())

		// Read content of file from differnt mount.
		gotContent, err := operations.ReadFile(path.Join(gOtherTestDirPath, t.fileName))

		assert.NoError(t.T(), err)
		assert.Equal(t.T(), t.fileSize, len(gotContent))
		assert.Equal(t.T(), expectedContent, string(gotContent))
	}
}

func (t *UnfinalizedAppendsSuite) TestAppendSessionInvalidatedByAnotherClientUponTakeover() {
	const contentToAppend = "appended content"
	// Initiate an append session using the suite's primary file handle opened in append mode.
	t.AppendToFile(FileContents)

	// Open a new file handle from a different mount to the same file.
	newAppendFileHandle, err := os.OpenFile(path.Join(gOtherTestDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(t.T(), err)
	defer func() {
		// Ensure that the new file handle is closed, regardless of test outcome.
		assert.NoError(t.T(), newAppendFileHandle.Close())
	}()

	// Attempt to append using the newly opened file handle.
	// This append should succeed, confirming the takeover.
	_, err = newAppendFileHandle.WriteString(contentToAppend)
	assert.NoError(t.T(), err)

	// Attempt to append using the original file handle.
	// This should now fail, as its append session has been invalidated by the takeover.
	_, err = t.appendFileHandle.WriteString(contentToAppend)
	err = t.appendFileHandle.Sync()
	assert.Error(t.T(), err)

	// Syncing from the newly created file handle must succeed since it holds the active
	// append session.
	err = newAppendFileHandle.Sync()
	assert.NoError(t.T(), err)
}

func (t *UnfinalizedAppendsSuite) TestContentAppendedInNonAppendModeNotVisibleTillClose() {
	// Skipping test for now until CreateObject() is supported for unfinalized objects.
	// Ref: b/424253611
	t.T().Skip()
	fileName := "append_obj_" + setup.GenerateRandomString(5)
	_ = CreateUnfinalizedObject(gCtx, t.T(), gStorageClient, path.Join(testDirName, fileName), SizeOfFileContents)
	// TODO(anushkadhn): Close the writer created above after FinalizeOnClose for writer is set to false by default.

	initialContent, err := operations.ReadFile(path.Join(gTestDirPath, fileName))
	require.NoError(t.T(), err)
	require.NotEmpty(t.T(), initialContent)

	// Append to the file from the primary mount in non-append mode
	wh, err := os.OpenFile(path.Join(gTestDirPath, fileName), os.O_WRONLY|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(t.T(), err)
	appendContent := []byte("Appended Content")
	n, err := wh.WriteAt(appendContent, int64(len(initialContent)))
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(appendContent), n)

	// Read from secondary mount to validate that data is not visible in GCS in realtime
	contentBeforeClose, err := operations.ReadFile(path.Join(gOtherTestDirPath, fileName))
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), initialContent, contentBeforeClose)

	// Close() from primary mount to ensure data persists in GCS.
	err = wh.Close()
	require.NoError(t.T(), err)

	// Read from secondary mount to validate that data is now visible.
	expectedContent := append(initialContent, appendContent...)
	contentAfterClose, err := operations.ReadFile(path.Join(gOtherTestDirPath, fileName))
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedContent, contentAfterClose)
}

func (t *UnfinalizedAppendsSuite) TestAppendedDataNotVisibleUntilClose() {
	fileName := "append_obj_" + setup.GenerateRandomString(5)
	initialContent := []byte("dummy content")
	CreateObjectInGCSTestDir(
		gCtx, gStorageClient, testDirName, fileName, string(initialContent), t.T())

	// Append to the finalized object from the primary mount.
	filePath := path.Join(gTestDirPath, fileName)
	fh, err := os.OpenFile(filePath, os.O_APPEND|os.O_RDWR|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(t.T(), err, "failed to open file in append mode")
	appendedContent := []byte("appended content")
	_, err = fh.Write(appendedContent)
	require.NoError(t.T(), err, "failed to append to file")

	// Read the object from secondary mount to validate that appended content is yet not visible on GCS.
	secondaryPath := path.Join(gOtherTestDirPath, fileName)
	contentBeforeClose, err := operations.ReadFile(secondaryPath)
	require.NoError(t.T(), err, "failed to read file before close")
	assert.Equal(t.T(), initialContent, contentBeforeClose, "appended data should not be visible before close")

	// Close the file handle used for appending.
	require.NoError(t.T(), fh.Close(), "failed to close file")

	// Read the object from secondary mount to validate that appended content is now visible on GCS.
	expectedContent := append(initialContent, appendedContent...)
	contentAfterClose, err := operations.ReadFile(secondaryPath)
	require.NoError(t.T(), err, "failed to read file after close")
	assert.Equal(t.T(), expectedContent, contentAfterClose, "appended data should be visible after close")
}

func (t *UnfinalizedAppendsSuite) TestAppendsVisibleInRealTimeWithConcurrentRPlusHandle() {
	fileName := "append_obj_" + setup.GenerateRandomString(5)
	fullPath := path.Join(testDirName, fileName)

	// Create an unfinalized object
	_ = CreateUnfinalizedObject(gCtx, t.T(), gStorageClient, fullPath, 0)

	primaryPath := path.Join(gTestDirPath, fileName)

	// Open first handle in append mode.
	ah, err := os.OpenFile(primaryPath, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(t.T(), err)
	defer ah.Close()

	// Open second handle in "r+" mode.
	rh, err := os.OpenFile(primaryPath, os.O_RDWR, operations.FilePermission_0600)
	require.NoError(t.T(), err)
	defer rh.Close()

	// Write initial content using append handle to trigger BW workflow.
	initialContent := []byte("dummy content")
	n, err := ah.Write(initialContent)
	require.NoError(t.T(), err)
	require.NotZero(t.T(), n)

	// Append additional content using "r+" handle.
	appendedContent := []byte("appended content")
	_, err = rh.WriteAt(appendedContent, int64(len(initialContent)))
	require.NoError(t.T(), err)

	// Sync changes
	operations.SyncFile(rh, t.T())

	// Read back content from secondary mount to validate visibility.
	secondaryPath := path.Join(gOtherTestDirPath, fileName)
	readContent, err := operations.ReadFile(secondaryPath)
	require.NoError(t.T(), err)
	expectedContent := append(initialContent, appendedContent...)
	assert.Equal(t.T(), expectedContent, readContent)
}

func (t *UnfinalizedAppendsSuite) TestRandomWritesVisibleAfterCloseWithConcurrentRPlusHandle() {
	// Skipping test for now until CreateObject() is supported for unfinalized objects.
	// Ref: b/424253611
	t.T().Skip()
	fileName := "append_obj_" + setup.GenerateRandomString(5)
	fullPath := path.Join(testDirName, fileName)

	// Create an unfinalized object
	_ = CreateUnfinalizedObject(gCtx, t.T(), gStorageClient, fullPath, 0)

	primaryPath := path.Join(gTestDirPath, fileName)

	// Open first handle in append mode.
	ah, err := os.OpenFile(primaryPath, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(t.T(), err)
	defer ah.Close()

	// Open second handle in "r+" mode.
	rh, err := os.OpenFile(primaryPath, os.O_RDWR, operations.FilePermission_0600)
	require.NoError(t.T(), err)

	// Write initial content using append handle to trigger BW workflow.
	initialContent := []byte("dummy content")
	n, err := ah.Write(initialContent)
	require.NoError(t.T(), err)
	require.NotZero(t.T(), n)

	// Random write additional content using "r+" handle.
	appendedContent := []byte("appended content")
	_, err = rh.WriteAt(appendedContent, int64(len(initialContent))+1)
	require.NoError(t.T(), err)

	// Read back content from secondary mount to validate data not visible before close().
	secondaryPath := path.Join(gOtherTestDirPath, fileName)
	contentBeforeClose, err := operations.ReadFile(secondaryPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), initialContent, contentBeforeClose)

	// Close the file handle.
	rh.Close()

	// Read back content from secondary mount to validate data visible after close().
	contentAfterClose, err := operations.ReadFile(secondaryPath)
	require.NoError(t.T(), err)
	expectedContent := append(initialContent, appendedContent...)
	assert.Equal(t.T(), expectedContent, contentAfterClose)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestUnfinalizedAppendsSuite(t *testing.T) {
	appendSuite := new(UnfinalizedAppendsSuite)
	suite.Run(t, appendSuite)
}
