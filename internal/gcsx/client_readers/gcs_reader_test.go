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
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/clock"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	testUtil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
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
		MetricHandle:         metrics.NewNoopMetrics(),
		MrdWrapper:           nil,
		SequentialReadSizeMb: sequentialReadSizeInMb,
		Config:               nil,
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
		MetricHandle:         metrics.NewNoopMetrics(),
		MrdWrapper:           nil,
		SequentialReadSizeMb: 200,
		Config:               nil,
	})

	assert.Equal(t.T(), object, gcsReader.object)
	assert.Equal(t.T(), t.mockBucket, gcsReader.bucket)
	assert.Equal(t.T(), metrics.ReadTypeSequential, gcsReader.readType.Load())
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
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(3)
	requestSize := 6

	readerResponse, err := t.readAt(2, int64(requestSize))
	// The reader should be the same as the one returned by NewReaderWithReadHandle.
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), rc, t.gcsReader.rangeReader.reader)
	assert.Equal(t.T(), requestSize, readerResponse.Size)
	assert.Equal(t.T(), uint64(requestSize), t.gcsReader.totalReadBytes.Load())
	assert.Equal(t.T(), int64(2+requestSize), t.gcsReader.expectedOffset.Load())
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
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(3)
	requestSize := 6

	readerResponse, err := t.readAt(0, int64(requestSize))
	// The reader should be nil as it was closed and a new one was not created.
	assert.NoError(t.T(), err)
	assert.Nil(t.T(), t.gcsReader.rangeReader.reader)
	assert.Equal(t.T(), int(t.object.Size), readerResponse.Size)
	assert.Equal(t.T(), int64(t.object.Size), t.gcsReader.expectedOffset.Load())
	assert.Equal(t.T(), []byte(nil), t.gcsReader.rangeReader.readHandle)
}

func (t *gcsReaderTest) Test_ReadAt_ExistingReaderIsFine() {
	t.object.Size = 6
	content := "xxx"
	// Simulate an existing reader
	t.gcsReader.rangeReader.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte(content)), Handle: []byte("fake")}
	t.gcsReader.rangeReader.cancel = func() {}
	t.gcsReader.rangeReader.start = 2
	t.gcsReader.totalReadBytes.Store(2)
	t.gcsReader.rangeReader.limit = 5
	requestSize := 3
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(3)

	readerResponse, err := t.readAt(2, int64(requestSize))

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 3, readerResponse.Size)
	assert.Equal(t.T(), uint64(5), t.gcsReader.totalReadBytes.Load())
	assert.Equal(t.T(), int64(5), t.gcsReader.expectedOffset.Load())
	assert.Equal(t.T(), []byte("fake"), t.gcsReader.rangeReader.readHandle)
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
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(3)
			requestSize := 6

			readerResponse, err := t.readAt(0, int64(requestSize))

			t.mockBucket.AssertExpectations(t.T())
			assert.NoError(t.T(), err)
			assert.Nil(t.T(), t.gcsReader.rangeReader.reader)
			assert.Equal(t.T(), int(t.object.Size), readerResponse.Size)
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
		expectedReadTypes []int64
		expectedSeeks     []int
	}{
		{
			name:              "SequentialReadFlat",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: false},
			readRanges:        [][]int{{0, 10}, {10, 20}, {20, 35}, {35, 50}},
			expectedReadTypes: []int64{metrics.ReadTypeSequential, metrics.ReadTypeSequential, metrics.ReadTypeSequential, metrics.ReadTypeSequential},
			expectedSeeks:     []int{0, 0, 0, 0, 0},
		},
		{
			name:              "SequentialReadZonal",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: true},
			readRanges:        [][]int{{0, 10}, {10, 20}, {20, 35}, {35, 50}},
			expectedReadTypes: []int64{metrics.ReadTypeSequential, metrics.ReadTypeSequential, metrics.ReadTypeSequential, metrics.ReadTypeSequential},
			expectedSeeks:     []int{0, 0, 0, 0, 0},
		},
		{
			name:              "RandomReadFlat",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: false},
			readRanges:        [][]int{{0, 50}, {30, 40}, {10, 20}, {20, 30}, {30, 40}},
			expectedReadTypes: []int64{metrics.ReadTypeSequential, metrics.ReadTypeSequential, metrics.ReadTypeRandom, metrics.ReadTypeRandom, metrics.ReadTypeRandom},
			expectedSeeks:     []int{0, 1, 2, 2, 2},
		},
		{
			name:              "RandomReadZonal",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: true},
			readRanges:        [][]int{{0, 50}, {30, 40}, {10, 20}, {20, 30}, {30, 40}},
			expectedReadTypes: []int64{metrics.ReadTypeSequential, metrics.ReadTypeSequential, metrics.ReadTypeRandom, metrics.ReadTypeRandom, metrics.ReadTypeRandom},
			expectedSeeks:     []int{0, 1, 2, 2, 2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.SetupTest()
			require.Equal(t.T(), len(tc.readRanges), len(tc.expectedReadTypes), "Test Parameter Error: readRanges and expectedReadTypes should have same length")
			t.gcsReader.mrr.isMRDInUse.Store(false)
			t.gcsReader.seeks.Store(0)
			t.gcsReader.rangeReader.readType = metrics.ReadTypeSequential
			t.gcsReader.expectedOffset.Store(0)
			t.object.Size = uint64(tc.dataSize)
			testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
			fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{}, &cfg.Config{})
			require.NoError(t.T(), err, "Error in creating MRDWrapper")
			t.gcsReader.mrr.mrdWrapper = &fakeMRDWrapper
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond))
			t.mockBucket.On("BucketType", mock.Anything).Return(tc.bucketType).Times(3 * len(tc.readRanges))

			for i, readRange := range tc.readRanges {
				t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(&fake.FakeReader{ReadCloser: getReadCloser(testContent)}, nil).Once()

				_, err = t.readAt(int64(readRange[0]), int64(readRange[1]-readRange[0]))

				assert.NoError(t.T(), err)
				assert.Equal(t.T(), tc.expectedReadTypes[i], t.gcsReader.readType.Load())
				assert.Equal(t.T(), int64(readRange[1]), t.gcsReader.expectedOffset.Load())
				assert.Equal(t.T(), uint64(tc.expectedSeeks[i]), t.gcsReader.seeks.Load())
			}
		})
	}
}

func (t *gcsReaderTest) Test_ReadAt_PropagatesCancellation() {
	t.gcsReader = NewGCSReader(t.object, t.mockBucket, &GCSReaderConfig{
		MetricHandle:         metrics.NewNoopMetrics(),
		MrdWrapper:           nil,
		SequentialReadSizeMb: sequentialReadSizeInMb,
		Config:               &cfg.Config{FileSystem: cfg.FileSystemConfig{IgnoreInterrupts: false}},
	})
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
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(3)

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

func (t *gcsReaderTest) Test_IsSeekNeeded() {
	testCases := []struct {
		name           string
		readType       int64
		offset         int64
		expectedOffset int64
		want           bool
	}{
		{
			name:           "First read, expectedOffset is 0",
			readType:       metrics.ReadTypeSequential,
			offset:         100,
			expectedOffset: 0,
			want:           false,
		},
		{
			name:           "Random read, same offset",
			readType:       metrics.ReadTypeRandom,
			offset:         100,
			expectedOffset: 100,
			want:           false,
		},
		{
			name:           "Random read, different offset",
			readType:       metrics.ReadTypeRandom,
			offset:         200,
			expectedOffset: 100,
			want:           true,
		},
		{
			name:           "Sequential read, same offset",
			readType:       metrics.ReadTypeSequential,
			offset:         100,
			expectedOffset: 100,
			want:           false,
		},
		{
			name:           "Sequential read, small forward jump within maxReadSize",
			readType:       metrics.ReadTypeSequential,
			offset:         100 + maxReadSize/2,
			expectedOffset: 100,
			want:           false,
		},
		{
			name:           "Sequential read, forward jump to boundary of maxReadSize",
			readType:       metrics.ReadTypeSequential,
			offset:         100 + maxReadSize,
			expectedOffset: 100,
			want:           false,
		},
		{
			name:           "Sequential read, large forward jump beyond maxReadSize",
			readType:       metrics.ReadTypeSequential,
			offset:         100 + maxReadSize + 1,
			expectedOffset: 100,
			want:           true,
		},
		{
			name:           "Sequential read, backward jump",
			readType:       metrics.ReadTypeSequential,
			offset:         99,
			expectedOffset: 100,
			want:           true,
		},
		{
			name:           "Unknown read type",
			readType:       -1, // An invalid read type
			offset:         200,
			expectedOffset: 100,
			want:           false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			got := isSeekNeeded(tc.readType, tc.offset, tc.expectedOffset)
			assert.Equal(t.T(), tc.want, got)
		})
	}
}

func (t *gcsReaderTest) Test_GetEndOffset() {
	testCases := []struct {
		name                  string
		start                 int64
		objectSize            int64
		initialReadType       int64
		initialNumSeeks       uint64
		initialTotalReadBytes uint64
		sequentialReadSizeMb  int32
		expectedEnd           int64
	}{
		{
			name:                  "Sequential Read, Fits in sequentialReadSizeMb",
			start:                 0,
			objectSize:            10 * MiB,
			initialReadType:       metrics.ReadTypeSequential,
			initialNumSeeks:       0,
			initialTotalReadBytes: 0,
			sequentialReadSizeMb:  22,
			expectedEnd:           10 * MiB,
		},
		{
			name:                  "Sequential Read, Object Larger than sequentialReadSizeMb",
			start:                 0,
			objectSize:            50 * MiB,
			initialReadType:       metrics.ReadTypeSequential,
			initialNumSeeks:       0,
			initialTotalReadBytes: 0,
			sequentialReadSizeMb:  22,
			expectedEnd:           22 * MiB,
		},
		{
			name:                  "Sequential Read, Respects object size",
			start:                 5 * MiB,
			objectSize:            7 * MiB,
			initialReadType:       metrics.ReadTypeSequential,
			initialNumSeeks:       0,
			initialTotalReadBytes: 0,
			sequentialReadSizeMb:  22,
			expectedEnd:           7 * MiB,
		},
		{
			name:                  "Random Read, Min read size",
			start:                 0,
			objectSize:            5 * MiB,
			initialReadType:       metrics.ReadTypeRandom,
			initialNumSeeks:       minSeeksForRandom,
			initialTotalReadBytes: 1000,
			sequentialReadSizeMb:  22,
			expectedEnd:           minReadSize,
		},
		{
			name:                  "Random Read, Averages less than minReadSize",
			start:                 0,
			objectSize:            50 * MiB,
			initialReadType:       metrics.ReadTypeRandom,
			initialNumSeeks:       minSeeksForRandom,
			initialTotalReadBytes: 100 * 1024, // 100KiB
			sequentialReadSizeMb:  22,
			expectedEnd:           minReadSize, // Should be atleast minReadSize
		},
		{
			name:                  "Random Read, Start Offset Non-Zero",
			start:                 5 * MiB,
			objectSize:            50 * MiB,
			initialReadType:       metrics.ReadTypeRandom,
			initialNumSeeks:       minSeeksForRandom,
			initialTotalReadBytes: 2 * MiB, // avg read bytes = 1MiB
			sequentialReadSizeMb:  22,
			expectedEnd:           5*MiB + 2*MiB, // avg read bytes + 1MiB
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.object.Size = uint64(tc.objectSize)
			t.gcsReader.readType.Store(tc.initialReadType)
			t.gcsReader.seeks.Store(tc.initialNumSeeks)
			t.gcsReader.totalReadBytes.Store(tc.initialTotalReadBytes)

			end := t.gcsReader.getEndOffset(tc.start)

			assert.Equal(t.T(), tc.expectedEnd, end, "End offset mismatch")
		})
	}
}

func (t *gcsReaderTest) Test_GetReadInfo() {
	testCases := []struct {
		name                  string
		offset                int64
		seekRecorded          bool
		initialReadType       int64
		initialExpOffset      int64
		initialNumSeeks       uint64
		initialTotalReadBytes uint64
		expectedReadType      int64
		expectedNumSeeks      uint64
	}{
		{
			name:                  "First Read",
			offset:                0,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      0,
			initialNumSeeks:       0,
			initialTotalReadBytes: 0,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      0,
		},
		{
			name:                  "Sequential Read",
			offset:                10,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      10,
			initialNumSeeks:       0,
			initialTotalReadBytes: 100,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      0,
		},
		{
			name:                  "Sequential read with small forward jump and high average read bytes is still sequential",
			offset:                100,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      10,
			initialNumSeeks:       0,
			initialTotalReadBytes: 10000000,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      0,
		},
		{
			name:                  "Sequential read with large forward jump is a seek",
			offset:                50 + maxReadSize + 1,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       0,
			initialTotalReadBytes: 50 * 1024,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      1,
		},
		{
			name:                  "Sequential read with backward jump is a seek",
			offset:                49,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       0,
			initialTotalReadBytes: 50 * 1024,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      1,
		},
		{
			name:                  "Contiguous random read is not a seek",
			offset:                50,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeRandom,
			initialExpOffset:      50,
			initialNumSeeks:       minSeeksForRandom,
			initialTotalReadBytes: 50 * 1024,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      minSeeksForRandom,
		},
		{
			name:                  "Non-contiguous random read is a seek",
			offset:                100,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeRandom,
			initialExpOffset:      50,
			initialNumSeeks:       minSeeksForRandom,
			initialTotalReadBytes: 50 * 1024,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      minSeeksForRandom + 1,
		},
		{
			name:                  "Switches to random read after enough seeks",
			offset:                50 + maxReadSize + 1,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       minSeeksForRandom - 1,
			initialTotalReadBytes: 1000,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      minSeeksForRandom,
		},
		{
			name:                  "Switches back to sequential with high average read bytes",
			offset:                100,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeRandom,
			initialExpOffset:      50,
			initialNumSeeks:       minSeeksForRandom,
			initialTotalReadBytes: maxReadSize * (minSeeksForRandom + 1),
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      minSeeksForRandom + 1,
		},
		{
			name:                  "Seek recorded: sequential large forward jump",
			offset:                50 + maxReadSize + 1,
			seekRecorded:          true,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       0,
			initialTotalReadBytes: 50 * 1024,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      0, // Not incremented
		},
		{
			name:                  "Seek recorded: sequential backward jump",
			offset:                49,
			seekRecorded:          true,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       1,
			initialTotalReadBytes: 50 * 1024,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      1, // Not incremented
		},
		{
			name:                  "Seek recorded: non-contiguous random read",
			offset:                100,
			seekRecorded:          true,
			initialReadType:       metrics.ReadTypeRandom,
			initialExpOffset:      50,
			initialNumSeeks:       minSeeksForRandom,
			initialTotalReadBytes: 50 * 1024,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      minSeeksForRandom, // Not incremented
		},
		{
			name:                  "Seek recorded: does not switch to random",
			offset:                50 + maxReadSize + 1,
			seekRecorded:          true,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       minSeeksForRandom - 1,
			initialTotalReadBytes: 1000,
			expectedReadType:      metrics.ReadTypeSequential, // Does not switch
			expectedNumSeeks:      minSeeksForRandom - 1,      // Not incremented
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.gcsReader.readType.Store(tc.initialReadType)
			t.gcsReader.expectedOffset.Store(tc.initialExpOffset)
			t.gcsReader.seeks.Store(tc.initialNumSeeks)
			t.gcsReader.totalReadBytes.Store(tc.initialTotalReadBytes)

			readInfo := t.gcsReader.getReadInfo(tc.offset, tc.seekRecorded)
			assert.Equal(t.T(), tc.expectedReadType, readInfo.readType, "Read type mismatch")
			assert.Equal(t.T(), tc.expectedNumSeeks, t.gcsReader.seeks.Load(), "Number of seeks mismatch")
		})
	}
}

// Validates:
// 1. No change in ReadAt behavior based inactiveStreamTimeout readConfig.
// 2. Valid timeout readConfig creates inactiveTimeoutReader instance of storage.Reader.
func (t *gcsReaderTest) Test_ReadAt_WithAndWithoutReadConfig() {
	testCases := []struct {
		name                        string
		config                      *cfg.Config
		expectInactiveTimeoutReader bool
	}{
		{
			name:                        "WithoutReadConfig",
			config:                      nil,
			expectInactiveTimeoutReader: false,
		},
		{
			name:                        "WithReadConfigAndZeroTimeout",
			config:                      &cfg.Config{Read: cfg.ReadConfig{InactiveStreamTimeout: 0}},
			expectInactiveTimeoutReader: false,
		},
		{
			name:                        "WithReadConfigAndPositiveTimeout",
			config:                      &cfg.Config{Read: cfg.ReadConfig{InactiveStreamTimeout: 10 * time.Millisecond}},
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

			t.gcsReader.rangeReader.config = tc.config
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
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: false}).Times(3)

			objectData, err := t.readAt(readOffset, int64(readLength))
			// The reader should be active as partial data read from the requested range.
			t.mockBucket.AssertExpectations(t.T())
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), readLength, objectData.Size)
			assert.NotNil(t.T(), t.gcsReader.rangeReader.reader, "Reader should be active as partial data read from the requested range.")
			assert.NotNil(t.T(), t.gcsReader.rangeReader.cancel)
			assert.Equal(t.T(), int64(readLength), t.gcsReader.rangeReader.start)
			assert.Equal(t.T(), int64(t.object.Size), t.gcsReader.rangeReader.limit)
			_, isInactiveTimeoutReader := t.gcsReader.rangeReader.reader.(*gcsx.InactiveTimeoutReader)
			assert.Equal(t.T(), tc.expectInactiveTimeoutReader, isInactiveTimeoutReader)
		})
	}
}

// This test validates the bug fix where seeks are not updated correctly in case of zonal bucket random reads (b/410904634).
func (t *gcsReaderTest) Test_ReadAt_ValidateZonalRandomReads() {
	t.gcsReader.rangeReader.reader = nil
	t.gcsReader.mrr.isMRDInUse.Store(false)
	t.gcsReader.seeks.Store(0)
	t.gcsReader.rangeReader.readType = metrics.ReadTypeSequential
	t.gcsReader.expectedOffset.Store(0)
	t.gcsReader.totalReadBytes.Store(0)
	t.object.Size = 20 * MiB
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true})
	testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
	fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{}, &cfg.Config{})
	assert.Nil(t.T(), err, "Error in creating MRDWrapper")
	t.gcsReader.mrr.mrdWrapper = &fakeMRDWrapper
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(&fake.FakeReader{ReadCloser: getReadCloser(testContent)}, nil).Twice()
	buf := make([]byte, 3*MiB)

	// Sequential read #1
	_, err = t.gcsReader.ReadAt(t.ctx, buf, 13*MiB)
	assert.NoError(t.T(), err)
	// Random read #1
	seeks := 1
	_, err = t.gcsReader.ReadAt(t.ctx, buf, 12*MiB)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), uint64(seeks), t.gcsReader.seeks.Load())

	readRanges := [][]int{{11 * MiB, 15 * MiB}, {12 * MiB, 14 * MiB}, {10 * MiB, 12 * MiB}, {9 * MiB, 11 * MiB}, {8 * MiB, 10 * MiB}}
	// Series of random reads to check if seeks are updated correctly and MRD is invoked always
	for _, readRange := range readRanges {
		seeks++
		t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond))
		buf := make([]byte, readRange[1]-readRange[0])

		_, err := t.gcsReader.ReadAt(t.ctx, buf, int64(readRange[0]))

		assert.NoError(t.T(), err)
		assert.Equal(t.T(), uint64(seeks), t.gcsReader.seeks.Load())
		assert.Equal(t.T(), metrics.ReadTypeRandom, t.gcsReader.readType.Load())
		assert.Equal(t.T(), int64(readRange[1]), t.gcsReader.expectedOffset.Load())
	}
}

func (t *gcsReaderTest) Test_ReadAt_MRDShortReadOnZonal() {
	t.object.Size = 200
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true})
	testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
	fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapper(t.mockBucket, t.object, &cfg.Config{})
	require.NoError(t.T(), err)
	t.gcsReader.mrr.mrdWrapper = &fakeMRDWrapper
	// First call to NewMultiRangeDownloader will return a short read.
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithShortRead(t.object, testContent), nil).Once()
	// Second call for retry will return the full content.
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloader(t.object, testContent), nil).Once()
	t.gcsReader.readType.Store(metrics.ReadTypeRandom)
	buf := make([]byte, t.object.Size)

	// Act
	readerResponse, err := t.gcsReader.ReadAt(t.ctx, buf, 0)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), int(t.object.Size), readerResponse.Size)
	assert.Equal(t.T(), testContent, buf)
	assert.Equal(t.T(), int64(t.object.Size), t.gcsReader.expectedOffset.Load())
	t.mockBucket.AssertExpectations(t.T())
}

func (t *gcsReaderTest) Test_ReadAt_ParallelRandomReads() {
	// Setup
	t.gcsReader.seeks.Store(minSeeksForRandom)
	t.gcsReader.readType.Store(metrics.ReadTypeRandom)
	t.object.Size = 20 * MiB
	testContent := testUtil.GenerateRandomBytes(int(t.object.Size))

	// Mock bucket and MRD
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true})
	fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapper(t.mockBucket, t.object, &cfg.Config{})
	require.NoError(t.T(), err)
	t.gcsReader.mrr.mrdWrapper = &fakeMRDWrapper
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloader(t.object, testContent), nil)

	// Parallel reads
	tasks := []struct {
		offset int64
		size   int
	}{
		{0, 1 * MiB},
		{2 * MiB, 2 * MiB},
		{5 * MiB, 1 * MiB},
		{10 * MiB, 5 * MiB},
	}

	var wg sync.WaitGroup
	var totalBytesReadFromTasks uint64

	for _, task := range tasks {
		wg.Add(1)
		totalBytesReadFromTasks += uint64(task.size)
		go func(offset int64, size int) {
			defer wg.Done()
			buf := make([]byte, size)
			// Each goroutine gets its own context.
			ctx := context.Background()
			objData, err := t.gcsReader.ReadAt(ctx, buf, offset)

			require.NoError(t.T(), err)
			require.Equal(t.T(), size, objData.Size)
			require.Equal(t.T(), testContent[offset:offset+int64(size)], buf)
		}(task.offset, task.size)
	}

	wg.Wait()

	// Validation
	assert.Equal(t.T(), totalBytesReadFromTasks, t.gcsReader.totalReadBytes.Load())
}
