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
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	sixteenMiB        = 16 * operations.MiB
	sixteenMiBInBytes = "16777216"
)

type MaxRequestSizeReadSuite struct {
	suite.Suite
	flags []string
}

func (s *MaxRequestSizeReadSuite) SetupSuite() {
	mustMountGCSFuseAndSetupTestDir(s.flags, testEnv.ctx, testEnv.storageClient)
}

func (s *MaxRequestSizeReadSuite) TearDownSuite() {
	tearDownOptimizationTest(s.T())
}

func (s *MaxRequestSizeReadSuite) TestKernelReaderLargeRead16MiB() {
	// 1. Check if the host kernel supports elevating max_pages limit to what is needed for a 16 MiB read.
	requiredPages := sixteenMiB / os.Getpagesize()
	data, err := os.ReadFile("/proc/sys/fs/fuse/max_pages_limit")
	if err != nil {
		s.T().Skipf("Skipping TestKernelReaderLargeRead16MiB: kernel does not support elevating max_pages limit (/proc/sys/fs/fuse/max_pages_limit: %v)", err)
	}
	if limit, err := strconv.Atoi(strings.TrimSpace(string(data))); err != nil || limit < requiredPages {
		s.T().Skipf("Skipping TestKernelReaderLargeRead16MiB: /proc/sys/fs/fuse/max_pages_limit (%s, err: %v) is less than required %d pages for 16 MiB read (page size %d bytes)", strings.TrimSpace(string(data)), err, requiredPages, os.Getpagesize())
	}

	// 2. Create 16 MiB test file in GCS.
	testName := strings.ReplaceAll(s.T().Name(), "/", "_")
	fileName := path.Join(testEnv.testDirPath, testName+"_16MiB.txt")
	operations.CreateFileOfSize(sixteenMiB, fileName, s.T())
	s.T().Cleanup(func() {
		_ = os.Remove(fileName)
	})

	// 3. Truncate log file to record only the read operation.
	err = os.Truncate(setup.LogFile(), 0)
	require.NoError(s.T(), err, "Failed to truncate log file")

	// 4. Perform a 16 MiB read.
	content, err := os.ReadFile(fileName)
	require.NoError(s.T(), err, "Failed to read 16 MiB file")
	require.Len(s.T(), content, sixteenMiB)

	// 5. Verify in gcsfuse trace log that exactly 1 read request of 16 MiB (16777216 bytes) occurred.
	logContent, err := os.ReadFile(setup.LogFile())
	require.NoError(s.T(), err, "Failed to read log file after read")

	lines := strings.Split(string(logContent), "\n")
	var readRequestCount int
	for _, line := range lines {
		if strings.Contains(line, readFileStartMsg) {
			readRequestCount++
			assert.Contains(s.T(), line, sixteenMiBInBytes, "Expected read request to be 16 MiB (%s bytes), but got line: %s", sixteenMiBInBytes, line)
		}
	}
	assert.Equal(s.T(), 1, readRequestCount, "Expected exactly 1 read request of 16 MiB when kernel reader and fuse max pages support are enabled, got %d", readRequestCount)
}

func TestKernelReaderLargeReadSuite(t *testing.T) {
	if setup.IsDynamicMount(testEnv.mountDir, testEnv.rootDir) {
		t.Skip("Skipping test for dynamic mounting")
	}
	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			log.Printf("Running tests with flags: %s", flags)
			s := &MaxRequestSizeReadSuite{
				flags: flags,
			}
			suite.Run(t, s)
		})
	}
}
