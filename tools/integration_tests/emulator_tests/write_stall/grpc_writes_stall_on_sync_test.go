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
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	emulator_tests "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/emulator_tests/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupGrpcWriteStallTest starts the gRPC proxy with configPath, mounts GCSFuse
// with baseFlags, registers cleanup, and returns a unique test file path.
func setupGrpcWriteStallTest(t *testing.T, configPath string, baseFlags []string) string {
	t.Helper()
	flags := append([]string{}, baseFlags...)
	proxyServerLogFile := setup.CreateProxyServerLogFile(t)
	port, proxyProcessId, err := emulator_tests.StartProxyServer(configPath, proxyServerLogFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		setup.UnmountGCSFuse(rootDir)
		assert.NoError(t, emulator_tests.KillProxyServerProcess(proxyProcessId))
		setup.SaveGCSFuseLogFileInCaseOfFailure(t)
		setup.SaveProxyServerLogFileInCaseOfFailure(proxyServerLogFile, t)
	})
	flags = append(flags, fmt.Sprintf("--custom-endpoint=localhost:%d", port))
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	testDirPath = setup.SetupTestDirectory("WriteStall" + setup.GenerateRandomString(5))
	return path.Join(testDirPath, "file.txt")
}

// TestGrpcChunkTransferTimeoutInfinity verifies that when chunk-transfer-timeout-secs
// is 0 (disabled), a gRPC BidiWriteObject upload waits out the full induced stall
// duration without aborting early and completes successfully.
func TestGrpcChunkTransferTimeoutInfinity(t *testing.T) {
	flags := []string{
		"--client-protocol=grpc",
		"--anonymous-access",
		"--chunk-transfer-timeout-secs=0",
	}
	filePath := setupGrpcWriteStallTest(t, "../configs/grpc_write_stall_40s.yaml", flags)

	elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, elapsedTime, stallTime)
	info, err := os.Stat(filePath)
	require.NoError(t, err)
	assert.EqualValues(t, fileSize, info.Size())
}

// TestGrpcChunkTransferTimeout verifies that gRPC BidiWriteObject uploads
// abort a stalled chunk transfer after chunk-transfer-timeout-secs (both default
// 10s and custom 5s) and automatically retry to completion on single and multiple stalls.
func TestGrpcChunkTransferTimeout(t *testing.T) {
	flagSets := [][]string{
		{"--client-protocol=grpc", "--anonymous-access"},
		{"--client-protocol=grpc", "--anonymous-access", "--chunk-transfer-timeout-secs=5"},
	}

	stallScenarios := []struct {
		name            string
		configPath      string
		expectedTimeout func(int) time.Duration
	}{
		{
			name:       "SingleStall",
			configPath: "../configs/grpc_write_stall_40s.yaml",
			expectedTimeout: func(chunkTransferTimeoutSecs int) time.Duration {
				return time.Duration(chunkTransferTimeoutSecs) * time.Second
			},
		},
		{
			name:       "MultipleStalls",
			configPath: "../configs/grpc_write_stall_twice_40s.yaml",
			expectedTimeout: func(chunkTransferTimeoutSecs int) time.Duration {
				return time.Duration(chunkTransferTimeoutSecs*2) * time.Second
			},
		},
	}

	for _, baseFlags := range flagSets {
		chunkTransferTimeoutSecs, err := emulator_tests.GetChunkTransferTimeoutFromFlags(baseFlags)
		require.NoError(t, err)

		t.Run(fmt.Sprintf("Timeout_%ds", chunkTransferTimeoutSecs), func(t *testing.T) {
			for _, scenario := range stallScenarios {
				t.Run(scenario.name, func(t *testing.T) {
					filePath := setupGrpcWriteStallTest(t, scenario.configPath, baseFlags)

					elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

					require.NoError(t, err, "failed to write file and sync")
					expectedTimeout := scenario.expectedTimeout(chunkTransferTimeoutSecs)
					assert.GreaterOrEqual(t, elapsedTime, expectedTimeout)
					assert.Less(t, elapsedTime, expectedTimeout+10*time.Second)
					info, err := os.Stat(filePath)
					require.NoError(t, err)
					assert.EqualValues(t, fileSize, info.Size())
				})
			}
		})
	}
}

// TestGrpcChunkRetryDeadline_SucceedsWhenStallUnderDeadline verifies that repeated
// gRPC chunk stalls succeed when total stall time (4 * 10s = 40s) is within
// chunk-retry-deadline-secs (120s).
func TestGrpcChunkRetryDeadline_SucceedsWhenStallUnderDeadline(t *testing.T) {
	flags := []string{
		"--client-protocol=grpc",
		"--anonymous-access",
		"--chunk-transfer-timeout-secs=10",
		"--chunk-retry-deadline-secs=120",
	}
	filePath := setupGrpcWriteStallTest(t, "../configs/grpc_write_stalls_four_times_60s.yaml", flags)

	elapsedTime, err := emulator_tests.WriteFileAndSync(filePath, fileSize)

	require.NoError(t, err, "expected success when total stall time is under chunk-retry-deadline-secs")
	assert.GreaterOrEqual(t, elapsedTime, 40*time.Second)
	assert.Less(t, elapsedTime, 55*time.Second)
	info, err := os.Stat(filePath)
	require.NoError(t, err)
	assert.EqualValues(t, fileSize, info.Size())
}

// TestGrpcChunkRetryDeadline_FailsWhenStallExceedsDeadline verifies that repeated
// gRPC chunk stalls fail once total stall time (4 * 10s = 40s) exceeds
// chunk-retry-deadline-secs (32s).
func TestGrpcChunkRetryDeadline_FailsWhenStallExceedsDeadline(t *testing.T) {
	flags := []string{
		"--client-protocol=grpc",
		"--anonymous-access",
		"--chunk-transfer-timeout-secs=10",
		"--chunk-retry-deadline-secs=32",
	}
	filePath := setupGrpcWriteStallTest(t, "../configs/grpc_write_stalls_four_times_60s.yaml", flags)

	startTime := time.Now()
	_, err := emulator_tests.WriteFileAndSync(filePath, fileSize)
	totalDuration := time.Since(startTime)

	require.Error(t, err, "expected failure when total stall time exceeds chunk-retry-deadline-secs")
	assert.GreaterOrEqual(t, totalDuration, 32*time.Second)
	assert.Less(t, totalDuration, 60*time.Second)
}
