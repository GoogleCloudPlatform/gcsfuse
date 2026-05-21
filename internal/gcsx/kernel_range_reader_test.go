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

package gcsx

import (
	"bytes"
	"context"
	"io"
	"testing"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
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
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 100, // Equal to object size
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.ErrorIs(t.T(), err, io.EOF)
	assert.Equal(t.T(), 0, resp.Size)
}

func (t *KernelRangeReaderTest) TestReadAt_PartialRead() {
	data := []byte("hello world") // length 11
	t.object.Size = 10            // Limit read to 10
	req := &ReadRequest{
		Buffer: make([]byte, 10),
		Offset: 5,
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
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}
	expectedErr := io.ErrUnexpectedEOF
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, expectedErr).Once()

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.ErrorContains(t.T(), err, "failed to create range reader")
	assert.Equal(t.T(), 0, resp.Size)
	t.bucket.AssertExpectations(t.T())
}

func (t *KernelRangeReaderTest) TestReadAt_NilObject() {
	t.reader.instance.SetMinObject(nil)
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.ErrorContains(t.T(), err, "KernelRangeReader::ReadAt: Nil MinObject")
	assert.Equal(t.T(), 0, resp.Size)
}

