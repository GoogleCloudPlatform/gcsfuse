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

	"github.com/googlecloudplatform/gcsfuse/v2/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
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
	writer io.WriteCloser

	// signalUploadFailure channel will propagate the upload error to file
	// inode so that it can trigger temp file flow and flush any current buffer.
	signalUploadFailure chan error

	// signalNonRecoverableFailure channel will propagate non recoverable failure
	// to file inode.
	signalNonRecoverableFailure chan error

	// tempFile channel will receive the temporary file writer where all the
	// non uploaded blocks will be written in case of failure.
	tempFile chan gcsx.TempFile

	// Parameters required for creating a new GCS chunk writer.
	bucket     gcs.Bucket
	objectName string
	blockSize  int64
}

// newUploadHandler creates the UploadHandler struct.
func newUploadHandler(objectName string, bucket gcs.Bucket, maxBlocks int64, freeBlocksCh chan block.Block, blockSize int64) *UploadHandler {
	uh := &UploadHandler{
		uploadCh:                    make(chan block.Block, maxBlocks),
		wg:                          sync.WaitGroup{},
		freeBlocksCh:                freeBlocksCh,
		bucket:                      bucket,
		objectName:                  objectName,
		blockSize:                   blockSize,
		signalUploadFailure:         make(chan error, 1),
		signalNonRecoverableFailure: make(chan error, 1),
		tempFile:                    make(chan gcsx.TempFile, 1),
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
			return fmt.Errorf("createObjectWriter failed: %w", err)
		}
		// Start the uploader goroutine.
		go uh.uploader()
	}

	uh.uploadCh <- block
	return nil
}

// createObjectWriter creates a GCS object writer.
func (uh *UploadHandler) createObjectWriter() (err error) {
	var preCond int64
	req := &gcs.CreateObjectRequest{
		Name:                   uh.objectName,
		GenerationPrecondition: &preCond,
		Metadata:               make(map[string]string),
	}
	// We need a new context here, since the first writeFile() call will be complete
	// (and context will be cancelled) by the time complete upload is done.
	uh.writer, err = uh.bucket.CreateObjectChunkWriter(context.Background(), req, int(uh.blockSize), nil)
	return
}

// uploader is the single-threaded goroutine that uploads blocks.
func (uh *UploadHandler) uploader() {
	for currBlock := range uh.uploadCh {
		currBlockReader := currBlock.Reader()
		_, err := io.Copy(uh.writer, currBlockReader)
		if err != nil {
			logger.Errorf("buffered write upload failed: error in io.Copy: %v", err)

			// Close the writer to finalize the object creation on GCS.
			if closeErr := uh.writer.Close(); closeErr != nil {
				logger.Errorf("Error closing writer: %v", closeErr)
				close(uh.signalNonRecoverableFailure)
				return
			}

			// Signal the upload failure to the caller so that it falls back to edit
			// flow and flushes the current buffer.
			logger.Warnf("Error while uploading: %v", err)
			uh.signalUploadFailure <- fmt.Errorf("error while uploading: %w", err)

			// Trigger the failure handler and get the temp file path.
			handlerErr := uh.handleUploadFailure(currBlockReader)
			if handlerErr != nil {
				logger.Errorf("Error while handling upload failure: %v", handlerErr)
				close(uh.signalNonRecoverableFailure)
				return
			}
			return
		}
		uh.wg.Done()

		// Put back the uploaded block on the freeBlocksChannel for re-use.
		uh.freeBlocksCh <- currBlock
	}
}

// Finalize finalizes the upload.
func (uh *UploadHandler) Finalize() error {
	uh.wg.Wait()
	close(uh.uploadCh)

	if uh.writer == nil {
		// Writer may not have been created for empty file creation flow or for very
		// small writes of size less than 1 block.
		err := uh.createObjectWriter()
		if err != nil {
			return fmt.Errorf("createObjectWriter: %w", err)
		}
	}

	err := uh.writer.Close()
	if err != nil {
		logger.Errorf("UploadHandler.Finalize(): %v", err)
		close(uh.signalNonRecoverableFailure)
		return fmt.Errorf("writer.Close: %w", err)
	}
	return nil
}

// handleUploadFailure handles the upload failure by writing
// non-uploaded blocks to the temporary file writer provided by the caller.
// This function assumes that the temporary file has already been created
// and populated with existing GCS content by the caller.
func (uh *UploadHandler) handleUploadFailure(failedBlockReader io.Reader) error {
	tmpFile := <-uh.tempFile
	if tmpFile == nil {
		return fmt.Errorf("no temp file provided")
	}

	// get size of temp file created.
	statResult, err := tmpFile.Stat()
	if err != nil {
		return fmt.Errorf("stat failed on temp file")
	}
	size := statResult.Size

	// Write the failed block to the temporary file.
	n, err := writeBlockToTempFile(size, failedBlockReader, tmpFile)
	if err != nil {
		return fmt.Errorf("failed to write failed block to temp file: %w", err)
	}
	size += int64(n)
	uh.wg.Done()

	// Drain the upload channel and write remaining blocks to the temporary file.
	// Do not put back channel for re-use. Any following write calls will be
	// redirected to temp file.
	for currBlock := range uh.uploadCh {
		n, err := writeBlockToTempFile(size, currBlock.Reader(), tmpFile)
		if err != nil {
			return fmt.Errorf("failed to write block to temp file: %w", err)
		}
		size += int64(n)
		uh.wg.Done()
	}

	return nil
}

func writeBlockToTempFile(offset int64, reader io.Reader, tmpFile gcsx.TempFile) (int, error) {
	remainingCurrBlockContent, err := io.ReadAll(reader)
	if err != nil {
		return 0, fmt.Errorf("failed to get data from block reader")
	}
	n, err := tmpFile.WriteAt(remainingCurrBlockContent, offset)
	if err != nil {
		return 0, fmt.Errorf("error writing block to temp file: %w", err)
	}
	return n, nil
}
