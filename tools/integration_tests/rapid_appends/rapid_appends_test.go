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
	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	// Create unfinalized object.
	t.fileContent = setup.GenerateRandomString(unfinalizedObjectSize)
	client.CreateUnfinalizedObject(ctx, t.T(), storageClient, path.Join(testDirName, t.fileName), t.fileContent)
}

func (t *RapidAppendsSuite) TearDownSubTest() {
	err := os.Remove(path.Join(primaryMntTestDirPath, t.fileName))
	require.NoError(t.T(), err)
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
	t.SetupSubTest()
	defer t.TearDownSubTest()

	// Initiate an append session using the primary file handle opened in append mode.
	appendFileHandle := operations.OpenFileInMode(t.T(), path.Join(primaryMntTestDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
	defer func() {
		// Closing file handle purely for resource cleanup.
		// FlushFile() triggered on Close() is expected to error out with stale NFS file handle.
		operations.CloseFileShouldThrowError(t.T(), appendFileHandle)
	}()
	t.appendToFile(appendFileHandle, initialContent)

	// Open a new file handle from the secondary mount to the same file.
	newAppendFileHandle := operations.OpenFileInMode(t.T(), path.Join(secondaryMntTestDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
	defer operations.CloseFileShouldNotThrowError(t.T(), newAppendFileHandle)

	// Attempt to append using the newly opened file handle.
	// This append should succeed, confirming the takeover.
	_, err := newAppendFileHandle.WriteString(appendContent)
	assert.NoError(t.T(), err)

	// Attempt to append using the original file handle.
	// This should now fail, as its append session has been invalidated by the takeover.
	_, _ = appendFileHandle.WriteString(appendContent)
	err = appendFileHandle.Sync()
	assert.Error(t.T(), err)

	// Syncing from the newly created file handle must succeed since it holds the active
	// append session.
	err = newAppendFileHandle.Sync()
	assert.NoError(t.T(), err)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestRapidAppendsSuite(t *testing.T) {
	rapidAppendsSuite := new(RapidAppendsSuite)
	suite.Run(t, rapidAppendsSuite)
}
