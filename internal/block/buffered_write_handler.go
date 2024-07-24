package block

import (
	"fmt"
	"math"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

// All the write operations take inode lock in fs.go, hence we dont need any locks here
// as we will get write operations serially.
type BufferedWriteHandler struct {
	current       Block
	blockPool     BlockPool
	UploadHandler UploadHandler
	totalSize     int64
	mtime         time.Time
	objName       string
	mu            locker.Locker
}

type WriteFileInfo struct {
	totalSize int64
	mtime     time.Time
}

const BlockSize = 1 * util.MiB
const MaxBlocks = 10

// InitBufferedWriteHandler - Pass all required param and do init.
func InitBufferedWriteHandler(objectName string, bucket gcs.Bucket) *BufferedWriteHandler {
	blockPool := *InitBlockPool(BlockSize, MaxBlocks)
	// pass the blockPool created here to UploadHandler
	return &BufferedWriteHandler{
		current:       nil,
		blockPool:     blockPool,
		UploadHandler: *InitUploadHandler(objectName, bucket, blockPool.blocksCh),
		totalSize:     0,
		mtime:         time.Now(),
		mu:            locker.NewRW("BufferedWriteHandler", func() {}),
	}
}

func (wh *BufferedWriteHandler) Write(data []byte, offset int64) (err error) {
	wh.mu.Lock()
	defer wh.mu.Unlock()
	dataWritten := int(0)

	if offset != wh.totalSize {
		// TODO: We encountered a non-sequential write, finalize and throw a custom error.
		return fmt.Errorf("non sequential writes")
	}

	for dataWritten < len(data) {
		if wh.current == nil {
			wh.current, err = wh.blockPool.Get()
			if err != nil {
				return
			}
		}

		bytesToCopy := int(math.Min(
			float64(wh.blockPool.blockSize-int64(wh.current.Size())),
			float64(len(data)-dataWritten)))
		err := wh.current.Write(data[dataWritten : dataWritten+bytesToCopy])
		if err != nil {
			return err
		}
		dataWritten += bytesToCopy

		if int64(wh.current.Size()) == wh.blockPool.blockSize {
			// trigger upload
			err := wh.UploadHandler.Upload(wh.current)
			if err != nil {
				return err
			}
			wh.current = nil
		}
	}

	wh.totalSize += int64(dataWritten)
	return
}

func (wh *BufferedWriteHandler) Finalize() (err error) {
	if wh.current != nil {
		err := wh.UploadHandler.Upload(wh.current)
		if err != nil {
			return err
		}
		wh.current = nil
	}
	return wh.UploadHandler.Finalize()
}

func (wh *BufferedWriteHandler) SetMtime(mtime time.Time) {
	wh.mtime = mtime
}

func (wh *BufferedWriteHandler) GetWriteFileInfo() WriteFileInfo {
	return WriteFileInfo{
		totalSize: wh.totalSize,
		mtime:     wh.mtime,
	}
}
