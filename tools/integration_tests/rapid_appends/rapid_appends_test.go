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

// Number of appends to perform on test file.
const numAppends = 3

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

// TODO: Split the suite in two suites single mount and multi-mount.
type RapidAppendsSuite struct {
	suite.Suite
	fileName         string
	fileContent      string
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
func (t *RapidAppendsSuite) createUnfinalizedObject() {
	t.fileName = client.FileName1 + setup.GenerateRandomString(5)
	// Create unfinalized object.
	t.fileContent = setup.GenerateRandomString(client.SizeOfFileContents)
	_ = client.CreateUnfinalizedObject(ctx, t.T(), storageClient, path.Join(testDirName, t.fileName), t.fileContent)
}

func (t *RapidAppendsSuite) SetupTest() {
	t.createUnfinalizedObject()
}

func (t *RapidAppendsSuite) SetupSubTest() {
	t.createUnfinalizedObject()
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
			for range numAppends {
				t.appendToFile(appendFileHandle, client.FileContents)
				// Sync the file if the test case requires it.
				if tc.syncNeeded {
					operations.SyncFile(appendFileHandle, t.T())
				}

				readPath := path.Join(tc.readMountPath, t.fileName)
				gotContent, err := operations.ReadFile(readPath)

				require.NoError(t.T(), err)
				assert.Equal(t.T(), t.fileContent, string(gotContent))
			}
		})
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestRapidAppendsSuite(t *testing.T) {
	rapidAppendsSuite := new(RapidAppendsSuite)
	suite.Run(t, rapidAppendsSuite)
}
