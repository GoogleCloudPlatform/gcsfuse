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

type defaultChunkTransferTimeout struct {
	flags []string
}

func (s *defaultChunkTransferTimeout) Setup(t *testing.T) {
}

func (s *defaultChunkTransferTimeout) Teardown(t *testing.T) {
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

// The chunk upload should stall, but ultimately complete successfully after 10 seconds.
// This means the overall file upload will experience an 10-second delay instead of a 40-second delay.
func (s *defaultChunkTransferTimeout) TestChunkTransferTimeout_HandlesWriteStalls(t *testing.T) {
	configPath := "./proxy_server/configs/write_stall_40s.yaml"
	emulator_tests.StartProxyServer(configPath)
	setup.MountGCSFuseWithGivenMountFunc(s.flags, mountFunc)
	testDir := "TestChunkTransferTimeout_HandlesWriteStalls"
	testDirPath = setup.SetupTestDirectory(testDir)
	filePath := path.Join(testDirPath, "file.txt")
	elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

	assert.NoError(t, err)
	// The chunk upload should stall but successfully complete after 10 seconds.
	// Overall file upload will face 10 seconds of stall instead of 40 seconds.
	assert.GreaterOrEqual(t, elapsedTime, 10*time.Second)
	assert.Less(t, elapsedTime, 15*time.Second)

	setup.UnmountGCSFuse(rootDir)
	assert.NoError(t, emulator_tests.KillProxyServerProcess(port))
}

// The chunk upload should stall twice, but after each 10-second timeout,
// a new request will be initiated. The chunk should be successfully uploaded after approximately 20 seconds.
func (s *defaultChunkTransferTimeout) TestChunkTransferTimeout_HandlesMultipleWriteStalls(t *testing.T) {
	configPath := "./proxy_server/configs/write_stall_twice_40s.yaml"
	emulator_tests.StartProxyServer(configPath)
	setup.MountGCSFuseWithGivenMountFunc(s.flags, mountFunc)
	testDir := "TestChunkTransferTimeout_HandlesMultipleWriteStalls"
	testDirPath = setup.SetupTestDirectory(testDir)
	filePath := path.Join(testDirPath, "file2.txt")

	elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

	assert.NoError(t, err)
	// The chunk upload should stall but successfully complete after 20 seconds.
	// Overall file upload will face 20 seconds of stall instead of 40 seconds.
	assert.GreaterOrEqual(t, elapsedTime, 20*time.Second)
	assert.Less(t, elapsedTime, 25*time.Second)

	setup.UnmountGCSFuse(rootDir)
	assert.NoError(t, emulator_tests.KillProxyServerProcess(port))
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestDefaultChunkTransferTimeout(t *testing.T) {
	ts := &defaultChunkTransferTimeout{}
	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--custom-endpoint=" + proxyEndpoint},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}

type finiteChunkTransferTimeout struct {
	flags []string
}

func (s *finiteChunkTransferTimeout) Setup(t *testing.T) {
}

func (s *finiteChunkTransferTimeout) Teardown(t *testing.T) {
}

// The chunk upload should stall, but ultimately complete successfully after 20 seconds.
// This means the overall file upload will experience an 20-second delay instead of a 40-second delay.
func (s *finiteChunkTransferTimeout) TestFiniteChunkTransferTimeout_HandlesMultipleWriteStalls(t *testing.T) {
	configPath := "./proxy_server/configs/write_stall_40s.yaml"
	emulator_tests.StartProxyServer(configPath)
	setup.MountGCSFuseWithGivenMountFunc(s.flags, mountFunc)
	testDir := "TestFiniteChunkTransferTimeout_HandlesMultipleWriteStalls"
	testDirPath = setup.SetupTestDirectory(testDir)
	filePath := path.Join(testDirPath, "file2.txt")

	elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

	assert.NoError(t, err)
	// The chunk upload should stall but successfully complete after 20 seconds.
	// Overall file upload will face 20 seconds of stall instead of 40 seconds.
	assert.GreaterOrEqual(t, elapsedTime, 20*time.Second)
	assert.Less(t, elapsedTime, 25*time.Second)

	setup.UnmountGCSFuse(rootDir)
	assert.NoError(t, emulator_tests.KillProxyServerProcess(port))
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestFiniteChunkTransferTimeout(t *testing.T) {
	ts := &finiteChunkTransferTimeout{}
	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--custom-endpoint=" + proxyEndpoint, "--chunk-transfer-timeout-secs=20"},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
