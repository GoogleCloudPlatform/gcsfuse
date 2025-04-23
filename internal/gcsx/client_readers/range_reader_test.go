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
}

func (t *rangeReaderTest) Test_ReadAt_ReadFailsWithTimeoutError() {
	r := iotest.OneByteReader(iotest.TimeoutReader(strings.NewReader("xxx")))
	rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(rc, nil).Once()
	t.mockBucket.On("BucketType").Return(gcs.BucketType{}).Once()

	_, err := t.ReadAt(0, int64(3))

	assert.Error(t.T(), err)
}

func (t *rangeReaderTest) Test_ReadAt_TestSuccessfulSingleRead() {
	content := []byte("hello world")
	t.object.Size = uint64(len(content))
	r := &fake.FakeReader{ReadCloser: getReadCloser(content)}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(r, nil).Once()

	resp, err := t.ReadAt(0, int64(t.object.Size))

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), content, resp.DataBuf)
	assert.Equal(t.T(), len(content), resp.Size)
}

func (t *rangeReaderTest) Test_ReadAt_TestSuccessfulMultipleReads() {
	content := []byte("abcdefghijklmnopq")
	t.object.Size = uint64(len(content))
	chunkSize := 5
	r := &fake.FakeReader{ReadCloser: getReadCloser(content)}
	// First read
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(r, nil).Once()
	resp1, err := t.ReadAt(0, int64(chunkSize))
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), content[:chunkSize], resp1.DataBuf)
	assert.Equal(t.T(), chunkSize, resp1.Size)

	// Second read, should reuse the existing reader
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(r, nil).Once()
	resp2, err := t.ReadAt(int64(chunkSize), int64(chunkSize))
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), content[chunkSize:2*chunkSize], resp2.DataBuf)
	assert.Equal(t.T(), chunkSize, resp2.Size)

	// Third read, should continue with the same reader
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(r, nil).Once()
	resp3, err := t.ReadAt(int64(2*chunkSize), int64(len(content)-2*chunkSize))
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), content[2*chunkSize:], resp3.DataBuf)
	assert.Equal(t.T(), len(content)-2*chunkSize, resp3.Size)
}
