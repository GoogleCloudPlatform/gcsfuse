// Copyright 2025 Google LLC
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

package bufferedread

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

type DownloadTask struct {
	workerpool.Task
	object       *gcs.MinObject
	bucket       gcs.Bucket
	metricHandle metrics.MetricHandle

	// block is the block to which the data will be downloaded.
	block block.PrefetchBlock

	// ctx is the context for the download task. It is used to cancel the download.
	ctx context.Context

	// Used for zonal bucket to bypass the auth & metadata checks.
	readHandle []byte

	// readHandleUpdater is called with the updated read handle after successful reading.
	// This allows the caller to update their read handle for future efficient reads.
	readHandleUpdater func([]byte)
}

func NewDownloadTask(ctx context.Context, object *gcs.MinObject, bucket gcs.Bucket, block block.PrefetchBlock, readHandle []byte, metricHandle metrics.MetricHandle, readHandleUpdater func([]byte)) *DownloadTask {
	return &DownloadTask{
		ctx:               ctx,
		object:            object,
		bucket:            bucket,
		block:             block,
		readHandle:        readHandle,
		metricHandle:      metricHandle,
		readHandleUpdater: readHandleUpdater,
	}
}

// Execute implements the workerpool.Task interface. It downloads the data from
// the GCS object to the block.
// After completion, it notifies the block consumer about the status of the
// download task. The status can be one of the following:
// - BlockStatusDownloaded: The download was successful.
// - BlockStatusDownloadFailed: The download failed due to an error.
func (dt *DownloadTask) Execute() {
	startOff := dt.block.AbsStartOff()
	blockId := startOff / dt.block.Cap()
	logger.Tracef("Download: <- block (%s, %v).", dt.object.Name, blockId)
	stime := time.Now()
	var err error
	defer func() {
		var status string
		dur := time.Since(stime)
		if err == nil {
			status = "successful"
			logger.Tracef("Download: -> block (%s, %v) Ok(%v).", dt.object.Name, blockId, dur)
			dt.block.NotifyReady(block.BlockStatus{State: block.BlockStateDownloaded})
		} else if errors.Is(err, context.Canceled) && dt.ctx.Err() == context.Canceled {
			status = "cancelled"
			logger.Tracef("Download: -> block (%s, %v) cancelled: %v.", dt.object.Name, blockId, err)
			dt.block.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadFailed, Err: err})
		} else {
			status = "failed"
			logger.Errorf("Download: -> block (%s, %v) failed: %v.", dt.object.Name, blockId, err)
			dt.block.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadFailed, Err: err})
		}
		dt.metricHandle.BufferedReadDownloadBlockLatency(dt.ctx, dur, status)
		dt.metricHandle.BufferedReadScheduledBlockCount(1, status)
	}()

	start := uint64(startOff)
	end := start + uint64(dt.block.Cap())
	if end > dt.object.Size {
		end = dt.object.Size
	}
	newReader, err := dt.bucket.NewReaderWithReadHandle(
		dt.ctx,
		&gcs.ReadObjectRequest{
			Name:       dt.object.Name,
			Generation: dt.object.Generation,
			Range: &gcs.ByteRange{
				Start: start,
				Limit: end,
			},
			ReadCompressed: dt.object.HasContentEncodingGzip(),
			ReadHandle:     dt.readHandle,
		})
	if err != nil {
		var notFoundError *gcs.NotFoundError
		if errors.As(err, &notFoundError) {
			err = &gcsfuse_errors.FileClobberedError{Err: err, ObjectName: dt.object.Name}
			return
		}
		err = fmt.Errorf("DownloadTask.Execute: while reader-creations: %w", err)
		return
	}
	defer newReader.Close()

	_, err = io.CopyN(dt.block, newReader, int64(end-start))
	if err != nil {
		err = fmt.Errorf("DownloadTask.Execute: while data-copy: %w", err)
		return
	}

	// Capture the updated read handle for future efficient reads
	if dt.readHandleUpdater != nil {
		updatedReadHandle := newReader.ReadHandle()
		dt.readHandleUpdater(updatedReadHandle)
	}
}
