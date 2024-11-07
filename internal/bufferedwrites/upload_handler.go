package bufferedwrites

import (
	"container/list"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

type uploadStatus string

const (
	NotStarted        uploadStatus = "NotStarted"
	Uploading         uploadStatus = "Uploading"
	ReadyForNextBlock uploadStatus = "ReadyForNextBlock"
	ReadyToFinalize   uploadStatus = "ReadyToFinalize"
	Finalized         uploadStatus = "Finalized"
	Failed            uploadStatus = "Failed"
)

// UploadHandler is responsible for synchronized uploads of the filled blocks
// to GCS and then putting them back for reuse once the block has been uploaded.
type UploadHandler struct {
	// Holds the list of blocks to be uploaded to GCS.
	blocks list.List

	// Mutex for thread-safe blocks list access.
	muBlocks sync.RWMutex

	// Mutex for sequential uploads of blocks to GCS.
	mu sync.RWMutex

	// Current block being uploaded.
	bufferInProgress block.Block

	// Current status of the buffered upload.
	status uploadStatus

	// Channel on which uploaded block will be posted for reuse.
	freeBlocksCh chan block.Block

	// writer to resumable upload the blocks to GCS.
	writer io.WriteCloser

	// Channel to wait and notify for write finalize completion.
	finalizeDone chan error

	// Parameters required for creating a new GCS chunk writer.
	bucket     gcs.Bucket
	objectName string
	blockSize  int64
	ctx        context.Context
}

// newUploadHandler creates the UploadHandler struct.
func newUploadHandler(objectName string, bucket gcs.Bucket, freeBlocksCh *chan block.Block, blockSize int64) *UploadHandler {
	return &UploadHandler{
		blocks:       list.List{},
		mu:           sync.RWMutex{},
		status:       NotStarted,
		freeBlocksCh: *freeBlocksCh,
		bucket:       bucket,
		objectName:   objectName,
		blockSize:    blockSize,
	}
}

// TODO: How to handle partial upload success, where we encountered an error and finalized the upload.
func (uh *UploadHandler) Upload(ctx context.Context, block block.Block) (err error) {
	uh.muBlocks.Lock()
	uh.blocks.PushBack(block)
	uh.muBlocks.Unlock()

	switch uh.status {
	case NotStarted:
		err = uh.createObjectWriter(ctx)
		fallthrough
	case ReadyForNextBlock:
		uh.status = Uploading
		err = uh.uploadBlock()
		if err != nil {
			return fmt.Errorf("uploadBlock(): %w", err)
		}
	case Uploading:
		// Block will be auto picked from blocks queue when all previous blocks are
		// uploaded.
	case ReadyToFinalize:
	case Finalized:
		return fmt.Errorf("upload already Finalized, can't upload more data")
	case Failed:
		return fmt.Errorf("upload status: Failed")
	}

	return nil
}

// createObjectWriter creates a gcs object writer and changes the status to
// ReadyForNextBlock.
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
	uh.status = ReadyForNextBlock
	return err
}

// Thread safe length of blocks list.
func (uh *UploadHandler) blocksLength() int {
	uh.muBlocks.RLock()
	defer uh.muBlocks.RUnlock()

	return uh.blocks.Len()
}

// Callback function called after every complete block is uploaded.
func (uh *UploadHandler) statusNotifier(bytesUploaded int64) {
	// No lock is required since we are taking a lock during upload. Next upload
	// can only be triggered after callback.
	logger.Infof("gcs: Req %#16x: -- CreateObject(%q): %20v bytes uploaded so far", uh.ctx.Value(gcs.ReqIdField), uh.objectName, bytesUploaded)

	// Put back the uploaded block on the freeBlocksChannel for re-use.
	uh.bufferInProgress.Reuse()
	uh.freeBlocksCh <- uh.bufferInProgress

	// Upload next block if available.
	if uh.blocksLength() > 0 {
		_ = uh.uploadBlock()
		return
	}

	// Finalize the upload if there are no blocks to be uploaded and finalize is
	// called.
	if uh.status == ReadyToFinalize {
		// Writer can't be closed until the callback function returns so calling
		// Finalize in a goroutine.
		go func() {
			_ = uh.Finalize()
		}()
		return
	}

	// If there are no more blocks to upload, update the status as
	// ReadyForNextBlock and wait for next block or finalize call.
	uh.status = ReadyForNextBlock
	return
}

// LOCKS_EXCLUDED(mu)
func (uh *UploadHandler) uploadBlock() error {
	if uh.blocksLength() == 0 {
		return fmt.Errorf("empty blocks list in uploadBlock")
	}

	uh.mu.Lock()
	uh.muBlocks.Lock()
	listEle := uh.blocks.Front()
	uh.blocks.Remove(listEle)
	uh.muBlocks.Unlock()
	uh.bufferInProgress = listEle.Value.(block.Block)

	go func() {
		_, err := io.Copy(uh.writer, uh.bufferInProgress.Reader())
		if err != nil {
			uh.status = Failed
			logger.Errorf("upload failed: error in io.Copy: %v", err)
			uh.mu.Unlock()
			return
		}
		uh.mu.Unlock()

		// If there are no more blocks to upload, close the writer since we might
		// not receive a callback for the final chunk, and there's no other way to
		// signal completion of io.Copy.
		if uh.blocksLength() == 0 && uh.status == ReadyToFinalize {
			_ = uh.Finalize()
		}
	}()

	return nil
}

func (uh *UploadHandler) Finalize() error {
	// If there are blocks still uploading, wait for upload completion.
	if uh.blocksLength() != 0 || uh.status == Uploading {
		//Changing status to ReadyToFinalize so that callback function finalizes the
		// upload once uploads are completed.
		uh.status = ReadyToFinalize
		uh.finalizeDone = make(chan error)
		err := <-uh.finalizeDone
		return err
	}

	var err error
	if uh.writer == nil {
		err = fmt.Errorf("unexpected nil writer")
		logger.Errorf("UploadHandler.Finalize(): %v", err)
	} else {
		err = uh.writer.Close()
		if err != nil {
			logger.Errorf("UploadHandler.Finalize(): %v", err)
		}
		uh.status = Finalized
	}

	if uh.finalizeDone != nil {
		uh.finalizeDone <- err
	}
	return err
}
