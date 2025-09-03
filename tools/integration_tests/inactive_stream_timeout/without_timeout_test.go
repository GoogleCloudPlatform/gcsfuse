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
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type timeoutDisabledSuite struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	suite.Suite
}

func (s *timeoutDisabledSuite) SetupTest() {
	mountGCSFuseAndSetupTestDir(s.ctx, s.flags, s.storageClient, kTestDirName)
}

func (s *timeoutDisabledSuite) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	if setup.MountedDirectory() == "" { // Only unmount if not using a pre-mounted directory
		setup.UnmountGCSFuseAndDeleteLogFile(gRootDir)
	}
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *timeoutDisabledSuite) TestNoReaderCloser() {
	timeoutDuration := kDefaultInactiveReadTimeoutInSeconds * time.Second
	gcsFileName := path.Join(kTestDirName, kTestFileName)
	mountFilePath := setupFile(s.ctx, s.storageClient, kTestFileName, kFileSize, s.T())

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

	// 4. Shouldn't be any `Close reader logs...`.
	validateInactiveReaderClosedLog(s.T(), setup.LogFile(), gcsFileName, false, endTimeRead, endTimeWait)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestTimeoutDisabledSuite(t *testing.T) {
	ts := &timeoutDisabledSuite{ctx: context.Background(), storageClient: gStorageClient}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, ts)
		return
	}

	flagsSet := []gcsfuseTestFlags{
		{ // Test with timeout disabled
			inactiveReadTimeout: 0 * time.Second, // Disable timeout
			clientProtocol:      kHTTP1ClientProtocol,
			fileName:            "zero_timeout.yaml",
		},
	}

	for _, flags := range flagsSet {
		configFilePath := createConfigFile(&flags)
		ts.flags = []string{"--config-file=" + configFilePath}
		if flags.cliFlags != nil {
			ts.flags = append(ts.flags, flags.cliFlags...)
		}
		log.Printf("Running inactive_read_timeout tests with flags: %s", ts.flags)

		suite.Run(t, ts)
	}
}
