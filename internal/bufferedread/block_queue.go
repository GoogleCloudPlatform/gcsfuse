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
)

// blockQueueEntry represents a block of object data and the associated task
// responsible for downloading it.
type blockQueueEntry struct {
	Block block.Block
	Task  *DownloadTask
}

// newBlockQueueEntry creates a new entry for the block queue.
func newBlockQueueEntry(b block.Block, t *DownloadTask) *blockQueueEntry {
	return &blockQueueEntry{
		Block: b,
		Task:  t,
	}
}

// blockQueue is a queue of blockQueueEntry instances.
type blockQueue struct {
	q common.Queue[*blockQueueEntry]
}

// newBlockQueue creates a new, empty queue for managing block download tasks.
func newBlockQueue() *blockQueue {
	return &blockQueue{q: common.NewLinkedListQueue[*blockQueueEntry]()}
}
