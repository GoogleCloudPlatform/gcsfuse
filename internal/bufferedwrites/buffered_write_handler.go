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
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"golang.org/x/sync/semaphore"
)

// Note: All the write operations take inode lock in fs.go, hence we don't need any locks here
// as we will get write operations serially.

type BufferedWriteHandler interface {
	// Write writes the given data to the buffer. It writes to an existing buffer if
	// the capacity is available otherwise writes to a new buffer.
	Write(ctx context.Context, data []byte, offset int64) (err error)

	// Sync uploads all the pending buffers to GCS.
	// Sync returns
	// 1. un-finalized object created on GCS for zonal buckets.
	// 2. nil object for non-zonal buckets.
	Sync(ctx context.Context) (*gcs.MinObject, error)

	// Flush finalizes the upload.
	Flush(ctx context.Context) (*gcs.MinObject, error)

	// SetMtime stores the mtime with the bufferedWriteHandler.
	SetMtime(mtime time.Time)

	// Truncate allows truncating the file to a larger size.
	Truncate(size int64) error

	// WriteFileInfo returns the file info i.e, how much data has been buffered so far
	// and the mtime.
	WriteFileInfo() WriteFileInfo

	// Destroy destroys the upload handler and then free up the buffers.
	Destroy() error

	// Unlink cancels the ongoing upload and free up the buffers.
	Unlink()
}

// bufferedWriteHandlerImpl is responsible for filling up the buffers with the data
// as it receives and handing over to uploadHandler which uploads to GCS.
type bufferedWriteHandlerImpl struct {
	filePath      string
	current       block.Block
	blockPool     *block.GenBlockPool[block.Block]
	uploadHandler *UploadHandler
	metricHandle  metrics.MetricHandle
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

	// Timing accumulators for fault attribution metrics:
	startTime          time.Time
	lastWriteEndTime   time.Time
	appWaitDuration    time.Duration
	blockWaitDuration  time.Duration
	outOfOrderDetected bool
}

// WriteFileInfo is used as part of serving fileInode attributes (GetInodeAttributes call).
type WriteFileInfo struct {
	TotalSize int64
	Mtime     time.Time
}

var ErrOutOfOrderWrite = errors.New("outOfOrder write detected")

type CreateBWHandlerRequest struct {
	Object                   *gcs.Object
	ObjectName               string
	FilePath                 string
	Bucket                   gcs.Bucket
	BlockSize                int64
	MaxBlocksPerFile         int64
	GlobalMaxBlocksSem       *semaphore.Weighted
	ChunkRetryDeadlineSecs   int64
	ChunkTransferTimeoutSecs int64
	TraceHandle              tracing.TraceHandle
	MetricHandle             metrics.MetricHandle
}

// NewBWHandler creates the bufferedWriteHandler struct.
func NewBWHandler(req *CreateBWHandlerRequest) (bwh BufferedWriteHandler, err error) {
	bp, err := block.NewBlockPool(req.BlockSize, req.MaxBlocksPerFile, 1, req.GlobalMaxBlocksSem)
	if err != nil {
		return
	}
	var size int64
	if req.Object != nil {
		size = int64(req.Object.Size)
	}

	mh := req.MetricHandle
	if mh == nil {
		mh = metrics.NewNoopMetrics()
	}

	filePath := req.FilePath
	if filePath == "" {
		filePath = req.ObjectName
	}

	bwh = &bufferedWriteHandlerImpl{
		filePath:  filePath,
		current:   nil,
		blockPool: bp,
		uploadHandler: newUploadHandler(&CreateUploadHandlerRequest{
			Object:                   req.Object,
			ObjectName:               req.ObjectName,
			Bucket:                   req.Bucket,
			BlockPool:                bp,
			MaxBlocksPerFile:         req.MaxBlocksPerFile,
			BlockSize:                req.BlockSize,
			ChunkRetryDeadlineSecs:   req.ChunkRetryDeadlineSecs,
			ChunkTransferTimeoutSecs: req.ChunkTransferTimeoutSecs,
			TraceHandle:              req.TraceHandle,
		}),
		metricHandle:  mh,
		totalSize:     size,
		mtime:         time.Now(),
		truncatedSize: -1,
	}

	logger.Infof("Buffered write handler initialized for file: %s", filePath)

	return
}

func (wh *bufferedWriteHandlerImpl) Write(ctx context.Context, data []byte, offset int64) (err error) {
	logger.Infof("Buffered write handler write for file: %s, offset: %d, size: %d", wh.filePath, offset, len(data))

	now := time.Now()
	if wh.startTime.IsZero() {
		wh.startTime = now
	} else if !wh.lastWriteEndTime.IsZero() {
		wh.appWaitDuration += now.Sub(wh.lastWriteEndTime)
	}
	defer func() {
		wh.lastWriteEndTime = time.Now()
	}()

	// Fail early if the uploadHandler has already failed.
	err = wh.uploadHandler.UploadError()
	if err != nil {
		return
	}
	// Once we write past the truncated size, any writes starting from the truncated
	// offset are considered out of order. For example, if a file is truncated to 10
	// bytes, and we write 10 bytes starting from offset 5, the total size becomes 15.
	// A subsequent write at offset 10 (the truncated size) will be rejected as an out of order write.
	if offset != wh.totalSize && (offset != wh.truncatedSize || wh.totalSize >= wh.truncatedSize) {
		wh.outOfOrderDetected = true
		logger.Errorf("BufferedWriteHandler.OutOfOrderError for object: %s, expectedOffset: %d, actualOffset: %d",
			wh.uploadHandler.objectName, wh.totalSize, offset)
		return ErrOutOfOrderWrite
	}

	if offset == wh.truncatedSize {
		// Check and update if any data filling has to be done.
		err = wh.writeDataForTruncatedSize(ctx)
		if err != nil {
			return
		}
	}

	return wh.appendBuffer(ctx, data)
}

func (wh *bufferedWriteHandlerImpl) appendBuffer(ctx context.Context, data []byte) (err error) {
	dataWritten := 0
	for dataWritten < len(data) {
		if wh.current == nil {
			startBlockGet := time.Now()
			wh.current, err = wh.blockPool.Get()
			wh.blockWaitDuration += time.Since(startBlockGet)
			if err != nil {
				return fmt.Errorf("failed to get new block: %w", err)
			}
		}

		remainingBlockSize := float64(wh.blockPool.BlockSize()) - float64(wh.current.Size())
		pendingDataForWrite := float64(len(data)) - float64(dataWritten)
		bytesToCopy := int(math.Min(remainingBlockSize, pendingDataForWrite))
		_, err := wh.current.Write(data[dataWritten : dataWritten+bytesToCopy])
		if err != nil {
			return err
		}

		dataWritten += bytesToCopy

		if wh.current.Size() == wh.blockPool.BlockSize() {
			err := wh.uploadHandler.Upload(ctx, wh.current)
			if err != nil {
				return err
			}
			wh.current = nil
		}
	}

	wh.totalSize += int64(dataWritten)

	// If the file size has surpassed the truncation point, the truncation requirement
	// is fulfilled and we can safely discard the stale offset.
	if wh.truncatedSize != -1 && wh.totalSize >= wh.truncatedSize {
		wh.truncatedSize = -1
	}

	return
}

func (wh *bufferedWriteHandlerImpl) Sync(ctx context.Context) (o *gcs.MinObject, err error) {
	logger.Infof("Buffered write handler sync for file: %s, total size: %d", wh.filePath, wh.totalSize)

	// Upload current block (for both regional and zonal buckets).
	if wh.current != nil && wh.current.Size() != 0 {
		err = wh.uploadHandler.Upload(ctx, wh.current)
		if err != nil {
			return nil, err
		}
		wh.current = nil
	}
	// Upload all the pending buffers.
	wh.uploadHandler.AwaitBlocksUpload()
	// The FlushPendingWrites method synchronizes all bytes currently residing in
	// the Writer's buffer to Cloud Storage, thereby making them available for
	// other operations like read.
	// This functionality is exclusively supported on rapid buckets.
	if wh.uploadHandler.bucket.BucketType().RapidWritesEnabled() {
		o, err = wh.uploadHandler.FlushPendingWrites(ctx)
		if err != nil {
			return nil, err
		}
		if o.Size != uint64(wh.totalSize) {
			return nil, fmt.Errorf("could not upload entire data, expected size %d, got %d", wh.totalSize, o.Size)
		}
	}
	// Release memory used by buffers.
	err = wh.blockPool.ClearFreeBlockChannel(false)
	if err != nil {
		// Only logging an error in case of resource leak as upload succeeded.
		logger.Errorf("blockPool.ClearFreeBlockChannel() failed during sync: %v", err)
	}
	err = wh.uploadHandler.UploadError()
	if err != nil {
		return nil, err
	}
	return o, nil
}

// Flush finalizes the upload.
func (wh *bufferedWriteHandlerImpl) Flush(ctx context.Context) (*gcs.MinObject, error) {
	logger.Infof("Buffered write handler flush for file: %s, total size: %d", wh.filePath, wh.totalSize)

	flushStart := time.Now()
	if wh.startTime.IsZero() {
		wh.startTime = flushStart
	} else if !wh.lastWriteEndTime.IsZero() {
		wh.appWaitDuration += flushStart.Sub(wh.lastWriteEndTime)
	}

	// Fail early if upload already failed.
	err := wh.uploadHandler.UploadError()
	if err != nil {
		return nil, err
	}

	// In case it is a truncated file, upload empty blocks as required.
	err = wh.writeDataForTruncatedSize(ctx)
	if err != nil {
		return nil, err
	}

	if wh.current != nil {
		err := wh.uploadHandler.Upload(ctx, wh.current)
		if err != nil {
			return nil, err
		}
		wh.current = nil
	}

	finalizeStart := time.Now()
	obj, err := wh.uploadHandler.Finalize(ctx)
	if err != nil {
		return nil, fmt.Errorf("BufferedWriteHandler.Flush(): %w", err)
	}

	logger.Infof("Buffered write handler finalized file: %s, size: %d", wh.filePath, wh.totalSize)

	if obj != nil && obj.Size != uint64(wh.totalSize) {
		return nil, fmt.Errorf("could not upload entire data, expected size %d, got %d", wh.totalSize, obj.Size)
	}

	err = wh.blockPool.ClearFreeBlockChannel(true)
	if err != nil {
		// Only logging an error in case of resource leak as upload succeeded.
		logger.Errorf("blockPool.ClearFreeBlockChannel() failed: %v", err)
	}

	finalizeDuration := time.Since(finalizeStart)
	totalDuration := time.Since(wh.startTime)
	if !wh.outOfOrderDetected {
		wh.recordMetrics(ctx, totalDuration, finalizeDuration)
	}

	return obj, nil
}

func (wh *bufferedWriteHandlerImpl) recordMetrics(ctx context.Context, totalDuration time.Duration, finalizeDuration time.Duration) {
	if wh.metricHandle == nil {
		return
	}

	bottleneck := metrics.BottleneckBalancedAttr
	if totalDuration > 0 {
		appRatio := float64(wh.appWaitDuration) / float64(totalDuration)
		blockWaitRatio := float64(wh.blockWaitDuration) / float64(totalDuration)
		finalizeRatio := float64(finalizeDuration) / float64(totalDuration)

		if appRatio >= 0.60 {
			bottleneck = metrics.BottleneckAppBoundAttr
		} else if blockWaitRatio >= 0.40 {
			bottleneck = metrics.BottleneckUploadBoundAttr
		} else if finalizeRatio >= 0.50 {
			bottleneck = metrics.BottleneckFinalizeBoundAttr
		} else if (blockWaitRatio + finalizeRatio) >= 0.60 {
			if finalizeRatio > blockWaitRatio {
				bottleneck = metrics.BottleneckFinalizeBoundAttr
			} else {
				bottleneck = metrics.BottleneckUploadBoundAttr
			}
		}
	}

	wh.metricHandle.BufferedWriteTotalLatency(ctx, totalDuration, bottleneck)
	wh.metricHandle.BufferedWriteAppWaitLatency(ctx, wh.appWaitDuration)
	wh.metricHandle.BufferedWriteBlockPoolWaitLatency(ctx, wh.blockWaitDuration)
	wh.metricHandle.BufferedWriteFinalizeLatency(ctx, finalizeDuration)
}

func (wh *bufferedWriteHandlerImpl) SetMtime(mtime time.Time) {
	wh.mtime = mtime
}

func (wh *bufferedWriteHandlerImpl) Truncate(size int64) error {
	now := time.Now()
	if wh.startTime.IsZero() {
		wh.startTime = now
	} else if !wh.lastWriteEndTime.IsZero() {
		wh.appWaitDuration += now.Sub(wh.lastWriteEndTime)
	}
	defer func() {
		wh.lastWriteEndTime = time.Now()
	}()

	if size < wh.totalSize {
		wh.outOfOrderDetected = true
		return ErrOutOfOrderWrite
	}

	wh.truncatedSize = size
	return nil
}

func (wh *bufferedWriteHandlerImpl) WriteFileInfo() WriteFileInfo {
	return WriteFileInfo{
		TotalSize: int64(math.Max(float64(wh.totalSize), float64(wh.truncatedSize))),
		Mtime:     wh.mtime,
	}
}

func (wh *bufferedWriteHandlerImpl) Destroy() error {
	if wh.current != nil {
		wh.blockPool.Release(wh.current)
		wh.current = nil
	}
	wh.uploadHandler.Destroy()
	return wh.blockPool.ClearFreeBlockChannel(true)
}

func (wh *bufferedWriteHandlerImpl) writeDataForTruncatedSize(ctx context.Context) error {
	// If totalSize is greater than truncatedSize, that means user has
	// written more data than they actually truncated in the beginning.
	if wh.totalSize >= wh.truncatedSize {
		return nil
	}

	// Otherwise append dummy data to match truncatedSize.
	diff := wh.truncatedSize - wh.totalSize
	// Create 1MB of data at a time to avoid OOM
	chunkSize := 1024 * 1024
	for i := 0; i < int(diff); i += chunkSize {
		size := math.Min(float64(chunkSize), float64(int(diff)-i))
		err := wh.appendBuffer(ctx, make([]byte, int(size)))
		if err != nil {
			return err
		}
	}

	return nil
}

func (wh *bufferedWriteHandlerImpl) Unlink() {
	wh.uploadHandler.CancelUpload()
	// Since bwh is not cleared after unlink, we will not release last block yet.
	// Last block will be released when file handle for this file is closed.
	err := wh.blockPool.ClearFreeBlockChannel(false)
	if err != nil {
		// Only logging an error in case of resource leak.
		logger.Errorf("blockPool.ClearFreeBlockChannel() failed: %v", err)
	}
}
