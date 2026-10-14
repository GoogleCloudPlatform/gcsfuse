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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

func TestSyncPoolBlock_LifecycleAndOperations(t *testing.T) {
	ResetSyncPools()
	b, err := createSyncPoolBlock(16)
	require.NoError(t, err)
	require.Equal(t, int64(0), b.Size())
	require.Equal(t, int64(16), b.Cap())

	// Write & Read
	n, err := b.Write([]byte("hello "))
	require.NoError(t, err)
	assert.Equal(t, 6, n)
	n, err = b.Write([]byte("world"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, int64(11), b.Size())
	assert.Equal(t, int64(16), b.Cap())

	// Write beyond capacity
	n, err = b.Write([]byte("123456"))
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.EqualError(t, err, outOfCapacityError)

	// Seek and Read
	off, err := b.Seek(6, io.SeekStart)
	require.NoError(t, err)
	assert.Equal(t, int64(6), off)
	buf := make([]byte, 5)
	n, err = b.Read(buf)
	require.True(t, err == nil || errors.Is(err, io.EOF))
	assert.Equal(t, 5, n)
	assert.Equal(t, "world", string(buf))

	// Reuse resets length and readSeek while preserving capacity
	b.Reuse()
	assert.Equal(t, int64(0), b.Size())
	assert.Equal(t, int64(16), b.Cap())
	out, err := io.ReadAll(b)
	require.NoError(t, err)
	assert.Empty(t, out)

	// Deallocate returns slice to pool and nil-outs buffer; second Deallocate errors
	require.NoError(t, b.Deallocate())
	assert.Equal(t, int64(0), b.Cap())
	err = b.Deallocate()
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid buffer")
}

func TestSyncPoolPrefetchBlock_LifecycleAndOperations(t *testing.T) {
	ResetSyncPools()
	pmb, err := createSyncPoolPrefetchBlock(16)
	require.NoError(t, err)
	require.Equal(t, int64(0), pmb.Size())
	require.Equal(t, int64(16), pmb.Cap())

	// ReadFrom
	readerData := []byte("0123456789abcdef")
	written, err := pmb.(io.ReaderFrom).ReadFrom(bytes.NewReader(readerData))
	require.NoError(t, err)
	assert.Equal(t, int64(16), written)
	assert.Equal(t, int64(16), pmb.Size())

	// ReadAt and ReadAtSlice
	part := make([]byte, 6)
	n, err := pmb.ReadAt(part, 10)
	require.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, []byte("abcdef"), part)

	slice, err := pmb.ReadAtSlice(0, 6)
	require.NoError(t, err)
	assert.Equal(t, []byte("012345"), slice)

	// AbsStartOff / SetAbsStartOff
	assert.Panics(t, func() { _ = pmb.AbsStartOff() })
	require.Error(t, pmb.SetAbsStartOff(-1))
	require.NoError(t, pmb.SetAbsStartOff(1024))
	assert.Equal(t, int64(1024), pmb.AbsStartOff())
	require.Error(t, pmb.SetAbsStartOff(2048))

	// RefCount
	assert.Equal(t, int32(0), pmb.RefCount())
	pmb.IncRef()
	pmb.IncRef()
	assert.Equal(t, int32(2), pmb.RefCount())
	assert.False(t, pmb.DecRef())
	assert.True(t, pmb.DecRef())
	assert.Equal(t, int32(0), pmb.RefCount())

	// NotifyReady & AwaitReady
	pmb.NotifyReady(BlockStatus{State: BlockStateDownloaded})
	st, err := pmb.AwaitReady(context.Background())
	require.NoError(t, err)
	assert.Equal(t, BlockStateDownloaded, st.State)

	// Reuse resets state
	pmb.Reuse()
	assert.Equal(t, int64(0), pmb.Size())
	assert.Equal(t, int64(16), pmb.Cap())
	assert.Equal(t, int32(0), pmb.RefCount())
	assert.Panics(t, func() { _ = pmb.AbsStartOff() })

	// Deallocate & double Deallocate
	require.NoError(t, pmb.Deallocate())
	err = pmb.Deallocate()
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid buffer")
}

type poolFactory struct {
	name    string
	newPool func(blockSize, maxBlocks, reservedBlocks int64, globalSem *semaphore.Weighted) (BlockPoolInterface[Block], error)
}

func getPoolFactories() []poolFactory {
	return []poolFactory{
		{
			name: "VariantA_SyncPoolBlockPool",
			newPool: func(blockSize, maxBlocks, reservedBlocks int64, globalSem *semaphore.Weighted) (BlockPoolInterface[Block], error) {
				return NewSyncPoolBlockPool(blockSize, maxBlocks, reservedBlocks, globalSem)
			},
		},
		{
			name: "VariantB_DirectSyncBlockPool",
			newPool: func(blockSize, maxBlocks, reservedBlocks int64, globalSem *semaphore.Weighted) (BlockPoolInterface[Block], error) {
				return NewDirectSyncBlockPool(blockSize, maxBlocks, reservedBlocks, globalSem)
			},
		},
	}
}

func TestSyncPoolVariants_ConfigValidation(t *testing.T) {
	for _, vf := range getPoolFactories() {
		t.Run(vf.name, func(t *testing.T) {
			_, err := vf.newPool(0, 10, 0, semaphore.NewWeighted(10))
			require.EqualError(t, err, fmt.Sprintf(invalidConfigError, 0, 10))

			_, err = vf.newPool(-5, 10, 0, semaphore.NewWeighted(10))
			require.EqualError(t, err, fmt.Sprintf(invalidConfigError, -5, 10))

			_, err = vf.newPool(1024, 0, 0, semaphore.NewWeighted(10))
			require.EqualError(t, err, fmt.Sprintf(invalidConfigError, 1024, 0))

			_, err = vf.newPool(1024, -2, 0, semaphore.NewWeighted(10))
			require.EqualError(t, err, fmt.Sprintf(invalidConfigError, 1024, -2))

			_, err = vf.newPool(1024, 10, -1, semaphore.NewWeighted(10))
			require.EqualError(t, err, "invalid reserved blocks count: -1, it should be between 0 and maxBlocks: 10")

			_, err = vf.newPool(1024, 10, 11, semaphore.NewWeighted(20))
			require.EqualError(t, err, "invalid reserved blocks count: 11, it should be between 0 and maxBlocks: 10")

			_, err = vf.newPool(1024, 10, 5, semaphore.NewWeighted(4))
			require.ErrorIs(t, err, CantAllocateAnyBlockError)
		})
	}
}

func TestSyncPoolVariants_ReservedAndGlobalSemaphoreEnforcement(t *testing.T) {
	for _, vf := range getPoolFactories() {
		t.Run(vf.name, func(t *testing.T) {
			ResetSyncPools()

			// Test ClearFreeBlockChannel(true) and ClearFreeBlockChannel(false) across reservedBlocks = 0..4
			for _, reserved := range []int64{0, 1, 2, 4} {
				semTrue := semaphore.NewWeighted(4)
				bp, err := vf.newPool(1024, 4, reserved, semTrue)
				require.NoError(t, err)
				assert.Equal(t, int64(1024), bp.BlockSize())

				blocks := make([]Block, 4)
				for i := range 4 {
					blk, err := bp.TryGet()
					require.NoError(t, err)
					_, err = blk.Write([]byte("payload"))
					require.NoError(t, err)
					blocks[i] = blk
				}

				// 5th TryGet must fail with CantAllocateAnyBlockError
				_, err = bp.TryGet()
				require.ErrorIs(t, err, CantAllocateAnyBlockError)

				for i := range 4 {
					bp.Release(blocks[i])
				}

				// ClearFreeBlockChannel(false) retains reserved permits
				require.NoError(t, bp.ClearFreeBlockChannel(false))
				nonReserved := 4 - reserved
				if nonReserved > 0 {
					require.True(t, semTrue.TryAcquire(nonReserved))
					semTrue.Release(nonReserved)
				}
				if reserved > 0 {
					require.False(t, semTrue.TryAcquire(4))
				}

				// ClearFreeBlockChannel(true) releases all 4 permits
				require.NoError(t, bp.ClearFreeBlockChannel(true))
				require.True(t, semTrue.TryAcquire(4))
				require.False(t, semTrue.TryAcquire(1))
				semTrue.Release(4)
			}
		})
	}
}

func TestSyncPoolVariants_BlockingGetAndUnblock(t *testing.T) {
	for _, vf := range getPoolFactories() {
		t.Run(vf.name, func(t *testing.T) {
			ResetSyncPools()

			// Case 1: Blocked on local maxBlocks limit, unblocked by local Release()
			globalSem := semaphore.NewWeighted(10)
			bp, err := vf.newPool(1024, 2, 1, globalSem)
			require.NoError(t, err)

			b1, err := bp.Get()
			require.NoError(t, err)
			b2, err := bp.Get()
			require.NoError(t, err)

			gotCh := make(chan Block, 1)
			go func() {
				b3, getErr := bp.Get()
				require.NoError(t, getErr)
				gotCh <- b3
			}()

			select {
			case <-gotCh:
				t.Fatal("Get() should have blocked when maxBlocks is reached")
			case <-time.After(30 * time.Millisecond):
			}

			bp.Release(b1)
			select {
			case b3 := <-gotCh:
				require.NotNil(t, b3)
				assert.Equal(t, int64(0), b3.Size())
				bp.Release(b3)
			case <-time.After(1 * time.Second):
				t.Fatal("Get() did not unblock after local Release()")
			}
			bp.Release(b2)
			require.NoError(t, bp.ClearFreeBlockChannel(true))

			// Case 2: Blocked on globalMaxBlocksSem (with 0 blocks & with >0 blocks), unblocked by another pool releasing
			sharedSem := semaphore.NewWeighted(2)
			pool1, err := vf.newPool(1024, 2, 0, sharedSem)
			require.NoError(t, err)
			pool2, err := vf.newPool(1024, 2, 0, sharedSem)
			require.NoError(t, err)

			p1b1, err := pool1.Get()
			require.NoError(t, err)
			p2b1, err := pool2.Get()
			require.NoError(t, err)

			// Now sharedSem (2) is exhausted, and pool2 has 1 block (< maxBlocks=2).
			// Calling pool2.Get() must wait for either pool2 local Release or sharedSem permit.
			p2GotCh := make(chan Block, 1)
			go func() {
				b, getErr := pool2.Get()
				require.NoError(t, getErr)
				p2GotCh <- b
			}()

			time.Sleep(30 * time.Millisecond)
			// Release pool1's block & clear pool1 so sharedSem gets 1 permit back.
			pool1.Release(p1b1)
			require.NoError(t, pool1.ClearFreeBlockChannel(true))

			select {
			case p2b2 := <-p2GotCh:
				require.NotNil(t, p2b2)
				pool2.Release(p2b2)
			case <-time.After(1 * time.Second):
				t.Fatal("pool2.Get() did not unblock after pool1 released global permit")
			}
			pool2.Release(p2b1)
			require.NoError(t, pool2.ClearFreeBlockChannel(true))
		})
	}
}

func TestSyncPoolVariants_CrossPoolBufferReuse(t *testing.T) {
	for _, vf := range getPoolFactories() {
		t.Run(vf.name, func(t *testing.T) {
			ResetSyncPools()
			sem := semaphore.NewWeighted(4)
			seenPtrs := make(map[*byte]int)
			reused := 0

			// Under -race, Go's sync.Pool randomly drops ~25% of Put calls to test
			// use-after-free bugs, so we run 20 lifecycles and verify cross-pool reuse occurs.
			for range 20 {
				bp, err := vf.newPool(4096, 2, 1, sem)
				require.NoError(t, err)
				b, err := bp.Get()
				require.NoError(t, err)
				assert.Equal(t, int64(0), b.Size())
				assert.Equal(t, int64(4096), b.Cap())
				_, err = b.Write([]byte("reuse-check"))
				require.NoError(t, err)
				ptr := unsafe.SliceData(b.(*syncPoolBlock).buffer[:cap(b.(*syncPoolBlock).buffer)])
				if seenPtrs[ptr] > 0 {
					reused++
				}
				seenPtrs[ptr]++
				bp.Release(b)
				require.NoError(t, bp.ClearFreeBlockChannel(true))
			}
			assert.Greater(t, reused, 0, "expected underlying slice backing array to be reused across pool lifecycles")
		})
	}
}

func TestSyncPoolVariants_PrefetchBlockPools(t *testing.T) {
	ResetSyncPools()
	sem := semaphore.NewWeighted(4)

	bpA, err := NewSyncPoolPrefetchBlockPool(2048, 2, 1, sem)
	require.NoError(t, err)
	blkA, err := bpA.Get()
	require.NoError(t, err)
	require.NoError(t, blkA.SetAbsStartOff(0))
	_, err = blkA.Write([]byte("prefetch-A"))
	require.NoError(t, err)
	bpA.Release(blkA)
	require.NoError(t, bpA.ClearFreeBlockChannel(true))

	bpB, err := NewDirectSyncPrefetchBlockPool(2048, 2, 1, sem)
	require.NoError(t, err)
	blkB, err := bpB.Get()
	require.NoError(t, err)
	require.NoError(t, blkB.SetAbsStartOff(2048))
	_, err = blkB.Write([]byte("prefetch-B"))
	require.NoError(t, err)
	bpB.Release(blkB)
	require.NoError(t, bpB.ClearFreeBlockChannel(true))

	require.True(t, sem.TryAcquire(4))
	sem.Release(4)
}

func TestSyncPoolVariants_PermitRollbackOnAllocationFailure(t *testing.T) {
	ResetSyncPools()
	for _, vf := range getPoolFactories() {
		t.Run(vf.name, func(t *testing.T) {
			sem := semaphore.NewWeighted(1)
			// Requesting > maxSyncPoolBlockSize (1 TiB) triggers createSyncPoolBlock error
			bp, err := vf.newPool(1024*1024*1024*1024, 2, 0, sem)
			require.NoError(t, err)

			_, err = bp.TryGet()
			require.Error(t, err)
			require.True(t, sem.TryAcquire(1), "permit leaked on TryGet failure")
			sem.Release(1)

			_, err = bp.Get()
			require.Error(t, err)
			require.True(t, sem.TryAcquire(1), "permit leaked on Get failure")
			sem.Release(1)
		})
	}
}

func TestSyncPoolVariants_ConcurrentRace(t *testing.T) {
	for _, vf := range getPoolFactories() {
		t.Run(vf.name, func(t *testing.T) {
			ResetSyncPools()
			const totalPermits = 12
			const workers = 16
			const iterations = 40
			const blockSize = 4096

			globalSem := semaphore.NewWeighted(totalPermits)
			var wg sync.WaitGroup
			wg.Add(workers)

			for w := range workers {
				go func(workerID int) {
					defer wg.Done()
					payload := bytes.Repeat([]byte{byte(workerID + 1)}, blockSize)
					readBuf := make([]byte, blockSize)

					for iter := range iterations {
						bp, err := vf.newPool(blockSize, 3, 1, globalSem)
						if err != nil {
							require.ErrorIs(t, err, CantAllocateAnyBlockError)
							continue
						}

						b1, err := bp.Get()
						require.NoError(t, err)
						_, err = b1.Write(payload)
						require.NoError(t, err)

						b2, err := bp.TryGet()
						if err == nil {
							_, err = b2.Write(payload)
							require.NoError(t, err)
						}

						// Verify b1 read
						_, err = b1.Seek(0, io.SeekStart)
						require.NoError(t, err)
						n, rErr := b1.Read(readBuf)
						require.True(t, rErr == nil || errors.Is(rErr, io.EOF))
						require.Equal(t, blockSize, n)
						require.Equal(t, payload[0], readBuf[0])

						// Release b1 and b2 (if acquired) from child goroutines to exercise Get/Release concurrency
						var relWg sync.WaitGroup
						relWg.Add(1)
						go func() {
							defer relWg.Done()
							bp.Release(b1)
						}()
						if b2 != nil {
							relWg.Add(1)
							go func() {
								defer relWg.Done()
								bp.Release(b2)
							}()
						}
						relWg.Wait()

						if iter%2 == 0 {
							require.NoError(t, bp.ClearFreeBlockChannel(false))
						}
						require.NoError(t, bp.ClearFreeBlockChannel(true))
					}
				}(w)
			}

			wg.Wait()
			// Verify all permits are returned after all workers complete
			require.True(t, globalSem.TryAcquire(totalPermits), "global semaphore permits leaked after concurrent churn")
			globalSem.Release(totalPermits)
		})
	}
}

func TestDirectSyncBlockPool_SharedPoolConcurrentGetRelease(t *testing.T) {
	ResetSyncPools()
	const totalPermits = 8
	const maxBlocks = 6
	const reservedBlocks = 2
	const workers = 12
	const iterations = 50
	const blockSize = 4096

	globalSem := semaphore.NewWeighted(totalPermits)
	bp, err := NewDirectSyncBlockPool(blockSize, maxBlocks, reservedBlocks, globalSem)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := range workers {
		go func(workerID int) {
			defer wg.Done()
			payload := bytes.Repeat([]byte{byte(workerID + 10)}, blockSize)
			readBuf := make([]byte, blockSize)
			for range iterations {
				blk, getErr := bp.Get()
				require.NoError(t, getErr)
				_, wErr := blk.Write(payload)
				require.NoError(t, wErr)
				_, sErr := blk.Seek(0, io.SeekStart)
				require.NoError(t, sErr)
				n, rErr := blk.Read(readBuf)
				require.True(t, rErr == nil || errors.Is(rErr, io.EOF))
				require.Equal(t, blockSize, n)
				require.Equal(t, payload[blockSize-1], readBuf[blockSize-1])
				bp.Release(blk)
			}
		}(w)
	}
	wg.Wait()

	require.NoError(t, bp.ClearFreeBlockChannel(true))
	require.True(t, globalSem.TryAcquire(totalPermits))
	globalSem.Release(totalPermits)
}
