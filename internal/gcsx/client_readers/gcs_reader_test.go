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

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/clock"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	testUtil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	sequential             = "Sequential"
	random                 = "Random"
	sequentialReadSizeInMb = 22
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
	t.gcsReader = NewGCSReader(t.object, t.mockBucket, &GCSReaderConfig{
		MetricHandle:         common.NewNoopMetrics(),
		MrdWrapper:           nil,
		SequentialReadSizeMb: sequentialReadSizeInMb,
		ReadConfig:           nil,
	})
	t.ctx = context.Background()
}

func (t *gcsReaderTest) TearDownTest() {
	t.gcsReader.Destroy()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *gcsReaderTest) Test_NewGCSReader() {
	object := &gcs.MinObject{
		Name:       testObject,
		Size:       30,
		Generation: 4321,
	}

	gcsReader := NewGCSReader(object, t.mockBucket, &GCSReaderConfig{
		MetricHandle:         common.NewNoopMetrics(),
		MrdWrapper:           nil,
		SequentialReadSizeMb: 200,
		ReadConfig:           nil,
	})

	assert.Equal(t.T(), object, gcsReader.object)
	assert.Equal(t.T(), t.mockBucket, gcsReader.bucket)
	assert.Equal(t.T(), testUtil.Sequential, gcsReader.readType)
}

func (t *gcsReaderTest) Test_ReadAt_InvalidOffset() {
	testCases := []struct {
		name       string
		objectSize int
		start      int
	}{
		{
			name:       "InvalidOffset",
			objectSize: 50,
			start:      68,
		},
		{
			name:       "NegativeOffset",
			objectSize: 100,
			start:      -50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.gcsReader.rangeReader.reader = nil
			t.object.Size = uint64(tc.objectSize)
			buf := make([]byte, tc.objectSize)

			_, err := t.gcsReader.ReadAt(t.ctx, buf, int64(tc.start))

			assert.Error(t.T(), err)
		})
	}
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
			Start: 2,
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

func (t *gcsReaderTest) Test_ReadAt_PropagatesCancellation() {
	// Set up a blocking reader
	finishRead := make(chan struct{})
	blocking := &blockingReader{c: finishRead}
	rc := io.NopCloser(blocking)
	// Assign it to the rangeReader
	t.gcsReader.rangeReader.reader = &fake.FakeReader{ReadCloser: rc}
	t.gcsReader.rangeReader.start = 0
	t.gcsReader.rangeReader.limit = 2
	// Track cancel invocation
	cancelCalled := make(chan struct{})
	t.gcsReader.rangeReader.cancel = func() { close(cancelCalled) }
	// Controlled context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Channel to track read completion
	readReturned := make(chan struct{})
	var err error

	go func() {
		_, err = t.gcsReader.ReadAt(ctx, make([]byte, 2), 0)

		assert.Error(t.T(), err)

		close(readReturned)
	}()

	// Wait a bit to ensure ReadAt is blocking
	select {
	case <-readReturned:
		t.T().Fatal("Read returned early — cancellation did not propagate properly.")
	case <-time.After(100 * time.Millisecond):
		// OK: Still blocked
	}
	// Cancel the context to trigger cancellation
	cancel()
	// Expect rr.cancel to be called
	select {
	case <-cancelCalled:
		// Pass
	case <-time.After(100 * time.Millisecond):
		t.T().Fatal("Expected rr.cancel to be called on ctx cancellation.")
	}
	// Unblock the reader so the read can complete
	close(finishRead)
	// Ensure read completes
	select {
	case <-readReturned:
		// Pass
	case <-time.After(100 * time.Millisecond):
		t.T().Fatal("Expected read to return after unblocking.")
	}
}

func (t *gcsReaderTest) Test_ReadInfo_WithInvalidInput() {
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
			start: 0,
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
			_, err := t.gcsReader.getReadInfo(tc.start, tc.size)

			assert.Error(t.T(), err)
		})
	}
}

func (t *gcsReaderTest) Test_ReadInfo_Sequential() {
	testCases := []struct {
		name        string
		start       int64
		objectSize  uint64
		expectedEnd int64
	}{
		{
			name:        "ExactSizeRead", // start 0, object = 10MB
			start:       0,
			objectSize:  10 * MiB,
			expectedEnd: 10 * MiB,
		},
		{
			name:        "ReadSizeGreaterThanObjectSize", // start near end, should clamp to objectSize
			start:       int64(10*MiB - 1),
			objectSize:  10 * MiB,
			expectedEnd: 10 * MiB,
		},
		{
			name:        "ObjectSizeGreaterThanReadSize", // default read size applies
			start:       0,
			objectSize:  50 * MiB,
			expectedEnd: 22 * MB, // equals to sequentialReadSizeInMb
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.SetupTest()
			t.object.Size = tc.objectSize

			end, err := t.gcsReader.getReadInfo(tc.start, int64(tc.objectSize))

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), sequential, t.gcsReader.readType)
			assert.Equal(t.T(), tc.expectedEnd, end)
		})
	}
}

func (t *gcsReaderTest) Test_ReadInfo_Random() {
	t.gcsReader.seeks = 2
	testCases := []struct {
		name           string
		start          int64
		objectSize     uint64
		totalReadBytes uint64
		expectedEnd    int64
	}{
		{
			name:           "RangeBetween1And8MB",
			start:          0,
			objectSize:     50 * MiB,
			totalReadBytes: 10 * MiB,
			expectedEnd:    6 * MiB,
		},
		{
			name:           "ReadSizeLessThan1MB",
			start:          0,
			objectSize:     50 * MiB,
			totalReadBytes: 1 * MiB, // avg = 0.5MB
			expectedEnd:    MB,      // equals to minReadSize
		},
		{
			name:           "ReadSizeGreaterThan8MB",
			start:          0,
			objectSize:     50 * MiB,
			totalReadBytes: 20 * MiB,
			expectedEnd:    22 * MB, // equals to sequentialReadSizeInMb
		},
		{
			name:           "ReadSizeGreaterThan8MB",
			start:          5*MiB - 1,
			objectSize:     5 * MiB,
			totalReadBytes: 2 * MiB,
			expectedEnd:    5 * MiB,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.object.Size = tc.objectSize
			t.gcsReader.totalReadBytes = tc.totalReadBytes

			end, err := t.gcsReader.getReadInfo(tc.start, int64(tc.objectSize))

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), random, t.gcsReader.readType)
			assert.Equal(t.T(), tc.expectedEnd, end)
		})
	}
}

// Validates:
// 1. No change in ReadAt behavior based inactiveStreamTimeout readConfig.
// 2. Valid timeout readConfig creates inactiveTimeoutReader instance of storage.Reader.
func (t *gcsReaderTest) Test_ReadAt_WithAndWithoutReadConfig() {
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

			t.gcsReader.rangeReader.readConfig = tc.config
			t.gcsReader.rangeReader.reader = nil // Ensure startRead path is taken in ReadAt
			t.object.Size = objectSize
			// Prepare fake content for the GCS object.
			// startRead will attempt to read the entire object [0, objectSize)
			// because objectSize is small compared to typical SequentialReadSizeMb.
			fakeReaderContent := testUtil.GenerateRandomBytes(int(t.object.Size))
			rc := &fake.FakeReader{ReadCloser: getReadCloser(fakeReaderContent)}
			expectedReadObjectRequest := &gcs.ReadObjectRequest{
				Name:       t.object.Name,
				Generation: t.object.Generation,
				Range: &gcs.ByteRange{
					Start: uint64(readOffset), // Read from the beginning
					Limit: t.object.Size,      // getReadInfo will determine this limit
				},
				ReadCompressed: t.object.HasContentEncodingGzip(),
				ReadHandle:     nil, // No existing read handle
			}
			t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, expectedReadObjectRequest).Return(rc, nil).Once()
			// BucketType is called by ReadAt -> getReadInfo -> readerType to determine reader strategy.
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: false}).Once()

			objectData, err := t.readAt(readOffset, int64(readLength))

			t.mockBucket.AssertExpectations(t.T())
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), readLength, objectData.Size)
			assert.Equal(t.T(), fakeReaderContent[:readLength], objectData.DataBuf[:objectData.Size]) // Ensure buffer is populated correctly
			assert.NotNil(t.T(), t.gcsReader.rangeReader.reader, "Reader should be active as partial data read from the requested range.")
			assert.NotNil(t.T(), t.gcsReader.rangeReader.cancel)
			assert.Equal(t.T(), int64(readLength), t.gcsReader.rangeReader.start)
			assert.Equal(t.T(), int64(t.object.Size), t.gcsReader.rangeReader.limit)
			_, isInactiveTimeoutReader := t.gcsReader.rangeReader.reader.(*gcsx.InactiveTimeoutReader)
			assert.Equal(t.T(), tc.expectInactiveTimeoutReader, isInactiveTimeoutReader)
		})
	}
}
