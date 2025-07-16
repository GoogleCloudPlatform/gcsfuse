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
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/clock"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	testutil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const TestTimeoutForMultiRangeRead = time.Second

type RandomReaderStretchrTest struct {
	suite.Suite
	object       *gcs.MinObject
	mockBucket   *storage.TestifyMockBucket
	rr           checkingRandomReader
	cacheDir     string
	jobManager   *downloader.JobManager
	cacheHandler *file.CacheHandler
}

func TestRandomReaderStretchrTestSuite(t *testing.T) {
	suite.Run(t, new(RandomReaderStretchrTest))
}

func (t *RandomReaderStretchrTest) SetupTest() {
	t.rr.ctx = context.Background()

	// Manufacture an object record.
	t.object = &gcs.MinObject{
		Name:       "foo",
		Size:       17,
		Generation: 1234,
	}

	// Create the bucket.
	t.mockBucket = new(storage.TestifyMockBucket)

	t.cacheDir = path.Join(os.Getenv("HOME"), "cache/dir")
	lruCache := lru.NewCache(cacheMaxSize)
	t.jobManager = downloader.NewJobManager(lruCache, util.DefaultFilePerm, util.DefaultDirPerm, t.cacheDir, sequentialReadSizeInMb, &cfg.FileCacheConfig{
		EnableCrc: false,
	}, nil)
	t.cacheHandler = file.NewCacheHandler(lruCache, t.jobManager, t.cacheDir, util.DefaultFilePerm, util.DefaultDirPerm, "")

	// Set up the reader.
	rr := NewRandomReader(t.object, t.mockBucket, sequentialReadSizeInMb, nil, false, common.NewNoopMetrics(), nil, nil)
	t.rr.wrapped = rr.(*randomReader)
}

func (t *RandomReaderStretchrTest) TearDownTest() {
	t.rr.Destroy()
}

func (t *RandomReaderStretchrTest) Test_ReadInfo() {
	t.object.Size = 10 * MiB
	testCases := []struct {
		name  string
		start int64
		size  int64
	}{
		{
			name:  "startLessThanZero",
			start: -1,
			size:  10,
		},
		{
			name:  "sizeLessThanZero",
			start: -0,
			size:  -1,
		},
		{
			name:  "startGreaterThanObjectSize",
			start: int64(t.object.Size + 1),
			size:  int64(t.object.Size),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			_, err := t.rr.wrapped.getReadInfo(tc.start, tc.size)

			assert.Error(t.T(), err)
			errorString := fmt.Sprintf(
				"range [%d, %d) is illegal for %d-byte object", tc.start, tc.start+tc.size, t.object.Size)
			assert.Equal(t.T(), errorString, err.Error())
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadInfo_Sequential() {
	var testCases = []struct {
		testName    string
		expectedEnd int64
		start       int64
		objectSize  uint64
	}{
		{"10MBObject", 10 * MiB, 0, 10 * MiB},
		{"ReadSizeGreaterThanObjectSize", 10 * MiB, int64(t.object.Size - 1), 10 * MiB},
		{"ObjectSizeGreaterThanReadSize", int64(sequentialReadSizeInBytes), 0, 50 * MiB},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func() {
			t.object.Size = tc.objectSize
			end, err := t.rr.wrapped.getReadInfo(tc.start, 10)

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), common.ReadTypeSequential, t.rr.wrapped.readType)
			assert.Equal(t.T(), tc.expectedEnd, end)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadInfo_Random() {
	t.rr.wrapped.seeks = 2
	var testCases = []struct {
		testName       string
		expectedEnd    int64
		start          int64
		objectSize     uint64
		totalReadBytes uint64
	}{
		// TotalReadByte is 10MB, so average is 10/2 = 5MB >1MB and <8MB
		{"RangeBetween1And8MB", 6 * MiB, 0, 50 * MiB, 10 * MiB},
		// TotalReadByte is 1MB, so average is 1/2 = 0.5MB which is <1MB
		{"ReadSizeLessThan1MB", minReadSize, 0, 50 * MiB, 1 * MiB},
		// TotalReadByte is 1MB, so average is 10/2 = 5MB which is <8MB
		{"ReadSizeLessThan8MB", 6 * MiB, 0, 50 * MiB, 10 * MiB},
		// TotalReadByte is 1MB, so average is 20/2 = 10MB which is >8MB
		{"ReadSizeGreaterThan8MB", sequentialReadSizeInBytes, 0, 50 * MiB, 20 * MiB},
		{"ReadSizeGreaterThanObjectSize", 5 * MiB, 5*MiB - 1, 5 * MiB, 2 * MiB},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func() {
			t.object.Size = tc.objectSize
			t.rr.wrapped.totalReadBytes = tc.totalReadBytes
			end, err := t.rr.wrapped.getReadInfo(tc.start, 10)

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), common.ReadTypeRandom, t.rr.wrapped.readType)
			assert.Equal(t.T(), tc.expectedEnd, end)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReaderType() {
	testCases := []struct {
		name       string
		readType   string
		start      int64
		end        int64
		bucketType gcs.BucketType
		readerType ReaderType
	}{
		{
			name:       "ZonalBucketRandomRead",
			readType:   common.ReadTypeRandom,
			start:      50,
			end:        68,
			bucketType: gcs.BucketType{Zonal: true},
			readerType: MultiRangeReader,
		},
		{
			name:       "ZonalBucketRandomReadLargerThan8MB",
			readType:   common.ReadTypeRandom,
			start:      0,
			end:        9 * MiB,
			bucketType: gcs.BucketType{Zonal: true},
			readerType: RangeReader,
		},
		{
			name:       "ZonalBucketSequentialRead",
			readType:   common.ReadTypeSequential,
			start:      50,
			end:        68,
			bucketType: gcs.BucketType{Zonal: true},
			readerType: RangeReader,
		},
		{
			name:       "RegularBucketRandomRead",
			readType:   common.ReadTypeRandom,
			start:      50,
			end:        68,
			bucketType: gcs.BucketType{Zonal: false},
			readerType: RangeReader,
		},
		{
			name:       "RegularBucketSequentialRead",
			readType:   common.ReadTypeSequential,
			start:      50,
			end:        68,
			bucketType: gcs.BucketType{Zonal: false},
			readerType: RangeReader,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			readerType := readerType(tc.readType, tc.start, tc.end, tc.bucketType)
			assert.Equal(t.T(), readerType, tc.readerType)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromRangeReader_WhenExistingReaderIsNil() {
	testCases := []struct {
		name             string
		inputReadHandle  []byte
		outputReadHandle []byte
	}{
		{
			name:            "ReadHandlePresent",
			inputReadHandle: []byte("fake-handle"),
		},
		{
			name:            "ReadHandleAbsent",
			inputReadHandle: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rr.wrapped.readHandle = tc.inputReadHandle
			t.rr.wrapped.reader = nil
			t.rr.wrapped.start = 0
			t.object.Size = 5
			dataSize := 5
			testContent := testutil.GenerateRandomBytes(int(t.object.Size))
			rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
			readObjectRequest := &gcs.ReadObjectRequest{
				Name:       t.rr.wrapped.object.Name,
				Generation: t.rr.wrapped.object.Generation,
				Range: &gcs.ByteRange{
					Start: uint64(0),
					Limit: t.object.Size,
				},
				ReadCompressed: t.rr.wrapped.object.HasContentEncodingGzip(),
				ReadHandle:     t.rr.wrapped.readHandle,
			}
			t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(rc, nil).Times(1)
			buf := make([]byte, dataSize)

			n, err := t.rr.wrapped.readFromRangeReader(t.rr.ctx, buf, 0, int64(t.object.Size), "unhandled")

			t.mockBucket.AssertExpectations(t.T())
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), dataSize, n)
			assert.Equal(t.T(), testContent[:dataSize], buf)
			// Verify the reader state.
			assert.Nil(t.T(), t.rr.wrapped.reader)
			assert.Nil(t.T(), t.rr.wrapped.cancel)
			assert.Equal(t.T(), int64(5), t.rr.wrapped.start)
			assert.Equal(t.T(), int64(5), t.rr.wrapped.limit)
			assert.Equal(t.T(), int64(t.object.Size), t.rr.wrapped.expectedOffset)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromRangeReader_WhenExistingReaderIsNotNil() {
	t.rr.wrapped.start = 4
	t.rr.wrapped.limit = 10
	t.rr.wrapped.totalReadBytes = 4
	t.object.Size = 10
	dataSize := 4
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.rr.wrapped.reader = rc
	t.rr.wrapped.cancel = func() {}
	buf := make([]byte, dataSize)

	n, err := t.rr.wrapped.readFromRangeReader(t.rr.ctx, buf, 4, 8, "unhandled")

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), dataSize, n)
	// Verify the reader state.
	assert.Equal(t.T(), rc, t.rr.wrapped.reader)
	assert.NotNil(t.T(), t.rr.wrapped.cancel)
	assert.Equal(t.T(), int64(8), t.rr.wrapped.start)
	assert.Equal(t.T(), int64(10), t.rr.wrapped.limit)
	assert.Equal(t.T(), uint64(8), t.rr.wrapped.totalReadBytes)
	assert.Equal(t.T(), int64(8), t.rr.wrapped.expectedOffset)
}

func (t *RandomReaderStretchrTest) Test_ReadFromRangeReader_WhenAllDataFromReaderIsRead() {
	testCases := []struct {
		name       string
		readHandle []byte
	}{
		{
			name:       "GCSReturnedReadHandle",
			readHandle: []byte("fake-handle"),
		},
		{
			name:       "GCSReturnedNoReadHandle",
			readHandle: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rr.wrapped.start = 4
			t.rr.wrapped.limit = 10
			t.rr.wrapped.totalReadBytes = 4
			t.object.Size = 10
			dataSize := 6
			testContent := testutil.GenerateRandomBytes(int(t.object.Size))
			rc := &fake.FakeReader{
				ReadCloser: getReadCloser(testContent),
				Handle:     tc.readHandle,
			}
			t.rr.wrapped.reader = rc
			t.rr.wrapped.cancel = func() {}
			buf := make([]byte, dataSize)

			n, err := t.rr.wrapped.readFromRangeReader(t.rr.ctx, buf, 4, 10, "unhandled")

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), dataSize, n)
			// Verify the reader state.
			assert.Nil(t.T(), t.rr.wrapped.reader)
			assert.Nil(t.T(), t.rr.wrapped.cancel)
			assert.Equal(t.T(), int64(10), t.rr.wrapped.start)
			assert.Equal(t.T(), int64(10), t.rr.wrapped.limit)
			assert.Equal(t.T(), uint64(10), t.rr.wrapped.totalReadBytes)
			assert.Equal(t.T(), int64(10), t.rr.wrapped.expectedOffset)
			expectedReadHandle := tc.readHandle
			assert.Equal(t.T(), expectedReadHandle, t.rr.wrapped.readHandle)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromRangeReader_WhenReaderHasLessDataThanRequested() {
	testCases := []struct {
		name       string
		readHandle []byte
	}{
		{
			name:       "GCSReturnedReadHandle",
			readHandle: []byte("fake-handle"),
		},
		{
			name:       "GCSReturnedNoReadHandle",
			readHandle: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rr.wrapped.start = 0
			t.rr.wrapped.limit = 6
			t.rr.wrapped.totalReadBytes = 0
			dataSize := 6
			testContent := testutil.GenerateRandomBytes(dataSize)
			rc := &fake.FakeReader{
				ReadCloser: getReadCloser(testContent),
				Handle:     tc.readHandle,
			}
			t.rr.wrapped.reader = rc
			t.rr.wrapped.cancel = func() {}
			buf := make([]byte, 10)

			n, err := t.rr.wrapped.readFromRangeReader(t.rr.ctx, buf, 0, 10, "unhandled")

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), dataSize, n)
			// Verify the reader state.
			assert.Nil(t.T(), t.rr.wrapped.reader)
			assert.Nil(t.T(), t.rr.wrapped.cancel)
			assert.Equal(t.T(), int64(dataSize), t.rr.wrapped.start)
			assert.Equal(t.T(), int64(dataSize), t.rr.wrapped.limit)
			assert.Equal(t.T(), uint64(dataSize), t.rr.wrapped.totalReadBytes)
			assert.Equal(t.T(), int64(dataSize), t.rr.wrapped.expectedOffset)
			expectedReadHandle := tc.readHandle
			assert.Equal(t.T(), expectedReadHandle, t.rr.wrapped.readHandle)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromRangeReader_WhenReaderReturnedMoreData() {
	testCases := []struct {
		name       string
		readHandle []byte
	}{
		{
			name:       "GCSReturnedReadHandle",
			readHandle: []byte("fake-handle"),
		},
		{
			name:       "GCSReturnedNoReadHandle",
			readHandle: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rr.wrapped.start = 0
			t.rr.wrapped.limit = 6
			t.rr.wrapped.totalReadBytes = 0
			dataSize := 8
			testContent := testutil.GenerateRandomBytes(dataSize)
			rc := &fake.FakeReader{
				ReadCloser: getReadCloser(testContent),
				Handle:     tc.readHandle,
			}
			t.rr.wrapped.reader = rc
			t.rr.wrapped.cancel = func() {}
			buf := make([]byte, 10)

			_, err := t.rr.wrapped.readFromRangeReader(t.rr.ctx, buf, 0, 10, "unhandled")

			assert.True(t.T(), strings.Contains(err.Error(), "extra bytes: 2"))
			assert.Nil(t.T(), t.rr.wrapped.reader)
			assert.Nil(t.T(), t.rr.wrapped.cancel)
			assert.Equal(t.T(), int64(-1), t.rr.wrapped.start)
			assert.Equal(t.T(), int64(-1), t.rr.wrapped.limit)
			assert.Equal(t.T(), uint64(8), t.rr.wrapped.totalReadBytes)
			assert.Equal(t.T(), int64(0), t.rr.wrapped.expectedOffset)
			expectedReadHandle := tc.readHandle
			assert.Equal(t.T(), expectedReadHandle, t.rr.wrapped.readHandle)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromRangeReader_WhenReaderReturnedEOF() {
	t.rr.wrapped.start = 0
	t.rr.wrapped.limit = 10
	dataSize := 6
	testContent := testutil.GenerateRandomBytes(dataSize)
	rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.rr.wrapped.reader = rc
	t.rr.wrapped.cancel = func() {}
	buf := make([]byte, 10)

	_, err := t.rr.wrapped.readFromRangeReader(t.rr.ctx, buf, 0, 10, "unhandled")

	assert.True(t.T(), strings.Contains(err.Error(), "skipping 4 bytes"))
	assert.Equal(t.T(), int64(0), t.rr.wrapped.expectedOffset)
}

func (t *RandomReaderStretchrTest) Test_ExistingReader_WrongOffset() {
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
			// Simulate an existing reader.
			t.rr.wrapped.readHandle = tc.readHandle
			t.rr.wrapped.reader = &fake.FakeReader{
				ReadCloser: io.NopCloser(strings.NewReader("xxx")),
				Handle:     tc.readHandle,
			}
			t.rr.wrapped.cancel = func() {}
			t.rr.wrapped.start = 2
			t.rr.wrapped.limit = 5
			readObjectRequest := &gcs.ReadObjectRequest{
				Name:       t.rr.wrapped.object.Name,
				Generation: t.rr.wrapped.object.Generation,
				Range: &gcs.ByteRange{
					Start: uint64(0),
					Limit: t.object.Size,
				},
				ReadCompressed: t.rr.wrapped.object.HasContentEncodingGzip(),
				ReadHandle:     t.rr.wrapped.readHandle,
			}
			t.mockBucket.
				On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).
				Return(nil, errors.New(string(tc.readHandle))).
				Times(1)
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(1)

			buf := make([]byte, 1)

			_, err := t.rr.ReadAt(buf, 0)

			t.mockBucket.AssertExpectations(t.T())
			assert.NotNil(t.T(), err)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadAt_ExistingReaderLimitIsLessThanRequestedDataSize() {
	t.object.Size = 10
	// Simulate an existing reader.
	t.rr.wrapped.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte("xxx")), Handle: []byte("fake")}
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 2
	t.rr.wrapped.limit = 5
	rc := &fake.FakeReader{ReadCloser: getReadCloser([]byte("abcdefgh"))}
	expectedHandleInRequest := []byte(t.rr.wrapped.reader.ReadHandle())
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       t.rr.wrapped.object.Name,
		Generation: t.rr.wrapped.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(2),
			Limit: t.object.Size,
		},
		ReadCompressed: t.rr.wrapped.object.HasContentEncodingGzip(),
		ReadHandle:     expectedHandleInRequest,
	}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(rc, nil)
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(1)
	requestSize := 6
	buf := make([]byte, requestSize)

	data, err := t.rr.ReadAt(buf, 2)

	require.Nil(t.T(), err)
	require.Equal(t.T(), rc, t.rr.wrapped.reader)
	require.Equal(t.T(), requestSize, data.Size)
	require.Equal(t.T(), "abcdef", string(buf[:data.Size]))
	assert.Equal(t.T(), uint64(requestSize), t.rr.wrapped.totalReadBytes)
	assert.Equal(t.T(), int64(2+requestSize), t.rr.wrapped.expectedOffset)
	assert.Equal(t.T(), expectedHandleInRequest, t.rr.wrapped.readHandle)
}

func (t *RandomReaderStretchrTest) Test_ReadAt_ExistingReaderLimitIsLessThanRequestedObjectSize() {
	t.object.Size = 5
	// Simulate an existing reader
	t.rr.wrapped.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte("xxx")), Handle: []byte("fake")}
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 0
	t.rr.wrapped.limit = 3
	rc := &fake.FakeReader{ReadCloser: getReadCloser([]byte("abcde"))}
	expectedHandleInRequest := t.rr.wrapped.reader.ReadHandle()
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       t.rr.wrapped.object.Name,
		Generation: t.rr.wrapped.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(0),
			Limit: t.object.Size,
		},
		ReadCompressed: t.rr.wrapped.object.HasContentEncodingGzip(),
		ReadHandle:     expectedHandleInRequest,
	}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(rc, nil)
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(1)
	requestSize := 6
	buf := make([]byte, requestSize)

	data, err := t.rr.ReadAt(buf, 0)

	require.Nil(t.T(), err)
	require.Nil(t.T(), t.rr.wrapped.reader)
	require.Equal(t.T(), int(t.object.Size), data.Size)
	require.Equal(t.T(), "abcde", string(buf[:data.Size]))
	assert.Equal(t.T(), t.object.Size, t.rr.wrapped.totalReadBytes)
	assert.Equal(t.T(), int64(t.object.Size), t.rr.wrapped.expectedOffset)
	assert.Equal(t.T(), []byte(nil), t.rr.wrapped.readHandle)
}

func (t *RandomReaderStretchrTest) Test_ReadAt_InvalidOffset() {
	testCases := []struct {
		name     string
		dataSize int
		start    int
	}{
		{
			name:     "InvalidOffset",
			dataSize: 50,
			start:    68,
		},
		{
			name:     "NegativeOffset",
			dataSize: 100,
			start:    -50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rr.wrapped.reader = nil
			t.object.Size = uint64(tc.dataSize)
			buf := make([]byte, tc.dataSize)

			_, err := t.rr.wrapped.ReadAt(t.rr.ctx, buf, int64(tc.start))

			assert.Error(t.T(), err)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadAt_ValidateReadType() {
	testCases := []struct {
		name              string
		dataSize          int
		bucketType        gcs.BucketType
		readRanges        [][]int
		expectedReadTypes []string
		expectedSeeks     []int
	}{
		{
			name:              "SequentialReadFlat",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: false},
			readRanges:        [][]int{{0, 10}, {10, 20}, {20, 35}, {35, 50}},
			expectedReadTypes: []string{common.ReadTypeSequential, common.ReadTypeSequential, common.ReadTypeSequential, common.ReadTypeSequential},
			expectedSeeks:     []int{0, 0, 0, 0, 0},
		},
		{
			name:              "SequentialReadZonal",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: true},
			readRanges:        [][]int{{0, 10}, {10, 20}, {20, 35}, {35, 50}},
			expectedReadTypes: []string{common.ReadTypeSequential, common.ReadTypeSequential, common.ReadTypeSequential, common.ReadTypeSequential},
			expectedSeeks:     []int{0, 0, 0, 0, 0},
		},
		{
			name:              "RandomReadFlat",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: false},
			readRanges:        [][]int{{0, 50}, {30, 40}, {10, 20}, {20, 30}, {30, 40}},
			expectedReadTypes: []string{common.ReadTypeSequential, common.ReadTypeSequential, common.ReadTypeRandom, common.ReadTypeRandom, common.ReadTypeRandom},
			expectedSeeks:     []int{0, 1, 2, 2, 2},
		},
		{
			name:              "RandomReadZonal",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: true},
			readRanges:        [][]int{{0, 50}, {30, 40}, {10, 20}, {20, 30}, {30, 40}},
			expectedReadTypes: []string{common.ReadTypeSequential, common.ReadTypeSequential, common.ReadTypeRandom, common.ReadTypeRandom, common.ReadTypeRandom},
			expectedSeeks:     []int{0, 1, 2, 2, 2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			assert.Equal(t.T(), len(tc.readRanges), len(tc.expectedReadTypes), "Test Parameter Error: readRanges and expectedReadTypes should have same length")
			t.rr.wrapped.reader = nil
			t.rr.wrapped.isMRDInUse = false
			t.rr.wrapped.seeks = 0
			t.rr.wrapped.readType = common.ReadTypeSequential
			t.rr.wrapped.expectedOffset = 0
			t.object.Size = uint64(tc.dataSize)
			testContent := testutil.GenerateRandomBytes(int(t.object.Size))
			fakeMRDWrapper, err := NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{}, &cfg.Config{})
			assert.Nil(t.T(), err, "Error in creating MRDWrapper")
			t.rr.wrapped.mrdWrapper = &fakeMRDWrapper
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond))
			t.mockBucket.On("BucketType", mock.Anything).Return(tc.bucketType).Times(len(tc.readRanges))

			for i, readRange := range tc.readRanges {
				t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(&fake.FakeReader{ReadCloser: getReadCloser(testContent)}, nil).Once()
				buf := make([]byte, readRange[1]-readRange[0])

				_, err := t.rr.wrapped.ReadAt(t.rr.ctx, buf, int64(readRange[0]))

				assert.NoError(t.T(), err)
				assert.Equal(t.T(), tc.expectedReadTypes[i], t.rr.wrapped.readType)
				assert.Equal(t.T(), int64(readRange[1]), t.rr.wrapped.expectedOffset)
				assert.Equal(t.T(), uint64(tc.expectedSeeks[i]), t.rr.wrapped.seeks)
			}
		})
	}
}

// This test validates the bug fix where seeks are not updated correctly in case of zonal bucket random reads (b/410904634).
func (t *RandomReaderStretchrTest) Test_ReadAt_ValidateZonalRandomReads() {
	t.rr.wrapped.reader = nil
	t.rr.wrapped.isMRDInUse = false
	t.rr.wrapped.seeks = 0
	t.rr.wrapped.readType = common.ReadTypeSequential
	t.rr.wrapped.expectedOffset = 0
	t.rr.wrapped.totalReadBytes = 0
	t.object.Size = 20 * MiB
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true})
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	fakeMRDWrapper, err := NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{}, &cfg.Config{})
	assert.Nil(t.T(), err, "Error in creating MRDWrapper")
	t.rr.wrapped.mrdWrapper = &fakeMRDWrapper
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(&fake.FakeReader{ReadCloser: getReadCloser(testContent)}, nil).Twice()
	buf := make([]byte, 3*MiB)

	// Sequential read #1
	_, err = t.rr.wrapped.ReadAt(t.rr.ctx, buf, 13*MiB)
	assert.NoError(t.T(), err)
	// Random read #1
	seeks := 1
	_, err = t.rr.wrapped.ReadAt(t.rr.ctx, buf, 12*MiB)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), uint64(seeks), t.rr.wrapped.seeks)

	readRanges := [][]int{{11 * MiB, 15 * MiB}, {12 * MiB, 14 * MiB}, {10 * MiB, 12 * MiB}, {9 * MiB, 11 * MiB}, {8 * MiB, 10 * MiB}}
	// Series of random reads to check if seeks are updated correctly and MRD is invoked always
	for _, readRange := range readRanges {
		seeks++
		t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond))
		buf := make([]byte, readRange[1]-readRange[0])

		_, err := t.rr.wrapped.ReadAt(t.rr.ctx, buf, int64(readRange[0]))

		assert.NoError(t.T(), err)
		assert.Equal(t.T(), common.ReadTypeRandom, t.rr.wrapped.readType)
		assert.Equal(t.T(), int64(readRange[1]), t.rr.wrapped.expectedOffset)
		assert.Equal(t.T(), uint64(seeks), t.rr.wrapped.seeks)
	}
}

func (t *RandomReaderStretchrTest) Test_ReadAt_MRDRead() {
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
			t.rr.wrapped.reader = nil
			t.rr.wrapped.isMRDInUse = false
			t.rr.wrapped.expectedOffset = 10
			t.rr.wrapped.seeks = minSeeksForRandom + 1
			t.object.Size = uint64(tc.dataSize)
			testContent := testutil.GenerateRandomBytes(int(t.object.Size))
			fakeMRDWrapper, err := NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{}, &cfg.Config{})
			assert.Nil(t.T(), err, "Error in creating MRDWrapper")
			t.rr.wrapped.mrdWrapper = &fakeMRDWrapper
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond)).Times(1)
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true}).Times(1)
			buf := make([]byte, tc.bytesToRead)

			objData, err := t.rr.wrapped.ReadAt(t.rr.ctx, buf, int64(tc.offset))

			t.mockBucket.AssertNotCalled(t.T(), "NewReaderWithReadHandle", mock.Anything)
			assert.NoError(t.T(), err)
			assert.Nil(t.T(), t.rr.wrapped.reader)
			assert.Equal(t.T(), tc.bytesToRead, objData.Size)
			assert.Equal(t.T(), testContent[tc.offset:tc.offset+tc.bytesToRead], objData.DataBuf[:objData.Size])
			if tc.bytesToRead != 0 {
				assert.Equal(t.T(), int64(tc.offset+tc.bytesToRead), t.rr.wrapped.expectedOffset)
			}
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromMultiRangeReader_ReadFull() {
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
			t.rr.wrapped.reader = nil
			t.rr.wrapped.isMRDInUse = false
			t.object.Size = uint64(tc.dataSize)
			testContent := testutil.GenerateRandomBytes(int(t.object.Size))
			fakeMRDWrapper, err := NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{}, &cfg.Config{})
			assert.Nil(t.T(), err, "Error in creating MRDWrapper")
			t.rr.wrapped.mrdWrapper = &fakeMRDWrapper
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond)).Times(1)
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true}).Times(1)
			buf := make([]byte, tc.dataSize+tc.extraSize)

			bytesRead, err := t.rr.wrapped.readFromMultiRangeReader(t.rr.ctx, buf, 0, int64(t.object.Size), TestTimeoutForMultiRangeRead)

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), tc.dataSize, bytesRead)
			assert.Equal(t.T(), testContent[:tc.dataSize], buf[:bytesRead])
			assert.Equal(t.T(), int64(t.object.Size), t.rr.wrapped.expectedOffset)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromMultiRangeReader_ReadChunk() {
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
		t.rr.wrapped.reader = nil
		t.object.Size = uint64(tc.dataSize)
		testContent := testutil.GenerateRandomBytes(int(t.object.Size))
		fakeMRDWrapper, err := NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{}, &cfg.Config{})
		assert.Nil(t.T(), err, "Error in creating MRDWrapper")
		t.rr.wrapped.mrdWrapper = &fakeMRDWrapper
		t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond)).Times(1)
		t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true}).Times(1)
		buf := make([]byte, tc.end-tc.start)

		bytesRead, err := t.rr.wrapped.readFromMultiRangeReader(t.rr.ctx, buf, int64(tc.start), int64(tc.end), TestTimeoutForMultiRangeRead)

		assert.NoError(t.T(), err)
		assert.Equal(t.T(), tc.end-tc.start, bytesRead)
		assert.Equal(t.T(), testContent[tc.start:tc.end], buf[:bytesRead])
		assert.Equal(t.T(), int64(tc.end), t.rr.wrapped.expectedOffset)
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromMultiRangeReader_NilMRDWrapper() {
	t.rr.wrapped.mrdWrapper = nil

	bytesRead, err := t.rr.wrapped.readFromMultiRangeReader(t.rr.ctx, make([]byte, t.object.Size), 0, int64(t.object.Size), TestTimeoutForMultiRangeRead)

	assert.ErrorContains(t.T(), err, "readFromMultiRangeReader: Invalid MultiRangeDownloaderWrapper")
	assert.Equal(t.T(), 0, bytesRead)
}

func (t *RandomReaderStretchrTest) Test_ReadFromMultiRangeReader_ValidateTimeout() {
	testCases := []struct {
		name               string
		dataSize           int
		timeout            time.Duration
		sleepTime          time.Duration
		expectedErrKeyword string
	}{
		{
			name:               "TimeoutPlusFiveMilliSecond",
			dataSize:           100,
			timeout:            5 * time.Millisecond,
			sleepTime:          10 * time.Millisecond,
			expectedErrKeyword: "timeout",
		},
		{
			name:               "TimeoutValue",
			dataSize:           100,
			timeout:            5 * time.Millisecond,
			sleepTime:          5 * time.Millisecond,
			expectedErrKeyword: "timeout",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rr.wrapped.reader = nil
			t.rr.wrapped.isMRDInUse = false
			t.object.Size = uint64(tc.dataSize)
			testContent := testutil.GenerateRandomBytes(int(t.object.Size))
			fakeMRDWrapper, err := NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{}, &cfg.Config{})
			assert.Nil(t.T(), err, "Error in creating MRDWrapper")
			t.rr.wrapped.mrdWrapper = &fakeMRDWrapper
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, tc.sleepTime)).Once()
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true}).Once()
			buf := make([]byte, tc.dataSize)

			bytesRead, err := t.rr.wrapped.readFromMultiRangeReader(t.rr.ctx, buf, 0, int64(t.object.Size), tc.timeout)

			if tc.name == "TimeoutValue" && bytesRead != 0 {
				assert.NoError(t.T(), err)
				assert.Equal(t.T(), tc.dataSize, bytesRead)
				assert.Equal(t.T(), testContent[:tc.dataSize], buf[:bytesRead])
				return
			}
			assert.ErrorContains(t.T(), err, tc.expectedErrKeyword)
		})
	}
}

// Validates:
// 1. No change in ReadAt behavior based inactiveStreamTimeout config.
// 2. Valid timeout config creates InactiveTimeoutReader instance of storage.Reader.
func (t *RandomReaderStretchrTest) Test_ReadAt_WithAndWithoutReadConfig() {
	testCases := []struct {
		name                        string
		config                      *cfg.ReadConfig
		expectInactiveTimeoutReader bool
	}{
		{
			name:                        "WithoutReadConfig",
			config:                      nil,
			expectInactiveTimeoutReader: false,
		},
		{
			name:                        "WithReadConfigAndZeroTimeout",
			config:                      &cfg.ReadConfig{InactiveStreamTimeout: 0},
			expectInactiveTimeoutReader: false,
		},
		{
			name:                        "WithReadConfigAndPositiveTimeout",
			config:                      &cfg.ReadConfig{InactiveStreamTimeout: 10 * time.Millisecond},
			expectInactiveTimeoutReader: true,
		},
	}

	objectSize := uint64(20)
	readOffset := int64(0)
	readLength := 10 // Reading only 10 bytes from the complete object reader.

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.SetupTest() // Resets mockBucket, rr, etc. for each sub-test
			defer t.TearDownTest()

			t.rr.wrapped.config = tc.config
			t.rr.wrapped.reader = nil // Ensure startRead path is taken in ReadAt
			t.object.Size = objectSize
			// Prepare fake content for the GCS object.
			// startRead will attempt to read the entire object [0, objectSize)
			// because objectSize is small compared to typical sequentialReadSizeMb.
			fakeReaderContent := testutil.GenerateRandomBytes(int(t.object.Size))
			rc := &fake.FakeReader{ReadCloser: getReadCloser(fakeReaderContent)}
			expectedReadObjectRequest := &gcs.ReadObjectRequest{
				Name:       t.rr.wrapped.object.Name,
				Generation: t.rr.wrapped.object.Generation,
				Range: &gcs.ByteRange{
					Start: uint64(readOffset),    // Read from the beginning
					Limit: uint64(t.object.Size), // getReadInfo will determine this limit
				},
				ReadCompressed: t.rr.wrapped.object.HasContentEncodingGzip(),
				ReadHandle:     nil, // No existing read handle
			}
			t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, expectedReadObjectRequest).Return(rc, nil).Once()
			// BucketType is called by ReadAt -> getReadInfo -> readerType to determine reader strategy.
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: false}).Once()
			buf := make([]byte, readLength)

			objectData, err := t.rr.ReadAt(buf, readOffset)

			t.mockBucket.AssertExpectations(t.T())
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), readLength, objectData.Size)
			assert.Equal(t.T(), fakeReaderContent[:readLength], buf[:objectData.Size]) // Ensure buffer is populated correctly
			assert.NotNil(t.T(), t.rr.wrapped.reader, "Reader should be active as partial data read from the requested range.")
			assert.NotNil(t.T(), t.rr.wrapped.cancel)
			assert.Equal(t.T(), int64(readLength), t.rr.wrapped.start)
			assert.Equal(t.T(), int64(t.object.Size), t.rr.wrapped.limit)
			_, isInactiveTimeoutReader := t.rr.wrapped.reader.(*InactiveTimeoutReader)
			assert.Equal(t.T(), tc.expectInactiveTimeoutReader, isInactiveTimeoutReader)
		})
	}
}
