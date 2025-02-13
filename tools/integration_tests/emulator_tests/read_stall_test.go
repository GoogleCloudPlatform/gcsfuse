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

package emulator_tests

import (
	"log"
	"path"
	"testing"
	"time"

	emulator_tests "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/emulator_tests/util"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/assert"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const readStallTime = 40 * time.Second

type readStall struct {
	flags []string
}

func (s *readStall) Setup(t *testing.T) {
	configPath := "./proxy_server/configs/read_stall_40s.yaml"
	emulator_tests.StartProxyServer(configPath)
	setup.MountGCSFuseWithGivenMountFunc(s.flags, mountFunc)
}

func (s *readStall) Teardown(t *testing.T) {
	setup.UnmountGCSFuse(rootDir)
	assert.NoError(t, emulator_tests.KillProxyServerProcess(port))
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

// TestReadFirstKBStallInducedShouldCompleteInLessThanStallTime verifies that reading the first 1KB
// of a file completes in less time than the configured stall time, even when a read stall is induced.
// It creates a file, reads the initial 1KB, and asserts that the elapsed time is less than the expected stall duration.
func (s *readStall) TestReadFirstKBStallInducedShouldCompleteInLessThanStallTime(t *testing.T) {
	testDir := "TestReadFirstKBStallInducedShouldCompleteInLessThanStallTime"
	testDirPath = setup.SetupTestDirectory(testDir)
	filePath := path.Join(testDirPath, "file.txt")
	operations.CreateFileOfSize(fileSize, filePath, t)

	elapsedTime, err := emulator_tests.ReadFirstKB(filePath)

	assert.NoError(t, err)
	assert.Less(t, elapsedTime, readStallTime)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestReadStall(t *testing.T) {
	ts := &readStall{}
	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--custom-endpoint=" + proxyEndpoint, "--enable-read-stall-retry=true"},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
