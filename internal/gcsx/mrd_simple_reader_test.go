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
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MrdSimpleReaderTest struct {
	suite.Suite
	object      *gcs.MinObject
	bucket      *storage.TestifyMockBucket
	cache       *lru.Cache
	inodeID     fuseops.InodeID
	config      *cfg.Config
	mrdInstance *MrdInstance
	reader      *MrdSimpleReader
}

func TestMrdSimpleReaderTestSuite(t *testing.T) {
	suite.Run(t, new(MrdSimpleReaderTest))
}

func (t *MrdSimpleReaderTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       "foo",
		Size:       1024 * MiB,
		Generation: 1234,
	}
	t.bucket = new(storage.TestifyMockBucket)
	t.cache = lru.NewCache(2)
	t.inodeID = 100
	t.config = &cfg.Config{Mrd: cfg.MrdConfig{PoolSize: 1}}

	t.mrdInstance = NewMrdInstance(t.object, t.bucket, t.cache, t.inodeID, t.config)
	t.reader = NewMrdSimpleReader(t.mrdInstance, metrics.NewNoopMetrics())
}

func (t *MrdSimpleReaderTest) TestNewMrdSimpleReader() {
	assert.NotNil(t.T(), t.reader)
	assert.Equal(t.T(), t.mrdInstance, t.reader.mrdInstance)
}

func (t *MrdSimpleReaderTest) TestReadAt_EmptyBuffer() {
	req := &ReadRequest{
		Buffer: []byte{},
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 0, resp.Size)
}

func (t *MrdSimpleReaderTest) TestReadAt_Success() {
	data := []byte("hello world")
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, data)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}
	// Verify initial refCount
	t.mrdInstance.refCountMu.Lock()
	assert.Equal(t.T(), int64(0), t.mrdInstance.refCount)
	t.mrdInstance.refCountMu.Unlock()

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, resp.Size)
	assert.Equal(t.T(), "hello", string(req.Buffer))
	// Verify refCount incremented
	t.mrdInstance.refCountMu.Lock()
	assert.Equal(t.T(), int64(1), t.mrdInstance.refCount)
	t.mrdInstance.refCountMu.Unlock()
}

func (t *MrdSimpleReaderTest) TestReadAt_MultipleCalls() {
	data := []byte("hello world")
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, data)
	// Expect NewMultiRangeDownloader only once because subsequent reads reuse the instance/pool
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}

	// First call
	_, err := t.reader.ReadAt(context.Background(), req)
	assert.NoError(t.T(), err)

	// Second call
	_, err = t.reader.ReadAt(context.Background(), req)
	assert.NoError(t.T(), err)

	// Verify refCount is still 1
	t.mrdInstance.refCountMu.Lock()
	assert.Equal(t.T(), int64(1), t.mrdInstance.refCount)
	t.mrdInstance.refCountMu.Unlock()
}

func (t *MrdSimpleReaderTest) TestReadAt_NilMrdInstance() {
	t.reader.mrdInstance = nil
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "mrdInstance is nil")
	assert.Equal(t.T(), 0, resp.Size)
}

func (t *MrdSimpleReaderTest) TestReadAt_ShortRead_RetrySuccess() {
	data := []byte("hello world")
	// First MRD returns short read.
	fakeMRD1 := fake.NewFakeMultiRangeDownloaderWithShortRead(t.object, data)
	// Second MRD returns full read.
	fakeMRD2 := fake.NewFakeMultiRangeDownloader(t.object, data)
	// Expectation:
	// 1. Initial Read calls ensureMRDPool -> NewMRDPool -> NewMultiRangeDownloader. Returns fakeMRD1.
	// 2. Read returns short read.
	// 3. ReadAt calls RecreateMRD -> NewMRDPool -> NewMultiRangeDownloader. Returns fakeMRD2.
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()
	buf := make([]byte, len(data))
	req := &ReadRequest{
		Buffer: buf,
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), len(data), resp.Size)
	assert.Equal(t.T(), string(data), string(buf))
	// Verify refCount incremented
	t.mrdInstance.refCountMu.Lock()
	assert.Equal(t.T(), int64(1), t.mrdInstance.refCount)
	t.mrdInstance.refCountMu.Unlock()
	t.bucket.AssertExpectations(t.T())
}

func (t *MrdSimpleReaderTest) TestReadAt_ShortRead_RetryFails() {
	data := []byte("hello world")
	// First MRD returns short read with io.EOF.
	fakeMRD1 := fake.NewFakeMultiRangeDownloaderWithShortRead(t.object, data)
	// Second MRD returns 0 bytes and an error.
	retryErr := status.Error(codes.Internal, "Internal error")
	fakeMRD2 := fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, []byte{}, 0, retryErr)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()
	buf := make([]byte, len(data))
	req := &ReadRequest{
		Buffer: buf,
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	// ReadAt returns the error from the last attempt (the retry).
	assert.ErrorIs(t.T(), err, retryErr)
	assert.Equal(t.T(), 5, resp.Size)
	assert.Equal(t.T(), "hello", string(buf[:5]))
	t.bucket.AssertExpectations(t.T())
}

func (t *MrdSimpleReaderTest) TestReadAt_OutOfRange_TriggersRetry() {
	data := []byte("hello world")
	// First MRD returns OutOfRange error.
	outOfRangeErr := status.Error(codes.OutOfRange, "Out of range")
	fakeMRD1 := fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, []byte{}, 0, outOfRangeErr)
	// Second MRD returns full read.
	fakeMRD2 := fake.NewFakeMultiRangeDownloader(t.object, data)
	// Expectation:
	// 1. Initial Read calls ensureMRDPool -> NewMRDPool -> NewMultiRangeDownloader. Returns fakeMRD1.
	// 2. Read returns OutOfRange. isShortRead detects this as recoverable.
	// 3. ReadAt calls RecreateMRD -> NewMRDPool -> NewMultiRangeDownloader. Returns fakeMRD2.
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()
	// Create the ReadRequest.
	buf := make([]byte, len(data))
	req := &ReadRequest{
		Buffer: buf,
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), len(data), resp.Size)
	assert.Equal(t.T(), string(data), string(buf))
	// Verify refCount incremented
	t.mrdInstance.refCountMu.Lock()
	assert.Equal(t.T(), int64(1), t.mrdInstance.refCount)
	t.mrdInstance.refCountMu.Unlock()
	t.bucket.AssertExpectations(t.T())
}

func (t *MrdSimpleReaderTest) TestReadAt_OutOfRange_RetryFails() {
	// First MRD returns OutOfRange error.
	outOfRangeErr := status.Error(codes.OutOfRange, "Out of range")
	fakeMRD1 := fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, []byte{}, 0, outOfRangeErr)
	// Second MRD returns Internal error.
	retryErr := status.Error(codes.Internal, "Internal error")
	fakeMRD2 := fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, []byte{}, 0, retryErr)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()
	// Create the ReadRequest.
	buf := make([]byte, 10)
	req := &ReadRequest{
		Buffer: buf,
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	// Should return the error from the retry.
	assert.ErrorIs(t.T(), err, retryErr)
	assert.Equal(t.T(), 0, resp.Size)
	t.bucket.AssertExpectations(t.T())
}

func TestIsShortRead(t *testing.T) {
	testCases := []struct {
		name       string
		bytesRead  int
		bufferSize int
		err        error
		expected   bool
	}{
		{
			name:       "Full read, no error",
			bytesRead:  10,
			bufferSize: 10,
			err:        nil,
			expected:   false,
		},
		{
			name:       "Full read, EOF",
			bytesRead:  10,
			bufferSize: 10,
			err:        io.EOF,
			expected:   false,
		},
		{
			name:       "Short read, no error",
			bytesRead:  5,
			bufferSize: 10,
			err:        nil,
			expected:   true,
		},
		{
			name:       "Short read, EOF",
			bytesRead:  5,
			bufferSize: 10,
			err:        io.EOF,
			expected:   true,
		},
		{
			name:       "Short read, UnexpectedEOF",
			bytesRead:  5,
			bufferSize: 10,
			err:        io.ErrUnexpectedEOF,
			expected:   true,
		},
		{
			name:       "Short read, OutOfRange",
			bytesRead:  0,
			bufferSize: 10,
			err:        status.Error(codes.OutOfRange, "out of range"),
			expected:   true,
		},
		{
			name:       "Short read, Wrapped OutOfRange",
			bytesRead:  0,
			bufferSize: 10,
			err:        fmt.Errorf("wrapped: %w", status.Error(codes.OutOfRange, "out of range")),
			expected:   true,
		},
		{
			name:       "Short read, Internal error",
			bytesRead:  5,
			bufferSize: 10,
			err:        status.Error(codes.Internal, "internal error"),
			expected:   false,
		},
		{
			name:       "Short read, Generic error",
			bytesRead:  5,
			bufferSize: 10,
			err:        errors.New("generic error"),
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isShortRead(tc.bytesRead, tc.bufferSize, tc.err))
		})
	}
}

func (t *MrdSimpleReaderTest) TestDestroy() {
	// Setup state where refCount is incremented
	t.reader.mrdInstanceInUse.Store(true)
	t.mrdInstance.refCount = 1

	t.reader.Destroy()

	assert.Nil(t.T(), t.reader.mrdInstance)
	assert.False(t.T(), t.reader.mrdInstanceInUse.Load())
	// Verify refCount decremented
	t.mrdInstance.refCountMu.Lock()
	assert.Equal(t.T(), int64(0), t.mrdInstance.refCount)
	t.mrdInstance.refCountMu.Unlock()
	// Verify that calling Destroy again doesn't panic
	t.reader.Destroy()
}

func (t *MrdSimpleReaderTest) TestReadAt_RecreateMRDFails_RetriesWithOldMRD() {
	data := []byte("hello world")
	// First MRD returns short read.
	fakeMRD1 := fake.NewFakeMultiRangeDownloaderWithShortRead(t.object, data)
	// Expectation:
	// 1. Initial Read calls ensureMRDPool -> NewMRDPool -> NewMultiRangeDownloader. Returns fakeMRD1.
	// 2. Read returns short read.
	// 3. ReadAt calls RecreateMRD -> NewMRDPool -> NewMultiRangeDownloader. Returns ERROR.
	// 4. ReadAt logs warning and retries with existing pool (fakeMRD1).
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, errors.New("recreate failed")).Once()
	buf := make([]byte, len(data))
	req := &ReadRequest{
		Buffer: buf,
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	// Assert
	// We verify that we didn't get the "recreate failed" error.
	if err != nil {
		assert.NotEqual(t.T(), "recreate failed", err.Error())
	}
	// We expect some data to be read (from first attempt + potentially second attempt).
	assert.Greater(t.T(), resp.Size, 0)
	t.bucket.AssertExpectations(t.T())
}

func (t *MrdSimpleReaderTest) TestDestroy_NoReadAt() {
	// Check initial refCount
	t.mrdInstance.refCountMu.Lock()
	assert.Equal(t.T(), int64(0), t.mrdInstance.refCount)
	t.mrdInstance.refCountMu.Unlock()
	// Capture logs
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	defer logger.SetOutput(os.Stdout)

	// Act
	t.reader.Destroy()

	// Assert
	t.mrdInstance.refCountMu.Lock()
	assert.Equal(t.T(), int64(0), t.mrdInstance.refCount)
	t.mrdInstance.refCountMu.Unlock()
	assert.NotContains(t.T(), buf.String(), "MrdInstance::DecrementRefCount: Refcount cannot be negative")
}

func (t *MrdSimpleReaderTest) TestReadAt_ContextCanceled() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := &ReadRequest{
		Buffer: make([]byte, 10),
		Offset: 0,
	}
	// Use sleep to ensure context cancellation is detected before read completes
	fakeMRD := fake.NewFakeMultiRangeDownloaderWithSleep(t.object, []byte("data"), 100*time.Millisecond)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()

	resp, err := t.reader.ReadAt(ctx, req)

	assert.ErrorIs(t.T(), err, context.Canceled)
	assert.Equal(t.T(), 0, resp.Size)
}
