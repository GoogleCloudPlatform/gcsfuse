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

package emulator_tests

import (
	"fmt"
	"log"
	"path"
	"testing"
	"time"

	emulator_tests "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/emulator_tests/util"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/assert"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const (
	fileSize  = 50 * 1024 * 1024
	stallTime = 40 * time.Second
)

type chunkTransferTimeoutInfinity struct {
	flags []string
}

func (s *chunkTransferTimeoutInfinity) Setup(t *testing.T) {
	configPath := "./proxy_server/configs/write_stall_40s.yaml"
	emulator_tests.StartProxyServer(configPath)
	setup.MountGCSFuseWithGivenMountFunc(s.flags, mountFunc)
}

func (s *chunkTransferTimeoutInfinity) Teardown(t *testing.T) {
	setup.UnmountGCSFuse(rootDir)
	assert.NoError(t, emulator_tests.KillProxyServerProcess(port))
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

// This test verifies that write operations stall for the expected duration
// when write stall is induced while uploading first chunk.
// It creates a file, writes data to it, and then calls Sync() to ensure
// the data is written to GCS. The test measures the time taken for the Sync()
// operation and asserts that it is greater than or equal to the configured stall time.
func (s *chunkTransferTimeoutInfinity) TestWriteStallCausesDelay(t *testing.T) {
	testDir := "TestWriteStallCausesDelay"
	testDirPath = setup.SetupTestDirectory(testDir)
	filePath := path.Join(testDirPath, "file.txt")

	elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, elapsedTime, stallTime)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestChunkTransferTimeoutInfinity(t *testing.T) {
	ts := &chunkTransferTimeoutInfinity{}
	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--custom-endpoint=" + proxyEndpoint, "--chunk-transfer-timeout-secs=0"},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}

func TestChunkTransferTimeout(t *testing.T) {
	flagSets := [][]string{
		{"--custom-endpoint=" + proxyEndpoint},
		{"--custom-endpoint=" + proxyEndpoint, "--chunk-transfer-timeout-secs=5"},
	}

	stallScenarios := []struct {
		name            string
		configPath      string
		expectedTimeout func(int) time.Duration
	}{
		{
			name:       "SingleStall",
			configPath: "./proxy_server/configs/write_stall_40s.yaml",
			expectedTimeout: func(chunkTransferTimeoutSecs int) time.Duration {
				return time.Duration(chunkTransferTimeoutSecs) * time.Second
			},
		},
		{
			name:       "MultipleStalls",
			configPath: "./proxy_server/configs/write_stall_twice_40s.yaml",
			// Expect total time to be greater than the timeout multiplied by the number of stalls (2 in this case).
			expectedTimeout: func(chunkTransferTimeoutSecs int) time.Duration {
				return time.Duration(chunkTransferTimeoutSecs*2) * time.Second
			},
		},
	}

	for _, flags := range flagSets {
		chunkTransferTimeoutSecs := emulator_tests.GetChunkTransferTimeoutFromFlags(flags)

		t.Run(fmt.Sprintf("Flags_%v", flags), func(t *testing.T) {
			for _, scenario := range stallScenarios {
				t.Run(scenario.name, func(t *testing.T) {
					emulator_tests.StartProxyServer(scenario.configPath)
					setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)

					defer func() { // Defer unmount and  killing the server.
						setup.UnmountGCSFuse(rootDir)
						assert.NoError(t, emulator_tests.KillProxyServerProcess(port))
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
