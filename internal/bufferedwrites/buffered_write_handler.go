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
	"fmt"
	"math"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
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
	// been uploaded to GCS as well.
	totalSize int64
	// Stores the mtime value updated by kernel as part of setInodeAttributes call.
	mtime time.Time
}

// WriteFileInfo is used as part of serving fileInode attributes (GetInodeAttributes call).
type WriteFileInfo struct {
	TotalSize int64
	Mtime     time.Time
}

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
	if offset > wh.totalSize {
		// TODO: Will be handled as part of ordered writes.
		return fmt.Errorf("non sequential writes")
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
	// TODO: Will be added after uploadHandler changes are done.
	return fmt.Errorf("not implemented")
}

// Flush finalizes the upload.
func (wh *BufferedWriteHandler) Flush() (err error) {
	if wh.current != nil {
		err := wh.uploadHandler.Upload(wh.current)
		if err != nil {
			return err
		}
		wh.current = nil
	}
	return wh.uploadHandler.Finalize()
}

// SetMtime stores the mtime with the bufferedWriteHandler.
func (wh *BufferedWriteHandler) SetMtime(mtime time.Time) {
	wh.mtime = mtime
}

// WriteFileInfo returns the file info i.e, how much data has been buffered so far
// and the mtime.
func (wh *BufferedWriteHandler) WriteFileInfo() WriteFileInfo {
	return WriteFileInfo{
		TotalSize: wh.totalSize,
		Mtime:     wh.mtime,
	}
}

func (wh *BufferedWriteHandler) SignalNonRecoverableFailure() chan error {
	return wh.uploadHandler.signalNonRecoverableFailure
}

func (wh *BufferedWriteHandler) SignalUploadFailure() chan error {
	return wh.uploadHandler.signalUploadFailure
}

func (wh *BufferedWriteHandler) TempFileChannel() chan gcsx.TempFile {
	return wh.uploadHandler.tempFile
}
