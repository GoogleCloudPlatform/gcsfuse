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
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"golang.org/x/sync/semaphore"
)

// Note: All the write operations take inode lock in fs.go, hence we don't need any locks here
// as we will get write operations serially.

// BufferedWriteHandler is responsible for filling up the buffers with the data
// as it receives and handing over to uploadHandler which uploads to GCS.
type BufferedWriteHandler struct {
	current       block.Block
	blockPool     *block.BlockPool
	uploadHandler *UploadHandler
	// Total size of data buffered so far. Some part of buffered data might have
	// been uploaded to GCS as well. Depending on the state we are in, it might or
	// might not include truncatedSize.
	totalSize int64
	// Stores the mtime value updated by kernel as part of setInodeAttributes call.
	mtime time.Time
	// Stores the size to truncate. No action is made when truncate is called.
	// Will be used as mentioned below:
	// 1. During flush if totalSize != truncatedSize, additional dummy data is
	// added before flush and uploaded.
	// 2. If write is started after the truncate offset, dummy data is created
	// as per the truncatedSize and then new data is appended to it.
	truncatedSize int64
}

// WriteFileInfo is used as part of serving fileInode attributes (GetInodeAttributes call).
type WriteFileInfo struct {
	TotalSize int64
	Mtime     time.Time
}

var ErrOutOfOrderWrite = errors.New("outOfOrder write detected")
var ErrUploadFailure = errors.New("error while uploading object to GCS")

// NewBWHandler creates the bufferedWriteHandler struct.
func NewBWHandler(objectName string, bucket gcs.Bucket, blockSize int64, maxBlocks int64, globalMaxBlocksSem *semaphore.Weighted) (bwh *BufferedWriteHandler, err error) {
	bp, err := block.NewBlockPool(blockSize, maxBlocks, globalMaxBlocksSem)
	if err != nil {
		return
	}

	bwh = &BufferedWriteHandler{
		current:       nil,
		blockPool:     bp,
		uploadHandler: newUploadHandler(objectName, bucket, maxBlocks, bp.FreeBlocksChannel(), blockSize),
		totalSize:     0,
		mtime:         time.Now(),
	}
	return
}

// Write writes the given data to the buffer. It writes to an existing buffer if
// the capacity is available otherwise writes to a new buffer.
func (wh *BufferedWriteHandler) Write(data []byte, offset int64) (err error) {
	if offset != wh.totalSize && offset != wh.truncatedSize {
		logger.Errorf("BufferedWriteHandler.OutOfOrderError for object: %s, expectedOffset: %d, actualOffset: %d",
			wh.uploadHandler.objectName, wh.totalSize, offset)
		return ErrOutOfOrderWrite
	}

	err = wh.writeDataForTruncatedSize()
	if err != nil {
		return
	}

	// Fail early if the uploadHandler has failed.
	select {
	case <-wh.uploadHandler.SignalUploadFailure():
		return ErrUploadFailure
	default:
		break
	}

	dataWritten := 0
	for dataWritten < len(data) {
		if wh.current == nil {
			wh.current, err = wh.blockPool.Get()
			if err != nil {
				return fmt.Errorf("failed to get new block: %w", err)
			}
		}

		remainingBlockSize := float64(wh.blockPool.BlockSize()) - float64(wh.current.Size())
		pendingDataForWrite := float64(len(data)) - float64(dataWritten)
		bytesToCopy := int(math.Min(remainingBlockSize, pendingDataForWrite))
		err := wh.current.Write(data[dataWritten : dataWritten+bytesToCopy])
		if err != nil {
			return err
		}

		dataWritten += bytesToCopy

		if wh.current.Size() == wh.blockPool.BlockSize() {
			err := wh.uploadHandler.Upload(wh.current)
			if err != nil {
				return err
			}
			wh.current = nil
		}
	}

	wh.totalSize += int64(dataWritten)
	return
}

// Sync uploads all the pending full buffers to GCS.
func (wh *BufferedWriteHandler) Sync() (err error) {
	wh.uploadHandler.AwaitBlocksUpload()

	select {
	case <-wh.uploadHandler.SignalUploadFailure():
		return ErrUploadFailure
	default:
		return nil
	}
}

// Flush finalizes the upload.
func (wh *BufferedWriteHandler) Flush() (*gcs.MinObject, error) {
	// In case it is a truncated file, upload empty blocks as required.
	err = wh.writeDataForTruncatedSize()
	if err != nil {
		return
	}

	if wh.current != nil {
		err := wh.uploadHandler.Upload(wh.current)
		if err != nil {
			return nil, err
		}
		wh.current = nil
	}

	obj, err := wh.uploadHandler.Finalize()
	if err != nil {
		return nil, fmt.Errorf("BufferedWriteHandler.Flush(): %w", err)
	}

	err = wh.blockPool.ClearFreeBlockChannel()
	if err != nil {
		// Only logging an error in case of resource leak as upload succeeded.
		logger.Errorf("blockPool.ClearFreeBlockChannel() failed: %v", err)
	}

	// Return an error along with object if the uploadHandler failed in between.
	select {
	case <-wh.uploadHandler.SignalUploadFailure():
		return obj, ErrUploadFailure
	default:
		break
	}

	return obj, nil
}

// SetMtime stores the mtime with the bufferedWriteHandler.
func (wh *BufferedWriteHandler) SetMtime(mtime time.Time) {
	wh.mtime = mtime
}

func (wh *BufferedWriteHandler) Truncate(size int64) error {
	if size < wh.totalSize {
		return fmt.Errorf("cannot truncate to lesser size when upload is in progress")
	}

	wh.truncatedSize = size
	return nil
}

// WriteFileInfo returns the file info i.e, how much data has been buffered so far
// and the mtime.
func (wh *BufferedWriteHandler) WriteFileInfo() WriteFileInfo {
	return WriteFileInfo{
		TotalSize: int64(math.Max(float64(wh.totalSize), float64(wh.truncatedSize))),
		Mtime:     wh.mtime,
	}
}

func (wh *BufferedWriteHandler) writeDataForTruncatedSize() error {
	// If totalSize is greater than truncatedSize, that means user has
	// written more data than they actually truncated in the beginning.
	if wh.totalSize < wh.truncatedSize {
		diff := wh.truncatedSize - wh.totalSize
		err := wh.Write(make([]byte, diff), wh.totalSize)
		if err != nil {
			return err
		}
	}

	return nil
}
