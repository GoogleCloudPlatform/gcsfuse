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
	"fmt"
	"log"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	emulator_tests "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/emulator_tests/util"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const (
	forcedStallTime = 40 * time.Second
	minReqTimeout   = 1500 * time.Millisecond
)

type readStall struct {
	flags []string
	suite.Suite
}

func (r *readStall) SetupSuite() {
	configPath := "./proxy_server/configs/read_stall_40s.yaml"
	emulator_tests.StartProxyServer(configPath)
	setup.MountGCSFuseWithGivenMountFunc(r.flags, mountFunc)
}

func (r *readStall) SetupTest() {
	testDirPath = setup.SetupTestDirectory(r.T().Name())
}

func (r *readStall) TearDownSuite() {
	setup.UnmountGCSFuse(rootDir)
	assert.NoError(r.T(), emulator_tests.KillProxyServerProcess(port))
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

// TestReadFirstByteStallInducedShouldCompleteInLessThanStallTime verifies that reading the first byte
// of a file completes in less time than the configured stall time, even when a read stall is induced.
// It creates a file, reads the first byte, and asserts that the elapsed time is less than the expected stall duration.
func (r *readStall) TestReadFirstByteStallInducedShouldCompleteInLessThanStallTime() {
	filePath := path.Join(testDirPath, "file.txt")
	operations.CreateFileOfSize(fileSize, filePath, r.T())

	elapsedTime, err := emulator_tests.ReadFirstByte(r.T(), filePath)

	assert.NoError(r.T(), err)
	assert.Greater(r.T(), elapsedTime, minReqTimeout)
	assert.Less(r.T(), 10*elapsedTime, forcedStallTime)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestReadStall(t *testing.T) {
	ts := &readStall{}
	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--custom-endpoint=" + proxyEndpoint, "--enable-read-stall-retry=true", "--read-stall-min-req-timeout=" + fmt.Sprintf("%dms", minReqTimeout.Milliseconds()), "--read-stall-initial-req-timeout=" + fmt.Sprintf("%dms", minReqTimeout.Milliseconds())},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
