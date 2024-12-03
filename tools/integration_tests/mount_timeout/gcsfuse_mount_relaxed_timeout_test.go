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

package mount_timeout

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestMountTimeoutWithRelaxedThreshold(t *testing.T) {
	if os.Getenv("TEST_ENV") == testEnvGCENonUSCentral {
		suite.Run(t, new(MountTimeoutTestWithRelaxedThreshold))
	} else {
		t.Skip()
	}
}

type MountTimeoutTestWithRelaxedThreshold struct {
	suite.Suite
	// Path to the gcsfuse binary.
	gcsfusePath string

	// A temporary directory into which a file system may be mounted. Removed in
	// TearDown.
	dir string
}

func (testSuite *MountTimeoutTestWithRelaxedThreshold) SetupTest() {
	var err error
	testSuite.gcsfusePath = path.Join(gBuildDir, "bin/gcsfuse")
	// Set up the temporary directory.
	testSuite.dir, err = os.MkdirTemp("", "mount_timeout_test")
	assert.NoError(testSuite.T(), err)
}

func (testSuite *MountTimeoutTestWithRelaxedThreshold) TearDownTest() {
	err := os.Remove(testSuite.dir)
	assert.NoError(testSuite.T(), err)
}

// mountOrTimeout mounts the bucket with the given client protocol. If the time taken
// exceeds threshold value of 2.5 seconds, an error is thrown and test will fail.
func (testSuite *MountTimeoutTestWithRelaxedThreshold) mountOrTimeout(bucketName, mountDir, clientProtocol string, expectedMountTime time.Duration) error {
	args := []string{"--client-protocol", clientProtocol, bucketName, testSuite.dir}
	start := time.Now()
	if err := mounting.MountGcsfuse(testSuite.gcsfusePath, args); err != nil {
		return err
	}
	defer func() {
		if err := util.Unmount(mountDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: unmount failed: %v\n", err)
		}
	}()

	if mountTime := time.Since(start); mountTime > expectedMountTime {
		return fmt.Errorf("[Client Protocol: %s]Mounting failed due to timeout(exceeding %f seconds).Time taken for the mounting %s: %f sec", clientProtocol, expectedMountTime.Seconds(), bucketName, mountTime.Seconds())
	}
	return nil
}

func (testSuite *MountTimeoutTestWithRelaxedThreshold) TestMountMultiRegionUSBucketWithTimeout() {
	testCases := []struct {
		name           string
		clientProtocol cfg.Protocol
	}{
		{
			name:           "multiRegionUSClientProtocolGRPC",
			clientProtocol: cfg.GRPC,
		},
		{
			name:           "multiRegionUSClientProtocolHttp1",
			clientProtocol: cfg.HTTP1,
		},
		{
			name:           "multiRegionUSClientProtocolHttp2",
			clientProtocol: cfg.HTTP2,
		},
	}
	for _, tc := range testCases {
		setup.SetLogFile(fmt.Sprintf("%s%s.txt", logfilePathPrefix, tc.name))

		err := testSuite.mountOrTimeout(multiRegionUSBucket, testSuite.dir, string(tc.clientProtocol), relaxedExpectedMountTime)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *MountTimeoutTestWithRelaxedThreshold) TestMountMultiRegionAsiaBucketWithTimeout() {
	testCases := []struct {
		name           string
		clientProtocol cfg.Protocol
	}{
		{
			name:           "multiRegionAsiaClientProtocolGRPC",
			clientProtocol: cfg.GRPC,
		},
		{
			name:           "multiRegionAsiaClientProtocolHttp1",
			clientProtocol: cfg.HTTP1,
		},
		{
			name:           "multiRegionAsiaClientProtocolHttp2",
			clientProtocol: cfg.HTTP2,
		},
	}
	for _, tc := range testCases {
		setup.SetLogFile(fmt.Sprintf("%s%s.txt", logfilePathPrefix, tc.name))

		err := testSuite.mountOrTimeout(multiRegionAsiaBucket, testSuite.dir, string(tc.clientProtocol), relaxedExpectedMountTime)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *MountTimeoutTestWithRelaxedThreshold) TestMountDualRegionUSBucketWithTimeout() {
	testCases := []struct {
		name           string
		clientProtocol cfg.Protocol
	}{
		{
			name:           "dualRegionUSClientProtocolGRPC",
			clientProtocol: cfg.GRPC,
		},
		{
			name:           "dualRegionUSClientProtocolHttp1",
			clientProtocol: cfg.HTTP1,
		},
		{
			name:           "dualRegionUSClientProtocolHttp2",
			clientProtocol: cfg.HTTP2,
		},
	}
	for _, tc := range testCases {
		setup.SetLogFile(fmt.Sprintf("%s%s.txt", logfilePathPrefix, tc.name))

		err := testSuite.mountOrTimeout(dualRegionUSBucket, testSuite.dir, string(tc.clientProtocol), relaxedExpectedMountTime)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *MountTimeoutTestWithRelaxedThreshold) TestMountDualRegionAsiaBucketWithTimeout() {
	testCases := []struct {
		name           string
		clientProtocol cfg.Protocol
	}{
		{
			name:           "dualRegionAsiaClientProtocolGRPC",
			clientProtocol: cfg.GRPC,
		},
		{
			name:           "dualRegionAsiaClientProtocolHttp1",
			clientProtocol: cfg.HTTP1,
		},
		{
			name:           "dualRegionAsiaClientProtocolHttp2",
			clientProtocol: cfg.HTTP2,
		},
	}
	for _, tc := range testCases {
		setup.SetLogFile(fmt.Sprintf("%s%s.txt", logfilePathPrefix, tc.name))

		err := testSuite.mountOrTimeout(dualRegionAsiaBucket, testSuite.dir, string(tc.clientProtocol), relaxedExpectedMountTime)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *MountTimeoutTestWithRelaxedThreshold) TestMountSingleRegionUSBucketWithTimeout() {
	testCases := []struct {
		name           string
		clientProtocol cfg.Protocol
	}{
		{
			name:           "singleRegionUSClientProtocolGRPC",
			clientProtocol: cfg.GRPC,
		},
		{
			name:           "singleRegionUSClientProtocolHttp1",
			clientProtocol: cfg.HTTP1,
		},
		{
			name:           "singleRegionUSClientProtocolHttp2",
			clientProtocol: cfg.HTTP2,
		},
	}
	for _, tc := range testCases {
		setup.SetLogFile(fmt.Sprintf("%s%s.txt", logfilePathPrefix, tc.name))

		err := testSuite.mountOrTimeout(singleRegionUSCentralBucket, testSuite.dir, string(tc.clientProtocol), relaxedExpectedMountTime)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *MountTimeoutTestWithRelaxedThreshold) TestMountSingleRegionAsiaBucketWithTimeout() {
	testCases := []struct {
		name           string
		clientProtocol cfg.Protocol
	}{
		{
			name:           "singleRegionAsiaClientProtocolGRPC",
			clientProtocol: cfg.GRPC,
		},
		{
			name:           "singleRegionAsiaClientProtocolHttp1",
			clientProtocol: cfg.HTTP1,
		},
		{
			name:           "singleRegionAsiaClientProtocolHttp2",
			clientProtocol: cfg.HTTP2,
		},
	}
	for _, tc := range testCases {
		setup.SetLogFile(fmt.Sprintf("%s%s.txt", logfilePathPrefix, tc.name))

		err := testSuite.mountOrTimeout(singleRegionAsiaEastBucket, testSuite.dir, string(tc.clientProtocol), relaxedExpectedMountTime)
		assert.NoError(testSuite.T(), err)
	}
}
