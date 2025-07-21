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
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"golang.org/x/net/context"
)

type BufferedReadConfig struct {
	PrefetchQueueCapacity   int64 // Maximum number of blocks that can be prefetched
	PrefetchBlockSizeBytes  int64 // Size of each block to be prefetched.
	InitialPrefetchBlockCnt int64 // Number of blocks to prefetch initially.
	PrefetchMultiplier      int64 // Multiplier for number of blocks to prefetch.
	RandomReadsThreshold    int64 // Number of random reads after which the reader falls back to GCS reader.
}

// blockQueueEntry holds a data block with a function
// to cancel its in-flight download.
type blockQueueEntry struct {
	// TODO: Add block.Block and context.CancelFunc
}

type BufferedReader struct {
	object *gcs.MinObject
	bucket gcs.Bucket
	config *BufferedReadConfig

	nextBlockIndexToPrefetch int64
	randomSeekCount          int64
	nextPrefetchBlockCount   int64

	blockQueue common.Queue[*blockQueueEntry]

	// TODO: Add readHandle for zonal bucket optimization.

	blockPool    *block.BlockPool
	workerPool   workerpool.WorkerPool
	metricHandle metrics.MetricHandle

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewBufferedReader(object *gcs.MinObject, bucket gcs.Bucket, config *BufferedReadConfig, blockPool *block.BlockPool, workerPool workerpool.WorkerPool, metricHandle metrics.MetricHandle) *BufferedReader {
	reader := &BufferedReader{
		object:                   object,
		bucket:                   bucket,
		config:                   config,
		nextBlockIndexToPrefetch: -1,
		randomSeekCount:          0,
		nextPrefetchBlockCount:   config.InitialPrefetchBlockCnt,
		blockQueue:               common.NewLinkedListQueue[*blockQueueEntry](),
		blockPool:                blockPool,
		workerPool:               workerPool,
		metricHandle:             metricHandle,
	}

	reader.ctx, reader.cancelFunc = context.WithCancel(context.Background())
	return reader
}
