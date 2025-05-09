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

package inactive_read_timeout

import (
	"context"
	"log"
	"path"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/require"
)

const DefaultSequentialReadSizeMb = 5

type timeoutEnabledSuite struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
}

func (s *timeoutEnabledSuite) Setup(t *testing.T) {
	mountGCSFuseAndSetupTestDir(s.ctx, s.flags, s.storageClient, kTestDirName)
}

func (s *timeoutEnabledSuite) Teardown(t *testing.T) {
	setup.SaveGCSFuseLogFileInCaseOfFailure(t)
	if setup.MountedDirectory() == "" { // Only unmount if not using a pre-mounted directory
		setup.UnmountGCSFuseAndDeleteLogFile(gRootDir)
	}
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *timeoutEnabledSuite) TestReaderCloses(t *testing.T) {
	timeoutDuration := kDefaultInactiveReadTimeoutInSeconds * time.Second
	gcsFileName := path.Join(kTestDirName, kTestFileName)
	mountFilePath := setupFile(s.ctx, s.storageClient, kTestFileName, kFileSize, t)

	// 1. Open file.
	fileHandle, err := operations.OpenFileAsReadonly(mountFilePath)
	require.NoError(t, err)
	defer fileHandle.Close()

	// 2. Read small chunk from 0 offset.
	buff := make([]byte, kChunkSizeToRead)
	_, err = fileHandle.ReadAt(buff, 0)
	require.NoError(t, err)
	endTimeRead := time.Now()

	// 3. Wait for timeout
	time.Sleep(2*timeoutDuration + 1*time.Second) // Add buffer
	endTimeWait := time.Now()

	// 4. "Closing reader" log should be present.
	validateInactiveReaderClosedLog(t, setup.LogFile(), gcsFileName, true, endTimeRead, endTimeWait)

	// 5. Further reads should work as it is.
	_, err = fileHandle.ReadAt(buff, 8)
	require.NoError(t, err)
}

func (s *timeoutEnabledSuite) TestReaderStaysOpenWithinTimeout(t *testing.T) {
	timeoutDuration := kDefaultInactiveReadTimeoutInSeconds * time.Second
	gcsFileName := path.Join(kTestDirName, kTestFileName)
	localFilePath := setupFile(s.ctx, s.storageClient, kTestFileName, kFileSize, t)

	fileHandle, err := operations.OpenFileAsReadonly(localFilePath)
	require.NoError(t, err)
	defer fileHandle.Close()

	// 1. First read.
	buff := make([]byte, kChunkSizeToRead)
	_, err = fileHandle.ReadAt(buff, 0)
	require.NoError(t, err)
	endTimeRead1 := time.Now()

	// 2. Wait for a period SHORTER than the timeout.
	time.Sleep(timeoutDuration / 2)
	startTimeRead2 := time.Now()

	// 3. Second read.
	_, err = fileHandle.ReadAt(buff, int64(kChunkSizeToRead)) // Read the next chunk
	require.NoError(t, err, "Second read within timeout failed")

	// 4. Check log: "Closing reader for object..." should NOT be present for this object
	// between the first read's end and the second read's start.
	validateInactiveReaderClosedLog(t, setup.LogFile(), gcsFileName, false, endTimeRead1, startTimeRead2)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestTimeoutEnabledSuite(t *testing.T) {
	ts := &timeoutEnabledSuite{ctx: context.Background(), storageClient: gStorageClient}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	flagsSet := []gcsfuseTestFlags{
		{ // Test with timeout enabled and grpc client protocol
			inactiveReadTimeout: kDefaultInactiveReadTimeoutInSeconds * time.Second,
			fileName:            "no_timeout.yaml",
			clientProtocol:      kGRPCClientProtocol,
		},
	}

	for _, flags := range flagsSet {
		configFilePath := createConfigFile(&flags)
		ts.flags = []string{"--config-file=" + configFilePath}
		if flags.cliFlags != nil {
			ts.flags = append(ts.flags, flags.cliFlags...)
		}
		log.Printf("Running inactive_read_timeout tests with flags: %s", ts.flags)

		test_setup.RunTests(t, ts)
	}
}
