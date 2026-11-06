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
	"fmt"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v4/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v4/tracing"
)

// zeroPaddingChunkSize is the size of the zero-filled buffer used to pad the
// object up to the size requested by a forward truncate.
const zeroPaddingChunkSize = 1024 * 1024

// errNotImplemented is returned by the unbufferedWriteHandlerImpl methods that
// are not implemented yet.
// TODO(b/581184283): Remove once Destroy is implemented.
var errNotImplemented = errors.New("unbuffered write handler: not implemented")

// unbufferedWriteHandlerImpl is the WriteHandler for parallel uploads. It
// passes writes straight to a Go SDK writer, which uploads the object in parts
// concurrently, so GCSFuse keeps no write buffers of its own. Only sequential
// writes are supported. Out-of-order writes and truncates below the current
// size return ErrOutOfOrderWrite, so that the caller falls back to staged
// writes.
//
// All methods assume the caller holds the FileInode lock, so no internal
// synchronization is needed.
//
// TODO(b/581184283): Implement Destroy and Unlink.
// TODO(b/581184469): Nothing creates this handler yet.
type unbufferedWriteHandlerImpl struct {
	// writer uploads the object. It's created on the first write, on a sync
	// that writes zero padding, or on flush.
	writer gcs.Writer
	// cancelFunc cancels the context of the writer.
	cancelFunc context.CancelFunc

	// Parameters for creating the writer.
	bucket                   gcs.Bucket
	objectName               string
	obj                      *gcs.Object
	blockSize                int64
	chunkRetryDeadlineSecs   int64
	chunkTransferTimeoutSecs int64
	traceHandle              tracing.TraceHandle

	// totalSize is the number of bytes handed to the writer so far, including
	// any zero padding already written.
	totalSize int64
	// truncatedSize is the size requested by a pending forward truncate, or -1
	// if there is none. The gap up to it is zero-padded lazily, on the next
	// write at truncatedSize, Sync or Flush.
	truncatedSize int64
	// mtime is the file's mtime. It starts at the handler's creation time and is
	// updated by the kernel through SetInodeAttributes.
	mtime time.Time

	// uploadError is the first error from the writer, including a failed writer
	// creation. Once set, Write, Sync and Flush return it without touching the
	// writer, so no new upload is started.
	uploadError error
}

var _ WriteHandler = (*unbufferedWriteHandlerImpl)(nil)

// newUnbufferedWriteHandler creates an unbufferedWriteHandlerImpl for the
// object described by req. Callers must only create this handler for new files
// or empty objects (size 0), so totalSize starts at 0.
func newUnbufferedWriteHandler(req *CreateWriteHandlerRequest) *unbufferedWriteHandlerImpl {
	return &unbufferedWriteHandlerImpl{
		bucket:                   req.Bucket,
		objectName:               req.ObjectName,
		obj:                      req.Object,
		blockSize:                req.BlockSize,
		chunkRetryDeadlineSecs:   req.ChunkRetryDeadlineSecs,
		chunkTransferTimeoutSecs: req.ChunkTransferTimeoutSecs,
		traceHandle:              req.TraceHandle,
		totalSize:                0,
		truncatedSize:            -1,
		mtime:                    time.Now(),
	}
}

func (wh *unbufferedWriteHandlerImpl) Write(ctx context.Context, data []byte, offset int64) error {
	if wh.uploadError != nil {
		return wh.uploadError
	}
	if !isSequentialWrite(offset, wh.totalSize, wh.truncatedSize) {
		logger.Errorf("UnbufferedWriteHandler.OutOfOrderError for object: %s, expectedOffset: %d, actualOffset: %d",
			wh.objectName, wh.totalSize, offset)
		return ErrOutOfOrderWrite
	}
	// A write at the truncated size first fills the gap up to it with zeros.
	if offset == wh.truncatedSize {
		if err := wh.writeZeroPadding(ctx); err != nil {
			return err
		}
	}
	return wh.write(ctx, data)
}

// Sync writes any pending zero padding to the writer and returns a nil object.
// It neither waits for the parts in flight nor finalizes the object, so the
// data isn't durable or visible on GCS until Flush. Without pending padding it
// doesn't touch the writer, so the sync that follows the handler's creation
// doesn't start an upload.
func (wh *unbufferedWriteHandlerImpl) Sync(ctx context.Context) (*gcs.MinObject, error) {
	if wh.uploadError != nil {
		return nil, wh.uploadError
	}
	return nil, wh.writeZeroPadding(ctx)
}

// Flush writes any pending zero padding and finalizes the object.
func (wh *unbufferedWriteHandlerImpl) Flush(ctx context.Context) (obj *gcs.MinObject, err error) {
	if wh.uploadError != nil {
		return nil, wh.uploadError
	}
	// Finalize under a context that is detached from the FUSE operation but
	// carries its trace.
	ctx = wh.traceHandle.PropagateTraceContext(context.Background(), ctx)
	var bytes int64
	ctx, finishSpan := wh.traceHandle.TraceUpload(ctx, tracing.StreamingUploadFinalize, wh.objectName, &bytes, &err)
	defer finishSpan()

	if err = wh.writeZeroPadding(ctx); err != nil {
		return nil, err
	}
	// An empty file has no writer yet: create one to create the empty object.
	if err = wh.ensureWriter(ctx); err != nil {
		return nil, err
	}
	obj, err = wh.bucket.FinalizeUpload(ctx, wh.writer)
	if err != nil {
		// FinalizeUpload already converts the error to a GCS error.
		err = fmt.Errorf("FinalizeUpload failed for object %s: %w", wh.objectName, err)
		wh.handleUploadFailure(err)
		return nil, err
	}
	// The upload is complete, so release the context of the writer.
	wh.cancelFunc()
	// The writer is closed now, so the errors below are sticky too: a later Flush
	// from another write handle must not finalize the object again.
	if obj == nil {
		err = fmt.Errorf("FinalizeUpload returned nil object for %s", wh.objectName)
		wh.handleUploadFailure(err)
		return nil, err
	}
	bytes = int64(obj.Size)
	if obj.Size != uint64(wh.totalSize) {
		err = fmt.Errorf("could not upload entire data, expected size %d, got %d", wh.totalSize, obj.Size)
		wh.handleUploadFailure(err)
		return nil, err
	}
	return obj, nil
}

func (wh *unbufferedWriteHandlerImpl) SetMtime(mtime time.Time) {
	wh.mtime = mtime
}

// Truncate records a forward truncate, which is zero-padded lazily. A truncate
// below the current size returns ErrOutOfOrderWrite, so that the caller falls
// back to staged writes.
func (wh *unbufferedWriteHandlerImpl) Truncate(size int64) error {
	if size < wh.totalSize {
		return ErrOutOfOrderWrite
	}
	// A truncate to the current size cancels any pending forward truncate.
	if size == wh.totalSize {
		wh.truncatedSize = -1
		return nil
	}
	wh.truncatedSize = size
	return nil
}

func (wh *unbufferedWriteHandlerImpl) WriteFileInfo() WriteFileInfo {
	return WriteFileInfo{
		TotalSize: max(wh.totalSize, wh.truncatedSize),
		Mtime:     wh.mtime,
	}
}

func (wh *unbufferedWriteHandlerImpl) Destroy() error {
	return errNotImplemented
}

func (wh *unbufferedWriteHandlerImpl) Unlink() {
	// TODO(b/581184283): Abort the upload and discard later writes.
}

// write writes data to the writer, creating the writer if needed.
func (wh *unbufferedWriteHandlerImpl) write(ctx context.Context, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if err := wh.ensureWriter(ctx); err != nil {
		return err
	}
	// The writer copies data before Write returns, so the caller can reuse the
	// buffer.
	n, err := wh.writer.Write(data)
	wh.totalSize += int64(n)
	if err != nil {
		// The writer uploads in the background, so this may also be the error of
		// an earlier write.
		// TODO(b/577926767): Map errors for files that exceed the size limit to
		// EFBIG.
		err = fmt.Errorf("write failed for object %s: %w", wh.objectName, gcs.GetGCSError(err))
		wh.handleUploadFailure(err)
		return err
	}
	// Once the file reaches the truncated size, the truncate is complete.
	if wh.truncatedSize != -1 && wh.totalSize >= wh.truncatedSize {
		wh.truncatedSize = -1
	}
	return nil
}

// writeZeroPadding pads the object with zeros up to the size requested by a
// forward truncate.
func (wh *unbufferedWriteHandlerImpl) writeZeroPadding(ctx context.Context) error {
	if wh.truncatedSize <= wh.totalSize {
		return nil
	}
	// Reuse one zero-filled buffer of at most zeroPaddingChunkSize bytes, to
	// bound memory use for large truncates.
	zeros := make([]byte, min(zeroPaddingChunkSize, wh.truncatedSize-wh.totalSize))
	for wh.truncatedSize > wh.totalSize {
		if err := wh.write(ctx, zeros[:min(int64(len(zeros)), wh.truncatedSize-wh.totalSize)]); err != nil {
			return err
		}
	}
	return nil
}

// ensureWriter creates the writer if it hasn't been created yet.
func (wh *unbufferedWriteHandlerImpl) ensureWriter(ctx context.Context) error {
	if wh.writer != nil {
		return nil
	}
	req := gcs.NewCreateObjectRequest(wh.obj, wh.objectName, nil, wh.chunkRetryDeadlineSecs, wh.chunkTransferTimeoutSecs)
	// ctx is scoped to the FUSE operation that creates the writer (e.g.
	// WriteFile) and may be canceled when that operation returns, but the writer
	// must live until the upload is finalized or canceled. So the writer gets its
	// own context, which carries only the operation's trace.
	writerCtx, cancel := context.WithCancel(wh.traceHandle.PropagateTraceContext(context.Background(), ctx))
	writer, err := wh.bucket.CreateObjectChunkWriter(writerCtx, req, int(wh.blockSize), nil)
	if err != nil {
		cancel()
		err = fmt.Errorf("CreateObjectChunkWriter failed for object %s: %w", wh.objectName, err)
		wh.handleUploadFailure(err)
		return err
	}
	wh.writer = writer
	wh.cancelFunc = cancel
	return nil
}

// handleUploadFailure makes err the sticky error of the upload and cancels the
// upload.
func (wh *unbufferedWriteHandlerImpl) handleUploadFailure(err error) {
	wh.uploadError = err
	if wh.cancelFunc != nil {
		// TODO(b/581184283): Also abort the upload, so that the parts uploaded so
		// far are discarded.
		wh.cancelFunc()
	}
}
