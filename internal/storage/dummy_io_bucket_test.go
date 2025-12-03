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

package storage

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDummyIOBucket(t *testing.T) {
	testCases := []struct {
		name     string
		wrapped  gcs.Bucket
		expected gcs.Bucket
	}{
		{
			name:     "nil_wrapped",
			wrapped:  nil,
			expected: nil,
		},
		{
			name:     "non_nil_wrapped",
			wrapped:  &TestifyMockBucket{},
			expected: &dummyIOBucket{wrapped: &TestifyMockBucket{}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NewDummyIOBucket(tc.wrapped, DummyIOBucketParams{})
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDummyIOBucket_Name(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	mockBucket.On("Name").Return("test-bucket")
	dummyBucket := NewDummyIOBucket(mockBucket, DummyIOBucketParams{})

	result := dummyBucket.Name()

	assert.Equal(t, "test-bucket", result)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_BucketType(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	expectedType := gcs.BucketType{Hierarchical: false, Zonal: false}
	mockBucket.On("BucketType").Return(expectedType)
	dummyBucket := NewDummyIOBucket(mockBucket, DummyIOBucketParams{})
	require.NotNil(t, dummyBucket)

	result := dummyBucket.BucketType()

	assert.Equal(t, expectedType, result)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_DeleteObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.DeleteObjectRequest{Name: "test-object"}
	mockBucket.On("DeleteObject", ctx, req).Return(nil)
	dummyBucket := NewDummyIOBucket(mockBucket, DummyIOBucketParams{})
	require.NotNil(t, dummyBucket)

	err := dummyBucket.DeleteObject(ctx, req)

	assert.NoError(t, err)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_DeleteObject_Error(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.DeleteObjectRequest{Name: "test-object"}
	expectedErr := errors.New("delete failed")
	mockBucket.On("DeleteObject", ctx, req).Return(expectedErr)
	dummyBucket := NewDummyIOBucket(mockBucket, DummyIOBucketParams{})
	require.NotNil(t, dummyBucket)

	err := dummyBucket.DeleteObject(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_StatObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.StatObjectRequest{Name: "test-object"}
	expectedMinObj := &gcs.MinObject{Name: "test-object"}
	expectedExtAttrs := &gcs.ExtendedObjectAttributes{}
	mockBucket.On("StatObject", ctx, req).Return(expectedMinObj, expectedExtAttrs, nil)
	dummyBucket := NewDummyIOBucket(mockBucket, DummyIOBucketParams{})
	require.NotNil(t, dummyBucket)

	minObj, extAttrs, err := dummyBucket.StatObject(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, expectedMinObj, minObj)
	assert.Equal(t, expectedExtAttrs, extAttrs)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_ListObjects(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.ListObjectsRequest{Prefix: "test-"}
	expectedListing := &gcs.Listing{}
	mockBucket.On("ListObjects", ctx, req).Return(expectedListing, nil)
	dummyBucket := NewDummyIOBucket(mockBucket, DummyIOBucketParams{})
	require.NotNil(t, dummyBucket)

	listing, err := dummyBucket.ListObjects(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, expectedListing, listing)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_CopyObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.CopyObjectRequest{
		SrcName: "source-object",
		DstName: "dest-object",
	}
	expectedObj := &gcs.Object{Name: "dest-object"}
	mockBucket.On("CopyObject", ctx, req).Return(expectedObj, nil)
	dummyBucket := NewDummyIOBucket(mockBucket, DummyIOBucketParams{})
	require.NotNil(t, dummyBucket)

	obj, err := dummyBucket.CopyObject(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, expectedObj, obj)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_DeleteFolder(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	folderName := "test-folder"
	mockBucket.On("DeleteFolder", ctx, folderName).Return(nil)
	dummyBucket := NewDummyIOBucket(mockBucket, DummyIOBucketParams{})
	require.NotNil(t, dummyBucket)

	err := dummyBucket.DeleteFolder(ctx, folderName)

	assert.NoError(t, err)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_GetFolder(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	folderName := "test-folder"
	expectedFolder := &gcs.Folder{Name: folderName}
	mockBucket.On("GetFolder", ctx, folderName).Return(expectedFolder, nil)
	dummyBucket := NewDummyIOBucket(mockBucket, DummyIOBucketParams{})
	require.NotNil(t, dummyBucket)

	folder, err := dummyBucket.GetFolder(ctx, folderName)

	assert.NoError(t, err)
	assert.Equal(t, expectedFolder, folder)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_CreateFolder(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	folderName := "new-folder"
	expectedFolder := &gcs.Folder{Name: folderName}
	mockBucket.On("CreateFolder", ctx, folderName).Return(expectedFolder, nil)
	dummyBucket := NewDummyIOBucket(mockBucket, DummyIOBucketParams{})
	require.NotNil(t, dummyBucket)

	folder, err := dummyBucket.CreateFolder(ctx, folderName)

	assert.NoError(t, err)
	assert.Equal(t, expectedFolder, folder)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_GCSName(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	obj := &gcs.MinObject{Name: "test-object"}
	expectedName := "gcs-name"
	mockBucket.On("GCSName", obj).Return(expectedName)
	dummyBucket := NewDummyIOBucket(mockBucket, DummyIOBucketParams{})
	require.NotNil(t, dummyBucket)

	name := dummyBucket.GCSName(obj)

	assert.Equal(t, expectedName, name)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_MoveObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.MoveObjectRequest{
		SrcName: "source-object",
		DstName: "dest-object",
	}
	expectedObj := &gcs.Object{Name: "dest-object"}
	mockBucket.On("MoveObject", ctx, req).Return(expectedObj, nil)
	dummyBucket := NewDummyIOBucket(mockBucket, DummyIOBucketParams{})
	require.NotNil(t, dummyBucket)

	obj, err := dummyBucket.MoveObject(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, expectedObj, obj)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_UpdateObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.UpdateObjectRequest{
		Name: "test-object",
	}
	expectedObj := &gcs.Object{Name: "test-object"}
	mockBucket.On("UpdateObject", ctx, req).Return(expectedObj, nil)
	dummyBucket := NewDummyIOBucket(mockBucket, DummyIOBucketParams{})
	require.NotNil(t, dummyBucket)

	obj, err := dummyBucket.UpdateObject(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, expectedObj, obj)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_NewReaderWithReadHandle(t *testing.T) {
	req := &gcs.ReadObjectRequest{
		Name: "test-object",
		Range: &gcs.ByteRange{
			Start: 0,
			Limit: 100,
		},
	}
	dummyBucket := NewDummyIOBucket(&TestifyMockBucket{}, DummyIOBucketParams{ReaderLatency: 0})

	reader, err := dummyBucket.NewReaderWithReadHandle(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, reader)
	assert.IsType(t, &dummyReader{}, reader)
	assert.Equal(t, uint64(100), reader.(*dummyReader).totalLen)
	assert.Equal(t, uint64(0), reader.(*dummyReader).bytesRead)
}

func TestDummyIOBucket_NewReaderWithReadHandle_NoRange(t *testing.T) {
	req := &gcs.ReadObjectRequest{
		Name: "test-object",
		// No Range specified
	}
	dummyBucket := NewDummyIOBucket(&TestifyMockBucket{}, DummyIOBucketParams{ReaderLatency: 0})

	reader, err := dummyBucket.NewReaderWithReadHandle(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, reader)
}

func TestDummyIOBucket_NewReaderWithReadHandle_InvalidRange(t *testing.T) {
	req := &gcs.ReadObjectRequest{
		Name: "test-object",
		Range: &gcs.ByteRange{
			Start: 100,
			Limit: 50, // Invalid range: Limit < Start
		},
	}
	dummyBucket := NewDummyIOBucket(&TestifyMockBucket{}, DummyIOBucketParams{ReaderLatency: 0})

	reader, err := dummyBucket.NewReaderWithReadHandle(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, reader)
}

func TestDummyIOBucket_NewReaderWithReadHandle_WithLatency(t *testing.T) {
	req := &gcs.ReadObjectRequest{
		Name: "test-object",
		Range: &gcs.ByteRange{
			Start: 0,
			Limit: 100,
		},
	}
	dummyBucket := NewDummyIOBucket(&TestifyMockBucket{}, DummyIOBucketParams{ReaderLatency: 5 * time.Millisecond})

	start := time.Now()
	reader, err := dummyBucket.NewReaderWithReadHandle(context.Background(), req)
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(5))
	assert.NoError(t, err)
	assert.NotNil(t, reader)
	assert.IsType(t, &dummyReader{}, reader)
	assert.Equal(t, uint64(100), reader.(*dummyReader).totalLen)
	assert.Equal(t, uint64(0), reader.(*dummyReader).bytesRead)
}

// //////////////////////////////////////////////////////////////////////
// Test for dummyReader
// //////////////////////////////////////////////////////////////////////
func TestDummyReader_NewDummyReader(t *testing.T) {
	dummyReader := newDummyReader(10, 0)

	assert.Equal(t, uint64(10), dummyReader.totalLen)
	assert.Equal(t, uint64(0), dummyReader.bytesRead)
	assert.NotNil(t, dummyReader.readHandle)
}

func TestDummyReader_ReadFull(t *testing.T) {
	dummyReader := newDummyReader(10, 0)

	buffer := make([]byte, 10)
	n, err := dummyReader.Read(buffer)

	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, 10, n)
	assert.Equal(t, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, buffer)
}

func TestDummyReader_ReadPartial(t *testing.T) {
	dummyReader := newDummyReader(10, 0)

	buffer := make([]byte, 5)
	n, err := dummyReader.Read(buffer)

	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte{0, 0, 0, 0, 0}, buffer)
}

func TestDummyReader_ReadBeyondEOF(t *testing.T) {
	dummyReader := newDummyReader(10, 0)
	// First read 8 bytes
	buffer1 := make([]byte, 8)
	n1, err1 := dummyReader.Read(buffer1)
	require.NoError(t, err1)
	require.Equal(t, 8, n1)
	require.Equal(t, []byte{0, 0, 0, 0, 0, 0, 0, 0}, buffer1)

	// Then read 5 bytes, which goes beyond EOF
	buffer2 := make([]byte, 5)
	n2, err2 := dummyReader.Read(buffer2)

	assert.Error(t, err2)
	assert.Equal(t, io.EOF, err2)
	assert.Equal(t, 2, n2)
	assert.Equal(t, []byte{0, 0}, buffer2[:n2])
}

func TestDummyReader_Close(t *testing.T) {
	dummyReader := newDummyReader(10, 0)

	err := dummyReader.Close()

	assert.NoError(t, err)
}

func TestDummyReader_ReadHandle(t *testing.T) {
	dummyReader := newDummyReader(10, 0)

	handle := dummyReader.ReadHandle()

	assert.NotNil(t, handle)
}

func TestDummyReader_ReadWithLatency(t *testing.T) {
	perMBLatency := 10 * time.Millisecond
	dummyReader := newDummyReader(1024*1024, perMBLatency) // 1 MB total length

	buffer := make([]byte, 512*1024) // Read 512 KB
	start := time.Now()
	n, err := dummyReader.Read(buffer)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 512*1024, n)
	assert.GreaterOrEqual(t, elapsed, 5*time.Millisecond)
}
