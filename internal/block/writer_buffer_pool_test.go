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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testPoolBlockSizeBytes = 4096
	hugeBlockSizeBytes     = 1 << 50 // Exceeds user virtual address space so mmap fails deterministically.
)

func newTestBudget(initialBytes int64) *atomic.Int64 {
	b := &atomic.Int64{}
	b.Store(initialBytes)
	return b
}

func newTestWriterBufferPool(t *testing.T, blockSizeBytes, maxBlocksPerFile int, budget *atomic.Int64) *WriterBufferPool {
	t.Helper()
	pool, err := NewWriterBufferPool(blockSizeBytes, maxBlocksPerFile, budget)
	require.NoError(t, err)
	t.Cleanup(pool.Destroy)
	return pool
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
			name:             "zero_max_blocks_per_file",
			blockSizeBytes:   testPoolBlockSizeBytes,
			maxBlocksPerFile: 0,
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

func TestWriterBufferPool_Get_AllocatesFromMmap(t *testing.T) {
	testCases := []struct {
		name                string
		initialBudget       int64
		expectedFinalBudget int64
	}{
		{
			name:                "within_budget",
			initialBudget:       2 * testPoolBlockSizeBytes,
			expectedFinalBudget: testPoolBlockSizeBytes,
		},
		{
			name:                "overcommits_when_budget_exhausted",
			initialBudget:       0,
			expectedFinalBudget: -testPoolBlockSizeBytes,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			budget := newTestBudget(tc.initialBudget)
			pool := newTestWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)

			buf, err := pool.Get(testPoolBlockSizeBytes)

			require.NoError(t, err)
			defer pool.Put(buf)
			require.Len(t, buf, testPoolBlockSizeBytes)
			assert.Equal(t, testPoolBlockSizeBytes, cap(buf))
			assert.Equal(t, tc.expectedFinalBudget, budget.Load())
		})
	}
}

func TestWriterBufferPool_Get_ReusesCachedBlock(t *testing.T) {
	budget := newTestBudget(2 * testPoolBlockSizeBytes)
	pool := newTestWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)
	seedBuf, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	seedPtr := &seedBuf[0]
	pool.Put(seedBuf)

	reused, err := pool.Get(testPoolBlockSizeBytes)

	require.NoError(t, err)
	defer pool.Put(reused)
	require.Len(t, reused, testPoolBlockSizeBytes)
	assert.Same(t, seedPtr, &reused[0])
	assert.Equal(t, int64(testPoolBlockSizeBytes), budget.Load())
}

func TestWriterBufferPool_Get_SizeMismatchReturnsError(t *testing.T) {
	testCases := []struct {
		name    string
		reqSize int
	}{
		{
			name:    "smaller_size",
			reqSize: testPoolBlockSizeBytes / 2,
		},
		{
			name:    "larger_size",
			reqSize: testPoolBlockSizeBytes * 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			budget := newTestBudget(2 * testPoolBlockSizeBytes)
			pool := newTestWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)
			cached, err := pool.Get(testPoolBlockSizeBytes)
			require.NoError(t, err)
			pool.Put(cached)

			buf, err := pool.Get(tc.reqSize)

			require.Error(t, err)
			assert.Nil(t, buf)
			assert.Equal(t, 1, len(pool.freeBlocksCh))
			assert.Equal(t, int64(testPoolBlockSizeBytes), budget.Load())
		})
	}
}

func TestWriterBufferPool_Get_MmapFailureRestoresBudget(t *testing.T) {
	const initialBudget = int64(2 * hugeBlockSizeBytes)
	budget := newTestBudget(initialBudget)
	pool := newTestWriterBufferPool(t, hugeBlockSizeBytes, 2, budget)

	buf, err := pool.Get(hugeBlockSizeBytes)

	require.Error(t, err)
	assert.Nil(t, buf)
	assert.ErrorContains(t, err, "mmap error")
	assert.Equal(t, initialBudget, budget.Load())
}

func TestWriterBufferPool_TryGet_AllocatesWhenBudgetSufficient(t *testing.T) {
	budget := newTestBudget(testPoolBlockSizeBytes)
	pool := newTestWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)

	buf, ok, err := pool.TryGet(testPoolBlockSizeBytes)

	require.NoError(t, err)
	require.True(t, ok)
	defer pool.Put(buf)
	require.Len(t, buf, testPoolBlockSizeBytes)
	assert.Equal(t, testPoolBlockSizeBytes, cap(buf))
	assert.Equal(t, int64(0), budget.Load())
}

func TestWriterBufferPool_TryGet_ReturnsFalseWhenBudgetInsufficient(t *testing.T) {
	const initialBudget = int64(testPoolBlockSizeBytes - 1)
	budget := newTestBudget(initialBudget)
	pool := newTestWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)

	buf, ok, err := pool.TryGet(testPoolBlockSizeBytes)

	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, buf)
	assert.Equal(t, initialBudget, budget.Load())
}

func TestWriterBufferPool_TryGet_ReusesCachedBlockWhenBudgetExhausted(t *testing.T) {
	budget := newTestBudget(testPoolBlockSizeBytes)
	pool := newTestWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)
	seedBuf, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	seedPtr := &seedBuf[0]
	pool.Put(seedBuf)

	reused, ok, err := pool.TryGet(testPoolBlockSizeBytes)

	require.NoError(t, err)
	require.True(t, ok)
	defer pool.Put(reused)
	require.Len(t, reused, testPoolBlockSizeBytes)
	assert.Same(t, seedPtr, &reused[0])
	assert.Equal(t, int64(0), budget.Load())
}

func TestWriterBufferPool_TryGet_SizeMismatchReturnsError(t *testing.T) {
	testCases := []struct {
		name    string
		reqSize int
	}{
		{
			name:    "smaller_size",
			reqSize: testPoolBlockSizeBytes / 2,
		},
		{
			name:    "larger_size",
			reqSize: testPoolBlockSizeBytes * 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			budget := newTestBudget(2 * testPoolBlockSizeBytes)
			pool := newTestWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)
			cached, err := pool.Get(testPoolBlockSizeBytes)
			require.NoError(t, err)
			pool.Put(cached)

			buf, ok, err := pool.TryGet(tc.reqSize)

			require.Error(t, err)
			assert.False(t, ok)
			assert.Nil(t, buf)
			assert.Equal(t, 1, len(pool.freeBlocksCh))
			assert.Equal(t, int64(testPoolBlockSizeBytes), budget.Load())
		})
	}
}

func TestWriterBufferPool_TryGet_MmapFailureRestoresBudget(t *testing.T) {
	const initialBudget = int64(2 * hugeBlockSizeBytes)
	budget := newTestBudget(initialBudget)
	pool := newTestWriterBufferPool(t, hugeBlockSizeBytes, 2, budget)

	buf, ok, err := pool.TryGet(hugeBlockSizeBytes)

	require.Error(t, err)
	assert.False(t, ok)
	assert.Nil(t, buf)
	assert.ErrorContains(t, err, "mmap error")
	assert.Equal(t, initialBudget, budget.Load())
}

func TestWriterBufferPool_Put_CachesBlockWhenWithinBudget(t *testing.T) {
	budget := newTestBudget(2 * testPoolBlockSizeBytes)
	pool := newTestWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)
	buf, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)

	pool.Put(buf)

	assert.Equal(t, 1, len(pool.freeBlocksCh))
	assert.Equal(t, int64(testPoolBlockSizeBytes), budget.Load())
}

func TestWriterBufferPool_Put_UnmapsImmediatelyWhenOverBudget(t *testing.T) {
	budget := newTestBudget(testPoolBlockSizeBytes / 2)
	pool := newTestWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)
	buf, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)

	pool.Put(buf)

	assert.Equal(t, 0, len(pool.freeBlocksCh))
	assert.Equal(t, int64(testPoolBlockSizeBytes/2), budget.Load())
}

func TestWriterBufferPool_Put_UnmapsWhenFreeChannelIsFull(t *testing.T) {
	budget := newTestBudget(3 * testPoolBlockSizeBytes)
	pool := newTestWriterBufferPool(t, testPoolBlockSizeBytes, 1, budget)
	buf1, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	buf2, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	pool.Put(buf1)

	pool.Put(buf2)

	assert.Equal(t, 1, len(pool.freeBlocksCh))
	assert.Equal(t, int64(2*testPoolBlockSizeBytes), budget.Load())
}

func TestWriterBufferPool_Put_RestoresSliceLength(t *testing.T) {
	budget := newTestBudget(2 * testPoolBlockSizeBytes)
	pool := newTestWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)
	buf, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	bufPtr := &buf[0]
	pool.Put(buf[:5])

	reused, err := pool.Get(testPoolBlockSizeBytes)

	require.NoError(t, err)
	defer pool.Put(reused)
	require.Len(t, reused, testPoolBlockSizeBytes)
	assert.Same(t, bufPtr, &reused[0])
}

func TestWriterBufferPool_Put_IgnoresUnexpectedCapacity(t *testing.T) {
	budget := newTestBudget(2 * testPoolBlockSizeBytes)
	pool := newTestWriterBufferPool(t, testPoolBlockSizeBytes, 2, budget)
	validBuf, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	defer pool.Put(validBuf)

	testCases := []struct {
		name   string
		badBuf []byte
	}{
		{
			name:   "nil_slice",
			badBuf: nil,
		},
		{
			name:   "larger_capacity",
			badBuf: make([]byte, testPoolBlockSizeBytes*2),
		},
		{
			name:   "front_resliced_block",
			badBuf: validBuf[1:],
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool.Put(tc.badBuf)

			assert.Equal(t, 0, len(pool.freeBlocksCh))
			assert.Equal(t, int64(testPoolBlockSizeBytes), budget.Load())
		})
	}
}

func TestWriterBufferPool_Destroy_DrainsCachedBlocksAndIsIdempotent(t *testing.T) {
	const initialBudget = int64(4 * testPoolBlockSizeBytes)
	budget := newTestBudget(initialBudget)
	pool, err := NewWriterBufferPool(testPoolBlockSizeBytes, 3, budget)
	require.NoError(t, err)
	buf1, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	buf2, err := pool.Get(testPoolBlockSizeBytes)
	require.NoError(t, err)
	pool.Put(buf1)
	pool.Put(buf2)

	pool.Destroy()
	pool.Destroy()

	assert.Equal(t, 0, len(pool.freeBlocksCh))
	assert.Equal(t, initialBudget, budget.Load())
}

func TestWriterBufferPool_ConcurrentAccessConservesGlobalBudget(t *testing.T) {
	const (
		initialBudget    = int64(2 * testPoolBlockSizeBytes)
		numWorkers       = 4
		maxBlocksPerFile = 2
		iterations       = 20
	)
	budget := newTestBudget(initialBudget)
	pool, err := NewWriterBufferPool(testPoolBlockSizeBytes, maxBlocksPerFile, budget)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for workerID := range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				if workerID == 0 {
					buf, err := pool.Get(testPoolBlockSizeBytes)
					if !assert.NoError(t, err) {
						return
					}
					pool.Put(buf)
				} else {
					buf, ok, err := pool.TryGet(testPoolBlockSizeBytes)
					if !assert.NoError(t, err) {
						return
					}
					if ok {
						pool.Put(buf)
					}
				}
			}
		}()
	}
	wg.Wait()
	pool.Destroy()

	assert.Equal(t, 0, len(pool.freeBlocksCh))
	assert.Equal(t, initialBudget, budget.Load())
}
