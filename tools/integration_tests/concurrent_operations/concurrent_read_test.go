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

package concurrent_operations

import (
	"bytes"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDirNameForRead = "concurrent_read_test"
)

var testDirPathForRead string

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type concurrentReadTest struct {
	flags []string
}

func (s *concurrentReadTest) Setup(t *testing.T) {
	mountGCSFuseAndSetupTestDir(s.flags, testDirNameForRead, t)
}

func (s *concurrentReadTest) Teardown(t *testing.T) {
	setup.UnmountGCSFuse(setup.MntDir())
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

// Test_ConcurrentSequentialAndRandomReads tests concurrent read operations where
// 5 goroutines read a 500MiB file sequentially and 5 goroutines read randomly.
// This test validates that concurrent sequential and random read patterns work
// correctly without deadlocks or race conditions. It also validates data integrity
// using CRC32 checksums for sequential reads and chunk validation for random reads.
func (s *concurrentReadTest) Test_ConcurrentSequentialAndRandomReads(t *testing.T) {
	const (
		fileSize        = 500 * operations.OneMiB // 500 MiB file
		chunkSize       = 64 * operations.OneKiB  // 64 KiB chunks for reads
		sequentialReads = 5                       // Number of sequential readers
		randomReads     = 5                       // Number of random readers
	)
	testCaseDir := "Test_ConcurrentSequentialAndRandomReads"
	operations.CreateDirectory(path.Join(testDirPathForRead, testCaseDir), t)
	// Create a 500MiB test file
	testFilePath := path.Join(testDirPathForRead, testCaseDir, "large_test_file.bin")
	operations.CreateFileOfSize(fileSize, testFilePath, t)
	// Calculate expected checksum of the entire file
	expectedChecksum, err := operations.CalculateFileCRC32(testFilePath)
	require.NoError(t, err, "Failed to calculate expected checksum")
	var wg sync.WaitGroup
	timeout := 300 * time.Second // 5 minutes timeout for 500MiB operations

	// Launch 5 sequential readers
	for i := 0; i < sequentialReads; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			// Use operations.ReadFileSequentially to read the entire file
			content, err := operations.ReadFileSequentially(testFilePath, chunkSize)
			require.NoError(t, err, "Sequential reader %d: failed to read file sequentially", readerID)
			require.Equal(t, fileSize, len(content), "Sequential reader %d: expected to read entire file", readerID)
			// Calculate checksum of read content
			readerChecksum, err := operations.CalculateCRC32(bytes.NewReader(content))
			require.NoError(t, err, "Sequential reader %d: failed to calculate checksum", readerID)
			// Validate checksum matches expected
			require.Equal(t, expectedChecksum, readerChecksum, "Sequential reader %d: checksum mismatch", readerID)
		}(i)
	}
	// Launch 5 random readers
	for i := 0; i < randomReads; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			numRandomReads := 200 // Number of random read operations per goroutine
			// Use a simple pseudo-random generator to avoid contention on global rand
			seed := time.Now().UnixNano() + int64(readerID)
			// Perform random reads using operations.ReadChunkFromFile
			for j := 0; j < numRandomReads; j++ {
				// Generate random offset within file bounds, aligned to chunk boundaries
				seed = (seed*1103515245 + 12345) % (1 << 31) // Simple LCG
				randomOffset := (seed % (fileSize / chunkSize)) * chunkSize
				// Use operations.ReadChunkFromFile for reading chunks
				chunk, err := operations.ReadChunkFromFile(testFilePath, chunkSize, randomOffset, os.O_RDONLY)
				require.NoError(t, err, "Random reader %d: ReadChunkFromFile failed at offset %d", readerID, randomOffset)
				if len(chunk) > 0 {
					// Validate the chunk by reading the same range sequentially and comparing
					expectedChunk, err := operations.ReadChunkFromFile(testFilePath, int64(len(chunk)), randomOffset, os.O_RDONLY)
					require.NoError(t, err, "Random reader %d: validation read failed at offset %d", readerID, randomOffset)
					require.Equal(t, expectedChunk, chunk, "Random reader %d: chunk content mismatch at offset %d", readerID, randomOffset)
				}
			}
		}(i)
	}
	// Wait for all goroutines or timeout
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		t.Log("All concurrent read operations completed successfully")
	case <-time.After(timeout):
		assert.FailNow(t, "Concurrent read operations timed out - possible deadlock or performance issue")
	}
}

// Test_ConcurrentSegmentReadsSharedHandle tests concurrent read operations where
// 5 goroutines read different segments of a file using the same shared file handle.
// This test validates that multiple goroutines can safely read from different
// parts of the same file using a single shared file handle without race conditions,
// with each reader handling a distinct segment of the file for comprehensive coverage.
func (s *concurrentReadTest) Test_ConcurrentSegmentReadsSharedHandle(t *testing.T) {
	const (
		fileSize    = 500 * operations.OneMiB // 500 MiB file
		numReaders  = 5                       // Number of concurrent readers
		segmentSize = fileSize / numReaders   // Each reader reads 100 MiB segment
	)
	testCaseDir := "Test_ConcurrentSegmentReadsSharedHandle"
	operations.CreateDirectory(path.Join(testDirPathForRead, testCaseDir), t)
	// Create a 500MiB test file
	testFilePath := path.Join(testDirPathForRead, testCaseDir, "segment_test_file.bin")
	operations.CreateFileOfSize(fileSize, testFilePath, t)
	// Calculate expected checksum of the entire file for validation
	expectedChecksum, err := operations.CalculateFileCRC32(testFilePath)
	require.NoError(t, err, "Failed to calculate expected checksum")
	// Open shared file handle that will be used by all goroutines
	sharedFile, err := os.Open(testFilePath)
	require.NoError(t, err, "Failed to open shared file handle")
	defer func() {
		err := sharedFile.Close()
		require.NoError(t, err, "Failed to close shared file handle")
	}()
	var wg sync.WaitGroup
	segmentData := make([][]byte, numReaders)
	timeout := 300 * time.Second // 5 minutes timeout

	// Launch 5 readers, each reading a different segment using the shared file handle
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			// Calculate segment boundaries
			segmentStart := int64(readerID) * segmentSize
			segmentEnd := segmentStart + segmentSize
			if readerID == numReaders-1 {
				// Last reader takes any remaining bytes
				segmentEnd = fileSize
			}
			actualSegmentSize := segmentEnd - segmentStart
			// Read segment using shared file handle with ReadAt
			buffer := make([]byte, actualSegmentSize)
			n, err := sharedFile.ReadAt(buffer, segmentStart)
			require.NoError(t, err, "Reader %d: ReadAt failed for segment %d-%d", readerID, segmentStart, segmentEnd-1)
			require.Equal(t, int(actualSegmentSize), n, "Reader %d: expected to read %d bytes, got %d", readerID, actualSegmentSize, n)
			// Store segment data for later validation
			segmentData[readerID] = buffer
		}(i)
	}
	// Wait for all goroutines or timeout
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		t.Log("All concurrent segment read operations completed successfully")
		// Reconstruct the full file from segments and validate checksum
		var fullContent bytes.Buffer
		for i, segment := range segmentData {
			n, err := fullContent.Write(segment)
			require.NoError(t, err, "Failed to write segment %d to buffer", i)
			require.Equal(t, len(segment), n, "Segment %d: wrote different number of bytes than expected", i)
		}
		// Validate total size
		require.Equal(t, fileSize, fullContent.Len(), "Reconstructed file size mismatch")
		// Validate checksum of reconstructed content
		reconstructedChecksum, err := operations.CalculateCRC32(bytes.NewReader(fullContent.Bytes()))
		require.NoError(t, err, "Failed to calculate reconstructed checksum")
		require.Equal(t, expectedChecksum, reconstructedChecksum, "Reconstructed file checksum mismatch")
	case <-time.After(timeout):
		assert.FailNow(t, "Concurrent segment read operations timed out - possible deadlock or performance issue")
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestConcurrentRead(t *testing.T) {
	ts := &concurrentReadTest{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	// Define flag sets specific for concurrent read tests
	flagsSet := [][]string{
		{},                         // For default read path.
		{"--enable-buffered-read"}, // For Buffered read enabled.
	}

	// Run tests with each flag set
	for _, flags := range flagsSet {
		ts.flags = flags
		test_setup.RunTests(t, ts)
	}
}
