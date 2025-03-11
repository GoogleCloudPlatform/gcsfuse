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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

const chunkTransferTimeoutSecs int64 = 10

var errUploadFailure = errors.New("error while uploading object to GCS")

type BufferedWriteTest struct {
	bwh BufferedWriteHandler
	suite.Suite
}

func TestBufferedWriteTestSuite(t *testing.T) {
	suite.Run(t, new(BufferedWriteTest))
}

func (testSuite *BufferedWriteTest) SetupTest() {
	bucket := fake.NewFakeBucket(timeutil.RealClock(), "FakeBucketName", gcs.BucketType{})
	bwh, err := NewBWHandler(&CreateBWHandlerRequest{
		Object:                   nil,
		ObjectName:               "testObject",
		Bucket:                   bucket,
		BlockSize:                blockSize,
		MaxBlocksPerFile:         10,
		GlobalMaxBlocksSem:       semaphore.NewWeighted(10),
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
	bwhImpl.uploadHandler.uploadError = errUploadFailure

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
	assert.Equal(testSuite.T(), 0, len(bwhImpl.blockPool.FreeBlocksChannel()))
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
	bwhImpl.uploadHandler.uploadError = errUploadFailure

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
	bwhImpl.uploadHandler.uploadError = errUploadFailure
	// Write 5 more blocks.
	for i := 0; i < 5; i++ {
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
	for i := 0; i < 5; i++ {
		err = testSuite.bwh.Write(buffer, int64(blockSize*i))
		require.Nil(testSuite.T(), err)
	}

	// Wait for 5 blocks to upload successfully.
	err = testSuite.bwh.Sync()

	assert.NoError(testSuite.T(), err)
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Equal(testSuite.T(), 0, len(bwhImpl.uploadHandler.uploadCh))
	assert.Equal(testSuite.T(), 0, len(bwhImpl.blockPool.FreeBlocksChannel()))
}

func (testSuite *BufferedWriteTest) TestSyncBlocksWithError() {
	buffer, err := operations.GenerateRandomData(blockSize)
	assert.NoError(testSuite.T(), err)
	// Write 5 blocks.
	for i := 0; i < 5; i++ {
		err = testSuite.bwh.Write(buffer, int64(blockSize*i))
		require.Nil(testSuite.T(), err)
	}
	// Set an error to simulate failure in uploader.
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	bwhImpl.uploadHandler.uploadError = errUploadFailure

	err = testSuite.bwh.Sync()

	assert.Error(testSuite.T(), err)
	assert.Equal(testSuite.T(), errUploadFailure, err)
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
	assert.Equal(testSuite.T(), 0, len(bwhImpl.blockPool.FreeBlocksChannel()))
	assert.Equal(testSuite.T(), 0, len(bwhImpl.uploadHandler.uploadCh))
}

func (testSuite *BufferedWriteTest) TestUnlinkBeforeWrite() {
	testSuite.bwh.Unlink()

	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	assert.Nil(testSuite.T(), bwhImpl.uploadHandler.cancelFunc)
	assert.Equal(testSuite.T(), 0, len(bwhImpl.uploadHandler.uploadCh))
	assert.Equal(testSuite.T(), 0, len(bwhImpl.blockPool.FreeBlocksChannel()))
}

func (testSuite *BufferedWriteTest) TestUnlinkAfterWrite() {
	buffer, err := operations.GenerateRandomData(blockSize)
	assert.NoError(testSuite.T(), err)
	// Write 5 blocks.
	for i := 0; i < 5; i++ {
		err = testSuite.bwh.Write(buffer, int64(blockSize*i))
		require.Nil(testSuite.T(), err)
	}
	cancelCalled := false
	bwhImpl := testSuite.bwh.(*bufferedWriteHandlerImpl)
	bwhImpl.uploadHandler.cancelFunc = func() { cancelCalled = true }

	testSuite.bwh.Unlink()

	assert.True(testSuite.T(), cancelCalled)
	assert.Equal(testSuite.T(), 0, len(bwhImpl.uploadHandler.uploadCh))
	assert.Equal(testSuite.T(), 0, len(bwhImpl.blockPool.FreeBlocksChannel()))
}

func (testSuite *BufferedWriteTest) TestReFlushAfterUploadFails() {
	testSuite.TestFlushWithMultiBlockWritesAndSignalUploadFailureInBetween()

	// Re-flush.
	obj, err := testSuite.bwh.Flush()

	require.Error(testSuite.T(), err)
	assert.Nil(testSuite.T(), obj)
	assert.ErrorContains(testSuite.T(), err, errUploadFailure.Error())
}
