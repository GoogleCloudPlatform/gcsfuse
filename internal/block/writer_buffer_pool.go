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

var _ experimental.BufferPool = (*WriterBufferPool)(nil)

// WriterBufferPool is a per-writer mmap-backed implementation of
// experimental.BufferPool for gRPC parallel uploads.
//
// Each Go SDK storage.Writer gets its own WriterBufferPool while sharing the
// mount-wide globalByteBudget across active uploads. Concurrent Get, TryGet,
// and Put calls are safe without a mutex. Destroy must be called only after
// Writer.Close or Writer.Abort returns.
type WriterBufferPool struct {
	blockSizeBytes   int
	globalByteBudget *atomic.Int64
	freeBlocksCh     chan []byte
}

// NewWriterBufferPool creates a WriterBufferPool that caches up to
// maxBlocksPerFile free blocks (the per-writer concurrency bound enforced by
// the Go SDK) and tracks allocations against globalByteBudget.
func NewWriterBufferPool(blockSizeBytes int, maxBlocksPerFile int, globalByteBudget *atomic.Int64) (*WriterBufferPool, error) {
	if blockSizeBytes <= 0 || maxBlocksPerFile <= 0 {
		return nil, fmt.Errorf("invalid configuration for WriterBufferPool, blockSizeBytes: %d, maxBlocksPerFile: %d", blockSizeBytes, maxBlocksPerFile)
	}
	if globalByteBudget == nil {
		return nil, fmt.Errorf("invalid configuration for WriterBufferPool: globalByteBudget must not be nil")
	}
	return &WriterBufferPool{
		blockSizeBytes:   blockSizeBytes,
		globalByteBudget: globalByteBudget,
		freeBlocksCh:     make(chan []byte, maxBlocksPerFile),
	}, nil
}

// Get always returns a block (reusing a cached one or allocating via mmap),
// even if globalByteBudget goes negative, so a writer never deadlocks.
func (p *WriterBufferPool) Get(maxSize int) ([]byte, error) {
	if maxSize != p.blockSizeBytes {
		return nil, fmt.Errorf("WriterBufferPool.Get: requested maxSize %d does not match blockSizeBytes %d", maxSize, p.blockSizeBytes)
	}

	select {
	case buf := <-p.freeBlocksCh:
		return buf, nil
	default:
	}

	blockBytes := int64(p.blockSizeBytes)
	remaining := p.globalByteBudget.Add(-blockBytes)
	buf, err := mmapBytes(p.blockSizeBytes)
	if err != nil {
		p.globalByteBudget.Add(blockBytes)
		return nil, err
	}
	if remaining < 0 {
		logger.Tracef("WriterBufferPool.Get: globalByteBudget over-committed (%d bytes)", remaining)
	}
	return buf, nil
}

// TryGet reuses a cached block, or allocates via mmap only when at least
// blockSizeBytes remain in globalByteBudget, throttling concurrency when the
// global memory limit is reached.
func (p *WriterBufferPool) TryGet(maxSize int) ([]byte, bool, error) {
	if maxSize != p.blockSizeBytes {
		return nil, false, fmt.Errorf("WriterBufferPool.TryGet: requested maxSize %d does not match blockSizeBytes %d", maxSize, p.blockSizeBytes)
	}

	select {
	case buf := <-p.freeBlocksCh:
		return buf, true, nil
	default:
	}

	if !p.tryReserveBudget() {
		return nil, false, nil
	}

	buf, err := mmapBytes(p.blockSizeBytes)
	if err != nil {
		p.globalByteBudget.Add(int64(p.blockSizeBytes))
		return nil, false, err
	}
	return buf, true, nil
}

// Put caches buf in freeBlocksCh for reuse, or immediately unmaps it to reclaim
// memory when globalByteBudget is negative or freeBlocksCh is full. Buffers
// with unexpected capacity are logged and ignored.
func (p *WriterBufferPool) Put(buf []byte) {
	if cap(buf) != p.blockSizeBytes {
		logger.Warnf("WriterBufferPool.Put: ignoring buffer with unexpected capacity %d (expected %d)", cap(buf), p.blockSizeBytes)
		return
	}

	if p.globalByteBudget.Load() < 0 {
		p.unmapAndRelease(buf)
		return
	}

	buf = buf[:p.blockSizeBytes]
	select {
	case p.freeBlocksCh <- buf:
	default:
		logger.Tracef("WriterBufferPool.Put: freeBlocksCh full (capacity %d), unmapping block", cap(p.freeBlocksCh))
		p.unmapAndRelease(buf)
	}
}

// Destroy drains freeBlocksCh, unmaps all cached blocks, and returns their
// bytes to globalByteBudget. Writer.Close and Writer.Abort guarantee all
// in-flight blocks are returned via Put before returning, so no Put can arrive
// during or after Destroy. Safe to call multiple times.
func (p *WriterBufferPool) Destroy() {
	for {
		select {
		case buf := <-p.freeBlocksCh:
			p.unmapAndRelease(buf)
		default:
			return
		}
	}
}

func (p *WriterBufferPool) unmapAndRelease(buf []byte) {
	if err := syscall.Munmap(buf[:cap(buf)]); err != nil {
		logger.Errorf("WriterBufferPool: munmap error: %v", err)
		return
	}
	p.globalByteBudget.Add(int64(p.blockSizeBytes))
}

func (p *WriterBufferPool) tryReserveBudget() bool {
	blockBytes := int64(p.blockSizeBytes)
	for {
		cur := p.globalByteBudget.Load()
		if cur < blockBytes {
			return false
		}
		if p.globalByteBudget.CompareAndSwap(cur, cur-blockBytes) {
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
