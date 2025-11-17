// Copyright 2024 Google LLC
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

package bufferedwrites

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/fake"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/gcs"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/storageutil"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"golang.org/x/sync/semaphore"
)

const chunkTransferTimeoutSecs int64 = 10

var errUploadFailure = errors.New("error while uploading object to GCS")

type BufferedWriteTest struct {
	bwh             BufferedWriteHandler
	globalSemaphore *semaphore.Weighted
	suite.Suite
}

func TestBufferedWriteTestSuite(t *testing.T) {
	suite.Run(t, new(BufferedWriteTest))
}

func (testSuite *BufferedWriteTest) SetupTest() {
	bucketType := gcs.BucketType{}
	testSuite.setupTestWithBucketType(bucketType)
}

func (testSuite *BufferedWriteTest) setupTestWithBucketType(bucketType gcs.BucketType) {
	bucket := fake.NewFakeBucket(timeutil.RealClock(), "FakeBucketName", bucketType)
	testSuite.globalSemaphore = semaphore.NewWeighted(10)
	bwh, err := NewBWHandler(&CreateBWHandlerRequest{
		Object:                   nil,
		ObjectName:               "testObject",
		Bucket:                   bucket,
		BlockSize:                blockSize,
		MaxBlocksPerFile:         10,
		GlobalMaxBlocksSem:       testSuite.globalSemaphore,
		ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
	})
	require.Nil(testSuite.T(), err)
	testSuite.bwh = bwh
}

func (testSuite *BufferedWriteTest) TestSetMTime() {
	testTime := time.Now()

	testSuite.bwh.SetMtime(testTime)

	assert.Equal(testSuite.T(), testTime, testSuite.bwh.WriteFileInfo().Mtime)
	assert.Equal(testSuite.T(), int64(0), testSuite.bwh.WriteFileInfo().TotalSize)
}

func (testSuite *BufferedWriteTest) TestWrite() {
	err := testSuite.bwh.Write([]byte("hi"), 0)

	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(2), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteWithEmptyBuffer() {
	err := testSuite.bwh.Write([]byte{}, 0)

	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(0), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteEqualToBlockSize() {
	size := 1024
	data := strings.Repeat("A", size)

	err := testSuite.bwh.Write([]byte(data), 0)

	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(size), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteDataSizeGreaterThanBlockSize() {
	size := 2000
	data := strings.Repeat("A", size)

	err := testSuite.bwh.Write([]byte(data), 0)

	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(size), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteWhenNextOffsetIsGreaterThanExpected() {
	err := testSuite.bwh.Write([]byte("hi"), 0)
	require.Nil(testSuite.T(), err)

	// Next offset should be 2, but we are calling with 5.
	err = testSuite.bwh.Write([]byte("hello"), 5)

	require.NotNil(testSuite.T(), err)
	require.Equal(testSuite.T(), err, ErrOutOfOrderWrite)
	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(2), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteWhenNextOffsetIsLessThanExpected() {
	err := testSuite.bwh.Write([]byte("hello"), 0)
	require.Nil(testSuite.T(), err)

	// Next offset should be 5, but we are calling with 2.
	err = testSuite.bwh.Write([]byte("abcdefgh"), 2)

	require.NotNil(testSuite.T(), err)
	require.Equal(testSuite.T(), err, ErrOutOfOrderWrite)
	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(5), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestMultipleWrites() {
	err := testSuite.bwh.Write([]byte("hello"), 0)
	require.Nil(testSuite.T(), err)

	err = testSuite.bwh.Write([]byte("abcdefgh"), 5)
	require.Nil(testSuite.T(), err)

	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(13), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteWithSignalUploadFailureInBetween() {
	err := testSuite.bwh.Write([]byte("hello"), 0)
	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(5), fileInfo.TotalSize)

	// Set an error to simulate failure in uploader.
	bwhImpl.uploadHandler.uploadError.Store(&errUploadFailure)

	err = testSuite.bwh.Write([]byte("hello"), 5)
	require.Error(testSuite.T(), err)
	assert.Equal(testSuite.T(), err, errUploadFailure)
}

func (testSuite *BufferedWriteTest) TestWriteAtTruncatedOffset() {
	// Truncate
	err := testSuite.bwh.Truncate(2)
	require.NoError(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	require.Equal(testSuite.T(), int64(2), bwhImpl.truncatedSize)

	// Write at offset = truncatedSize
	err = testSuite.bwh.Write([]byte("hello"), 2)

	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(7), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteAfterTruncateAtCurrentSize() {
	err := testSuite.bwh.Write([]byte("hello"), 0)
	require.Nil(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	require.Equal(testSuite.T(), int64(5), bwhImpl.totalSize)
	// Truncate
	err = testSuite.bwh.Truncate(20)
	require.NoError(testSuite.T(), err)
	require.Equal(testSuite.T(), int64(20), bwhImpl.truncatedSize)
	require.Equal(testSuite.T(), int64(20), testSuite.bwh.WriteFileInfo().TotalSize)

	// Write at offset=bwh.totalSize
	err = testSuite.bwh.Write([]byte("abcde"), 5)

	require.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), int64(10), bwhImpl.totalSize)
	assert.Equal(testSuite.T(), int64(20), testSuite.bwh.WriteFileInfo().TotalSize)
}

func (testSuite *BufferedWriteTest) TestFlushWithNonNilCurrentBlock() {
	err := testSuite.bwh.Write([]byte("hi"), 0)
	require.Nil(testSuite.T(), err)

	obj, err := testSuite.bwh.Flush()

	require.NoError(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), nil, bwhImpl.current)
	// Validate object.
	assert.NotNil(testSuite.T(), obj)
	assert.Equal(testSuite.T(), uint64(2), obj.Size)
	// Validate that all blocks have been freed up.
	assert.Equal(testSuite.T(), 0, bwhImpl.uploadHandler.blockPool.TotalFreeBlocks())
}

func (testSuite *BufferedWriteTest) TestFlushWithNilCurrentBlock() {
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	require.Nil(testSuite.T(), bwhImpl.current)

	obj, err := testSuite.bwh.Flush()

	assert.NoError(testSuite.T(), err)
	// Validate empty object created.
	assert.NotNil(testSuite.T(), obj)
	assert.Equal(testSuite.T(), uint64(0), obj.Size)
}

func (testSuite *BufferedWriteTest) TestFlushWithSignalUploadFailureDuringWrite() {
	err := testSuite.bwh.Write([]byte("hi"), 0)
	require.Nil(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)

	// Set an error to simulate failure in uploader.
	bwhImpl.uploadHandler.uploadError.Store(&errUploadFailure)

	obj, err := testSuite.bwh.Flush()
	require.Error(testSuite.T(), err)
	assert.Equal(testSuite.T(), err, errUploadFailure)
	assert.Nil(testSuite.T(), obj)
}

func (testSuite *BufferedWriteTest) TestFlushWithMultiBlockWritesAndSignalUploadFailureInBetween() {
	buffer, err := operations.GenerateRandomData(blockSize)
	assert.NoError(testSuite.T(), err)
	// Upload and sync 5 blocks.
	testSuite.TestSync5InProgressBlocks()
	// Set an error to simulate failure in uploader.
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	bwhImpl.uploadHandler.uploadError.Store(&errUploadFailure)
	// Write 5 more blocks.
	for i := range 5 {
		err := testSuite.bwh.Write(buffer, int64(blockSize*(i+5)))
		require.Error(testSuite.T(), err)
		assert.Equal(testSuite.T(), errUploadFailure, err)
	}

	obj, err := testSuite.bwh.Flush()

	require.Error(testSuite.T(), err)
	assert.Equal(testSuite.T(), err, errUploadFailure)
	assert.Nil(testSuite.T(), obj)
}

func (testSuite *BufferedWriteTest) TestSync5InProgressBlocks() {
	buffer, err := operations.GenerateRandomData(blockSize)
	assert.NoError(testSuite.T(), err)
	// Write 5 blocks.
	for i := range 5 {
		err = testSuite.bwh.Write(buffer, int64(blockSize*i))
		require.Nil(testSuite.T(), err)
	}

	// Wait for 5 blocks to upload successfully.
	o, err := testSuite.bwh.Sync()

	assert.NoError(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), 0, len(bwhImpl.uploadHandler.uploadCh))
	assert.Equal(testSuite.T(), 0, bwhImpl.uploadHandler.blockPool.TotalFreeBlocks())
	assert.Nil(testSuite.T(), o)
}

func (testSuite *BufferedWriteTest) TestSyncPartialBlockTableDriven() {
	testCases := []struct {
		name       string
		bucketType gcs.BucketType
		numBlocks  float32
	}{
		{
			name:       "multi_regional_bucket_2.5_blocks",
			bucketType: gcs.BucketType{},
			numBlocks:  2.5,
		},
		{
			name:       "multi_regional_bucket_.5_blocks",
			bucketType: gcs.BucketType{},
			numBlocks:  .5,
		},
		{
			name:       "zonal_bucket_2.5_blocks",
			bucketType: gcs.BucketType{Zonal: true},
			numBlocks:  2.5,
		},
		{
			name:       "zonal_bucket_.5_blocks",
			bucketType: gcs.BucketType{Zonal: true},
			numBlocks:  .5,
		},
	}

	for _, tc := range testCases {
		testSuite.Run(tc.name, func() {
			testSuite.setupTestWithBucketType(tc.bucketType)
			buffer, err := operations.GenerateRandomData(int64(blockSize * tc.numBlocks))
			assert.NoError(testSuite.T(), err)
			err = testSuite.bwh.Write(buffer, 0)
			require.Nil(testSuite.T(), err)

			// Wait for blocks to upload successfully.
			o, err := testSuite.bwh.Sync()

			require.NoError(testSuite.T(), err)
			bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
			// Current block should also be uploaded.
			assert.Nil(testSuite.T(), bwhImpl.current)
			assert.Equal(testSuite.T(), 0, len(bwhImpl.uploadHandler.uploadCh))
			assert.Equal(testSuite.T(), 0, bwhImpl.uploadHandler.blockPool.TotalFreeBlocks())
			// Read the object from back door.
			content, err := storageutil.ReadObject(context.Background(), bwhImpl.uploadHandler.bucket, bwhImpl.uploadHandler.objectName)
			if tc.bucketType.Zonal {
				require.NotNil(testSuite.T(), o)
				assert.EqualValues(testSuite.T(), int64(blockSize*tc.numBlocks), o.Size)
				require.NoError(testSuite.T(), err)
				assert.Equal(testSuite.T(), buffer, content)
			} else {
				require.Nil(testSuite.T(), o)
				// Since the object is not finalized, the object will not be available
				// on GCS for non-zonal buckets.
				require.Error(testSuite.T(), err)
				var notFoundErr *gcs.NotFoundError
				assert.ErrorAs(testSuite.T(), err, &notFoundErr)
			}
		})
	}
}

func (testSuite *BufferedWriteTest) TestSyncBlocksWithError() {
	buffer, err := operations.GenerateRandomData(blockSize)
	assert.NoError(testSuite.T(), err)
	// Write 5 blocks.
	for i := range 5 {
		err = testSuite.bwh.Write(buffer, int64(blockSize*i))
		require.Nil(testSuite.T(), err)
	}
	// Set an error to simulate failure in uploader.
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	bwhImpl.uploadHandler.uploadError.Store(&errUploadFailure)

	o, err := testSuite.bwh.Sync()

	assert.Error(testSuite.T(), err)
	assert.Equal(testSuite.T(), errUploadFailure, err)
	assert.Nil(testSuite.T(), o)
}

func (testSuite *BufferedWriteTest) TestFlushWithNonZeroTruncatedLengthForEmptyObject() {
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	require.Nil(testSuite.T(), bwhImpl.current)
	bwhImpl.truncatedSize = 10

	_, err := testSuite.bwh.Flush()

	assert.NoError(testSuite.T(), err)
	assert.Equal(testSuite.T(), bwhImpl.truncatedSize, bwhImpl.totalSize)
}

func (testSuite *BufferedWriteTest) TestFlushWithTruncatedLengthGreaterThanObjectSize() {
	err := testSuite.bwh.Write([]byte("hi"), 0)
	require.Nil(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	bwhImpl.truncatedSize = 10

	_, err = testSuite.bwh.Flush()

	assert.NoError(testSuite.T(), err)
	assert.Equal(testSuite.T(), bwhImpl.truncatedSize, bwhImpl.totalSize)
}

func (testSuite *BufferedWriteTest) TestTruncateWithLesserSize() {
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	bwhImpl.totalSize = 10

	err := testSuite.bwh.Truncate(2)

	assert.Error(testSuite.T(), err)
	assert.Equal(testSuite.T(), ErrOutOfOrderWrite, err)
}

func (testSuite *BufferedWriteTest) TestTruncateWithSizeGreaterThanCurrentObjectSize() {
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	bwhImpl.totalSize = 10

	err := testSuite.bwh.Truncate(12)

	assert.NoError(testSuite.T(), err)
	assert.Equal(testSuite.T(), int64(12), bwhImpl.truncatedSize)
}

func (testSuite *BufferedWriteTest) TestWriteFileInfoWithTruncatedLengthLessThanTotalSize() {
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	bwhImpl.totalSize = 10
	bwhImpl.truncatedSize = 5

	fileInfo := testSuite.bwh.WriteFileInfo()

	assert.Equal(testSuite.T(), bwhImpl.totalSize, fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteFileInfoWithTruncatedLengthGreaterThanTotalSize() {
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	bwhImpl.totalSize = 10
	bwhImpl.truncatedSize = 20

	fileInfo := testSuite.bwh.WriteFileInfo()

	assert.Equal(testSuite.T(), bwhImpl.truncatedSize, fileInfo.TotalSize)
}
func (testSuite *BufferedWriteTest) TestDestroyShouldClearFreeBlockChannel() {
	// Try to write 4 blocks of data.
	contents := strings.Repeat("A", blockSize*4)
	err := testSuite.bwh.Write([]byte(contents), 0)
	require.Nil(testSuite.T(), err)

	err = testSuite.bwh.Destroy()

	require.Nil(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), 0, bwhImpl.uploadHandler.blockPool.TotalFreeBlocks())
	assert.Equal(testSuite.T(), 0, len(bwhImpl.uploadHandler.uploadCh))
}

func (testSuite *BufferedWriteTest) TestUnlinkBeforeWrite() {
	testSuite.bwh.Unlink()

	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Nil(testSuite.T(), bwhImpl.uploadHandler.cancelFunc)
	assert.Equal(testSuite.T(), 0, len(bwhImpl.uploadHandler.uploadCh))
	assert.Equal(testSuite.T(), 0, bwhImpl.uploadHandler.blockPool.TotalFreeBlocks())
	// Check if semaphore is released correctly. Last block should not be released.
	assert.True(testSuite.T(), testSuite.globalSemaphore.TryAcquire(9))
	assert.False(testSuite.T(), testSuite.globalSemaphore.TryAcquire(1))
}

func (testSuite *BufferedWriteTest) TestUnlinkAfterWrite() {
	buffer, err := operations.GenerateRandomData(blockSize)
	assert.NoError(testSuite.T(), err)
	// Write 5 blocks.
	for i := range 5 {
		err = testSuite.bwh.Write(buffer, int64(blockSize*i))
		require.Nil(testSuite.T(), err)
	}
	cancelCalled := false
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	bwhImpl.uploadHandler.cancelFunc = func() { cancelCalled = true }

	testSuite.bwh.Unlink()

	assert.True(testSuite.T(), cancelCalled)
	assert.Equal(testSuite.T(), 0, len(bwhImpl.uploadHandler.uploadCh))
	assert.Equal(testSuite.T(), 0, bwhImpl.uploadHandler.blockPool.TotalFreeBlocks())
	// Check if semaphore is released correctly. Last block should not be released.
	assert.True(testSuite.T(), testSuite.globalSemaphore.TryAcquire(9))
	assert.False(testSuite.T(), testSuite.globalSemaphore.TryAcquire(1))
}

func (testSuite *BufferedWriteTest) TestReFlushAfterUploadFails() {
	testSuite.TestFlushWithMultiBlockWritesAndSignalUploadFailureInBetween()

	// Re-flush.
	obj, err := testSuite.bwh.Flush()

	require.Error(testSuite.T(), err)
	assert.Nil(testSuite.T(), obj)
	assert.ErrorContains(testSuite.T(), err, errUploadFailure.Error())
}
