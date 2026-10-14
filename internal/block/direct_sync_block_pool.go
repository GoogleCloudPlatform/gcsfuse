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
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sync/semaphore"
)

// BlockPoolInterface defines the common operations across GenBlockPool (Baseline & Variant A)
// and DirectSyncBlockPool (Variant B) for unified testing and benchmarking.
type BlockPoolInterface[T GenBlock] interface {
	Get() (T, error)
	TryGet() (T, error)
	Release(b T)
	ClearFreeBlockChannel(releaseReservedBlocks bool) error
	BlockSize() int64
	TotalFreeBlocks() int
}

var (
	directBlockPools         sync.Map // map[int64]*sync.Pool
	directPrefetchBlockPools sync.Map // map[int64]*sync.Pool
)

func getDirectBlockSyncPool(blockSize int64) *sync.Pool {
	if p, ok := directBlockPools.Load(blockSize); ok {
		return p.(*sync.Pool)
	}
	newPool := &sync.Pool{}
	actual, _ := directBlockPools.LoadOrStore(blockSize, newPool)
	return actual.(*sync.Pool)
}

func getDirectPrefetchBlockSyncPool(blockSize int64) *sync.Pool {
	if p, ok := directPrefetchBlockPools.Load(blockSize); ok {
		return p.(*sync.Pool)
	}
	newPool := &sync.Pool{}
	actual, _ := directPrefetchBlockPools.LoadOrStore(blockSize, newPool)
	return actual.(*sync.Pool)
}

// DirectSyncBlockPool implements Variant B: a direct sync.Pool-backed block pool
// governed by per-pool (maxBlocks, reservedBlocks) and global (globalMaxBlocksSem)
// concurrency limits without per-pool channel hoarding.
type DirectSyncBlockPool[T GenBlock] struct {
	blockSize          int64
	maxBlocks          int64
	reservedBlocks     int64
	globalMaxBlocksSem *semaphore.Weighted
	createBlockFunc    func(blockSize int64) (T, error)
	pool               *sync.Pool

	mu           sync.Mutex
	inUse        int64
	reservedHeld int64
	waiters      []chan struct{}
}

// NewGenDirectSyncBlockPool creates a DirectSyncBlockPool with an explicit createBlockFunc and optional *sync.Pool.
func NewGenDirectSyncBlockPool[T GenBlock](
	blockSize int64,
	maxBlocks int64,
	reservedBlocks int64,
	globalMaxBlocksSem *semaphore.Weighted,
	createBlockFunc func(blockSize int64) (T, error),
	sharedPool *sync.Pool,
) (*DirectSyncBlockPool[T], error) {
	if blockSize <= 0 || maxBlocks <= 0 {
		return nil, fmt.Errorf("invalid configuration provided for blockPool, blocksize: %d, maxBlocks: %d", blockSize, maxBlocks)
	}
	if reservedBlocks < 0 || reservedBlocks > maxBlocks {
		return nil, fmt.Errorf("invalid reserved blocks count: %d, it should be between 0 and maxBlocks: %d", reservedBlocks, maxBlocks)
	}
	if !globalMaxBlocksSem.TryAcquire(reservedBlocks) {
		return nil, CantAllocateAnyBlockError
	}
	if sharedPool == nil {
		sharedPool = &sync.Pool{}
	}
	return &DirectSyncBlockPool[T]{
		blockSize:          blockSize,
		maxBlocks:          maxBlocks,
		reservedBlocks:     reservedBlocks,
		globalMaxBlocksSem: globalMaxBlocksSem,
		createBlockFunc:    createBlockFunc,
		pool:               sharedPool,
		inUse:              0,
		reservedHeld:       reservedBlocks,
	}, nil
}

// NewDirectSyncBlockPool creates Variant B for block.Block using a shared per-blockSize sync.Pool.
func NewDirectSyncBlockPool(blockSize int64, maxBlocks int64, reservedBlocks int64, globalMaxBlocksSem *semaphore.Weighted) (*DirectSyncBlockPool[Block], error) {
	return NewGenDirectSyncBlockPool(blockSize, maxBlocks, reservedBlocks, globalMaxBlocksSem, createSyncPoolBlock, getDirectBlockSyncPool(blockSize))
}

// NewDirectSyncPrefetchBlockPool creates Variant B for block.PrefetchBlock using a shared per-blockSize sync.Pool.
func NewDirectSyncPrefetchBlockPool(blockSize int64, maxBlocks int64, reservedBlocks int64, globalMaxBlocksSem *semaphore.Weighted) (*DirectSyncBlockPool[PrefetchBlock], error) {
	return NewGenDirectSyncBlockPool(blockSize, maxBlocks, reservedBlocks, globalMaxBlocksSem, createSyncPoolPrefetchBlock, getDirectPrefetchBlockSyncPool(blockSize))
}

// TryGet attempts to acquire a block without blocking.
// Returns CantAllocateAnyBlockError if either the per-pool maxBlocks or globalMaxBlocksSem is exhausted.
func (bp *DirectSyncBlockPool[T]) TryGet() (T, error) {
	bp.mu.Lock()
	if bp.inUse >= bp.maxBlocks {
		bp.mu.Unlock()
		var zero T
		return zero, CantAllocateAnyBlockError
	}
	if bp.inUse >= bp.reservedHeld {
		if !bp.globalMaxBlocksSem.TryAcquire(1) {
			bp.mu.Unlock()
			var zero T
			return zero, CantAllocateAnyBlockError
		}
	}
	bp.inUse++
	bp.mu.Unlock()

	b, err := bp.acquireBlock()
	if err != nil {
		bp.releasePermit()
		var zero T
		return zero, err
	}
	return b, nil
}

// Get acquires a block, blocking if either the per-pool maxBlocks or globalMaxBlocksSem is currently exhausted.
func (bp *DirectSyncBlockPool[T]) Get() (T, error) {
	b, err := bp.TryGet()
	if err == nil {
		return b, nil
	}
	if !errors.Is(err, CantAllocateAnyBlockError) {
		return b, err
	}

	if err := bp.waitForPermit(); err != nil {
		var zero T
		return zero, err
	}

	b, err = bp.acquireBlock()
	if err != nil {
		bp.releasePermit()
		var zero T
		return zero, err
	}
	return b, nil
}

func (bp *DirectSyncBlockPool[T]) waitForPermit() error {
	bp.mu.Lock()
	if bp.inUse < bp.maxBlocks {
		if bp.inUse < bp.reservedHeld || bp.globalMaxBlocksSem.TryAcquire(1) {
			bp.inUse++
			bp.mu.Unlock()
			return nil
		}
	}

	w := make(chan struct{})
	bp.waiters = append(bp.waiters, w)
	canWaitGlobal := bp.inUse < bp.maxBlocks
	bp.mu.Unlock()

	if canWaitGlobal {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			if err := bp.globalMaxBlocksSem.Acquire(ctx, 1); err == nil {
				bp.mu.Lock()
				if bp.inUse < bp.maxBlocks && bp.removeWaiterLocked(w) {
					if bp.inUse < bp.reservedHeld {
						bp.globalMaxBlocksSem.Release(1)
					}
					bp.inUse++
					bp.mu.Unlock()
					close(w)
					return
				}
				bp.mu.Unlock()
				bp.globalMaxBlocksSem.Release(1)
			}
		}()
	}

	<-w
	return nil
}

func (bp *DirectSyncBlockPool[T]) removeWaiterLocked(target chan struct{}) bool {
	for i, w := range bp.waiters {
		if w == target {
			bp.waiters = append(bp.waiters[:i], bp.waiters[i+1:]...)
			return true
		}
	}
	return false
}

func (bp *DirectSyncBlockPool[T]) acquireBlock() (T, error) {
	if v := bp.pool.Get(); v != nil {
		b := v.(T)
		if blk, ok := any(b).(interface{ Cap() int64 }); !ok || blk.Cap() == bp.blockSize {
			b.Reuse()
			return b, nil
		}
	}
	b, err := bp.createBlockFunc(bp.blockSize)
	if err != nil {
		var zero T
		return zero, err
	}
	b.Reuse()
	return b, nil
}

// Release resets the block, returns it to the shared sync.Pool, and releases or transfers the permit.
func (bp *DirectSyncBlockPool[T]) Release(b T) {
	if blk, ok := any(b).(interface{ Cap() int64 }); ok && blk.Cap() == 0 {
		bp.releasePermit()
		return
	}
	b.Reuse()
	bp.pool.Put(b)
	bp.releasePermit()
}

func (bp *DirectSyncBlockPool[T]) releasePermit() {
	bp.mu.Lock()
	if len(bp.waiters) > 0 {
		w := bp.waiters[0]
		bp.waiters = bp.waiters[1:]
		bp.mu.Unlock()
		close(w)
		return
	}
	if bp.inUse <= 0 {
		bp.mu.Unlock()
		panic("DirectSyncBlockPool: Release called more times than Get/TryGet")
	}
	if bp.inUse > bp.reservedHeld {
		bp.globalMaxBlocksSem.Release(1)
	}
	bp.inUse--
	bp.mu.Unlock()
}

// BlockSize returns the configured block size of the pool.
func (bp *DirectSyncBlockPool[T]) BlockSize() int64 {
	return bp.blockSize
}

// ClearFreeBlockChannel releases idle reserved semaphore permits if releaseReservedBlocks is true.
// Non-reserved permits and free blocks are already returned to globalMaxBlocksSem and sync.Pool upon Release.
func (bp *DirectSyncBlockPool[T]) ClearFreeBlockChannel(releaseReservedBlocks bool) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	if releaseReservedBlocks && bp.reservedHeld > 0 {
		idleReserved := bp.reservedHeld - min(bp.inUse, bp.reservedHeld)
		if idleReserved > 0 {
			bp.globalMaxBlocksSem.Release(idleReserved)
		}
		bp.reservedHeld = 0
	}
	return nil
}

// TotalFreeBlocks returns the count of per-pool hoarded free blocks (always 0 for DirectSyncBlockPool).
func (bp *DirectSyncBlockPool[T]) TotalFreeBlocks() int {
	return 0
}
