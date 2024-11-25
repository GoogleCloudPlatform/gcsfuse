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
	bwh, err := NewBWHandler("testObject", bucket, 1024, 10, semaphore.NewWeighted(10))
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

func (testSuite *BufferedWriteTest) TestFlushWithNonNilCurrentBlock() {
	err := testSuite.bwh.Write([]byte("hi"), 0)
	currentBlock := testSuite.bwh.current
	require.Nil(testSuite.T(), err)

	err = testSuite.bwh.Flush()

	require.NoError(testSuite.T(), err)
	assert.Equal(testSuite.T(), nil, testSuite.bwh.current)
	// The current block should be available on the free channel as flush triggers
	// an upload before finalize.
	freeCh := testSuite.bwh.blockPool.FreeBlocksChannel()
	got := <-freeCh
	assert.Equal(testSuite.T(), &currentBlock, &got)
}

func (testSuite *BufferedWriteTest) TestFlushWithNilCurrentBlock() {
	require.Nil(testSuite.T(), testSuite.bwh.current)

	err := testSuite.bwh.Flush()

	assert.NoError(testSuite.T(), err)
}

func TestBufferedWriteHandler_SignalUploadFailure(t *testing.T) {
	mockSignalUploadFailure := make(chan error)
	mockUploadHandler := &UploadHandler{
		signalUploadFailure: mockSignalUploadFailure,
	}
	wh := &BufferedWriteHandler{
		uploadHandler: mockUploadHandler,
	}

	actualChannel := wh.SignalUploadFailure()

	assert.Equal(t, mockSignalUploadFailure, actualChannel)
}
