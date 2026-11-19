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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPoolBlockSize = 4096

func newTestBudget(initial int64) *atomic.Int64 {
	b := &atomic.Int64{}
	b.Store(initial)
	return b
}

func mustNewWriterBufferPool(t *testing.T, blockSize, maxBlocksPerFile int, budget *atomic.Int64) *WriterBufferPool {
	t.Helper()
	pool, err := NewWriterBufferPool(blockSize, maxBlocksPerFile, budget)
	require.NoError(t, err)
	t.Cleanup(pool.Destroy)
	return pool
}

func TestNewWriterBufferPool_Success(t *testing.T) {
	budget := newTestBudget(8)

	pool, err := NewWriterBufferPool(testPoolBlockSize, 4, budget)

	require.NoError(t, err)
	require.NotNil(t, pool)
	assert.Equal(t, testPoolBlockSize, pool.blockSize)
	assert.Equal(t, 4, cap(pool.freeBlocksCh))
	assert.Equal(t, 0, len(pool.freeBlocksCh))
	assert.Equal(t, int64(8), budget.Load())
}

func TestNewWriterBufferPool_InvalidConfig(t *testing.T) {
	validBudget := newTestBudget(4)
	testCases := []struct {
		name             string
		blockSize        int
		maxBlocksPerFile int
		budget           *atomic.Int64
	}{
		{
			name:             "zero_block_size",
			blockSize:        0,
			maxBlocksPerFile: 4,
			budget:           validBudget,
		},
		{
			name:             "negative_block_size",
			blockSize:        -1,
			maxBlocksPerFile: 4,
			budget:           validBudget,
		},
		{
			name:             "zero_max_blocks_per_file",
			blockSize:        testPoolBlockSize,
			maxBlocksPerFile: 0,
			budget:           validBudget,
		},
		{
			name:             "negative_max_blocks_per_file",
			blockSize:        testPoolBlockSize,
			maxBlocksPerFile: -2,
			budget:           validBudget,
		},
		{
			name:             "nil_global_block_budget",
			blockSize:        testPoolBlockSize,
			maxBlocksPerFile: 4,
			budget:           nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := NewWriterBufferPool(tc.blockSize, tc.maxBlocksPerFile, tc.budget)

			require.Error(t, err)
			assert.Nil(t, pool)
		})
	}
}

func TestWriterBufferPool_Get_AllocatesAndOvercommitsBudget(t *testing.T) {
	budget := newTestBudget(0)
	pool := mustNewWriterBufferPool(t, testPoolBlockSize, 2, budget)

	buf, err := pool.Get(testPoolBlockSize)

	require.NoError(t, err)
	require.Len(t, buf, testPoolBlockSize)
	assert.Equal(t, testPoolBlockSize, cap(buf))
	assert.Equal(t, int64(-1), budget.Load())

	// Verify the mmap-backed slice is readable and writable, then return it.
	copy(buf, []byte("hello parallel upload"))
	assert.Equal(t, []byte("hello parallel upload"), buf[:21])
	pool.Put(buf)
	assert.Equal(t, int64(0), budget.Load())
	assert.Equal(t, 0, len(pool.freeBlocksCh))
}

func TestWriterBufferPool_Get_ReusesCachedBlockWithoutChangingBudget(t *testing.T) {
	budget := newTestBudget(2)
	pool := mustNewWriterBufferPool(t, testPoolBlockSize, 2, budget)

	first, err := pool.Get(testPoolBlockSize)
	require.NoError(t, err)
	assert.Equal(t, int64(1), budget.Load())
	firstPtr := &first[0]
	pool.Put(first)
	require.Equal(t, 1, len(pool.freeBlocksCh))

	reused, err := pool.Get(testPoolBlockSize)

	require.NoError(t, err)
	require.Len(t, reused, testPoolBlockSize)
	assert.Same(t, firstPtr, &reused[0])
	assert.Equal(t, int64(1), budget.Load())
	assert.Equal(t, 0, len(pool.freeBlocksCh))
	pool.Put(reused)
}

func TestWriterBufferPool_Get_SizeMismatchDoesNotConsumeCachedBlockOrBudget(t *testing.T) {
	budget := newTestBudget(2)
	pool := mustNewWriterBufferPool(t, testPoolBlockSize, 2, budget)
	cached, err := pool.Get(testPoolBlockSize)
	require.NoError(t, err)
	pool.Put(cached)
	require.Equal(t, 1, len(pool.freeBlocksCh))
	require.Equal(t, int64(1), budget.Load())

	buf, err := pool.Get(testPoolBlockSize / 2)

	require.Error(t, err)
	assert.Nil(t, buf)
	assert.Equal(t, 1, len(pool.freeBlocksCh))
	assert.Equal(t, int64(1), budget.Load())
}

func TestWriterBufferPool_Get_MmapFailureRestoresBudget(t *testing.T) {
	const hugeBlockSize = 1 << 50 // 1 PiB exceeds user virtual address space and fails mmap deterministically.
	budget := newTestBudget(3)
	pool := mustNewWriterBufferPool(t, hugeBlockSize, 2, budget)

	buf, err := pool.Get(hugeBlockSize)

	require.Error(t, err)
	assert.Nil(t, buf)
	assert.ErrorContains(t, err, "mmap error")
	assert.Equal(t, int64(3), budget.Load())
}

func TestWriterBufferPool_TryGet_AllocatesWhenBudgetPositive(t *testing.T) {
	budget := newTestBudget(2)
	pool := mustNewWriterBufferPool(t, testPoolBlockSize, 2, budget)

	buf1, ok1, err1 := pool.TryGet(testPoolBlockSize)
	require.NoError(t, err1)
	require.True(t, ok1)
	require.Len(t, buf1, testPoolBlockSize)
	assert.Equal(t, int64(1), budget.Load())

	buf2, ok2, err2 := pool.TryGet(testPoolBlockSize)
	require.NoError(t, err2)
	require.True(t, ok2)
	require.Len(t, buf2, testPoolBlockSize)
	assert.Equal(t, int64(0), budget.Load())

	// Third TryGet sees budget == 0 and returns ok = false with nil error.
	buf3, ok3, err3 := pool.TryGet(testPoolBlockSize)
	require.NoError(t, err3)
	assert.False(t, ok3)
	assert.Nil(t, buf3)
	assert.Equal(t, int64(0), budget.Load())

	pool.Put(buf1)
	pool.Put(buf2)
}

func TestWriterBufferPool_TryGet_ReturnsFalseWhenBudgetZeroOrNegative(t *testing.T) {
	testCases := []struct {
		name          string
		initialBudget int64
	}{
		{name: "zero_budget", initialBudget: 0},
		{name: "negative_budget", initialBudget: -3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			budget := newTestBudget(tc.initialBudget)
			pool := mustNewWriterBufferPool(t, testPoolBlockSize, 2, budget)

			buf, ok, err := pool.TryGet(testPoolBlockSize)

			require.NoError(t, err)
			assert.False(t, ok)
			assert.Nil(t, buf)
			assert.Equal(t, tc.initialBudget, budget.Load())
		})
	}
}

func TestWriterBufferPool_TryGet_ReusesCachedBlockEvenWhenBudgetExhausted(t *testing.T) {
	budget := newTestBudget(1)
	pool := mustNewWriterBufferPool(t, testPoolBlockSize, 2, budget)

	buf, err := pool.Get(testPoolBlockSize)
	require.NoError(t, err)
	assert.Equal(t, int64(0), budget.Load())
	origPtr := &buf[0]
	pool.Put(buf)
	require.Equal(t, 1, len(pool.freeBlocksCh))

	// Even with globalBlockBudget == 0, TryGet reuses this writer's cached block.
	reused, ok, err := pool.TryGet(testPoolBlockSize)

	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, reused, testPoolBlockSize)
	assert.Same(t, origPtr, &reused[0])
	assert.Equal(t, int64(0), budget.Load())
	pool.Put(reused)
}

func TestWriterBufferPool_TryGet_SizeMismatchReturnsError(t *testing.T) {
	budget := newTestBudget(2)
	pool := mustNewWriterBufferPool(t, testPoolBlockSize, 2, budget)

	buf, ok, err := pool.TryGet(testPoolBlockSize * 2)

	require.Error(t, err)
	assert.False(t, ok)
	assert.Nil(t, buf)
	assert.Equal(t, int64(2), budget.Load())
}

func TestWriterBufferPool_TryGet_MmapFailureRestoresBudget(t *testing.T) {
	const hugeBlockSize = 1 << 50
	budget := newTestBudget(2)
	pool := mustNewWriterBufferPool(t, hugeBlockSize, 2, budget)

	buf, ok, err := pool.TryGet(hugeBlockSize)

	require.Error(t, err)
	assert.False(t, ok)
	assert.Nil(t, buf)
	assert.ErrorContains(t, err, "mmap error")
	assert.Equal(t, int64(2), budget.Load())
}

func TestWriterBufferPool_Put_RestoresSliceLengthWithoutZeroingData(t *testing.T) {
	budget := newTestBudget(2)
	pool := mustNewWriterBufferPool(t, testPoolBlockSize, 2, budget)

	buf, err := pool.Get(testPoolBlockSize)
	require.NoError(t, err)
	copy(buf, []byte("dirty-payload"))

	// Reslice length down to simulate caller reslicing from index 0.
	pool.Put(buf[:5])
	require.Equal(t, 1, len(pool.freeBlocksCh))

	reused, err := pool.Get(testPoolBlockSize)
	require.NoError(t, err)
	assert.Len(t, reused, testPoolBlockSize)
	assert.Equal(t, testPoolBlockSize, cap(reused))
	assert.Equal(t, []byte("dirty-payload"), reused[:13])
	pool.Put(reused)
}

func TestWriterBufferPool_Put_IgnoresUnexpectedCapacity(t *testing.T) {
	budget := newTestBudget(2)
	pool := mustNewWriterBufferPool(t, testPoolBlockSize, 2, budget)

	validBuf, err := pool.Get(testPoolBlockSize)
	require.NoError(t, err)
	require.Equal(t, int64(1), budget.Load())

	pool.Put(nil)
	pool.Put(make([]byte, 128))
	pool.Put(make([]byte, testPoolBlockSize*2))
	pool.Put(validBuf[1:]) // Front-resliced buffer has cap == blockSize-1 and must not be munmapped.

	assert.Equal(t, 0, len(pool.freeBlocksCh))
	assert.Equal(t, int64(1), budget.Load())

	// The original validBuf is still mapped and safe to return properly.
	pool.Put(validBuf)
	assert.Equal(t, 1, len(pool.freeBlocksCh))
}

func TestWriterBufferPool_Put_UnmapsImmediatelyWhenOverBudget(t *testing.T) {
	budget := newTestBudget(1)
	pool1 := mustNewWriterBufferPool(t, testPoolBlockSize, 2, budget)
	pool2 := mustNewWriterBufferPool(t, testPoolBlockSize, 2, budget)

	buf1, err := pool1.Get(testPoolBlockSize)
	require.NoError(t, err)
	assert.Equal(t, int64(0), budget.Load())

	// Second writer's guaranteed first block pushes globalBlockBudget below 0.
	buf2, err := pool2.Get(testPoolBlockSize)
	require.NoError(t, err)
	assert.Equal(t, int64(-1), budget.Load())

	// Putting buf1 while budget < 0 unmaps immediately instead of caching.
	pool1.Put(buf1)
	assert.Equal(t, 0, len(pool1.freeBlocksCh))
	assert.Equal(t, int64(0), budget.Load())

	// Now budget == 0, so pool2.Put caches buf2 locally until pool2.Destroy().
	pool2.Put(buf2)
	assert.Equal(t, 1, len(pool2.freeBlocksCh))
	assert.Equal(t, int64(0), budget.Load())

	pool2.Destroy()
	assert.Equal(t, 0, len(pool2.freeBlocksCh))
	assert.Equal(t, int64(1), budget.Load())
}

func TestWriterBufferPool_Put_UnmapsWhenFreeChannelFull(t *testing.T) {
	budget := newTestBudget(5)
	pool := mustNewWriterBufferPool(t, testPoolBlockSize, 1, budget)

	buf1, err := pool.Get(testPoolBlockSize)
	require.NoError(t, err)
	buf2, err := pool.Get(testPoolBlockSize)
	require.NoError(t, err)
	assert.Equal(t, int64(3), budget.Load())

	pool.Put(buf1)
	assert.Equal(t, 1, len(pool.freeBlocksCh))
	assert.Equal(t, int64(3), budget.Load())

	// Channel capacity is 1, so buf2 is unmapped immediately and budget increments.
	pool.Put(buf2)
	assert.Equal(t, 1, len(pool.freeBlocksCh))
	assert.Equal(t, int64(4), budget.Load())

	pool.Destroy()
	assert.Equal(t, 0, len(pool.freeBlocksCh))
	assert.Equal(t, int64(5), budget.Load())
}

func TestWriterBufferPool_Destroy_DrainsAndIsIdempotent(t *testing.T) {
	budget := newTestBudget(4)
	pool, err := NewWriterBufferPool(testPoolBlockSize, 3, budget)
	require.NoError(t, err)

	// Calling Destroy on an empty pool is a no-op.
	pool.Destroy()
	assert.Equal(t, int64(4), budget.Load())

	b1, err := pool.Get(testPoolBlockSize)
	require.NoError(t, err)
	b2, ok, err := pool.TryGet(testPoolBlockSize)
	require.NoError(t, err)
	require.True(t, ok)
	b3, ok, err := pool.TryGet(testPoolBlockSize)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(1), budget.Load())

	pool.Put(b1)
	pool.Put(b2)
	pool.Put(b3)
	assert.Equal(t, 3, len(pool.freeBlocksCh))
	assert.Equal(t, int64(1), budget.Load())

	pool.Destroy()
	assert.Equal(t, 0, len(pool.freeBlocksCh))
	assert.Equal(t, int64(4), budget.Load())

	// Second Destroy call is safe and idempotent.
	pool.Destroy()
	assert.Equal(t, 0, len(pool.freeBlocksCh))
	assert.Equal(t, int64(4), budget.Load())
}

func TestWriterBufferPool_ConcurrentWritersConserveGlobalBudget(t *testing.T) {
	const (
		initialBudget    = int64(6)
		numWriters       = 8
		maxBlocksPerFile = 4
		partsPerWriter   = 24
	)
	budget := newTestBudget(initialBudget)

	var writersWG sync.WaitGroup
	for writerID := range numWriters {
		writersWG.Add(1)
		go func(wID int) {
			defer writersWG.Done()
			pool, err := NewWriterBufferPool(testPoolBlockSize, maxBlocksPerFile, budget)
			require.NoError(t, err)

			var partsWG sync.WaitGroup
			var inflight atomic.Int32
			slots := make(chan struct{}, maxBlocksPerFile)

			for partIdx := range partsPerWriter {
				slots <- struct{}{}
				var buf []byte
				if inflight.Load() == 0 {
					buf, err = pool.Get(testPoolBlockSize)
					require.NoError(t, err)
				} else {
					var ok bool
					buf, ok, err = pool.TryGet(testPoolBlockSize)
					require.NoError(t, err)
					if !ok {
						// Budget full; fall back to guaranteed Get when inflight drains to 0,
						// or wait for an in-flight worker slot and allocate via Get.
						buf, err = pool.Get(testPoolBlockSize)
						require.NoError(t, err)
					}
				}

				inflight.Add(1)
				partsWG.Add(1)
				pattern := byte((wID*31 + partIdx + 1) & 0xff)
				go func(b []byte, pat byte) {
					defer func() {
						inflight.Add(-1)
						<-slots
						partsWG.Done()
					}()
					fill := bytes.Repeat([]byte{pat}, len(b))
					copy(b, fill)
					assert.True(t, bytes.Equal(fill, b), "concurrent buffer aliasing detected")
					pool.Put(b[:testPoolBlockSize/2])
				}(buf, pattern)
			}

			partsWG.Wait()
			pool.Destroy()
			pool.Destroy()
			assert.Equal(t, 0, len(pool.freeBlocksCh))
		}(writerID)
	}

	writersWG.Wait()
	assert.Equal(t, initialBudget, budget.Load())
}
