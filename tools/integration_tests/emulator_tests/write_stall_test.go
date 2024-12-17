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

// Test suite for chunk transfer timeouts.
type chunkTransferTimeoutSuite struct {
	flags []string
	// Timeout for chunk transfers, in seconds.
	chunkTransferTimeoutSecs int
}

// Setup for each test case.
func (s *chunkTransferTimeoutSuite) Setup(t *testing.T) {
}

// Teardown for each test case.
func (s *chunkTransferTimeoutSuite) Teardown(t *testing.T) {
	setup.UnmountGCSFuse(rootDir)
	assert.NoError(t, emulator_tests.KillProxyServerProcess(port))
}

// Test case: single write stall.
func (s *chunkTransferTimeoutSuite) TestHandlesWriteStalls(t *testing.T) {
	configPath := "./proxy_server/configs/write_stall_40s.yaml"
	emulator_tests.StartProxyServer(configPath)
	setup.MountGCSFuseWithGivenMountFunc(s.flags, mountFunc)

	testDir := "TestHandlesWriteStalls" + setup.GenerateRandomString(3)
	testDirPath = setup.SetupTestDirectory(testDir)
	filePath := path.Join(testDirPath, "file.txt")

	elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

	assert.NoError(t, err)
	// The chunk upload should stall but successfully complete after the expected timeout.
	expectedTimeout := time.Duration(s.chunkTransferTimeoutSecs) * time.Second
	assert.GreaterOrEqual(t, elapsedTime, expectedTimeout)
	assert.Less(t, elapsedTime, expectedTimeout+5*time.Second) // Allow some buffer
}

// Test case: multiple write stalls.
func (s *chunkTransferTimeoutSuite) TestHandlesMultipleWriteStalls(t *testing.T) {
	configPath := "./proxy_server/configs/write_stall_twice_40s.yaml"
	emulator_tests.StartProxyServer(configPath)
	setup.MountGCSFuseWithGivenMountFunc(s.flags, mountFunc)

	testDir := "TestHandlesMultipleWriteStalls" + setup.GenerateRandomString(3)
	testDirPath = setup.SetupTestDirectory(testDir)
	filePath := path.Join(testDirPath, "file.txt")

	elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

	assert.NoError(t, err)
	// The chunk upload should stall multiple times but ultimately succeed.
	//  Expect total time to be greater than the timeout multiplied by the number of stalls (2 in this case).
	expectedTimeout := time.Duration(s.chunkTransferTimeoutSecs*2) * time.Second
	assert.GreaterOrEqual(t, elapsedTime, expectedTimeout)
	assert.Less(t, elapsedTime, expectedTimeout+5*time.Second) // Allow some buffer
}

// Test function to run the suite with different flag sets.
func TestChunkTransferTimeout(t *testing.T) {
	testCases := []struct {
		name                     string
		flags                    []string
		chunkTransferTimeoutSecs int
	}{
		{
			name:                     "DefaultChunkTransferTimeout",
			flags:                    []string{"--custom-endpoint=" + proxyEndpoint},
			chunkTransferTimeoutSecs: 10, // Default timeout
		},
		{
			name:                     "FiniteChunkTransferTimeout",
			flags:                    []string{"--custom-endpoint=" + proxyEndpoint, "--chunk-transfer-timeout-secs=5"},
			chunkTransferTimeoutSecs: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts := &chunkTransferTimeoutSuite{
				flags:                    tc.flags,
				chunkTransferTimeoutSecs: tc.chunkTransferTimeoutSecs,
			}
			log.Printf("Running tests with flags: %s", ts.flags)
			test_setup.RunTests(t, ts)
		})
	}
}
