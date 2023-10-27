package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"google.golang.org/api/googleapi"
)

type UploaderState int

const (
	// Initialized is the state of successful initialization, no writes so far.
	Initialized UploaderState = iota
	// Uploading is the state when an asynchronous upload is in progress.
	Uploading
	// UploadError is the state when an Upload failed.
	UploadError
	// Closed is the state of an uploader which has been finalized.
	Closed
)

// A chunkUploader is an implementation of ChunkUploader
// interface, which uses storage.Writer from go-storage-client
// for resumable upload.
//
// It also stores the current state of the uploader.
type chunkUploader struct {
	// Internal objects for business logic.
	writer     *storage.Writer
	objectName string
	mu         sync.Mutex

	// Attributes for providing updates to user.
	totalWriteInitiatedSoFar int64
	// Total number of bytes successfully written by this writer so far.
	totalWriteSucceededSoFar atomic.Int64
	userProgressFunc         func(int64)

	// Internal state for lifecycle management.
	state UploaderState
}

// NewChunkUploader creates a new instance of chunkUploader,
// for the given inputs.
func NewChunkUploader(ctx context.Context, obj *storage.ObjectHandle, req *gcs.CreateChunkUploaderRequest, writeChunkSize int, progressFunc func(int64)) (gcs.ChunkUploader, error) {
	if ctx == nil {
		return nil, fmt.Errorf("ctx is nil")
	}
	if obj == nil || req == nil {
		return nil, fmt.Errorf("nil ObjectHandle or CreateChunkUploaderRequest")
	}

	if obj.ObjectName() != req.Name {
		return nil, fmt.Errorf("names of passed ObjectHandle and CreateChunkUploaderRequest don't match: ObjectHandle.Name=%s CreateChunkUploaderRequest.Name=%s", obj.ObjectName(), req.Name)
	}

	if req.GenerationPrecondition != nil && *req.GenerationPrecondition != 0 {
		return nil, fmt.Errorf("request received for pre-existing object %s, supported only for new objects", req.Name)
	}

	if writeChunkSize <= 0 {
		return nil, fmt.Errorf("chunkSize <= 0")
	}

	// Raw initialization.
	uploader := chunkUploader{}

	// Store references to necessary parameters.
	uploader.objectName = obj.ObjectName()
	uploader.userProgressFunc = progressFunc

	// Create a NewWriter with the requested attributes, using Go Storage Client.
	// NewWriter never returns nil, so no nil-check is needed on it.
	wc := obj.NewWriter(ctx)
	wc = storageutil.SetAttrsInWriter(wc, &gcs.CreateObjectRequest{CreateChunkUploaderRequest: *req})
	wc.ChunkSize = writeChunkSize
	wc.ProgressFunc = func(n int64) {
		uploader.totalWriteSucceededSoFar.Store(n)
		logger.Debugf("%d bytes copied so far for object/file %s. chunk-size = %d", n, req.Name, wc.ChunkSize)

		if uploader.userProgressFunc != nil {
			uploader.userProgressFunc(n)
		}
	}

	uploader.writer = wc
	uploader.state = Initialized

	return &uploader, nil
}

// BytesUploadedSoFar returns the total number of bytes successfully uploaded
// so far using this uploader.
// This is thread-safe against the
// in-progress calls to BytesUploadedSoFar/UploadChunkAsync/Close
// invoked from other threads/go-routines.
func (uploader *chunkUploader) BytesUploadedSoFar() int64 {
	return uploader.totalWriteSucceededSoFar.Load()
}

func (uploader *chunkUploader) readyToUpload() bool {
	switch uploader.state {
	case Initialized, Uploading:
		return true
	default:
		return false
	}
}

func (uploader *chunkUploader) readyToClose() bool {
	switch uploader.state {
	case Initialized, Uploading, UploadError:
		return true
	default:
		return false
	}
}

// UploadChunkAsync uploads the passed content to GCS.
// This waits (using mutex) until the completion of
// in-progress calls to BytesUploadedSoFar/UploadChunkAsync/Close
// invoked from other threads/go-routines.
// The call times out if it is taking longer than StorageCopyTimeoutDuration.
func (uploader *chunkUploader) UploadChunkAsync(contents io.Reader) error {
	uploader.mu.Lock()
	defer uploader.mu.Unlock()

	if !uploader.readyToUpload() {
		return fmt.Errorf("writer not ready to write: object: %s, status = %v, writer: %v", uploader.objectName, uploader.state, uploader.writer)
	}

	n, err := io.Copy(uploader.writer, contents)
	if err != nil {
		uploader.state = UploadError
		return fmt.Errorf("upload failed for object %s: totalSizeUploaded-so-far=%d, successfully-uploaded-in-last-upload=%d, chunk-size=%d, %v", uploader.objectName, uploader.BytesUploadedSoFar(),
			n, uploader.writer.ChunkSize,
			err)
	}
	if n == 0 {
		return nil
	}

	uploader.totalWriteInitiatedSoFar += n
	uploader.state = Uploading
	return nil
}

// Close finalizes the chunk uploads and returns the
// created GCS object.
// If a chunk upload was in progress at the time of call,
// it will be waited on and be completed before going ahead
// with the finalization.
// This waits (using mutex) until the completion of
// in-progress calls to BytesUploadedSoFar/UploadChunkAsync/Close
// invoked from other threads/go-routines.
func (uploader *chunkUploader) Close() (*gcs.Object, error) {
	uploader.mu.Lock()
	defer uploader.mu.Unlock()

	defer func() {
		uploader.state = Closed
		uploader.writer = nil
	}()

	if !uploader.readyToClose() {
		return nil, fmt.Errorf("improper state (%v) for finalizing object %s", uploader.state, uploader.objectName)
	}

	if err := uploader.writer.Close(); err != nil {
		var gErr *googleapi.Error
		if errors.As(err, &gErr) {
			if gErr.Code == http.StatusPreconditionFailed {
				return nil, &gcs.PreconditionError{Err: err}
			}
		}
		return nil, fmt.Errorf("error in closing writer : %w", err)
	}

	// Retrieving the attributes of the created object.
	attrs := uploader.writer.Attrs()
	// Converting attrs to type *Object.
	return storageutil.ObjectAttrsToBucketObject(attrs), nil
}
