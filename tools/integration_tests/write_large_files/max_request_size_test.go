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

package write_large_files

import (
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	writeFileStartMsg = "<- WriteFile"
	sixteenMiB        = 16 * operations.MiB
	oneMiBInBytes     = "1048576"
)

func isKernelMaxPagesSupported(requiredBytes int) bool {
	requiredPages := requiredBytes / os.Getpagesize()
	maxPagesData, err := os.ReadFile("/proc/sys/fs/fuse/max_pages_limit")
	if err != nil {
		return false
	}
	limit, err := strconv.Atoi(strings.TrimSpace(string(maxPagesData)))
	return err == nil && limit >= requiredPages
}

func runWriteMaxRequestSize16MiBTest(t *testing.T) {
	// Setup test directory and target file.
	writeDir := setup.SetupTestDirectory("dirForMaxRequestSizeWrite")
	filePath := path.Join(writeDir, "16MiB_write_test_"+setup.GenerateRandomString(5)+".txt")

	// Truncate the log file right before writing so only our write operations are logged.
	err := os.Truncate(setup.LogFile(), 0)
	require.NoError(t, err, "Failed to truncate log file")

	// Generate 16 MiB of random data (memory-aligned to os.Getpagesize() for O_DIRECT write).
	data, err := operations.GenerateRandomData(sixteenMiB)
	require.NoError(t, err, "Failed to generate random data")

	// Open file with O_DIRECT so writes go straight to FUSE without kernel page-cache buffering.
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|syscall.O_DIRECT, setup.FilePermission_0600)
	require.NoError(t, err, "Failed to open file for direct write")

	// Write 16 MiB in a single buffer call.
	n, err := f.Write(data)
	require.NoError(t, err, "Failed to write 16 MiB to file")
	require.Equal(t, sixteenMiB, n)

	err = f.Close()
	require.NoError(t, err, "Failed to close file")

	// Read log content after write.
	logContent, err := os.ReadFile(setup.LogFile())
	require.NoError(t, err, "Failed to read log file after write")

	lines := strings.Split(string(logContent), "\n")
	var writeRequestCount int
	for _, line := range lines {
		if strings.Contains(line, writeFileStartMsg) {
			writeRequestCount++
			assert.Contains(t, line, oneMiBInBytes, "Expected write request to be 1 MiB (%s bytes), but got line: %s", oneMiBInBytes, line)
		}
	}
	assert.Equal(t, 16, writeRequestCount, "Expected exactly 16 write requests of 1 MiB each for 16 MiB write, got %d", writeRequestCount)
}

func TestWriteMaxRequestSize16MiB(t *testing.T) {
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		runWriteMaxRequestSize16MiBTest(t)
		return
	}

	if !isKernelMaxPagesSupported(sixteenMiB) {
		t.Skip("Skipping TestWriteMaxRequestSize16MiB: kernel /proc/sys/fs/fuse/max_pages_limit does not support 16 MiB request size")
	}

	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			mustMountGCSFuseAndSetupTestDir(flags, testEnv.ctx, testEnv.storageClient)
			t.Cleanup(func() {
				setup.SaveGCSFuseLogFileInCaseOfFailure(t)
				setup.UnmountGCSFuse(testEnv.rootDir)
			})

			runWriteMaxRequestSize16MiBTest(t)
		})
	}
}
