// Copyright 2025 Google LLC
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
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/semaphore"
)

// MockGenBlock is a mock implementation of GenBlock for testing purposes.
type MockGenBlock struct {
	// Add any fields you need for your mock implementation
}

func (m *MockGenBlock) Reuse() {
	// Implement mock behavior
}

func (m *MockGenBlock) Deallocate() error {
	// Implement mock behavior
	return nil
}

// TestNewGenericBlockPool tests the creation of a GenericBlockPool.
func TestNewGenericBlockPool(t *testing.T) {
	blockSize := int64(1024)
	maxBlocks := int64(10)
	globalMaxBlocksSem := semaphore.NewWeighted(10)
	createBlockFunc := func(blockSize int64) (*MockGenBlock, error) {
		return &MockGenBlock{}, nil
	}

	bp, err := NewGenericBlockPool(blockSize, maxBlocks, globalMaxBlocksSem, createBlockFunc)

	assert.NoError(t, err)
	assert.NotNil(t, bp)
	assert.Equal(t, blockSize, bp.blockSize)
	assert.Equal(t, maxBlocks, bp.maxBlocks)
	assert.Equal(t, int64(0), bp.totalBlocks)
}

// TestGetExistingBlock tests getting an existing block from the pool.
func TestGetExistingBlock(t *testing.T) {
	blockSize := int64(1024)
	maxBlocks := int64(10)
	globalMaxBlocksSem := semaphore.NewWeighted(10)
	createBlockFunc := func(blockSize int64) (*MockGenBlock, error) {
		return &MockGenBlock{}, nil
	}
	bp, err := NewGenericBlockPool(blockSize, maxBlocks, globalMaxBlocksSem, createBlockFunc)
	assert.NoError(t, err)

	block, err := bp.Get()

	assert.NoError(t, err)
	assert.NotNil(t, block)
	// Releasing the block back to the pool
	bp.Release(block)
}

// TestCanAllocateBlock tests the canAllocateBlock method.
func TestCanAllocateBlockTrue(t *testing.T) {
	blockSize := int64(1024)
	maxBlocks := int64(10)
	globalMaxBlocksSem := semaphore.NewWeighted(10)
	createBlockFunc := func(blockSize int64) (*MockGenBlock, error) {
		return &MockGenBlock{}, nil
	}
	bp, err := NewGenericBlockPool(blockSize, maxBlocks, globalMaxBlocksSem, createBlockFunc)
	assert.NoError(t, err)

	assert.True(t, bp.canAllocateBlock())
}

// TestCanAllocateBlock tests the canAllocateBlock method.
func TestCanAllocateBlockFalse(t *testing.T) {
	blockSize := int64(1024)
	maxBlocks := int64(10)
	globalMaxBlocksSem := semaphore.NewWeighted(10)
	createBlockFunc := func(blockSize int64) (*MockGenBlock, error) {
		return &MockGenBlock{}, nil
	}
	bp, err := NewGenericBlockPool(blockSize, maxBlocks, globalMaxBlocksSem, createBlockFunc)
	assert.NoError(t, err)
	// Fill the pool to its max capacity
	for i := int64(0); i < maxBlocks; i++ {
		block, err := bp.Get()
		assert.NoError(t, err)
		assert.NotNil(t, block)
	}

	// Now we should not be able to allocate any more blocks
	assert.False(t, bp.canAllocateBlock())
}

// TestReleaseBlock tests the Release method of the GenericBlockPool.
func TestReleaseBlock(t *testing.T) {
	blockSize := int64(1024)
	maxBlocks := int64(10)
	globalMaxBlocksSem := semaphore.NewWeighted(10)
	createBlockFunc := func(blockSize int64) (*MockGenBlock, error) {
		return &MockGenBlock{}, nil
	}
	bp, err := NewGenericBlockPool(blockSize, maxBlocks, globalMaxBlocksSem, createBlockFunc)
	assert.NoError(t, err)
	block, err := bp.Get()
	assert.NoError(t, err)

	// Release the block back to the pool
	bp.Release(block)

	// Now we should be able to get the same block again
	retrievedBlock, err := bp.Get()
	assert.NoError(t, err)
	assert.Equal(t, block, retrievedBlock)
}

// TestReleaseBlockWhenFull tests releasing a block when the pool is full.
func TestReleaseBlockWhenFull(t *testing.T) {
	blockSize := int64(1024)
	maxBlocks := int64(10)
	globalMaxBlocksSem := semaphore.NewWeighted(10)
	createBlockFunc := func(blockSize int64) (*MockGenBlock, error) {
		return &MockGenBlock{}, nil
	}
	bp, err := NewGenericBlockPool(blockSize, maxBlocks, globalMaxBlocksSem, createBlockFunc)
	assert.NoError(t, err)
	// Fill the pool to its max capacity
	// create a block slice
	blocks := make([]*MockGenBlock, maxBlocks)
	for i := int64(0); i < maxBlocks; i++ {
		block, err := bp.Get()
		blocks[i] = block
		assert.NoError(t, err)
		assert.NotNil(t, block)
	}
	for i := int64(0); i < maxBlocks; i++ {
		bp.Release(blocks[i])
	}

	// Attempt to release a block when the pool is full
	assert.Panics(t, func() {
		bp.Release(&MockGenBlock{})
	})
}

// TestClearFreeBlockChannel tests clearing the free blocks channel.
func TestClearFreeBlockChannel(t *testing.T) {
	blockSize := int64(1024)
	maxBlocks := int64(10)
	globalMaxBlocksSem := semaphore.NewWeighted(10)
	createBlockFunc := func(blockSize int64) (*MockGenBlock, error) {
		return &MockGenBlock{}, nil
	}
	bp, err := NewGenericBlockPool(blockSize, maxBlocks, globalMaxBlocksSem, createBlockFunc)
	assert.NoError(t, err)

	// Fill the pool to its max capacity
	for i := int64(0); i < maxBlocks; i++ {
		block, err := bp.Get()
		assert.NoError(t, err)
		assert.NotNil(t, block)
	}

	// Clear the free blocks channel
	bp.ClearFreeBlockChannel(true)
}

func TestNewGenericBlockPoolWithblock(t *testing.T) {
	blockSize := int64(1024)
	maxBlocks := int64(10)
	globalMaxBlocksSem := semaphore.NewWeighted(10)
	createBlockFunc := func(blockSize int64) (Block, error) {
		return createBlock(blockSize)
	}

	bp, err := NewGenericBlockPool(blockSize, maxBlocks, globalMaxBlocksSem, createBlockFunc)

	assert.NoError(t, err)
	assert.NotNil(t, bp)
	assert.Equal(t, blockSize, bp.blockSize)
	assert.Equal(t, maxBlocks, bp.maxBlocks)
	assert.Equal(t, int64(0), bp.totalBlocks)
}
