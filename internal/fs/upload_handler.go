package fs

/*
import (
	"container/list"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"golang.org/x/net/context"
)

type uploadHandler struct {
	// Holds the list of chunks to be uploaded to GCS
	chunks list.List
	// Current chunk being uploaded.
	bufferInProgress *Block
	bucket           gcs.Bucket
	writer           *storage.Writer
	// Provides the status of upload
	status    uploadStatus
	mu        locker.Locker
	chunkSize int64
	// Channel on which uploaded blocks will be posted for reuse.
	blocksCh   chan<- *Block
	ctx        context.Context
	uploadDone chan error
}

type uploadStatus string

const (
	NotStarted uploadStatus = "NotStarted"
	Uploading  uploadStatus = "Uploading"
	// This specifies the chunk is uploaded and waiting for further chunks.
	ChunkUploaded   uploadStatus = "ChunkUploaded"
	ReadyToFinalize uploadStatus = "ReadyToFinalize"
	Finalized       uploadStatus = "Finalized"
	Failed          uploadStatus = "Failed"
)

// Method to initiate uploadHandler.Pass all uploadHandler struct parameters as args.
func InitUploadHandler() *uploadHandler {
	uh := uploadHandler{}

	return &uh
}

// TODO: How to handle partial upload success, where we encountered an error and finalized the upload.
func (uh *uploadHandler) Upload(block Block) (err error) {
	uh.chunks.PushBack(block)
	uh.mu.Lock()
	defer uh.mu.Unlock()

	switch uh.status {
	case NotStarted:
		err = uh.startUpload()
		return
	case Uploading:
		// The progress function will take care of uploading the chunks from the list.
		return
	case ChunkUploaded:
		uh.uploadChunk()
		return
	case Finalized:
		// Already finalized, return error
		return
	case Failed:
		// return error
		return

	}
}

func (uh *uploadHandler) startUpload() (err error) {
	uh.writer, err = uh.bucket.CreateObjectInChunks(
		uh.ctx,
		nil, // pass request
		int(uh.chunkSize),
		uh.statusNotifier)

	uh.bufferInProgress = uh.chunks.Front().Value.(*Block)
	err = uh.bucket.UploadChunk(uh.writer, *uh.bufferInProgress)
	if err != nil {
		uh.status = Uploading
	}
	return
}

func (uh *uploadHandler) statusNotifier(bytesUploaded int64) {
	// Put back the block on the channel for reuse.
	uh.blocksCh <- uh.bufferInProgress
	uh.mu.Lock()
	defer uh.mu.Unlock()

	// Upload next chunk if available.
	if uh.chunks.Len() > 0 {
		uh.uploadChunk()
	}

	// Finalize the upload if there are no pending chunks and finalize is called.
	if uh.status == ReadyToFinalize {
		uh.finalize()
		return
	}
	// If there are no chunks to upload, update the status as chunkUploaded and
	// wait for next chunk or finalize call
	uh.status = ChunkUploaded
	return
}

func (uh *uploadHandler) uploadChunk() {
	uh.bufferInProgress = uh.chunks.Front().Value.(*Block)
	err := uh.bucket.UploadChunk(uh.writer, *uh.bufferInProgress)
	if err != nil {
		err = uh.finalize()
		if err != nil {
			uh.status = Failed
		}
	}
}

func (uh *uploadHandler) finalize() error {
	// Notstarted, finalized
	// Throw error

	if uh.chunks.Len() != 0 || uh.status == Uploading {
		uh.status = ReadyToFinalize
		uh.uploadDone = make(chan error)
		err := <-uh.uploadDone
		return err
	}

	// readtoFinalize, ChunkUploaded, failed
	err := uh.writer.Close()
	if uh.uploadDone != nil {
		uh.uploadDone <- err
	}

	return err
}
*/
