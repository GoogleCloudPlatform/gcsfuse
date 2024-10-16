package block

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MemoryBlockTest struct {
	suite.Suite
}

func TestMemoryBlockTestSuite(t *testing.T) {
	suite.Run(t, new(MemoryBlockTest))
}

func createBlock(size uint32) Block {
	mb := memoryBlock{
		buffer: make([]byte, size),
		offset: offset{0, 0},
	}

	return &mb
}

func (testSuite *MemoryBlockTest) TestMemoryBlockWrite() {
	mb := createBlock(12)
	content := []byte("hi")
	err := mb.Write(content)

	assert.Nil(testSuite.T(), err)
	output, err := io.ReadAll(mb.Reader())
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), content, output)
	assert.Equal(testSuite.T(), int64(2), mb.Size())
}

func (testSuite *MemoryBlockTest) TestMemoryBlockWriteWithDataGreaterThanCapacity() {
	mb := createBlock(1)
	content := []byte("hi")
	err := mb.Write(content)

	assert.NotNil(testSuite.T(), err)
	assert.EqualError(testSuite.T(), err, fmt.Sprintf("received data more than capacity of the block"))
}

func (testSuite *MemoryBlockTest) TestMemoryBlockWriteWithMultipleWrites() {
	mb := createBlock(12)
	err := mb.Write([]byte("hi"))
	assert.Nil(testSuite.T(), err)
	err = mb.Write([]byte("hello"))
	assert.Nil(testSuite.T(), err)

	output, err := io.ReadAll(mb.Reader())
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), []byte("hihello"), output)
	assert.Equal(testSuite.T(), int64(7), mb.Size())
}

func (testSuite *MemoryBlockTest) TestMemoryBlockWriteWith2ndWriteBeyondCapacity() {
	mb := createBlock(2)
	content := []byte("hi")
	err := mb.Write(content)
	assert.Nil(testSuite.T(), err)
	err = mb.Write(content)

	assert.NotNil(testSuite.T(), err)
	assert.EqualError(testSuite.T(), err, fmt.Sprintf("received data more than capacity of the block"))
}

func (testSuite *MemoryBlockTest) TestMemoryBlockReuse() {
	mb := createBlock(12)
	content := []byte("hi")
	err := mb.Write(content)
	assert.Nil(testSuite.T(), err)
	output, err := io.ReadAll(mb.Reader())
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), content, output)
	assert.Equal(testSuite.T(), int64(2), mb.Size())

	mb.Reuse()

	output, err = io.ReadAll(mb.Reader())
	assert.Nil(testSuite.T(), err)
	assert.Empty(testSuite.T(), output)
	assert.Equal(testSuite.T(), int64(0), mb.Size())
}

// Other cases for Size are covered as part of write tests.
func (testSuite *MemoryBlockTest) TestMemoryBlockSizeForEmptyBlock() {
	mb := createBlock(12)

	assert.Equal(testSuite.T(), int64(0), mb.Size())
}

// Other cases for reader are covered as part of write tests.
func (testSuite *MemoryBlockTest) TestMemoryBlockReaderForEmptyBlock() {
	mb := createBlock(12)

	output, err := io.ReadAll(mb.Reader())
	assert.Nil(testSuite.T(), err)
	assert.Empty(testSuite.T(), output)
	assert.Equal(testSuite.T(), int64(0), mb.Size())
}
