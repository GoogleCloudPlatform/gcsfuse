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

	// Parameters required for creating a new GCS chunk writer.
	bucket     gcs.Bucket
	objectName string
	blockSize  int64
}

// newUploadHandler creates the UploadHandler struct.
func newUploadHandler(objectName string, bucket gcs.Bucket, freeBlocksCh chan block.Block, blockSize int64) *UploadHandler {
	uh := &UploadHandler{
		uploadCh:     make(chan block.Block),
		wg:           sync.WaitGroup{},
		freeBlocksCh: freeBlocksCh,
		bucket:       bucket,
		objectName:   objectName,
		blockSize:    blockSize,
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
			return fmt.Errorf("createObjectWriter: %w", err)
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
		_, err := io.Copy(uh.writer, currBlock.Reader())
		if err != nil {
			logger.Errorf("upload failed: error in io.Copy: %v", err)
			uh.wg.Done()
			// TODO: handle failure scenario: finalize the upload and trigger edit flow.
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
		// Writer may not have been created for empty file creation flow.
		err := uh.createObjectWriter()
		if err != nil {
			return fmt.Errorf("createObjectWriter: %w", err)
		}
	}

	err := uh.writer.Close()
	if err != nil {
		logger.Errorf("UploadHandler.Finalize(): %v", err)
		return fmt.Errorf("writer.Close: %w", err)
	}
	return nil
}