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
	"log"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const numAppends = 3             // Number of appends to perform on test file.
const appendSize = 10            // Size in bytes for each append.
const unfinalizedObjectSize = 10 // Size in bytes of initial unfinalized Object.

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

// TODO: Split the suite in two suites single mount and multi-mount.
type RapidAppendsSuite struct {
	suite.Suite
	fileName    string
	fileContent string
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (t *RapidAppendsSuite) SetupSuite() {
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	secondaryMntTestDirPath = setup.SetupTestDirectory(testDirName)
}

func (t *RapidAppendsSuite) TearDownSuite() {
	setup.UnmountGCSFuse(secondaryMntRootDir)
	if t.T().Failed() {
		log.Println("Secondary mount log file:")
		setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
		log.Println("Primary mount log file:")
		setup.SetLogFile(primaryMntLogFilePath)
		setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
	}
}

func (t *RapidAppendsSuite) SetupSubTest() {
	t.createUnfinalizedObject()
}

func (t *RapidAppendsSuite) SetupTest() {
	t.createUnfinalizedObject()
}

func (t *RapidAppendsSuite) createUnfinalizedObject() {
	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	// Create unfinalized object.
	t.fileContent = setup.GenerateRandomString(unfinalizedObjectSize)
	client.CreateUnfinalizedObject(ctx, t.T(), storageClient, path.Join(testDirName, t.fileName), t.fileContent)
}

// appendToFile appends "appendContent" to the given file.
func (t *RapidAppendsSuite) appendToFile(file *os.File, appendContent string) {
	t.T().Helper()
	n, err := file.WriteString(appendContent)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), len(appendContent), n)
	t.fileContent += appendContent
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *RapidAppendsSuite) TestAppendsAndRead() {
	testCases := []struct {
		name          string
		readMountPath string
		syncNeeded    bool
	}{
		{
			name:          "reading_seq_from_same_mount",
			readMountPath: primaryMntTestDirPath,
			syncNeeded:    false, // Sync is not required when reading from the same mount.
		},
		{
			name:          "reading_seq_from_different_mount",
			readMountPath: secondaryMntTestDirPath,
			syncNeeded:    true, // Sync is required for writes to be visible on another mount.
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			// Open the file for appending on the primary mount.
			appendFileHandle := operations.OpenFileInMode(t.T(), path.Join(primaryMntTestDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
			defer operations.CloseFileShouldNotThrowError(t.T(), appendFileHandle)
			readPath := path.Join(tc.readMountPath, t.fileName)
			for range numAppends {
				t.appendToFile(appendFileHandle, setup.GenerateRandomString(appendSize))
				// Sync the file if the test case requires it.
				if tc.syncNeeded {
					operations.SyncFile(appendFileHandle, t.T())
				}

				gotContent, err := operations.ReadFile(readPath)

				require.NoError(t.T(), err)
				assert.Equal(t.T(), t.fileContent, string(gotContent))
			}
		})
	}
}

func (t *RapidAppendsSuite) TestAppendSessionInvalidatedByAnotherClientUponTakeover() {
	// Initiate an append session using the primary file handle opened in append mode.
	appendFileHandle := operations.OpenFileInMode(t.T(), path.Join(primaryMntTestDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
	_, err := appendFileHandle.WriteString(initialContent)
	require.NoError(t.T(), err)

	// Open a new file handle from the secondary mount to the same file.
	newAppendFileHandle := operations.OpenFileInMode(t.T(), path.Join(secondaryMntTestDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
	defer operations.CloseFileShouldNotThrowError(t.T(), newAppendFileHandle)

	// Attempt to append using the newly opened file handle.
	// This append should succeed, confirming the takeover.
	_, err = newAppendFileHandle.WriteString(appendContent)
	assert.NoError(t.T(), err)

	// Attempt to append using the original file handle.
	// This should now fail, as its append session has been invalidated by the takeover.
	_, _ = appendFileHandle.WriteString(appendContent)
	err = appendFileHandle.Sync()
	operations.ValidateESTALEError(t.T(), err)

	// Syncing from the newly created file handle must succeed since it holds the active
	// append session.
	err = newAppendFileHandle.Sync()
	assert.NoError(t.T(), err)

	// Read from primary mount to validate the contents which has persisted in GCS after
	// takeover from the secondary mount.
	// Close the open append handle before issuing read on the file as Sync() triggered on
	// ReadFile() due to BWH still being initialized, is expected to error out with stale NFS file handle.
	operations.CloseFileShouldThrowError(t.T(), appendFileHandle)
	expectedContent := t.fileContent + appendContent
	content, err := operations.ReadFile(path.Join(primaryMntTestDirPath, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), expectedContent, string(content))
}

func (t *RapidAppendsSuite) TestContentAppendedInNonAppendModeNotVisibleTillClose() {
	// Skipping test for now until CreateObject() is supported for unfinalized objects.
	// Ref: b/424253611
	t.T().Skip()

	initialContent := t.fileContent
	// Append to the file from the primary mount in non-append mode
	wh, err := os.OpenFile(path.Join(primaryMntTestDirPath, t.fileName), os.O_WRONLY|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(t.T(), err)

	// Write sufficient data to the end of file.
	data := setup.GenerateRandomString(contentSizeForBW * operations.OneMiB)
	n, err := wh.WriteAt([]byte(data), int64(len(initialContent)))
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(data), n)

	// Read from back-door to validate that appended content is yet not visible on GCS.
	contentBeforeClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), initialContent, contentBeforeClose)

	// Close() from primary mount to ensure data persists in GCS.
	err = wh.Close()
	require.NoError(t.T(), err)

	// Validate that appended content is visible in GCS.
	expectedContent := initialContent + data
	contentAfterClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(path.Join(testDirName, t.fileName)))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), expectedContent, contentAfterClose)
}

func (t *RapidAppendsSuite) TestAppendsToFinalizedObjectNotVisibleUntilClose() {
	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	// Create Finalized Object in the GCS bucket.
	client.CreateObjectInGCSTestDir(
		ctx, storageClient, testDirName, t.fileName, initialContent, t.T())
	defer func() {
		err := os.Remove(path.Join(primaryMntTestDirPath, t.fileName))
		require.NoError(t.T(), err)
	}()

	// Append to the finalized object from the primary mount.
	filePath := path.Join(primaryMntTestDirPath, t.fileName)
	fh, err := os.OpenFile(filePath, os.O_APPEND|os.O_RDWR|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(t.T(), err)
	_, err = fh.Write([]byte(appendContent))
	require.NoError(t.T(), err)

	// Read the object from secondary mount to validate that appended content is yet not visible on GCS.
	secondaryPath := path.Join(secondaryMntTestDirPath, t.fileName)
	contentBeforeClose, err := operations.ReadFile(secondaryPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), initialContent, string(contentBeforeClose))

	// Close the file handle used for appending.
	require.NoError(t.T(), fh.Close())

	// Read the object from secondary mount to validate that appended content is now visible on GCS.
	expectedContent := initialContent + appendContent
	contentAfterClose, err := operations.ReadFile(secondaryPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), expectedContent, string(contentAfterClose))
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestRapidAppendsSuite(t *testing.T) {
	rapidAppendsSuite := new(RapidAppendsSuite)
	suite.Run(t, rapidAppendsSuite)
}
