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
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	testutil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

const (
	testBlockSize = 500
)

type DownloadTaskTestSuite struct {
	workerpool.Task
	suite.Suite
	object       *gcs.MinObject
	mockBucket   *storage.TestifyMockBucket
	blockPool    *block.GenBlockPool[block.PrefetchBlock]
	metricHandle metrics.MetricHandle
}

func TestDownloadTaskTestSuite(t *testing.T) {
	suite.Run(t, new(DownloadTaskTestSuite))
}

func (dts *DownloadTaskTestSuite) SetupTest() {
	dts.object = &gcs.MinObject{
		Name:       "test-object",
		Size:       1024,
		Generation: 1234567890,
	}
	dts.mockBucket = new(storage.TestifyMockBucket)
	var err error
	dts.blockPool, err = block.NewPrefetchBlockPool(testBlockSize, 10, 1, semaphore.NewWeighted(100))
	dts.metricHandle = metrics.NewNoopMetrics()
	require.NoError(dts.T(), err, "Failed to create block pool")
}

func getReadCloser(content []byte) io.ReadCloser {
	r := bytes.NewReader(content)
	rc := io.NopCloser(r)
	return rc
}

func (dts *DownloadTaskTestSuite) TestExecuteSuccess() {
	downloadBlock, err := dts.blockPool.Get()
	require.Nil(dts.T(), err)
	err = downloadBlock.SetAbsStartOff(0)
	require.Nil(dts.T(), err)
	task := NewDownloadTask(context.Background(), dts.object, dts.mockBucket, downloadBlock, nil, dts.metricHandle, nil)
	testContent := testutil.GenerateRandomBytes(testBlockSize)
	rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       dts.object.Name,
		Generation: dts.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(0),
			Limit: uint64(testBlockSize),
		},
	}
	dts.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(rc, nil).Times(1)

	task.Execute()

	assert.Equal(dts.T(), int64(len(testContent)), downloadBlock.Size())
	assert.Equal(dts.T(), int64(testBlockSize), downloadBlock.Cap())
	assert.NoError(dts.T(), err)
	dts.mockBucket.AssertExpectations(dts.T())
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer cancelFunc()
	status, err := downloadBlock.AwaitReady(ctx)
	assert.Equal(dts.T(), block.BlockStatus{State: block.BlockStateDownloaded}, status)
	assert.NoError(dts.T(), err)
}

func (dts *DownloadTaskTestSuite) TestExecuteError() {
	downloadBlock, err := dts.blockPool.Get()
	require.Nil(dts.T(), err)
	err = downloadBlock.SetAbsStartOff(0)
	require.Nil(dts.T(), err)
	task := NewDownloadTask(context.Background(), dts.object, dts.mockBucket, downloadBlock, nil, dts.metricHandle, nil)
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       dts.object.Name,
		Generation: dts.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(0),
			Limit: uint64(testBlockSize),
		},
	}
	dts.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(nil, errors.New("read error")).Times(1)

	task.Execute()

	dts.mockBucket.AssertExpectations(dts.T())
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer cancelFunc()
	status, err := downloadBlock.AwaitReady(ctx)
	assert.Equal(dts.T(), block.BlockStateDownloadFailed, status.State)
	assert.NotNil(dts.T(), status.Err)
	assert.NoError(dts.T(), err)
}

func (dts *DownloadTaskTestSuite) TestExecuteContextDeadlineExceededByServerTreatedAsFailed() {
	downloadBlock, err := dts.blockPool.Get()
	require.Nil(dts.T(), err)
	err = downloadBlock.SetAbsStartOff(0)
	require.Nil(dts.T(), err)
	taskCtx, taskCancelFunc := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer taskCancelFunc() // Ensure the context is cancelled after the test.
	task := NewDownloadTask(taskCtx, dts.object, dts.mockBucket, downloadBlock, nil, dts.metricHandle, nil)
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       dts.object.Name,
		Generation: dts.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(0),
			Limit: uint64(testBlockSize),
		},
	}
	dts.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(nil, context.DeadlineExceeded).Times(1)

	task.Execute()

	assert.Error(dts.T(), context.DeadlineExceeded)
	dts.mockBucket.AssertExpectations(dts.T())
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer cancelFunc()
	status, err := downloadBlock.AwaitReady(ctx)
	assert.NoError(dts.T(), err)
	assert.Equal(dts.T(), block.BlockStateDownloadFailed, status.State)
	assert.NotNil(dts.T(), status.Err)
}

func (dts *DownloadTaskTestSuite) TestExecuteContextCancelledWhileReaderCreation() {
	downloadBlock, err := dts.blockPool.Get()
	require.Nil(dts.T(), err)
	err = downloadBlock.SetAbsStartOff(0)
	require.Nil(dts.T(), err)
	taskCtx, taskCancelFunc := context.WithCancel(context.TODO())
	task := NewDownloadTask(taskCtx, dts.object, dts.mockBucket, downloadBlock, nil, dts.metricHandle, nil)
	rc := &fake.FakeReader{ReadCloser: getReadCloser(nil)} // No content since context is cancelled
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       dts.object.Name,
		Generation: dts.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(0),
			Limit: uint64(testBlockSize),
		},
	}
	dts.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(rc, context.Canceled).Times(1)
	taskCancelFunc() // Ensure client side cancellation.

	task.Execute()

	assert.Error(dts.T(), context.Canceled)
	dts.mockBucket.AssertExpectations(dts.T())
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer cancelFunc()
	status, err := downloadBlock.AwaitReady(ctx)
	assert.NoError(dts.T(), err)
	assert.Equal(dts.T(), block.BlockStateDownloadFailed, status.State)
	assert.ErrorIs(dts.T(), status.Err, context.Canceled)
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

func (dts *DownloadTaskTestSuite) TestExecuteContextCancelledWhileReadingFromReader() {
	downloadBlock, err := dts.blockPool.Get()
	require.Nil(dts.T(), err)
	err = downloadBlock.SetAbsStartOff(0)
	require.Nil(dts.T(), err)
	taskCtx, taskCancelFunc := context.WithCancel(context.TODO())
	task := NewDownloadTask(taskCtx, dts.object, dts.mockBucket, downloadBlock, nil, dts.metricHandle, nil)
	rc := &fake.FakeReader{ReadCloser: new(ctxCancelledReader)}
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       dts.object.Name,
		Generation: dts.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(0),
			Limit: uint64(testBlockSize),
		},
	}
	dts.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(rc, nil).Times(1)
	taskCancelFunc() // Ensure client side cancellation.

	task.Execute()

	assert.Error(dts.T(), context.Canceled)
	dts.mockBucket.AssertExpectations(dts.T())
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer cancelFunc()
	status, err := downloadBlock.AwaitReady(ctx)
	assert.NoError(dts.T(), err)
	assert.Equal(dts.T(), block.BlockStateDownloadFailed, status.State)
	assert.ErrorIs(dts.T(), status.Err, context.Canceled)
}

func (dts *DownloadTaskTestSuite) TestExecuteClobbered() {
	downloadBlock, err := dts.blockPool.Get()
	require.Nil(dts.T(), err)
	err = downloadBlock.SetAbsStartOff(0)
	require.Nil(dts.T(), err)
	task := NewDownloadTask(context.Background(), dts.object, dts.mockBucket, downloadBlock, nil, dts.metricHandle, nil)
	// Simulate NewReaderWithReadHandle returning a NotFoundError.
	notFoundErr := &gcs.NotFoundError{Err: errors.New("object not found")}
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       dts.object.Name,
		Generation: dts.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(0),
			Limit: uint64(testBlockSize),
		},
	}
	dts.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(nil, notFoundErr).Times(1)

	task.Execute()

	dts.mockBucket.AssertExpectations(dts.T())
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer cancelFunc()
	status, err := downloadBlock.AwaitReady(ctx)
	assert.NoError(dts.T(), err)
	assert.Equal(dts.T(), block.BlockStateDownloadFailed, status.State)
	var fileClobberedError *gcsfuse_errors.FileClobberedError
	assert.True(dts.T(), errors.As(status.Err, &fileClobberedError))
}
