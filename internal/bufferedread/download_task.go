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
	blocks []block.PrefetchBlock

	// ctx is the context for the download task. It is used to cancel the download.
	ctx context.Context

	// Used for zonal bucket to bypass the auth & metadata checks.
	readHandle []byte
}

func NewDownloadTask(ctx context.Context, object *gcs.MinObject, bucket gcs.Bucket, block block.PrefetchBlock, readHandle []byte, metricHandle metrics.MetricHandle) *DownloadTask {
	dt := &DownloadTask{
		ctx:          ctx,
		object:       object,
		bucket:       bucket,
		readHandle:   readHandle,
		metricHandle: metricHandle,
	}
	dt.blocks = append(dt.blocks, block)
	return dt
}

func NewDownloadTaskWithMultipleBlocks(ctx context.Context, object *gcs.MinObject, bucket gcs.Bucket, blocks []block.PrefetchBlock, readHandle []byte, metricHandle metrics.MetricHandle) *DownloadTask {
	return &DownloadTask{
		ctx:          ctx,
		object:       object,
		bucket:       bucket,
		blocks:       blocks,
		readHandle:   readHandle,
		metricHandle: metricHandle,
	}
}

// Execute implements the workerpool.Task interface. It downloads the data from
// the GCS object to the block.
// After completion, it notifies the block consumer about the status of the
// download task. The status can be one of the following:
// - BlockStatusDownloaded: The download was successful.
// - BlockStatusDownloadFailed: The download failed due to an error.
func (p *DownloadTask) Execute() {
	if len(p.blocks) == 0 {
		return
	}
	startOffset := p.blocks[0].AbsStartOff()
	start := uint64(startOffset)
	end := min(start+uint64(p.blocks[0].Cap())*uint64(len(p.blocks)), p.object.Size)
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
		var notFoundError *gcs.NotFoundError
		if errors.As(err, &notFoundError) {
			err = &gcsfuse_errors.FileClobberedError{Err: err, ObjectName: p.object.Name}
			return
		}
		err = fmt.Errorf("DownloadTask.Execute: while reader-creations: %w", err)
		return
	}
	defer newReader.Close()

	for _, b := range p.blocks {
		_, err := p.downloadSingleBlock(b, newReader)
		if err != nil {
			return
		}
	}
}

func (p *DownloadTask) downloadSingleBlock(b block.PrefetchBlock, reader gcs.StorageReader) (n int64, err error) {
	startOff := b.AbsStartOff()
	blockId := startOff / b.Cap()
	logger.Tracef("Download: <- block (%s, %v).", p.object.Name, startOff/b.Cap())
	start := uint64(b.AbsStartOff())
	end := min(start+uint64(b.Cap()), p.object.Size)
	stime := time.Now()
	defer func() {
		dur := time.Since(stime)
		if err == nil {
			logger.Tracef("Download: -> block (%s, %v) Ok(%v).", p.object.Name, blockId, dur)
			b.NotifyReady(block.BlockStatus{State: block.BlockStateDownloaded})
		} else if errors.Is(err, context.Canceled) && p.ctx.Err() == context.Canceled {
			logger.Tracef("Download: -> block (%s, %v) cancelled: %v.", p.object.Name, blockId, err)
			b.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadFailed, Err: err})
		} else {
			logger.Errorf("Download: -> block (%s, %v) failed: %v.", p.object.Name, blockId, err)
			b.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadFailed, Err: err})
		}
		p.metricHandle.GcsDownloadBytesCount(n, metrics.ReadTypeBufferedAttr)
	}()
	n, err = io.CopyN(b, reader, int64(end-start))
	if err != nil {
		err = fmt.Errorf("DownloadTask.Execute: while data-copy: %w", err)
		return
	}
	return
}
