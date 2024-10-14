package block

import (
	"container/list"
	"context"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

type UploadHandler struct {
	// Holds the list of chunks to be uploaded to GCS
	chunks list.List
	// Current chunk being uploaded.
	bufferInProgress Block
	bucket           gcs.Bucket
	writer           *storage.Writer
	// Provides the status of upload
	status    uploadStatus
	mu        locker.Locker
	chunkSize int64
	// Channel on which uploaded blocks will be posted for reuse.
	blocksCh   chan Block
	uploadDone chan error
	objectName string
}

type uploadStatus string

const (
	NotStarted uploadStatus = "NotStarted"
	Uploading  uploadStatus = "Uploading"
	// ChunkUploaded specifies the chunk is uploaded and waiting for further chunks.
	ChunkUploaded   uploadStatus = "ChunkUploaded"
	ReadyToFinalize uploadStatus = "ReadyToFinalize"
	Finalized       uploadStatus = "Finalized"
	Failed          uploadStatus = "Failed"
)

// InitUploadHandler to initiate UploadHandler.Pass all UploadHandler struct parameters as args.
func InitUploadHandler(objectName string, bucket gcs.Bucket, blockChan *chan Block) *UploadHandler {
	uh := UploadHandler{
		chunks:           list.List{},
		bufferInProgress: nil,
		bucket:           bucket,
		writer:           nil,
		status:           NotStarted,
		mu:               locker.NewRW("UploadHandler", func() {}),
		chunkSize:        BlockSize,
		blocksCh:         *blockChan,
		objectName:       objectName,
	}

	return &uh
}

// TODO: How to handle partial upload success, where we encountered an error and finalized the upload.
func (uh *UploadHandler) Upload(block Block) (err error) {
	uh.mu.Lock()
	uh.chunks.PushBack(block)
	uh.mu.Unlock()

	switch uh.status {
	case NotStarted:
		err = uh.startUpload()
		return
	case Uploading:
		return
	case ChunkUploaded:
		err = uh.uploadChunk()
		if err != nil {
			return err
		}
		return
	case ReadyToFinalize:
	case Finalized:
		// Already finalizing or finalized, return error
		return
	case Failed:
		// return error
		return

	}
	return
}

func (uh *UploadHandler) startUpload() (err error) {
	var req *gcs.CreateObjectRequest
	var preCond int64
	metadataMap := make(map[string]string)
	req = &gcs.CreateObjectRequest{
		Name:                   uh.objectName,
		GenerationPrecondition: &preCond,
		Metadata:               metadataMap,
	}

	// We need to create a new context, since the first writeFile() call will be
	// done by the time total upload is done. So can't use that context.
	// TODO: Check if ctx needs to be persisted in UploadHandler.
	uh.writer, err = uh.bucket.CreateObjectInChunks(context.Background(),
		req,
		int(uh.chunkSize),
		uh.statusNotifier)
	if err != nil {
		return
	}

	// start upload
	uh.status = Uploading
	err = uh.uploadChunk()
	if err != nil {
		return err
	}

	//listEle := uh.chunks.Front()
	//uh.bufferInProgress = listEle.Value.(Block)
	//uh.chunks.Remove(listEle)
	//go func() {
	//	uh.status = Uploading
	//	err = uh.bucket.Upload(uh.writer, uh.bufferInProgress)
	//	uh.uploadDone <- err
	//}()
	return
}

func (uh *UploadHandler) statusNotifier(bytesUploaded int64) {
	// Put back the block on the channel for reuse.
	uh.bufferInProgress.Reuse(uh.blocksCh)
	//uh.blocksCh <- uh.bufferInProgress
	uh.mu.Lock()
	defer uh.mu.Unlock()

	// Upload next chunk if available.
	if uh.chunks.Len() > 0 {
		if uh.chunks.Len() == 1 && uh.status == ReadyToFinalize {
			_ = uh.uploadFinalChunk()
			return
		}
		err := uh.uploadChunk()
		if err != nil {
			return
		}
		return
	}

	// Finalize the upload if there are no pending chunks and finalize is called.
	if uh.status == ReadyToFinalize {
		// Call in go routine otherwise writer won't close.
		go func() {
			_ = uh.Finalize()
		}()
		return
	}
	// If there are no chunks to upload, update the status as chunkUploaded and
	// wait for next chunk or finalize call
	uh.status = ChunkUploaded
	return
}

func (uh *UploadHandler) uploadChunk() error {
	listEle := uh.chunks.Front()
	if listEle == nil {
		logger.Errorf("Got empty list in upload chunk")
	}
	uh.bufferInProgress = listEle.Value.(Block)
	uh.chunks.Remove(listEle)

	go func() {
		err := uh.bucket.Upload(uh.writer, uh.bufferInProgress)

		if err != nil {
			logger.Warnf("Upload failed: %v", err)
			uh.status = Failed
		}
	}()
	return nil
}

func (uh *UploadHandler) uploadFinalChunk() (err error) {
	listEle := uh.chunks.Front()
	if listEle == nil {
		logger.Errorf("Got empty list in upload chunk")
	}
	uh.bufferInProgress = listEle.Value.(Block)
	uh.chunks.Remove(listEle)

	go func() {
		err = uh.bucket.Upload(uh.writer, uh.bufferInProgress)
		if err != nil {
			logger.Warnf("Upload failed: %v", err)
			uh.status = Failed
			return
		}
		err = uh.Finalize()
		if err != nil {
			return
		}
	}()
	return nil
}

func (uh *UploadHandler) Finalize() error {
	// Notstarted
	// Empty file, createobject on GCS

	//  finalized
	// Throw error

	if uh.chunks.Len() != 0 || uh.status == Uploading {
		//Changing status to ready to finalise so that on callback function call upload is finalised
		uh.status = ReadyToFinalize
		uh.uploadDone = make(chan error)
		err := <-uh.uploadDone
		return err
	}

	// readToFinalize, ChunkUploaded, failed
	err := uh.writer.Close()
	if err != nil {
		logger.Errorf("error in closing Writer: %v", err)
	}
	if uh.uploadDone != nil {
		uh.uploadDone <- err
	}
	return err
}
