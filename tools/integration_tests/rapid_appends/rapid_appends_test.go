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

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type RapidAppendsSuite struct {
	appendFileHandle *os.File
	fileName         string
	fileSize         int
	initialContent   string
	suite.Suite
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (t *RapidAppendsSuite) SetupSuite() {
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	rootDir = setup.MntDir()
	logFilePath = setup.LogFile()
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func (t *RapidAppendsSuite) TearDownSuite() {
	setup.UnmountGCSFuse(rootDir)
	setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
	if t.T().Failed() {
		log.Println("Secondary mount log file:")
		setup.SetLogFile(otherLogFilePath)
		setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
	}
}

func (t *RapidAppendsSuite) SetupTest() {
	t.fileName = FileName1 + setup.GenerateRandomString(5)
	// Create unfinalized object.
	_ = CreateUnfinalizedObject(ctx, t.T(), storageClient, path.Join(testDirName, t.fileName), SizeOfFileContents)
	t.fileSize = SizeOfFileContents
	// Create append file handle.
	var err error
	t.appendFileHandle, err = os.OpenFile(path.Join(testDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(t.T(), err)
	initialContent, err := operations.ReadFile(path.Join(testDirPath, t.fileName))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), SizeOfFileContents, len(initialContent))
	t.initialContent = string(initialContent)
}

func (t *RapidAppendsSuite) TearDownTest() {
	err := t.appendFileHandle.Close()
	require.NoError(t.T(), err)
}

// AppendToFile appends "appendContent" using
// existing appendFileHandle in the first mount.
func (t *RapidAppendsSuite) AppendToFile(appendContent string) {
	n, err := t.appendFileHandle.WriteString(appendContent)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), len(appendContent), n)
	t.fileSize += n
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *RapidAppendsSuite) TestAppendsAndSequentialReadFromSameMount() {
	expectedContent := t.initialContent
	for range 3 {
		t.AppendToFile(FileContents)
		expectedContent += FileContents

		// Read content of file from same mount.
		gotContent, err := operations.ReadFile(path.Join(testDirPath, t.fileName))

		require.NoError(t.T(), err)
		assert.Equal(t.T(), expectedContent, string(gotContent))
	}
}

func (t *RapidAppendsSuite) TestAppendsAndSequentialReadFromDifferentMount() {
	expectedContent := t.initialContent
	for range 3 {
		t.AppendToFile(FileContents)
		expectedContent += FileContents
		operations.SyncFile(t.appendFileHandle, t.T())

		// Read content of file from differnt mount.
		gotContent, err := operations.ReadFile(path.Join(otherTestDirPath, t.fileName))

		require.NoError(t.T(), err)
		assert.Equal(t.T(), expectedContent, string(gotContent))
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestUnfinalizedAppendsSuite(t *testing.T) {
	appendSuite := new(RapidAppendsSuite)
	suite.Run(t, appendSuite)
}
