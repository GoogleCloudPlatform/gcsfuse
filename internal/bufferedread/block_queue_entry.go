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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

// blockQueueEntry holds a data block with a function
// to cancel its in-flight download.
type blockQueueEntry struct {
	block  block.PrefetchBlock
	cancel context.CancelFunc

	// wasEvicted is true if the block has been removed from the block queue but
	// still has outstanding references.
	wasEvicted bool
}

// cancelAndWait cancels the download context for the entry and waits for the
// download goroutine to finish. It logs a warning if the download terminates
// with an error other than context.Canceled.
func (bqe *blockQueueEntry) cancelAndWait() {
	bqe.cancel()
	// We wait for the block's worker goroutine to finish. We expect its
	// status to contain a context.Canceled error because we just called cancel.
	status, err := bqe.block.AwaitReady(context.Background())
	if err != nil {
		logger.Warnf("cancelAndWait: AwaitReady for block starting at %d failed: %v",
			bqe.block.AbsStartOff(), err)
	} else if status.Err != nil && !errors.Is(status.Err, context.Canceled) {
		logger.Warnf("cancelAndWait: block starting at %d terminated with an unexpected error: %v",
			bqe.block.AbsStartOff(), status.Err)
	}
}
