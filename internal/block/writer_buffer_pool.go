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
// Each parallel storage.Writer gets its own freeBlocksCh while sharing the
// mount-wide globalByteBudget. Concurrent Get, TryGet, and Put calls are safe
// without a mutex, and Destroy must be called only after Writer.Close or
// Writer.Abort returns (once all background upload goroutines have finished Put).
type WriterBufferPool struct {
	blockSizeBytes   int
	globalByteBudget *atomic.Int64
	freeBlocksCh     chan []byte
}

// NewWriterBufferPool creates a WriterBufferPool with maxBlocksPerFile local
// capacity backed by globalByteBudget.
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

// Get returns a cached block or allocates a new mmap block, allowing
// globalByteBudget to go negative to guarantee forward progress.
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
	p.globalByteBudget.Add(-blockBytes)
	buf, err := mmapBytes(p.blockSizeBytes)
	if err != nil {
		p.globalByteBudget.Add(blockBytes)
		return nil, err
	}
	return buf, nil
}

// TryGet returns a cached block, or allocates a new mmap block only when at
// least blockSizeBytes remain in globalByteBudget.
func (p *WriterBufferPool) TryGet(maxSize int) ([]byte, bool, error) {
	if maxSize != p.blockSizeBytes {
		return nil, false, fmt.Errorf("WriterBufferPool.TryGet: requested maxSize %d does not match blockSizeBytes %d", maxSize, p.blockSizeBytes)
	}

	select {
	case buf := <-p.freeBlocksCh:
		return buf, true, nil
	default:
	}

	blockBytes := int64(p.blockSizeBytes)
	if !tryReserveBytes(p.globalByteBudget, blockBytes) {
		return nil, false, nil
	}

	buf, err := mmapBytes(p.blockSizeBytes)
	if err != nil {
		p.globalByteBudget.Add(blockBytes)
		return nil, false, err
	}
	return buf, true, nil
}

// Put caches buf for reuse within the writer, or unmaps it immediately when
// globalByteBudget is negative or freeBlocksCh is full.
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
		p.unmapAndRelease(buf)
	}
}

// Destroy drains and unmaps all cached blocks, returning their bytes to
// globalByteBudget. Safe to call multiple times after Close or Abort returns.
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
				p.globalByteBudget.Add(freedBlocks * int64(p.blockSizeBytes))
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
	p.globalByteBudget.Add(int64(p.blockSizeBytes))
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
