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

	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const numAppends = 3

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

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

func (t *RapidAppendsSuite) SetupTest() {
	t.fileName = FileName1 + setup.GenerateRandomString(5)
	// Create unfinalized object.
	_ = CreateUnfinalizedObject(ctx, t.T(), storageClient, path.Join(testDirName, t.fileName), SizeOfFileContents)
	initialContent, err := operations.ReadFile(path.Join(primaryMntTestDirPath, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), SizeOfFileContents, len(initialContent))
	t.fileContent = string(initialContent)
}

func (t *RapidAppendsSuite) TearDownTest() {
	err := os.Remove(path.Join(primaryMntTestDirPath, t.fileName))
	require.NoError(t.T(), err)
}

// appendToFile appends "appendContent" to the given file.
func (t *RapidAppendsSuite) appendToFile(file *os.File, appendContent string) {
	t.T().Helper()
	n, err := file.WriteString(appendContent)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), len(appendContent), n)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *RapidAppendsSuite) TestAppendsAndSequentialReadFromSameMount() {
	appendFileHandle := operations.OpenFileInMode(t.T(), path.Join(primaryMntTestDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
	defer func() {
		err := appendFileHandle.Close()
		require.NoError(t.T(), err)
	}()
	expectedContent := t.fileContent
	for range numAppends {
		t.appendToFile(appendFileHandle, FileContents)
		expectedContent += FileContents

		// Read content of file from same mount.
		gotContent, err := operations.ReadFile(path.Join(primaryMntTestDirPath, t.fileName))

		require.NoError(t.T(), err)
		assert.Equal(t.T(), expectedContent, string(gotContent))
	}
}

func (t *RapidAppendsSuite) TestAppendsAndSequentialReadFromDifferentMount() {
	appendFileHandle := operations.OpenFileInMode(t.T(), path.Join(primaryMntTestDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
	defer func() {
		err := appendFileHandle.Close()
		require.NoError(t.T(), err)
	}()
	expectedContent := t.fileContent
	for range numAppends {
		t.appendToFile(appendFileHandle, FileContents)
		expectedContent += FileContents
		operations.SyncFile(appendFileHandle, t.T())

		// Read content of file from differnt mount.
		gotContent, err := operations.ReadFile(path.Join(secondaryMntTestDirPath, t.fileName))

		require.NoError(t.T(), err)
		assert.Equal(t.T(), expectedContent, string(gotContent))
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestRapidAppendsSuite(t *testing.T) {
	rapidAppendsSuite := new(RapidAppendsSuite)
	suite.Run(t, rapidAppendsSuite)
}
