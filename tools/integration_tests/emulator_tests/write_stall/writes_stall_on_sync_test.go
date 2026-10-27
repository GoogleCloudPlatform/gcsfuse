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

var (
	protocols      = []string{"http1", "grpc"}
	grpcConfigPath = map[string]string{
		"../configs/write_stall_40s.yaml":             "../configs/grpc_write_stall_40s.yaml",
		"../configs/write_stall_twice_40s.yaml":       "../configs/grpc_write_stall_twice_40s.yaml",
		"../configs/write_stalls_four_times_60s.yaml": "../configs/grpc_write_stalls_four_times_60s.yaml",
	}
)

type writeStallTestSuite struct {
	port               int
	proxyProcessId     int
	proxyServerLogFile string
	protocol           string
	suite.Suite
}

func (s *writeStallTestSuite) setupTest(configPath string, testFlags []string) {
	if s.protocol == "grpc" {
		configPath = grpcConfigPath[configPath]
		require.NotEmpty(s.T(), configPath)
	}

	s.proxyServerLogFile = setup.CreateProxyServerLogFile(s.T())
	var err error
	s.port, s.proxyProcessId, err = emulator_tests.StartProxyServer(configPath, s.proxyServerLogFile)
	require.NoError(s.T(), err)

	flags := append([]string{"--client-protocol=" + s.protocol}, testFlags...)
	if s.protocol == "grpc" {
		flags = append(flags, "--anonymous-access", fmt.Sprintf("--custom-endpoint=localhost:%d", s.port))
	} else {
		// TODO: Remove --enable-hns=false after proxy server supports multiplexing
		// HTTP and gRPC on the same port.
		// Disable HNS to prevent gRPC Control Client initialization.
		// Legacy emulator proxy servers run on HTTP/1.1, which causes the
		// gRPC HTTP/2 dialer to crash the mount sequence.
		flags = append(flags, "--enable-hns=false")
		setup.AppendProxyEndpointToFlagSet(&flags, s.port)
	}
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
}

func (s *writeStallTestSuite) tearDownTest() {
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
func (s *writeStallTestSuite) TestChunkTransferTimeoutInfinity() {
	s.setupTest("../configs/write_stall_40s.yaml", []string{"--chunk-transfer-timeout-secs=0"})
	defer s.tearDownTest()

	testDir := "TestWriteStallCausesDelay" + setup.GenerateRandomString(3)
	testDirPath = setup.SetupTestDirectory(testDir)
	filePath := path.Join(testDirPath, "file.txt")

	elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

	assert.NoError(s.T(), err)
	assert.GreaterOrEqual(s.T(), elapsedTime, stallTime)
}

func (s *writeStallTestSuite) TestChunkTransferTimeout() {
	flagSets := [][]string{
		{},
		{"--chunk-transfer-timeout-secs=5"},
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
		require.NoError(s.T(), err, "Invalid chunk-transfer-timeout-secs flag")

		s.Run(fmt.Sprintf("Flags_%v", flags), func() {
			for _, scenario := range stallScenarios {
				s.Run(scenario.name, func() {
					s.setupTest(scenario.configPath, flags)
					defer s.tearDownTest()

					testDir := scenario.name + setup.GenerateRandomString(3)
					testDirPath = setup.SetupTestDirectory(testDir)
					filePath := path.Join(testDirPath, "file.txt")

					elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

					assert.NoError(s.T(), err, "failed to write file and sync")
					expectedTimeout := scenario.expectedTimeout(chunkTransferTimeoutSecs)
					assert.GreaterOrEqual(s.T(), elapsedTime, expectedTimeout)
					assert.Less(s.T(), elapsedTime, expectedTimeout+5*time.Second)
				})
			}
		})
	}
}

func (s *writeStallTestSuite) TestChunkRetryDeadline() {
	scenarios := []struct {
		name            string
		flags           []string
		expectedSuccess bool
		expectedStall   time.Duration
	}{
		{
			name:            "StallUnderDeadline_Pass",
			flags:           []string{"--chunk-transfer-timeout-secs=10", "--chunk-retry-deadline-secs=120"},
			expectedStall:   40 * time.Second,
			expectedSuccess: true,
		},
		{
			name:            "StallOverDeadline_Fail",
			flags:           []string{"--chunk-transfer-timeout-secs=10", "--chunk-retry-deadline-secs=32"},
			expectedStall:   40 * time.Second,
			expectedSuccess: false,
		},
	}
	// The proxy is configured to stall the first 4 requests (4 stalls).
	// With a 10s chunk-transfer-timeout, the client will timeout after 10s for each of the first 4 attempts.
	// This results in 40s of total elapsed time before the 5th attempt (which is the 4th retry) succeeds.
	configPath := "../configs/write_stalls_four_times_60s.yaml"

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			s.setupTest(configPath, scenario.flags)
			defer s.tearDownTest()

			testDir := scenario.name + setup.GenerateRandomString(3)
			testDirPath = setup.SetupTestDirectory(testDir)
			filePath := path.Join(testDirPath, "file.txt")

			elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

			if scenario.expectedSuccess {
				assert.NoError(s.T(), err, "expected success for chunk-retry-deadline-secs")
				assert.GreaterOrEqual(s.T(), elapsedTime, scenario.expectedStall)
			} else {
				assert.Error(s.T(), err, "expected failure due to chunk-retry-deadline-secs")
			}
		})
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestWriteStall(t *testing.T) {
	for _, protocol := range protocols {
		t.Run(protocol, func(t *testing.T) {
			log.Printf("Running write stall tests with protocol=%s", protocol)
			suite.Run(t, &writeStallTestSuite{protocol: protocol})
		})
	}
}
