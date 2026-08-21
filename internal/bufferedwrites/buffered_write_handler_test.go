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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	storagemock "github.com/googlecloudplatform/gcsfuse/v3/internal/storage/mock"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	metricSdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"golang.org/x/sync/semaphore"
)

const (
	chunkRetryDeadlineSecs   int64 = 120
	chunkTransferTimeoutSecs int64 = 10
)

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
		ChunkRetryDeadlineSecs:   chunkRetryDeadlineSecs,
		ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
		TraceHandle:              tracing.NewNoopTracer(),
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
	err := testSuite.bwh.Write(context.Background(), []byte("hi"), 0)

	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(2), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteWithEmptyBuffer() {
	err := testSuite.bwh.Write(context.Background(), []byte{}, 0)

	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(0), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteEqualToBlockSize() {
	size := 1024
	data := strings.Repeat("A", size)

	err := testSuite.bwh.Write(context.Background(), []byte(data), 0)

	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(size), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteDataSizeGreaterThanBlockSize() {
	size := 2000
	data := strings.Repeat("A", size)

	err := testSuite.bwh.Write(context.Background(), []byte(data), 0)

	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(size), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteWhenNextOffsetIsGreaterThanExpected() {
	err := testSuite.bwh.Write(context.Background(), []byte("hi"), 0)
	require.Nil(testSuite.T(), err)

	// Next offset should be 2, but we are calling with 5.
	err = testSuite.bwh.Write(context.Background(), []byte("hello"), 5)

	require.NotNil(testSuite.T(), err)
	require.Equal(testSuite.T(), err, ErrOutOfOrderWrite)
	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(2), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteWhenNextOffsetIsLessThanExpected() {
	err := testSuite.bwh.Write(context.Background(), []byte("hello"), 0)
	require.Nil(testSuite.T(), err)

	// Next offset should be 5, but we are calling with 2.
	err = testSuite.bwh.Write(context.Background(), []byte("abcdefgh"), 2)

	require.NotNil(testSuite.T(), err)
	require.Equal(testSuite.T(), err, ErrOutOfOrderWrite)
	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(5), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestMultipleWrites() {
	err := testSuite.bwh.Write(context.Background(), []byte("hello"), 0)
	require.Nil(testSuite.T(), err)

	err = testSuite.bwh.Write(context.Background(), []byte("abcdefgh"), 5)
	require.Nil(testSuite.T(), err)

	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(13), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteWithSignalUploadFailureInBetween() {
	err := testSuite.bwh.Write(context.Background(), []byte("hello"), 0)
	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(5), fileInfo.TotalSize)

	// Set an error to simulate failure in uploader.
	bwhImpl.uploadHandler.uploadError.Store(&errUploadFailure)

	err = testSuite.bwh.Write(context.Background(), []byte("hello"), 5)
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
	err = testSuite.bwh.Write(context.Background(), []byte("hello"), 2)

	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	assert.Equal(testSuite.T(), bwhImpl.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(7), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteAfterTruncateAtCurrentSize() {
	err := testSuite.bwh.Write(context.Background(), []byte("hello"), 0)
	require.Nil(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	require.Equal(testSuite.T(), int64(5), bwhImpl.totalSize)
	// Truncate
	err = testSuite.bwh.Truncate(20)
	require.NoError(testSuite.T(), err)
	require.Equal(testSuite.T(), int64(20), bwhImpl.truncatedSize)
	require.Equal(testSuite.T(), int64(20), testSuite.bwh.WriteFileInfo().TotalSize)

	// Write at offset=bwh.totalSize
	err = testSuite.bwh.Write(context.Background(), []byte("abcde"), 5)

	require.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), int64(10), bwhImpl.totalSize)
	assert.Equal(testSuite.T(), int64(20), testSuite.bwh.WriteFileInfo().TotalSize)
}

func (testSuite *BufferedWriteTest) TestOutOfOrderWriteAtStaleTruncatedSize() {
	err := testSuite.bwh.Write(context.Background(), []byte("hello"), 0)
	require.Nil(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	// Truncate to a larger size
	err = testSuite.bwh.Truncate(10)
	require.NoError(testSuite.T(), err)
	// Write past the truncated size (from offset 5, writing 10 bytes -> totalSize = 15)
	err = testSuite.bwh.Write(context.Background(), []byte("0123456789"), 5)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), int64(-1), bwhImpl.truncatedSize)
	require.Equal(testSuite.T(), int64(15), bwhImpl.totalSize)

	// Attempt to seek backwards and write exactly at the stale truncatedSize (10)
	err = testSuite.bwh.Write(context.Background(), []byte("abc"), 10)

	require.Error(testSuite.T(), err)
	assert.Equal(testSuite.T(), ErrOutOfOrderWrite, err)
}

func (testSuite *BufferedWriteTest) TestFlushWithNonNilCurrentBlock() {
	err := testSuite.bwh.Write(context.Background(), []byte("hi"), 0)
	require.Nil(testSuite.T(), err)

	obj, err := testSuite.bwh.Flush(context.Background())

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

	obj, err := testSuite.bwh.Flush(context.Background())

	assert.NoError(testSuite.T(), err)
	// Validate empty object created.
	assert.NotNil(testSuite.T(), obj)
	assert.Equal(testSuite.T(), uint64(0), obj.Size)
}

func (testSuite *BufferedWriteTest) TestFlushWithSignalUploadFailureDuringWrite() {
	err := testSuite.bwh.Write(context.Background(), []byte("hi"), 0)
	require.Nil(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)

	// Set an error to simulate failure in uploader.
	bwhImpl.uploadHandler.uploadError.Store(&errUploadFailure)

	obj, err := testSuite.bwh.Flush(context.Background())
	require.Error(testSuite.T(), err)
	assert.Equal(testSuite.T(), err, errUploadFailure)
	assert.Nil(testSuite.T(), obj)
}

func (testSuite *BufferedWriteTest) TestFlush_SizeMismatch_ReturnsError() {
	testCases := []struct {
		name       string
		bucketType gcs.BucketType
		obj        *gcs.Object
	}{
		{
			name:       "non_zonal",
			bucketType: gcs.BucketType{Zonal: false},
		},
		{
			name:       "zonal_new_file",
			bucketType: gcs.BucketType{Zonal: true},
		},
		{
			name:       "zonal_append",
			bucketType: gcs.BucketType{Zonal: true},
			obj:        &gcs.Object{Name: "testObject", Size: 0},
		},
		{
			name:       "pirlo_new_file_rapid_writes",
			bucketType: gcs.BucketType{Pirlo: gcs.PirloStateRapidWritesEnabled},
		},
		{
			name:       "pirlo_append_rapid_writes",
			bucketType: gcs.BucketType{Pirlo: gcs.PirloStateRapidWritesEnabled},
			obj:        &gcs.Object{Name: "testObject", Size: 0},
		},
	}
	for _, tc := range testCases {
		testSuite.Run(tc.name, func() {
			mockBucket := new(storagemock.TestifyMockBucket)
			mockBucket.On("BucketType").Return(tc.bucketType)
			writer := &storagemock.Writer{}
			writer.On("Write", mock.Anything).Return(2, nil)
			if tc.bucketType.RapidWritesEnabled() && tc.obj != nil {
				mockBucket.On("CreateAppendableObjectWriter", mock.Anything, mock.Anything).Return(writer, nil)
			} else {
				mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
			}
			mockObj := &gcs.MinObject{Name: "testObject", Size: 0}
			mockBucket.On("FinalizeUpload", mock.Anything, writer).Return(mockObj, nil)
			bwh, err := NewBWHandler(&CreateBWHandlerRequest{
				Object:                   tc.obj,
				ObjectName:               "testObject",
				Bucket:                   mockBucket,
				BlockSize:                blockSize,
				MaxBlocksPerFile:         10,
				GlobalMaxBlocksSem:       testSuite.globalSemaphore,
				ChunkRetryDeadlineSecs:   chunkRetryDeadlineSecs,
				ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
				TraceHandle:              tracing.NewNoopTracer(),
			})
			require.Nil(testSuite.T(), err)
			err = bwh.Write(context.Background(), []byte("hi"), 0)
			require.Nil(testSuite.T(), err)

			obj, err := bwh.Flush(context.Background())

			require.Error(testSuite.T(), err)
			assert.Contains(testSuite.T(), err.Error(), "could not upload entire data, expected size 2, got 0")
			assert.Nil(testSuite.T(), obj)
		})
	}
}

func (testSuite *BufferedWriteTest) TestSync_SizeMismatch_ReturnsError() {
	testCases := []struct {
		name       string
		bucketType gcs.BucketType
		obj        *gcs.Object
	}{
		{
			name:       "zonal_new_file",
			bucketType: gcs.BucketType{Zonal: true},
		},
		{
			name:       "zonal_append",
			bucketType: gcs.BucketType{Zonal: true},
			obj:        &gcs.Object{Name: "testObject", Size: 0},
		},
		{
			name:       "pirlo_new_file_rapid_writes",
			bucketType: gcs.BucketType{Pirlo: gcs.PirloStateRapidWritesEnabled},
		},
		{
			name:       "pirlo_append_rapid_writes",
			bucketType: gcs.BucketType{Pirlo: gcs.PirloStateRapidWritesEnabled},
			obj:        &gcs.Object{Name: "testObject", Size: 0},
		},
	}
	for _, tc := range testCases {
		testSuite.Run(tc.name, func() {
			mockBucket := new(storagemock.TestifyMockBucket)
			mockBucket.On("BucketType").Return(tc.bucketType)
			writer := &storagemock.Writer{}
			writer.On("Write", mock.Anything).Return(2, nil)
			if tc.bucketType.RapidWritesEnabled() && tc.obj != nil {
				mockBucket.On("CreateAppendableObjectWriter", mock.Anything, mock.Anything).Return(writer, nil)
			} else {
				mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
			}
			mockObj := &gcs.MinObject{Name: "testObject", Size: 0}
			mockBucket.On("FlushPendingWrites", mock.Anything, writer).Return(mockObj, nil)
			bwh, err := NewBWHandler(&CreateBWHandlerRequest{
				Object:                   tc.obj,
				ObjectName:               "testObject",
				Bucket:                   mockBucket,
				BlockSize:                blockSize,
				MaxBlocksPerFile:         10,
				GlobalMaxBlocksSem:       testSuite.globalSemaphore,
				ChunkRetryDeadlineSecs:   chunkRetryDeadlineSecs,
				ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
				TraceHandle:              tracing.NewNoopTracer(),
			})
			require.Nil(testSuite.T(), err)
			err = bwh.Write(context.Background(), []byte("hi"), 0)
			require.Nil(testSuite.T(), err)

			obj, err := bwh.Sync(context.Background())

			require.Error(testSuite.T(), err)
			assert.Contains(testSuite.T(), err.Error(), "could not upload entire data, expected size 2, got 0")
			assert.Nil(testSuite.T(), obj)
		})
	}
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
		err := testSuite.bwh.Write(context.Background(), buffer, int64(blockSize*(i+5)))
		require.Error(testSuite.T(), err)
		assert.Equal(testSuite.T(), errUploadFailure, err)
	}

	obj, err := testSuite.bwh.Flush(context.Background())

	require.Error(testSuite.T(), err)
	assert.Equal(testSuite.T(), err, errUploadFailure)
	assert.Nil(testSuite.T(), obj)
}

func (testSuite *BufferedWriteTest) TestSync5InProgressBlocks() {
	buffer, err := operations.GenerateRandomData(blockSize)
	assert.NoError(testSuite.T(), err)
	// Write 5 blocks.
	for i := range 5 {
		err = testSuite.bwh.Write(context.Background(), buffer, int64(blockSize*i))
		require.Nil(testSuite.T(), err)
	}

	// Wait for 5 blocks to upload successfully.
	o, err := testSuite.bwh.Sync(context.Background())

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
		{
			name:       "pirlo_bucket_rapid_writes_2.5_blocks",
			bucketType: gcs.BucketType{Pirlo: gcs.PirloStateRapidWritesEnabled},
			numBlocks:  2.5,
		},
		{
			name:       "pirlo_bucket_rapid_writes_.5_blocks",
			bucketType: gcs.BucketType{Pirlo: gcs.PirloStateRapidWritesEnabled},
			numBlocks:  .5,
		},
	}

	for _, tc := range testCases {
		testSuite.Run(tc.name, func() {
			testSuite.setupTestWithBucketType(tc.bucketType)
			buffer, err := operations.GenerateRandomData(int64(blockSize * tc.numBlocks))
			assert.NoError(testSuite.T(), err)
			err = testSuite.bwh.Write(context.Background(), buffer, 0)
			require.Nil(testSuite.T(), err)

			// Wait for blocks to upload successfully.
			o, err := testSuite.bwh.Sync(context.Background())

			require.NoError(testSuite.T(), err)
			bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
			// Current block should also be uploaded.
			assert.Nil(testSuite.T(), bwhImpl.current)
			assert.Equal(testSuite.T(), 0, len(bwhImpl.uploadHandler.uploadCh))
			assert.Equal(testSuite.T(), 0, bwhImpl.uploadHandler.blockPool.TotalFreeBlocks())
			// Read the object from back door.
			content, err := storageutil.ReadObject(context.Background(), bwhImpl.uploadHandler.bucket, bwhImpl.uploadHandler.objectName)
			if tc.bucketType.RapidWritesEnabled() {
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
		err = testSuite.bwh.Write(context.Background(), buffer, int64(blockSize*i))
		require.Nil(testSuite.T(), err)
	}
	// Set an error to simulate failure in uploader.
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	bwhImpl.uploadHandler.uploadError.Store(&errUploadFailure)

	o, err := testSuite.bwh.Sync(context.Background())

	assert.Error(testSuite.T(), err)
	assert.Equal(testSuite.T(), errUploadFailure, err)
	assert.Nil(testSuite.T(), o)
}

func (testSuite *BufferedWriteTest) TestFlushWithNonZeroTruncatedLengthForEmptyObject() {
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	require.Nil(testSuite.T(), bwhImpl.current)
	bwhImpl.truncatedSize = 10

	_, err := testSuite.bwh.Flush(context.Background())

	assert.NoError(testSuite.T(), err)
	assert.Equal(testSuite.T(), int64(10), bwhImpl.totalSize)
	assert.Equal(testSuite.T(), int64(-1), bwhImpl.truncatedSize)
}

func (testSuite *BufferedWriteTest) TestFlushWithTruncatedLengthGreaterThanObjectSize() {
	err := testSuite.bwh.Write(context.Background(), []byte("hi"), 0)
	require.Nil(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	bwhImpl.truncatedSize = 10

	_, err = testSuite.bwh.Flush(context.Background())

	assert.NoError(testSuite.T(), err)
	assert.Equal(testSuite.T(), int64(10), bwhImpl.totalSize)
	assert.Equal(testSuite.T(), int64(-1), bwhImpl.truncatedSize)
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
	err := testSuite.bwh.Write(context.Background(), []byte(contents), 0)
	require.Nil(testSuite.T(), err)

	err = testSuite.bwh.Destroy()

	require.Nil(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Nil(testSuite.T(), bwhImpl.current)
	assert.Equal(testSuite.T(), 0, bwhImpl.uploadHandler.blockPool.TotalFreeBlocks())
	assert.Equal(testSuite.T(), 0, len(bwhImpl.uploadHandler.uploadCh))
	// Check if all semaphore permits are released correctly.
	assert.True(testSuite.T(), testSuite.globalSemaphore.TryAcquire(10))
}

func (testSuite *BufferedWriteTest) TestDestroyWithPartialBlockInCurrent() {
	// Write a partial block (less than blockSize) so it stays in wh.current and is not uploaded.
	partialData := strings.Repeat("B", blockSize/2)
	err := testSuite.bwh.Write(context.Background(), []byte(partialData), 0)
	require.Nil(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	require.NotNil(testSuite.T(), bwhImpl.current)
	assert.Equal(testSuite.T(), int64(blockSize/2), bwhImpl.current.Size())

	err = testSuite.bwh.Destroy()

	require.Nil(testSuite.T(), err)
	assert.Nil(testSuite.T(), bwhImpl.current)
	assert.Equal(testSuite.T(), 0, bwhImpl.uploadHandler.blockPool.TotalFreeBlocks())
	assert.Equal(testSuite.T(), 0, len(bwhImpl.uploadHandler.uploadCh))
	// Verify that all 10 permits (including the one used by wh.current) are released.
	assert.True(testSuite.T(), testSuite.globalSemaphore.TryAcquire(10))
}

func (testSuite *BufferedWriteTest) TestDestroyWithMultiBlocksAndPartialBlockInCurrent() {
	// Write 2.5 blocks of data: 2 full blocks sent to uploadCh, 0.5 block stays in wh.current.
	data := strings.Repeat("C", int(blockSize*2.5))
	err := testSuite.bwh.Write(context.Background(), []byte(data), 0)
	require.Nil(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	require.NotNil(testSuite.T(), bwhImpl.current)
	assert.Equal(testSuite.T(), int64(blockSize/2), bwhImpl.current.Size())

	err = testSuite.bwh.Destroy()

	require.Nil(testSuite.T(), err)
	assert.Nil(testSuite.T(), bwhImpl.current)
	assert.Equal(testSuite.T(), 0, bwhImpl.uploadHandler.blockPool.TotalFreeBlocks())
	assert.Equal(testSuite.T(), 0, len(bwhImpl.uploadHandler.uploadCh))
	// Verify that all 10 permits are released back to global semaphore.
	assert.True(testSuite.T(), testSuite.globalSemaphore.TryAcquire(10))
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
		err = testSuite.bwh.Write(context.Background(), buffer, int64(blockSize*i))
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
	obj, err := testSuite.bwh.Flush(context.Background())

	require.Error(testSuite.T(), err)
	assert.Nil(testSuite.T(), obj)
	assert.ErrorContains(testSuite.T(), err, errUploadFailure.Error())
}

func (testSuite *BufferedWriteTest) TestBufferedWriteMetrics() {
	ctx := context.Background()
	origProvider := otel.GetMeterProvider()
	defer otel.SetMeterProvider(origProvider)

	reader := metricSdk.NewManualReader()
	provider := metricSdk.NewMeterProvider(metricSdk.WithReader(reader))
	otel.SetMeterProvider(provider)

	mh, err := metrics.NewOTelMetrics(ctx, 1, 100)
	require.NoError(testSuite.T(), err)

	bucket := fake.NewFakeBucket(timeutil.RealClock(), "FakeBucketName", gcs.BucketType{})
	sem := semaphore.NewWeighted(10)
	bwh, err := NewBWHandler(&CreateBWHandlerRequest{
		Object:                   nil,
		ObjectName:               "testMetricsObject",
		Bucket:                   bucket,
		BlockSize:                blockSize,
		MaxBlocksPerFile:         10,
		GlobalMaxBlocksSem:       sem,
		ChunkRetryDeadlineSecs:   chunkRetryDeadlineSecs,
		ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
		TraceHandle:              tracing.NewNoopTracer(),
		MetricHandle:             mh,
	})
	require.NoError(testSuite.T(), err)

	buffer, err := operations.GenerateRandomData(blockSize)
	require.NoError(testSuite.T(), err)

	// Perform 2 sequential writes with simulated app delay.
	err = bwh.Write(ctx, buffer, 0)
	require.NoError(testSuite.T(), err)

	time.Sleep(10 * time.Millisecond)

	err = bwh.Write(ctx, buffer, int64(blockSize))
	require.NoError(testSuite.T(), err)

	// Flush
	obj, err := bwh.Flush(ctx)
	require.NoError(testSuite.T(), err)
	require.NotNil(testSuite.T(), obj)

	time.Sleep(5 * time.Millisecond)

	// Verify metrics recorded
	metrics.VerifyHistogramMetric(testSuite.T(), ctx, reader, "buffered_write/total_latency", attribute.NewSet(attribute.String("bottleneck", "app_bound")), 1)
	metrics.VerifyHistogramMetric(testSuite.T(), ctx, reader, "buffered_write/app_wait_latency", attribute.NewSet(), 1)
	metrics.VerifyHistogramMetric(testSuite.T(), ctx, reader, "buffered_write/block_pool_wait_latency", attribute.NewSet(), 1)
	metrics.VerifyHistogramMetric(testSuite.T(), ctx, reader, "buffered_write/finalize_latency", attribute.NewSet(), 1)
}

func (testSuite *BufferedWriteTest) TestBufferedWriteMetrics_NoMetricsOnOutOfOrderFallback() {
	ctx := context.Background()
	origProvider := otel.GetMeterProvider()
	defer otel.SetMeterProvider(origProvider)

	reader := metricSdk.NewManualReader()
	provider := metricSdk.NewMeterProvider(metricSdk.WithReader(reader))
	otel.SetMeterProvider(provider)

	mh, err := metrics.NewOTelMetrics(ctx, 1, 100)
	require.NoError(testSuite.T(), err)

	bucket := fake.NewFakeBucket(timeutil.RealClock(), "FakeBucketName", gcs.BucketType{})
	sem := semaphore.NewWeighted(10)
	bwh, err := NewBWHandler(&CreateBWHandlerRequest{
		Object:                   nil,
		ObjectName:               "testFallbackMetricsObject",
		Bucket:                   bucket,
		BlockSize:                blockSize,
		MaxBlocksPerFile:         10,
		GlobalMaxBlocksSem:       sem,
		ChunkRetryDeadlineSecs:   chunkRetryDeadlineSecs,
		ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
		TraceHandle:              tracing.NewNoopTracer(),
		MetricHandle:             mh,
	})
	require.NoError(testSuite.T(), err)

	buffer, err := operations.GenerateRandomData(blockSize)
	require.NoError(testSuite.T(), err)

	// Write first block at offset 0
	err = bwh.Write(ctx, buffer, 0)
	require.NoError(testSuite.T(), err)

	// Out of order write (e.g. offset blockSize*3 instead of blockSize)
	err = bwh.Write(ctx, buffer, int64(blockSize*3))
	require.ErrorIs(testSuite.T(), err, ErrOutOfOrderWrite)

	// Flush is called by FileInode to finalize what was written so far before dropping to temp file.
	obj, err := bwh.Flush(ctx)
	require.NoError(testSuite.T(), err)
	require.NotNil(testSuite.T(), obj)

	time.Sleep(5 * time.Millisecond)

	// Verify that NO buffered_write metrics are recorded because this was a fallback.
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(testSuite.T(), err)
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if strings.HasPrefix(m.Name, "buffered_write/") {
				if hist, ok := m.Data.(metricdata.Histogram[int64]); ok {
					assert.Equal(testSuite.T(), 0, len(hist.DataPoints), "unexpected data points recorded for %s", m.Name)
				}
			}
		}
	}
}
