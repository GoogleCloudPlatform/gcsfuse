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

package write_stall

import (
	"path"
	"testing"
	"time"

	emulator_tests "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/emulator_tests/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrpcChunkTransferTimeout(t *testing.T) {
	baseFlags := []string{
		"--client-protocol=grpc",
		"--anonymous-access",
		"--chunk-transfer-timeout-secs=5",
	}

	stallScenarios := []struct {
		name            string
		configPath      string
		expectedTimeout time.Duration
	}{
		{
			name:            "SingleStall",
			configPath:      "../configs/grpc_write_stall_40s.yaml",
			expectedTimeout: 5 * time.Second,
		},
		{
			name:            "MultipleStalls",
			configPath:      "../configs/grpc_write_stall_twice_40s.yaml",
			expectedTimeout: 10 * time.Second,
		},
	}

	for _, scenario := range stallScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			flags := append([]string{}, baseFlags...)
			proxyServerLogFile := setup.CreateProxyServerLogFile(t)
			port, proxyProcessId, err := emulator_tests.StartProxyServer(scenario.configPath, proxyServerLogFile)
			require.NoError(t, err)
			defer func() {
				setup.UnmountGCSFuse(rootDir)
				assert.NoError(t, emulator_tests.KillProxyServerProcess(proxyProcessId))
				setup.SaveGCSFuseLogFileInCaseOfFailure(t)
				setup.SaveProxyServerLogFileInCaseOfFailure(proxyServerLogFile, t)
			}()
			setup.AppendProxyEndpointToFlagSet(&flags, port)
			setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)

			testDir := scenario.name + setup.GenerateRandomString(3)
			testDirPath = setup.SetupTestDirectory(testDir)
			filePath := path.Join(testDirPath, "file.txt")

			elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

			assert.NoError(t, err, "failed to write file and sync")
			assert.GreaterOrEqual(t, elapsedTime, scenario.expectedTimeout)
			assert.Less(t, elapsedTime, scenario.expectedTimeout+10*time.Second)
		})
	}
}

func TestGrpcChunkRetryDeadline(t *testing.T) {
	scenarios := []struct {
		name            string
		flags           []string
		expectedSuccess bool
		expectedStall   time.Duration
	}{
		{
			name: "StallUnderDeadline_Pass",
			flags: []string{
				"--client-protocol=grpc",
				"--anonymous-access",
				"--chunk-transfer-timeout-secs=10",
				"--chunk-retry-deadline-secs=120",
			},
			expectedStall:   40 * time.Second,
			expectedSuccess: true,
		},
		{
			name: "StallOverDeadline_Fail",
			flags: []string{
				"--client-protocol=grpc",
				"--anonymous-access",
				"--chunk-transfer-timeout-secs=10",
				"--chunk-retry-deadline-secs=32",
			},
			expectedStall:   40 * time.Second,
			expectedSuccess: false,
		},
	}
	configPath := "../configs/grpc_write_stalls_four_times_60s.yaml"

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			proxyServerLogFile := setup.CreateProxyServerLogFile(t)
			flags := append([]string{}, scenario.flags...)
			port, proxyProcessId, err := emulator_tests.StartProxyServer(configPath, proxyServerLogFile)
			require.NoError(t, err)
			defer func() {
				setup.UnmountGCSFuse(rootDir)
				assert.NoError(t, emulator_tests.KillProxyServerProcess(proxyProcessId))
				setup.SaveGCSFuseLogFileInCaseOfFailure(t)
				setup.SaveProxyServerLogFileInCaseOfFailure(proxyServerLogFile, t)
			}()
			setup.AppendProxyEndpointToFlagSet(&flags, port)
			setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
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
