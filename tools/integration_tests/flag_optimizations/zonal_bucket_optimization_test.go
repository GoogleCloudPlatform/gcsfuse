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

package flag_optimizations

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/kernelparams"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sys/unix"
)

const (
	// kernelReaderInitMsg indicates the MRD pool initialization.
	// MRD pool is used only by kernel reader. This confirms that the kernel reader is enabled and initializing.
	kernelReaderInitMsg = "Initializing MRD Pool with size:"

	// fileCacheMsg indicates that the file cache is being used.
	fileCacheMsg = "FileCache("

	// bufferedReaderSchedMsg indicates the buffered reader is scheduling a block download.
	bufferedReaderSchedMsg = "Scheduling block:"

	// readFileStartMsg indicates the start of a ReadFile operation.
	readFileStartMsg = "<- ReadFile"

	// readFileEndMsg indicates the completion of a ReadFile operation.
	readFileEndMsg = "-> ReadFile"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (s *KernelReaderParamsSuite) verifyKernelParam(path string, expectedVal string, optimizedVal string) {
	s.T().Helper()
	content, err := os.ReadFile(path)
	require.NoError(s.T(), err)
	val := strings.TrimSpace(string(content))

	if expectedVal != "" {
		assert.Equal(s.T(), expectedVal, val, "Param %s mismatch", path)
	} else if setup.IsDynamicMount(testEnv.mountDir, testEnv.rootDir) {
		assert.NotEqual(s.T(), optimizedVal, val, "Param %s should NOT match optimized default for dynamic mount", path)
	} else {
		assert.Equal(s.T(), optimizedVal, val, "Param %s should NOT match optimized default", path)
	}
}

func (s *ReadStrategySuite) validateParallelReads(logContent string) {
	s.T().Helper()
	lines := strings.Split(logContent, "\n")
	currentParallelism := 0
	maxParallelism := 0
	for _, line := range lines {
		if strings.Contains(line, readFileStartMsg) {
			currentParallelism++
		}
		if strings.Contains(line, readFileEndMsg) {
			currentParallelism--
		}
		if currentParallelism > maxParallelism {
			maxParallelism = currentParallelism
		}
		if maxParallelism >= 2 {
			break
		}
	}
	assert.Greater(s.T(), maxParallelism, 1, "Expected parallel reads (max parallelism > 1)")
}

func createAndReadFile(t *testing.T, testName string) {
	t.Helper()
	testName = strings.ReplaceAll(testName, "/", "_")
	fileName := testEnv.testDirPath + "/" + testName + "_test_file.txt"
	// Use operations.CreateFileOfSize which uses O_DIRECT to avoid polluting page cache during write.
	operations.CreateFileOfSize(10*1024*1024, fileName, t)
	require.NoError(t, os.Truncate(setup.LogFile(), 0), "Failed to truncate log file")

	// Read the file using os.ReadFile which uses page cache to trigger kernel readahead.
	_, err := os.ReadFile(fileName)

	require.NoError(t, err, "Failed to read file")
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

// KernelParamsSuite tests the behavior of zonal bucket optimizations,
// specifically verifying kernel parameters.
type KernelReaderParamsSuite struct {
	suite.Suite
	flags                       []string
	expectedReadAhead           string
	expectedMaxBackground       string
	expectedCongestionThreshold string
}

func (s *KernelReaderParamsSuite) SetupSuite() {
	mustMountGCSFuseAndSetupTestDir(s.flags, testEnv.ctx, testEnv.storageClient)
}

func (s *KernelReaderParamsSuite) TearDownSuite() {
	tearDownOptimizationTest(s.T())
}

// TestKernelParamVerification verifies the values of max_read_ahead_kb,
// max_background, and congestion_threshold for Zonal Buckets.
// For non dynamic ZB mounts, they should be updated to the optimized values
// (unless explicitly changed via config or CLI).
func (s *KernelReaderParamsSuite) TestKernelParamVerification() {
	// Verify kernel parameters in /sys
	var stat unix.Stat_t
	err := unix.Stat(setup.MntDir(), &stat)
	require.NoError(s.T(), err)
	devMajor := unix.Major(stat.Dev)
	devMinor := unix.Minor(stat.Dev)
	readAheadPath, err := kernelparams.PathForParam(kernelparams.MaxReadAheadKb, devMajor, devMinor)
	require.NoError(s.T(), err)
	maxBackgroundPath, err := kernelparams.PathForParam(kernelparams.MaxBackgroundRequests, devMajor, devMinor)
	require.NoError(s.T(), err)
	congestionThresholdPath, err := kernelparams.PathForParam(kernelparams.CongestionWindowThreshold, devMajor, devMinor)
	require.NoError(s.T(), err)

	optimizedReadAhead := "16384"
	optimizedMaxBackground := fmt.Sprintf("%d", cfg.DefaultMaxBackground())
	optimizedCongestion := fmt.Sprintf("%d", cfg.DefaultCongestionThreshold())

	s.verifyKernelParam(readAheadPath, s.expectedReadAhead, optimizedReadAhead)
	s.verifyKernelParam(maxBackgroundPath, s.expectedMaxBackground, optimizedMaxBackground)
	s.verifyKernelParam(congestionThresholdPath, s.expectedCongestionThreshold, optimizedCongestion)
}

func TestZonalBucketOptimizations(t *testing.T) {
	if setup.IsDynamicMount(testEnv.mountDir, testEnv.rootDir) {
		t.Skip("Skipping test for dynamic mounting")
	}
	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run("", func(t *testing.T) {
			log.Printf("Running tests with flags: %s", flags)
			s := &KernelReaderParamsSuite{
				flags: flags,
			}
			suite.Run(t, s)
		})
	}
}

func TestZonalBucketOptimizations_ExplicitOverrides(t *testing.T) {
	if setup.IsDynamicMount(testEnv.mountDir, testEnv.rootDir) {
		t.Skip("Skipping test for dynamic mounting")
	}
	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run("", func(t *testing.T) {
			log.Printf("Running tests with flags: %s", flags)
			s := &KernelReaderParamsSuite{
				flags:                       flags,
				expectedReadAhead:           "2048",
				expectedMaxBackground:       "50",
				expectedCongestionThreshold: "30",
			}
			suite.Run(t, s)
		})
	}
}

func TestZonalBucketOptimizations_Dynamic(t *testing.T) {
	if !setup.IsDynamicMount(testEnv.mountDir, testEnv.rootDir) {
		t.Skip("Skipping test for non dynamic mounting")
	}
	flags := []string{"--log-severity=trace"}
	log.Printf("Running tests with flags: %s", flags)
	s := &KernelReaderParamsSuite{
		flags: flags,
	}
	suite.Run(t, s)
}

// ReadStrategySuite tests the behavior of the kernel reader under different configurations,
// verifying log output and read parallelism.
type ReadStrategySuite struct {
	suite.Suite
	flags               []string
	expectedLog         string
	unexpectedLog       string
	validateParallelism bool
}

func (s *ReadStrategySuite) SetupSuite() {
	err := mountGCSFuseAndSetupTestDir(s.flags, testEnv.ctx, testEnv.storageClient)
	require.NoError(s.T(), err)
}

func (s *ReadStrategySuite) TearDownSuite() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseAndDeleteLogFile(testEnv.rootDir)
}

// TestKernelReaderBehavior verifies the read strategy behavior based on flags.
// Specifically for Zonal Buckets, it checks that if not explicitly disabled,
// Kernel Reader is used (taking precedence) even if Buffered Read or File Cache
// are enabled.
func (s *ReadStrategySuite) TestKernelReaderBehavior() {
	createAndReadFile(s.T(), s.T().Name())

	logContent, err := os.ReadFile(setup.LogFile())

	require.NoError(s.T(), err, "Failed to read log file")
	if s.expectedLog != "" {
		assert.Contains(s.T(), string(logContent), s.expectedLog, "Expected log '%s' not found in logs", s.expectedLog)
	}
	if s.unexpectedLog != "" {
		assert.NotContains(s.T(), string(logContent), s.unexpectedLog, "Unexpected log '%s' found in logs", s.unexpectedLog)
	}
	if s.validateParallelism {
		s.validateParallelReads(string(logContent))
	}
}

func TestKernelReader(t *testing.T) {
	if setup.IsDynamicMount(testEnv.mountDir, testEnv.rootDir) {
		t.Skip("Skipping test for dynamic mounting")
	}
	testCases := []struct {
		configName          string
		expectedLog         string
		unexpectedLog       string
		validateParallelism bool
	}{
		{
			configName:          "TestKernelReader_DefaultAndPrecedence",
			expectedLog:         kernelReaderInitMsg,
			validateParallelism: true,
		},
		{
			configName:          "TestFileCache_KernelReaderDisabled",
			expectedLog:         fileCacheMsg,
			unexpectedLog:       kernelReaderInitMsg,
			validateParallelism: false,
		},
		{
			configName:          "TestBufferedReader_KernelReaderDisabled",
			expectedLog:         bufferedReaderSchedMsg,
			unexpectedLog:       kernelReaderInitMsg,
			validateParallelism: false,
		},
	}

	for _, tc := range testCases {
		flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, tc.configName)
		for _, flags := range flagsSet {
			t.Run(tc.configName, func(t *testing.T) {
				log.Printf("Running tests with flags: %s", flags)
				s := &ReadStrategySuite{
					flags:               flags,
					expectedLog:         tc.expectedLog,
					unexpectedLog:       tc.unexpectedLog,
					validateParallelism: tc.validateParallelism,
				}
				suite.Run(t, s)
			})
		}
	}
}

func TestKernelReader_Dynamic(t *testing.T) {
	if !setup.IsDynamicMount(testEnv.mountDir, testEnv.rootDir) {
		t.Skip("Skipping test for non dynamic mounting")
	}
	configName := "TestKernelReader_Dynamic"
	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, configName)
	for _, flags := range flagsSet {
		t.Run(configName, func(t *testing.T) {
			log.Printf("Running tests with flags: %s", flags)
			s := &ReadStrategySuite{
				flags:               flags,
				unexpectedLog:       kernelReaderInitMsg,
				validateParallelism: false,
			}
			suite.Run(t, s)
		})
	}
}
