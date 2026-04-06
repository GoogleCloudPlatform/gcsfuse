// Copyright 2024 Google LLC
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

package write_stall

import (
	"fmt"
	"log"
	"path"
	"testing"
	"time"

	emulator_tests "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/emulator_tests/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const (
	fileSize  = 50 * 1024 * 1024
	stallTime = 40 * time.Second
)

type chunkTransferTimeoutInfinity struct {
	port               int
	proxyProcessId     int
	proxyServerLogFile string
	flags              []string
	suite.Suite
}

func (s *chunkTransferTimeoutInfinity) SetupTest() {
	configPath := "../configs/write_stall_40s.yaml"
	s.proxyServerLogFile = setup.CreateProxyServerLogFile(s.T())
	var err error
	s.port, s.proxyProcessId, err = emulator_tests.StartProxyServer(configPath, s.proxyServerLogFile)
	require.NoError(s.T(), err)
	setup.AppendProxyEndpointToFlagSet(&s.flags, s.port)
	setup.MountGCSFuseWithGivenMountFunc(s.flags, mountFunc)
}

func (s *chunkTransferTimeoutInfinity) TearDownTest() {
	setup.UnmountGCSFuse(rootDir)
	assert.NoError(s.T(), emulator_tests.KillProxyServerProcess(s.proxyProcessId))
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.SaveProxyServerLogFileInCaseOfFailure(s.proxyServerLogFile, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

// This test verifies that write operations stall for the expected duration
// when write stall is induced while uploading first chunk.
// It creates a file, writes data to it, and then calls Sync() to ensure
// the data is written to GCS. The test measures the time taken for the Sync()
// operation and asserts that it is greater than or equal to the configured stall time.
func (s *chunkTransferTimeoutInfinity) TestWriteStallCausesDelay() {
	testDir := "TestWriteStallCausesDelay"
	testDirPath = setup.SetupTestDirectory(testDir)
	filePath := path.Join(testDirPath, "file.txt")

	elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

	assert.NoError(s.T(), err)
	assert.GreaterOrEqual(s.T(), elapsedTime, stallTime)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestChunkTransferTimeoutInfinity(t *testing.T) {
	ts := &chunkTransferTimeoutInfinity{}
	// Define flag set to run the tests.
	flagsSet := [][]string{
		{
			"--chunk-transfer-timeout-secs=0",
			// Disable HNS to prevent gRPC Control Client initialization.
			// Legacy emulator proxy servers run on HTTP/1.1, which causes the
			// gRPC HTTP/2 dialer to crash the mount sequence.
			"--enable-hns=false",
		},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}

func TestChunkTransferTimeout(t *testing.T) {
	flagSets := [][]string{
		{"--enable-hns=false"},
		{"--chunk-transfer-timeout-secs=5", "--enable-hns=false"},
	}

	stallScenarios := []struct {
		name            string
		configPath      string
		expectedTimeout func(int) time.Duration
	}{
		{
			name:       "SingleStall",
			configPath: "../configs/write_stall_40s.yaml",
			expectedTimeout: func(chunkTransferTimeoutSecs int) time.Duration {
				return time.Duration(chunkTransferTimeoutSecs) * time.Second
			},
		},
		{
			name:       "MultipleStalls",
			configPath: "../configs/write_stall_twice_40s.yaml", // 2 stalls
			// Expect total time to be greater than the timeout multiplied by the number of stalls (2 in this case).
			expectedTimeout: func(chunkTransferTimeoutSecs int) time.Duration {
				return time.Duration(chunkTransferTimeoutSecs*2) * time.Second
			},
		},
	}

	for _, flags := range flagSets {
		chunkTransferTimeoutSecs, err := emulator_tests.GetChunkTransferTimeoutFromFlags(flags)
		if err != nil {
			t.Fatalf("Invalid chunk-transfer-timeout-secs flag: %v", err)
		}

		t.Run(fmt.Sprintf("Flags_%v", flags), func(t *testing.T) {
			for _, scenario := range stallScenarios {
				t.Run(scenario.name, func(t *testing.T) {
					proxyServerLogFile := setup.CreateProxyServerLogFile(t)
					port, proxyProcessId, err := emulator_tests.StartProxyServer(scenario.configPath, proxyServerLogFile)
					require.NoError(t, err)
					setup.AppendProxyEndpointToFlagSet(&flags, port)
					setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)

					defer func() { // Defer unmount, killing the proxy server and saving log files.
						setup.UnmountGCSFuse(rootDir)
						assert.NoError(t, emulator_tests.KillProxyServerProcess(proxyProcessId))
						setup.SaveGCSFuseLogFileInCaseOfFailure(t)
						setup.SaveProxyServerLogFileInCaseOfFailure(proxyServerLogFile, t)
					}()

					testDir := scenario.name + setup.GenerateRandomString(3)
					testDirPath = setup.SetupTestDirectory(testDir)
					filePath := path.Join(testDirPath, "file.txt")

					elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

					assert.NoError(t, err, "failed to write file and sync")
					expectedTimeout := scenario.expectedTimeout(chunkTransferTimeoutSecs)
					assert.GreaterOrEqual(t, elapsedTime, expectedTimeout)
					assert.Less(t, elapsedTime, expectedTimeout+5*time.Second)
				})
			}
		})
	}
}

func TestChunkRetryDeadline(t *testing.T) {
	scenarios := []struct {
		name            string
		flags           []string
		expectedSuccess bool
		expectedStall   time.Duration
	}{
		{
			name:            "StallUnderDeadline_Pass",
			flags:           []string{"--chunk-transfer-timeout-secs=10", "--chunk-retry-deadline-secs=120", "--enable-hns=false"},
			expectedStall:   40 * time.Second,
			expectedSuccess: true,
		},
		{
			name:            "StallOverDeadline_Fail",
			flags:           []string{"--chunk-transfer-timeout-secs=10", "--chunk-retry-deadline-secs=32", "--enable-hns=false"},
			expectedStall:   40 * time.Second,
			expectedSuccess: false,
		},
	}
	// The proxy is configured to stall the first 4 requests (4 stalls).
	// With a 10s chunk-transfer-timeout, the client will timeout after 10s for each of the first 4 attempts.
	// This results in 40s of total elapsed time before the 5th attempt (which is the 4th retry) succeeds.
	configPath := "../configs/write_stalls_four_times_60s.yaml"

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			proxyServerLogFile := setup.CreateProxyServerLogFile(t)
			flags := append([]string{}, scenario.flags...)
			port, proxyProcessId, err := emulator_tests.StartProxyServer(configPath, proxyServerLogFile)
			require.NoError(t, err)
			setup.AppendProxyEndpointToFlagSet(&flags, port)
			setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
			defer func() {
				setup.UnmountGCSFuse(rootDir)
				assert.NoError(t, emulator_tests.KillProxyServerProcess(proxyProcessId))
				setup.SaveGCSFuseLogFileInCaseOfFailure(t)
				setup.SaveProxyServerLogFileInCaseOfFailure(proxyServerLogFile, t)
			}()
			testDir := scenario.name + setup.GenerateRandomString(3)
			testDirPath = setup.SetupTestDirectory(testDir)
			filePath := path.Join(testDirPath, "file.txt")

			elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

			if scenario.expectedSuccess {
				assert.NoError(t, err, "expected success for chunk-retry-deadline-secs")
				assert.GreaterOrEqual(t, elapsedTime, scenario.expectedStall)
			} else {
				assert.Error(t, err, "expected failure due to chunk-retry-deadline-secs")
			}
		})
	}
}
