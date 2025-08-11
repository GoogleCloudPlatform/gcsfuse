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
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
)

// minBlockToStartBufferedRead returns the minimum number of blocks available
// to start buffered read based on the block size.
// Number of blocks logic is determined based on performance testing results
// and is subject to change.
func minBlockToStartBufferedRead(blockSize int64) (int, error) {
	if blockSize <= 0 {
		return 0, fmt.Errorf("invalid block size: %d", blockSize)
	}

	if blockSize <= 4*util.MiB {
		return 6, nil
	}

	if blockSize <= 8*util.MiB {
		return 4, nil
	}

	return 2, nil
}
