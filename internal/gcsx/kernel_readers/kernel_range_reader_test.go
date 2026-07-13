// Copyright 2026 Google LLC
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

package kernel_readers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/buffer"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type mockStorageReader struct {
	io.ReadCloser
}

func (m *mockStorageReader) ReadHandle() storagev2.ReadHandle {
	return storagev2.ReadHandle{}
}

type KernelRangeReaderTest struct {
	suite.Suite
	object *gcs.MinObject
	bucket *storage.TestifyMockBucket
	reader *KernelRangeReader
}

func TestKernelRangeReaderTestSuite(t *testing.T) {
	suite.Run(t, new(KernelRangeReaderTest))
}

func (t *KernelRangeReaderTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       "foo",
		Size:       100,
		Generation: 1234,
	}
	t.bucket = new(storage.TestifyMockBucket)
	instance := NewKernelRangeReaderInstance(t.object)
	t.reader = NewKernelRangeReader(t.bucket, instance, nil)
}

func (t *KernelRangeReaderTest) TestNewKernelRangeReader() {
	assert.NotNil(t.T(), t.reader)
	assert.Equal(t.T(), t.bucket, t.reader.bucket)
	assert.Equal(t.T(), t.object, t.reader.instance.GetMinObject())
}

func (t *KernelRangeReaderTest) TestReaderName() {
	assert.Equal(t.T(), "KernelRangeReader", t.reader.ReaderName())
}

func (t *KernelRangeReaderTest) TestReadAt_Success() {
	data := []byte("hello world")
	req := &gcsx.ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
		Size:   5,
	}
	mockReader := &mockStorageReader{io.NopCloser(bytes.NewReader(data))}
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(mockReader, nil).Once()

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, resp.Size)
	assert.Equal(t.T(), "hello", string(req.Buffer))
	t.bucket.AssertExpectations(t.T())
}

func (t *KernelRangeReaderTest) TestReadAt_EOF() {
	req := &gcsx.ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 100, // Equal to object size
		Size:   5,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.ErrorIs(t.T(), err, io.EOF)
	assert.Equal(t.T(), 0, resp.Size)
}

func (t *KernelRangeReaderTest) TestReadAt_PartialRead() {
	data := []byte("hello world") // length 11
	t.object.Size = 10            // Limit read to 10
	req := &gcsx.ReadRequest{
		Buffer: make([]byte, 10),
		Offset: 5,
		Size:   10,
	}
	// Expected read range: [5, 10), size = 5
	expectedData := data[5:10]
	mockReader := &mockStorageReader{io.NopCloser(bytes.NewReader(expectedData))}
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(mockReader, nil).Once()

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, resp.Size)
	assert.Equal(t.T(), string(expectedData), string(req.Buffer[:resp.Size]))
	t.bucket.AssertExpectations(t.T())
}

func (t *KernelRangeReaderTest) TestReadAt_NewReaderError() {
	req := &gcsx.ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
		Size:   5,
	}
	expectedErr := io.ErrUnexpectedEOF
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, expectedErr).Once()

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.ErrorContains(t.T(), err, "KernelRangeReader::ReadAt Failed to create range reader")
	assert.Equal(t.T(), 0, resp.Size)
	t.bucket.AssertExpectations(t.T())
}

func (t *KernelRangeReaderTest) TestReadAt_NilObject() {
	t.reader.instance.SetMinObject(nil)
	req := &gcsx.ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
		Size:   5,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.ErrorContains(t.T(), err, "KernelRangeReader::ReadAt Nil MinObject")
	assert.Equal(t.T(), 0, resp.Size)
}

func (t *KernelRangeReaderTest) TestReadAt_ClobberedError() {
	req := &gcsx.ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
		Size:   5,
	}
	gcsErr := &gcs.NotFoundError{Err: errors.New("not found")}
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, gcsErr).Once()

	resp, err := t.reader.ReadAt(context.Background(), req)

	var clobberedErr *gcsfuse_errors.FileClobberedError
	assert.ErrorAs(t.T(), err, &clobberedErr)
	assert.Equal(t.T(), "foo", clobberedErr.ObjectName)
	assert.ErrorContains(t.T(), err, "KernelRangeReader::ReadAt Failed to create range reader")
	assert.Equal(t.T(), 0, resp.Size)
	t.bucket.AssertExpectations(t.T())
}

func (t *KernelRangeReaderTest) TestReadAt_BufferPool_Success() {
	data := []byte("abcdefghij") // length 10
	pool := &buffer.FakeBufferPool{
		Buffers: [][]byte{
			make([]byte, 3),
			make([]byte, 2),
			make([]byte, 4),
		},
	}
	req := &gcsx.ReadRequest{
		BufferPool: pool,
		Offset:     0,
		Size:       9,
	}
	mockReader := &mockStorageReader{io.NopCloser(bytes.NewReader(data))}
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(mockReader, nil).Once()

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 9, resp.Size) // Sum of buffers capacity is 3+2+4 = 9
	require.Len(t.T(), resp.Data, 3)
	assert.Equal(t.T(), "abc", string(resp.Data[0]))
	assert.Equal(t.T(), "de", string(resp.Data[1]))
	assert.Equal(t.T(), "fghi", string(resp.Data[2]))
	t.bucket.AssertExpectations(t.T())
}
