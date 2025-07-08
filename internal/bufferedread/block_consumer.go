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

	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
)

type BlockConsumer struct {
	blockPool  block.BlockPool
	workerPool workerpool.WorkerPool
	blockQueue common.Queue[BlockQueueEntry]

	bucket gcs.Bucket
	object *gcs.MinObject
}

func NewBlockConsumer(bucket gcs.Bucket, object *gcs.MinObject, blockPool block.BlockPool, workerPool workerpool.WorkerPool) *BlockConsumer {
	return &BlockConsumer{
		bucket:     bucket,
		object:     object,
		blockPool:  blockPool,
		workerPool: workerPool,
	}
}

func (bc *BlockConsumer) Consume(block block.PrefetchBlock, prefetch bool) {
	task := NewPrefetchTask(context.Background(), bc.object, bc.bucket, block, prefetch)
	bc.workerPool.Schedule(prefetch, task)

}

