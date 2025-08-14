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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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
	// Create config file.
	configFile := createConfigFile(s.testFlags)
	// Create the final flags slice.
	flags := []string{"--config-file=" + configFile}
	// Mount GCSFuse.
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
}

func (s *fallbackSuiteBase) TearDownSuite() {
	setup.UnmountGCSFuse(setup.MntDir())
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
	fileSize := 2 * s.testFlags.blockSizeMB * util.MiB
	chunkSize := int64(1 * util.KiB)
	err := os.Truncate(setup.LogFile(), 0)
	require.NoError(s.T(), err, "Failed to truncate log file")
	testDir := setup.SetupTestDirectory(testDirName)
	fileName := setupFileInTestDir(ctx, storageClient, testDir, fileSize, s.T())

	// Open and read the file. Since BufferedReader creation should fail, the read
	// will be served by the GCSReader.
	filePath := path.Join(testDir, fileName)
	content, err := operations.ReadChunkFromFile(filePath, chunkSize, 0, os.O_RDONLY|syscall.O_DIRECT)
	require.NoError(s.T(), err, "Failed to read file")
	client.ValidateObjectChunkFromGCS(ctx, storageClient, path.Base(testDir), fileName, 0, chunkSize, string(content), s.T())

	// Validate logs.
	// 1. Check for the warning that BufferedReader creation failed.
	warningMsg := "creating block-pool: reserving blocks: can't allocate any block from the pool"
	found := operations.CheckLogFileForMessage(s.T(), warningMsg, setup.LogFile())
	assert.True(s.T(), found, "Expected warning message not found in log file")

	// 2. Check that no buffered read logs were generated.
	logEntries := parseBufferedReadLogs(s.T())
	assert.Empty(s.T(), logEntries, "Expected no buffered read log entries")
}

func (s *RandomReadFallbackSuite) TestRandomRead_LargeFile_Fallback() {
	const randomReadsThreshold = 3
	blockSizeInBytes := s.testFlags.blockSizeMB * util.MiB
	// File size is large enough to exceed random seek threshold.
	fileSize := blockSizeInBytes * (randomReadsThreshold + 1) * 2
	chunkSize := int64(1 * util.KiB)
	err := os.Truncate(setup.LogFile(), 0)
	require.NoError(s.T(), err, "Failed to truncate log file")
	testDir := setup.SetupTestDirectory(testDirName)
	fileName := setupFileInTestDir(ctx, storageClient, testDir, fileSize, s.T())
	filePath := path.Join(testDir, fileName)

	// Open the file once to get a persistent file handle.
	f, err := os.OpenFile(filePath, os.O_RDONLY|syscall.O_DIRECT, 0)
	require.NoError(s.T(), err)
	defer operations.CloseFileShouldNotThrowError(s.T(), f)

	// Perform random reads to trigger fallback.
	offsets := []int64{5 * blockSizeInBytes, 10 * blockSizeInBytes, 1 * blockSizeInBytes, 15 * blockSizeInBytes}
	require.Greater(s.T(), len(offsets), randomReadsThreshold, "Number of reads must be greater than randomReadsThreshold")

	readBuffer := make([]byte, chunkSize)
	for _, offset := range offsets {
		_, err := f.ReadAt(readBuffer, offset)
		require.NoError(s.T(), err, "ReadAt failed at offset %d", offset)
		client.ValidateObjectChunkFromGCS(ctx, storageClient, path.Base(testDir), fileName, offset, chunkSize, string(readBuffer), s.T())
	}

	// Validate logs.
	bufferedReadLogEntry := parseAndValidateSingleBufferedReadLog(s.T())
	expected := &Expected{BucketName: setup.TestBucket(), ObjectName: path.Join(path.Base(testDir), fileName)}
	validate(expected, bufferedReadLogEntry, true, s.T())
	assert.Equal(s.T(), int64(randomReadsThreshold+1), bufferedReadLogEntry.RandomSeekCount, "RandomSeekCount should match number of random reads.")
}

func (s *RandomReadFallbackSuite) TestRandomRead_SmallFile_NoFallback() {
	blockSizeInBytes := s.testFlags.blockSizeMB * util.MiB
	// File size is small, less than one block.
	fileSize := blockSizeInBytes / 2
	chunkSize := int64(1 * util.KiB)
	err := os.Truncate(setup.LogFile(), 0)
	require.NoError(s.T(), err, "Failed to truncate log file")
	testDir := setup.SetupTestDirectory(testDirName)
	fileName := setupFileInTestDir(ctx, storageClient, testDir, fileSize, s.T())
	filePath := path.Join(testDir, fileName)

	f, err := os.OpenFile(filePath, os.O_RDONLY|syscall.O_DIRECT, 0)
	require.NoError(s.T(), err)
	defer operations.CloseFileShouldNotThrowError(s.T(), f)

	// Perform a couple of reads. The first read (at offset 0) is sequential.
	// The second read should be served from the prefetched block and not be a random seek.
	readAndValidateChunk(f, path.Base(testDir), fileName, 0, chunkSize, s.T())
	readAndValidateChunk(f, path.Base(testDir), fileName, blockSizeInBytes/4, chunkSize, s.T())

	// Validate logs.
	bufferedReadLogEntry := parseAndValidateSingleBufferedReadLog(s.T())
	expected := &Expected{BucketName: setup.TestBucket(), ObjectName: path.Join(path.Base(testDir), fileName)}
	validate(expected, bufferedReadLogEntry, false, s.T())
	assert.Equal(s.T(), int64(0), bufferedReadLogEntry.RandomSeekCount, "RandomSeekCount should be 0 for small file reads.")
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestFallbackSuites(t *testing.T) {
	// Run the suite for insufficient pool at creation time.
	insufficientPoolFlags := gcsfuseTestFlags{
		enableBufferedRead:   true,
		blockSizeMB:          8,
		minBlocksPerHandle:   2,
		globalMaxBlocks:      1, // Less than min-blocks-per-handle
		maxBlocksPerHandle:   10,
		startBlocksPerHandle: 2,
		clientProtocol:       clientProtocolHTTP1,
	}
	suite.Run(t, &InsufficientPoolCreationSuite{fallbackSuiteBase{testFlags: &insufficientPoolFlags}})

	// Run the suite for random read fallback scenarios.
	randomReadFlags := gcsfuseTestFlags{
		enableBufferedRead:   true,
		blockSizeMB:          8,
		maxBlocksPerHandle:   20,
		startBlocksPerHandle: 2,
		minBlocksPerHandle:   2,
		clientProtocol:       clientProtocolHTTP1,
	}
	suite.Run(t, &RandomReadFallbackSuite{fallbackSuiteBase{testFlags: &randomReadFlags}})
}

func readAndValidateChunk(f *os.File, testDir, fileName string, offset, chunkSize int64, t *testing.T) {
	t.Helper()
	readBuffer := make([]byte, chunkSize)

	_, err := f.ReadAt(readBuffer, offset)

	require.NoError(t, err, "ReadAt failed at offset %d", offset)
	client.ValidateObjectChunkFromGCS(ctx, storageClient, testDir, fileName, offset, chunkSize, string(readBuffer), t)
}
