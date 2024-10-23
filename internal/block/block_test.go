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
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const outOfCapacityError string = "received data more than capacity of the block"

type MemoryBlockTest struct {
	suite.Suite
}

func TestMemoryBlockTestSuite(t *testing.T) {
	suite.Run(t, new(MemoryBlockTest))
}

func (testSuite *MemoryBlockTest) TestMemoryBlockWrite() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hi")
	err = mb.Write(content)

	assert.Nil(testSuite.T(), err)
	output, err := io.ReadAll(mb.Reader())
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), content, output)
	assert.Equal(testSuite.T(), int64(2), mb.Size())
}

func (testSuite *MemoryBlockTest) TestMemoryBlockWriteWithDataGreaterThanCapacity() {
	mb, err := createBlock(1)
	require.Nil(testSuite.T(), err)
	content := []byte("hi")
	err = mb.Write(content)

	assert.NotNil(testSuite.T(), err)
	assert.EqualError(testSuite.T(), err, outOfCapacityError)
}

func (testSuite *MemoryBlockTest) TestMemoryBlockWriteWithMultipleWrites() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	err = mb.Write([]byte("hi"))
	assert.Nil(testSuite.T(), err)
	err = mb.Write([]byte("hello"))
	assert.Nil(testSuite.T(), err)

	output, err := io.ReadAll(mb.Reader())
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), []byte("hihello"), output)
	assert.Equal(testSuite.T(), int64(7), mb.Size())
}

func (testSuite *MemoryBlockTest) TestMemoryBlockWriteWith2ndWriteBeyondCapacity() {
	mb, err := createBlock(2)
	require.Nil(testSuite.T(), err)
	content := []byte("hi")
	err = mb.Write(content)
	assert.Nil(testSuite.T(), err)
	err = mb.Write(content)

	assert.NotNil(testSuite.T(), err)
	assert.EqualError(testSuite.T(), err, outOfCapacityError)
}

func (testSuite *MemoryBlockTest) TestMemoryBlockReuse() {
	mb, err := createBlock(12)
	content := []byte("hi")
	err = mb.Write(content)
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
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)

	assert.Equal(testSuite.T(), int64(0), mb.Size())
}

// Other cases for reader are covered as part of write tests.
func (testSuite *MemoryBlockTest) TestMemoryBlockReaderForEmptyBlock() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)

	output, err := io.ReadAll(mb.Reader())
	assert.Nil(testSuite.T(), err)
	assert.Empty(testSuite.T(), output)
	assert.Equal(testSuite.T(), int64(0), mb.Size())
}

func (testSuite *MemoryBlockTest) TestMemoryBlockDeAllocate() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hi")
	err = mb.Write(content)
	require.Nil(testSuite.T(), err)
	output, err := io.ReadAll(mb.Reader())
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), content, output)
	require.Equal(testSuite.T(), int64(2), mb.Size())

	err = mb.Deallocate()

	require.Nil(testSuite.T(), err)
	require.Nil(testSuite.T(), mb.(*memoryBlock).buffer)
}
