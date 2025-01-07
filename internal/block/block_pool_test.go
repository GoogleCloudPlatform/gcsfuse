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
	"io"
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
	bp, err := NewBlockPool(1024, 10, semaphore.NewWeighted(10))

	require.Nil(t.T(), err)
	require.NotNil(t.T(), bp)
	assert.Equal(t.T(), int64(1024), bp.blockSize)
	assert.Equal(t.T(), int64(10), bp.maxBlocks)
	assert.Equal(t.T(), int64(0), bp.totalBlocks)
}

func (t *BlockPoolTest) TestInitBlockPoolForZeroBlockSize() {
	_, err := NewBlockPool(0, 10, semaphore.NewWeighted(10))

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), fmt.Errorf(invalidConfigError, 0, 10), err)
}

func (t *BlockPoolTest) TestInitBlockPoolForNegativeBlockSize() {
	_, err := NewBlockPool(-1, 10, semaphore.NewWeighted(10))

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), fmt.Errorf(invalidConfigError, -1, 10), err)
}

func (t *BlockPoolTest) TestInitBlockPoolForZeroMaxBlocks() {
	_, err := NewBlockPool(10, 0, semaphore.NewWeighted(10))

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), fmt.Errorf(invalidConfigError, 10, 0), err)
}

func (t *BlockPoolTest) TestInitBlockPoolForNegativeMaxBlocks() {
	_, err := NewBlockPool(10, -1, semaphore.NewWeighted(10))

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), fmt.Errorf(invalidConfigError, 10, -1), err)
}

// Represents when block is available on the freeBlocksCh.
func (t *BlockPoolTest) TestGetWhenBlockIsAvailableForReuse() {
	bp, err := NewBlockPool(1024, 10, semaphore.NewWeighted(10))
	require.Nil(t.T(), err)
	// Creating a block with some data and send it to blockCh.
	b, err := createBlock(2)
	require.Nil(t.T(), err)
	content := []byte("hi")
	err = b.Write(content)
	require.Nil(t.T(), err)
	// Validating the content of the block
	output, err := io.ReadAll(b.Reader())
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), content, output)
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
	bp, err := NewBlockPool(1024, 10, semaphore.NewWeighted(10))
	require.Nil(t.T(), err)

	block, err := bp.Get()

	require.Nil(t.T(), err)
	require.NotNil(t.T(), block)
	assert.Equal(t.T(), int64(0), block.Size())
}

func (t *BlockPoolTest) TestCreateBlockWithLargeSize() {
	// Creating block of size 1TB
	bp, err := NewBlockPool(1024*1024*1024*1024, 10, semaphore.NewWeighted(10))
	require.Nil(t.T(), err)

	_, err = bp.Get()

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), "mmap error: cannot allocate memory", err.Error())
}

func (t *BlockPoolTest) TestBlockSize() {
	bp, err := NewBlockPool(1024, 10, semaphore.NewWeighted(10))

	require.Nil(t.T(), err)
	require.Equal(t.T(), int64(1024), bp.BlockSize())
}

func (t *BlockPoolTest) TestClearFreeBlockChannel() {
	bp, err := NewBlockPool(1024, 10, semaphore.NewWeighted(3))
	require.Nil(t.T(), err)
	blocks := make([]Block, 4)
	for i := 0; i < 4; i++ {
		blocks[i] = t.validateGetBlockIsNotBlocked(bp)
	}
	// Adding 2 blocks to freeBlocksCh
	bp.freeBlocksCh <- blocks[0]
	bp.freeBlocksCh <- blocks[1]
	require.Equal(t.T(), int64(4), bp.totalBlocks)

	err = bp.ClearFreeBlockChannel()

	require.Nil(t.T(), err)
	require.Equal(t.T(), int64(2), bp.totalBlocks)
	require.Nil(t.T(), blocks[0].(*memoryBlock).buffer)
	require.Nil(t.T(), blocks[1].(*memoryBlock).buffer)
	require.NotNil(t.T(), blocks[2].(*memoryBlock).buffer)
	require.NotNil(t.T(), blocks[3].(*memoryBlock).buffer)
	// Check if semaphore is released correctly.
	require.True(t.T(), bp.globalMaxBlocksSem.TryAcquire(2))
	require.False(t.T(), bp.globalMaxBlocksSem.TryAcquire(1))
}

func (t *BlockPoolTest) TestFirstBlockIsCreatedWithoutAcquiringGlobalSem() {
	bp, err := NewBlockPool(1024, 3, semaphore.NewWeighted(0))
	require.Nil(t.T(), err)
	b1, err := bp.Get()
	require.Nil(t.T(), err)
	require.NotNil(t.T(), b1)
	// Adding block to freeBlocksCh
	bp.freeBlocksCh <- b1
	require.Equal(t.T(), int64(1), bp.totalBlocks)

	err = bp.ClearFreeBlockChannel()

	require.Nil(t.T(), err)
	require.Equal(t.T(), int64(0), bp.totalBlocks)
	require.Nil(t.T(), b1.(*memoryBlock).buffer)
}

func (t *BlockPoolTest) TestClearFreeBlockChannelWithMultipleBlockPools() {
	globalMaxBlocksSem := semaphore.NewWeighted(1)
	bp1, err := NewBlockPool(1024, 3, globalMaxBlocksSem)
	require.Nil(t.T(), err)
	bp2, err := NewBlockPool(1024, 3, globalMaxBlocksSem)
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
	err = bp1.ClearFreeBlockChannel()
	require.Nil(t.T(), err)
	require.Nil(t.T(), b1.(*memoryBlock).buffer)
	require.Nil(t.T(), b2.(*memoryBlock).buffer)

	// After bp1 is freed up, 1 more block can be created in bp2.
	b4 := t.validateGetBlockIsNotBlocked(bp2)
	require.Equal(t.T(), int64(2), bp2.totalBlocks)

	// Freeing up bp2.
	bp2.freeBlocksCh <- b3
	bp2.freeBlocksCh <- b4
	err = bp2.ClearFreeBlockChannel()
	require.Nil(t.T(), err)
	require.Nil(t.T(), b3.(*memoryBlock).buffer)
	require.Nil(t.T(), b4.(*memoryBlock).buffer)
}

func (t *BlockPoolTest) TestGetWhenGlobalMaxBlocksIsZero() {
	bp, err := NewBlockPool(1024, 10, semaphore.NewWeighted(0))
	require.Nil(t.T(), err)

	// First block is allowed even with globalMaxBlocks being zero.
	b1, err := bp.Get()
	require.Nil(t.T(), err)
	require.NotNil(t.T(), b1)
	// We shouldn't be allowed to create another block.
	t.validateGetBlockIsBlocked(bp)
}

func (t *BlockPoolTest) TestGetWhenLimitedByGlobalBlocks() {
	bp, err := NewBlockPool(1024, 10, semaphore.NewWeighted(2))
	require.Nil(t.T(), err)

	// 3 blocks can be created.
	for i := 0; i < 3; i++ {
		_ = t.validateGetBlockIsNotBlocked(bp)
	}
	require.Equal(t.T(), int64(3), bp.totalBlocks)

	t.validateGetBlockIsBlocked(bp)
}

func (t *BlockPoolTest) TestGetWhenTotalBlocksEqualToMaxBlocks() {
	bp, err := NewBlockPool(1024, 10, semaphore.NewWeighted(2))
	require.Nil(t.T(), err)
	bp.totalBlocks = 10

	t.validateGetBlockIsBlocked(bp)
}

func (t *BlockPoolTest) validateGetBlockIsBlocked(bp *BlockPool) {
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

func (t *BlockPoolTest) validateGetBlockIsNotBlocked(bp *BlockPool) Block {
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

func (t *BlockPoolTest) TestBlockPool_FreeBlocksChannel() {
	freeBlocksCh := make(chan Block)
	bp := &BlockPool{
		freeBlocksCh: freeBlocksCh,
	}

	ch := bp.FreeBlocksChannel()

	assert.NotNil(t.T(), ch)
	assert.Equal(t.T(), freeBlocksCh, ch)
}

func (t *BlockPoolTest) TestBlockPool_canAllocateBlock() {
	tests := []struct {
		name        string
		maxBlocks   int64
		totalBlocks int64
		globalSem   *semaphore.Weighted
		expected    bool
	}{
		{
			name:        "max_blocks_reached",
			maxBlocks:   10,
			totalBlocks: 10,
			globalSem:   semaphore.NewWeighted(0),
			expected:    false,
		},
		{
			name:        "first_block",
			maxBlocks:   10,
			totalBlocks: 0,
			globalSem:   semaphore.NewWeighted(0),
			expected:    true,
		},
		{
			name:        "semaphore_acquirable",
			maxBlocks:   10,
			totalBlocks: 5,
			globalSem:   semaphore.NewWeighted(1),
			expected:    true,
		},
		{
			name:        "semaphore_not_acquirable",
			maxBlocks:   10,
			totalBlocks: 5,
			globalSem:   semaphore.NewWeighted(0),
			expected:    false,
		},
		{
			name:        "equal_max_blocks_and_total_blocks_0",
			maxBlocks:   0,
			totalBlocks: 0,
			globalSem:   semaphore.NewWeighted(0),
			expected:    false,
		},
		{
			name:        "total_blocks_more_than_max_blocks",
			maxBlocks:   0,
			totalBlocks: 1,
			globalSem:   semaphore.NewWeighted(0),
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func() {
			bp := &BlockPool{
				maxBlocks:          tt.maxBlocks,
				totalBlocks:        tt.totalBlocks,
				globalMaxBlocksSem: tt.globalSem,
			}

			got := bp.canAllocateBlock()

			assert.Equal(t.T(), tt.expected, got)
		})
	}
}
