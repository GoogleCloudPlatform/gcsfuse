// Copyright 2025 Google LLC
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

package buffered_read

import (
	"fmt"
	"log"
	"os"
	"path"
	"syscall"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Test Suite Boilerplate
////////////////////////////////////////////////////////////////////////

type SequentialReadSuite struct {
	suite.Suite
	flags []string
}

func (s *SequentialReadSuite) SetupSuite() {
	if setup.MountedDirectory() != "" {
		setupForMountedDirectoryTests()
		return
	}
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, s.flags, mountFunc)
}

func (s *SequentialReadSuite) TearDownSuite() {
	if setup.MountedDirectory() != "" {
		setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
		return
	}
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

// //////////////////////////////////////////////////////////////////////
// Test Cases
// //////////////////////////////////////////////////////////////////////
func (s *SequentialReadSuite) TestSequentialRead() {
	fileSizeTests := []struct {
		name     string
		fileSize int64
	}{
		{
			name:     "SmallFile",
			fileSize: blockSizeInBytes / 2,
		},
		{
			name:     "LargeFile",
			fileSize: blockSizeInBytes * 2,
		},
	}

	chunkSizesToRead := []int64{128 * util.KiB, 512 * util.KiB, 1 * util.MiB}

	for _, fsTest := range fileSizeTests {
		for _, chunkSize := range chunkSizesToRead {
			testName := fmt.Sprintf("%s_%dKiB_Chunk", fsTest.name, chunkSize/util.KiB)
			fsTest := fsTest       // Capture range variable.
			chunkSize := chunkSize // Capture range variable.

			s.T().Run(testName, func(t *testing.T) {
				err := os.Truncate(setup.LogFile(), 0)
				require.NoError(t, err, "Failed to truncate log file")
				testDir := setup.SetupTestDirectory(testDirName)
				fileName := setupFileInTestDir(testEnv.ctx, testEnv.storageClient, testDir, fsTest.fileSize, t)

				expected := readFileAndValidate(testEnv.ctx, testEnv.storageClient, testDir, fileName, true, 0, chunkSize, t)

				bufferedReadLogEntry := parseAndValidateSingleBufferedReadLog(t)
				validate(expected, bufferedReadLogEntry, false, t)
				assert.Equal(t, int64(0), bufferedReadLogEntry.RandomSeekCount, "RandomSeekCount should be 0 for sequential reads.")
			})
		}
	}
}

// TestReadHeaderFooterAndBody verifies that a single file handle can correctly handle
// a mix of random reads (header and footer) followed by a large sequential read.
// The key validation is that all these operations should be served from a single
// buffered read log entry, indicating efficient handling.
func (s *SequentialReadSuite) TestReadHeaderFooterAndBody() {
	// Header and footer sizes (10KB each)
	headerSize := 10 * util.KiB
	footerSize := 10 * util.KiB
	s.T().Run("Read header footer then body from one file handle", func(t *testing.T) {
		err := os.Truncate(setup.LogFile(), 0)
		require.NoError(t, err, "Failed to truncate log file")
		testDir := setup.SetupTestDirectory(testDirName)
		fileSize := blockSizeInBytes * 2
		// Create a file of a given size in the test directory.
		fileName := setupFileInTestDir(testEnv.ctx, testEnv.storageClient, testDir, fileSize, t)
		filePath := path.Join(testDir, fileName)
		// Get the actual file size.
		fi, err := os.Stat(filePath)
		require.NoError(t, err)
		actualFileSize := fi.Size()
		// The size of the main content to read sequentially
		bodySize := actualFileSize - int64(headerSize) - int64(footerSize)
		f, err := os.OpenFile(filePath, os.O_RDONLY|syscall.O_DIRECT, 0)
		require.NoError(t, err)
		expected := &Expected{
			StartTimeStampSeconds: time.Now().Unix(),
			BucketName:            setup.TestBucket(),
			ObjectName:            path.Join(path.Base(testDir), fileName),
		}
		if setup.DynamicBucketMounted() != "" {
			expected.BucketName = setup.DynamicBucketMounted()
		}

		// (a) Read first 10KB (header)
		readAndValidateChunk(f, path.Base(testDir), fileName, 0, int64(headerSize), t)
		// (b) Read last 10KB (footer)
		readAndValidateChunk(f, path.Base(testDir), fileName, actualFileSize-int64(footerSize), int64(footerSize), t)
		// (c) Read the remaining content sequentially
		readAndValidateChunk(f, path.Base(testDir), fileName, int64(headerSize), bodySize, t)

		// Close the file handle to trigger log generation.
		operations.CloseFileShouldNotThrowError(t, f)
		expected.EndTimeStampSeconds = time.Now().Unix()
		// Since all reads were on the same handle, there should be one log entry.
		bufferedReadLogEntry := parseAndValidateSingleBufferedReadLog(t)
		validate(expected, bufferedReadLogEntry, false, t)
	})
}

// TestReadSpanningTwoBlocks verifies that a read spanning two buffer blocks is
// handled correctly.
func (s *SequentialReadSuite) TestReadSpanningTwoBlocks() {
	// Ensure file is large enough for multi-block reads.
	fileSize := 3 * blockSizeInBytes
	// We want to read 512KB, with 256KB in the first block and 256KB in the second.
	readSize := int64(512 * util.KiB)
	// Start the read 256KB before the end of the first block.
	readOffset := blockSizeInBytes - (256 * util.KiB)
	testDir := setup.SetupTestDirectory(testDirName)
	// Truncate the log file before the read operation.
	err := os.Truncate(setup.LogFile(), 0)
	require.NoError(s.T(), err, "Failed to truncate log file")
	fileName := setupFileInTestDir(testEnv.ctx, testEnv.storageClient, testDir, fileSize, s.T())

	// readFileAndValidate opens, reads, and closes the file in one go.
	expected := readFileAndValidate(testEnv.ctx, testEnv.storageClient, testDir, fileName, false, readOffset, readSize, s.T())

	bufferedReadLogEntry := parseAndValidateSingleBufferedReadLog(s.T())
	validate(expected, bufferedReadLogEntry, false, s.T())
	assert.Equal(s.T(), int64(0), bufferedReadLogEntry.RandomSeekCount)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestSequentialReadSuite(t *testing.T) {
	ts := &SequentialReadSuite{}

	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, ts)
		return
	}

	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, ts.flags = range flagsSet {
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
