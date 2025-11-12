// Copyright 2024 Google LLC
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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

const invalidConfigError string = "invalid configuration provided for blockPool, blocksize: %d, maxBlocks: %d"

type BlockPoolTest struct {
	suite.Suite
}

func TestBlockPoolTestSuite(t *testing.T) {
	suite.Run(t, new(BlockPoolTest))
}

func (t *BlockPoolTest) TestInitBlockPool() {
	bp, err := NewGenBlockPool(1024, 10, 0, semaphore.NewWeighted(10), createBlock)

	require.Nil(t.T(), err)
	require.NotNil(t.T(), bp)
	assert.Equal(t.T(), int64(1024), bp.blockSize)
	assert.Equal(t.T(), int64(10), bp.maxBlocks)
	assert.Equal(t.T(), int64(0), bp.totalBlocks)
}

func (t *BlockPoolTest) TestInitBlockPoolForZeroBlockSize() {
	_, err := NewGenBlockPool(0, 10, 0, semaphore.NewWeighted(10), createBlock)

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), fmt.Errorf(invalidConfigError, 0, 10), err)
}

func (t *BlockPoolTest) TestInitBlockPoolForNegativeBlockSize() {
	_, err := NewGenBlockPool(-1, 10, 0, semaphore.NewWeighted(10), createBlock)

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), fmt.Errorf(invalidConfigError, -1, 10), err)
}

func (t *BlockPoolTest) TestInitBlockPoolForZeroMaxBlocks() {
	_, err := NewGenBlockPool(10, 0, 0, semaphore.NewWeighted(10), createBlock)

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), fmt.Errorf(invalidConfigError, 10, 0), err)
}

func (t *BlockPoolTest) TestInitBlockPoolForNegativeMaxBlocks() {
	_, err := NewGenBlockPool(10, -1, 0, semaphore.NewWeighted(10), createBlock)

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), fmt.Errorf(invalidConfigError, 10, -1), err)
}

// Represents when block is available on the freeBlocksCh.
func (t *BlockPoolTest) TestTryGetWhenBlockIsAvailableForReuse() {
	bp, err := NewGenBlockPool(1024, 10, 0, semaphore.NewWeighted(10), createBlock)
	require.Nil(t.T(), err)
	// Creating a block with some data and send it to blockCh.
	b, err := createBlock(2)
	require.Nil(t.T(), err)
	bp.freeBlocksCh <- b
	// Setting totalBlocks same as maxBlocks to ensure no new blocks are created.
	bp.totalBlocks = 10

	block, err := bp.TryGet()

	require.Nil(t.T(), err)
	require.NotNil(t.T(), block)
	// This ensures the block is reset.
	assert.Equal(t.T(), int64(0), block.Size())
}

func (t *BlockPoolTest) TestTryGetWhenTotalBlocksIsLessThanThanMaxBlocks() {
	bp, err := NewGenBlockPool(1024, 10, 0, semaphore.NewWeighted(10), createBlock)
	require.Nil(t.T(), err)

	block, err := bp.TryGet()

	require.Nil(t.T(), err)
	require.NotNil(t.T(), block)
	assert.Equal(t.T(), int64(0), block.Size())
}

func (t *BlockPoolTest) TestTryGetToCreateLargeBlock() {
	// Creating block of size 1TB
	bp, err := NewGenBlockPool(1024*1024*1024*1024, 10, 0, semaphore.NewWeighted(10), createBlock)
	require.Nil(t.T(), err)

	_, err = bp.TryGet()

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), "mmap error: cannot allocate memory", err.Error())
}

// Represents when block is available on the freeBlocksCh.
func (t *BlockPoolTest) TestGetWhenBlockIsAvailableForReuse() {
	bp, err := NewGenBlockPool(1024, 10, 0, semaphore.NewWeighted(10), createBlock)
	require.Nil(t.T(), err)
	// Creating a block with some data and send it to blockCh.
	b, err := createBlock(2)
	require.Nil(t.T(), err)
	bp.freeBlocksCh <- b
	// Setting totalBlocks same as maxBlocks to ensure no new blocks are created.
	bp.totalBlocks = 10

	block, err := bp.Get()

	require.Nil(t.T(), err)
	require.NotNil(t.T(), block)
	// This ensures the block is reset.
	assert.Equal(t.T(), int64(0), block.Size())
}

func (t *BlockPoolTest) TestGetWhenTotalBlocksIsLessThanThanMaxBlocks() {
	bp, err := NewGenBlockPool(1024, 10, 0, semaphore.NewWeighted(10), createBlock)
	require.Nil(t.T(), err)

	block, err := bp.Get()

	require.Nil(t.T(), err)
	require.NotNil(t.T(), block)
	assert.Equal(t.T(), int64(0), block.Size())
}

func (t *BlockPoolTest) TestCreateBlockWithLargeSize() {
	// Creating block of size 1TB
	bp, err := NewGenBlockPool(1024*1024*1024*1024, 10, 0, semaphore.NewWeighted(10), createBlock)
	require.Nil(t.T(), err)

	_, err = bp.Get()

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), "mmap error: cannot allocate memory", err.Error())
}

func (t *BlockPoolTest) TestBlockSize() {
	bp, err := NewGenBlockPool(1024, 10, 0, semaphore.NewWeighted(10), createBlock)

	require.Nil(t.T(), err)
	require.Equal(t.T(), int64(1024), bp.BlockSize())
}

func (t *BlockPoolTest) TestClearFreeBlockChannelWithReleaseReservedBlocksTrue() {
	tests := []struct {
		name            string
		reservedBlocks  int64
		performGetBlock int
	}{
		{
			name:           "with_0_reserved_blocks",
			reservedBlocks: 0,
		},
		{
			name:           "with_1_reserved_blocks",
			reservedBlocks: 1,
		},
		{
			name:           "with_2_reserved_blocks",
			reservedBlocks: 2,
		},
		{
			name:           "with_3_reserved_blocks",
			reservedBlocks: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func() {
			bp, err := NewGenBlockPool(1024, 4, tt.reservedBlocks, semaphore.NewWeighted(4), createBlock)
			require.Nil(t.T(), err)
			blocks := make([]Block, 4)
			for i := range 4 {
				blocks[i] = t.validateGetBlockIsNotBlocked(bp)
			}
			// Adding all blocks to freeBlocksCh
			for i := range 4 {
				bp.freeBlocksCh <- blocks[i]
			}
			require.Equal(t.T(), int64(4), bp.totalBlocks)

			err = bp.ClearFreeBlockChannel(true)

			require.Nil(t.T(), err)
			require.EqualValues(t.T(), 0, bp.totalBlocks)
			for i := range 4 {
				require.Nil(t.T(), blocks[i].(*memoryBlock).buffer)
			}
			// All 4 semaphore slots should be available to acquire.
			require.True(t.T(), bp.globalMaxBlocksSem.TryAcquire(4))
			require.False(t.T(), bp.globalMaxBlocksSem.TryAcquire(1))
		})
	}
}

func (t *BlockPoolTest) TestClearFreeBlockChannelWithReleaseReservedBlocksFalse() {
	tests := []struct {
		name                   string
		releaseReservedBlocks  bool
		reservedBlocks         int64
		possibleSemaphoreSlots int64
	}{
		{
			name:                   "with_0_reserved_blocks",
			reservedBlocks:         0,
			possibleSemaphoreSlots: 4,
		},
		{
			name:                   "with_1_reserved_blocks",
			reservedBlocks:         1,
			possibleSemaphoreSlots: 3, // 4 - 1 reserved
		},
		{
			name:                   "with_2_reserved_blocks",
			reservedBlocks:         2,
			possibleSemaphoreSlots: 2, // 4 - 2 reserved
		},
		{
			name:                   "all_4_reserved_blocks",
			reservedBlocks:         4,
			possibleSemaphoreSlots: 0, // 4 - 4 reserved
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func() {
			bp, err := NewGenBlockPool(1024, 4, tt.reservedBlocks, semaphore.NewWeighted(4), createBlock)
			require.Nil(t.T(), err)
			blocks := make([]Block, 4)
			for i := range 4 {
				blocks[i] = t.validateGetBlockIsNotBlocked(bp)
			}
			// Adding all blocks to freeBlocksCh
			for i := range 4 {
				bp.freeBlocksCh <- blocks[i]
			}
			require.Equal(t.T(), int64(4), bp.totalBlocks)

			err = bp.ClearFreeBlockChannel(false)

			require.Nil(t.T(), err)
			require.EqualValues(t.T(), 0, bp.totalBlocks)
			for i := range 4 {
				require.Nil(t.T(), blocks[i].(*memoryBlock).buffer)
			}
			// Only reserved blocks semaphore slots should be available to acquire.
			require.True(t.T(), bp.globalMaxBlocksSem.TryAcquire(tt.possibleSemaphoreSlots))
			require.False(t.T(), bp.globalMaxBlocksSem.TryAcquire(1))
		})
	}
}

func (t *BlockPoolTest) TestClearFreeBlockChannelWhenTotalBlocksIsZero() {
	bp, err := NewGenBlockPool(1024, 10, 0, semaphore.NewWeighted(1), createBlock)
	require.Nil(t.T(), err)
	require.Equal(t.T(), int64(0), bp.totalBlocks)

	err = bp.ClearFreeBlockChannel(true)

	require.Nil(t.T(), err)
	require.Equal(t.T(), int64(0), bp.totalBlocks)
	// Check if semaphore is released correctly.
	require.True(t.T(), bp.globalMaxBlocksSem.TryAcquire(1))
	require.False(t.T(), bp.globalMaxBlocksSem.TryAcquire(1))
}

func (t *BlockPoolTest) TestBlockPoolCreationAcquiresGlobalSem() {
	globalBlocksSem := semaphore.NewWeighted(1)

	bp, err := NewGenBlockPool(1024, 3, 1, globalBlocksSem, createBlock)

	require.Nil(t.T(), err)
	// Validate that semaphore got acquired.
	acquired := globalBlocksSem.TryAcquire(1)
	assert.False(t.T(), acquired)
	// Validate that 1st block can be created as it was reserved.
	b1, err := bp.Get()
	require.Nil(t.T(), err)
	require.NotNil(t.T(), b1)
	// Validate that adding block to freeBlocksCh and clearing it up releases the semaphore
	bp.freeBlocksCh <- b1
	require.Equal(t.T(), int64(1), bp.totalBlocks)
	err = bp.ClearFreeBlockChannel(true)
	require.Nil(t.T(), err)
	require.Equal(t.T(), int64(0), bp.totalBlocks)
	require.Nil(t.T(), b1.(*memoryBlock).buffer)
	// Validate that semaphore can be acquired now.
	acquired = globalBlocksSem.TryAcquire(1)
	assert.True(t.T(), acquired)
}

func (t *BlockPoolTest) TestClearFreeBlockChannelWithMultipleBlockPools() {
	globalMaxBlocksSem := semaphore.NewWeighted(3)
	bp1, err := NewGenBlockPool(1024, 3, 1, globalMaxBlocksSem, createBlock)
	require.Nil(t.T(), err)
	bp2, err := NewGenBlockPool(1024, 3, 1, globalMaxBlocksSem, createBlock)
	require.Nil(t.T(), err)
	// Create 2 blocks in bp1.
	b1 := t.validateGetBlockIsNotBlocked(bp1)
	b2 := t.validateGetBlockIsNotBlocked(bp1)
	require.Equal(t.T(), int64(2), bp1.totalBlocks)
	// Create 1 block in bp2.
	b3 := t.validateGetBlockIsNotBlocked(bp2)
	require.Equal(t.T(), int64(1), bp2.totalBlocks)
	// Freeing up bp1.
	bp1.freeBlocksCh <- b1
	bp1.freeBlocksCh <- b2
	err = bp1.ClearFreeBlockChannel(true)
	require.Nil(t.T(), err)
	require.Nil(t.T(), b1.(*memoryBlock).buffer)
	require.Nil(t.T(), b2.(*memoryBlock).buffer)

	// After bp1 is freed up, 1 more block can be created in bp2.
	b4 := t.validateGetBlockIsNotBlocked(bp2)
	require.Equal(t.T(), int64(2), bp2.totalBlocks)

	// Freeing up bp2.
	bp2.freeBlocksCh <- b3
	bp2.freeBlocksCh <- b4
	err = bp2.ClearFreeBlockChannel(true)
	require.Nil(t.T(), err)
	require.Nil(t.T(), b3.(*memoryBlock).buffer)
	require.Nil(t.T(), b4.(*memoryBlock).buffer)
}

func (t *BlockPoolTest) TestBlockPoolCreationFailsWhenGlobalMaxBlocksIsZero() {
	bp, err := NewGenBlockPool(1024, 10, 1, semaphore.NewWeighted(0), createBlock)

	require.Error(t.T(), err)
	assert.Nil(t.T(), bp)
	assert.ErrorContains(t.T(), err, CantAllocateAnyBlockError.Error())
}

func (t *BlockPoolTest) TestTryGetWhenLimitedByGlobalBlocks() {
	bp, err := NewGenBlockPool(1024, 10, 1, semaphore.NewWeighted(2), createBlock)
	require.Nil(t.T(), err)
	// 2 blocks can be created.
	b1, err1 := bp.TryGet()
	require.Nil(t.T(), err1)
	require.NotNil(t.T(), b1)
	b2, err2 := bp.TryGet()
	require.Nil(t.T(), err2)
	require.NotNil(t.T(), b2)

	b3, err3 := bp.TryGet()

	require.Nil(t.T(), b3)
	require.NotNil(t.T(), err3)
	require.ErrorIs(t.T(), err3, CantAllocateAnyBlockError)
	require.Equal(t.T(), int64(2), bp.totalBlocks)
}

func (t *BlockPoolTest) TestTryGetWhenTotalBlocksEqualToMaxBlocks() {
	bp, err := NewGenBlockPool(1024, 10, 0, semaphore.NewWeighted(10), createBlock)
	require.Nil(t.T(), err)
	bp.totalBlocks = 10

	b, err := bp.TryGet()

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), CantAllocateAnyBlockError, err)
	require.Nil(t.T(), b)
}

func (t *BlockPoolTest) TestGetWhenLimitedByGlobalBlocks() {
	bp, err := NewGenBlockPool(1024, 10, 1, semaphore.NewWeighted(2), createBlock)
	require.Nil(t.T(), err)

	// 2 blocks can be created.
	for range 2 {
		_ = t.validateGetBlockIsNotBlocked(bp)
	}
	require.Equal(t.T(), int64(2), bp.totalBlocks)

	t.validateGetBlockIsBlocked(bp)
}

func (t *BlockPoolTest) TestGetWhenTotalBlocksEqualToMaxBlocks() {
	bp, err := NewGenBlockPool(1024, 10, 0, semaphore.NewWeighted(10), createBlock)
	require.Nil(t.T(), err)
	bp.totalBlocks = 10

	t.validateGetBlockIsBlocked(bp)
}

func (t *BlockPoolTest) validateGetBlockIsBlocked(bp *GenBlockPool[Block]) {
	t.T().Helper()
	done := make(chan bool, 1)
	go func() {
		b, err := bp.Get()
		require.Nil(t.T(), err)
		require.NotNil(t.T(), b)
		done <- true
	}()

	select {
	case <-done:
		assert.FailNow(t.T(), "Able to get/create a block when it is not allowed")
	case <-time.After(1 * time.Second):
	}
}

func (t *BlockPoolTest) validateGetBlockIsNotBlocked(bp *GenBlockPool[Block]) Block {
	t.T().Helper()
	done := make(chan Block, 1)
	go func() {
		b, err := bp.Get()
		require.Nil(t.T(), err)
		require.NotNil(t.T(), b)
		done <- b
	}()

	select {
	case block := <-done:
		return block
	case <-time.After(1 * time.Second):
		assert.FailNow(t.T(), "Not able to get/create a block")
		return nil
	}
}

func (t *BlockPoolTest) TestCanAllocateBlock() {
	tests := []struct {
		name           string
		maxBlocks      int64
		totalBlocks    int64
		reservedBlocks int64
		globalSem      *semaphore.Weighted
		expected       bool
	}{
		{
			name:           "max_blocks_reached",
			maxBlocks:      10,
			totalBlocks:    10,
			reservedBlocks: 0,
			globalSem:      semaphore.NewWeighted(0),
			expected:       false,
		},
		{
			name:           "first_block",
			maxBlocks:      10,
			totalBlocks:    0,
			reservedBlocks: 0,
			globalSem:      semaphore.NewWeighted(1),
			expected:       true,
		},
		{
			name:           "semaphore_acquirable",
			maxBlocks:      10,
			totalBlocks:    5,
			reservedBlocks: 0,
			globalSem:      semaphore.NewWeighted(1),
			expected:       true,
		},
		{
			name:           "semaphore_not_acquirable",
			maxBlocks:      10,
			totalBlocks:    5,
			reservedBlocks: 0,
			globalSem:      semaphore.NewWeighted(0),
			expected:       false,
		},
		{
			name:           "equal_max_blocks_and_total_blocks_0",
			maxBlocks:      0,
			totalBlocks:    0,
			reservedBlocks: 0,
			globalSem:      semaphore.NewWeighted(0),
			expected:       false,
		},
		{
			name:           "total_blocks_more_than_max_blocks",
			maxBlocks:      0,
			totalBlocks:    1,
			reservedBlocks: 0,
			globalSem:      semaphore.NewWeighted(0),
			expected:       false,
		},
		{
			name:           "reserved_blocks_equal_to_max_blocks",
			maxBlocks:      10,
			totalBlocks:    0,
			reservedBlocks: 10,
			globalSem:      semaphore.NewWeighted(10),
			expected:       true,
		},
		{
			name:           "reserved_blocks_less_than_max_blocks",
			maxBlocks:      10,
			totalBlocks:    0,
			reservedBlocks: 5,
			globalSem:      semaphore.NewWeighted(10),
			expected:       true,
		},
		{
			name:           "reserved_blocks_equal_to_total_blocks",
			maxBlocks:      10,
			totalBlocks:    5,
			reservedBlocks: 5,
			globalSem:      semaphore.NewWeighted(0),
			expected:       false,
		},
		{
			name:           "reserved_blocks_less_than_total_blocks",
			maxBlocks:      10,
			totalBlocks:    6,
			reservedBlocks: 5,
			globalSem:      semaphore.NewWeighted(0),
			expected:       false,
		},
		{
			name:           "reserved_blocks_more_than_total_blocks",
			maxBlocks:      10,
			totalBlocks:    4,
			reservedBlocks: 5,
			globalSem:      semaphore.NewWeighted(0),
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func() {
			bp := &GenBlockPool[Block]{
				maxBlocks:          tt.maxBlocks,
				totalBlocks:        tt.totalBlocks,
				globalMaxBlocksSem: tt.globalSem,
			}

			got := bp.canAllocateBlock()

			assert.Equal(t.T(), tt.expected, got)
		})
	}
}

func (t *BlockPoolTest) TestBlockPoolCreationWithReservedBlocksSuccess() {
	tests := []struct {
		name           string
		reservedBlocks int64
		maxBlocks      int64
	}{
		{
			name:           "zero_reserved_blocks",
			reservedBlocks: 0,
			maxBlocks:      10,
		},
		{
			name:           "one_reserved_block",
			reservedBlocks: 1,
			maxBlocks:      10,
		},
		{
			name:           "two_reserved_blocks",
			reservedBlocks: 2,
			maxBlocks:      10,
		},
		{
			name:           "max_blocks_equal_to_reserved_blocks",
			reservedBlocks: 10,
			maxBlocks:      10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func() {
			bp, err := NewGenBlockPool(1024, 10, tt.reservedBlocks, semaphore.NewWeighted(20), createBlock)

			require.NoError(t.T(), err)
			require.NotNil(t.T(), bp)
		})
	}
}

func (t *BlockPoolTest) TestBlockPoolCreationWithReservedBlocksFailure() {
	tests := []struct {
		name            string
		reservedBlocks  int64
		maxBlocks       int64
		globalMaxBlocks int64
		expectedErrMsg  string
	}{
		{
			name:            "reserved_blocks_greater_than_max_blocks",
			reservedBlocks:  11,
			maxBlocks:       10,
			globalMaxBlocks: 20,
			expectedErrMsg:  "invalid reserved blocks count: 11, it should be between 0 and maxBlocks: 10",
		},
		{
			name:            "negative_reserved_blocks",
			reservedBlocks:  -1,
			maxBlocks:       10,
			globalMaxBlocks: 20,
			expectedErrMsg:  "invalid reserved blocks count: -1, it should be between 0 and maxBlocks: 10",
		},
		{
			name:            "reserved_blocks_greater_than_global_max_blocks",
			reservedBlocks:  7,
			maxBlocks:       7,
			globalMaxBlocks: 6,
			expectedErrMsg:  "cant allocate any block as global max blocks limit is reached",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func() {
			bp, err := NewGenBlockPool(1024, 10, tt.reservedBlocks, semaphore.NewWeighted(tt.globalMaxBlocks), createBlock)

			require.Error(t.T(), err)
			assert.Nil(t.T(), bp)
			assert.EqualError(t.T(), err, tt.expectedErrMsg)
		})
	}
}
