// Copyright 2026 Google LLC
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

package control_client_stall

import (
	"fmt"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	emulator_tests "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/emulator_tests/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

type controlClientStall struct {
	port               int
	proxyProcessId     int
	proxyServerLogFile string
	flags              []string
	configPath         string
	forcedStallTime    time.Duration
	suite.Suite
}

func (c *controlClientStall) SetupTest() {
	c.proxyServerLogFile = setup.CreateProxyServerLogFile(c.T())
	var err error
	c.port, c.proxyProcessId, err = emulator_tests.StartProxyServer(c.configPath, c.proxyServerLogFile)
	require.NoError(c.T(), err)

	// Add proxy server endpoint and configure gRPC testing
	c.flags = append(c.flags, fmt.Sprintf("--custom-endpoint=localhost:%d", c.port))
	c.flags = append(c.flags, "--anonymous-access") // Required for gRPC localhost endpoint

	setup.MountGCSFuseWithGivenMountFunc(c.flags, mountFunc)
	testDirPath = setup.SetupTestDirectory(c.T().Name())
}

func (c *controlClientStall) TearDownTest() {
	setup.UnmountGCSFuse(rootDir)
	assert.NoError(c.T(), emulator_tests.KillProxyServerProcess(c.proxyProcessId))
	setup.SaveGCSFuseLogFileInCaseOfFailure(c.T())
	setup.SaveProxyServerLogFileInCaseOfFailure(c.proxyServerLogFile, c.T())
}

// TestCreateFolderStallInducedShouldComplete verifies that creating a folder via
// os.MkdirAll completes successfully even when a stall is induced,
// proving that the experimental retry logic correctly handles the delayed gRPC response.
func (c *controlClientStall) TestCreateFolderStallInducedShouldComplete() {
	folderPath := path.Join(testDirPath, "stalled_folder")

	startTime := time.Now()
	err := os.MkdirAll(folderPath, 0755)
	elapsedTime := time.Since(startTime)

	assert.NoError(c.T(), err)

	// Ensure the operation took at least as long as our forced stall time
	assert.GreaterOrEqual(c.T(), elapsedTime, c.forcedStallTime)

	// Validate the directory actually exists
	info, err := os.Stat(folderPath)
	assert.NoError(c.T(), err)
	assert.True(c.T(), info.IsDir())
}

func TestControlClientStall(t *testing.T) {
	// Test matrix
	flagsSet := [][]string{
		{
			"--client-protocol=grpc",
			"--rename-dir-limit=0",                                // Force use of Control API for folders instead of legacy workarounds
			"--experimental-nonrapid-folder-api-stall-retry=true", // Enable the new flag we are testing!
		},
	}

	for _, flags := range flagsSet {
		// Run test for 2s stall
		ts2s := &controlClientStall{
			flags:           flags,
			configPath:      "../configs/control_client_stall_2s.yaml",
			forcedStallTime: 2 * time.Second,
		}
		log.Printf("Running Control Client 2s Stall tests with flags: %s", ts2s.flags)
		suite.Run(t, ts2s)

		// Run test for 500ms stall
		ts500ms := &controlClientStall{
			flags:           flags,
			configPath:      "../configs/control_client_stall_500ms.yaml",
			forcedStallTime: 500 * time.Millisecond,
		}
		log.Printf("Running Control Client 500ms Stall tests with flags: %s", ts500ms.flags)
		suite.Run(t, ts500ms)
	}
}
