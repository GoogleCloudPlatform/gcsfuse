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
	"fmt"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

type staticBufferInfo struct {
	slice  []byte
	isMmap bool
}

// BufferPool manages a pool of 1MB buffers.
// It keeps up to maxStatic buffers static (pre-allocated, preferably using mmap).
// It starts by pre-allocating initialStatic buffers, and allocates more on-demand
// up to maxStatic.
// If more buffers are needed concurrently, it falls back to a sync.Pool of Go-allocated buffers.
type BufferPool struct {
	staticBuffers chan []byte
	staticMap     sync.Map // map[uintptr]staticBufferInfo
	fallbackPool  sync.Pool
	bufferSize    int
	maxStatic     int
	allocated     int32 // atomic count of allocated static buffers
}

// NewBufferPool creates and initializes a BufferPool.
func NewBufferPool(maxStatic int, initialStatic int, bufferSize int) *BufferPool {
	if initialStatic > maxStatic {
		initialStatic = maxStatic
	}

	bp := &BufferPool{
		staticBuffers: make(chan []byte, maxStatic),
		bufferSize:    bufferSize,
		maxStatic:     maxStatic,
		fallbackPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, bufferSize)
			},
		},
	}

	for i := 0; i < initialStatic; i++ {
		buf, _ := bp.allocateAndRegisterStatic()
		bp.staticBuffers <- buf
	}
	bp.allocated = int32(initialStatic)

	return bp
}

// allocateAndRegisterStatic allocates a single static buffer (preferably via mmap)
// and registers it in the staticMap.
func (bp *BufferPool) allocateAndRegisterStatic() ([]byte, error) {
	var buf []byte
	var isMmap bool
	var err error

	buf, err = allocateMmap(bp.bufferSize)
	if err != nil {
		logger.Warnf("Failed to allocate static buffer via mmap: %v. Falling back to Go heap allocation.", err)
		buf = make([]byte, bp.bufferSize)
		isMmap = false
	} else {
		isMmap = true
	}

	bp.staticMap.Store(uintptr(unsafe.Pointer(&buf[0])), staticBufferInfo{
		slice:  buf,
		isMmap: isMmap,
	})
	return buf, nil
}

// Get retrieves a buffer of bufferSize from the pool.
// It attempts to get a static buffer first (either from the pre-allocated channel
// or by allocating a new one on-demand if the limit is not reached),
// falling back to the sync.Pool.
func (bp *BufferPool) Get() []byte {
	select {
	case buf := <-bp.staticBuffers:
		return buf
	default:
		// Try to allocate a new static buffer on-demand if we haven't reached maxStatic.
		for {
			curr := atomic.LoadInt32(&bp.allocated)
			if curr >= int32(bp.maxStatic) {
				break
			}
			if atomic.CompareAndSwapInt32(&bp.allocated, curr, curr+1) {
				buf, _ := bp.allocateAndRegisterStatic()
				return buf
			}
		}

		// Fallback to sync.Pool.
		return bp.fallbackPool.Get().([]byte)
	}
}

// Put returns a buffer to the pool.
// It routes the buffer back to the static pool if it was one of the pre-allocated static buffers,
// otherwise it returns it to the fallback sync.Pool.
func (bp *BufferPool) Put(buf []byte) {
	if len(buf) == 0 {
		return
	}

	// Restore the slice to its full capacity before checking and returning.
	buf = buf[:cap(buf)]
	ptr := uintptr(unsafe.Pointer(&buf[0]))

	if val, ok := bp.staticMap.Load(ptr); ok {
		info := val.(staticBufferInfo)
		bp.staticBuffers <- info.slice
	} else {
		bp.fallbackPool.Put(buf)
	}
}

// Close deallocates all static buffers that were allocated via mmap.
func (bp *BufferPool) Close() error {
	var errs []error
	bp.staticMap.Range(func(key, value interface{}) bool {
		info := value.(staticBufferInfo)
		if info.isMmap {
			if err := deallocateMmap(info.slice); err != nil {
				errs = append(errs, err)
			}
		}
		return true
	})
	if len(errs) > 0 {
		return fmt.Errorf("errors closing buffer pool: %v", errs)
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
