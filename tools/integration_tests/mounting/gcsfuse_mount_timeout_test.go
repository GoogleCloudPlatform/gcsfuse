// Copyright 2024 Google LLC
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

package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/util"
	. "github.com/jacobsa/ogletest"
)

func TestMountTimeout(t *testing.T) { RunTests(t) }

type MountTimeoutTest struct {
	// Path to the gcsfuse binary.
	gcsfusePath string

	// A temporary directory into which a file system may be mounted. Removed in
	// TearDown.
	dir string
}

const (
	expectedMountTime float64 = 2.5
)

var _ SetUpInterface = &MountTimeoutTest{}
var _ TearDownInterface = &MountTimeoutTest{}

func init() { RegisterTestSuite(&MountTimeoutTest{}) }

func (t *MountTimeoutTest) SetUp(_ *TestInfo) {
	var err error
	t.gcsfusePath = path.Join(gBuildDir, "bin/gcsfuse")
	// Set up the temporary directory.
	t.dir, err = os.MkdirTemp("", "mount_timeout_test")
	AssertEq(nil, err)
}

func (t *MountTimeoutTest) TearDown() {
	err := os.Remove(t.dir)
	AssertEq(nil, err)
}

// Create an appropriate exec.Cmd for running gcsfuse, setting the required
// environment.
func (t *MountTimeoutTest) gcsfuseCommand(args []string, env []string) (cmd *exec.Cmd) {
	cmd = exec.Command(t.gcsfusePath, args...)
	cmd.Env = make([]string, len(env))
	copy(cmd.Env, env)

	// Teach gcsfuse where fusermount lives.
	cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s", path.Dir(gFusermountPath)))

	return
}

// Call gcsfuse with the supplied args and environment variable,
// waiting for it to exit. Return nil only if it exits successfully.
func (t *MountTimeoutTest) runGcsfuseWithEnv(args []string, env []string) (err error) {
	cmd := t.gcsfuseCommand(args, env)

	// Run.
	output, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("error %w running gcsfuse; output:\n%s", err, output)
		return
	}

	return
}

// Call gcsfuse with the supplied args, waiting for it to exit. Return nil only
// if it exits successfully.
func (t *MountTimeoutTest) runGcsfuse(args []string) (err error) {
	return t.runGcsfuseWithEnv(args, nil)
}

func (t *MountTimeoutTest) mountOrTimeout(bucketName, mountDir, clientProtocol string) error {
	start := time.Now()
	args := []string{"--client-protocol", clientProtocol, bucketName, t.dir}
	if err := t.runGcsfuse(args); err != nil {
		return err
	}
	defer func() {
		if err := util.Unmount(mountDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: unmount failed: %v\n", err)
		}
	}()
	if mountTime := time.Since(start).Seconds(); mountTime > expectedMountTime {
		return fmt.Errorf("Mounting failed due to timeout(exceeding %f seconds).Time taken for the mount: %f sec.", expectedMountTime, mountTime)
	}
	return nil
}

func (testSuite *MountTimeoutTest) MountGcsfuseWithTimeout() {
	testCases := []struct {
		name           string
		clientProtocol cfg.Protocol
	}{
		{
			name:           "clientProtocolGRPC",
			clientProtocol: cfg.GRPC,
		},
		{
			name:           "clientProtocolHttp1",
			clientProtocol: cfg.HTTP1,
		},
		{
			name:           "clientProtocolHttp2",
			clientProtocol: cfg.HTTP2,
		},
	}
	for _, tc := range testCases {
		err := testSuite.mountOrTimeout(setup.TestBucket(), testSuite.dir, string(tc.clientProtocol))
		ExpectEq(nil, err)
	}
}
