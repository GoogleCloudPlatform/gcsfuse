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

package prefetch

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type blockpoolTestSuite struct {
	suite.Suite
	assert *assert.Assertions
}

func (suite *blockpoolTestSuite) SetupTest() {
}

func (suite *blockpoolTestSuite) cleanupTest() {
}

func validateNullData(b *Block) bool {
	for i := 0; i < len(b.data); i++ {
		if b.data[i] != 0 {
			return false
		}
	}

	return true
}

func (suite *blockpoolTestSuite) TestAllocate() {
	suite.assert = assert.New(suite.T())

	bp := NewBlockPool(0, 0)
	suite.assert.Nil(bp)

	bp = NewBlockPool(1, 0)
	suite.assert.Nil(bp)

	bp = NewBlockPool(1, 1)
	suite.assert.NotNil(bp)
	suite.assert.NotNil(bp.blocksCh)
	suite.assert.NotNil(bp.resetBlockCh)
	suite.assert.NotNil(bp.zeroBlock)
	suite.assert.True(validateNullData(bp.zeroBlock))

	bp.Terminate()
	suite.assert.Equal(len(bp.blocksCh), 0)
	suite.assert.Equal(len(bp.resetBlockCh), 0)
	suite.assert.Equal(len(bp.zeroBlock.data), 0)
}

func (suite *blockpoolTestSuite) TestGetRelease() {
	suite.assert = assert.New(suite.T())

	bp := NewBlockPool(1, 5)
	suite.assert.NotNil(bp)
	suite.assert.NotNil(bp.blocksCh)
	suite.assert.NotNil(bp.resetBlockCh)
	suite.assert.NotNil(bp.zeroBlock)
	suite.assert.Equal(len(bp.blocksCh), 4)
	suite.assert.Equal(len(bp.resetBlockCh), 0)
	suite.assert.True(validateNullData(bp.zeroBlock))

	b := bp.MustGet()
	suite.assert.NotNil(b)
	suite.assert.Equal(len(bp.blocksCh), 3)

	bp.Release(b)
	time.Sleep(1 * time.Second)
	suite.assert.Equal(len(bp.blocksCh), 4)

	b = bp.TryGet()
	suite.assert.NotNil(b)
	suite.assert.Equal(len(bp.blocksCh), 3)

	bp.Release(b)
	time.Sleep(1 * time.Second)
	suite.assert.Equal(len(bp.blocksCh), 4)

	bp.Terminate()
	suite.assert.Equal(len(bp.blocksCh), 0)
	suite.assert.Equal(len(bp.resetBlockCh), 0)
	suite.assert.Equal(len(bp.zeroBlock.data), 0)
}

func (suite *blockpoolTestSuite) TestUsage() {
	suite.assert = assert.New(suite.T())

	bp := NewBlockPool(1, 5)
	suite.assert.NotNil(bp)
	suite.assert.NotNil(bp.blocksCh)
	suite.assert.NotNil(bp.resetBlockCh)
	suite.assert.NotNil(bp.zeroBlock)
	suite.assert.Equal(len(bp.blocksCh), 4)
	suite.assert.Equal(len(bp.resetBlockCh), 0)
	suite.assert.True(validateNullData(bp.zeroBlock))

	var blocks []*Block
	b := bp.MustGet()
	suite.assert.NotNil(b)
	blocks = append(blocks, b)

	usage := bp.Usage()
	suite.assert.Equal(usage, uint32(40))

	b = bp.TryGet()
	suite.assert.NotNil(b)
	blocks = append(blocks, b)

	usage = bp.Usage()
	suite.assert.Equal(usage, uint32(60))

	for _, blk := range blocks {
		bp.Release(blk)
	}

	// adding wait for the blocks to be reset and pushed back to the blocks channel
	time.Sleep(2 * time.Second)

	usage = bp.Usage()
	suite.assert.Equal(usage, uint32(20)) // because of zeroBlock

	bp.Terminate()
	suite.assert.Equal(len(bp.blocksCh), 0)
	suite.assert.Equal(len(bp.resetBlockCh), 0)
	suite.assert.Equal(len(bp.zeroBlock.data), 0)
}

func (suite *blockpoolTestSuite) TestBufferExhaution() {
	suite.assert = assert.New(suite.T())

	bp := NewBlockPool(1, 5)
	suite.assert.NotNil(bp)
	suite.assert.NotNil(bp.blocksCh)
	suite.assert.NotNil(bp.resetBlockCh)
	suite.assert.NotNil(bp.zeroBlock)
	suite.assert.Equal(len(bp.blocksCh), 4)
	suite.assert.Equal(len(bp.resetBlockCh), 0)
	suite.assert.True(validateNullData(bp.zeroBlock))

	var blocks []*Block
	for i := 0; i < 5; i++ {
		b := bp.MustGet()
		suite.assert.NotNil(b)
		blocks = append(blocks, b)
	}

	usage := bp.Usage()
	suite.assert.Equal(usage, uint32(100))

	b := bp.TryGet()
	suite.assert.Nil(b)

	b = bp.MustGet()
	suite.assert.NotNil(b)
	blocks = append(blocks, b)

	for _, blk := range blocks {
		bp.Release(blk)
	}

	bp.Terminate()
	suite.assert.Equal(len(bp.blocksCh), 0)
	suite.assert.Equal(len(bp.resetBlockCh), 0)
	suite.assert.Equal(len(bp.zeroBlock.data), 0)
}

// get n blocks
func getBlocks(suite *blockpoolTestSuite, bp *BlockPool, n int) []*Block {
	var blocks []*Block
	for i := 0; i < n; i++ {
		b := bp.TryGet()
		suite.assert.NotNil(b)

		// validate that the block has null data
		suite.assert.True(validateNullData(b))
		blocks = append(blocks, b)
	}
	return blocks
}

func releaseBlocks(suite *blockpoolTestSuite, bp *BlockPool, blocks []*Block) {
	for _, b := range blocks {
		b.data[0] = byte(rand.Int()%100 + 1)
		b.data[1] = byte(rand.Int()%100 + 1)

		// validate that the block being released does not have null data
		suite.assert.False(validateNullData(b))
		bp.Release(b)
	}
}

func (suite *blockpoolTestSuite) TestBlockReset() {
	suite.assert = assert.New(suite.T())

	bp := NewBlockPool(2, 10)
	suite.assert.NotNil(bp)
	suite.assert.NotNil(bp.blocksCh)
	suite.assert.NotNil(bp.resetBlockCh)
	suite.assert.NotNil(bp.zeroBlock)
	suite.assert.Equal(len(bp.blocksCh), 4)
	suite.assert.Equal(len(bp.resetBlockCh), 0)
	suite.assert.True(validateNullData(bp.zeroBlock))

	blocks := getBlocks(suite, bp, 4)

	releaseBlocks(suite, bp, blocks)

	// adding wait for the blocks to be reset and pushed back to the blocks channel
	time.Sleep(2 * time.Second)

	blocks = getBlocks(suite, bp, 4)

	releaseBlocks(suite, bp, blocks)

	bp.Terminate()
	suite.assert.Equal(len(bp.blocksCh), 0)
	suite.assert.Equal(len(bp.resetBlockCh), 0)
	suite.assert.Equal(len(bp.zeroBlock.data), 0)
}

func TestBlockPoolSuite(t *testing.T) {
	suite.Run(t, new(blockpoolTestSuite))
}
