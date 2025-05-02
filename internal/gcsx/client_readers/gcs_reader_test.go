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
	"io"
	"strings"
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

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (t *gcsReaderTest) readAt(offset int64, size int64) (gcsx.ReaderResponse, error) {
	t.gcsReader.CheckInvariants()
	defer t.gcsReader.CheckInvariants()
	return t.gcsReader.ReadAt(t.ctx, make([]byte, size), offset)
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type gcsReaderTest struct {
	suite.Suite
	ctx        context.Context
	object     *gcs.MinObject
	mockBucket *storage.TestifyMockBucket
	gcsReader  *GCSReader
}

func TestGCSReaderTestSuite(t *testing.T) {
	suite.Run(t, new(gcsReaderTest))
}

func (t *gcsReaderTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       testObject,
		Size:       17,
		Generation: 1234,
	}
	t.mockBucket = new(storage.TestifyMockBucket)
	t.gcsReader = NewGCSReader(t.object, t.mockBucket, common.NewNoopMetrics(), nil, 200)
	t.ctx = context.Background()
}

func (t *gcsReaderTest) TearDownTest() {
	t.gcsReader.Destroy()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *gcsReaderTest) Test_NewGCSReader() {
	// The setup instantiates gcsReader with NewGCSReader.
	assert.Equal(t.T(), t.object, t.gcsReader.object)
	assert.Equal(t.T(), t.mockBucket, t.gcsReader.bucket)
	assert.Equal(t.T(), testUtil.Sequential, t.gcsReader.readType)
}

func (t *gcsReaderTest) Test_ReadAt_ExistingReaderLimitIsLessThanRequestedDataSize() {
	t.object.Size = 10
	// Simulate an existing reader.
	t.gcsReader.rangeReader.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte("xxx")), Handle: []byte("fake")}
	t.gcsReader.rangeReader.cancel = func() {}
	t.gcsReader.rangeReader.start = 2
	t.gcsReader.rangeReader.limit = 5
	content := "verify"
	rc := &fake.FakeReader{ReadCloser: getReadCloser([]byte(content))}
	expectedHandleInRequest := []byte(t.gcsReader.rangeReader.reader.ReadHandle())
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       t.gcsReader.rangeReader.object.Name,
		Generation: t.gcsReader.rangeReader.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(2),
			Limit: t.object.Size,
		},
		ReadCompressed: t.gcsReader.rangeReader.object.HasContentEncodingGzip(),
		ReadHandle:     expectedHandleInRequest,
	}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(rc, nil)
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(1)
	requestSize := 6

	readerResponse, err := t.readAt(2, int64(requestSize))

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), rc, t.gcsReader.rangeReader.reader)
	assert.Equal(t.T(), requestSize, readerResponse.Size)
	assert.Equal(t.T(), content, string(readerResponse.DataBuf[:readerResponse.Size]))
	assert.Equal(t.T(), uint64(requestSize), t.gcsReader.totalReadBytes)
	assert.Equal(t.T(), expectedHandleInRequest, t.gcsReader.rangeReader.readHandle)
}

func (t *gcsReaderTest) Test_ReadAt_ExistingReaderLimitIsLessThanRequestedObjectSize() {
	t.object.Size = 5
	// Simulate an existing reader
	t.gcsReader.rangeReader.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte("xxx")), Handle: []byte("fake")}
	t.gcsReader.rangeReader.cancel = func() {}
	t.gcsReader.rangeReader.start = 0
	t.gcsReader.rangeReader.limit = 3
	content := "abcde"
	rc := &fake.FakeReader{ReadCloser: getReadCloser([]byte(content))}
	expectedHandleInRequest := t.gcsReader.rangeReader.reader.ReadHandle()
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       t.gcsReader.rangeReader.object.Name,
		Generation: t.gcsReader.rangeReader.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(0),
			Limit: t.object.Size,
		},
		ReadCompressed: t.gcsReader.rangeReader.object.HasContentEncodingGzip(),
		ReadHandle:     expectedHandleInRequest,
	}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(rc, nil)
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(1)
	requestSize := 6

	readerResponse, err := t.readAt(0, int64(requestSize))

	assert.NoError(t.T(), err)
	assert.Nil(t.T(), t.gcsReader.rangeReader.reader)
	assert.Equal(t.T(), int(t.object.Size), readerResponse.Size)
	assert.Equal(t.T(), content, string(readerResponse.DataBuf[:readerResponse.Size]))
	assert.Equal(t.T(), []byte(nil), t.gcsReader.rangeReader.readHandle)
}

func (t *gcsReaderTest) Test_ExistingReader_WrongOffset() {
	testCases := []struct {
		name       string
		readHandle []byte
	}{
		{
			name:       "ReaderHasReadHandle",
			readHandle: []byte("fake-handle"),
		},
		{
			name:       "ReaderHasNoReadHandle",
			readHandle: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.object.Size = 5
			// Simulate an existing reader.
			t.gcsReader.rangeReader.readHandle = tc.readHandle
			t.gcsReader.rangeReader.reader = &fake.FakeReader{
				ReadCloser: io.NopCloser(strings.NewReader("xxx")),
				Handle:     tc.readHandle,
			}
			t.gcsReader.rangeReader.cancel = func() {}
			t.gcsReader.rangeReader.start = 2
			t.gcsReader.rangeReader.limit = 5
			content := "abcde"
			rc := &fake.FakeReader{ReadCloser: getReadCloser([]byte(content))}
			t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(rc, nil).Times(1)
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(1)
			requestSize := 6

			readerResponse, err := t.readAt(0, int64(requestSize))

			t.mockBucket.AssertExpectations(t.T())
			assert.NoError(t.T(), err)
			assert.Nil(t.T(), t.gcsReader.rangeReader.reader)
			assert.Equal(t.T(), int(t.object.Size), readerResponse.Size)
			assert.Equal(t.T(), content, string(readerResponse.DataBuf[:readerResponse.Size]))
			assert.Equal(t.T(), []byte(nil), t.gcsReader.rangeReader.readHandle)
		})
	}
}

func (t *gcsReaderTest) Test_ReadAt_ValidateReadType() {
	testCases := []struct {
		name              string
		dataSize          int
		bucketType        gcs.BucketType
		readRanges        [][]int
		expectedReadTypes []string
	}{
		{
			name:              "SequentialReadFlat",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: false},
			readRanges:        [][]int{{0, 10}, {10, 20}, {20, 35}, {35, 50}},
			expectedReadTypes: []string{testUtil.Sequential, testUtil.Sequential, testUtil.Sequential, testUtil.Sequential},
		},
		{
			name:              "SequentialReadZonal",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: true},
			readRanges:        [][]int{{0, 10}, {10, 20}, {20, 35}, {35, 50}},
			expectedReadTypes: []string{testUtil.Sequential, testUtil.Sequential, testUtil.Sequential, testUtil.Sequential},
		},
		{
			name:              "RandomReadFlat",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: false},
			readRanges:        [][]int{{0, 50}, {30, 40}, {10, 20}, {20, 30}, {30, 40}},
			expectedReadTypes: []string{testUtil.Sequential, testUtil.Sequential, testUtil.Random, testUtil.Random, testUtil.Random},
		},
		{
			name:              "RandomReadZonal",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: true},
			readRanges:        [][]int{{0, 50}, {30, 40}, {10, 20}, {20, 30}, {30, 40}},
			expectedReadTypes: []string{testUtil.Sequential, testUtil.Sequential, testUtil.Random, testUtil.Random, testUtil.Random},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.SetupTest()
			require.Equal(t.T(), len(tc.readRanges), len(tc.expectedReadTypes), "Test Parameter Error: readRanges and expectedReadTypes should have same length")
			t.gcsReader.mrr.isMRDInUse = false
			t.gcsReader.seeks = 0
			t.gcsReader.rangeReader.readType = testUtil.Sequential
			t.object.Size = uint64(tc.dataSize)
			testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
			fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{})
			require.NoError(t.T(), err, "Error in creating MRDWrapper")
			t.gcsReader.mrr.mrdWrapper = &fakeMRDWrapper
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond))
			t.mockBucket.On("BucketType", mock.Anything).Return(tc.bucketType).Times(len(tc.readRanges))

			for i, readRange := range tc.readRanges {
				t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(&fake.FakeReader{ReadCloser: getReadCloser(testContent)}, nil).Once()

				_, err = t.readAt(int64(readRange[0]), int64(readRange[1]-readRange[0]))

				assert.NoError(t.T(), err)
				assert.Equal(t.T(), tc.expectedReadTypes[i], t.gcsReader.readType)
			}
		})
	}
}
