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

package gcsx

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/clock"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	testUtil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	TestTimeoutForMultiRangeRead = time.Second
)

type multiRangeReaderTest struct {
	suite.Suite
	ctx              context.Context
	object           *gcs.MinObject
	mockBucket       *storage.TestifyMockBucket
	multiRangeReader *MultiRangeReader
}

func (t *multiRangeReaderTest) readAt(offset int64, size int64) (gcsx.ReaderResponse, error) {
	req := &gcsx.GCSReaderRequest{
		Offset:    offset,
		EndOffset: offset + size,
		Buffer:    make([]byte, size),
	}
	return t.multiRangeReader.ReadAt(t.ctx, req)
}

func TestMultiRangeReaderTestSuite(t *testing.T) {
	suite.Run(t, new(multiRangeReaderTest))
}

func (t *multiRangeReaderTest) SetupTest() {
	t.mockBucket = new(storage.TestifyMockBucket)
	t.ctx = context.Background()
	t.object = &gcs.MinObject{
		Name:       "testObject",
		Size:       17,
		Generation: 1234,
	}
	t.multiRangeReader = NewMultiRangeReader(t.object, common.NewNoopMetrics(), nil)
}

func (t *multiRangeReaderTest) TearDownTest() {
	t.multiRangeReader.destroy()
}

func (t *multiRangeReaderTest) Test_ReadFromMultiRangeReader_ReadFull() {
	testCases := []struct {
		name      string
		dataSize  int
		extraSize int
	}{
		{
			name:      "ReadFull",
			dataSize:  100,
			extraSize: 0,
		},
		{
			name:      "ReadWithLargerBuffer",
			dataSize:  100,
			extraSize: 10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.multiRangeReader.isMRDInUse = false
			t.object.Size = uint64(tc.dataSize)
			testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
			fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{})
			require.NoError(t.T(), err, "Error in creating MRDWrapper")
			t.multiRangeReader.mrdWrapper = &fakeMRDWrapper
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond)).Times(1)
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true}).Times(1)
			buf := make([]byte, tc.dataSize+tc.extraSize)

			bytesRead, err := t.multiRangeReader.readFromMultiRangeReader(t.ctx, buf, 0, int64(t.object.Size), TestTimeoutForMultiRangeRead)

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), tc.dataSize, bytesRead)
			assert.Equal(t.T(), testContent[:tc.dataSize], buf[:bytesRead])
		})
	}
}

func (t *multiRangeReaderTest) Test_ReadFromMultiRangeReader_TimeoutExceeded() {
	t.multiRangeReader.isMRDInUse = false
	dataSize := 100
	t.object.Size = uint64(dataSize)
	testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
	fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{})
	require.NoError(t.T(), err, "Error in creating MRDWrapper")
	t.multiRangeReader.mrdWrapper = &fakeMRDWrapper
	sleepTime := 10 * time.Millisecond
	timeout := 5 * time.Millisecond
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, sleepTime)).Once()
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true}).Once()
	buf := make([]byte, dataSize)

	_, err = t.multiRangeReader.readFromMultiRangeReader(t.ctx, buf, 0, int64(t.object.Size), timeout)

	assert.Error(t.T(), err)
	assert.ErrorContains(t.T(), err, "Timeout")
}

func (t *multiRangeReaderTest) Test_ReadFromMultiRangeReader_TimeoutNotExceeded() {
	t.multiRangeReader.isMRDInUse = false
	dataSize := 100
	t.object.Size = uint64(dataSize)
	testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
	fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{})
	require.NoError(t.T(), err, "Error in creating MRDWrapper")
	t.multiRangeReader.mrdWrapper = &fakeMRDWrapper
	sleepTime := 2 * time.Millisecond
	timeout := 5 * time.Millisecond
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, sleepTime)).Once()
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true}).Once()
	buf := make([]byte, dataSize)

	bytesRead, err := t.multiRangeReader.readFromMultiRangeReader(t.ctx, buf, 0, int64(t.object.Size), timeout)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), dataSize, bytesRead)
	assert.Equal(t.T(), testContent[:dataSize], buf[:bytesRead])
}

func (t *multiRangeReaderTest) Test_ReadFromMultiRangeReader_NilMRDWrapper() {
	t.multiRangeReader.mrdWrapper = nil

	bytesRead, err := t.multiRangeReader.readFromMultiRangeReader(t.ctx, make([]byte, t.object.Size), 0, int64(t.object.Size), TestTimeoutForMultiRangeRead)

	assert.Error(t.T(), err)
	assert.ErrorContains(t.T(), err, "readFromMultiRangeReader: Invalid MultiRangeDownloaderWrapper")
	assert.Equal(t.T(), 0, bytesRead)
}

func (t *multiRangeReaderTest) Test_ReadFromMultiRangeReader_ReadChunk() {
	testCases := []struct {
		name     string
		dataSize int
		start    int
		end      int
	}{
		{
			name:     "ReadChunk",
			dataSize: 100,
			start:    37,
			end:      93,
		},
	}

	for _, tc := range testCases {
		t.object.Size = uint64(tc.dataSize)
		testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
		fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{})
		require.NoError(t.T(), err, "Error in creating MRDWrapper")
		t.multiRangeReader.mrdWrapper = &fakeMRDWrapper
		t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond)).Times(1)
		t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true}).Times(1)
		buf := make([]byte, tc.end-tc.start)

		bytesRead, err := t.multiRangeReader.readFromMultiRangeReader(t.ctx, buf, int64(tc.start), int64(tc.end), TestTimeoutForMultiRangeRead)

		assert.NoError(t.T(), err)
		assert.Equal(t.T(), tc.end-tc.start, bytesRead)
		assert.Equal(t.T(), testContent[tc.start:tc.end], buf[:bytesRead])
	}
}

func (t *multiRangeReaderTest) Test_ReadAt_MRDRead() {
	testCases := []struct {
		name        string
		dataSize    int
		offset      int
		bytesToRead int
	}{
		{
			name:        "ReadChunk",
			dataSize:    100,
			offset:      37,
			bytesToRead: 43,
		},
		{
			name:        "ReadZeroByte",
			dataSize:    100,
			offset:      37,
			bytesToRead: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.multiRangeReader.isMRDInUse = false
			t.object.Size = uint64(tc.dataSize)
			testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
			fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{})
			require.NoError(t.T(), err, "Error in creating MRDWrapper")
			t.multiRangeReader.mrdWrapper = &fakeMRDWrapper
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond)).Times(1)
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true}).Times(1)

			readerResponse, err := t.readAt(int64(tc.offset), int64(tc.bytesToRead))

			t.mockBucket.AssertNotCalled(t.T(), "NewReaderWithReadHandle", mock.Anything)
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), tc.bytesToRead, readerResponse.Size)
			assert.Equal(t.T(), testContent[tc.offset:tc.offset+tc.bytesToRead], readerResponse.DataBuf[:readerResponse.Size])
		})
	}
}

func (t *multiRangeReaderTest) Test_ReadAt_InvalidOffset() {
	t.object.Size = 50

	_, err := t.readAt(65, int64(t.object.Size))

	assert.True(t.T(), errors.Is(err, io.EOF), "expected %v error got %v", io.EOF, err)
}
