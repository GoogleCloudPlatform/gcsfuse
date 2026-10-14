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
	"sync"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

// BlockPool is a type alias for GenBlockPool[Block].
type BlockPool = GenBlockPool[Block]

// PrefetchBlockPool is a type alias for GenBlockPool[PrefetchBlock].
type PrefetchBlockPool = GenBlockPool[PrefetchBlock]

// maxSyncPoolBlockSize guards against unallocatable test sizes (e.g. 1 TiB in
// OOM unit tests) so Go's runtime returns an error rather than crashing with fatal OOM.
const maxSyncPoolBlockSize int64 = 1 << 34 // 16 GiB

// slicePools maps blockSize (int64) -> *sync.Pool (storing *[]byte of cap == blockSize).
// Storing *[]byte avoids the 24-byte slice-header heap allocation when boxing into any on Put.
var slicePools sync.Map

// getSlicePool returns the shared *sync.Pool of *[]byte slices for the given blockSize.
func getSlicePool(blockSize int64) *sync.Pool {
	if p, ok := slicePools.Load(blockSize); ok {
		return p.(*sync.Pool)
	}
	newPool := &sync.Pool{
		New: func() any {
			s := make([]byte, 0, int(blockSize))
			return &s
		},
	}
	actual, _ := slicePools.LoadOrStore(blockSize, newPool)
	return actual.(*sync.Pool)
}

// ResetSyncPools clears all shared slice and direct-block sync.Pools.
// This is useful between benchmark regimes to ensure a clean baseline.
func ResetSyncPools() {
	slicePools.Range(func(key, _ any) bool {
		slicePools.Delete(key)
		return true
	})
	directBlockPools.Range(func(key, _ any) bool {
		directBlockPools.Delete(key)
		return true
	})
	directPrefetchBlockPools.Range(func(key, _ any) bool {
		directPrefetchBlockPools.Delete(key)
		return true
	})
}

// syncPoolBlock implements Block using a sync.Pool-managed []byte slice instead of mmap.
type syncPoolBlock struct {
	memoryBlock
	slicePtr *[]byte
	pool     *sync.Pool
}

func (b *syncPoolBlock) Deallocate() error {
	if b.buffer == nil {
		return fmt.Errorf("invalid buffer")
	}
	if b.slicePtr != nil && b.pool != nil {
		*b.slicePtr = b.buffer[:0]
		b.pool.Put(b.slicePtr)
		b.slicePtr = nil
	}
	b.buffer = nil
	b.readSeek = 0
	return nil
}

// createSyncPoolBlock creates a Block backed by the shared per-blockSize sync.Pool.
func createSyncPoolBlock(blockSize int64) (Block, error) {
	return createSyncPoolBlockWithPool(blockSize, getSlicePool(blockSize))
}

func createSyncPoolBlockWithPool(blockSize int64, pool *sync.Pool) (Block, error) {
	if blockSize <= 0 || blockSize > maxSyncPoolBlockSize {
		return nil, fmt.Errorf("slice allocation error: cannot allocate memory")
	}
	var slicePtr *[]byte
	if v := pool.Get(); v != nil {
		slicePtr = v.(*[]byte)
	}
	if slicePtr == nil || int64(cap(*slicePtr)) != blockSize {
		s := make([]byte, 0, int(blockSize))
		slicePtr = &s
	}
	buf := (*slicePtr)[:0]
	return &syncPoolBlock{
		memoryBlock: memoryBlock{buffer: buf},
		slicePtr:    slicePtr,
		pool:        pool,
	}, nil
}

// syncPoolPrefetchBlock implements PrefetchBlock using a sync.Pool-managed []byte slice.
type syncPoolPrefetchBlock struct {
	prefetchMemoryBlock
	slicePtr *[]byte
	pool     *sync.Pool
}

func (pmb *syncPoolPrefetchBlock) Deallocate() error {
	if pmb.buffer == nil {
		return fmt.Errorf("invalid buffer")
	}
	if pmb.slicePtr != nil && pmb.pool != nil {
		*pmb.slicePtr = pmb.buffer[:0]
		pmb.pool.Put(pmb.slicePtr)
		pmb.slicePtr = nil
	}
	pmb.buffer = nil
	pmb.readSeek = 0
	return nil
}

// createSyncPoolPrefetchBlock creates a PrefetchBlock backed by the shared per-blockSize sync.Pool.
func createSyncPoolPrefetchBlock(blockSize int64) (PrefetchBlock, error) {
	return createSyncPoolPrefetchBlockWithPool(blockSize, getSlicePool(blockSize))
}

func createSyncPoolPrefetchBlockWithPool(blockSize int64, pool *sync.Pool) (PrefetchBlock, error) {
	if blockSize <= 0 || blockSize > maxSyncPoolBlockSize {
		return nil, fmt.Errorf("createSyncPoolPrefetchBlock: cannot allocate memory")
	}
	var slicePtr *[]byte
	if v := pool.Get(); v != nil {
		slicePtr = v.(*[]byte)
	}
	if slicePtr == nil || int64(cap(*slicePtr)) != blockSize {
		s := make([]byte, 0, int(blockSize))
		slicePtr = &s
	}
	buf := (*slicePtr)[:0]
	return &syncPoolPrefetchBlock{
		prefetchMemoryBlock: prefetchMemoryBlock{
			memoryBlock:  memoryBlock{buffer: buf},
			status:       BlockStatus{State: BlockStateInProgress},
			notification: make(chan BlockStatus, 1),
			absStartOff:  -1,
			refCount:     atomic.Int32{},
		},
		slicePtr: slicePtr,
		pool:     pool,
	}, nil
}

// NewSyncPoolBlockPool creates Variant A (Hybrid GenBlockPool + sync.Pool backing slices) for Block.
func NewSyncPoolBlockPool(blockSize int64, maxBlocks int64, reservedBlocks int64, globalMaxBlocksSem *semaphore.Weighted) (*BlockPool, error) {
	return NewGenBlockPool(blockSize, maxBlocks, reservedBlocks, globalMaxBlocksSem, createSyncPoolBlock)
}

// NewSyncPoolPrefetchBlockPool creates Variant A (Hybrid GenBlockPool + sync.Pool backing slices) for PrefetchBlock.
func NewSyncPoolPrefetchBlockPool(blockSize int64, maxBlocks int64, reservedBlocks int64, globalMaxBlocksSem *semaphore.Weighted) (*PrefetchBlockPool, error) {
	return NewGenBlockPool(blockSize, maxBlocks, reservedBlocks, globalMaxBlocksSem, createSyncPoolPrefetchBlock)
}
