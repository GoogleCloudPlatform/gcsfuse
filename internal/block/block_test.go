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
	"bytes"
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
	assert.Equal(testSuite.T(), 2, mb.Size())
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
	assert.Equal(testSuite.T(), 7, mb.Size())
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
	require.Equal(testSuite.T(), 2, mb.Size())

	mb.Reuse()

	assert.Equal(testSuite.T(), int64(0), mb.(*memoryBlock).readSeek)
	output, err = io.ReadAll(mb)
	assert.Nil(testSuite.T(), err)
	assert.Empty(testSuite.T(), output)
	assert.Equal(testSuite.T(), 0, mb.Size())
}

// Other cases for Size are covered as part of write tests.
func (testSuite *MemoryBlockTest) TestMemoryBlockSizeForEmptyBlock() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)

	assert.Equal(testSuite.T(), 0, mb.Size())
}

// Other cases for reader are covered as part of write tests.
func (testSuite *MemoryBlockTest) TestMemoryBlockReaderForEmptyBlock() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)

	assert.Equal(testSuite.T(), int64(0), mb.(*memoryBlock).readSeek)
	output, err := io.ReadAll(mb)
	assert.Nil(testSuite.T(), err)
	assert.Empty(testSuite.T(), output)
	assert.Equal(testSuite.T(), 0, mb.Size())
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
	require.Equal(testSuite.T(), 2, mb.Size())

	err = mb.Deallocate()

	assert.Nil(testSuite.T(), err)
	assert.Nil(testSuite.T(), mb.(*memoryBlock).buffer)
}

func (testSuite *MemoryBlockTest) TestMemoryBlockCap() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)

	assert.Equal(testSuite.T(), 12, mb.Cap())
}

func (testSuite *MemoryBlockTest) TestMemoryBlockCapAfterWrite() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hi")
	n, err := mb.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), 2, n)

	assert.Equal(testSuite.T(), 12, mb.Cap())
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

func (testSuite *MemoryBlockTest) TestMemoryBlockLimitedReadFromSuccess() {
	mb, err := createBlock(20)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	reader := bytes.NewReader(content)

	n, err := mb.LimitedReadFrom(reader, len(content))

	require.NoError(testSuite.T(), err)
	assert.Equal(testSuite.T(), len(content), n)
	assert.Equal(testSuite.T(), len(content), mb.Size())
	readContent := make([]byte, len(content))
	_, err = mb.Read(readContent)
	require.ErrorIs(testSuite.T(), err, io.EOF)
	assert.Equal(testSuite.T(), content, readContent)
}

func (testSuite *MemoryBlockTest) TestMemoryBlockLimitedReadFromLimitMoreThanCapacity() {
	mb, err := createBlock(10)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world") // 11 bytes
	reader := bytes.NewReader(content)

	n, err := mb.LimitedReadFrom(reader, len(content))

	assert.Error(testSuite.T(), err)
	assert.Contains(testSuite.T(), err.Error(), "limit is more than remaining capacity of block")
	assert.Equal(testSuite.T(), 0, n)
	assert.Equal(testSuite.T(), 0, mb.Size())
}

func (testSuite *MemoryBlockTest) TestMemoryBlockLimitedReadFromReaderWithLessData() {
	mb, err := createBlock(20)
	require.Nil(testSuite.T(), err)
	content := []byte("hello") // 5 bytes
	reader := bytes.NewReader(content)

	n, err := mb.LimitedReadFrom(reader, 10)

	// io.ReadFull returns io.ErrUnexpectedEOF if the reader provides fewer bytes than requested.
	// This is expected behavior in this test case.
	require.ErrorIs(testSuite.T(), err, io.ErrUnexpectedEOF)
	assert.Equal(testSuite.T(), len(content), n)
	assert.Equal(testSuite.T(), len(content), mb.Size())
	readContent := make([]byte, len(content))
	_, readErr := mb.Read(readContent)
	require.ErrorIs(testSuite.T(), readErr, io.EOF)
	assert.Equal(testSuite.T(), content, readContent)
}

func (testSuite *MemoryBlockTest) TestMemoryBlockLimitedReadFromMultipleReads() {
	mb, err := createBlock(30)
	require.Nil(testSuite.T(), err)
	content1 := []byte("hello ")
	reader1 := bytes.NewReader(content1)
	content2 := []byte("world")
	reader2 := bytes.NewReader(content2)

	n1, err1 := mb.LimitedReadFrom(reader1, len(content1))
	require.NoError(testSuite.T(), err1)
	assert.Equal(testSuite.T(), len(content1), n1)
	assert.Equal(testSuite.T(), len(content1), mb.Size())

	n2, err2 := mb.LimitedReadFrom(reader2, len(content2))
	require.NoError(testSuite.T(), err2)
	assert.Equal(testSuite.T(), len(content2), n2)
	assert.Equal(testSuite.T(), len(content1)+len(content2), mb.Size())

	readContent := make([]byte, mb.Size())
	_, readErr := mb.Read(readContent)
	require.ErrorIs(testSuite.T(), readErr, io.EOF)
	assert.Equal(testSuite.T(), "hello world", string(readContent))
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

func (testSuite *MemoryBlockTest) TestMemoryBlockSeekInvalidWhence() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	_, err = mb.Write([]byte("hello"))
	require.Nil(testSuite.T(), err)

	_, err = mb.Seek(0, 4)

	assert.ErrorContains(testSuite.T(), err, "invalid whence value")
}

func (testSuite *MemoryBlockTest) TestMemoryBlockSeekOutOfBounds() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	_, err = mb.Write([]byte("hello"))
	require.Nil(testSuite.T(), err)

	tests := []struct {
		name   string
		offset int64
		whence int
	}{
		{"SeekStartNegative", -1, io.SeekStart},
		{"SeekStartBeyondEnd", 6, io.SeekStart},
		{"SeekCurrentNegative", -1, io.SeekCurrent},
		{"SeekCurrentBeyondEnd", 6, io.SeekCurrent},
		{"SeekEndNegative", -6, io.SeekEnd},
		{"SeekEndBeyondEnd", 1, io.SeekEnd},
	}

	for _, tt := range tests {
		testSuite.T().Run(tt.name, func(t *testing.T) {
			// Reset readSeek to 0 before each test case
			_, _ = mb.Seek(0, io.SeekStart)

			_, err := mb.Seek(tt.offset, tt.whence)

			assert.ErrorContains(t, err, "out of bounds")
		})
	}
}

func (testSuite *MemoryBlockTest) TestMemoryBlockDoubleDeallocate() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	err = mb.Deallocate()
	require.Nil(testSuite.T(), err)

	err = mb.Deallocate()

	assert.ErrorContains(testSuite.T(), err, "invalid buffer")
}
