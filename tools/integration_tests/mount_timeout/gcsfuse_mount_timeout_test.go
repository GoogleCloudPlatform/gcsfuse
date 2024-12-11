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
	"math"
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

const (
	iterations int = 10
)

func TestMountTimeout(t *testing.T) {
	if os.Getenv("TEST_ENV") == testEnvGCEUSCentral {
		// Set strict region based timeout values if testing environment is GCE VM in us-central.
		timeout := RegionWiseTimeouts{
			multiRegionUSTimeout:         multiRegionUSExpectedMountTime,
			multiRegionAsiaTimeout:       multiRegionAsiaExpectedMountTime,
			dualRegionUSTimeout:          dualRegionUSExpectedMountTime,
			dualRegionAsiaTimeout:        dualRegionAsiaExpectedMountTime,
			singleRegionUSCentralTimeout: singleRegionUSCentralExpectedMountTime,
			singleRegionAsiaEastTimeout:  singleRegionAsiaEastExpectedMountTime,
		}
		t.Log("Running tests with region based timeout values since the GCE VM is located in us-central...\n")
		suite.Run(t, &MountTimeoutTest{timeouts: timeout})
	} else if os.Getenv("TEST_ENV") == testEnvGCENonUSCentral {
		// Set common relaxed timeout values if testing environment is GCE VM not in us-central.
		timeout := RegionWiseTimeouts{
			multiRegionUSTimeout:         relaxedExpectedMountTime,
			multiRegionAsiaTimeout:       relaxedExpectedMountTime,
			dualRegionUSTimeout:          relaxedExpectedMountTime,
			dualRegionAsiaTimeout:        relaxedExpectedMountTime,
			singleRegionUSCentralTimeout: relaxedExpectedMountTime,
			singleRegionAsiaEastTimeout:  relaxedExpectedMountTime,
		}
		t.Logf("Running tests with relaxed timeout of %f sec for all scenarios since the GCE VM is not located in us-central...\n", relaxedExpectedMountTime.Seconds())
		suite.Run(t, &MountTimeoutTest{timeouts: timeout})
	} else {
		// Skip the tests if the testing environment is not GCE VM.
		t.Log("Skipping tests since the testing environment is not GCE VM...\n")
		t.Skip()
	}
}

type RegionWiseTimeouts struct {
	multiRegionUSTimeout         time.Duration
	multiRegionAsiaTimeout       time.Duration
	dualRegionUSTimeout          time.Duration
	dualRegionAsiaTimeout        time.Duration
	singleRegionUSCentralTimeout time.Duration
	singleRegionAsiaEastTimeout  time.Duration
}

type MountTimeoutTest struct {
	suite.Suite
	// Path to the gcsfuse binary.
	gcsfusePath string

	// A temporary directory into which a file system may be mounted. Removed in
	// TearDown.
	dir      string
	timeouts RegionWiseTimeouts
}

func (testSuite *MountTimeoutTest) SetupTest() {
	var err error
	testSuite.gcsfusePath = path.Join(gBuildDir, "bin/gcsfuse")
	// Set up the temporary directory.
	testSuite.dir, err = os.MkdirTemp("", "mount_timeout_test")
	assert.NoError(testSuite.T(), err)
}

func (testSuite *MountTimeoutTest) TearDownTest() {
	err := os.Remove(testSuite.dir)
	assert.NoError(testSuite.T(), err)
}

// mountOrTimeout mounts the bucket with the given client protocol. If the time taken
// exceeds the expected for the particular test case , an error is thrown and test will fail.
func (testSuite *MountTimeoutTest) mountOrTimeout(bucketName, mountDir, clientProtocol string, expectedMountTime time.Duration) error {
	minMountTime := time.Duration(math.MaxInt64)

	// Iterating 10 times to account for randomness in time taken to mount.
	for i := 0; i < iterations; i++ {
		args := []string{"--client-protocol", clientProtocol, bucketName, testSuite.dir}
		start := time.Now()
		if err := mounting.MountGcsfuse(testSuite.gcsfusePath, args); err != nil {
			return err
		}
		mountTime := time.Since(start)

		minMountTime = time.Duration(math.Min(float64(minMountTime), float64(mountTime)))

		if err := util.Unmount(mountDir); err != nil {
			err = fmt.Errorf("Warning: unmount failed: %v\n", err)
			return err
		}
	}

	if minMountTime > expectedMountTime {
		return fmt.Errorf("[Client Protocol: %s] Mounting failed due to timeout (exceeding %f seconds). Time taken for mounting %s: %f sec", clientProtocol, expectedMountTime.Seconds(), bucketName, minMountTime.Seconds())
	}
	return nil
}

func (testSuite *MountTimeoutTest) TestMountMultiRegionUSBucketWithTimeout() {
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

		err := testSuite.mountOrTimeout(multiRegionUSBucket, testSuite.dir, string(tc.clientProtocol), testSuite.timeouts.multiRegionUSTimeout)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *MountTimeoutTest) TestMountMultiRegionAsiaBucketWithTimeout() {
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

		err := testSuite.mountOrTimeout(multiRegionAsiaBucket, testSuite.dir, string(tc.clientProtocol), testSuite.timeouts.multiRegionAsiaTimeout)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *MountTimeoutTest) TestMountDualRegionUSBucketWithTimeout() {
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

		err := testSuite.mountOrTimeout(dualRegionUSBucket, testSuite.dir, string(tc.clientProtocol), testSuite.timeouts.dualRegionUSTimeout)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *MountTimeoutTest) TestMountDualRegionAsiaBucketWithTimeout() {
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

		err := testSuite.mountOrTimeout(dualRegionAsiaBucket, testSuite.dir, string(tc.clientProtocol), testSuite.timeouts.dualRegionAsiaTimeout)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *MountTimeoutTest) TestMountSingleRegionUSBucketWithTimeout() {
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

		err := testSuite.mountOrTimeout(singleRegionUSCentralBucket, testSuite.dir, string(tc.clientProtocol), testSuite.timeouts.singleRegionUSCentralTimeout)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *MountTimeoutTest) TestMountSingleRegionAsiaBucketWithTimeout() {
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

		err := testSuite.mountOrTimeout(singleRegionAsiaEastBucket, testSuite.dir, string(tc.clientProtocol), testSuite.timeouts.singleRegionAsiaEastTimeout)
		assert.NoError(testSuite.T(), err)
	}
}
