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

package block

import (
	"fmt"
	"sync/atomic"
	"syscall"

	"cloud.google.com/go/storage/experimental"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/logger"
)

// Compile-time check that *WriterBufferPool implements experimental.BufferPool.
var _ experimental.BufferPool = (*WriterBufferPool)(nil)

// WriterBufferPool is a per-writer mmap-backed implementation of the Go Storage
// SDK's experimental.BufferPool interface for gRPC parallel uploads.
//
// Each parallel storage.Writer receives its own WriterBufferPool with a bounded
// free-block channel (sized to maxBlocksPerFile) while sharing a mount-wide
// atomic globalByteBudget across writers. Because freeBlocksCh is a buffered
// channel and globalByteBudget is an atomic.Int64, concurrent Get, TryGet, and
// Put calls across Go SDK part-upload goroutines require no explicit mutex.
//
// No destroyed flag is needed because each pool is bound 1:1 to a single
// storage.Writer, and Destroy is called only after Writer.Close or Writer.Abort
// blocks until all background upload and hedge goroutines exit and finish
// calling Put, after which Get, TryGet, and Put are never called again.
type WriterBufferPool struct {
	blockSize        int
	globalByteBudget *atomic.Int64 // mount-wide byte budget (initialized from write.global-mem-limit-mb)
	freeBlocksCh     chan []byte   // buffered channel (capacity maxBlocksPerFile) of reusable mmap blocks
}

// NewWriterBufferPool creates a per-writer buffer pool with fixed blockSize
// bytes per block, caching up to maxBlocksPerFile blocks locally and tracking
// allocations in bytes against globalByteBudget.
func NewWriterBufferPool(blockSize int, maxBlocksPerFile int, globalByteBudget *atomic.Int64) (*WriterBufferPool, error) {
	if blockSize <= 0 || maxBlocksPerFile <= 0 {
		return nil, fmt.Errorf("invalid configuration for WriterBufferPool, blockSize: %d, maxBlocksPerFile: %d", blockSize, maxBlocksPerFile)
	}
	if globalByteBudget == nil {
		return nil, fmt.Errorf("invalid configuration for WriterBufferPool: globalByteBudget must not be nil")
	}
	return &WriterBufferPool{
		blockSize:        blockSize,
		globalByteBudget: globalByteBudget,
		freeBlocksCh:     make(chan []byte, maxBlocksPerFile),
	}, nil
}

// Get returns a blockSize buffer when no part upload is currently in flight to
// guarantee forward progress. It reuses a cached block from freeBlocksCh if
// available, or decrements globalByteBudget by blockSize (even below 0) and
// allocates a new block via mmap.
func (p *WriterBufferPool) Get(maxSize int) ([]byte, error) {
	if maxSize != p.blockSize {
		return nil, fmt.Errorf("WriterBufferPool.Get: requested maxSize %d does not match blockSize %d", maxSize, p.blockSize)
	}

	select {
	case buf := <-p.freeBlocksCh:
		return buf, nil
	default:
	}

	blockBytes := int64(p.blockSize)
	p.globalByteBudget.Add(-blockBytes)
	buf, err := mmapBytes(p.blockSize)
	if err != nil {
		p.globalByteBudget.Add(blockBytes)
		return nil, err
	}
	return buf, nil
}

// TryGet performs a non-blocking buffer acquisition when a part upload is
// already in flight. It reuses a cached block from freeBlocksCh if available,
// or atomically reserves blockSize bytes from globalByteBudget when sufficient
// budget remains (globalByteBudget >= blockSize) and allocates via mmap. When
// globalByteBudget < blockSize and no cached block is available, it returns
// (nil, false, nil).
func (p *WriterBufferPool) TryGet(maxSize int) ([]byte, bool, error) {
	if maxSize != p.blockSize {
		return nil, false, fmt.Errorf("WriterBufferPool.TryGet: requested maxSize %d does not match blockSize %d", maxSize, p.blockSize)
	}

	select {
	case buf := <-p.freeBlocksCh:
		return buf, true, nil
	default:
	}

	blockBytes := int64(p.blockSize)
	if !tryReserveBytes(p.globalByteBudget, blockBytes) {
		return nil, false, nil
	}

	buf, err := mmapBytes(p.blockSize)
	if err != nil {
		p.globalByteBudget.Add(blockBytes)
		return nil, false, err
	}
	return buf, true, nil
}

// Put returns a buffer previously obtained from Get or TryGet after a part
// upload finishes, fails, or aborts. Buffers whose capacity does not match
// blockSize are logged and ignored. Over-budget blocks (globalByteBudget < 0)
// or blocks exceeding freeBlocksCh capacity are unmapped immediately; otherwise
// buf is resliced to full blockSize length (without zeroing memory) and cached
// in freeBlocksCh for reuse within the same file.
func (p *WriterBufferPool) Put(buf []byte) {
	if cap(buf) != p.blockSize {
		logger.Warnf("WriterBufferPool.Put: ignoring buffer with unexpected capacity %d (expected %d)", cap(buf), p.blockSize)
		return
	}

	if p.globalByteBudget.Load() < 0 {
		p.unmapAndRelease(buf)
		return
	}

	buf = buf[:p.blockSize]
	select {
	case p.freeBlocksCh <- buf:
	default:
		p.unmapAndRelease(buf)
	}
}

// Destroy drains freeBlocksCh, unmaps all cached blocks, and returns their
// total bytes to globalByteBudget. Callers must invoke Destroy only after
// Writer.Close or Writer.Abort has returned (once all Go SDK part workers have
// finished calling Put). Destroy is idempotent and never closes freeBlocksCh.
func (p *WriterBufferPool) Destroy() {
	var freedBlocks int64
	for {
		select {
		case buf := <-p.freeBlocksCh:
			if err := munmapBytes(buf); err != nil {
				logger.Errorf("WriterBufferPool.Destroy: %v", err)
				continue
			}
			freedBlocks++
		default:
			if freedBlocks > 0 {
				p.globalByteBudget.Add(freedBlocks * int64(p.blockSize))
			}
			return
		}
	}
}

func (p *WriterBufferPool) unmapAndRelease(buf []byte) {
	if err := munmapBytes(buf); err != nil {
		logger.Errorf("WriterBufferPool: %v", err)
		return
	}
	p.globalByteBudget.Add(int64(p.blockSize))
}

func tryReserveBytes(budget *atomic.Int64, bytes int64) bool {
	for {
		cur := budget.Load()
		if cur < bytes {
			return false
		}
		if budget.CompareAndSwap(cur, cur-bytes) {
			return true
		}
	}
}

func mmapBytes(size int) ([]byte, error) {
	prot, flags := syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE
	buf, err := syscall.Mmap(-1, 0, size, prot, flags)
	if err != nil {
		return nil, fmt.Errorf("mmap error: %w", err)
	}
	return buf, nil
}

func munmapBytes(buf []byte) error {
	if err := syscall.Munmap(buf[:cap(buf)]); err != nil {
		return fmt.Errorf("munmap error: %w", err)
	}
	return nil
}
