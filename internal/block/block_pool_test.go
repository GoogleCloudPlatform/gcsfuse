package block

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const invalidConfigError string = "invalid configuration provided for blockPool, blocksize: %d, maxBlocks: %d"

type BlockPoolTest struct {
	suite.Suite
}

func TestBlockPoolTestSuite(t *testing.T) {
	suite.Run(t, new(BlockPoolTest))
}

func (t *BlockPoolTest) TestInitBlockPool() {
	bp, err := InitBlockPool(1024, 10)

	require.Nil(t.T(), err)
	assert.NotNil(t.T(), bp)
	assert.Equal(t.T(), int64(1024), bp.blockSize)
	assert.Equal(t.T(), int32(10), bp.maxBlocks)
	assert.Equal(t.T(), int32(0), bp.totalBlocks)
}

func (t *BlockPoolTest) TestInitBlockPoolForZeroBlockSize() {
	_, err := InitBlockPool(0, 10)

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), fmt.Errorf(invalidConfigError, 0, 10), err)
}

func (t *BlockPoolTest) TestInitBlockPoolForNegativeBlockSize() {
	_, err := InitBlockPool(-1, 10)

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), fmt.Errorf(invalidConfigError, -1, 10), err)
}

func (t *BlockPoolTest) TestInitBlockPoolForZeroMaxBlocks() {
	_, err := InitBlockPool(10, 0)

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), fmt.Errorf(invalidConfigError, 10, 0), err)
}

func (t *BlockPoolTest) TestInitBlockPoolForNegativeMaxBlocks() {
	_, err := InitBlockPool(10, -1)

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), fmt.Errorf(invalidConfigError, 10, -1), err)
}

// Represents when block is available on the blocksCh.
func (t *BlockPoolTest) TestGetWhenBlockIsAvailableForReuse() {
	bp, err := InitBlockPool(1024, 10)
	require.Nil(t.T(), err)
	// Creating a block with some data and send it to blockCh.
	b := createBlock(1024)
	content := []byte("hi")
	err = b.Write(content)
	require.Nil(t.T(), err)
	// Validating the content of the block
	output, err := io.ReadAll(b.Reader())
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), content, output)
	bp.blocksCh <- b
	// Setting totalBlocks same as maxBlocks to ensure no new blocks are created.
	bp.totalBlocks = 10

	block, err := bp.Get()

	require.Nil(t.T(), err)
	require.NotNil(t.T(), block)
	// This ensures the block is reset.
	assert.Equal(t.T(), int64(0), block.Size())
}

func (t *BlockPoolTest) TestGetWhenTotalBlocksIsLessThanThanMaxBlocks() {
	bp, err := InitBlockPool(1024, 10)
	require.Nil(t.T(), err)

	block, err := bp.Get()

	require.Nil(t.T(), err)
	require.NotNil(t.T(), block)
	assert.Equal(t.T(), int64(0), block.Size())
}

func (t *BlockPoolTest) TestCreateBlock() {
	bp, err := InitBlockPool(1024, 10)
	require.Nil(t.T(), err)

	block, err := bp.Get()

	require.Nil(t.T(), err)
	require.NotNil(t.T(), block)
	assert.Equal(t.T(), int64(0), block.Size())
}

func (t *BlockPoolTest) TestCreateBlockWithLargeSize() {
	// Creating block of size 1TB
	bp, err := InitBlockPool(1024*1024*1024*1024, 10)
	require.Nil(t.T(), err)

	_, err = bp.Get()

	require.NotNil(t.T(), err)
	assert.Equal(t.T(), "mmap error: cannot allocate memory", err.Error())
}

func (t *BlockPoolTest) TestBlockSize() {
	bp, err := InitBlockPool(1024, 10)

	require.Nil(t.T(), err)
	require.Equal(t.T(), int64(1024), bp.BlockSize())
}
