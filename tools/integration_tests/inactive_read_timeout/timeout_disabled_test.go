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
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/require"
)

type disabledSuite struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
}

func (s *disabledSuite) Setup(t *testing.T) {
	if setup.MountedDirectory() != "" {
		// Assuming log file is already set by TestMain for mounted directory
	} else {
		operations.RemoveDir(path.Join(setup.TestDir(), "temp_cache_inactive_read"))
	}
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient, testDirName)
}

func (s *disabledSuite) Teardown(t *testing.T) {
	setup.SaveGCSFuseLogFileInCaseOfFailure(t)
	if setup.MountedDirectory() == "" { // Only unmount if not using a pre-mounted directory
		setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	}
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *disabledSuite) TestReaderCloses(t *testing.T) {
	timeoutDuration := defaultInactiveReadTimeoutInSeconds * time.Second
	gcsFileName := path.Join(testDirName, testFileName)
	mountFilePath := setupFile(s.ctx, s.storageClient, testFileName, fileSize, t)

	// 1. Open file.
	fileHandle, err := operations.OpenFileAsReadonly(mountFilePath)
	require.NoError(t, err)
	defer fileHandle.Close()

	// 2. Read small chunk from 0 offset.
	buff := make([]byte, chunkSizeToRead)
	_, err = fileHandle.ReadAt(buff, 0)
	require.NoError(t, err)
	endTimeRead := time.Now()

	// 3. Wait for timeout
	time.Sleep(2*timeoutDuration + 1*time.Second) // Add buffer
	endTimeWait := time.Now()

	// 3. Check log for "Closing reader"
	validateInactiveReaderClosedLog(t, setup.LogFile(), gcsFileName, false, endTimeRead, endTimeWait)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestDisabledSuite(t *testing.T) {
	ts := &disabledSuite{ctx: context.Background()}

	// Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&ts.ctx, &ts.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			t.Errorf("closeStorageClient failed: %v", err)
		}
	}()

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	flagsSet := []gcsfuseTestFlags{
		{ // Test with timeout enabled and http1 client protocol
			inactiveReadTimeout: defaultInactiveReadTimeoutInSeconds * time.Second,
			fileName:            configFileName,
			clientProtocol:      http1ClientProtocol,
		},
		{ // Test with timeout disabled
			inactiveReadTimeout: 0 * time.Second, // Disable timeout
			clientProtocol:      http1ClientProtocol,
			fileName:            configFileName + "_disabled",
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
