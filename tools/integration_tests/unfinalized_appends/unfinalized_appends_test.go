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
	"io"
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
	// Mounting secondary mount
	setup.SetMntDir(gOtherRootDir)
	setup.SetLogFile(gOtherLogFilePath)
	setup.MountGCSFuseWithGivenMountFunc(gFlags, gMountFunc)
	// Mounting primary mount
	setup.SetMntDir(gRootDir)
	setup.SetLogFile(gLogFilePath)
	setup.MountGCSFuseWithGivenMountFunc(gFlags, gMountFunc)
}

func (t *UnfinalizedAppendsSuite) TearDownSuite() {
	setup.UnmountGCSFuse(gRootDir)
	setup.UnmountGCSFuse(gOtherRootDir)
	// Save both Log File in case of failure.
	setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
	setup.SetLogFile(gOtherLogFilePath)
	setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
}

func (t *UnfinalizedAppendsSuite) SetupTest() {
	t.fileName = FileName1 + setup.GenerateRandomString(5)
	var err error
	// Create unfinalized object of
	t.appendableWriter = CreateUnfinalizedObject(gCtx, t.T(), gStorageClient, path.Join(testDirName, t.fileName), SizeOfFileContents)
	t.fileSize = SizeOfFileContents

	t.appendFileHandle, err = os.OpenFile(path.Join(gTestDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(t.T(), err)
	initialContent, err := operations.ReadFile(path.Join(gTestDirPath, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), SizeOfFileContents, len(initialContent))
	t.initialContent = string(initialContent)
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

func (t *UnfinalizedAppendsSuite) TestReadWithSeparateHandleWhileAppending() {
	// This test validates that a separate read-only file handle can see
	// data being appended by an append handle on the same mount in real time.
	// The write-handle remains open throughout the test, to simulate append in progress.

	filePath := path.Join(gTestDirPath, t.fileName)
	expectedContent := t.initialContent

	// 1. First append using the suite's default write handle (opened in 'a' mode).
	appendContent1 := "appended content\n"
	_, err := t.appendFileHandle.WriteString(appendContent1)
	assert.NoError(t.T(), err)
	expectedContent += appendContent1

	// 2. Open a new, separate handle for reading ('r' mode).
	readHandle1, err := os.OpenFile(filePath, os.O_RDONLY|syscall.O_DIRECT, 0)
	require.NoError(t.T(), err)

	// 3. Read all content with the new handle and verify it sees the appended data.
	gotContent1, err := io.ReadAll(readHandle1)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedContent, string(gotContent1))

	// Close the read handle. The write handle remains open.
	require.NoError(t.T(), readHandle1.Close())

	// 4. Second append using the original, still-open write handle.
	appendContent2 := "appened content twice\n"
	_, err = t.appendFileHandle.WriteString(appendContent2)
	assert.NoError(t.T(), err)
	expectedContent += appendContent2

	// 5. Open another new handle for reading to verify again.
	readHandle2, err := os.OpenFile(filePath, os.O_RDONLY|syscall.O_DIRECT, 0)
	require.NoError(t.T(), err)

	// 6. Read and verify the final content.
	gotContent2, err := io.ReadAll(readHandle2)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedContent, string(gotContent2))
	require.NoError(t.T(), readHandle2.Close())

}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestUnfinalizedAppendsSuite(t *testing.T) {
	appendSuite := new(UnfinalizedAppendsSuite)
	suite.Run(t, appendSuite)
}
