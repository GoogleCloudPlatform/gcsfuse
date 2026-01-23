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

package kernel_reader

import (
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Test Suite
////////////////////////////////////////////////////////////////////////

type kernelReaderTest struct {
	suite.Suite
	flags []string
}

func TestKernelReaderSuite(t *testing.T) {
	suite.Run(t, new(kernelReaderTest))
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (s *kernelReaderTest) TearDownSubTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseAndDeleteLogFile(testEnv.rootDir)
}

func (s *kernelReaderTest) SetupSubTest() {
	// The flags use /gcsfuse-tmp/TestName.log, which is mapped to setup.TestDir()/gcsfuse-tmp/TestName.log
	logDir := path.Join(setup.TestDir(), "gcsfuse-tmp")
	require.NoError(s.T(), os.MkdirAll(logDir, 0755), "Failed to create log directory")
	logFileName := strings.ReplaceAll(s.T().Name(), "/", "_")
	setup.SetLogFile(path.Join(logDir, logFileName+".log"))
	testEnv.cfg.LogFile = setup.LogFile()
	mountGCSFuseAndSetupTestDir(s.flags, testEnv.ctx, testEnv.storageClient)
}

func (s *kernelReaderTest) validateParallelReads(logContent string) {
	lines := strings.Split(logContent, "\n")
	currentParallelism := 0
	maxParallelism := 0
	for _, line := range lines {
		if strings.Contains(line, "<- ReadFile") {
			currentParallelism++
		}
		if strings.Contains(line, "-> ReadFile") {
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

func (s *kernelReaderTest) readFileAndValidateLogs(expectedLog string, unexpectedLog string) string {
	testName := strings.ReplaceAll(s.T().Name(), "/", "_")
	if len(testName) > 200 {
		testName = testName[:200] + "_" + setup.GenerateRandomString(5)
	}
	fileName := testEnv.testDirPath + "/" + testName + "_test_file.txt"
	// Use operations.CreateFileOfSize which uses O_DIRECT to avoid polluting page cache during write.
	// 10MB is large enough to trigger chunked downloads/buffering.
	operations.CreateFileOfSize(10*1024*1024, fileName, s.T())
	require.NoError(s.T(), os.Truncate(setup.LogFile(), 0), "Failed to truncate log file")

	// Read the file using os.ReadFile which uses page cache to trigger kernel readahead.
	_, err := os.ReadFile(fileName)
	require.NoError(s.T(), err, "Failed to read file")

	logContent, err := os.ReadFile(setup.LogFile())
	require.NoError(s.T(), err, "Failed to read log file")
	logString := string(logContent)
	if expectedLog != "" {
		assert.Contains(s.T(), logString, expectedLog, "Expected log '%s' not found in logs", expectedLog)
	}
	if unexpectedLog != "" {
		assert.NotContains(s.T(), logString, unexpectedLog, "Unexpected log '%s' found in logs", unexpectedLog)
	}
	return logString
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *kernelReaderTest) TestKernelReader() {
	testCases := []struct {
		configName          string
		expectedLog         string
		unexpectedLog       string
		validateParallelism bool
	}{
		{
			configName:          "TestKernelReader_DefaultAndPrecedence",
			expectedLog:         "Initializing MRD Pool with size:",
			validateParallelism: true,
		},
		{
			configName:          "TestFileCache_KernelReaderDisabled",
			expectedLog:         "FileCache(",
			unexpectedLog:       "Initializing MRD Pool with size:",
			validateParallelism: false,
		},
		{
			configName:          "TestBufferedReader_KernelReaderDisabled",
			expectedLog:         "Scheduling block:",
			unexpectedLog:       "Initializing MRD Pool with size:",
			validateParallelism: false,
		},
	}

	for _, tc := range testCases {
		flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, tc.configName)
		for _, flags := range flagsSet {
			s.flags = flags
			s.Run(tc.configName, func() {
				log.Printf("Running tests with flags: %s", flags)
				logContent := s.readFileAndValidateLogs(tc.expectedLog, tc.unexpectedLog)
				if tc.validateParallelism {
					s.validateParallelReads(logContent)
				}
			})
		}
	}
}
