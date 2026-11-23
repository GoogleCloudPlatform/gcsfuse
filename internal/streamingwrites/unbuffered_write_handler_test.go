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

package streamingwrites

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/gcs"
	storagemock "github.com/googlecloudplatform/gcsfuse/v4/internal/storage/mock"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v4/tracing"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errAbortFailure = errors.New("error while aborting the upload")

// handlerOp is an operation on a write handler, used to build test scenarios.
type handlerOp func(ctx context.Context, wh WriteHandler) error

func writeOp(data string, offset int64) handlerOp {
	return func(ctx context.Context, wh WriteHandler) error {
		return wh.Write(ctx, []byte(data), offset)
	}
}

func truncateOp(size int64) handlerOp {
	return func(_ context.Context, wh WriteHandler) error {
		return wh.Truncate(size)
	}
}

func flushOp() handlerOp {
	return func(ctx context.Context, wh WriteHandler) error {
		_, err := wh.Flush(ctx)
		return err
	}
}

func unlinkOp() handlerOp {
	return func(ctx context.Context, wh WriteHandler) error {
		wh.Unlink(ctx)
		return nil
	}
}

func newUnbufferedWriteHandlerForTest(bucket gcs.Bucket, obj *gcs.Object) *unbufferedWriteHandlerImpl {
	return newUnbufferedWriteHandler(&CreateWriteHandlerRequest{
		Object:                   obj,
		ObjectName:               objectName,
		Bucket:                   bucket,
		ChunkRetryDeadlineSecs:   chunkRetryDeadlineSecs,
		ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
		TraceHandle:              tracing.NewNoopTracer(),
	})
}

func newFakeBucket() gcs.Bucket {
	return fake.NewFakeBucket(timeutil.RealClock(), "FakeBucketName", gcs.BucketType{})
}

// newMockBucketWithWriter returns a mock bucket that creates writer. writerCtx
// returns the context that the writer was created with.
func newMockBucketWithWriter(writer gcs.Writer) (bucket *storagemock.TestifyMockBucket, writerCtx func() context.Context) {
	var ctx context.Context
	bucket = new(storagemock.TestifyMockBucket)
	bucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { ctx = args.Get(0).(context.Context) }).
		Return(writer, nil)
	return bucket, func() context.Context { return ctx }
}

func TestNewUnbufferedWriteHandler(t *testing.T) {
	bucket := new(storagemock.TestifyMockBucket)
	obj := &gcs.Object{Name: objectName, Generation: 7}
	traceHandle := tracing.NewNoopTracer()
	before := time.Now()

	wh := newUnbufferedWriteHandler(&CreateWriteHandlerRequest{
		Object:                   obj,
		ObjectName:               objectName,
		Bucket:                   bucket,
		ChunkRetryDeadlineSecs:   chunkRetryDeadlineSecs,
		ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
		TraceHandle:              traceHandle,
	})

	assert.Nil(t, wh.writer)
	assert.Same(t, bucket, wh.bucket)
	assert.Equal(t, objectName, wh.objectName)
	assert.Same(t, obj, wh.obj)
	assert.Equal(t, chunkRetryDeadlineSecs, wh.chunkRetryDeadlineSecs)
	assert.Equal(t, chunkTransferTimeoutSecs, wh.chunkTransferTimeoutSecs)
	assert.Equal(t, traceHandle, wh.traceHandle)
	assert.Equal(t, int64(0), wh.totalSize)
	assert.Equal(t, int64(-1), wh.truncatedSize)
	assert.False(t, wh.mtime.Before(before))
	assert.False(t, wh.mtime.After(time.Now()))
}

func TestUnbufferedWriteHandler_WriteFileInfo(t *testing.T) {
	mtime := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		name              string
		ops               []handlerOp
		expectedTotalSize int64
	}{
		{
			name:              "NoWrites",
			expectedTotalSize: 0,
		},
		{
			name:              "SequentialWrites",
			ops:               []handlerOp{writeOp("foo", 0), writeOp("bar", 3)},
			expectedTotalSize: 6,
		},
		{
			name:              "PendingForwardTruncate",
			ops:               []handlerOp{writeOp("foo", 0), truncateOp(10)},
			expectedTotalSize: 10,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			wh := newUnbufferedWriteHandlerForTest(newFakeBucket(), nil)
			for _, op := range tc.ops {
				require.NoError(t, op(ctx, wh))
			}
			wh.SetMtime(mtime)

			info := wh.WriteFileInfo()

			assert.Equal(t, tc.expectedTotalSize, info.TotalSize)
			assert.Equal(t, mtime, info.Mtime)
		})
	}
}

func TestUnbufferedWriteHandler_FlushFinalizesWrittenContent(t *testing.T) {
	testCases := []struct {
		name            string
		ops             []handlerOp
		expectedContent string
	}{
		{
			name:            "EmptyFile",
			expectedContent: "",
		},
		{
			name:            "SequentialWrites",
			ops:             []handlerOp{writeOp("foo", 0), writeOp("bar", 3)},
			expectedContent: "foobar",
		},
		{
			name:            "EmptyWrite",
			ops:             []handlerOp{writeOp("foo", 0), writeOp("", 3)},
			expectedContent: "foo",
		},
		{
			name:            "TruncateBeyondSizePadsWithZeros",
			ops:             []handlerOp{writeOp("foo", 0), truncateOp(6)},
			expectedContent: "foo\x00\x00\x00",
		},
		{
			name:            "TruncatePadsMoreThanOneChunk",
			ops:             []handlerOp{writeOp("foo", 0), truncateOp(3 + 2*zeroPaddingChunkSize + 5)},
			expectedContent: "foo" + strings.Repeat("\x00", 2*zeroPaddingChunkSize+5),
		},
		{
			name:            "WriteAtTruncatedSizePadsTheGap",
			ops:             []handlerOp{writeOp("foo", 0), truncateOp(5), writeOp("bar", 5)},
			expectedContent: "foo\x00\x00bar",
		},
		{
			name:            "WriteAtCurrentSizeWithPendingTruncate",
			ops:             []handlerOp{writeOp("foo", 0), truncateOp(8), writeOp("bar", 3)},
			expectedContent: "foobar\x00\x00",
		},
		{
			name:            "WriteAtCurrentSizePastPendingTruncate",
			ops:             []handlerOp{writeOp("foo", 0), truncateOp(4), writeOp("barbaz", 3)},
			expectedContent: "foobarbaz",
		},
		{
			name:            "TruncateToCurrentSizeCancelsPendingTruncate",
			ops:             []handlerOp{writeOp("foo", 0), truncateOp(10), truncateOp(3)},
			expectedContent: "foo",
		},
		{
			name:            "TruncateWithoutWritesPadsWithZeros",
			ops:             []handlerOp{truncateOp(3)},
			expectedContent: "\x00\x00\x00",
		},
		{
			name:            "LoweringPendingTruncatePadsToNewSize",
			ops:             []handlerOp{writeOp("foo", 0), truncateOp(10), truncateOp(5)},
			expectedContent: "foo\x00\x00",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			bucket := newFakeBucket()
			wh := newUnbufferedWriteHandlerForTest(bucket, nil)
			for _, op := range tc.ops {
				require.NoError(t, op(ctx, wh))
			}

			obj, err := wh.Flush(ctx)

			require.NoError(t, err)
			require.NotNil(t, obj)
			assert.Equal(t, uint64(len(tc.expectedContent)), obj.Size)
			content, err := storageutil.ReadObject(ctx, bucket, objectName)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedContent, string(content))
		})
	}
}

func TestUnbufferedWriteHandler_OutOfOrderWrite(t *testing.T) {
	testCases := []struct {
		name   string
		ops    []handlerOp
		offset int64
	}{
		{
			name:   "BeforeCurrentSize",
			ops:    []handlerOp{writeOp("foobar", 0)},
			offset: 3,
		},
		{
			name:   "PastCurrentSize",
			ops:    []handlerOp{writeOp("foo", 0)},
			offset: 5,
		},
		{
			name:   "BetweenCurrentAndTruncatedSize",
			ops:    []handlerOp{writeOp("foo", 0), truncateOp(10)},
			offset: 5,
		},
		{
			name:   "AtTruncatedSizeAfterWritingPastIt",
			ops:    []handlerOp{writeOp("foo", 0), truncateOp(5), writeOp("abcdef", 3)},
			offset: 5,
		},
		{
			name:   "PastTruncatedSize",
			ops:    []handlerOp{writeOp("foo", 0), truncateOp(5)},
			offset: 7,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			wh := newUnbufferedWriteHandlerForTest(newFakeBucket(), nil)
			for _, op := range tc.ops {
				require.NoError(t, op(ctx, wh))
			}
			sizeBefore := wh.WriteFileInfo().TotalSize

			err := wh.Write(ctx, []byte("x"), tc.offset)

			assert.ErrorIs(t, err, ErrOutOfOrderWrite)
			assert.Equal(t, sizeBefore, wh.WriteFileInfo().TotalSize)
		})
	}
}

func TestUnbufferedWriteHandler_Truncate(t *testing.T) {
	// Each test case writes initialData first, so the current size is 3.
	const initialData = "foo"
	testCases := []struct {
		name                  string
		size                  int64
		expectedErr           error
		expectedSize          int64
		expectedTruncatedSize int64
	}{
		{
			name:                  "BelowCurrentSize",
			size:                  2,
			expectedErr:           ErrOutOfOrderWrite,
			expectedSize:          3,
			expectedTruncatedSize: -1,
		},
		{
			name:                  "ToCurrentSize",
			size:                  3,
			expectedSize:          3,
			expectedTruncatedSize: -1,
		},
		{
			name:                  "BeyondCurrentSize",
			size:                  10,
			expectedSize:          10,
			expectedTruncatedSize: 10,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			wh := newUnbufferedWriteHandlerForTest(newFakeBucket(), nil)
			require.NoError(t, wh.Write(ctx, []byte(initialData), 0))

			err := wh.Truncate(tc.size)

			assert.ErrorIs(t, err, tc.expectedErr)
			assert.Equal(t, tc.expectedSize, wh.WriteFileInfo().TotalSize)
			assert.Equal(t, tc.expectedTruncatedSize, wh.truncatedSize)
		})
	}
}

func TestUnbufferedWriteHandler_CreatesWriterOnceWithPrecondition(t *testing.T) {
	testCases := []struct {
		name                 string
		obj                  *gcs.Object
		expectedPrecondition int64
	}{
		{
			name:                 "LocalFile",
			obj:                  nil,
			expectedPrecondition: 0,
		},
		{
			name:                 "EmptyObject",
			obj:                  &gcs.Object{Name: objectName, Size: 0, Generation: 7},
			expectedPrecondition: 7,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			writer := new(storagemock.Writer)
			writer.On("Write", mock.Anything).Return(3, nil)
			bucket, _ := newMockBucketWithWriter(writer)
			wh := newUnbufferedWriteHandlerForTest(bucket, tc.obj)

			require.NoError(t, wh.Write(ctx, []byte("foo"), 0))
			require.NoError(t, wh.Write(ctx, []byte("bar"), 3))

			bucket.AssertNumberOfCalls(t, "CreateObjectChunkWriter", 1)
			bucket.AssertCalled(t, "CreateObjectChunkWriter", mock.Anything, mock.MatchedBy(func(req *gcs.CreateObjectRequest) bool {
				return req.Name == objectName && req.GenerationPrecondition != nil && *req.GenerationPrecondition == tc.expectedPrecondition
			}))
			writer.AssertNumberOfCalls(t, "Write", 2)
			assert.Equal(t, int64(6), wh.WriteFileInfo().TotalSize)
		})
	}
}

func TestUnbufferedWriteHandler_WriterContextOutlivesOperationContext(t *testing.T) {
	writer := new(storagemock.Writer)
	writer.On("Write", mock.Anything).Return(3, nil)
	bucket, writerCtx := newMockBucketWithWriter(writer)
	wh := newUnbufferedWriteHandlerForTest(bucket, nil)
	opCtx, cancelOp := context.WithCancel(context.Background())
	require.NoError(t, wh.Write(opCtx, []byte("foo"), 0))

	cancelOp()

	assert.NoError(t, writerCtx().Err())
}

func TestUnbufferedWriteHandler_CreateObjectChunkWriterErrorIsSticky(t *testing.T) {
	testCases := []struct {
		name string
		// setup runs before the operation that creates the writer.
		setup     []handlerOp
		failingOp handlerOp
	}{
		{
			name:      "FirstWrite",
			failingOp: writeOp("foo", 0),
		},
		{
			name:      "WriteAtTruncatedSize",
			setup:     []handlerOp{truncateOp(2)},
			failingOp: writeOp("ab", 2),
		},
		{
			name:      "FlushOfEmptyFile",
			failingOp: flushOp(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			bucket := new(storagemock.TestifyMockBucket)
			bucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything).Return(nil, errUploadFailure).Once()
			wh := newUnbufferedWriteHandlerForTest(bucket, nil)
			for _, op := range tc.setup {
				require.NoError(t, op(ctx, wh))
			}
			require.ErrorIs(t, tc.failingOp(ctx, wh), errUploadFailure)

			writeErr := wh.Write(ctx, []byte("foo"), 0)
			_, syncErr := wh.Sync(ctx)
			_, flushErr := wh.Flush(ctx)

			assert.ErrorIs(t, writeErr, errUploadFailure)
			assert.ErrorIs(t, syncErr, errUploadFailure)
			assert.ErrorIs(t, flushErr, errUploadFailure)
			// No new upload is started.
			bucket.AssertNumberOfCalls(t, "CreateObjectChunkWriter", 1)
		})
	}
}

func TestUnbufferedWriteHandler_ZeroPaddingErrorIsSticky(t *testing.T) {
	testCases := []struct {
		name      string
		failingOp handlerOp
	}{
		{
			name:      "WriteAtTruncatedSize",
			failingOp: writeOp("cd", 4),
		},
		{
			name:      "Flush",
			failingOp: flushOp(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			writer := new(storagemock.Writer)
			writer.On("Write", []byte("ab")).Return(2, nil)
			writer.On("Write", []byte{0, 0}).Return(0, errUploadFailure)
			bucket, writerCtx := newMockBucketWithWriter(writer)
			bucket.On("AbortUpload", mock.Anything, writer).Return(nil)
			wh := newUnbufferedWriteHandlerForTest(bucket, nil)
			require.NoError(t, wh.Write(ctx, []byte("ab"), 0))
			require.NoError(t, wh.Truncate(4))

			err := tc.failingOp(ctx, wh)
			writeErr := wh.Write(ctx, []byte("cd"), 4)
			_, syncErr := wh.Sync(ctx)
			_, flushErr := wh.Flush(ctx)

			assert.ErrorIs(t, err, errUploadFailure)
			bucket.AssertNumberOfCalls(t, "AbortUpload", 1)
			assert.ErrorIs(t, writerCtx().Err(), context.Canceled)
			assert.ErrorIs(t, writeErr, errUploadFailure)
			assert.ErrorIs(t, syncErr, errUploadFailure)
			assert.ErrorIs(t, flushErr, errUploadFailure)
			// Only the data and the failed padding reach the writer: the write at
			// the truncated size isn't attempted, and nothing is finalized.
			writer.AssertNumberOfCalls(t, "Write", 2)
		})
	}
}

func TestUnbufferedWriteHandler_WriteErrorAbortsUploadAndIsSticky(t *testing.T) {
	ctx := context.Background()
	writer := new(storagemock.Writer)
	writer.On("Write", mock.Anything).Return(0, errUploadFailure)
	bucket, writerCtx := newMockBucketWithWriter(writer)
	bucket.On("AbortUpload", mock.Anything, writer).Return(nil)
	wh := newUnbufferedWriteHandlerForTest(bucket, nil)
	require.ErrorIs(t, wh.Write(ctx, []byte("foo"), 0), errUploadFailure)

	writeErr := wh.Write(ctx, []byte("foo"), 0)
	outOfOrderWriteErr := wh.Write(ctx, []byte("x"), 5)
	_, syncErr := wh.Sync(ctx)
	_, flushErr := wh.Flush(ctx)

	bucket.AssertNumberOfCalls(t, "AbortUpload", 1)
	assert.ErrorIs(t, writerCtx().Err(), context.Canceled)
	assert.ErrorIs(t, writeErr, errUploadFailure)
	// The upload error is returned even for an out-of-order write, so that the
	// caller doesn't fall back to staged writes.
	assert.ErrorIs(t, outOfOrderWriteErr, errUploadFailure)
	assert.ErrorIs(t, syncErr, errUploadFailure)
	assert.ErrorIs(t, flushErr, errUploadFailure)
	// The writer isn't used again and no new upload is started.
	writer.AssertNumberOfCalls(t, "Write", 1)
	bucket.AssertNumberOfCalls(t, "CreateObjectChunkWriter", 1)
}

func TestUnbufferedWriteHandler_WriteErrorIsReturnedWhenAbortFails(t *testing.T) {
	ctx := context.Background()
	writer := new(storagemock.Writer)
	writer.On("Write", mock.Anything).Return(0, errUploadFailure)
	bucket, writerCtx := newMockBucketWithWriter(writer)
	bucket.On("AbortUpload", mock.Anything, writer).Return(errAbortFailure)
	wh := newUnbufferedWriteHandlerForTest(bucket, nil)

	err := wh.Write(ctx, []byte("foo"), 0)

	assert.ErrorIs(t, err, errUploadFailure)
	assert.NotErrorIs(t, err, errAbortFailure)
	bucket.AssertNumberOfCalls(t, "AbortUpload", 1)
	assert.ErrorIs(t, writerCtx().Err(), context.Canceled)
}

func TestUnbufferedWriteHandler_WriteErrorKeepsGCSErrorType(t *testing.T) {
	testCases := []struct {
		name      string
		writeErr  error
		assertErr func(t *testing.T, err error)
	}{
		{
			name:     "NotFound",
			writeErr: status.Error(codes.NotFound, "object deleted"),
			assertErr: func(t *testing.T, err error) {
				var notFoundErr *gcs.NotFoundError
				assert.ErrorAs(t, err, &notFoundErr)
			},
		},
		{
			name:     "FailedPrecondition",
			writeErr: status.Error(codes.FailedPrecondition, "generation mismatch"),
			assertErr: func(t *testing.T, err error) {
				var preconditionErr *gcs.PreconditionError
				assert.ErrorAs(t, err, &preconditionErr)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			writer := new(storagemock.Writer)
			writer.On("Write", mock.Anything).Return(0, tc.writeErr)
			bucket, _ := newMockBucketWithWriter(writer)
			bucket.On("AbortUpload", mock.Anything, writer).Return(nil)
			wh := newUnbufferedWriteHandlerForTest(bucket, nil)

			err := wh.Write(context.Background(), []byte("foo"), 0)

			tc.assertErr(t, err)
		})
	}
}

func TestUnbufferedWriteHandler_FinalizeErrorAbortsUploadAndIsSticky(t *testing.T) {
	ctx := context.Background()
	writer := new(storagemock.Writer)
	writer.On("Write", mock.Anything).Return(3, nil)
	bucket, writerCtx := newMockBucketWithWriter(writer)
	bucket.On("FinalizeUpload", mock.Anything, writer).Return((*gcs.MinObject)(nil), errUploadFailure)
	bucket.On("AbortUpload", mock.Anything, writer).Return(nil)
	wh := newUnbufferedWriteHandlerForTest(bucket, nil)
	require.NoError(t, wh.Write(ctx, []byte("foo"), 0))
	_, err := wh.Flush(ctx)
	require.ErrorIs(t, err, errUploadFailure)

	writeErr := wh.Write(ctx, []byte("bar"), 3)
	_, syncErr := wh.Sync(ctx)
	_, flushErr := wh.Flush(ctx)

	bucket.AssertNumberOfCalls(t, "AbortUpload", 1)
	assert.ErrorIs(t, writerCtx().Err(), context.Canceled)
	assert.ErrorIs(t, writeErr, errUploadFailure)
	assert.ErrorIs(t, syncErr, errUploadFailure)
	assert.ErrorIs(t, flushErr, errUploadFailure)
	writer.AssertNumberOfCalls(t, "Write", 1)
	bucket.AssertNumberOfCalls(t, "FinalizeUpload", 1)
}

func TestUnbufferedWriteHandler_InvalidFinalizedObjectErrorIsSticky(t *testing.T) {
	testCases := []struct {
		name          string
		finalizedObj  *gcs.MinObject
		expectedError string
	}{
		{
			name:          "NilObject",
			finalizedObj:  nil,
			expectedError: "FinalizeUpload returned nil object",
		},
		{
			name:          "SizeMismatch",
			finalizedObj:  &gcs.MinObject{Name: objectName, Size: 1},
			expectedError: "could not upload entire data",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			writer := new(storagemock.Writer)
			writer.On("Write", mock.Anything).Return(3, nil)
			bucket, _ := newMockBucketWithWriter(writer)
			bucket.On("FinalizeUpload", mock.Anything, writer).Return(tc.finalizedObj, nil)
			wh := newUnbufferedWriteHandlerForTest(bucket, nil)
			require.NoError(t, wh.Write(ctx, []byte("foo"), 0))

			obj, err := wh.Flush(ctx)
			writeErr := wh.Write(ctx, []byte("bar"), 3)
			_, syncErr := wh.Sync(ctx)
			_, flushErr := wh.Flush(ctx)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
			assert.Nil(t, obj)
			assert.ErrorIs(t, writeErr, err)
			assert.ErrorIs(t, syncErr, err)
			assert.ErrorIs(t, flushErr, err)
			// The closed writer isn't used again: it isn't written to or finalized a
			// second time.
			writer.AssertNumberOfCalls(t, "Write", 1)
			bucket.AssertNumberOfCalls(t, "FinalizeUpload", 1)
		})
	}
}

func TestUnbufferedWriteHandler_FlushReleasesWriterContext(t *testing.T) {
	ctx := context.Background()
	writer := new(storagemock.Writer)
	writer.On("Write", mock.Anything).Return(3, nil)
	bucket, writerCtx := newMockBucketWithWriter(writer)
	bucket.On("FinalizeUpload", mock.Anything, writer).Return(&gcs.MinObject{Name: objectName, Size: 3}, nil)
	wh := newUnbufferedWriteHandlerForTest(bucket, nil)
	require.NoError(t, wh.Write(ctx, []byte("foo"), 0))

	obj, err := wh.Flush(ctx)

	require.NoError(t, err)
	assert.Equal(t, uint64(3), obj.Size)
	assert.ErrorIs(t, writerCtx().Err(), context.Canceled)
}

func TestUnbufferedWriteHandler_UnlinkAbortsUpload(t *testing.T) {
	testCases := []struct {
		name     string
		abortErr error
	}{
		{
			name: "AbortSucceeds",
		},
		{
			name:     "AbortFails",
			abortErr: errAbortFailure,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			writer := new(storagemock.Writer)
			writer.On("Write", mock.Anything).Return(3, nil)
			bucket, writerCtx := newMockBucketWithWriter(writer)
			var writerCtxErrOnAbort error
			bucket.On("AbortUpload", mock.Anything, writer).
				Run(func(mock.Arguments) { writerCtxErrOnAbort = writerCtx().Err() }).
				Return(tc.abortErr)
			wh := newUnbufferedWriteHandlerForTest(bucket, nil)
			require.NoError(t, wh.Write(ctx, []byte("foo"), 0))

			wh.Unlink(ctx)

			bucket.AssertNumberOfCalls(t, "AbortUpload", 1)
			// The upload is aborted before its context is canceled.
			assert.NoError(t, writerCtxErrOnAbort)
			assert.ErrorIs(t, writerCtx().Err(), context.Canceled)
		})
	}
}

// opCtxKey marks the context of an operation on the handler.
type opCtxKey struct{}

func TestUnbufferedWriteHandler_AbortRunsOnOperationContext(t *testing.T) {
	testCases := []struct {
		name  string
		abort func(ctx context.Context, wh *unbufferedWriteHandlerImpl)
	}{
		{
			name:  "Unlink",
			abort: func(ctx context.Context, wh *unbufferedWriteHandlerImpl) { wh.Unlink(ctx) },
		},
		{
			name:  "Destroy",
			abort: func(ctx context.Context, wh *unbufferedWriteHandlerImpl) { _ = wh.Destroy(ctx) },
		},
		{
			name:  "WriteError",
			abort: func(ctx context.Context, wh *unbufferedWriteHandlerImpl) { _ = wh.Write(ctx, []byte("bar"), 3) },
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			writer := new(storagemock.Writer)
			writer.On("Write", []byte("foo")).Return(3, nil)
			writer.On("Write", []byte("bar")).Return(0, errUploadFailure)
			bucket, _ := newMockBucketWithWriter(writer)
			var abortCtx context.Context
			bucket.On("AbortUpload", mock.Anything, writer).
				Run(func(args mock.Arguments) { abortCtx = args.Get(0).(context.Context) }).
				Return(nil)
			wh := newUnbufferedWriteHandlerForTest(bucket, nil)
			require.NoError(t, wh.Write(context.Background(), []byte("foo"), 0))
			opCtx := context.WithValue(context.Background(), opCtxKey{}, tc.name)

			tc.abort(opCtx, wh)

			// The abort belongs to the operation, so that its metrics and trace span
			// are linked to it.
			require.NotNil(t, abortCtx)
			assert.Equal(t, tc.name, abortCtx.Value(opCtxKey{}))
		})
	}
}

func TestUnbufferedWriteHandler_OperationsAfterUnlinkDoNotTouchWriter(t *testing.T) {
	testCases := []struct {
		name string
		// setup runs before the unlink. writeN and writeErr are what the writer
		// returns for its write of "foo".
		setup    []handlerOp
		writeN   int
		writeErr error
		// nextOffset is the offset of the next sequential write.
		nextOffset      int64
		expectedWriters int
		expectedAborts  int
	}{
		{
			name: "BeforeAnyWrite",
		},
		{
			name:            "AfterWrite",
			setup:           []handlerOp{writeOp("foo", 0)},
			writeN:          3,
			nextOffset:      3,
			expectedWriters: 1,
			expectedAborts:  1,
		},
		{
			name:            "AfterUploadError",
			setup:           []handlerOp{writeOp("foo", 0)},
			writeErr:        errUploadFailure,
			expectedWriters: 1,
			expectedAborts:  1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			writer := new(storagemock.Writer)
			writer.On("Write", []byte("foo")).Return(tc.writeN, tc.writeErr)
			bucket, _ := newMockBucketWithWriter(writer)
			bucket.On("AbortUpload", mock.Anything, writer).Return(nil)
			wh := newUnbufferedWriteHandlerForTest(bucket, nil)
			for _, op := range tc.setup {
				require.ErrorIs(t, op(ctx, wh), tc.writeErr)
			}
			wh.Unlink(ctx)

			writeErr := wh.Write(ctx, []byte("bar"), tc.nextOffset)
			outOfOrderWriteErr := wh.Write(ctx, []byte("x"), 100)
			syncObj, syncErr := wh.Sync(ctx)
			flushObj, flushErr := wh.Flush(ctx)

			assert.NoError(t, writeErr)
			assert.NoError(t, outOfOrderWriteErr)
			assert.Nil(t, syncObj)
			assert.NoError(t, syncErr)
			assert.Nil(t, flushObj)
			assert.NoError(t, flushErr)
			// No writer is created after the unlink, and the upload is aborted at
			// most once.
			bucket.AssertNumberOfCalls(t, "CreateObjectChunkWriter", tc.expectedWriters)
			bucket.AssertNumberOfCalls(t, "AbortUpload", tc.expectedAborts)
		})
	}
}

func TestUnbufferedWriteHandler_TruncateBelowSizeAfterUnlinkDoesNotFallBack(t *testing.T) {
	ctx := context.Background()
	writer := new(storagemock.Writer)
	writer.On("Write", mock.Anything).Return(3, nil)
	bucket, _ := newMockBucketWithWriter(writer)
	bucket.On("AbortUpload", mock.Anything, writer).Return(nil)
	wh := newUnbufferedWriteHandlerForTest(bucket, nil)
	require.NoError(t, wh.Write(ctx, []byte("foo"), 0))
	wh.Unlink(ctx)

	err := wh.Truncate(1)

	assert.NoError(t, err)
}

func TestUnbufferedWriteHandler_FlushAfterUnlinkDoesNotCreateObject(t *testing.T) {
	ctx := context.Background()
	bucket := newFakeBucket()
	wh := newUnbufferedWriteHandlerForTest(bucket, nil)
	require.NoError(t, wh.Write(ctx, []byte("foo"), 0))
	wh.Unlink(ctx)

	obj, err := wh.Flush(ctx)

	require.NoError(t, err)
	assert.Nil(t, obj)
	_, _, err = bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: objectName})
	var notFoundErr *gcs.NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestUnbufferedWriteHandler_DestroyAbortsUnfinalizedUpload(t *testing.T) {
	testCases := []struct {
		name     string
		abortErr error
	}{
		{
			name: "AbortSucceeds",
		},
		{
			name:     "AbortFails",
			abortErr: errAbortFailure,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			writer := new(storagemock.Writer)
			writer.On("Write", mock.Anything).Return(3, nil)
			bucket, writerCtx := newMockBucketWithWriter(writer)
			bucket.On("AbortUpload", mock.Anything, writer).Return(tc.abortErr)
			wh := newUnbufferedWriteHandlerForTest(bucket, nil)
			require.NoError(t, wh.Write(ctx, []byte("foo"), 0))

			err := wh.Destroy(ctx)

			assert.ErrorIs(t, err, tc.abortErr)
			bucket.AssertNumberOfCalls(t, "AbortUpload", 1)
			assert.ErrorIs(t, writerCtx().Err(), context.Canceled)
		})
	}
}

func TestUnbufferedWriteHandler_DestroyWithoutActiveUploadDoesNotAbort(t *testing.T) {
	testCases := []struct {
		name string
		// setup runs before Destroy; each op returns setupErr.
		setup    []handlerOp
		setupErr error
		// expectedAborts is the number of aborts before Destroy.
		expectedAborts int
	}{
		{
			name: "NoUpload",
		},
		{
			name:  "FinalizedUpload",
			setup: []handlerOp{writeOp("foo", 0), flushOp()},
		},
		{
			name:           "AbortedUpload",
			setup:          []handlerOp{writeOp("foo", 0), unlinkOp()},
			expectedAborts: 1,
		},
		{
			name:           "FailedUpload",
			setup:          []handlerOp{writeOp("bar", 0)},
			setupErr:       errUploadFailure,
			expectedAborts: 1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			writer := new(storagemock.Writer)
			writer.On("Write", []byte("foo")).Return(3, nil)
			writer.On("Write", []byte("bar")).Return(0, errUploadFailure)
			bucket, _ := newMockBucketWithWriter(writer)
			bucket.On("FinalizeUpload", mock.Anything, writer).Return(&gcs.MinObject{Name: objectName, Size: 3}, nil)
			bucket.On("AbortUpload", mock.Anything, writer).Return(nil)
			wh := newUnbufferedWriteHandlerForTest(bucket, nil)
			for _, op := range tc.setup {
				require.ErrorIs(t, op(ctx, wh), tc.setupErr)
			}

			err := wh.Destroy(ctx)

			assert.NoError(t, err)
			bucket.AssertNumberOfCalls(t, "AbortUpload", tc.expectedAborts)
		})
	}
}
