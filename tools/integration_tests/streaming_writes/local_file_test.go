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

package streaming_writes

import (
	"path"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/local_file"
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

type streamingWritesLocalFileTestSuite struct {
	StreamingWritesSuite
	suite.Suite
}

func (t *streamingWritesLocalFileTestSuite) SetupTest() {
	t.createLocalFile()
}

func (t *streamingWritesLocalFileTestSuite) SetupSubTest() {
	t.createLocalFile()
}

func (t *streamingWritesLocalFileTestSuite) createLocalFile() {
	t.fileName = FileName1 + setup.GenerateRandomString(5)
	t.filePath = path.Join(testDirPath, t.fileName)
	// Create a local file with O_DIRECT.
	t.f1 = operations.OpenFileWithODirect(t.T(), t.filePath)
}

// Executes all tests that run with single streamingWrites configuration for localFiles.
func TestStreamingWritesLocalFileTestSuite(t *testing.T) {
	s := new(streamingWritesLocalFileTestSuite)
	s.StreamingWritesSuite.TestifySuite = &s.Suite
	suite.Run(t, s)
}

type existingLocalFileTestSuite struct {
	CommonLocalFileTestSuite
	suite.Suite
}

func (t *existingLocalFileTestSuite) SetupSuite() {
	SetCtx(ctx)
	SetStorageClient(storageClient)
	SetTestDirName(testDirName)

	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
}

func (t *existingLocalFileTestSuite) TearDownSuite() {
	setup.UnmountGCSFuse(rootDir)
	setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
}

// Executes all tests that run with single streamingWrites configuration for localFiles.
func TestExistingLocalFileTest(t *testing.T) {
	if !setup.IsZonalBucketRun() {
		s := new(existingLocalFileTestSuite)
		s.CommonLocalFileTestSuite.TestifySuite = &s.Suite
		suite.Run(t, s)
	}
}
