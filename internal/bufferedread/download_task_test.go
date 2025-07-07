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

package bufferedread

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	testutil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DownloadTaskTestSuite struct {
	workerpool.Task
	suite.Suite
	object     *gcs.MinObject
	mockBucket *storage.TestifyMockBucket
}

func TestDownloadTaskTestSuite(t *testing.T) {
	suite.Run(t, new(DownloadTaskTestSuite))
}

func (pts *DownloadTaskTestSuite) SetupTest() {
	pts.object = &gcs.MinObject{
		Name:       "test-object",
		Size:       1024,
		Generation: 1234567890,
	}
	pts.mockBucket = new(storage.TestifyMockBucket)
}

func getReadCloser(content []byte) io.ReadCloser {
	r := bytes.NewReader(content)
	rc := io.NopCloser(r)
	return rc
}

func (pts *DownloadTaskTestSuite) TestExecuteSuccess() {
	blockSize := 500
	downloadBlock, err := block.CreateBlock(int64(blockSize))
	require.Nil(pts.T(), err)
	err = downloadBlock.SetAbsStartOff(0)
	require.Nil(pts.T(), err)
	task := NewDownloadTask(context.Background(), pts.object, pts.mockBucket, downloadBlock, nil)
	testContent := testutil.GenerateRandomBytes(blockSize)
	rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       pts.object.Name,
		Generation: pts.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(0),
			Limit: uint64(blockSize),
		},
	}
	pts.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(rc, nil).Times(1)

	task.Execute()

	assert.Equal(pts.T(), int64(len(testContent)), downloadBlock.Size())
	assert.Equal(pts.T(), int64(blockSize), downloadBlock.Cap())
	assert.NoError(pts.T(), err)
	pts.mockBucket.AssertExpectations(pts.T())
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer cancelFunc()
	status, err := downloadBlock.AwaitReady(ctx)
	assert.Equal(pts.T(), block.BlockStatusDownloaded, status)
	assert.NoError(pts.T(), err)
}

func (pts *DownloadTaskTestSuite) TestExecuteError() {
	blockSize := 500
	downloadBlock, err := block.CreateBlock(int64(blockSize))
	require.Nil(pts.T(), err)
	err = downloadBlock.SetAbsStartOff(0)
	require.Nil(pts.T(), err)
	task := NewDownloadTask(context.Background(), pts.object, pts.mockBucket, downloadBlock, nil)
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       pts.object.Name,
		Generation: pts.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(0),
			Limit: uint64(blockSize),
		},
	}
	expectedError := errors.New("read error")
	pts.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(nil, expectedError).Times(1)

	task.Execute()

	assert.Error(pts.T(), expectedError)
	pts.mockBucket.AssertExpectations(pts.T())
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer cancelFunc()
	status, err := downloadBlock.AwaitReady(ctx)
	assert.Equal(pts.T(), block.BlockStatusDownloadFailed, status)
	assert.NoError(pts.T(), err)
}

func (pts *DownloadTaskTestSuite) TestExecuteContextCancelledWhileReaderCreation() {
	blockSize := 500
	downloadBlock, err := block.CreateBlock(int64(blockSize))
	require.Nil(pts.T(), err)
	err = downloadBlock.SetAbsStartOff(0)
	require.Nil(pts.T(), err)
	task := NewDownloadTask(context.Background(), pts.object, pts.mockBucket, downloadBlock, nil)
	rc := &fake.FakeReader{ReadCloser: getReadCloser(nil)} // No content since context is cancelled
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       pts.object.Name,
		Generation: pts.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(0),
			Limit: uint64(blockSize),
		},
	}
	pts.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(rc, context.Canceled).Times(1)

	task.Execute()

	assert.Error(pts.T(), context.Canceled)
	pts.mockBucket.AssertExpectations(pts.T())
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer cancelFunc()
	status, err := downloadBlock.AwaitReady(ctx)
	assert.Equal(pts.T(), block.BlockStatusDownloadCancelled, status)
	assert.NoError(pts.T(), err)
}

// ctxCancelledReader is a mock reader that simulates a context cancellation error while reading.
type ctxCancelledReader struct {
	io.Reader
	io.Closer
}

func (r *ctxCancelledReader) Read(p []byte) (n int, err error) {
	return 0, context.Canceled
}

func (r *ctxCancelledReader) Close() error {
	return nil
}

func (pts *DownloadTaskTestSuite) TestExecuteContextCancelledWhileReadingFromReader() {
	blockSize := 500
	downloadBlock, err := block.CreateBlock(int64(blockSize))
	require.Nil(pts.T(), err)
	err = downloadBlock.SetAbsStartOff(0)
	require.Nil(pts.T(), err)
	task := NewDownloadTask(context.Background(), pts.object, pts.mockBucket, downloadBlock, nil)
	rc := &fake.FakeReader{ReadCloser: new(ctxCancelledReader)}
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       pts.object.Name,
		Generation: pts.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(0),
			Limit: uint64(blockSize),
		},
	}
	pts.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(rc, nil).Times(1)

	task.Execute()

	assert.Error(pts.T(), context.Canceled)
	pts.mockBucket.AssertExpectations(pts.T())
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer cancelFunc()
	status, err := downloadBlock.AwaitReady(ctx)
	assert.Equal(pts.T(), block.BlockStatusDownloadCancelled, status)
	assert.NoError(pts.T(), err)
}
