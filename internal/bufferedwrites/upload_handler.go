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

	"github.com/googlecloudplatform/gcsfuse/v2/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

// UploadHandler is responsible for synchronized uploads of the filled blocks
// to GCS and then putting them back for reuse once the block has been uploaded.
type UploadHandler struct {
	// Channel for receiving blocks to be uploaded to GCS.
	uploadCh chan block.Block

	// Wait group for waiting for the uploader goroutine to finish.
	wg sync.WaitGroup

	// Channel on which uploaded block will be posted for reuse.
	freeBlocksCh chan block.Block

	// writer to resumable upload the blocks to GCS.
	writer gcs.Writer

	// uploadError stores atomic pointer to the error seen by uploader.
	uploadError atomic.Pointer[error]
	// CancelFunc persisted to cancel the uploads in case of unlink operation.
	cancelFunc context.CancelFunc

	// Parameters required for creating a new GCS chunk writer.
	bucket               gcs.Bucket
	objectName           string
	obj                  *gcs.Object
	chunkTransferTimeout int64
	blockSize            int64
}

type CreateUploadHandlerRequest struct {
	Object                   *gcs.Object
	ObjectName               string
	Bucket                   gcs.Bucket
	FreeBlocksCh             chan block.Block
	MaxBlocksPerFile         int64
	BlockSize                int64
	ChunkTransferTimeoutSecs int64
}

// newUploadHandler creates the UploadHandler struct.
func newUploadHandler(req *CreateUploadHandlerRequest) *UploadHandler {
	uh := &UploadHandler{
		uploadCh:             make(chan block.Block, req.MaxBlocksPerFile),
		wg:                   sync.WaitGroup{},
		freeBlocksCh:         req.FreeBlocksCh,
		bucket:               req.Bucket,
		objectName:           req.ObjectName,
		obj:                  req.Object,
		blockSize:            req.BlockSize,
		chunkTransferTimeout: req.ChunkTransferTimeoutSecs,
	}
	return uh
}

// Upload adds a block to the upload queue.
func (uh *UploadHandler) Upload(block block.Block) error {
	uh.wg.Add(1)

	if uh.writer == nil {
		// Lazily create the object writer.
		err := uh.createObjectWriter()
		if err != nil {
			// createObjectWriter can only fail here due to throttling, so we will not
			// handle this error explicitly or fall back to temp file flow.
			return fmt.Errorf("createObjectWriter failed for object %s: %w", uh.objectName, err)
		}
		// Start the uploader goroutine.
		go uh.uploader()
	}

	uh.uploadCh <- block
	return nil
}

// createObjectWriter creates a GCS object writer.
func (uh *UploadHandler) createObjectWriter() (err error) {
	// TODO: b/381479965: Dynamically set chunkTransferTimeoutSecs based on chunk size. 0 here means no timeout.
	req := gcs.NewCreateObjectRequest(uh.obj, uh.objectName, nil, 0)
	// We need a new context here, since the first writeFile() call will be complete
	// (and context will be cancelled) by the time complete upload is done.
	var ctx context.Context
	ctx, uh.cancelFunc = context.WithCancel(context.Background())
	uh.writer, err = uh.bucket.CreateObjectChunkWriter(ctx, req, int(uh.blockSize), nil)
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
		if uh.UploadError() != nil {
			uh.wg.Done()
			continue
		}
		_, err := io.Copy(uh.writer, currBlock.Reader())
		if errors.Is(err, context.Canceled) {
			// Context canceled error indicates that the file was deleted from the
			// same mount. In this case, we suppress the error to match local
			// filesystem behavior.
			err = nil
		}
		if err != nil {
			logger.Errorf("buffered write upload failed for object %s: error in io.Copy: %v", uh.objectName, err)
			err = gcs.GetGCSError(err)
			uh.uploadError.Store(&err)
		}
		// Put back the uploaded block on the freeBlocksChannel for re-use.
		uh.freeBlocksCh <- currBlock
		uh.wg.Done()
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
		err := uh.createObjectWriter()
		if err != nil {
			return fmt.Errorf("createObjectWriter failed for object %s: %w", uh.objectName, err)
		}
	}
	return nil
}

// FlushPendingWrites uploads any data in the write buffer.
func (uh *UploadHandler) FlushPendingWrites() (int64, error) {
	uh.wg.Wait()

	// Writer may not have been created for empty file creation flow or for very
	// small writes of size less than 1 block.
	err := uh.ensureWriter()
	if err != nil {
		return 0, fmt.Errorf("uh.ensureWriter() failed: %v", err)
	}

	offset, err := uh.bucket.FlushPendingWrites(context.Background(), uh.writer)
	if err != nil {
		// FlushUpload already returns GCS error so no need to convert again.
		uh.uploadError.Store(&err)
		logger.Errorf("FlushUpload failed for object %s: %v", uh.objectName, err)
		return 0, err
	}
	return offset, nil
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
			uh.freeBlocksCh <- currBlock
			// Marking as wg.Done to ensure any waiters are unblocked.
			uh.wg.Done()
		default:
			// This will get executed when there are no blocks pending in uploadCh and its not closed.
			close(uh.uploadCh)
			return
		}
	}
}
