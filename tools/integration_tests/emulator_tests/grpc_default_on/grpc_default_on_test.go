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
	"os/exec"
	"path"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	emulator_tests "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/emulator_tests/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

// startRejectingLocalListener binds an ephemeral local TCP listener for the lifetime of t
// and immediately closes every accepted connection.
func startRejectingLocalListener(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, listener.Close())
	})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Dropping the connection is the point; its close error is irrelevant.
			_ = conn.Close()
		}
	}()
	return listener.Addr().String()
}

// openFUSEDevViaFusermount mounts dir using fusermount and returns the pre-opened /dev/fuse
// file descriptor so gcsfuse can mount "/dev/fd/3" just like GKE's CSI driver sidecar.
func openFUSEDevViaFusermount(t *testing.T, dir string) *os.File {
	t.Helper()
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	require.NoError(t, err)
	parentSock := os.NewFile(uintptr(fds[0]), "fusermount-parent")
	childSock := os.NewFile(uintptr(fds[1]), "fusermount-child")
	defer func() {
		assert.NoError(t, parentSock.Close())
	}()

	fusermountCmd := exec.Command("fusermount", "-o", "rw,nosuid,nodev,fsname=gcsfuse,subtype=gcsfuse", "--", dir)
	fusermountCmd.Env = append(os.Environ(), "_FUSE_COMMFD=3")
	fusermountCmd.ExtraFiles = []*os.File{childSock}
	output, err := fusermountCmd.CombinedOutput()
	assert.NoError(t, childSock.Close())
	require.NoError(t, err, "fusermount failed: %s", string(output))

	buf := make([]byte, 4)
	oob := make([]byte, syscall.CmsgSpace(4))
	_, oobn, _, _, err := syscall.Recvmsg(int(parentSock.Fd()), buf, oob, 0)
	require.NoError(t, err)
	scms, err := syscall.ParseSocketControlMessage(oob[:oobn])
	require.NoError(t, err)
	require.Len(t, scms, 1)
	receivedFDs, err := syscall.ParseUnixRights(&scms[0])
	require.NoError(t, err)
	require.Len(t, receivedFDs, 1)

	return os.NewFile(uintptr(receivedFDs[0]), "/dev/fuse")
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

type grpcDefaultOnSuite struct {
	suite.Suite
	// gRPC proxy that rejects requests without DirectPath headers.
	grpcProxyPort      int
	grpcProxyProcessId int
	grpcProxyLogFile   string
	httpProxyPort      int
	httpProxyProcessId int
	httpProxyLogFile   string
	logFile            string
	testDirPath        string
	unmount            func()
}

// mountGCSFuseAsGKE mounts GCSFuse over "/dev/fd/3" like GKE's CSI sidecar. static_mounting can't be used:
// in background mode daemonize.Run passes only its status pipe as FD 3, so /dev/fuse would never reach gcsfuse.
func (s *grpcDefaultOnSuite) mountGCSFuseAsGKE(flags []string) {
	t := s.T()
	t.Helper()
	// Mount rootDir via fusermount and get the pre-opened /dev/fuse connection.
	fuseFile := openFUSEDevViaFusermount(t, rootDir)
	s.unmount = func() {
		setup.UnmountGCSFuse(rootDir)
	}

	args := append([]string{"--foreground", "--log-severity=trace", "--log-file=" + s.logFile}, flags...)
	// Pass "/dev/fd/3" to mark IsGKE=true and mount gcsfuse over FD 3.
	args = append(args, setup.TestBucket(), "/dev/fd/3")

	cmd := exec.Command(setup.BinFile(), args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	// Attach fuseFile as FD 3 (ExtraFiles[0]) in the spawned gcsfuse child process.
	cmd.ExtraFiles = []*os.File{fuseFile}
	startErr := cmd.Start()
	// Close the parent's copy so os.Stat below fails fast instead of hanging if gcsfuse exits early.
	assert.NoError(t, fuseFile.Close())
	require.NoError(t, startErr)

	s.unmount = func() {
		setup.UnmountGCSFuse(rootDir)
		assert.NoError(t, cmd.Wait(), "gcsfuse did not exit cleanly after unmount")
	}

	_, err := os.Stat(rootDir)
	require.NoError(t, err, "Failed waiting for GCSFuse mount on /dev/fd/3")
}

func (s *grpcDefaultOnSuite) mountGCSFuseStatic(flags []string) {
	t := s.T()
	t.Helper()
	config := &test_suite.TestConfig{
		TestBucket:              setup.TestBucket(),
		GCSFuseMountedDirectory: rootDir,
		LogFile:                 s.logFile,
	}
	err := static_mounting.MountGcsfuseWithStaticMountingWithConfigFile(config, flags)
	require.NoError(t, err, "Expected mount to succeed for flags %v", flags)

	s.unmount = func() {
		setup.UnmountGCSFuse(rootDir)
	}
}

func (s *grpcDefaultOnSuite) SetupTest() {
	var err error
	s.grpcProxyLogFile = setup.CreateProxyServerLogFile(s.T())
	s.grpcProxyPort, s.grpcProxyProcessId, err = emulator_tests.StartProxyServer("../configs/grpc_directpath_validation.yaml", s.grpcProxyLogFile)
	require.NoError(s.T(), err)

	s.httpProxyLogFile = setup.CreateProxyServerLogFile(s.T())
	s.httpProxyPort, s.httpProxyProcessId, err = emulator_tests.StartProxyServer("../configs/config.yaml", s.httpProxyLogFile)
	require.NoError(s.T(), err)

	// StartProxyServer points STORAGE_EMULATOR_HOST at the last proxy started (without a scheme), so set it explicitly.
	s.T().Setenv("STORAGE_EMULATOR_HOST", fmt.Sprintf("http://localhost:%d", s.httpProxyPort))
}

func (s *grpcDefaultOnSuite) SetupSubTest() {
	t := s.T()
	s.logFile = path.Join(setup.TestDir(), fmt.Sprintf("grpc_default_on_%s.log", setup.GenerateRandomString(5)))
	setup.SetLogFile(s.logFile)
	// Create the log file up front so failure-log saving works even if gcsfuse never starts.
	require.NoError(t, os.WriteFile(s.logFile, nil, 0644))
	// Clear proxy logs so each subtest is checked against only its own traffic.
	require.NoError(t, os.Truncate(s.grpcProxyLogFile, 0))
	require.NoError(t, os.Truncate(s.httpProxyLogFile, 0))
}

// cleanUpAndUnmount deletes the test dir while still mounted, then unmounts; safe to call twice.
func (s *grpcDefaultOnSuite) cleanUpAndUnmount() {
	if s.testDirPath != "" {
		setup.CleanUpDir(s.testDirPath)
		s.testDirPath = ""
	}
	if s.unmount != nil {
		s.unmount()
		s.unmount = nil
	}
}

func (s *grpcDefaultOnSuite) TearDownSubTest() {
	s.cleanUpAndUnmount()
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.SaveProxyServerLogFileInCaseOfFailure(s.grpcProxyLogFile, s.T())
	setup.SaveProxyServerLogFileInCaseOfFailure(s.httpProxyLogFile, s.T())
}

func (s *grpcDefaultOnSuite) TearDownTest() {
	for _, proxyProcessId := range []int{s.grpcProxyProcessId, s.httpProxyProcessId} {
		if proxyProcessId > 0 {
			assert.NoError(s.T(), emulator_tests.KillProxyServerProcess(proxyProcessId))
		}
	}
}

func (s *grpcDefaultOnSuite) TestProtocolSelection() {
	testCases := []struct {
		name                  string
		mountAsGKE            bool
		directPathUnavailable bool
		// explicitGRPC sends gRPC straight to the emulator, since explicit --client-protocol=grpc doesn't send
		// the DirectPath headers the gRPC proxy requires. No HTTP traffic plus a working read/write proves gRPC.
		explicitGRPC     bool
		flags            []string
		expectedProtocol cfg.Protocol
	}{
		// --enable-grpc-by-default set directly
		{
			name:             "EnableGRPCByDefaultTrue_DirectPathAvailable_UsesGRPC",
			flags:            []string{"--enable-grpc-by-default=true"},
			expectedProtocol: cfg.GRPC,
		},
		{
			name:                  "EnableGRPCByDefaultTrue_DirectPathUnavailable_FallsBackToHTTP",
			directPathUnavailable: true,
			flags:                 []string{"--enable-grpc-by-default=true"},
			expectedProtocol:      cfg.HTTP1,
		},
		{
			name:             "EnableGRPCByDefaultFalse_UsesHTTP",
			flags:            []string{"--enable-grpc-by-default=false"},
			expectedProtocol: cfg.HTTP1,
		},
		// GKE: an explicit --enable-grpc-by-default overrides the machine-type optimization in both directions.
		// false is passed by the CSI driver on LRO-unsupported ct6e-* machines.
		{
			name:             "GKE_TPUv6e_EnableGRPCByDefaultFalse_UsesHTTP",
			mountAsGKE:       true,
			flags:            []string{"--machine-type=ct6e-standard-4t", "--enable-grpc-by-default=false"},
			expectedProtocol: cfg.HTTP1,
		},
		{
			name:             "GKE_LowEndMachine_EnableGRPCByDefaultTrue_UsesGRPC",
			mountAsGKE:       true,
			flags:            []string{"--machine-type=n1-standard-1", "--enable-grpc-by-default=true"},
			expectedProtocol: cfg.GRPC,
		},
		// Flag unset: machine-type optimization enables gRPC on high-performance machines, on GKE only
		{
			name:             "GKE_HighEndMachine_UsesGRPC",
			mountAsGKE:       true,
			flags:            []string{"--machine-type=a3-highgpu-8g"},
			expectedProtocol: cfg.GRPC,
		},
		{
			name:             "GKE_LowEndMachine_UsesHTTP",
			mountAsGKE:       true,
			flags:            []string{"--machine-type=n1-standard-1"},
			expectedProtocol: cfg.HTTP1,
		},
		{
			name:             "NonGKE_HighEndMachine_UsesHTTP",
			flags:            []string{"--machine-type=a3-highgpu-8g"},
			expectedProtocol: cfg.HTTP1,
		},
		// Customer-specified --client-protocol takes precedence over all defaults
		{
			name:             "EnableGRPCByDefaultTrue_ClientProtocolHTTP1_UsesHTTP",
			flags:            []string{"--enable-grpc-by-default=true", "--client-protocol=http1"},
			expectedProtocol: cfg.HTTP1,
		},
		{
			name:             "GKE_HighEndMachine_ClientProtocolHTTP1_UsesHTTP",
			mountAsGKE:       true,
			flags:            []string{"--machine-type=a3-highgpu-8g", "--client-protocol=http1"},
			expectedProtocol: cfg.HTTP1,
		},
		{
			name:             "GKE_TPUv6e_EnableGRPCByDefaultFalse_ClientProtocolGRPC_UsesGRPC",
			mountAsGKE:       true,
			explicitGRPC:     true,
			flags:            []string{"--machine-type=ct6e-standard-4t", "--enable-grpc-by-default=false", "--client-protocol=grpc"},
			expectedProtocol: cfg.GRPC,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			t := s.T()
			// --enable-hns=false skips the gRPC control client, which the emulator proxies don't support.
			// --anonymous-access is required for gRPC to a localhost endpoint.
			mountFlags := append([]string{"--enable-hns=false", "--anonymous-access"}, tc.flags...)
			switch {
			case tc.directPathUnavailable:
				t.Setenv("STORAGE_EMULATOR_HOST_GRPC", startRejectingLocalListener(t))
			case tc.explicitGRPC:
				// The emulator's gRPC port, as started by emulator_tests.sh.
				t.Setenv("STORAGE_EMULATOR_HOST_GRPC", "localhost:8888")
			default:
				t.Setenv("STORAGE_EMULATOR_HOST_GRPC", fmt.Sprintf("localhost:%d", s.grpcProxyPort))
			}

			if tc.mountAsGKE {
				s.mountGCSFuseAsGKE(mountFlags)
			} else {
				s.mountGCSFuseStatic(mountFlags)
			}

			s.testDirPath = setup.SetupTestDirectory(t.Name())
			verifyFileReadWrite(t, s.testDirPath)
			// Unmount before reading proxy logs so cleanup/unmount traffic is also checked.
			s.cleanUpAndUnmount()

			grpcProxyLog, err := os.ReadFile(s.grpcProxyLogFile)
			require.NoError(t, err)
			httpProxyLog, err := os.ReadFile(s.httpProxyLogFile)
			require.NoError(t, err)

			switch tc.expectedProtocol {
			case cfg.GRPC:
				assert.Empty(t, string(httpProxyLog))
				if !tc.explicitGRPC {
					assert.Contains(t, string(grpcProxyLog), "/google.storage.v2.Storage/BidiWriteObject")
					assert.Contains(t, string(grpcProxyLog), "/google.storage.v2.Storage/ReadObject")
				}
			case cfg.HTTP1:
				assert.Contains(t, string(httpProxyLog), "RequestType: JsonCreate")
				assert.Contains(t, string(httpProxyLog), "RequestType: XmlRead")
				assert.Empty(t, string(grpcProxyLog))
			}
		})
	}
}

func TestGRPCDefaultOn(t *testing.T) {
	suite.Run(t, new(grpcDefaultOnSuite))
}
