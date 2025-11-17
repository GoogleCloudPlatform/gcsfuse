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
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/internal/util"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Test Suite Boilerplate
////////////////////////////////////////////////////////////////////////

// fallbackSuiteBase provides shared setup and teardown logic for fallback-related test suites.
type fallbackSuiteBase struct {
	suite.Suite
	testFlags *gcsfuseTestFlags
}

func (s *fallbackSuiteBase) SetupSuite() {
	if setup.MountedDirectory() != "" {
		setupForMountedDirectoryTests()
		return
	}
	configFile := createConfigFile(s.testFlags)
	flags := []string{"--config-file=" + configFile}
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
}

func (s *fallbackSuiteBase) SetupTest() {
	err := os.Truncate(setup.LogFile(), 0)
	require.NoError(s.T(), err, "Failed to truncate log file")
}

func (s *fallbackSuiteBase) TearDownSuite() {
	if setup.MountedDirectory() != "" {
		setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
		return
	}
	setup.UnmountGCSFuse(rootDir)
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

// InsufficientPoolCreationSuite tests scenarios where the buffered reader is not
// created due to an insufficient global block pool at the time of file opening.
type InsufficientPoolCreationSuite struct {
	fallbackSuiteBase
}

// RandomReadFallbackSuite tests fallback scenarios related to random reads.
type RandomReadFallbackSuite struct {
	fallbackSuiteBase
}

////////////////////////////////////////////////////////////////////////
// Test Cases
////////////////////////////////////////////////////////////////////////

// TestNewBufferedReader_InsufficientGlobalPool_NoReaderAdded tests that when
// there are not enough blocks in the global pool to satisfy `min-blocks-per-handle`,
// the BufferedReader is not created, and reads fall back to the next reader
// without any buffered reading.
func (s *InsufficientPoolCreationSuite) TestNewBufferedReader_InsufficientGlobalPool_NoReaderAdded() {
	fileSize := 3 * s.testFlags.blockSizeMB * util.MiB
	chunkSize := int64(1 * util.MiB)
	testDir := setup.SetupTestDirectory(testDirName)
	fileName := setupFileInTestDir(ctx, storageClient, testDir, fileSize, s.T())
	filePath := path.Join(testDir, fileName)

	// Open and read the file. Since BufferedReader creation should fail, the read
	// will be served by the GCSReader.
	content, err := operations.ReadChunkFromFile(filePath, chunkSize, 0, os.O_RDONLY|syscall.O_DIRECT)

	require.NoError(s.T(), err, "Failed to read file")
	client.ValidateObjectChunkFromGCS(ctx, storageClient, path.Base(testDir), fileName, 0, chunkSize, string(content), s.T())
	warningMsg := "Failed to create bufferedReader"
	found := operations.CheckLogFileForMessage(s.T(), warningMsg, setup.LogFile())
	assert.True(s.T(), found, "Expected warning message not found in log file")
	logEntries := parseBufferedReadLogs(s.T())
	assert.Empty(s.T(), logEntries, "Expected no buffered read log entries")
}

func (s *RandomReadFallbackSuite) TestRandomRead_LargeFile_Fallback() {
	const randomReadsThreshold = 3
	blockSizeInBytes := s.testFlags.blockSizeMB * util.MiB
	// The distant block to read is just outside the initial prefetch window.
	// Initial prefetch window size = (1 + initial_prefetch_blocks) blocks.
	// So, we read the block at index (initial_prefetch_blocks + 1).
	distantBlockIndex := s.testFlags.startBlocksPerHandle + 1
	fileSize := blockSizeInBytes * (distantBlockIndex + 1)
	chunkSize := int64(1 * util.KiB)
	testDir := setup.SetupTestDirectory(testDirName)
	fileName := setupFileInTestDir(ctx, storageClient, testDir, fileSize, s.T())
	filePath := path.Join(testDir, fileName)
	f, err := os.OpenFile(filePath, os.O_RDONLY|syscall.O_DIRECT, 0)
	require.NoError(s.T(), err)
	defer operations.CloseFileShouldNotThrowError(s.T(), f)
	distantOffset := distantBlockIndex * blockSizeInBytes

	induceRandomReadFallback(s.T(), f, path.Base(testDir), fileName, chunkSize, distantOffset, randomReadsThreshold)

	bufferedReadLogEntry := parseAndValidateSingleBufferedReadLog(s.T())
	expected := &Expected{BucketName: setup.TestBucket(), ObjectName: path.Join(path.Base(testDir), fileName)}
	validate(expected, bufferedReadLogEntry, true, s.T())
	assert.Equal(s.T(), int64(randomReadsThreshold+1), bufferedReadLogEntry.RandomSeekCount, "RandomSeekCount should be one greater than the threshold.")
}

func (s *RandomReadFallbackSuite) TestRandomRead_SmallFile_NoFallback() {
	blockSizeInBytes := s.testFlags.blockSizeMB * util.MiB
	// File size is small, less than one block.
	fileSize := blockSizeInBytes / 2
	chunkSize := int64(1 * util.KiB)
	testDir := setup.SetupTestDirectory(testDirName)
	fileName := setupFileInTestDir(ctx, storageClient, testDir, fileSize, s.T())
	filePath := path.Join(testDir, fileName)
	f, err := os.OpenFile(filePath, os.O_RDONLY|syscall.O_DIRECT, 0)
	require.NoError(s.T(), err)
	defer operations.CloseFileShouldNotThrowError(s.T(), f)
	// The first read (at offset 0) is sequential.
	readAndValidateChunk(f, path.Base(testDir), fileName, 0, chunkSize, s.T())

	// The second read should be served from the prefetched block and not be a random seek.
	readAndValidateChunk(f, path.Base(testDir), fileName, fileSize/2, chunkSize, s.T())

	bufferedReadLogEntry := parseAndValidateSingleBufferedReadLog(s.T())
	expected := &Expected{BucketName: setup.TestBucket(), ObjectName: path.Join(path.Base(testDir), fileName)}
	validate(expected, bufferedReadLogEntry, false, s.T())
	assert.Equal(s.T(), int64(0), bufferedReadLogEntry.RandomSeekCount, "RandomSeekCount should be 0 for small file reads.")
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestFallbackSuites(t *testing.T) {
	// Define base flags for insufficient pool creation tests.
	baseInsufficientPoolFlags := gcsfuseTestFlags{
		enableBufferedRead:   true,
		blockSizeMB:          8,
		minBlocksPerHandle:   2,
		globalMaxBlocks:      1, // Less than min-blocks-per-handle
		maxBlocksPerHandle:   10,
		startBlocksPerHandle: 2,
	}

	// Define base flags for random read fallback tests.
	baseRandomReadFlags := gcsfuseTestFlags{
		enableBufferedRead:   true,
		blockSizeMB:          8,
		maxBlocksPerHandle:   20,
		startBlocksPerHandle: 2,
		minBlocksPerHandle:   2,
	}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, &InsufficientPoolCreationSuite{fallbackSuiteBase{testFlags: &baseInsufficientPoolFlags}})
		suite.Run(t, &RandomReadFallbackSuite{fallbackSuiteBase{testFlags: &baseRandomReadFlags}})
		return
	}

	protocols := []string{clientProtocolHTTP1, clientProtocolGRPC}

	for _, protocol := range protocols {
		t.Run(protocol, func(t *testing.T) {
			// Run the suite for insufficient pool at creation time.
			insufficientPoolFlags := baseInsufficientPoolFlags
			insufficientPoolFlags.clientProtocol = protocol
			suite.Run(t, &InsufficientPoolCreationSuite{fallbackSuiteBase{testFlags: &insufficientPoolFlags}})

			// Run the suite for random read fallback scenarios.
			randomReadFlags := baseRandomReadFlags
			randomReadFlags.clientProtocol = protocol
			suite.Run(t, &RandomReadFallbackSuite{fallbackSuiteBase{testFlags: &randomReadFlags}})
		})
	}
}
