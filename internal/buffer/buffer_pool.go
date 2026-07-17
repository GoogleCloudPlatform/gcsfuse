// Copyright 2026 Google LLC
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

package buffer

import (
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
)

// Pool defines an interface for on-demand buffer allocation and recycling.
type Pool interface {
	Get() []byte
	Put([]byte)
}

// readPoolBufferSize is the size of each buffer in readBufferPool (1 MiB).
const readPoolBufferSize = util.MiB

// Ensure FixedSizePool implements Pool at compile time.
var _ Pool = (*FixedSizePool)(nil)

type FixedSizePool struct {
	// Pool is the underlying sync.Pool storing fixed-size byte slices.
	Pool *sync.Pool
}

// NewFixedSizePool creates a new FixedSizePool with 1 MiB buffers.
func NewFixedSizePool() *FixedSizePool {
	return &FixedSizePool{
		Pool: &sync.Pool{
			New: func() any {
				return new([readPoolBufferSize]byte)
			},
		},
	}
}

func (bp *FixedSizePool) Get() []byte {
	return bp.Pool.Get().(*[readPoolBufferSize]byte)[:]
}

func (bp *FixedSizePool) Put(buf []byte) {
	if cap(buf) != readPoolBufferSize {
		logger.Errorf("FixedSizePool::Put Buffer capacity does not match readPoolBufferSize: %d vs %d", cap(buf), readPoolBufferSize)
		return
	}
	bp.Pool.Put((*[readPoolBufferSize]byte)(buf[:cap(buf)]))
}
