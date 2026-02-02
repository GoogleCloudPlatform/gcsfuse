// Copyright 2024 Google LLC
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

// Note: All the write operations take inode lock in fs.go, hence we don't need
// any locks here as we will get calls to these methods serially.

package bufferedwrites

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// UploadHandler is responsible for synchronized uploads of the filled blocks
// to GCS and then putting them back for reuse once the block has been uploaded.
type UploadHandler struct {
	// Channel for receiving blocks to be uploaded to GCS.
	uploadCh chan block.Block

	// Wait group for waiting for the uploader goroutine to finish.
	wg sync.WaitGroup

	// Used to release the free (uploaded) block back to the pool.
	blockPool *block.GenBlockPool[block.Block]

	// writer to resumable upload the blocks to GCS.
	writer gcs.Writer

	// uploadError stores atomic pointer to the error seen by uploader.
	uploadError atomic.Pointer[error]
	// CancelFunc persisted to cancel the uploads in case of unlink operation.
	cancelFunc    context.CancelFunc
	startUploader sync.Once

	// Parameters required for creating a new GCS chunk writer.
	bucket               gcs.Bucket
	objectName           string
	obj                  *gcs.Object
	chunkTransferTimeout int64
	chunkRetryDeadline   int64
	blockSize            int64
}

type CreateUploadHandlerRequest struct {
	Object                   *gcs.Object
	ObjectName               string
	Bucket                   gcs.Bucket
	BlockPool                *block.GenBlockPool[block.Block]
	MaxBlocksPerFile         int64
	BlockSize                int64
	ChunkTransferTimeoutSecs int64
	ChunkRetryDeadlineSecs   int64
}

// newUploadHandler creates the UploadHandler struct.
func newUploadHandler(req *CreateUploadHandlerRequest) *UploadHandler {
	uh := &UploadHandler{
		uploadCh:             make(chan block.Block, req.MaxBlocksPerFile),
		wg:                   sync.WaitGroup{},
		blockPool:            req.BlockPool,
		bucket:               req.Bucket,
		objectName:           req.ObjectName,
		obj:                  req.Object,
		blockSize:            req.BlockSize,
		chunkTransferTimeout: req.ChunkTransferTimeoutSecs,
		chunkRetryDeadline:   req.ChunkRetryDeadlineSecs,
	}
	return uh
}

// Upload adds a block to the upload queue.
func (uh *UploadHandler) Upload(block block.Block) error {
	uh.wg.Add(1)

	err := uh.ensureWriter()
	if err != nil {
		return fmt.Errorf("uh.ensureWriter() failed: %v", err)
	}
	// Start the uploader goroutine but only once.
	uh.startUploader.Do(func() {
		go uh.uploader()
	})
	uh.uploadCh <- block
	return nil
}

// createObjectWriter creates a GCS object writer.
func (uh *UploadHandler) createObjectWriter() (err error) {
	req := gcs.NewCreateObjectRequest(uh.obj, uh.objectName, nil, uh.chunkTransferTimeout, uh.chunkRetryDeadline)
	// We need a new context here, since the first writeFile() call will be complete
	// (and context will be cancelled) by the time complete upload is done.
	var ctx context.Context
	ctx, uh.cancelFunc = context.WithCancel(context.Background())
	if uh.bucket.BucketType().Zonal && (uh.obj != nil && uh.obj.Finalized.IsZero()) {
		chunkWriterReq := gcs.CreateObjectChunkWriterRequest{
			CreateObjectRequest: *req,
			ChunkSize:           int(uh.blockSize),
			Offset:              int64(uh.obj.Size),
		}
		uh.writer, err = uh.bucket.CreateAppendableObjectWriter(ctx, &chunkWriterReq)
	} else {
		uh.writer, err = uh.bucket.CreateObjectChunkWriter(ctx, req, int(uh.blockSize), nil)
	}
	return
}

func (uh *UploadHandler) UploadError() (err error) {
	if uploadError := uh.uploadError.Load(); uploadError != nil {
		err = *uploadError
	}
	return
}

// uploader is the single-threaded goroutine that uploads blocks.
func (uh *UploadHandler) uploader() {
	for currBlock := range uh.uploadCh {
		uh.uploadBlock(currBlock)

		// Put back the uploaded block to the pool for re-use,
		// irrespective of whether the upload was successful or not.
		uh.blockPool.Release(currBlock)
		uh.wg.Done()
	}
}

// uploadBlock uploads the block content to GCS writer.
// It is called by the uploader goroutine.
// If the block is nil, it logs a warning and returns.
// If there is already an error in uploadError, it returns without doing anything.
// If there is an error during upload, it returns after storing the error in uploadError.
func (uh *UploadHandler) uploadBlock(b block.Block) {
	if b == nil {
		logger.Warnf("uploadBlock: received nil block for object %s", uh.objectName)
		return
	}

	if uh.UploadError() != nil {
		return
	}

	// Reset the readSeek to 0 before uploading.
	if off, err := b.Seek(0, io.SeekStart); err != nil || off != 0 {
		err := fmt.Errorf("buffered write upload failed for object %s: error in block.Seek: %v with offset: %d", uh.objectName, err, off)
		uh.uploadError.Store(&err)
		logger.Errorf("uploadBlock: %v", err)
		return
	}

	_, err := io.Copy(uh.writer, b)
	if errors.Is(err, context.Canceled) {
		// Context canceled error indicates that the file was deleted from the
		// same mount. In this case, we suppress the error to match local
		// filesystem behavior.
		err = nil
	}
	if err != nil {
		err = gcs.GetGCSError(err)
		uh.uploadError.Store(&err)
		logger.Errorf("uploadBlock: failed for object %s: error in io.Copy: %v", uh.objectName, err)
	}
}

// Finalize finalizes the upload.
func (uh *UploadHandler) Finalize() (*gcs.MinObject, error) {
	uh.wg.Wait()
	close(uh.uploadCh)

	// Writer may not have been created for empty file creation flow or for very
	// small writes of size less than 1 block.
	err := uh.ensureWriter()
	if err != nil {
		return nil, fmt.Errorf("uh.ensureWriter() failed: %v", err)
	}

	obj, err := uh.bucket.FinalizeUpload(context.Background(), uh.writer)
	if err != nil {
		// FinalizeUpload already returns GCSerror so no need to convert again.
		uh.uploadError.Store(&err)
		logger.Errorf("FinalizeUpload failed for object %s: %v", uh.objectName, err)
		return nil, err
	}
	return obj, nil
}

func (uh *UploadHandler) ensureWriter() error {
	if uh.writer == nil {
		if err := uh.createObjectWriter(); err != nil {
			return fmt.Errorf("createObjectWriter failed for object %s: %w", uh.objectName, err)
		}
	}
	return nil
}

// FlushPendingWrites uploads any data in the write buffer.
func (uh *UploadHandler) FlushPendingWrites() (*gcs.MinObject, error) {
	uh.wg.Wait()

	// Writer may not have been created for empty file creation flow or for very
	// small writes of size less than 1 block.
	err := uh.ensureWriter()
	if err != nil {
		return nil, fmt.Errorf("uh.ensureWriter() failed: %v", err)
	}

	o, err := uh.bucket.FlushPendingWrites(context.Background(), uh.writer)
	if err != nil {
		// FlushUpload already returns GCS error so no need to convert again.
		uh.uploadError.Store(&err)
		logger.Errorf("FlushUpload failed for object %s: %v", uh.objectName, err)
		return nil, err
	}
	return o, nil
}

func (uh *UploadHandler) CancelUpload() {
	if uh.cancelFunc != nil {
		// cancel the context to cancel the ongoing GCS upload.
		uh.cancelFunc()
	}
	// Wait for all in progress buffers to be added to the free channel.
	uh.wg.Wait()
}

func (uh *UploadHandler) AwaitBlocksUpload() {
	uh.wg.Wait()
}

func (uh *UploadHandler) Destroy() {
	// Move all pending blocks to freeBlockCh and close the channel if not done.
	for {
		select {
		case currBlock, ok := <-uh.uploadCh:
			// Not ok means channel closed. Return.
			if !ok {
				return
			}
			uh.blockPool.Release(currBlock)
			// Marking as wg.Done to ensure any waiters are unblocked.
			uh.wg.Done()
		default:
			// This will get executed when there are no blocks pending in uploadCh and its not closed.
			close(uh.uploadCh)
			return
		}
	}
}
