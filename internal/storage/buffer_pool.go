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

package storage

import (
	"sync"
	"unsafe"

	"github.com/langhuihui/gomem"
)

type gomemBufferPool struct {
	mu        sync.Mutex
	allocator *gomem.ScalableMemoryAllocator
	slicePool sync.Pool
	sizes     map[uintptr]int
}

func newGomemBufferPool(initialSize int) *gomemBufferPool {
	return &gomemBufferPool{
		allocator: gomem.NewScalableMemoryAllocator(initialSize),
		slicePool: sync.Pool{
			New: func() any {
				return new([]byte)
			},
		},
		sizes: make(map[uintptr]int),
	}
}

// Get returns a buffer with specified length from the pool.
func (p *gomemBufferPool) Get(length int) *[]byte {
	if length <= 0 {
		buf := make([]byte, 0)
		bufPtr := p.slicePool.Get().(*[]byte)
		*bufPtr = buf
		return bufPtr
	}

	p.mu.Lock()
	buf := p.allocator.Malloc(length)
	if len(buf) > 0 {
		ptr := uintptr(unsafe.Pointer(&buf[0]))
		p.sizes[ptr] = length
	}
	p.mu.Unlock()

	bufPtr := p.slicePool.Get().(*[]byte)
	*bufPtr = buf
	return bufPtr
}

// Put returns a buffer to the pool.
func (p *gomemBufferPool) Put(bufPtr *[]byte) {
	if bufPtr == nil {
		return
	}
	buf := *bufPtr
	if len(buf) > 0 {
		ptr := uintptr(unsafe.Pointer(&buf[0]))
		p.mu.Lock()
		if length, found := p.sizes[ptr]; found {
			delete(p.sizes, ptr)
			originalSlice := buf[:length]
			p.allocator.Free(originalSlice)
		}
		p.mu.Unlock()
	}
	*bufPtr = nil
	p.slicePool.Put(bufPtr)
}
