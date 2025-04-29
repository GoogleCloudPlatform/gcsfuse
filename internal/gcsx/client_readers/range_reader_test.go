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

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	testUtil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
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
	t.rangeReader = NewRangeReader(t.object, t.mockBucket, common.NewNoopMetrics())
	t.ctx = context.Background()
}

func (t *rangeReaderTest) TearDown() {
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

func (t *rangeReaderTest) readAt(offset int64, size int64) (gcsx.ReaderResponse, error) {
	req := &gcsx.GCSReaderRequest{
		Offset:    offset,
		EndOffset: offset + size,
		Buffer:    make([]byte, size),
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
	// The setup instantiates rangeReader with NewRangeReader.
	assert.Equal(t.T(), t.object, t.rangeReader.object)
	assert.Equal(t.T(), t.mockBucket, t.rangeReader.bucket)
	assert.Equal(t.T(), common.NewNoopMetrics(), t.rangeReader.metricHandle)
	assert.Equal(t.T(), int64(-1), t.rangeReader.start)
	assert.Equal(t.T(), int64(-1), t.rangeReader.limit)
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

	readerResponse, err := t.readAt(0, int64(len(content)))

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "timeout")
	assert.Zero(t.T(), readerResponse.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *rangeReaderTest) Test_ReadAt_SuccessfulRead() {
	offset := int64(0)
	size := int64(5)
	content := []byte("hello world")
	r := &fake.FakeReader{ReadCloser: getReadCloser(content)}
	t.mockNewReaderWithHandleCallForTestBucket(uint64(offset), uint64(offset+size), r)

	resp, err := t.readAt(offset, size)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), int(size), resp.Size)
	assert.Equal(t.T(), content[:size], resp.DataBuf)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *rangeReaderTest) Test_ReadAt_PartialReadWithEOF() {
	offset := int64(0)
	size := int64(10)                                       // Shorter than requested
	partialReader := io.NopCloser(iotest.ErrReader(io.EOF)) // Simulates early EOF
	r := &fake.FakeReader{ReadCloser: partialReader}
	t.mockNewReaderWithHandleCallForTestBucket(uint64(offset), uint64(offset+size), r)

	resp, err := t.readAt(offset, size)

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "reader returned early by skipping")
	assert.Zero(t.T(), resp.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *rangeReaderTest) Test_ReadAt_StartReadNotFound() {
	offset := int64(0)
	size := int64(5)
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, &gcs.NotFoundError{}).Once()

	resp, err := t.readAt(offset, size)

	var fcErr *gcsfuse_errors.FileClobberedError
	assert.ErrorAs(t.T(), err, &fcErr)
	assert.Zero(t.T(), resp.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *rangeReaderTest) Test_ReadAt_StartReadUnexpectedError() {
	offset := int64(0)
	size := int64(5)
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, errors.New("network error")).Once()

	resp, err := t.readAt(offset, size)

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
			name: "InvalidateReaderDueToWrongOffset",
			readerSetup: func() {
				t.rangeReader.reader = &fake.FakeReader{ReadCloser: getReader(100)}
				t.rangeReader.start = 50 // misaligned
				t.rangeReader.limit = 1000
			},
			offset:             200,
			bufferSize:         100,
			expectIncreaseSeek: true,
			expectReaderNil:    true,
		},
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

			result := t.rangeReader.invalidateReaderIfMisalignedOrTooSmall(tt.offset, make([]byte, tt.bufferSize))

			assert.Equal(t.T(), tt.expectIncreaseSeek, result, "invalidateReaderIfMisalignedOrTooSmall() result")
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
			readHandle: []byte("fake-handle"),
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
			rc := &fake.FakeReader{
				ReadCloser: getReadCloser(testContent),
				Handle:     tc.readHandle,
			}
			t.rangeReader.reader = rc
			t.rangeReader.cancel = func() {}

			n, err := t.rangeReader.readFromRangeReader(t.ctx, make([]byte, 10), 0, 10, "unhandled")

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
	t.rangeReader.start = 1
	t.rangeReader.limit = 4
	// Snoop on when cancel is called.
	cancelCalled := make(chan struct{})
	t.rangeReader.cancel = func() { close(cancelCalled) }
	ctx, cancel := context.WithCancel(context.Background())
	bufSize := 2

	// Successfully read two bytes using a context whose cancellation we control.
	readerResponse, err := t.rangeReader.ReadAt(ctx, &gcsx.GCSReaderRequest{
		Buffer:    make([]byte, bufSize),
		Offset:    0,
		EndOffset: 2,
	})

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), bufSize, readerResponse.Size)
	assert.Equal(t.T(), content[:bufSize], string(readerResponse.DataBuf[:readerResponse.Size]))
	// If we cancel the calling context now, it should not cause the underlying
	// read context to be cancelled.
	cancel()
	select {
	case <-time.After(10 * time.Millisecond):
	case <-cancelCalled:
		t.T().Fatal("Read context unexpectedly cancelled")
	}
}

func (t *rangeReaderTest) Test_ReadAt_ReaderExhaustedReadFinished() {
	r := &countingCloser{Reader: getReader(4)}
	t.rangeReader.reader = &fake.FakeReader{ReadCloser: r}
	var offset int64 = 0
	t.rangeReader.start = offset
	t.rangeReader.limit = 2
	t.rangeReader.cancel = func() {}
	var bufSize int64 = 2

	// The reader's start becomes equal to the limit.
	resp, err := t.readAt(offset, bufSize)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), t.rangeReader.limit, t.rangeReader.start)
	assert.Equal(t.T(), 1, r.closeCount)
	assert.Equal(t.T(), int(bufSize), resp.Size)
}

func (t *rangeReaderTest) Test_ReadAt_ReaderNotExhausted() {
	// Set up a reader that has three bytes left to give.
	content := "abc"
	cc := &countingCloser{
		Reader: strings.NewReader(content),
	}
	rc := &fake.FakeReader{ReadCloser: cc}
	t.rangeReader.reader = &fake.FakeReader{ReadCloser: cc}
	var offset int64 = 1
	t.rangeReader.start = offset
	t.rangeReader.limit = 4
	t.rangeReader.cancel = func() {}
	var bufSize int64 = 2

	// Read two bytes.
	resp, err := t.readAt(offset, bufSize)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), content[:bufSize], string(resp.DataBuf[:resp.Size]))
	assert.Zero(t.T(), cc.closeCount)
	assert.Equal(t.T(), rc, t.rangeReader.reader)
	assert.Equal(t.T(), offset+bufSize, t.rangeReader.start)
}

func (t *rangeReaderTest) Test_ReadAt_EOFWithReaderNilClearsError() {
	partialReader := io.NopCloser(iotest.ErrReader(io.ErrUnexpectedEOF)) // Simulates early EOF
	r := &fake.FakeReader{ReadCloser: partialReader}
	t.rangeReader.reader = &fake.FakeReader{ReadCloser: r}
	var offset int64 = 2
	t.rangeReader.start = offset
	t.rangeReader.limit = 2
	t.rangeReader.cancel = func() {}
	var bufSize int64 = 2

	resp, err := t.readAt(offset, bufSize)

	assert.NoError(t.T(), err)
	assert.Nil(t.T(), t.rangeReader.reader)
	assert.Zero(t.T(), resp.Size)
}
