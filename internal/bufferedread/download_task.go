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
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
)

type DownloadTask struct {
	workerpool.Task
	object *gcs.MinObject
	bucket gcs.Bucket

	// block is the block to which the data will be downloaded.
	block block.PrefetchBlock

	// ctx is the context for the download task. It is used to cancel the download.
	ctx context.Context

	// Used for zonal bucket to bypass the auth & metadata checks.
	readHandle []byte
}

func NewDownloadTask(ctx context.Context, object *gcs.MinObject, bucket gcs.Bucket, block block.PrefetchBlock, readHandle []byte) *DownloadTask {
	return &DownloadTask{
		ctx:        ctx,
		object:     object,
		bucket:     bucket,
		block:      block,
		readHandle: readHandle,
	}
}

// Execute implements the workerpool.Task interface. It downloads the data from
// the GCS object to the block.
// After completion, it notifies the block consumer about the status of the
// download task. The status can be one of the following:
// - BlockStatusDownloaded: The download was successful.
// - BlockStatusDownloadFailed: The download failed due to an error.
func (p *DownloadTask) Execute() {
	startOff := p.block.AbsStartOff()
	blockId := startOff / p.block.Cap()
	logger.Tracef("Download: <- block (%s, %v).", p.object.Name, blockId)
	stime := time.Now()
	var err error
	defer func() {
		if err == nil {
			logger.Tracef("Download: -> block (%s, %v) completed in: %v.", p.object.Name, blockId, time.Since(stime))
			p.block.NotifyReady(block.BlockStatus{State: block.BlockStateDownloaded})
		} else if errors.Is(err, context.Canceled) && p.ctx.Err() == context.Canceled {
			logger.Tracef("Download: -> block (%s, %v) cancelled: %v.", p.object.Name, blockId, err)
			p.block.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadFailed, Err: err})
		} else {
			logger.Errorf("Download: -> block (%s, %v) failed: %v.", p.object.Name, blockId, err)
			p.block.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadFailed, Err: err})
		}
	}()

	start := uint64(startOff)
	end := start + uint64(p.block.Cap())
	if end > p.object.Size {
		end = p.object.Size
	}
	newReader, err := p.bucket.NewReaderWithReadHandle(
		p.ctx,
		&gcs.ReadObjectRequest{
			Name:       p.object.Name,
			Generation: p.object.Generation,
			Range: &gcs.ByteRange{
				Start: start,
				Limit: end,
			},
			ReadCompressed: p.object.HasContentEncodingGzip(),
			ReadHandle:     p.readHandle,
		})
	if err != nil {
		err = fmt.Errorf("while reader-creations: %w", err)
		return
	}

	_, err = io.CopyN(p.block, newReader, int64(end-start))
	if err != nil {
		err = fmt.Errorf("while copying data: %w", err)
		return
	}
}
