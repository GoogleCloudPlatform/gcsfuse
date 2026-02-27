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

package client_readers

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
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

func (t *gcsReaderTest) readAt(ctx context.Context, req *gcsx.ReadRequest) gcsx.ReadResponse {
	t.gcsReader.CheckInvariants()
	defer t.gcsReader.CheckInvariants()
	readInfo := t.gcsReader.readTypeClassifier.GetReadInfo(req.Offset, false)
	resp, err := t.gcsReader.ReadAt(ctx, &gcsx.ReadRequest{
		Buffer:   req.Buffer,
		Offset:   req.Offset,
		ReadInfo: readInfo,
	})
	require.NoError(t.T(), err)
	t.gcsReader.readTypeClassifier.RecordRead(req.Offset, int64(resp.Size))
	return resp
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
		MetricHandle:       metrics.NewNoopMetrics(),
		MrdWrapper:         nil,
		Config:             nil,
		ReadTypeClassifier: gcsx.NewReadTypeClassifier(int64(sequentialReadSizeInMb), 0),
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
		MetricHandle:       metrics.NewNoopMetrics(),
		MrdWrapper:         nil,
		Config:             nil,
		ReadTypeClassifier: gcsx.NewReadTypeClassifier(sequentialReadSizeInMb, 0),
	})

	assert.Equal(t.T(), object, gcsReader.object)
	assert.Equal(t.T(), t.mockBucket, gcsReader.bucket)
	assert.True(t.T(), t.gcsReader.readTypeClassifier.IsReadSequential())
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

			_, err := t.gcsReader.ReadAt(t.ctx, &gcsx.ReadRequest{
				Buffer: buf,
				Offset: int64(tc.start),
			})

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
	buf := make([]byte, requestSize)

	readResponse := t.readAt(t.ctx, &gcsx.ReadRequest{
		Buffer: buf,
		Offset: 2,
	})

	assert.Equal(t.T(), rc, t.gcsReader.rangeReader.reader)
	assert.Equal(t.T(), requestSize, readResponse.Size)
	assert.Equal(t.T(), content, string(buf[:readResponse.Size]))
	assert.Equal(t.T(), int64(2+requestSize), t.gcsReader.readTypeClassifier.NextExpectedOffset())
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
	buf := make([]byte, requestSize)

	readResponse := t.readAt(t.ctx, &gcsx.ReadRequest{
		Buffer: buf,
		Offset: 0,
	})

	assert.Nil(t.T(), t.gcsReader.rangeReader.reader)
	assert.Equal(t.T(), int(t.object.Size), readResponse.Size)
	assert.Equal(t.T(), content, string(buf[:readResponse.Size]))
	assert.Equal(t.T(), int64(t.object.Size), t.gcsReader.readTypeClassifier.NextExpectedOffset())
	assert.Equal(t.T(), []byte(nil), t.gcsReader.rangeReader.readHandle)
}

func (t *gcsReaderTest) Test_ReadAt_ExistingReaderIsFine() {
	t.object.Size = 6
	content := "xxx"
	// Simulate an existing reader
	t.gcsReader.rangeReader.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte(content)), Handle: []byte("fake")}
	t.gcsReader.rangeReader.cancel = func() {}
	t.gcsReader.rangeReader.start = 2
	t.gcsReader.rangeReader.limit = 5
	requestSize := 3
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(3)
	buf := make([]byte, requestSize)

	readResponse := t.readAt(t.ctx, &gcsx.ReadRequest{
		Buffer: buf,
		Offset: 2,
	})

	assert.Equal(t.T(), 3, readResponse.Size)
	assert.Equal(t.T(), content, string(buf[:readResponse.Size]))
	assert.Equal(t.T(), int64(5), t.gcsReader.readTypeClassifier.NextExpectedOffset())
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
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(2)
			requestSize := 6
			buf := make([]byte, requestSize)

			readResponse := t.readAt(t.ctx, &gcsx.ReadRequest{
				Buffer: buf,
				Offset: 0,
			})

			t.mockBucket.AssertExpectations(t.T())
			assert.Nil(t.T(), t.gcsReader.rangeReader.reader)
			assert.Equal(t.T(), int(t.object.Size), readResponse.Size)
			assert.Equal(t.T(), content, string(buf[:readResponse.Size]))
			assert.Equal(t.T(), []byte(nil), t.gcsReader.rangeReader.readHandle)
		})
	}
}

func (t *gcsReaderTest) Test_ReadAt_PropagatesCancellation() {
	t.gcsReader = NewGCSReader(t.object, t.mockBucket, &GCSReaderConfig{
		MetricHandle:       metrics.NewNoopMetrics(),
		MrdWrapper:         nil,
		Config:             &cfg.Config{FileSystem: cfg.FileSystemConfig{IgnoreInterrupts: false}},
		ReadTypeClassifier: gcsx.NewReadTypeClassifier(sequentialReadSizeInMb, 0),
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
	req := &gcsx.ReadRequest{
		Buffer: make([]byte, 2),
		Offset: 0,
	}
	go func() {
		_, err = t.gcsReader.ReadAt(ctx, req)

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
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: false}).Times(2)
			buf := make([]byte, readLength)

			objectData := t.readAt(t.ctx, &gcsx.ReadRequest{
				Buffer: buf,
				Offset: readOffset,
			})

			t.mockBucket.AssertExpectations(t.T())
			assert.Equal(t.T(), readLength, objectData.Size)
			assert.Equal(t.T(), fakeReaderContent[:readLength], buf[:objectData.Size]) // Ensure buffer is populated correctly
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
	// Re-initialize GCSReader with initialOffset 13 MiB to force Random read type.
	t.gcsReader = NewGCSReader(t.object, t.mockBucket, &GCSReaderConfig{
		MetricHandle:       metrics.NewNoopMetrics(),
		MrdWrapper:         nil,
		Config:             nil,
		ReadTypeClassifier: gcsx.NewReadTypeClassifier(int64(sequentialReadSizeInMb), 13*MiB),
	})
	t.gcsReader.rangeReader.reader = nil
	t.gcsReader.mrr.isMRDInUse.Store(false)
	t.object.Size = 20 * MiB
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true})
	testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
	fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapper(t.mockBucket, t.object, &cfg.Config{}, nil)
	assert.NoError(t.T(), err, "Error in creating MRDWrapper")
	t.gcsReader.mrr.mrdWrapper = fakeMRDWrapper
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloader(t.object, testContent), nil).Once()

	readRanges := [][]int{{11 * MiB, 15 * MiB}, {12 * MiB, 14 * MiB}, {10 * MiB, 12 * MiB}, {9 * MiB, 11 * MiB}, {8 * MiB, 10 * MiB}}
	// Series of random reads to check if seeks are updated correctly and MRD is invoked always
	seeks := 0
	for _, readRange := range readRanges {
		buf := make([]byte, readRange[1]-readRange[0])

		t.readAt(t.ctx, &gcsx.ReadRequest{
			Buffer: buf,
			Offset: int64(readRange[0]),
		})

		assert.Equal(t.T(), uint64(seeks), t.gcsReader.readTypeClassifier.GetSeeks())
		assert.False(t.T(), t.gcsReader.readTypeClassifier.IsReadSequential())
		assert.Equal(t.T(), int64(readRange[1]), t.gcsReader.readTypeClassifier.NextExpectedOffset())
		seeks++
	}
}

func (t *gcsReaderTest) Test_ReadAt_MRDShortReadOnZonal() {
	// Re-initialize GCSReader with initialOffset 1 to force Random read type.
	t.gcsReader = NewGCSReader(t.object, t.mockBucket, &GCSReaderConfig{
		MetricHandle:       metrics.NewNoopMetrics(),
		MrdWrapper:         nil,
		Config:             nil,
		ReadTypeClassifier: gcsx.NewReadTypeClassifier(int64(sequentialReadSizeInMb), 1),
	})
	t.object.Size = 200
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true})
	testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
	fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapper(t.mockBucket, t.object, &cfg.Config{}, nil)
	require.NoError(t.T(), err)
	t.gcsReader.mrr.mrdWrapper = fakeMRDWrapper

	// First call to NewMultiRangeDownloader will return a short read, which will trigger a retry.
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithShortRead(t.object, testContent), nil).Once()
	// Second call for retry will return the full content.
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloader(t.object, testContent), nil).Once()
	buf := make([]byte, t.object.Size-1)

	// Act
	readResponse := t.readAt(t.ctx, &gcsx.ReadRequest{
		Buffer: buf,
		Offset: 1,
	})

	// Assert
	assert.Equal(t.T(), int(t.object.Size)-1, readResponse.Size)
	assert.Equal(t.T(), testContent[1:], buf)
	assert.Equal(t.T(), int64(t.object.Size), t.gcsReader.readTypeClassifier.NextExpectedOffset())
	t.mockBucket.AssertExpectations(t.T())
}

func (t *gcsReaderTest) Test_ReadAt_ParallelRandomReads() {
	// Re-initialize GCSReader with initialOffset 1 to force Random read type.
	t.gcsReader = NewGCSReader(t.object, t.mockBucket, &GCSReaderConfig{
		MetricHandle:       metrics.NewNoopMetrics(),
		MrdWrapper:         nil,
		Config:             nil,
		ReadTypeClassifier: gcsx.NewReadTypeClassifier(int64(sequentialReadSizeInMb), 1),
	})

	// Setup
	t.object.Size = 20 * MiB
	testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
	// Mock bucket and MRD
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true})
	fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapper(t.mockBucket, t.object, &cfg.Config{}, nil)
	require.NoError(t.T(), err)
	t.gcsReader.mrr.mrdWrapper = fakeMRDWrapper
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloader(t.object, testContent), nil)

	// Parallel reads
	tasks := []struct {
		offset int64
		size   int
	}{
		{1, 1 * MiB},
		{3 * MiB, 2 * MiB},
		{6 * MiB, 1 * MiB},
		{10 * MiB, 5 * MiB},
	}

	var wg sync.WaitGroup

	for _, task := range tasks {
		wg.Add(1)
		go func(offset int64, size int) {
			defer wg.Done()
			buf := make([]byte, size)
			// Each goroutine gets its own context.
			ctx := context.Background()
			objData := t.readAt(ctx, &gcsx.ReadRequest{
				Buffer: buf,
				Offset: offset,
			})

			require.Equal(t.T(), size, objData.Size)
			require.Equal(t.T(), testContent[offset:offset+int64(size)], buf)
		}(task.offset, task.size)
	}
	wg.Wait()
}
