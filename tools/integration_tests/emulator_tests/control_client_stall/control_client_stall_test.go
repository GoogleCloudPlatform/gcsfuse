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
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

type controlClientStallBase struct {
	port                 int
	proxyProcessId       int
	proxyServerLogFile   string
	flags                []string
	configFileName       string
	maxMountDurationSecs int
	forcedStallTime      time.Duration
	testDirPath          string
	suite.Suite
}

func (c *controlClientStallBase) SetupTest() {
	c.proxyServerLogFile = setup.CreateProxyServerLogFile(c.T())
	var err error
	c.port, c.proxyProcessId, err = emulator_tests.StartProxyServer(c.configFileName, c.proxyServerLogFile)
	require.NoError(c.T(), err)

	// Add proxy server endpoint and configure gRPC testing
	// Copy flags to avoid mutating the original slice across suites
	mountFlags := append([]string(nil), c.flags...)
	mountFlags = append(mountFlags, fmt.Sprintf("--custom-endpoint=localhost:%d", c.port))
	mountFlags = append(mountFlags, "--anonymous-access") // Required for gRPC localhost endpoint

	startTime := time.Now()
	setup.MountGCSFuseWithGivenMountFunc(mountFlags, static_mounting.MountGcsfuseWithStaticMounting)
	mountTime := time.Since(startTime)

	if c.maxMountDurationSecs != -1 {
		assert.True(c.T(), mountTime < time.Duration(c.maxMountDurationSecs)*time.Second, "Mount time %v should be less than %ds", mountTime, c.maxMountDurationSecs)
	}

	c.testDirPath = setup.SetupTestDirectory(c.T().Name())
}

func (c *controlClientStallBase) TearDownTest() {
	setup.UnmountGCSFuse(setup.MntDir())
	assert.NoError(c.T(), emulator_tests.KillProxyServerProcess(c.proxyProcessId))
	setup.SaveGCSFuseLogFileInCaseOfFailure(c.T())
	setup.SaveProxyServerLogFileInCaseOfFailure(c.proxyServerLogFile, c.T())
}

// --- Test Suites ---

type createFolderStallSuite struct{ controlClientStallBase }

// TestCreateFolderStallInducedShouldComplete verifies that creating a folder via
// os.MkdirAll completes successfully even when a stall is induced,
// proving that the experimental retry logic correctly handles the delayed gRPC response.
func (c *createFolderStallSuite) TestCreateFolderStallInducedShouldComplete() {
	folderPath := path.Join(c.testDirPath, "stalled_folder")

	startTime := time.Now()
	err := os.MkdirAll(folderPath, 0755)
	elapsedTime := time.Since(startTime)

	assert.NoError(c.T(), err)

	// Ensure the operation took less than 32 seconds because of the internal abortion and retry.
	assert.True(c.T(), elapsedTime < 32*time.Second, "Elapsed time %v should be less than 32s", elapsedTime)

	// Validate the directory actually exists
	info, err := os.Stat(folderPath)
	assert.NoError(c.T(), err)
	assert.True(c.T(), info.IsDir())
}

type getFolderStallSuite struct{ controlClientStallBase }

func (c *getFolderStallSuite) TestGetFolderStallInducedShouldComplete() {
	folderPath := path.Join(c.testDirPath, "stalled_folder_for_get")
	err := os.MkdirAll(folderPath, 0755)
	assert.NoError(c.T(), err)

	startTime := time.Now()
	_, err = os.Stat(folderPath)
	elapsedTime := time.Since(startTime)

	assert.NoError(c.T(), err)
	assert.True(c.T(), elapsedTime < 32*time.Second, "Elapsed time %v should be less than 32s", elapsedTime)
}

type deleteFolderStallSuite struct{ controlClientStallBase }

func (c *deleteFolderStallSuite) TestDeleteFolderStallInducedShouldComplete() {
	folderPath := path.Join(c.testDirPath, "stalled_folder_for_delete")
	err := os.MkdirAll(folderPath, 0755)
	assert.NoError(c.T(), err)

	startTime := time.Now()
	err = os.Remove(folderPath)
	elapsedTime := time.Since(startTime)

	assert.NoError(c.T(), err)
	assert.True(c.T(), elapsedTime < 32*time.Second, "Elapsed time %v should be less than 32s", elapsedTime)

	// Validate the directory actually deleted
	_, err = os.Stat(folderPath)
	assert.True(c.T(), os.IsNotExist(err))
}

type renameFolderStallSuite struct{ controlClientStallBase }

func (c *renameFolderStallSuite) TestRenameFolderStallInducedShouldComplete() {
	folderPath := path.Join(c.testDirPath, "stalled_folder_for_rename")
	err := os.MkdirAll(folderPath, 0755)
	assert.NoError(c.T(), err)

	destPath := path.Join(c.testDirPath, "stalled_folder_renamed")
	startTime := time.Now()
	err = os.Rename(folderPath, destPath)
	elapsedTime := time.Since(startTime)

	assert.NoError(c.T(), err)
	assert.True(c.T(), elapsedTime < 32*time.Second, "Elapsed time %v should be less than 32s", elapsedTime)

	// Validate the directory actually renamed
	_, err = os.Stat(destPath)
	assert.NoError(c.T(), err)
}

type getStorageLayoutStallSuite struct{ controlClientStallBase }

func (c *getStorageLayoutStallSuite) TestGetStorageLayoutStallInducedShouldComplete() {
	// The stall for GetStorageLayout is actually triggered during the bucket mount
	// phase within SetupTest() when GCSFuse queries the bucket layout to determine
	// if Hierarchical Namespace (HNS) is enabled.
	// The fact that this test function is executing means the mount succeeded,
	// effectively proving that GetStorageLayout correctly timed out after 30s,
	// retried successfully, and allowed the mount to complete!

	// Just perform a basic verification to ensure the mount is functional.
	folderPath := path.Join(c.testDirPath, "stalled_folder_for_layout")

	err := os.MkdirAll(folderPath, 0755)
	assert.NoError(c.T(), err)

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
			"--metadata-cache-ttl-secs=0",                         // Disable cache so calls definitely hit the backend
		},
	}

	for _, flags := range flagsSet {
		suites := []suite.TestingSuite{
			&createFolderStallSuite{
				controlClientStallBase: controlClientStallBase{
					flags:                flags,
					configFileName:       "../configs/control_client_stall_create_40s.yaml",
					maxMountDurationSecs: -1,
					forcedStallTime:      40 * time.Second,
				},
			},
			&getFolderStallSuite{
				controlClientStallBase: controlClientStallBase{
					flags:                flags,
					configFileName:       "../configs/control_client_stall_get_40s.yaml",
					maxMountDurationSecs: -1,
					forcedStallTime:      40 * time.Second,
				},
			},
			&deleteFolderStallSuite{
				controlClientStallBase: controlClientStallBase{
					flags:                flags,
					configFileName:       "../configs/control_client_stall_delete_40s.yaml",
					maxMountDurationSecs: -1,
					forcedStallTime:      40 * time.Second,
				},
			},
			&renameFolderStallSuite{
				controlClientStallBase: controlClientStallBase{
					flags:                flags,
					configFileName:       "../configs/control_client_stall_rename_40s.yaml",
					maxMountDurationSecs: -1,
					forcedStallTime:      40 * time.Second,
				},
			},
			&getStorageLayoutStallSuite{
				controlClientStallBase: controlClientStallBase{
					flags:                flags,
					configFileName:       "../configs/control_client_stall_layout_60s.yaml",
					maxMountDurationSecs: 40,
					forcedStallTime:      60 * time.Second,
				},
			},
		}

		for _, s := range suites {
			log.Printf("Running Control Client Stall test suite %T with flags: %s", s, flags)
			suite.Run(t, s)
		}
	}
}
