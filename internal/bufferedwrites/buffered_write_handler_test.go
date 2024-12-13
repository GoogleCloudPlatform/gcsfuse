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

type BufferedWriteTest struct {
	bwh *BufferedWriteHandler
	suite.Suite
}

func TestBufferedWriteTestSuite(t *testing.T) {
	suite.Run(t, new(BufferedWriteTest))
}

func (testSuite *BufferedWriteTest) SetupTest() {
	bucket := fake.NewFakeBucket(timeutil.RealClock(), "FakeBucketName", gcs.NonHierarchical)
	bwh, err := NewBWHandler("testObject", bucket, blockSize, 10, semaphore.NewWeighted(10))
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
	assert.Equal(testSuite.T(), testSuite.bwh.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(2), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteWithEmptyBuffer() {
	err := testSuite.bwh.Write([]byte{}, 0)

	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	assert.Equal(testSuite.T(), testSuite.bwh.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(0), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteEqualToBlockSize() {
	size := 1024
	data := strings.Repeat("A", size)

	err := testSuite.bwh.Write([]byte(data), 0)

	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	assert.Equal(testSuite.T(), testSuite.bwh.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(size), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteDataSizeGreaterThanBlockSize() {
	size := 2000
	data := strings.Repeat("A", size)

	err := testSuite.bwh.Write([]byte(data), 0)

	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	assert.Equal(testSuite.T(), testSuite.bwh.mtime, fileInfo.Mtime)
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
	assert.Equal(testSuite.T(), testSuite.bwh.mtime, fileInfo.Mtime)
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
	assert.Equal(testSuite.T(), testSuite.bwh.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(5), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestMultipleWrites() {
	err := testSuite.bwh.Write([]byte("hello"), 0)
	require.Nil(testSuite.T(), err)

	err = testSuite.bwh.Write([]byte("abcdefgh"), 5)
	require.Nil(testSuite.T(), err)

	fileInfo := testSuite.bwh.WriteFileInfo()
	assert.Equal(testSuite.T(), testSuite.bwh.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(13), fileInfo.TotalSize)
}

func (testSuite *BufferedWriteTest) TestWriteWithSignalUploadFailureInBetween() {
	err := testSuite.bwh.Write([]byte("hello"), 0)
	require.Nil(testSuite.T(), err)
	fileInfo := testSuite.bwh.WriteFileInfo()
	assert.Equal(testSuite.T(), testSuite.bwh.mtime, fileInfo.Mtime)
	assert.Equal(testSuite.T(), int64(5), fileInfo.TotalSize)

	// Close the channel to simulate failure in uploader.
	close(testSuite.bwh.uploadHandler.SignalUploadFailure())

	err = testSuite.bwh.Write([]byte("hello"), 5)
	require.Error(testSuite.T(), err)
	assert.Equal(testSuite.T(), err, ErrUploadFailure)
}

func (testSuite *BufferedWriteTest) TestFlushWithNonNilCurrentBlock() {
	err := testSuite.bwh.Write([]byte("hi"), 0)
	require.Nil(testSuite.T(), err)

	obj, err := testSuite.bwh.Flush()

	require.NoError(testSuite.T(), err)
	assert.Equal(testSuite.T(), nil, testSuite.bwh.current)
	// Validate object.
	assert.NotNil(testSuite.T(), obj)
	assert.Equal(testSuite.T(), uint64(2), obj.Size)
	// Validate that all blocks have been freed up.
	assert.Equal(testSuite.T(), 0, len(testSuite.bwh.blockPool.FreeBlocksChannel()))
}

func (testSuite *BufferedWriteTest) TestFlushWithNilCurrentBlock() {
	require.Nil(testSuite.T(), testSuite.bwh.current)

	obj, err := testSuite.bwh.Flush()

	assert.NoError(testSuite.T(), err)
	// Validate empty object created.
	assert.NotNil(testSuite.T(), obj)
	assert.Equal(testSuite.T(), uint64(0), obj.Size)
}

func (testSuite *BufferedWriteTest) TestFlushWithSignalUploadFailureDuringWrite() {
	err := testSuite.bwh.Write([]byte("hi"), 0)
	require.Nil(testSuite.T(), err)

	// Close the channel to simulate failure in uploader.
	close(testSuite.bwh.uploadHandler.SignalUploadFailure())

	obj, err := testSuite.bwh.Flush()
	require.Error(testSuite.T(), err)
	assert.Equal(testSuite.T(), err, ErrUploadFailure)
	// Whatever could be finalized, got finalized (empty object in this case).
	assert.NotNil(testSuite.T(), obj)
	assert.Equal(testSuite.T(), uint64(0), obj.Size)
}

func (testSuite *BufferedWriteTest) TestFlushWithMultiBlockWritesAndSignalUploadFailureInBetween() {
	buffer, err := operations.GenerateRandomData(blockSize)
	assert.NoError(testSuite.T(), err)
	// Upload and sync 5 blocks.
	testSuite.TestSync5InProgressBlocks()
	// Close the channel to simulate failure in uploader.
	close(testSuite.bwh.uploadHandler.SignalUploadFailure())
	// Write 5 more blocks.
	for i := 0; i < 5; i++ {
		err := testSuite.bwh.Write(buffer, int64(blockSize*(i+5)))
		require.Error(testSuite.T(), err)
	}

	obj, err := testSuite.bwh.Flush()

	require.Error(testSuite.T(), err)
	assert.Equal(testSuite.T(), err, ErrUploadFailure)
	// Whatever could be finalized, got finalized.
	assert.NotNil(testSuite.T(), obj)
	assert.Equal(testSuite.T(), uint64(5*blockSize), obj.Size)
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
}

func (testSuite *BufferedWriteTest) TestSyncBlocksWithError() {
	buffer, err := operations.GenerateRandomData(blockSize)
	assert.NoError(testSuite.T(), err)
	// Write 5 blocks.
	for i := 0; i < 5; i++ {
		err = testSuite.bwh.Write(buffer, int64(blockSize*i))
		require.Nil(testSuite.T(), err)
	}
	// Close the channel to simulate failure in uploader.
	close(testSuite.bwh.uploadHandler.SignalUploadFailure())

	err = testSuite.bwh.Sync()

	assert.Error(testSuite.T(), err)
	assert.Equal(testSuite.T(), ErrUploadFailure, err)
}
