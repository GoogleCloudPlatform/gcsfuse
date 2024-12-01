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
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting"
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
	expectedMountTime time.Duration = 2500 * time.Millisecond
	logfilePathPrefix string        = "/tmp/gcsfuse_mount_timeout_"
)

var _ SetUpInterface = &MountTimeoutTest{}
var _ TearDownInterface = &MountTimeoutTest{}

func init() { RegisterTestSuite(&MountTimeoutTest{}) }

func (testSuite *MountTimeoutTest) SetUp(_ *TestInfo) {
	var err error
	testSuite.gcsfusePath = path.Join(gBuildDir, "bin/gcsfuse")
	// Set up the temporary directory.
	testSuite.dir, err = os.MkdirTemp("", "mount_timeout_test")
	AssertEq(nil, err)
}

func (testSuite *MountTimeoutTest) TearDown() {
	err := os.Remove(testSuite.dir)
	AssertEq(nil, err)
}

// mountOrTimeout mounts the bucket with the given client protocol. If the time taken
// exceeds threshold value of 2.5 seconds, an error is thrown and test will fail.
func (testSuite *MountTimeoutTest) mountOrTimeout(bucketName, mountDir, clientProtocol string) error {
	start := time.Now()
	args := []string{"--client-protocol", clientProtocol, bucketName, testSuite.dir}
	if err := mounting.MountGcsfuse(testSuite.gcsfusePath, args); err != nil {
		return err
	}
	defer func() {
		if err := util.Unmount(mountDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: unmount failed: %v\n", err)
		}
	}()

	if mountTime := time.Since(start); mountTime > expectedMountTime {
		return fmt.Errorf("[Client Protocol: %s]Mounting failed due to timeout(exceeding %f seconds).Time taken for the mount: %f sec", clientProtocol, expectedMountTime.Seconds(), mountTime.Seconds())
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
		setup.SetLogFile(fmt.Sprintf("%s%s.txt", logfilePathPrefix, tc.name))
		err := testSuite.mountOrTimeout(setup.TestBucket(), testSuite.dir, string(tc.clientProtocol))
		ExpectEq(nil, err)
	}
}
