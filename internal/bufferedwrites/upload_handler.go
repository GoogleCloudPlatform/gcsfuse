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

package bufferedwrites

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

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

	// signalUploadFailure channel will propagate the upload error to file
	// inode. This signals permanent failure in the buffered write job.
	signalUploadFailure chan error

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
		signalUploadFailure:  make(chan error, 1),
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
	req := gcs.NewCreateObjectRequest(uh.obj, uh.objectName, nil, uh.chunkTransferTimeout)
	// We need a new context here, since the first writeFile() call will be complete
	// (and context will be cancelled) by the time complete upload is done.
	var ctx context.Context
	ctx, uh.cancelFunc = context.WithCancel(context.Background())
	uh.writer, err = uh.bucket.CreateObjectChunkWriter(ctx, req, int(uh.blockSize), nil)
	return
}

// uploader is the single-threaded goroutine that uploads blocks.
func (uh *UploadHandler) uploader() {
	for currBlock := range uh.uploadCh {
		select {
		case <-uh.signalUploadFailure:
		default:
			_, err := io.Copy(uh.writer, currBlock.Reader())
			if err != nil {
				logger.Errorf("buffered write upload failed for object %s: error in io.Copy: %v", uh.objectName, err)
				// Close the channel to signal upload failure.
				close(uh.signalUploadFailure)
			}
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

	if uh.writer == nil {
		// Writer may not have been created for empty file creation flow or for very
		// small writes of size less than 1 block.
		err := uh.createObjectWriter()
		if err != nil {
			return nil, fmt.Errorf("createObjectWriter failed for object %s: %w", uh.objectName, err)
		}
	}

	obj, err := uh.bucket.FinalizeUpload(context.Background(), uh.writer)
	if err != nil {
		return nil, fmt.Errorf("FinalizeUpload failed for object %s: %w", uh.objectName, err)
	}
	return obj, nil
}

func (uh *UploadHandler) CancelUpload() {
	if uh.cancelFunc != nil {
		// cancel the context to cancel the ongoing GCS upload.
		uh.cancelFunc()
	}
	// Wait for all in progress buffers to be added to the free channel.
	uh.wg.Wait()
}

func (uh *UploadHandler) SignalUploadFailure() chan error {
	return uh.signalUploadFailure
}

func (uh *UploadHandler) AwaitBlocksUpload() {
	uh.wg.Wait()
}

func (uh *UploadHandler) Destroy() {
	// Waiting for upload routine to move all blocks to freeChannel.
	timedOut := waitTimeout(&uh.wg, 10*time.Second)

	// TimedOut means there are some blocks which are still in uploadChannel,
	// either because upload is stuck in uploading a chunk or the upload
	// go-routine crashed. Copying all pending blocks to freeBlock channel for cleanup.
	// We can clean up from uploadChannel also, but to ensure clean up happens
	// from one place we are copying them to freeBlock channel.
	if timedOut {
		for currBlock := range uh.uploadCh {
			uh.freeBlocksCh <- currBlock
			// Marking as wg.Done to ensure any waiters are unblocked.
			uh.wg.Done()
		}
	}

	// This code is safer when not executed from multiple go-routines at once.
	// Since destroy takes a inode lock in fileHandle, this is called by
	// only one goroutine at once.
	select {
	case <-uh.uploadCh:
	default:
		close(uh.uploadCh)
	}
}

// TODO: Move this method to util and add unit tests.
// waitTimeout waits for the waitGroup for the specified max timeout.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()

	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
