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

package fs

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

// BufferPool manages a pool of bufferSize buffers.
// It allocates static buffers via mmap on-demand up to maxStatic.
// If mmap allocation fails or more buffers are needed concurrently, it falls back
// to a sync.Pool of Go-allocated buffers.
type BufferPool struct {
	mu            sync.Mutex
	staticBuffers [][]byte // LIFO stack of static buffers
	staticMap     sync.Map // map[uintptr][]byte for routing
	fallbackPool  sync.Pool
	bufferSize    int
	maxStatic     int
	allocated     int32 // atomic count of allocated static buffers
}

// NewBufferPool creates and initializes a BufferPool.
func NewBufferPool(maxStatic int, bufferSize int) *BufferPool {
	return &BufferPool{
		staticBuffers: make([][]byte, 0, maxStatic),
		bufferSize:    bufferSize,
		maxStatic:     maxStatic,
		fallbackPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, bufferSize)
			},
		},
	}
}

// allocateAndRegisterStatic allocates a single static buffer via mmap
// and registers it in the staticMap.
func (bp *BufferPool) allocateAndRegisterStatic() ([]byte, error) {
	buf, err := allocateMmap(bp.bufferSize)
	if err != nil {
		return nil, err
	}

	bp.staticMap.Store(uintptr(unsafe.Pointer(&buf[0])), buf)
	return buf, nil
}

// Get retrieves a buffer of bufferSize from the pool.
// It attempts to get a static buffer first (either from the pre-allocated stack
// or by allocating a new one on-demand if the limit is not reached),
// falling back to the sync.Pool.
func (bp *BufferPool) Get() []byte {
	if buf := bp.pop(); buf != nil {
		return buf
	}

	// Try to allocate a new static buffer on-demand if we haven't reached maxStatic.
	if bp.tryAllocate() {
		buf, err := bp.allocateAndRegisterStatic()
		if err == nil {
			return buf
		}
		logger.Warnf("Failed to allocate static buffer via mmap: %v. Falling back to sync.Pool.", err)
		atomic.AddInt32(&bp.allocated, -1)
	}

	// Fallback to sync.Pool.
	return bp.fallbackPool.Get().([]byte)
}

// Put returns a buffer to the pool.
// It routes the buffer back to the static pool if it was one of the mmap'ed static buffers,
// otherwise it returns it to the fallback sync.Pool.
func (bp *BufferPool) Put(buf []byte) {
	if len(buf) == 0 {
		return
	}

	// Restore the slice to its full capacity before checking and returning.
	buf = buf[:cap(buf)]
	ptr := uintptr(unsafe.Pointer(&buf[0]))

	if val, ok := bp.staticMap.Load(ptr); ok {
		bp.push(val.([]byte))
	} else {
		bp.fallbackPool.Put(buf)
	}
}

// pop retrieves a buffer from the static stack in a thread-safe manner.
func (bp *BufferPool) pop() []byte {
	bp.mu.Lock()
	n := len(bp.staticBuffers)
	if n == 0 {
		bp.mu.Unlock()
		return nil
	}
	buf := bp.staticBuffers[n-1]
	bp.staticBuffers = bp.staticBuffers[:n-1]
	bp.mu.Unlock()
	return buf
}

// push adds a buffer back to the static stack in a thread-safe manner.
func (bp *BufferPool) push(buf []byte) {
	bp.mu.Lock()
	bp.staticBuffers = append(bp.staticBuffers, buf)
	bp.mu.Unlock()
}

// tryAllocate atomically attempts to reserve a slot for a new static buffer.
// It returns true if a slot was successfully reserved, or false if the limit has been reached.
func (bp *BufferPool) tryAllocate() bool {
	for {
		curr := atomic.LoadInt32(&bp.allocated)
		if curr >= int32(bp.maxStatic) {
			return false
		}
		if atomic.CompareAndSwapInt32(&bp.allocated, curr, curr+1) {
			return true
		}
	}
}

// Close deallocates all static buffers that were allocated via mmap.
func (bp *BufferPool) Close() error {
	var errs []error
	bp.staticMap.Range(func(key, value interface{}) bool {
		buf := value.([]byte)
		if err := deallocateMmap(buf); err != nil {
			errs = append(errs, err)
		}
		return true
	})
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("errors closing buffer pool: %w", err)
	}
	return nil
}

func allocateMmap(size int) ([]byte, error) {
	prot := syscall.PROT_READ | syscall.PROT_WRITE
	flags := syscall.MAP_ANON | syscall.MAP_PRIVATE
	addr, err := syscall.Mmap(-1, 0, size, prot, flags)
	if err != nil {
		return nil, err
	}
	return addr, nil
}

func deallocateMmap(buf []byte) error {
	if buf == nil {
		return nil
	}
	return syscall.Munmap(buf)
}
