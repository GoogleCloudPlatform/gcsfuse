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
	t.rangeReader.Destroy()
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
	t.rangeReader.CheckInvariants()
	defer t.rangeReader.CheckInvariants()
	return t.rangeReader.ReadAt(t.ctx, req)
}

func (t *rangeReaderTest) mockNewReaderWithHandleCallForTestBucket(start uint64, limit uint64, rd gcs.StorageReader) {
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(rg *gcs.ReadObjectRequest) bool {
		return rg != nil && (*rg.Range).Start == start && (*rg.Range).Limit == limit
	})).Return(rd, nil).Once()
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
				assert.Panics(t.T(), func() { rr.CheckInvariants() }, "Expected panic")
			} else {
				assert.NotPanics(t.T(), func() { rr.CheckInvariants() }, "Expected no panic")
			}
		})
	}
}

func (t *rangeReaderTest) Test_Destroy_NonNilReader() {
	t.rangeReader.reader = getReader(2)

	t.rangeReader.Destroy()

	assert.Nil(t.T(), t.rangeReader.Reader)
	assert.Nil(t.T(), t.rangeReader.cancel)
	assert.Equal(t.T(), []byte(fakeHandleData), t.rangeReader.readHandle)
}

func (t *rangeReaderTest) Test_ReadAt_ReadFailsWithTimeoutError() {
	content := "xxx"
	r := iotest.OneByteReader(iotest.TimeoutReader(strings.NewReader(content)))
	rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}
	t.mockNewReaderWithHandleCallForTestBucket(0, uint64(len(content)), rc)

	_, err := t.readAt(0, int64(len(content)))

	assert.Error(t.T(), err)
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
	size := int64(10)                                       // shorter than requested
	partialReader := io.NopCloser(iotest.ErrReader(io.EOF)) // Simulates early EOF
	r := &fake.FakeReader{ReadCloser: partialReader}
	t.mockNewReaderWithHandleCallForTestBucket(uint64(offset), uint64(offset+size), r)

	resp, err := t.readAt(offset, size)

	assert.Error(t.T(), err)
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

func (t *rangeReaderTest) Test_invalidateReaderIfMisalignedOrTooSmall_InvalidateReaderDueToWrongOffset() {
	t.rangeReader.reader = &fake.FakeReader{ReadCloser: getReader(100)}
	t.rangeReader.start = 50 // misaligned
	t.rangeReader.limit = 1000

	result := t.rangeReader.invalidateReaderIfMisalignedOrTooSmall(200, make([]byte, 100))

	assert.True(t.T(), result, "Expected invalidateReaderIfMisalignedOrTooSmall to return true")
	assert.Nil(t.T(), t.rangeReader.reader, "Expected reader to be nil after invalidation")
}

func (t *rangeReaderTest) Test_invalidateReaderIfMisalignedOrTooSmall_InvalidateReaderDueToTooSmall() {
	t.rangeReader.reader = &fake.FakeReader{ReadCloser: getReader(100)}
	t.rangeReader.start = 200
	t.rangeReader.limit = 250 // too small to serve full request
	t.rangeReader.object.Size = 500

	result := t.rangeReader.invalidateReaderIfMisalignedOrTooSmall(200, make([]byte, 100))

	assert.False(t.T(), result, "Expected invalidateReaderIfMisalignedOrTooSmall to return false due to size, not misalignment")
	assert.Nil(t.T(), t.rangeReader.reader, "Expected reader to be nil after invalidation")
}

func (t *rangeReaderTest) Test_invalidateReaderIfMisalignedOrTooSmall_KeepReaderIfValid() {
	t.rangeReader.reader = &fake.FakeReader{ReadCloser: getReader(100)}
	t.rangeReader.start = 200
	t.rangeReader.limit = 400

	result := t.rangeReader.invalidateReaderIfMisalignedOrTooSmall(200, make([]byte, 100))

	assert.False(t.T(), result, "Expected reader to be retained")
	assert.NotNil(t.T(), t.rangeReader.reader, "Expected reader to be intact")
}
