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
	testutil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	fakeHandleData = "fake-handle"
	testObject     = "testObject"
)

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
	t.rangeReader.Destroy()
}

func getReadCloser(content []byte) io.ReadCloser {
	r := bytes.NewReader(content)
	rc := io.NopCloser(r)
	return rc
}

func getReader() *fake.FakeReader {
	testContent := testutil.GenerateRandomBytes(2)
	return &fake.FakeReader{
		ReadCloser: getReadCloser(testContent),
		Handle:     []byte(fakeHandleData),
	}
}

////////////////////////////////////////////////////////////////////////
// Blocking reader
////////////////////////////////////////////////////////////////////////

// A reader that blocks until a channel is closed, then returns an error.
type blockingReader struct {
	c chan struct{}
}

func (br *blockingReader) Read(p []byte) (n int, err error) {
	<-br.c
	err = errors.New("blockingReader")
	return
}

func (t *rangeReaderTest) ReadAt(offset int64, size int64) (gcsx.ReaderResponse, error) {
	req := &gcsx.GCSReaderRequest{
		Offset:    offset,
		EndOffset: offset + size,
		Buffer:    make([]byte, size),
	}
	t.rangeReader.CheckInvariants()
	defer t.rangeReader.CheckInvariants()
	return t.rangeReader.ReadAt(t.ctx, req)
}

func (t *rangeReaderTest) mockNewReaderWithHandleCallForTestBucket(start uint64, limit uint64, rd gcs.StorageReader) {
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(rg *gcs.ReadObjectRequest) bool {
		return rg != nil && (*rg.Range).Start == start && (*rg.Range).Limit == limit
	})).Return(rd, nil).Once()
}

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
				t.rangeReader.reader = getReader()
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
				t.rangeReader.reader = getReader()
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
				assert.Panics(t.T(), func() { rr.CheckInvariants() }, "Expected panic")
			} else {
				assert.NotPanics(t.T(), func() { rr.CheckInvariants() }, "Expected no panic")
			}
		})
	}
}

func (t *rangeReaderTest) Test_Destroy_NonNilReader() {
	t.rangeReader.reader = getReader()

	t.rangeReader.Destroy()

	assert.Nil(t.T(), t.rangeReader.Reader)
	assert.Nil(t.T(), t.rangeReader.cancel)
	assert.Equal(t.T(), []byte(fakeHandleData), t.rangeReader.readHandle)
}

func (t *rangeReaderTest) Test_ReadAt_NewReaderReturnsError() {
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, errors.New("taco")).Once()

	_, err := t.ReadAt(0, int64(1))

	assert.Error(t.T(), err)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *rangeReaderTest) Test_ReadAt_ReadFailsWithTimeoutError() {
	r := iotest.OneByteReader(iotest.TimeoutReader(strings.NewReader("xxx")))
	rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(rc, nil).Once()

	_, err := t.ReadAt(0, int64(3))

	assert.Error(t.T(), err)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *rangeReaderTest) TestReadAt_SuccessfulRead() {
	offset := int64(0)
	size := int64(5)
	content := []byte("hello world")
	r := &fake.FakeReader{ReadCloser: getReadCloser(content)}
	t.mockNewReaderWithHandleCallForTestBucket(uint64(offset), uint64(offset+size), r)

	resp, err := t.ReadAt(offset, size)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), int(size), resp.Size)
	assert.Equal(t.T(), content[:size], resp.DataBuf)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *rangeReaderTest) TestReadAt_PartialReadWithEOF() {
	offset := int64(0)
	size := int64(10)                                       // shorter than requested
	partialReader := io.NopCloser(iotest.ErrReader(io.EOF)) // Simulates early EOF
	r := &fake.FakeReader{ReadCloser: partialReader}
	t.mockNewReaderWithHandleCallForTestBucket(uint64(offset), uint64(offset+size), r)

	resp, err := t.ReadAt(offset, size)

	assert.Error(t.T(), err)
	assert.Zero(t.T(), resp.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *rangeReaderTest) TestReadAt_StartReadNotFound() {
	offset := int64(0)
	size := int64(5)
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, &gcs.NotFoundError{}).Once()

	resp, err := t.ReadAt(offset, size)

	var fcErr *gcsfuse_errors.FileClobberedError
	assert.ErrorAs(t.T(), err, &fcErr)
	assert.Zero(t.T(), resp.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *rangeReaderTest) TestReadAt_StartReadUnexpectedError() {
	offset := int64(0)
	size := int64(5)
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, errors.New("network error")).Once()

	resp, err := t.ReadAt(offset, size)

	assert.Error(t.T(), err)
	assert.Zero(t.T(), resp.Size)
	t.mockBucket.AssertExpectations(t.T())
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
			testContent := testutil.GenerateRandomBytes(8)
			rc := &fake.FakeReader{
				ReadCloser: getReadCloser(testContent),
				Handle:     tc.readHandle,
			}
			t.rangeReader.reader = rc
			t.rangeReader.cancel = func() {}

			_, err := t.rangeReader.readFromRangeReader(t.ctx, make([]byte, 10), 0, 10, "unhandled")

			assert.Error(t.T(), err)
			assert.True(t.T(), strings.Contains(err.Error(), "extra bytes: 2"))
			assert.Nil(t.T(), t.rangeReader.reader)
			assert.Nil(t.T(), t.rangeReader.reader)
			assert.Equal(t.T(), int64(-1), t.rangeReader.start)
			assert.Equal(t.T(), int64(-1), t.rangeReader.limit)
			expectedReadHandle := tc.readHandle
			assert.Equal(t.T(), expectedReadHandle, t.rangeReader.readHandle)
		})
	}
}

func (t *rangeReaderTest) TestPropagatesCancellation() {
	// Arrange
	// Setup a blocking reader
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

	// Act
	go func() {
		buf := make([]byte, 2)
		_, _ = t.rangeReader.ReadAt(ctx, &gcsx.GCSReaderRequest{
			Buffer:    buf,
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

	// Assert
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
