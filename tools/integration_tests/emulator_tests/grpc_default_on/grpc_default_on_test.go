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

package grpc_default_on

import (
	"fmt"
	"net"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	emulator_tests "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/emulator_tests/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

// startRejectingLocalListener binds an ephemeral local TCP listener for the lifetime of t
// and immediately closes every accepted connection, simulating DirectPath failure without
// port reuse races (TOCTOU) or blackhole SYN timeouts.
func startRejectingLocalListener(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = listener.Close()
	})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	return listener.Addr().String()
}

func verifyFileReadWrite(t *testing.T, dirPath string) {
	t.Helper()
	filePath := path.Join(dirPath, fmt.Sprintf("default_on_item_%s.txt", setup.GenerateRandomString(5)))
	expectedContent := []byte("verifying file read/write with default protocol selection")

	require.NoError(t, os.WriteFile(filePath, expectedContent, 0644))
	defer func() {
		assert.NoError(t, os.Remove(filePath))
	}()

	actualContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, actualContent)
}

func TestGRPCDefaultOn(t *testing.T) {
	testCases := []struct {
		name                  string
		flags                 []string
		useDirectPathProxy    bool
		induceDirectPathError bool
		expectMountError      bool
		expectedLogSubstrings []string
		unexpectedLogString   string
	}{
		{
			name: "DirectPathAvailable_ConnectsViaGRPC",
			flags: []string{
				"--enable-grpc-by-default=true",
				"--enable-hns=false",
				"--anonymous-access",
				"--log-severity=TRACE",
			},
			useDirectPathProxy: true,
			expectedLogSubstrings: []string{
				"Verifying DirectPath connectivity for bucket",
				"DirectPath verification succeeded, continuing with DirectPath.",
			},
		},
		{
			name: "DirectPathUnavailable_FallsBackToHTTP",
			flags: []string{
				"--enable-grpc-by-default=true",
				"--enable-hns=false",
				"--anonymous-access",
				"--log-severity=TRACE",
			},
			induceDirectPathError: true,
			expectedLogSubstrings: []string{
				"DirectPath verification failed with error:",
				"Grpc dp is not available and falling back to Http.",
			},
		},
		{
			name: "DirectPathUnavailable_DirectPathOnlyFailsMount",
			flags: []string{
				"--enable-grpc-by-default=true",
				"--grpc-path-strategy=direct-path-only",
				"--enable-hns=false",
				"--anonymous-access",
				"--log-severity=TRACE",
			},
			induceDirectPathError: true,
			expectMountError:      true,
			expectedLogSubstrings: []string{
				"Grpc dp is not available and not falling back to Http as gRPC path strategy is set to DirectPathOnly",
			},
		},
		{
			name: "DisabledByDefault_UsesHTTPWithoutDirectPathProbe",
			flags: []string{
				"--enable-hns=false",
				"--anonymous-access",
				"--log-severity=TRACE",
			},
			unexpectedLogString: "Verifying DirectPath connectivity",
		},
		{
			name: "ExplicitHTTP1OverridesDefaultOn_UsesHTTPWithoutDirectPathProbe",
			flags: []string{
				"--enable-grpc-by-default=true",
				"--client-protocol=http1",
				"--enable-hns=false",
				"--anonymous-access",
				"--log-severity=TRACE",
			},
			unexpectedLogString: "Verifying DirectPath connectivity",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.induceDirectPathError {
				t.Setenv("STORAGE_EMULATOR_HOST_GRPC", startRejectingLocalListener(t))
			}

			mountFlags := append([]string{}, tc.flags...)
			var (
				proxyPid           int
				proxyServerLogFile string
			)
			stopProxy := func() {
				if proxyPid > 0 {
					assert.NoError(t, emulator_tests.KillProxyServerProcess(proxyPid))
					setup.SaveProxyServerLogFileInCaseOfFailure(proxyServerLogFile, t)
					proxyPid = 0
				}
			}
			if tc.useDirectPathProxy {
				proxyServerLogFile = setup.CreateProxyServerLogFile(t)
				port, pid, err := emulator_tests.StartProxyServer("../configs/grpc_directpath_validation.yaml", proxyServerLogFile)
				require.NoError(t, err)
				proxyPid = pid
				defer stopProxy()
				mountFlags = append(mountFlags, fmt.Sprintf("--custom-endpoint=localhost:%d", port))
			}

			logFile := path.Join(setup.TestDir(), fmt.Sprintf("grpc_default_on_%s.log", setup.GenerateRandomString(5)))
			setup.SetLogFile(logFile)
			defer setup.SaveGCSFuseLogFileInCaseOfFailure(t)

			config := &test_suite.TestConfig{
				TestBucket:              setup.TestBucket(),
				GKEMountedDirectory:     setup.MountedDirectory(),
				GCSFuseMountedDirectory: rootDir,
				LogFile:                 logFile,
			}

			err := setup.MayMountGCSFuseWithGivenMountWithConfigFunc(
				config,
				mountFlags,
				static_mounting.MountGcsfuseWithStaticMountingWithConfigFile,
			)
			mounted := false
			unmount := func() {
				if mounted {
					setup.UnmountGCSFuse(rootDir)
					mounted = false
				}
			}
			defer unmount()

			if tc.expectMountError {
				require.Error(t, err, "Expected mount to fail for flags %v", mountFlags)
			} else {
				require.NoError(t, err, "Expected mount to succeed for flags %v", mountFlags)
				mounted = true

				testDirPath := setup.SetupTestDirectory(t.Name())
				verifyFileReadWrite(t, testDirPath)
			}

			// Unmount GCSFuse and stop proxy server before inspecting logs so all log buffers are flushed.
			unmount()
			stopProxy()

			logBytes, err := os.ReadFile(logFile)
			require.NoError(t, err)
			logStr := string(logBytes)
			for _, expected := range tc.expectedLogSubstrings {
				assert.Contains(t, logStr, expected)
			}
			if tc.unexpectedLogString != "" {
				assert.NotContains(t, logStr, tc.unexpectedLogString)
			}

			if tc.useDirectPathProxy {
				proxyLogBytes, err := os.ReadFile(proxyServerLogFile)
				require.NoError(t, err)
				proxyLogStr := string(proxyLogBytes)
				assert.Contains(t, proxyLogStr, "Metadata validation passed")
				assert.Contains(t, proxyLogStr, "force_direct_connectivity=ENFORCED")
				assert.Contains(t, proxyLogStr, "/google.storage.v2.Storage/GetObject")
				assert.Contains(t, proxyLogStr, "/google.storage.v2.Storage/BidiWriteObject")
				assert.Contains(t, proxyLogStr, "/google.storage.v2.Storage/ReadObject")
			}
		})
	}
}
