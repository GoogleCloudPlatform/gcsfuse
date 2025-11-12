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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
)

// blockQueueEntry holds a data block with a function to cancel its in-flight download.
type blockQueueEntry struct {
	block             block.PrefetchBlock
	cancel            context.CancelFunc
	prefetchTriggered bool
	wasEvicted        bool
}

// Size returns the size of the block in bytes to implement lru.ValueType.
func (bqe *blockQueueEntry) Size() uint64 {
	return uint64(bqe.block.Cap())
}
