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

package gcsx

import (
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
)

// BufferPool defines an interface for on-demand buffer allocation and recycling.
type BufferPool interface {
	Get() []byte
	Put([]byte)
}

// readPoolBufferSize is the size of each buffer in readBufferPool (1 MiB).
const readPoolBufferSize = util.MiB

// Ensure FixedSizeBufferPool implements BufferPool at compile time.
var _ BufferPool = (*FixedSizeBufferPool)(nil)

type FixedSizeBufferPool struct {
	// Pool is the underlying sync.Pool storing fixed-size byte slices.
	Pool *sync.Pool
}

// NewFixedSizeBufferPool creates a new FixedSizeBufferPool with 1 MiB buffers.
func NewFixedSizeBufferPool() *FixedSizeBufferPool {
	return &FixedSizeBufferPool{
		Pool: &sync.Pool{
			New: func() any {
				return new([readPoolBufferSize]byte)
			},
		},
	}
}

func (bp *FixedSizeBufferPool) Get() []byte {
	return bp.Pool.Get().(*[readPoolBufferSize]byte)[:]
}

func (bp *FixedSizeBufferPool) Put(buf []byte) {
	if cap(buf) != readPoolBufferSize {
		logger.Errorf("FixedSizeBufferPool::Put Buffer capacity does not match readPoolBufferSize: %d vs %d", cap(buf), readPoolBufferSize)
		return
	}
	bp.Pool.Put((*[readPoolBufferSize]byte)(buf[:cap(buf)]))
}
