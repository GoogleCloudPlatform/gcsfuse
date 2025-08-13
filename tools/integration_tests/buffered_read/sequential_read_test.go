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

package buffered_read

import (
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	chunkSizeToRead = 1 * 1024 * 1024 // 1MB
)

////////////////////////////////////////////////////////////////////////
// Test Suite Boilerplate
////////////////////////////////////////////////////////////////////////

type SequentialReadSuite struct {
	suite.Suite
	// Helper struct with test flags.
	testFlags *gcsfuseTestFlags
}

func (s *SequentialReadSuite) SetupSuite() {
	// Create config file.
	configFile := createConfigFile(s.testFlags)
	// Create the final flags slice.
	// The static mounting helper adds --log-file and --log-severity flags, so we only need to add the format.
	flags := []string{"--config-file=" + configFile}
	// Mount GCSFuse.
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
}

func (s *SequentialReadSuite) TearDownSuite() {
	setup.UnmountGCSFuse(setup.MntDir())
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Cases
////////////////////////////////////////////////////////////////////////
func (s *SequentialReadSuite) TestSequentialRead() {
	testCases := []struct {
		name       string
		sizeFactor float64
	}{
		{
			name:       "SmallFile",
			sizeFactor: 0.5, // file-size = blockSize/2
		},
		{
			name:       "LargeFile",
			sizeFactor: 2.0, // file-size = blockSize*2
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			err := os.Truncate(setup.LogFile(), 0)
			require.NoError(t, err, "Failed to truncate log file")
			fileSize := int64(float64(s.testFlags.blockSizeMB*1024*1024) * tc.sizeFactor)
			testDir := setup.SetupTestDirectory(testDirName)
			fileName := setupFileInTestDir(ctx, storageClient, testDir, fileSize, t)

			expected := readFileAndValidate(ctx, storageClient, testDir, fileName, true, 0, chunkSizeToRead, t)

			f, err := os.Open(setup.LogFile())
			require.NoError(t, err, "Failed to open log file")
			defer f.Close()
			logEntries, err := read_logs.ParseBufferedReadLogsFromLogReader(f)
			require.NoError(t, err, "Failed to parse log file")
			require.Equal(t, 1, len(logEntries), "Expected one buffered read log entry for sequential read.")
			var bufferedReadLogEntry *read_logs.BufferedReadLogEntry
			for _, entry := range logEntries {
				bufferedReadLogEntry = entry // Get the single entry.
			}
			validate(expected, bufferedReadLogEntry, false, t)
			assert.Equal(t, int64(0), bufferedReadLogEntry.RandomSeekCount, "RandomSeekCount should be 0 for sequential reads.")
		})
	}
}

////////////////////////////////////////////////////////////////////////
// Test Harness
////////////////////////////////////////////////////////////////////////

func TestSequentialReadSuite(t *testing.T) {
	// Define the different flag configurations to test against.
	baseTestFlags := gcsfuseTestFlags{
		enableBufferedRead:   true,
		blockSizeMB:          8,
		maxBlocksPerHandle:   20,
		startBlocksPerHandle: 1,
		minBlocksPerHandle:   2,
	}

	protocols := []string{clientProtocolHTTP1, clientProtocolGRPC}

	for _, protocol := range protocols {
		// Create a new suite for each protocol.
		t.Run(protocol, func(t *testing.T) {
			testFlags := baseTestFlags
			testFlags.clientProtocol = protocol

			suite.Run(t, &SequentialReadSuite{testFlags: &testFlags})
		})
	}
}
