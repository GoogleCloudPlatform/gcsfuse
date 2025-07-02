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

package buffered_reader

import (
	"fmt"
	"io"
	"time"

	"errors"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

// One PrefetchTask to be scheduled
type PrefetchTask struct {
	object   *gcs.MinObject
	bucket   gcs.Bucket
	block    *Block // Block to hold data for this item
	prefetch bool   // Flag marking this is a prefetch request or not
	failCnt  int32
	ctx      context.Context
}

/**
* download downloads a range of bytes from the object.
* This method is used by the thread-pool scheduler.
 */
func Download(task *PrefetchTask) {
	blockId := task.block.id
	logger.Tracef("Download: <- block (%s, %v).", task.object.Name, blockId)
	stime := time.Now()

	var err error
	defer func() {
		if err != nil {
			logger.Tracef("Download: -> block (%s, %v) failed with error: %v.", task.object.Name, blockId, err)
		} else {
			logger.Tracef("Download: -> block (%s, %v): %v completed.", task.object.Name, blockId, time.Since(stime))
		}
	}()

	start := uint64(task.block.offset)
	end := task.block.offset + GetBlockSize(task.block, uint64(len(task.block.data)), task.object.Size)

	newReader, err := task.bucket.NewReaderWithReadHandle(
		task.ctx,
		&gcs.ReadObjectRequest{
			Name:       task.object.Name,
			Generation: task.object.Generation,
			Range: &gcs.ByteRange{
				Start: start,
				Limit: end,
			},
			ReadCompressed: task.object.HasContentEncodingGzip(),
			ReadHandle:     nil,
		})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Warnf("Download block (%s, %v): %v failed with context cancelled.", task.object.Name, blockId, err)
			task.block.Ready(BlockStatusDownloadCancelled)
		} else {
			err = fmt.Errorf("downloadRange: error in creating reader(%d, %d), error: %v", start, end, err)
			task.block.Ready(BlockStatusDownloadFailed)
		}
		return
	}

	_, err = io.CopyN(task.block, newReader, int64(end-start))
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Warnf("Download block (%s, %v): %v failed with context cancelled.", task.object.Name, blockId, err)
			task.block.Ready(BlockStatusDownloadCancelled)
		} else {
			err = fmt.Errorf("downloadRange: error copying the content to block: %v", err)
			task.block.Ready(BlockStatusDownloadFailed)
		}
		return
	}

	task.block.Ready(BlockStatusDownloaded)
}

func GetBlockSize(block *Block, blockSize uint64, objectSize uint64) uint64 {
	return min(blockSize, objectSize-block.offset)
}
