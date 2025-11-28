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

package inactive_stream_timeout

import (
	"context"
	"log"
	"path"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type timeoutEnabledSuite struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	baseTestName  string
	suite.Suite
}

func (s *timeoutEnabledSuite) SetupSuite() {
	setup.SetUpLogFilePath(s.baseTestName, GKETempDir, OldGKElogFilePath, testEnv.cfg)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *timeoutEnabledSuite) SetupTest() {
	gcsDir := path.Join(kTestDirName, s.T().Name())
	testEnv.testDirPath = path.Join(mountDir, gcsDir)
	SetupNestedTestDir(testEnv.testDirPath, 0755, s.T())
	setup.CleanupDirectoryOnGCS(s.ctx, s.storageClient, gcsDir)
	client.SetupFileInTestDirectory(s.ctx, s.storageClient, gcsDir, kTestFileName, kFileSize, s.T())
}

func (s *timeoutEnabledSuite) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *timeoutEnabledSuite) TearDownTest() {
	setup.CleanupDirectoryOnGCS(s.ctx, s.storageClient, path.Join(kTestDirName, s.T().Name()))
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *timeoutEnabledSuite) TestReaderCloses() {
	timeoutDuration := kDefaultInactiveReadTimeoutInSeconds * time.Second
	mountFilePath := path.Join(testEnv.testDirPath, kTestFileName)

	// 1. Open file.
	fileHandle, err := operations.OpenFileAsReadonly(mountFilePath)
	require.NoError(s.T(), err)
	defer fileHandle.Close()

	// 2. Read small chunk from 0 offset.
	buff := make([]byte, kChunkSizeToRead)
	_, err = fileHandle.ReadAt(buff, 0)
	require.NoError(s.T(), err)
	endTimeRead := time.Now()

	// 3. Wait for timeout
	time.Sleep(2*timeoutDuration + 1*time.Second) // Add buffer
	endTimeWait := time.Now()

	// 4. "Closing reader" log should be present.
	validateInactiveReaderClosedLog(s.T(), testEnv.cfg.LogFile, path.Join(kTestDirName, s.T().Name(), kTestFileName), true, endTimeRead, endTimeWait)

	// 5. Further reads should work as it is, yeah it will create a new reader.
	_, err = fileHandle.ReadAt(buff, 8)
	require.NoError(s.T(), err)
}

func (s *timeoutEnabledSuite) TestReaderStaysOpenWithinTimeout() {
	timeoutDuration := kDefaultInactiveReadTimeoutInSeconds * time.Second
	mountFilePath := path.Join(testEnv.testDirPath, kTestFileName)

	fileHandle, err := operations.OpenFileAsReadonly(mountFilePath)
	require.NoError(s.T(), err)
	defer fileHandle.Close()

	// 1. First read.
	buff := make([]byte, kChunkSizeToRead)
	_, err = fileHandle.ReadAt(buff, 0)
	require.NoError(s.T(), err)
	endTimeRead1 := time.Now()

	// 2. Wait for a period SHORTER than the timeout.
	time.Sleep(timeoutDuration / 2)
	startTimeRead2 := time.Now()

	// 3. Second read.
	_, err = fileHandle.ReadAt(buff, int64(kChunkSizeToRead)) // Read the next chunk
	require.NoError(s.T(), err, "Second read within timeout failed")

	// 4. Check log: "Closing reader for object..." should NOT be present for this object
	// between the first read's end and the second read's start.
	validateInactiveReaderClosedLog(s.T(), testEnv.cfg.LogFile, path.Join(kTestDirName, s.T().Name(), kTestFileName), false, endTimeRead1, startTimeRead2)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestTimeoutEnabledSuite(t *testing.T) {
	ts := &timeoutEnabledSuite{
		ctx:           context.Background(),
		storageClient: testEnv.storageClient,
		baseTestName:  t.Name(),
	}
	// Run tests for mounted directory if the flag is set.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, ts)
		return
	}

	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, ts.flags = range flagsSet {
		log.Printf("Running inactive_read_timeout tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
