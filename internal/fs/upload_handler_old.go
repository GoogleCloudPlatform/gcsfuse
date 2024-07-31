package fs

//
//import (
//	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
//	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
//	"golang.org/x/net/context"
//)
//import "container/list"
//
//type uploadHandler struct {
//	chunks           list.List
//	bufferInProgress *Block
//	chunkUploader    gcs.ChunkUploader
//	bucket           gcs.Bucket
//	status           uploadStatus
//	mu               locker.Locker
//	chunkSize        int64
//	blocksCh         chan<- *Block
//}
//
//type uploadStatus string
//
//const (
//	NotStarted      uploadStatus = "NotStarted"
//	Uploading       uploadStatus = "Uploading"
//	ChunkUploaded   uploadStatus = "ChunkUploaded"
//	ReadyToFinalize uploadStatus = "ReadyToFinalize"
//	Finalized       uploadStatus = "Finalized"
//	Failed          uploadStatus = "Failed"
//)
//
//func Init(blocksChan chan<- *Block) *uploadHandler {
//	uh := uploadHandler{
//		blocksCh: blocksChan,
//	}
//
//	return &uh
//}
//
//// TODO: How to handle partial upload success, where we encountered an error and finalized the upload.
//func (uh *uploadHandler) upload(block Block) {
//	uh.chunks.PushBack(block)
//	uh.mu.Lock()
//	defer uh.mu.Unlock()
//
//	switch uh.status {
//	case NotStarted:
//		uh.startUpload()
//	case Uploading:
//		return
//	case ChunkUploaded:
//		return
//	case Finalized:
//		// Already finalized, return error
//		return
//	case Failed:
//		// return error
//		return
//
//	}
//}
//
//func (uh *uploadHandler) startUpload() (err error) {
//	uh.chunkUploader, err = uh.bucket.CreateChunkUploader(context.Background(), nil, int(uh.chunkSize), uh.statusNotifier)
//	uh.bufferInProgress = uh.chunks.Front().Value.(*Block)
//	err = uh.chunkUploader.Upload(context.Background(), *uh.bufferInProgress)
//	return
//}
//
//func (uh *uploadHandler) statusNotifier(bytesUploaded int64) {
//	uh.blocksCh <- uh.bufferInProgress
//	uh.mu.Lock()
//	defer uh.mu.Unlock()
//	if uh.chunks.Len() == 0 {
//		if uh.status == ReadyToFinalize {
//			uh.finalize()
//			return
//		}
//
//		uh.status = ChunkUploaded
//		return
//	}
//
//	uh.bufferInProgress = uh.chunks.Front().Value.(*Block)
//	err := uh.chunkUploader.Upload(context.Background(), *uh.bufferInProgress)
//	if err != nil {
//		if uh.status == ReadyToFinalize {
//			uh.finalize()
//			return
//		}
//		uh.status = Failed
//	}
//}
//
//func (uh *uploadHandler) finalize() {
//	if uh.chunks.Len() != 0 || uh.status == Uploading {
//		uh.status = ReadyToFinalize
//		return
//	}
//
//	uh.chunkUploader.Close(context.Background())
//}
