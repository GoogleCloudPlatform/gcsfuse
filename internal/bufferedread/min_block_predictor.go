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

type MinBlockPredictor interface {
	// PredictMinBlockCount predicts the minimum number of blocks required to
	// start buffered read based on the block size and object size.
	PredictMinBlockCount(blockSize int64, objectSize uint64) (uint, error)
}

// defaultMinBlockPredictor determines the minimum number of blocks required
// to start buffered read based on performance testing results and is subject
// to change.
// Also ensures, returned block-count * block-size is less than or equal to the
// object size.
type defaultMinBlockPredictor struct{}

func (d *defaultMinBlockPredictor) PredictMinBlockCount(blockSize int64, objectSize uint64) (uint, error) {
	if blockSize <= 0 {
		return 0, fmt.Errorf("invalid block-size: %d", blockSize)
	}

	// Cap the block count based on the object size.
	maxBlockCount := (objectSize + uint64(blockSize) - 1) / uint64(blockSize)

	if blockSize <= 4*util.MiB {
		return min(6, uint(maxBlockCount)), nil
	}

	if blockSize <= 8*util.MiB {
		return min(4, uint(maxBlockCount)), nil
	}

	return min(2, uint(maxBlockCount)), nil
}

// staticMinBlockPredictor is a MinBlockPredictor that returns a static
// minimum block count.
// Used for testing purposes.
type staticMinBlockPredictor struct {
	minBlockCount uint
}

func (s *staticMinBlockPredictor) PredictMinBlockCount(blockSize int64, _ uint64) (uint, error) {
	if blockSize <= 0 {
		return 0, fmt.Errorf("invalid block-size: %d", blockSize)
	}

	if s.minBlockCount == 0 {
		return 0, fmt.Errorf("static min-block count is not set")
	}

	return s.minBlockCount, nil
}
