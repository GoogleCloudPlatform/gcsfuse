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

package read_stall

import (
	"fmt"
	"log"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/stretchr/testify/assert"
	emulator_tests "github.com/vipnydav/gcsfuse/v3/tools/integration_tests/emulator_tests/util"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const (
	fileSize        = 5 * 1024 * 1024
	forcedStallTime = 5 * time.Second
	minReqTimeout   = 1500 * time.Millisecond
)

type readStall struct {
	port               int
	proxyProcessId     int
	proxyServerLogFile string
	flags              []string
	configPath         string
	suite.Suite
}

func (r *readStall) SetupTest() {
	r.configPath = "../configs/read_stall_5s.yaml"
	r.proxyServerLogFile = setup.CreateProxyServerLogFile(r.T())
	var err error
	r.port, r.proxyProcessId, err = emulator_tests.StartProxyServer(r.configPath, r.proxyServerLogFile)
	require.NoError(r.T(), err)
	setup.AppendProxyEndpointToFlagSet(&r.flags, r.port)
	setup.MountGCSFuseWithGivenMountFunc(r.flags, mountFunc)
	testDirPath = setup.SetupTestDirectory(r.T().Name())
}

func (r *readStall) TearDownTest() {
	setup.UnmountGCSFuse(rootDir)
	assert.NoError(r.T(), emulator_tests.KillProxyServerProcess(r.proxyProcessId))
	setup.SaveGCSFuseLogFileInCaseOfFailure(r.T())
	setup.SaveProxyServerLogFileInCaseOfFailure(r.proxyServerLogFile, r.T())
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
	assert.Less(r.T(), elapsedTime, forcedStallTime)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestReadStall(t *testing.T) {
	ts := &readStall{}
	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--enable-read-stall-retry=true", "--read-stall-min-req-timeout=" + fmt.Sprintf("%dms", minReqTimeout.Milliseconds()), "--read-stall-initial-req-timeout=" + fmt.Sprintf("%dms", minReqTimeout.Milliseconds())},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
