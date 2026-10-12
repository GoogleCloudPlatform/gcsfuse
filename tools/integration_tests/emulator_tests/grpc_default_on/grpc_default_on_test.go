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
	"sync"
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

// openFUSEDevViaFusermount mounts dir using fusermount3 and returns the pre-opened /dev/fuse
// file descriptor so gcsfuse can mount "/dev/fd/3" just like GKE's CSI driver sidecar.
func openFUSEDevViaFusermount(t *testing.T, dir string) *os.File {
	t.Helper()
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	require.NoError(t, err)
	parentSock := os.NewFile(uintptr(fds[0]), "fusermount-parent")
	childSock := os.NewFile(uintptr(fds[1]), "fusermount-child")
	defer func() { _ = parentSock.Close() }()
	defer func() { _ = childSock.Close() }()

	fusermountCmd := exec.Command("fusermount3", "-o", "rw,nosuid,nodev,fsname=gcsfuse,subtype=gcsfuse", "--", dir)
	fusermountCmd.Env = append(os.Environ(), "_FUSE_COMMFD=3")
	fusermountCmd.ExtraFiles = []*os.File{childSock}
	output, err := fusermountCmd.CombinedOutput()
	_ = childSock.Close()
	require.NoError(t, err, "fusermount3 failed: %s", string(output))

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

func mountGCSFuseAsGKE(t *testing.T, dir string, logFile string, flags []string) func() {
	t.Helper()
	// Mount dir via fusermount3 and get the pre-opened /dev/fuse connection.
	fuseFile := openFUSEDevViaFusermount(t, dir)
	defer func() { _ = fuseFile.Close() }()

	args := append([]string{"--foreground", "--log-severity=trace", "--log-file=" + logFile}, flags...)
	// Pass "/dev/fd/3" to mark IsGKE=true and mount gcsfuse over FD 3.
	args = append(args, setup.TestBucket(), "/dev/fd/3")

	cmd := exec.Command(setup.BinFile(), args...)
	// Attach fuseFile as FD 3 (ExtraFiles[0]) in the spawned gcsfuse child process.
	cmd.ExtraFiles = []*os.File{fuseFile}
	require.NoError(t, cmd.Start())

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			setup.UnmountGCSFuse(dir)
			<-waitCh
		})
	}
	t.Cleanup(stop)

	_, err := os.Stat(dir)
	require.NoError(t, err, "Failed waiting for GCSFuse mount on /dev/fd/3")

	return stop
}

func mountGCSFuseStatic(t *testing.T, dir string, logFile string, flags []string) func() {
	t.Helper()
	config := &test_suite.TestConfig{
		TestBucket:              setup.TestBucket(),
		GCSFuseMountedDirectory: dir,
		LogFile:                 logFile,
	}
	err := static_mounting.MountGcsfuseWithStaticMountingWithConfigFile(config, flags)
	require.NoError(t, err, "Expected mount to succeed for flags %v", flags)

	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			setup.UnmountGCSFuse(dir)
		})
	}
	t.Cleanup(stop)
	return stop
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
	grpcProxyPort      int
	grpcProxyProcessId int
	grpcProxyLogFile   string
	httpProxyPort      int
	httpProxyProcessId int
	httpProxyLogFile   string
	logFile            string
}

func (s *grpcDefaultOnSuite) SetupTest() {
	var err error
	s.grpcProxyLogFile = setup.CreateProxyServerLogFile(s.T())
	s.grpcProxyPort, s.grpcProxyProcessId, err = emulator_tests.StartProxyServer("../configs/grpc_directpath_validation.yaml", s.grpcProxyLogFile)
	require.NoError(s.T(), err)

	s.httpProxyLogFile = setup.CreateProxyServerLogFile(s.T())
	s.httpProxyPort, s.httpProxyProcessId, err = emulator_tests.StartProxyServer("../configs/config.yaml", s.httpProxyLogFile)
	require.NoError(s.T(), err)

	s.T().Setenv("STORAGE_EMULATOR_HOST", fmt.Sprintf("http://localhost:%d", s.httpProxyPort))
}

func (s *grpcDefaultOnSuite) SetupSubTest() {
	s.logFile = path.Join(setup.TestDir(), fmt.Sprintf("grpc_default_on_%s.log", setup.GenerateRandomString(5)))
	setup.SetLogFile(s.logFile)
}

func (s *grpcDefaultOnSuite) TearDownTest() {
	assert.NoError(s.T(), emulator_tests.KillProxyServerProcess(s.grpcProxyProcessId))
	assert.NoError(s.T(), emulator_tests.KillProxyServerProcess(s.httpProxyProcessId))
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.SaveProxyServerLogFileInCaseOfFailure(s.grpcProxyLogFile, s.T())
	setup.SaveProxyServerLogFileInCaseOfFailure(s.httpProxyLogFile, s.T())
}

func (s *grpcDefaultOnSuite) TestGRPCDefaultOn() {
	testCases := []struct {
		name                  string
		mountAsGKE            bool
		directPathUnavailable bool
		flags                 []string
		expectedProtocol      cfg.Protocol
	}{
		// Direct --enable-grpc-by-default flag behavior
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
		{
			name:             "EnableGRPCByDefaultTrue_ClientProtocolHTTP1_UsesHTTP",
			flags:            []string{"--enable-grpc-by-default=true", "--client-protocol=http1"},
			expectedProtocol: cfg.HTTP1,
		},
		// Machine-type optimization behavior
		{
			name:             "GKE_HighEndMachine_UsesGRPC",
			mountAsGKE:       true,
			flags:            []string{"--machine-type=a3-highgpu-8g"},
			expectedProtocol: cfg.GRPC,
		},
		{
			name:             "GKE_HighEndMachine_EnableGRPCByDefaultFalse_UsesHTTP",
			mountAsGKE:       true,
			flags:            []string{"--machine-type=a3-highgpu-8g", "--enable-grpc-by-default=false"},
			expectedProtocol: cfg.HTTP1,
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
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			t := s.T()
			mountFlags := append([]string{"--enable-hns=false", "--anonymous-access"}, tc.flags...)
			if tc.directPathUnavailable {
				t.Setenv("STORAGE_EMULATOR_HOST_GRPC", startRejectingLocalListener(t))
			} else {
				t.Setenv("STORAGE_EMULATOR_HOST_GRPC", fmt.Sprintf("localhost:%d", s.grpcProxyPort))
			}

			var unmount func()
			if tc.mountAsGKE {
				unmount = mountGCSFuseAsGKE(t, rootDir, s.logFile, mountFlags)
			} else {
				unmount = mountGCSFuseStatic(t, rootDir, s.logFile, mountFlags)
			}

			testDirPath := setup.SetupTestDirectory(t.Name())
			require.NoError(t, os.Truncate(s.grpcProxyLogFile, 0))
			require.NoError(t, os.Truncate(s.httpProxyLogFile, 0))
			verifyFileReadWrite(t, testDirPath)
			setup.CleanUpDir(testDirPath)
			unmount()

			grpcProxyLog, err := os.ReadFile(s.grpcProxyLogFile)
			require.NoError(t, err)
			httpProxyLog, err := os.ReadFile(s.httpProxyLogFile)
			require.NoError(t, err)

			switch tc.expectedProtocol {
			case cfg.GRPC:
				assert.Contains(t, string(grpcProxyLog), "/google.storage.v2.Storage/BidiWriteObject")
				assert.Contains(t, string(grpcProxyLog), "/google.storage.v2.Storage/ReadObject")
				assert.Empty(t, string(httpProxyLog))
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
