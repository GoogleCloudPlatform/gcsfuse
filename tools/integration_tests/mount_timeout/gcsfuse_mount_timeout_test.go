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
	"bufio"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	iterations int = 10
)

func TestMountTimeout(t *testing.T) {
	if setup.IsZonalBucketRun() {
		zone := os.Getenv("TEST_ENV")
		switch zone {
		case testEnvZoneGCEUSCentral1A:
			// Set strict zone-based config values.
			config := ZBMountTimeoutTestCaseConfig{
				sameZoneZonalBucket:   zonalUSCentral1ABucket,
				crossZoneZonalBucket:  zonalUSWest4ABucket,
				sameZoneMountTimeout:  zonalSameZoneExpectedMountTime,
				crossZoneMountTimeout: zonalCrossZoneExpectedMountTime,
			}
			t.Log("Running tests with region based timeout values since the GCE VM is located in us-central...\n")
			suite.Run(t, &ZBMountTimeoutTest{config: config})
		case testEnvZoneGCEUSWEST4A:
			// Set strict zone-based config values.
			config := ZBMountTimeoutTestCaseConfig{
				sameZoneZonalBucket:   zonalUSWest4ABucket,
				crossZoneZonalBucket:  zonalUSCentral1ABucket,
				sameZoneMountTimeout:  relaxedExpectedMountTime,
				crossZoneMountTimeout: relaxedExpectedMountTime,
			}
			t.Logf("Running tests with relaxed timeout of %f sec for all scenarios since the GCE VM is not located in us-central...\n", relaxedExpectedMountTime.Seconds())
			suite.Run(t, &ZBMountTimeoutTest{config: config})
		default:
			// Skip the tests if the testing environment is not GCE VM.
			t.Logf("Skipping tests since the testing environment (%q) is not a ZB supported region...\n", zone)
			t.Skip()
		}
	} else {
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
			suite.Run(t, &NonZBMountTimeoutTest{timeouts: timeout})
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
			suite.Run(t, &NonZBMountTimeoutTest{timeouts: timeout})
		} else {
			// Skip the tests if the testing environment is not GCE VM.
			t.Log("Skipping tests since the testing environment is not GCE VM...\n")
			t.Skip()
		}
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

type ZBMountTimeoutTestCaseConfig struct {
	sameZoneZonalBucket   string
	crossZoneZonalBucket  string
	sameZoneMountTimeout  time.Duration
	crossZoneMountTimeout time.Duration
}

type MountTimeoutTest struct {
	suite.Suite
	// Path to the gcsfuse binary.
	gcsfusePath string

	// A temporary directory into which a file system may be mounted. Removed in
	// TearDown.
	dir string
}

type NonZBMountTimeoutTest struct {
	MountTimeoutTest
	timeouts RegionWiseTimeouts
}

type ZBMountTimeoutTest struct {
	MountTimeoutTest
	config ZBMountTimeoutTestCaseConfig
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
func (testSuite *MountTimeoutTest) mountOrTimeout(bucketName, clientProtocol string, expectedMountTime time.Duration) (err error) {
	minMountTime := time.Duration(math.MaxInt64)
	logFile := setup.LogFile()
	defer func() {
		if err != nil {
			setup.SaveLogFileAsArtifact(logFile, setup.GCSFuseLogFilePrefix+filepath.Base(logFile))
		}
	}()

	// Iterating 10 times to account for randomness in time taken to mount.
	args := []string{"--client-protocol", clientProtocol, "--log-severity=trace", "--log-file=" + logFile, bucketName, testSuite.dir}
	for i := range iterations {
		start := time.Now()
		if err = mounting.MountGcsfuse(testSuite.gcsfusePath, args); err != nil {
			err = fmt.Errorf("mount failed for bucket %q (client protocol: %s) on attempt#%v: %w", bucketName, clientProtocol, i, err)
			return err
		}
		mountTime := time.Since(start)

		minMountTime = time.Duration(math.Min(float64(minMountTime), float64(mountTime)))

		if err = unmountAndWait(testSuite.dir); err != nil {
			err = fmt.Errorf("unmountAndWait failed for bucket %q on attempt#%v: %w", bucketName, i, err)
			return err
		}
	}

	if minMountTime > expectedMountTime {
		err = fmt.Errorf("mount took too long for bucket %q (client protocol: %s). expected: %v, actual minimum mount time: %v", bucketName, clientProtocol, expectedMountTime, minMountTime)
		return err
	}
	return nil
}

// unmountAndWait unmounts the given directory and waits for the associated
// resources to be released by polling, with a timeout.
func unmountAndWait(mountDir string) error {
	// isMounted checks if a directory is currently a mount point by reading /proc/mounts.
	isMounted := func(mountPoint string) (bool, error) {
		file, err := os.Open("/proc/mounts")
		if err != nil {
			// On non-Linux systems, /proc/mounts doesn't exist, so we can't poll.
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, fmt.Errorf("could not open /proc/mounts: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			// The second field in /proc/mounts is the mount point.
			if len(fields) >= 2 && fields[1] == mountPoint {
				return true, nil // Found the mount point
			}
		}
		return false, scanner.Err()
	}

	if err := util.Unmount(mountDir); err != nil {
		// It might already be unmounted or in a weird state. If the error
		// indicates it's not mounted, we can proceed to verify.
		if !strings.Contains(err.Error(), "not mounted") && !strings.Contains(err.Error(), "no such file or directory") {
			return fmt.Errorf("unmount call failed: %w", err)
		}
	}

	// Poll for up to 5 seconds for the unmount to complete.
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(5 * time.Second)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timed out waiting for %q to unmount", mountDir)
		case <-ticker.C:
			mounted, err := isMounted(mountDir)
			if err != nil || !mounted {
				return err // Success or error checking mount status.
			}
		}
	}
}

func (testSuite *NonZBMountTimeoutTest) TestMountMultiRegionUSBucketWithTimeout() {
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

		err := testSuite.mountOrTimeout(multiRegionUSBucket, string(tc.clientProtocol), testSuite.timeouts.multiRegionUSTimeout)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *NonZBMountTimeoutTest) TestMountMultiRegionAsiaBucketWithTimeout() {
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

		err := testSuite.mountOrTimeout(multiRegionAsiaBucket, string(tc.clientProtocol), testSuite.timeouts.multiRegionAsiaTimeout)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *NonZBMountTimeoutTest) TestMountDualRegionUSBucketWithTimeout() {
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

		err := testSuite.mountOrTimeout(dualRegionUSBucket, string(tc.clientProtocol), testSuite.timeouts.dualRegionUSTimeout)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *NonZBMountTimeoutTest) TestMountDualRegionAsiaBucketWithTimeout() {
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

		err := testSuite.mountOrTimeout(dualRegionAsiaBucket, string(tc.clientProtocol), testSuite.timeouts.dualRegionAsiaTimeout)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *NonZBMountTimeoutTest) TestMountSingleRegionUSBucketWithTimeout() {
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

		err := testSuite.mountOrTimeout(singleRegionUSCentralBucket, string(tc.clientProtocol), testSuite.timeouts.singleRegionUSCentralTimeout)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *NonZBMountTimeoutTest) TestMountSingleRegionAsiaBucketWithTimeout() {
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

		err := testSuite.mountOrTimeout(singleRegionAsiaEastBucket, string(tc.clientProtocol), testSuite.timeouts.singleRegionAsiaEastTimeout)
		assert.NoError(testSuite.T(), err)
	}
}

func (testSuite *ZBMountTimeoutTest) TestMountSameZoneZonalBucketWithTimeout() {
	setup.SetLogFile(fmt.Sprintf("%s%s.txt", logfilePathPrefix, "SameZoneZonalBucket"))

	err := testSuite.mountOrTimeout(testSuite.config.sameZoneZonalBucket, string(cfg.GRPC), testSuite.config.sameZoneMountTimeout)
	assert.NoError(testSuite.T(), err)
}

func (testSuite *ZBMountTimeoutTest) TestMountCrossZoneZonalBucketWithTimeout() {
	setup.SetLogFile(fmt.Sprintf("%s%s.txt", logfilePathPrefix, "CrossZoneZonalBucket"))

	err := testSuite.mountOrTimeout(testSuite.config.crossZoneZonalBucket, string(cfg.GRPC), testSuite.config.crossZoneMountTimeout)
	assert.NoError(testSuite.T(), err)
}
