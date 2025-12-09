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

type downloadTask struct {
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
}

// Execute implements the workerpool.Task interface. It downloads the data from
// the GCS object to the block.
// After completion, it notifies the block consumer about the status of the
// download task. The status can be one of the following:
// - BlockStatusDownloaded: The download was successful.
// - BlockStatusDownloadFailed: The download failed due to an error.
func (p *downloadTask) Execute() {
	startOff := p.block.AbsStartOff()
	blockId := startOff / p.block.Cap()
	logger.Tracef("Download: <- block (%s, %v).", p.object.Name, blockId)
	stime := time.Now()
	var err error
	var n int64
	defer func() {
		dur := time.Since(stime)
		if err == nil {
			logger.Tracef("Download: -> block (%s, %v) Ok(%v).", p.object.Name, blockId, dur)
			p.block.NotifyReady(block.BlockStatus{State: block.BlockStateDownloaded})
		} else if errors.Is(err, context.Canceled) && p.ctx.Err() == context.Canceled {
			logger.Tracef("Download: -> block (%s, %v) cancelled: %v.", p.object.Name, blockId, err)
			p.block.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadFailed, Err: err})
		} else {
			logger.Errorf("Download: -> block (%s, %v) failed: %v.", p.object.Name, blockId, err)
			p.block.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadFailed, Err: err})
		}
		p.metricHandle.GcsDownloadBytesCount(n, metrics.ReadTypeBufferedAttr)
	}()

	start := uint64(startOff)
	end := min(start+uint64(p.block.Cap()), p.object.Size)
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

	n, err = io.Copy(p.block, newReader)
	if err != nil {
		err = fmt.Errorf("DownloadTask.Execute: while data-copy: %w", err)
		return
	}
	if n != int64(end-start) {
		err = fmt.Errorf("DownloadTask.Execute: downloaded %d bytes, expected %d", n, int64(end-start))
		return
	}
}
