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
	"fmt"
	"math/rand"
	"os"
	"path"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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
	suite.Suite
}

func (s *concurrentReadTest) SetupTest() {
	mountGCSFuseAndSetupTestDir(s.flags, testDirNameForRead, s.T())
}

func (s *concurrentReadTest) TearDownTest() {
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
func (s *concurrentReadTest) Test_ConcurrentSequentialAndRandomReads() {
	const (
		fileSize        = 500 * operations.OneMiB // 500 MiB file
		chunkSize       = 64 * operations.OneKiB  // 64 KiB chunks for reads
		sequentialReads = 5                       // Number of sequential readers
		randomReads     = 5                       // Number of random readers
	)
	// Create a 500MiB test file
	testFilePath := path.Join(testDirPathForRead, "large_test_file.bin")
	operations.CreateFileOfSize(fileSize, testFilePath, s.T())
	var wg sync.WaitGroup
	timeout := 300 * time.Second // 5 minutes timeout for 500MiB operations

	// Launch 5 sequential readers
	for i := 0; i < sequentialReads; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			// Use operations.ReadFileSequentially to read the entire file
			content, err := operations.ReadFileSequentially(testFilePath, chunkSize)
			require.NoError(s.T(), err, "Sequential reader %d: read failed.", readerID)
			require.Equal(s.T(), fileSize, len(content), "Sequential reader %d: expected to read entire file", readerID)
			obj := storageClient.Bucket(setup.TestBucket()).Object(path.Join(path.Base(testDirPathForRead), "large_test_file.bin"))
			attrs, err := obj.Attrs(ctx)
			require.NoError(s.T(), err, "obj.Attrs")
			localCRC32C, err := operations.CalculateCRC32(bytes.NewReader(content))
			require.NoError(s.T(), err, "Sequential reader %d: failed to calculate local CRC32C", readerID)
			assert.Equal(s.T(), attrs.CRC32C, localCRC32C, "Sequential reader %d: CRC32C mismatch. GCS: %d, Local: %d", readerID, attrs.CRC32C, localCRC32C)
		}(i)
	}
	// Launch 5 random readers
	for i := 0; i < randomReads; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			numRandomReads := 200 // Number of random read operations per goroutine
			rand.New(rand.NewSource(time.Now().UnixNano() + int64(readerID)))
			for j := 0; j < numRandomReads; j++ {
				// Generate random offset within bound.
				randomOffset := int64(rand.Intn(fileSize/chunkSize)) * chunkSize
				// Use operations.ReadChunkFromFile for reading chunks
				chunk, err := operations.ReadChunkFromFile(testFilePath, chunkSize, randomOffset, os.O_RDONLY)
				require.NoError(s.T(), err, "Random reader %d: ReadChunkFromFile failed at offset %d", readerID, randomOffset)
				client.ValidateObjectChunkFromGCS(ctx, storageClient, path.Base(testDirPathForRead), "large_test_file.bin", randomOffset, int64(len(chunk)), string(chunk), s.T())
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
		s.T().Log("All concurrent read operations completed successfully")
	case <-time.After(timeout):
		assert.FailNow(s.T(), "Concurrent read operations timed out - possible deadlock or performance issue")
	}
}

// Test_ConcurrentSegmentReadsSharedHandle tests concurrent read operations where
// 5 goroutines read different segments of a file using the same shared file handle.
// This test validates that multiple goroutines can safely read from different
// parts of the same file using a single shared file handle without race conditions,
// with each reader handling a distinct segment of the file for comprehensive coverage.
func (s *concurrentReadTest) Test_ConcurrentSegmentReadsSharedHandle() {
	const (
		fileSize    = 500 * operations.OneMiB // 500 MiB file
		numReaders  = 5                       // Number of concurrent readers
		segmentSize = fileSize / numReaders   // Each reader reads 100 MiB segment
	)
	// Create a 500MiB test file
	testFilePath := path.Join(testDirPathForRead, "segment_test_file.bin")
	operations.CreateFileOfSize(fileSize, testFilePath, s.T())
	// Open shared file handle that will be used by all goroutines
	sharedFile, err := os.Open(testFilePath)
	require.NoError(s.T(), err, "Failed to open shared file handle")
	defer func() {
		err := sharedFile.Close()
		require.NoError(s.T(), err, "Failed to close shared file handle")
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
			require.NoError(s.T(), err, "Reader %d: ReadAt failed for segment %d-%d", readerID, segmentStart, segmentEnd-1)
			require.Equal(s.T(), int(actualSegmentSize), n, "Reader %d: expected to read %d bytes, got %d", readerID, actualSegmentSize, n)
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
		s.T().Log("All concurrent segment read operations completed successfully")
		// Reconstruct the full file from segments and validate checksum
		var fullContent bytes.Buffer
		for i, segment := range segmentData {
			n, err := fullContent.Write(segment)
			require.NoError(s.T(), err, "Failed to write segment %d to buffer", i)
			require.Equal(s.T(), len(segment), n, "Segment %d: wrote different number of bytes than expected", i)
		}
		// Validate total size
		require.Equal(s.T(), fileSize, fullContent.Len(), "Reconstructed file size mismatch")
		// Validate checksum of reconstructed content
		reconstructedChecksum, err := operations.CalculateCRC32(bytes.NewReader(fullContent.Bytes()))
		require.NoError(s.T(), err, "Failed to calculate reconstructed checksum")
		obj := storageClient.Bucket(setup.TestBucket()).Object(path.Join(path.Base(testDirPathForRead), "segment_test_file.bin"))
		attrs, err := obj.Attrs(ctx)
		require.NoError(s.T(), err, "obj.Attrs")
		assert.Equal(s.T(), attrs.CRC32C, reconstructedChecksum, "CRC32C mismatch. GCS: %d, Local: %d", attrs.CRC32C, reconstructedChecksum)
	case <-time.After(timeout):
		assert.FailNow(s.T(), "Concurrent segment read operations timed out - possible deadlock or performance issue")
	}
}

func (s *concurrentReadTest) Test_ConcurrentReadPlusWrite() {
	const (
		fileSize      = 32 * operations.OneMiB  // 32 MiB file
		numGoRoutines = 10                      // Number of concurrent readers
		chunkSize     = 128 * operations.OneKiB // 128 KiB chunks for reads
	)
	var wg sync.WaitGroup
	wg.Add(numGoRoutines)
	timeout := 300 * time.Second

	for i := 0; i < numGoRoutines; i++ {
		go func(workerId int) {
			defer wg.Done()

			fileName := fmt.Sprintf("test_%d.bin", workerId)
			filePath := path.Join(testDirPathForRead, fileName)
			f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC|syscall.O_DIRECT, setup.FilePermission_0600) //nolint:staticcheck
			require.NoError(s.T(), err)
			randomData, err := operations.GenerateRandomData(fileSize)
			require.NoError(s.T(), err)
			n, err := f.Write(randomData)
			require.NoError(s.T(), err)
			require.Equal(s.T(), fileSize, n)
			operations.CloseFileShouldNotThrowError(s.T(), f)

			content, err := operations.ReadFileSequentially(filePath, chunkSize)
			require.NoError(s.T(), err, "Sequential reader %d: read failed.", workerId)
			require.Equal(s.T(), fileSize, len(content), "Sequential reader %d: expected to read entire file", workerId)
			obj := storageClient.Bucket(setup.TestBucket()).Object(path.Join(path.Base(testDirPathForRead), fileName))
			attrs, err := obj.Attrs(ctx)
			require.NoError(s.T(), err, "obj.Attrs")
			localCRC32C, err := operations.CalculateCRC32(bytes.NewReader(content))
			require.NoError(s.T(), err, "Sequential reader %d: failed to calculate local CRC32C", workerId)
			assert.Equal(s.T(), attrs.CRC32C, localCRC32C, "Sequential reader %d: CRC32C mismatch. GCS: %d, Local: %d", workerId, attrs.CRC32C, localCRC32C)
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
		s.T().Log("All concurrent goroutines completed successfully")
	case <-time.After(timeout):
		assert.FailNow(s.T(), "Concurrent go routines timed out.")
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestConcurrentRead(t *testing.T) {
	ts := &concurrentReadTest{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, ts)
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
		suite.Run(t, ts)
	}
}
