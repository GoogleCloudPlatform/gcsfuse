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
	ctx        context.Context
}

// newUploadHandler creates the UploadHandler struct.
func newUploadHandler(objectName string, bucket gcs.Bucket, freeBlocksCh *chan block.Block, blockSize int64) *UploadHandler {
	uh := &UploadHandler{
		uploadCh:     make(chan block.Block),
		wg:           sync.WaitGroup{},
		freeBlocksCh: *freeBlocksCh,
		bucket:       bucket,
		objectName:   objectName,
		blockSize:    blockSize,
	}
	return uh
}

// Upload adds a block to the upload queue.
func (uh *UploadHandler) Upload(ctx context.Context, block block.Block) (err error) {
	uh.wg.Add(1)

	if uh.writer == nil {
		// Lazily create the object writer.
		err = uh.createObjectWriter(ctx)
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
func (uh *UploadHandler) createObjectWriter(ctx context.Context) (err error) {
	var preCond int64
	req := &gcs.CreateObjectRequest{
		Name:                   uh.objectName,
		GenerationPrecondition: &preCond,
		Metadata:               make(map[string]string),
	}
	// We need to create a non-cancellable context, since the first writeFile()
	// call will be done by the time total upload is done.
	uh.ctx = context.WithoutCancel(ctx)
	uh.writer, err = uh.bucket.CreateObjectChunkWriter(uh.ctx, req, int(uh.blockSize), uh.statusNotifier)
	return err
}

// statusNotifier is a callback function called after every complete block is uploaded.
func (uh *UploadHandler) statusNotifier(bytesUploaded int64) {
	logger.Infof("gcs: Req %#16x: -- CreateObject(%q): %20v bytes uploaded so far", uh.ctx.Value(gcs.ReqIdField), uh.objectName, bytesUploaded)
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
		currBlock.Reuse()
		uh.freeBlocksCh <- currBlock
	}
}

// Finalize finalizes the upload.
func (uh *UploadHandler) Finalize() error {
	uh.wg.Wait()
	close(uh.uploadCh)

	if uh.writer == nil {
		return fmt.Errorf("unexpected nil writer")
	}

	err := uh.writer.Close()
	if err != nil {
		logger.Errorf("UploadHandler.Finalize(): %v", err)
		return fmt.Errorf("writer.CLose: %w", err)
	}
	return nil
}
