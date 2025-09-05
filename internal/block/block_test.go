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
	"errors"
	"fmt"
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
	n, err := mb.Write(content)

	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), len(content), n)
	assert.Equal(testSuite.T(), int64(0), mb.(*memoryBlock).readSeek)
	output, err := io.ReadAll(mb)
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), content, output)
	assert.Equal(testSuite.T(), int64(2), mb.Size())
}

func (testSuite *MemoryBlockTest) TestMemoryBlockWriteWithDataGreaterThanCapacity() {
	mb, err := createBlock(1)
	require.Nil(testSuite.T(), err)
	content := []byte("hi")
	n, err := mb.Write(content)

	assert.NotNil(testSuite.T(), err)
	assert.Equal(testSuite.T(), 0, n)
	assert.EqualError(testSuite.T(), err, outOfCapacityError)
}

func (testSuite *MemoryBlockTest) TestMemoryBlockWriteWithMultipleWrites() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	n, err := mb.Write([]byte("hi"))
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), 2, n)
	n, err = mb.Write([]byte("hello"))
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), 5, n)

	assert.Equal(testSuite.T(), int64(0), mb.(*memoryBlock).readSeek)
	output, err := io.ReadAll(mb)
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), []byte("hihello"), output)
	assert.Equal(testSuite.T(), int64(7), mb.Size())
}

func (testSuite *MemoryBlockTest) TestMemoryBlockWriteWith2ndWriteBeyondCapacity() {
	mb, err := createBlock(2)
	require.Nil(testSuite.T(), err)
	content := []byte("hi")
	n, err := mb.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), 2, n)
	n, err = mb.Write(content)

	assert.NotNil(testSuite.T(), err)
	assert.Equal(testSuite.T(), 0, n)
	assert.EqualError(testSuite.T(), err, outOfCapacityError)
}

func (testSuite *MemoryBlockTest) TestMemoryBlockReuse() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hi")
	n, err := mb.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), 2, n)
	require.Equal(testSuite.T(), int64(0), mb.(*memoryBlock).readSeek)
	output, err := io.ReadAll(mb)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), content, output)
	require.Equal(testSuite.T(), int64(2), mb.Size())

	mb.Reuse()

	assert.Equal(testSuite.T(), int64(0), mb.(*memoryBlock).readSeek)
	output, err = io.ReadAll(mb)
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

	assert.Equal(testSuite.T(), int64(0), mb.(*memoryBlock).readSeek)
	output, err := io.ReadAll(mb)
	assert.Nil(testSuite.T(), err)
	assert.Empty(testSuite.T(), output)
	assert.Equal(testSuite.T(), int64(0), mb.Size())
}

func (testSuite *MemoryBlockTest) TestMemoryBlockDeAllocate() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hi")
	n, err := mb.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), 2, n)
	require.Equal(testSuite.T(), int64(0), mb.(*memoryBlock).readSeek)
	output, err := io.ReadAll(mb)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), content, output)
	require.Equal(testSuite.T(), int64(2), mb.Size())

	err = mb.Deallocate()

	assert.Nil(testSuite.T(), err)
	assert.Nil(testSuite.T(), mb.(*memoryBlock).buffer)
}

func (testSuite *MemoryBlockTest) TestMemoryBlockCap() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)

	assert.Equal(testSuite.T(), int64(12), mb.Cap())
}

func (testSuite *MemoryBlockTest) TestMemoryBlockCapAfterWrite() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hi")
	n, err := mb.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), 2, n)

	assert.Equal(testSuite.T(), int64(12), mb.Cap())
}

func (testSuite *MemoryBlockTest) TestMemoryBlockReadSuccess() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	n, err := mb.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), len(content), n)
	readBuffer := make([]byte, 5)

	n, err = mb.Read(readBuffer)

	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), 5, n)
	assert.Equal(testSuite.T(), "hello", string(readBuffer))
}

func (testSuite *MemoryBlockTest) TestMemoryBlockReadWithReadBufferMoreThanBlockSize() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	n, err := mb.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), len(content), n)
	readBuffer := make([]byte, 20)

	n, err = mb.Read(readBuffer)

	require.Error(testSuite.T(), io.EOF, err)
	require.Equal(testSuite.T(), 11, n) // Read should return all bytes written.
}

func (testSuite *MemoryBlockTest) TestMemoryBlockReadSeekBeyondEnd() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	n, err := mb.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), len(content), n)
	readBuffer := make([]byte, 12)
	mb.(*memoryBlock).readSeek = 13 // Set readSeek to a position beyond the end of the block.

	n, err = mb.Read(readBuffer)

	require.Equal(testSuite.T(), io.EOF, err)
	require.Equal(testSuite.T(), 0, n)
}

func (testSuite *MemoryBlockTest) TestMemoryBlockReadFailure() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	readBuffer := make([]byte, 5)
	mb.(*memoryBlock).readSeek = -1 // Simulate an invalid readSeek position.
	require.Nil(testSuite.T(), err)

	n, err := mb.Read(readBuffer)

	require.Equal(testSuite.T(), errors.New("readSeek -1 is less than start offset 0"), err)
	require.Equal(testSuite.T(), 0, n) // Read should return 0 bytes read.
}

func (testSuite *MemoryBlockTest) TestMemoryBlockSeek() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	n, err := mb.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), len(content), n)

	tests := []struct {
		whence         int
		offset         int64
		expectedOutput string
		expectedOffset int64
	}{
		{io.SeekStart, 0, "hello", 0},   // After this, readSeek = 5
		{io.SeekCurrent, 1, "world", 6}, // After this readSeek = 11
		{io.SeekEnd, -6, " worl", 5},
	}

	for _, tt := range tests {
		testSuite.T().Run(fmt.Sprintf("whence=%d, offset=%d, expectedOutput:%s", tt.whence, tt.offset, tt.expectedOutput), func(t *testing.T) {
			offset, err := mb.Seek(tt.offset, tt.whence)

			require.Nil(t, err)
			require.Equal(t, tt.expectedOffset, offset)
			readBuffer := make([]byte, 5)
			n, err = mb.Read(readBuffer)
			require.Condition(t, func() bool {
				return err == nil || errors.Is(err, io.EOF)
			}, "Read err can be nil or io.EOF")
			require.Equal(t, 5, n)
			assert.Equal(t, tt.expectedOutput, string(readBuffer))
		})
	}
}

////////////////////////////////////////////////////////////////////////
// Disk Block Tests
////////////////////////////////////////////////////////////////////////

type DiskBlockTest struct {
	suite.Suite
}

func TestDiskBlockTestSuite(t *testing.T) {
	suite.Run(t, new(DiskBlockTest))
}

func (testSuite *DiskBlockTest) TestDiskBlockWrite() {
	db, err := CreateBlock(12, DiskBlock)
	require.Nil(testSuite.T(), err)
	defer db.Deallocate()

	content := []byte("hi")
	n, err := db.Write(content)

	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), len(content), n)
	assert.Equal(testSuite.T(), int64(0), db.(*diskBlock).readSeek)
	output, err := io.ReadAll(db)
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), content, output)
	assert.Equal(testSuite.T(), int64(2), db.Size())
}

func (testSuite *DiskBlockTest) TestDiskBlockWriteExceedsCapacity() {
	db, err := CreateBlock(5, DiskBlock)
	require.Nil(testSuite.T(), err)
	defer db.Deallocate()

	content := []byte("hello world")
	n, err := db.Write(content)

	assert.Equal(testSuite.T(), 0, n)
	assert.NotNil(testSuite.T(), err)
	assert.Contains(testSuite.T(), err.Error(), outOfCapacityError)
}

func (testSuite *DiskBlockTest) TestDiskBlockReuse() {
	db, err := CreateBlock(12, DiskBlock)
	require.Nil(testSuite.T(), err)
	defer db.Deallocate()

	content := []byte("hi")
	n, err := db.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), 2, n)
	require.Equal(testSuite.T(), int64(0), db.(*diskBlock).readSeek)
	output, err := io.ReadAll(db)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), content, output)
	require.Equal(testSuite.T(), int64(2), db.Size())

	db.Reuse()

	assert.Equal(testSuite.T(), int64(0), db.(*diskBlock).readSeek)
	output, err = io.ReadAll(db)
	assert.Nil(testSuite.T(), err)
	assert.Empty(testSuite.T(), output)
	assert.Equal(testSuite.T(), int64(0), db.Size())
}

func (testSuite *DiskBlockTest) TestDiskBlockRead() {
	db, err := CreateBlock(12, DiskBlock)
	require.Nil(testSuite.T(), err)
	defer db.Deallocate()

	content := []byte("hello world")
	n, err := db.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), 11, n)

	output := make([]byte, 5)
	n, err = db.Read(output)
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), 5, n)
	assert.Equal(testSuite.T(), "hello", string(output))

	output = make([]byte, 6)
	n, err = db.Read(output)
	assert.Equal(testSuite.T(), io.EOF, err)
	assert.Equal(testSuite.T(), 6, n)
	assert.Equal(testSuite.T(), " world", string(output))
}

func (testSuite *DiskBlockTest) TestDiskBlockSeek() {
	db, err := CreateBlock(12, DiskBlock)
	require.Nil(testSuite.T(), err)
	defer db.Deallocate()

	content := []byte("hello world")
	n, err := db.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), 11, n)

	type testCase struct {
		whence         int
		offset         int64
		expectedOutput string
		expectedOffset int64
	}

	tests := []testCase{
		{io.SeekStart, 0, "hello", 0},   // After this, readSeek = 5
		{io.SeekCurrent, 1, "world", 6}, // After this readSeek = 11
		{io.SeekEnd, -6, " worl", 5},
	}

	for _, tt := range tests {
		testSuite.T().Run(fmt.Sprintf("whence=%d, offset=%d, expectedOutput:%s", tt.whence, tt.offset, tt.expectedOutput), func(t *testing.T) {
			offset, err := db.Seek(tt.offset, tt.whence)

			require.Nil(t, err)
			require.Equal(t, tt.expectedOffset, offset)
			readBuffer := make([]byte, 5)
			n, err = db.Read(readBuffer)
			require.Condition(t, func() bool {
				return err == nil || errors.Is(err, io.EOF)
			}, "Read err can be nil or io.EOF")
			require.Equal(t, 5, n)
			assert.Equal(t, tt.expectedOutput, string(readBuffer))
		})
	}
}

func (testSuite *DiskBlockTest) TestCreateBlockTypes() {
	// Test memory block creation
	mb, err := CreateBlock(1024, MemoryBlock)
	require.Nil(testSuite.T(), err)
	require.NotNil(testSuite.T(), mb)
	assert.Equal(testSuite.T(), int64(1024), mb.Cap())
	mb.Deallocate()

	// Test disk block creation
	db, err := CreateBlock(1024, DiskBlock)
	require.Nil(testSuite.T(), err)
	require.NotNil(testSuite.T(), db)
	assert.Equal(testSuite.T(), int64(1024), db.Cap())
	db.Deallocate()

	// Test invalid block type
	_, err = CreateBlock(1024, BlockType(999))
	require.NotNil(testSuite.T(), err)
	assert.Contains(testSuite.T(), err.Error(), "unsupported block type")
}

func (testSuite *DiskBlockTest) TestBackwardCompatibility() {
	// Test that the legacy createBlock function still works
	mb, err := createBlock(512)
	require.Nil(testSuite.T(), err)
	require.NotNil(testSuite.T(), mb)
	assert.Equal(testSuite.T(), int64(512), mb.Cap())
	mb.Deallocate()
}

func (testSuite *DiskBlockTest) TestDiskVsMemoryBlockBehavior() {
	// Test that both block types behave identically for basic operations
	const blockSize = 1024
	const testData = "Hello, disk vs memory blocks!"

	// Create both types of blocks
	memBlock, err := CreateBlock(blockSize, MemoryBlock)
	require.Nil(testSuite.T(), err)
	defer memBlock.Deallocate()

	diskBlock, err := CreateBlock(blockSize, DiskBlock)
	require.Nil(testSuite.T(), err)
	defer diskBlock.Deallocate()

	// Both blocks should have the same capacity
	assert.Equal(testSuite.T(), memBlock.Cap(), diskBlock.Cap())

	// Write the same data to both blocks
	memN, memErr := memBlock.Write([]byte(testData))
	diskN, diskErr := diskBlock.Write([]byte(testData))

	// Both should succeed identically
	require.Nil(testSuite.T(), memErr)
	require.Nil(testSuite.T(), diskErr)
	assert.Equal(testSuite.T(), memN, diskN)
	assert.Equal(testSuite.T(), len(testData), memN)
	assert.Equal(testSuite.T(), len(testData), diskN)

	// Both should report the same size
	assert.Equal(testSuite.T(), memBlock.Size(), diskBlock.Size())
	assert.Equal(testSuite.T(), int64(len(testData)), memBlock.Size())

	// Reading should produce identical results
	memData, memReadErr := io.ReadAll(memBlock)
	diskData, diskReadErr := io.ReadAll(diskBlock)

	require.Nil(testSuite.T(), memReadErr)
	require.Nil(testSuite.T(), diskReadErr)
	assert.Equal(testSuite.T(), testData, string(memData))
	assert.Equal(testSuite.T(), testData, string(diskData))
	assert.Equal(testSuite.T(), memData, diskData)
}
