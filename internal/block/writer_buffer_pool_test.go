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

const testPoolBlockSizeBytes = 4096

func newTestBudget(initialBytes int64) *atomic.Int64 {
	b := &atomic.Int64{}
	b.Store(initialBytes)
	return b
}

func mustNewWriterBufferPool(t *testing.T, blockSizeBytes, maxBlocksPerFile int, budget *atomic.Int64) *WriterBufferPool {
	t.Helper()
	pool, err := NewWriterBufferPool(blockSizeBytes, maxBlocksPerFile, budget)
	require.NoError(t, err)
	t.Cleanup(pool.Destroy)
	return pool
}

func TestNewWriterBufferPool_Success(t *testing.T) {
	const initialBudget = int64(8 * testPoolBlockSizeBytes)
	budget := newTestBudget(initialBudget)

	pool, err := NewWriterBufferPool(testPoolBlockSizeBytes, 4, budget)

	require.NoError(t, err)
	require.NotNil(t, pool)
	assert.Equal(t, testPoolBlockSizeBytes, pool.blockSizeBytes)
	assert.Equal(t, 4, cap(pool.freeBlocksCh))
	assert.Equal(t, 0, len(pool.freeBlocksCh))
	assert.Equal(t, initialBudget, budget.Load())
}

func TestNewWriterBufferPool_InvalidConfig(t *testing.T) {
	validBudget := newTestBudget(4 * testPoolBlockSizeBytes)
	testCases := []struct {
		name             string
		blockSizeBytes   int
		maxBlocksPerFile int
		budget           *atomic.Int64
	}{
		{
			name:             "zero_block_size",
			blockSizeBytes:   0,
			maxBlocksPerFile: 4,
			budget:           validBudget,
		},
		{
			name:             "negative_block_size",
			blockSizeBytes:   -1,
			maxBlocksPerFile: 4,
			budget:           validBudget,
		},
		{
			name:             "zero_max_blocks_per_file",
			blockSizeBytes:   testPoolBlockSizeBytes,
			maxBlocksPerFile: 0,
			budget:           validBudget,
		},
		{
			name:             "negative_max_blocks_per_file",
			blockSizeBytes:   testPoolBlockSizeBytes,
			maxBlocksPerFile: -2,
			budget:           validBudget,
		},
		{
			name:             "nil_global_byte_budget",
			blockSizeBytes:   testPoolBlockSizeBytes,
			maxBlocksPerFile: 4,
			budget:           nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := NewWriterBufferPool(tc.blockSizeBytes, tc.maxBlocksPerFile, tc.budget)

			require.Error(t, err)
			assert.Nil(t, pool)
		})
	}
}

func TestWriterBufferPool_Get_AllocatesAndOvercommitsBudget(t *testing.T) {
	budget := newTestBudget(0)
	pool := mustNewWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)

	buf, err := pool.Get(testPoolBlockSizeBytes)

	require.NoError(t, err)
	require.Len(t, buf, testPoolBlockSizeBytes)
	assert.Equal(t, testPoolBlockSizeBytes, cap(buf))
	assert.Equal(t, int64(-testPoolBlockSizeBytes), budget.Load())

	copy(buf, []byte("hello parallel upload"))
	assert.Equal(t, []byte("hello parallel upload"), buf[:21])
	pool.Put(buf)
	assert.Equal(t, int64(0), budget.Load())
	assert.Equal(t, 0, len(pool.freeBlocksCh))
}

func TestWriterBufferPool_Get_ReusesCachedBlockWithoutChangingBudget(t *testing.T) {
	budget := newTestBudget(2 * testPoolBlockSizeBytes)
	pool := mustNewWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)

	first, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	assert.Equal(t, int64(testPoolBlockSizeBytes), budget.Load())
	firstPtr := &first[0]
	pool.Put(first)
	require.Equal(t, 1, len(pool.freeBlocksCh))

	reused, err := pool.Get(testPoolBlockSizeBytes)

	require.NoError(t, err)
	require.Len(t, reused, testPoolBlockSizeBytes)
	assert.Same(t, firstPtr, &reused[0])
	assert.Equal(t, int64(testPoolBlockSizeBytes), budget.Load())
	assert.Equal(t, 0, len(pool.freeBlocksCh))
	pool.Put(reused)
}

func TestWriterBufferPool_Get_SizeMismatchDoesNotConsumeCachedBlockOrBudget(t *testing.T) {
	budget := newTestBudget(2 * testPoolBlockSizeBytes)
	pool := mustNewWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)
	cached, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	pool.Put(cached)
	require.Equal(t, 1, len(pool.freeBlocksCh))
	require.Equal(t, int64(testPoolBlockSizeBytes), budget.Load())

	buf, err := pool.Get(testPoolBlockSizeBytes / 2)

	require.Error(t, err)
	assert.Nil(t, buf)
	assert.Equal(t, 1, len(pool.freeBlocksCh))
	assert.Equal(t, int64(testPoolBlockSizeBytes), budget.Load())
}

func TestWriterBufferPool_Get_MmapFailureRestoresBudget(t *testing.T) {
	const hugeBlockSizeBytes = 1 << 50 // Exceeds user virtual address space so mmap fails.
	const initialBudget = int64(3 * testPoolBlockSizeBytes)
	budget := newTestBudget(initialBudget)
	pool := mustNewWriterBufferPool(t, hugeBlockSizeBytes, 2, budget)

	buf, err := pool.Get(hugeBlockSizeBytes)

	require.Error(t, err)
	assert.Nil(t, buf)
	assert.ErrorContains(t, err, "mmap error")
	assert.Equal(t, initialBudget, budget.Load())
}

func TestWriterBufferPool_TryGet_AllocatesWhenBudgetSufficient(t *testing.T) {
	budget := newTestBudget(2 * testPoolBlockSizeBytes)
	pool := mustNewWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)

	buf1, ok1, err1 := pool.TryGet(testPoolBlockSizeBytes)
	require.NoError(t, err1)
	require.True(t, ok1)
	require.Len(t, buf1, testPoolBlockSizeBytes)
	assert.Equal(t, int64(testPoolBlockSizeBytes), budget.Load())

	buf2, ok2, err2 := pool.TryGet(testPoolBlockSizeBytes)
	require.NoError(t, err2)
	require.True(t, ok2)
	require.Len(t, buf2, testPoolBlockSizeBytes)
	assert.Equal(t, int64(0), budget.Load())

	buf3, ok3, err3 := pool.TryGet(testPoolBlockSizeBytes)
	require.NoError(t, err3)
	assert.False(t, ok3)
	assert.Nil(t, buf3)
	assert.Equal(t, int64(0), budget.Load())

	pool.Put(buf1)
	pool.Put(buf2)
}

func TestWriterBufferPool_TryGet_ReturnsFalseWhenBudgetInsufficient(t *testing.T) {
	testCases := []struct {
		name          string
		initialBudget int64
	}{
		{name: "partial_budget_less_than_block_size", initialBudget: testPoolBlockSizeBytes - 1},
		{name: "zero_budget", initialBudget: 0},
		{name: "negative_budget", initialBudget: -3 * testPoolBlockSizeBytes},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			budget := newTestBudget(tc.initialBudget)
			pool := mustNewWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)

			buf, ok, err := pool.TryGet(testPoolBlockSizeBytes)

			require.NoError(t, err)
			assert.False(t, ok)
			assert.Nil(t, buf)
			assert.Equal(t, tc.initialBudget, budget.Load())
		})
	}
}

func TestWriterBufferPool_TryGet_ReusesCachedBlockEvenWhenBudgetExhausted(t *testing.T) {
	budget := newTestBudget(testPoolBlockSizeBytes)
	pool := mustNewWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)

	buf, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	assert.Equal(t, int64(0), budget.Load())
	origPtr := &buf[0]
	pool.Put(buf)
	require.Equal(t, 1, len(pool.freeBlocksCh))

	reused, ok, err := pool.TryGet(testPoolBlockSizeBytes)

	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, reused, testPoolBlockSizeBytes)
	assert.Same(t, origPtr, &reused[0])
	assert.Equal(t, int64(0), budget.Load())
	pool.Put(reused)
}

func TestWriterBufferPool_TryGet_SizeMismatchReturnsError(t *testing.T) {
	budget := newTestBudget(2 * testPoolBlockSizeBytes)
	pool := mustNewWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)

	buf, ok, err := pool.TryGet(testPoolBlockSizeBytes * 2)

	require.Error(t, err)
	assert.False(t, ok)
	assert.Nil(t, buf)
	assert.Equal(t, int64(2*testPoolBlockSizeBytes), budget.Load())
}

func TestWriterBufferPool_TryGet_MmapFailureRestoresBudget(t *testing.T) {
	const hugeBlockSizeBytes = 1 << 50
	const initialBudget = int64(2 * hugeBlockSizeBytes)
	budget := newTestBudget(initialBudget)
	pool := mustNewWriterBufferPool(t, hugeBlockSizeBytes, 2, budget)

	buf, ok, err := pool.TryGet(hugeBlockSizeBytes)

	require.Error(t, err)
	assert.False(t, ok)
	assert.Nil(t, buf)
	assert.ErrorContains(t, err, "mmap error")
	assert.Equal(t, initialBudget, budget.Load())
}

func TestWriterBufferPool_Put_RestoresSliceLengthWithoutZeroingData(t *testing.T) {
	budget := newTestBudget(2 * testPoolBlockSizeBytes)
	pool := mustNewWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)

	buf, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	copy(buf, []byte("dirty-payload"))

	pool.Put(buf[:5])
	require.Equal(t, 1, len(pool.freeBlocksCh))

	reused, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	assert.Len(t, reused, testPoolBlockSizeBytes)
	assert.Equal(t, testPoolBlockSizeBytes, cap(reused))
	assert.Equal(t, []byte("dirty-payload"), reused[:13])
	pool.Put(reused)
}

func TestWriterBufferPool_Put_IgnoresUnexpectedCapacity(t *testing.T) {
	budget := newTestBudget(2 * testPoolBlockSizeBytes)
	pool := mustNewWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)

	validBuf, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	require.Equal(t, int64(testPoolBlockSizeBytes), budget.Load())

	pool.Put(nil)
	pool.Put(make([]byte, 128))
	pool.Put(make([]byte, testPoolBlockSizeBytes*2))
	pool.Put(validBuf[1:]) // cap == blockSizeBytes - 1

	assert.Equal(t, 0, len(pool.freeBlocksCh))
	assert.Equal(t, int64(testPoolBlockSizeBytes), budget.Load())

	pool.Put(validBuf)
	assert.Equal(t, 1, len(pool.freeBlocksCh))
}

func TestWriterBufferPool_Put_UnmapsImmediatelyWhenOverBudget(t *testing.T) {
	const initialBudget = int64(testPoolBlockSizeBytes + testPoolBlockSizeBytes/2)
	budget := newTestBudget(initialBudget)
	pool1 := mustNewWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)
	pool2 := mustNewWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)

	buf1, err := pool1.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	assert.Equal(t, int64(testPoolBlockSizeBytes/2), budget.Load())

	buf2, err := pool2.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	assert.Equal(t, int64(-testPoolBlockSizeBytes/2), budget.Load())

	pool1.Put(buf1)
	assert.Equal(t, 0, len(pool1.freeBlocksCh))
	assert.Equal(t, int64(testPoolBlockSizeBytes/2), budget.Load())

	pool2.Put(buf2)
	assert.Equal(t, 1, len(pool2.freeBlocksCh))
	assert.Equal(t, int64(testPoolBlockSizeBytes/2), budget.Load())

	pool2.Destroy()
	assert.Equal(t, 0, len(pool2.freeBlocksCh))
	assert.Equal(t, initialBudget, budget.Load())
}

func TestWriterBufferPool_Put_UnmapsWhenFreeChannelFull(t *testing.T) {
	budget := newTestBudget(5 * testPoolBlockSizeBytes)
	pool := mustNewWriterBufferPool(t, testPoolBlockSizeBytes, 1, budget)

	buf1, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	buf2, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	assert.Equal(t, int64(3*testPoolBlockSizeBytes), budget.Load())

	pool.Put(buf1)
	assert.Equal(t, 1, len(pool.freeBlocksCh))
	assert.Equal(t, int64(3*testPoolBlockSizeBytes), budget.Load())

	pool.Put(buf2)
	assert.Equal(t, 1, len(pool.freeBlocksCh))
	assert.Equal(t, int64(4*testPoolBlockSizeBytes), budget.Load())

	pool.Destroy()
	assert.Equal(t, 0, len(pool.freeBlocksCh))
	assert.Equal(t, int64(5*testPoolBlockSizeBytes), budget.Load())
}

func TestWriterBufferPool_Destroy_DrainsAndIsIdempotent(t *testing.T) {
	const initialBudget = int64(4 * testPoolBlockSizeBytes)
	budget := newTestBudget(initialBudget)
	pool, err := NewWriterBufferPool(testPoolBlockSizeBytes, 3, budget)
	require.NoError(t, err)

	pool.Destroy()
	assert.Equal(t, initialBudget, budget.Load())

	b1, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	b2, ok, err := pool.TryGet(testPoolBlockSizeBytes)
	require.NoError(t, err)
	require.True(t, ok)
	b3, ok, err := pool.TryGet(testPoolBlockSizeBytes)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(testPoolBlockSizeBytes), budget.Load())

	pool.Put(b1)
	pool.Put(b2)
	pool.Put(b3)
	assert.Equal(t, 3, len(pool.freeBlocksCh))
	assert.Equal(t, int64(testPoolBlockSizeBytes), budget.Load())

	pool.Destroy()
	assert.Equal(t, 0, len(pool.freeBlocksCh))
	assert.Equal(t, initialBudget, budget.Load())

	pool.Destroy()
	assert.Equal(t, 0, len(pool.freeBlocksCh))
	assert.Equal(t, initialBudget, budget.Load())
}

func TestWriterBufferPool_ConcurrentWritersConserveGlobalBudget(t *testing.T) {
	const (
		initialBudget    = int64(10 * testPoolBlockSizeBytes)
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
			writerBlockSizeBytes := testPoolBlockSizeBytes
			if wID%2 == 1 {
				writerBlockSizeBytes = 2 * testPoolBlockSizeBytes
			}
			pool, err := NewWriterBufferPool(writerBlockSizeBytes, maxBlocksPerFile, budget)
			require.NoError(t, err)

			var partsWG sync.WaitGroup
			var inflight atomic.Int32
			slots := make(chan struct{}, maxBlocksPerFile)

			for partIdx := range partsPerWriter {
				slots <- struct{}{}
				var buf []byte
				if inflight.Load() == 0 {
					buf, err = pool.Get(writerBlockSizeBytes)
					require.NoError(t, err)
				} else {
					var ok bool
					buf, ok, err = pool.TryGet(writerBlockSizeBytes)
					require.NoError(t, err)
					if !ok {
						buf, err = pool.Get(writerBlockSizeBytes)
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
					pool.Put(b[:writerBlockSizeBytes/2])
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
