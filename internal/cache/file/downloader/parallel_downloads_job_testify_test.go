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

package downloader

import (
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/cfg"
	"github.com/vipnydav/gcsfuse/v3/internal/cache/data"
	"github.com/vipnydav/gcsfuse/v3/internal/cache/util"
	"github.com/vipnydav/gcsfuse/v3/internal/storage"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/fake"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/gcs"
	testutil "github.com/vipnydav/gcsfuse/v3/internal/util"
	"golang.org/x/net/context"
)

type ParallelDownloaderJobTestifyTest struct {
	JobTestifyTest
}

func TestParallelDownloaderJobTestifyTestSuite(testSuite *testing.T) {
	suite.Run(testSuite, new(ParallelDownloaderJobTestifyTest))
}

func (t *ParallelDownloaderJobTestifyTest) SetupTest() {
	t.defaultFileCacheConfig = &cfg.FileCacheConfig{
		EnableParallelDownloads:  true,
		ParallelDownloadsPerFile: 3,
		DownloadChunkSizeMb:      3,
		EnableCrc:                true,
		WriteBufferSize:          4 * 1024 * 1024,
	}
	t.ctx, _ = context.WithCancel(context.Background())
	t.mockBucket = new(storage.TestifyMockBucket)
}

func (t *ParallelDownloaderJobTestifyTest) Test_ParallelDownloadObjectToFile_NewReaderWithReadHandle() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 10 * util.MiB
	chunkSize := 3 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	t.initReadCacheTestifyTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), func() {})
	t.job.cancelCtx, t.job.cancelFunc = context.WithCancel(context.Background())
	// Add subscriber
	subscribedOffset := int64(1 * util.MiB)
	notificationC := t.job.subscribe(subscribedOffset)
	file, err := util.CreateFile(data.FileSpec{Path: t.job.fileSpec.Path,
		FilePerm: os.FileMode(0600), DirPerm: os.FileMode(0700)}, os.O_TRUNC|os.O_RDWR)
	assert.Equal(t.T(), nil, err)
	defer func() {
		_ = file.Close()
	}()
	// To download a file of 10mb using ParallelDownloadsPerFile = 3 and
	// DownloadChunkSizeMb = 3mb there will be one call to NewReaderWithReadHandle
	// with read handle.
	handle := []byte("opaque-handle")
	var (
		actualCallCount    int64
		nilHandleCallCount int64
		mu                 sync.Mutex // To protect counter updates from concurrent mock calls
	)
	// Reset counters
	actualCallCount = 0
	nilHandleCallCount = 0

	// Helper function to increment counters based on the actual request seen by the mock
	incrementCounters := func(args mock.Arguments) {
		req := args.Get(1).(*gcs.ReadObjectRequest) // Second arg to NewReaderWithReadHandle
		mu.Lock()
		actualCallCount++
		if req.ReadHandle == nil {
			nilHandleCallCount++
		}
		mu.Unlock()
	}

	// Define ranges for clarity
	rangeR1 := &gcs.ByteRange{Start: uint64(0 * chunkSize), Limit: uint64(1 * chunkSize)}
	rangeR2 := &gcs.ByteRange{Start: uint64(1 * chunkSize), Limit: uint64(2 * chunkSize)}
	rangeR3 := &gcs.ByteRange{Start: uint64(2 * chunkSize), Limit: uint64(3 * chunkSize)}
	rangeR4 := &gcs.ByteRange{Start: uint64(3 * chunkSize), Limit: uint64(objectSize)} // Last chunk

	// Create FakeReaders for each chunk, all will return the same propagatedHandle
	readerR1 := &fake.FakeReader{ReadCloser: io.NopCloser(strings.NewReader(string(objectContent[0*chunkSize : 1*chunkSize]))), Handle: handle}
	readerR2 := &fake.FakeReader{ReadCloser: io.NopCloser(strings.NewReader(string(objectContent[1*chunkSize : 2*chunkSize]))), Handle: handle}
	readerR3 := &fake.FakeReader{ReadCloser: io.NopCloser(strings.NewReader(string(objectContent[2*chunkSize : 3*chunkSize]))), Handle: handle}
	readerR4 := &fake.FakeReader{ReadCloser: io.NopCloser(strings.NewReader(string(objectContent[3*chunkSize : objectSize]))), Handle: handle}

	t.mockBucket.On("Name").Return(storage.TestBucketName)

	// Chunk 1 (R1): Must be ReadHandle: nil
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
		return req.Range.Start == rangeR1.Start && req.Range.Limit == rangeR1.Limit &&
			req.ReadHandle == nil
	})).Run(incrementCounters).Return(readerR1, nil).Once()

	// Chunk 2 (R2): ReadHandle can be nil or propagated. Match primarily on range.
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
		return req.Range.Start == rangeR2.Start && req.Range.Limit == rangeR2.Limit
	})).Run(incrementCounters).Return(readerR2, nil).Once()

	// Chunk 3 (R3): ReadHandle can be nil or propagated. Match primarily on range.
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
		return req.Range.Start == rangeR3.Start && req.Range.Limit == rangeR3.Limit
	})).Run(incrementCounters).Return(readerR3, nil).Once()

	// Chunk 4 (R4): ReadHandle should not be nil
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
		return req.Range.Start == rangeR4.Start && req.Range.Limit == rangeR4.Limit && req.ReadHandle != nil
	})).Run(incrementCounters).Return(readerR4, nil).Once()

	// Start download
	err = t.job.parallelDownloadObjectToFile(file)

	assert.Equal(t.T(), nil, err, "parallelDownloadObjectToFile should not return an error")
	assert.Equal(t.T(), int64(4), actualCallCount, "Total calls to NewReaderWithReadHandle should be 4")
	// Assert the number of calls with ReadHandle: nil falls within the expected range.
	// For ParallelDownloadsPerFile = 3 and 4 chunks:
	// - nilHandleCallCount must be at least 1 (for the very first chunk processed by any worker).
	// - nilHandleCallCount can be at most ParallelDownloadsPerFile (3), as each of the 3 launched workers
	//   uses a nil handle only for its first operation.
	// 1 <= nilHandleCallCount <= 3.
	minExpectedNilCalls := int64(1)
	maxExpectedNilCalls := int64(3)
	numberOfChunks := int64(4) // Based on objectSize and chunkSize

	assert.True(t.T(), nilHandleCallCount >= minExpectedNilCalls && nilHandleCallCount <= maxExpectedNilCalls,
		"Expected nilHandleCallCount to be between %d and %d (inclusive), but got %d. ParallelDownloadsPerFile=%d, Chunks=%d",
		minExpectedNilCalls, maxExpectedNilCalls, nilHandleCallCount, t.job.fileCacheConfig.ParallelDownloadsPerFile, numberOfChunks)

	t.mockBucket.AssertExpectations(t.T())
	assert.Equal(t.T(), nil, err)
	jobStatus, ok := <-notificationC
	assert.Equal(t.T(), true, ok)
	// Check the notification is sent after subscribed offset
	assert.GreaterOrEqual(t.T(), jobStatus.Offset, subscribedOffset)
	t.job.mu.Lock()
	defer t.job.mu.Unlock()
	// Verify file is downloaded
	verifyCompleteFile(t.T(), t.fileSpec, objectContent)
	// Verify fileInfoCache update
	verifyFileInfoEntry(t.T(), t.mockBucket, t.object, t.cache, uint64(objectSize))
}
