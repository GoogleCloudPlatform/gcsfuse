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

type PrefetchTask struct {
	workerpool.Task
	block  block.PrefetchBlock
	object *gcs.MinObject
	bucket gcs.Bucket
	ctx    context.Context
}

func NewPrefetchTask(ctx context.Context, object *gcs.MinObject, bucket gcs.Bucket, block block.PrefetchBlock, prefetch bool) *PrefetchTask {
	return &PrefetchTask{
		ctx:    ctx,
		object: object,
		bucket: bucket,
		block:  block,
	}
}

func (p *PrefetchTask) Execute() {
	startOff := p.block.GetAbsStartOff()
	blockId := startOff / p.block.Cap()
	logger.Tracef("Download: <- block (%s, %v).", p.object.Name, blockId)
	stime := time.Now()

	var err error
	defer func() {
		if err != nil {
			logger.Tracef("Download: -> block (%s, %v) failed with error: %v.", p.object.Name, blockId, err)
		} else {
			logger.Tracef("Download: -> block (%s, %v): %v completed.", p.object.Name, blockId, time.Since(stime))
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
			ReadHandle:     nil,
		})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Warnf("Download block (%s, %v): %v failed with context cancelled while reader creation.", p.object.Name, blockId, err)
			p.block.NotifyReady(block.BlockStatusDownloadCancelled)
		} else {
			err = fmt.Errorf("downloadRange: error in creating reader(%d, %d), error: %v", start, end, err)
			p.block.NotifyReady(block.BlockStatusDownloadFailed)
		}
		return
	}

	_, err = io.CopyN(p.block, newReader, int64(end-start))
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Warnf("Download block (%s, %v): %v failed with context cancelled while reading.", p.object.Name, blockId, err)
			p.block.NotifyReady(block.BlockStatusDownloadCancelled)
		} else {
			err = fmt.Errorf("downloadRange: error copying the content to block: %v", err)
			p.block.NotifyReady(block.BlockStatusDownloadFailed)
		}
		return
	}

	p.block.NotifyReady(block.BlockStatusDownloaded)
}
