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
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/cfg"
	"github.com/vipnydav/gcsfuse/v3/internal/cache/util"
	"github.com/vipnydav/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/vipnydav/gcsfuse/v3/internal/gcsx"
	"github.com/vipnydav/gcsfuse/v3/internal/storage"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/fake"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/gcs"
	testUtil "github.com/vipnydav/gcsfuse/v3/internal/util"
	"github.com/vipnydav/gcsfuse/v3/metrics"
)

const (
	fakeHandleData = "fake-handle"
	testObject     = "testObject"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type rangeReaderTest struct {
	suite.Suite
	ctx         context.Context
	object      *gcs.MinObject
	mockBucket  *storage.TestifyMockBucket
	rangeReader *RangeReader
}

func TestRangeReaderTestSuite(t *testing.T) {
	suite.Run(t, new(rangeReaderTest))
}

func (t *rangeReaderTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       testObject,
		Size:       17,
		Generation: 1234,
	}
	t.mockBucket = new(storage.TestifyMockBucket)
	t.rangeReader = NewRangeReader(t.object, t.mockBucket, nil, metrics.NewNoopMetrics())
	t.ctx = context.Background()
}

func (t *rangeReaderTest) TearDownTest() {
	t.rangeReader.destroy()
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func getReadCloser(content []byte) io.ReadCloser {
	r := bytes.NewReader(content)
	rc := io.NopCloser(r)
	return rc
}

func getReader(size int) *fake.FakeReader {
	testContent := testUtil.GenerateRandomBytes(size)
	return &fake.FakeReader{
		ReadCloser: getReadCloser(testContent),
		Handle:     []byte(fakeHandleData),
	}
}

func (t *rangeReaderTest) readAt(dst []byte, offset int64) (gcsx.ReadResponse, error) {
	req := &gcsx.GCSReaderRequest{
		Offset:    offset,
		EndOffset: offset + int64(len(dst)),
		Buffer:    dst,
	}
	t.rangeReader.checkInvariants()
	defer t.rangeReader.checkInvariants()
	return t.rangeReader.ReadAt(t.ctx, req)
}

func (t *rangeReaderTest) mockNewReaderWithHandleCallForTestBucket(start uint64, limit uint64, rd gcs.StorageReader) {
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(rg *gcs.ReadObjectRequest) bool {
		return rg != nil && (*rg.Range).Start == start && (*rg.Range).Limit == limit
	})).Return(rd, nil).Once()
}

////////////////////////////////////////////////////////////////////////
// Blocking reader
////////////////////////////////////////////////////////////////////////

// A reader that blocks until a channel is closed, then returns an error.
type blockingReader struct {
	c chan struct{}
}

func (br *blockingReader) Read([]byte) (int, error) {
	<-br.c
	return 0, errors.New("blockingReader")
}

////////////////////////////////////////////////////////////////////////
// Counting closer
////////////////////////////////////////////////////////////////////////

type countingCloser struct {
	io.Reader
	closeCount int
}

func (cc *countingCloser) Close() (err error) {
	cc.closeCount++
	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *rangeReaderTest) Test_NewRangeReader() {
	object := &gcs.MinObject{
		Name:       testObject,
		Size:       30,
		Generation: 4321,
	}

	reader := NewRangeReader(object, t.mockBucket, nil, metrics.NewNoopMetrics())

	assert.Equal(t.T(), object, reader.object)
	assert.Equal(t.T(), t.mockBucket, reader.bucket)
	assert.Equal(t.T(), metrics.NewNoopMetrics(), reader.metricHandle)
	assert.Equal(t.T(), int64(-1), reader.start)
	assert.Equal(t.T(), int64(-1), reader.limit)
}

func (t *rangeReaderTest) Test_CheckInvariants() {
	tests := []struct {
		name        string
		setup       func() *RangeReader
		shouldPanic bool
	}{
		{
			name: "valid no reader",
			setup: func() *RangeReader {
				return &RangeReader{
					start:  0,
					limit:  10,
					reader: nil,
					cancel: nil,
				}
			},
			shouldPanic: false,
		},
		{
			name: "reader without cancel",
			setup: func() *RangeReader {
				t.rangeReader.reader = getReader(2)
				return &RangeReader{
					start:  0,
					limit:  10,
					reader: t.rangeReader.reader,
					cancel: nil,
				}
			},
			shouldPanic: true,
		},
		{
			name: "cancel without reader",
			setup: func() *RangeReader {
				return &RangeReader{
					start:  0,
					limit:  10,
					reader: nil,
					cancel: func() {},
				}
			},
			shouldPanic: true,
		},
		{
			name: "invalid range",
			setup: func() *RangeReader {
				return &RangeReader{
					start:  20,
					limit:  10,
					reader: nil,
					cancel: nil,
				}
			},
			shouldPanic: true,
		},
		{
			name: "negative limit with valid reader",
			setup: func() *RangeReader {
				t.rangeReader.reader = getReader(2)
				return &RangeReader{
					start:  -10,
					limit:  -5,
					reader: t.rangeReader.reader,
					cancel: func() {},
				}
			},
			shouldPanic: true,
		},
		{
			name: "negative limit with nil reader",
			setup: func() *RangeReader {
				return &RangeReader{
					start:  -10,
					limit:  -5,
					reader: nil,
					cancel: nil,
				}
			},
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func() {
			rr := tt.setup()
			if tt.shouldPanic {
				assert.Panics(t.T(), func() { rr.checkInvariants() }, "Expected panic")
			} else {
				assert.NotPanics(t.T(), func() { rr.checkInvariants() }, "Expected no panic")
			}
		})
	}
}

func (t *rangeReaderTest) Test_Destroy_NonNilReader() {
	t.rangeReader.reader = getReader(2)

	t.rangeReader.destroy()

	assert.Nil(t.T(), t.rangeReader.reader)
	assert.Nil(t.T(), t.rangeReader.cancel)
	assert.Equal(t.T(), []byte(fakeHandleData), t.rangeReader.readHandle)
}

func (t *rangeReaderTest) Test_ReadAt_ReadFailsWithTimeoutError() {
	content := "xxx"
	r := iotest.OneByteReader(iotest.TimeoutReader(strings.NewReader(content)))
	rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}
	t.mockNewReaderWithHandleCallForTestBucket(0, uint64(len(content)), rc)
	buf := make([]byte, len(content))

	readResponse, err := t.readAt(buf, 0)

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "timeout")
	assert.Zero(t.T(), readResponse.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *rangeReaderTest) Test_ReadAt_SuccessfulRead() {
	offset := int64(0)
	size := int64(5)
	content := []byte("hello world")
	r := &fake.FakeReader{ReadCloser: getReadCloser(content)}
	t.mockNewReaderWithHandleCallForTestBucket(uint64(offset), uint64(offset+size), r)
	buf := make([]byte, size)

	resp, err := t.readAt(buf, offset)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), int(size), resp.Size)
	assert.Equal(t.T(), content[:size], buf[:resp.Size])
	t.mockBucket.AssertExpectations(t.T())
}

func (t *rangeReaderTest) Test_ReadAt_PartialReadWithEOF() {
	offset := int64(0)
	size := int64(10)                                       // Shorter than requested
	partialReader := io.NopCloser(iotest.ErrReader(io.EOF)) // Simulates early EOF
	r := &fake.FakeReader{ReadCloser: partialReader}
	t.mockNewReaderWithHandleCallForTestBucket(uint64(offset), uint64(offset+size), r)
	buf := make([]byte, size)

	resp, err := t.readAt(buf, offset)

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "reader returned early by skipping")
	assert.Zero(t.T(), resp.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *rangeReaderTest) Test_ReadAt_StartReadNotFound() {
	offset := int64(0)
	size := int64(5)
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, &gcs.NotFoundError{}).Once()

	resp, err := t.readAt(make([]byte, size), offset)

	var fcErr *gcsfuse_errors.FileClobberedError
	assert.ErrorAs(t.T(), err, &fcErr)
	assert.Zero(t.T(), resp.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *rangeReaderTest) Test_ReadAt_StartReadUnexpectedError() {
	offset := int64(0)
	size := int64(5)
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, errors.New("network error")).Once()

	resp, err := t.readAt(make([]byte, size), offset)

	assert.Error(t.T(), err)
	assert.Zero(t.T(), resp.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *rangeReaderTest) Test_invalidateReaderIfMisalignedOrTooSmall() {
	tests := []struct {
		name               string
		readerSetup        func()
		offset             int64
		bufferSize         int
		expectIncreaseSeek bool
		expectReaderNil    bool
	}{
		{
			name: "InvalidateReaderDueToTooSmall",
			readerSetup: func() {
				t.rangeReader.reader = &fake.FakeReader{ReadCloser: getReader(100)}
				t.rangeReader.start = 200
				t.rangeReader.limit = 250 // too small
				t.rangeReader.object.Size = 500
			},
			offset:             200,
			bufferSize:         100,
			expectIncreaseSeek: false,
			expectReaderNil:    true,
		},
		{
			name: "KeepReaderIfValid",
			readerSetup: func() {
				t.rangeReader.reader = &fake.FakeReader{ReadCloser: getReader(100)}
				t.rangeReader.start = 200
				t.rangeReader.limit = 400
			},
			offset:             200,
			bufferSize:         100,
			expectIncreaseSeek: false,
			expectReaderNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func() {
			tt.readerSetup()

			t.rangeReader.invalidateReaderIfMisalignedOrTooSmall(tt.offset, tt.offset+int64(tt.bufferSize))

			if tt.expectReaderNil {
				assert.Nil(t.T(), t.rangeReader.reader, "rangeReader.reader should be nil")
			} else {
				assert.NotNil(t.T(), t.rangeReader.reader, "rangeReader.reader should not be nil")
			}
		})
	}
}

func (t *rangeReaderTest) Test_ReadFromRangeReader_WhenReaderReturnedMoreData() {
	testCases := []struct {
		name       string
		readHandle []byte
	}{
		{
			name:       "GCSReturnedReadHandle",
			readHandle: []byte(fakeHandleData),
		},
		{
			name:       "GCSReturnedNoReadHandle",
			readHandle: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rangeReader.start = 0
			t.rangeReader.limit = 6
			testContent := testUtil.GenerateRandomBytes(8)
			t.rangeReader.reader = &fake.FakeReader{
				ReadCloser: getReadCloser(testContent),
				Handle:     tc.readHandle,
			}
			t.rangeReader.cancel = func() {}

			n, err := t.rangeReader.readFromRangeReader(t.ctx, make([]byte, 10), 0, 10, metrics.ReadTypeUnknown)

			assert.Error(t.T(), err)
			assert.Zero(t.T(), n)
			assert.Nil(t.T(), t.rangeReader.reader)
			assert.Equal(t.T(), int64(-1), t.rangeReader.start)
			assert.Equal(t.T(), int64(-1), t.rangeReader.limit)
			assert.Equal(t.T(), tc.readHandle, t.rangeReader.readHandle)
		})
	}
}

func (t *rangeReaderTest) Test_ReadAt_PropagatesCancellation() {
	t.rangeReader = NewRangeReader(t.object, t.mockBucket, &cfg.Config{FileSystem: cfg.FileSystemConfig{IgnoreInterrupts: false}}, metrics.NewNoopMetrics())
	// Set up a blocking reader
	finishRead := make(chan struct{})
	blocking := &blockingReader{c: finishRead}
	rc := io.NopCloser(blocking)
	// Assign it to the rangeReader
	t.rangeReader.reader = &fake.FakeReader{ReadCloser: rc}
	t.rangeReader.start = 0
	t.rangeReader.limit = 2
	// Track cancel invocation
	cancelCalled := make(chan struct{})
	t.rangeReader.cancel = func() { close(cancelCalled) }
	// Controlled context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Channel to track read completion
	readReturned := make(chan struct{})

	go func() {
		_, _ = t.rangeReader.ReadAt(ctx, &gcsx.GCSReaderRequest{
			Buffer:    make([]byte, 2),
			Offset:    0,
			EndOffset: 2,
		})
		close(readReturned)
	}()

	// Wait a bit to ensure ReadAt is blocking
	select {
	case <-readReturned:
		t.T().Fatal("Read returned early — cancellation did not propagate properly.")
	case <-time.After(10 * time.Millisecond):
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

func (t *rangeReaderTest) Test_ReadAt_DoesntPropagateCancellationAfterReturning() {
	// Set up a reader that will return three bytes.
	content := "xyz"
	t.rangeReader.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte(content))}
	t.rangeReader.start = 0
	t.rangeReader.limit = 3
	// Snoop on when cancel is called.
	cancelCalled := make(chan struct{})
	t.rangeReader.cancel = func() { close(cancelCalled) }
	ctx, cancel := context.WithCancel(context.Background())
	bufSize := 2
	buf := make([]byte, bufSize)

	// Successfully read two bytes using a context whose cancellation we control.
	readResponse, err := t.rangeReader.ReadAt(ctx, &gcsx.GCSReaderRequest{
		Buffer:    buf,
		Offset:    0,
		EndOffset: 2,
	})

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), bufSize, readResponse.Size)
	assert.Equal(t.T(), content[:bufSize], string(buf[:readResponse.Size]))
	// If we cancel the calling context now, it should not cause the underlying
	// read context to be cancelled.
	cancel()
	select {
	case <-time.After(10 * time.Millisecond):
	case <-cancelCalled:
		t.T().Fatal("Read context unexpectedly cancelled")
	}
}

func (t *rangeReaderTest) Test_ReadFromRangeReader_WhenAllDataFromReaderIsRead() {
	testCases := []struct {
		name       string
		readHandle []byte
	}{
		{
			name:       "GCSReturnedReadHandle",
			readHandle: []byte(fakeHandleData),
		},
		{
			name:       "GCSReturnedNoReadHandle",
			readHandle: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rangeReader.start = 4
			t.rangeReader.limit = 10
			t.object.Size = 10
			dataSize := 6
			testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
			rc := &fake.FakeReader{
				ReadCloser: getReadCloser(testContent),
				Handle:     tc.readHandle,
			}
			t.rangeReader.reader = rc
			t.rangeReader.cancel = func() {}
			buf := make([]byte, dataSize)

			n, err := t.rangeReader.readFromRangeReader(t.ctx, buf, 4, 10, metrics.ReadTypeUnknown)

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), dataSize, n)
			// Verify the reader state.
			assert.Nil(t.T(), t.rangeReader.reader)
			assert.Nil(t.T(), t.rangeReader.cancel)
			assert.Equal(t.T(), int64(10), t.rangeReader.start)
			assert.Equal(t.T(), int64(10), t.rangeReader.limit)
			assert.Equal(t.T(), tc.readHandle, t.rangeReader.readHandle)
		})
	}
}

func (t *rangeReaderTest) Test_ReadFromRangeReader_WhenReaderHasLessDataThanRequested() {
	testCases := []struct {
		name       string
		readHandle []byte
	}{
		{
			name:       "GCSReturnedReadHandle",
			readHandle: []byte(fakeHandleData),
		},
		{
			name:       "GCSReturnedNoReadHandle",
			readHandle: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rangeReader.start = 0
			t.rangeReader.limit = 6
			dataSize := 6
			testContent := testUtil.GenerateRandomBytes(dataSize)
			rc := &fake.FakeReader{
				ReadCloser: getReadCloser(testContent),
				Handle:     tc.readHandle,
			}
			t.rangeReader.reader = rc
			t.rangeReader.cancel = func() {}
			buf := make([]byte, 10)

			n, err := t.rangeReader.readFromRangeReader(t.ctx, buf, 0, 10, metrics.ReadTypeUnknown)

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), dataSize, n)
			// Verify the reader state.
			assert.Nil(t.T(), t.rangeReader.reader)
			assert.Nil(t.T(), t.rangeReader.cancel)
			assert.Equal(t.T(), int64(dataSize), t.rangeReader.start)
			assert.Equal(t.T(), int64(dataSize), t.rangeReader.limit)
			assert.Equal(t.T(), tc.readHandle, t.rangeReader.readHandle)
		})
	}
}

func (t *rangeReaderTest) Test_ReadAt_ReaderNotExhausted() {
	// Set up a reader that has three bytes left to give.
	content := "abc"
	cc := &countingCloser{
		Reader: strings.NewReader(content),
	}
	rc := &fake.FakeReader{ReadCloser: cc}
	t.rangeReader.reader = rc
	var offset int64 = 1
	t.rangeReader.start = offset
	t.rangeReader.limit = 4
	t.rangeReader.cancel = func() {}
	var bufSize int64 = 2
	buf := make([]byte, bufSize)

	// Read two bytes.
	resp, err := t.readAt(buf, offset)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), content[:bufSize], string(buf[:resp.Size]))
	assert.Zero(t.T(), cc.closeCount)
	assert.Equal(t.T(), rc, t.rangeReader.reader)
	assert.Equal(t.T(), offset+bufSize, t.rangeReader.start)
}

func (t *rangeReaderTest) Test_ReadAt_ShortRead() {
	offset := int64(0)
	size := int64(10)
	// Create a reader that will return less data than requested
	shortContent := []byte("hello")
	r := &fake.FakeReader{ReadCloser: getReadCloser(shortContent)}
	t.mockNewReaderWithHandleCallForTestBucket(uint64(offset), uint64(offset+size), r)
	buf := make([]byte, size)

	resp, err := t.readAt(buf, offset)

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "reader returned early by skipping 5 bytes: short read")
	assert.ErrorIs(t.T(), err, util.ErrShortRead)
	assert.Zero(t.T(), resp.Size)
	t.mockBucket.AssertExpectations(t.T())
}

// Write a unit test to force recreate a reader and verify that the reader was force created and read was successful
func (t *rangeReaderTest) Test_ReadAt_ForceCreateReader() {
	offset := int64(0)
	size := int64(10)
	readSize := int64(3)
	content1 := []byte("first-content")
	content2 := []byte("second-content")

	// 1. First reader
	r1 := &fake.FakeReader{ReadCloser: getReadCloser(content1)}
	t.mockNewReaderWithHandleCallForTestBucket(uint64(offset), uint64(offset+size), r1)

	// 2. Read with forceCreateReader = false (default)
	req1 := &gcsx.GCSReaderRequest{
		Offset:    offset,
		EndOffset: offset + size,
		Buffer:    make([]byte, readSize),
	}
	resp1, err := t.rangeReader.ReadAt(t.ctx, req1)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), int(readSize), resp1.Size)
	assert.Equal(t.T(), content1[:readSize], req1.Buffer)
	assert.NotNil(t.T(), t.rangeReader.reader) // Reader should be active
	firstReader := t.rangeReader.reader

	// 3. Second reader (will be created due to forceCreateReader = true)
	r2 := &fake.FakeReader{ReadCloser: getReadCloser(content2)}
	t.mockNewReaderWithHandleCallForTestBucket(uint64(offset+readSize), uint64(offset+size), r2)
	readsize2 := int64(4)

	// 4. Read with forceCreateReader = true. The existing reader can serve this
	// request, but it will be discarded because ForceCreateReader is true.
	req2 := &gcsx.GCSReaderRequest{
		Offset:            offset + readSize,
		EndOffset:         offset + size,
		Buffer:            make([]byte, readsize2),
		ForceCreateReader: true,
	}
	resp2, err := t.rangeReader.ReadAt(t.ctx, req2)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), int(readsize2), resp2.Size)
	assert.Equal(t.T(), content2[:readsize2], req2.Buffer)
	assert.NotNil(t.T(), t.rangeReader.reader) // New reader should not be nil
	secondReader := t.rangeReader.reader
	assert.NotEqual(t.T(), firstReader, secondReader)
	t.mockBucket.AssertExpectations(t.T())
}
