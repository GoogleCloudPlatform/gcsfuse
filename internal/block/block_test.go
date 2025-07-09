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
	"context"
	"io"
	"sync"
	"testing"
	"time"

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
	output, err := io.ReadAll(mb.Reader())
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

	output, err := io.ReadAll(mb.Reader())
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
	output, err := io.ReadAll(mb.Reader())
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), content, output)
	require.Equal(testSuite.T(), int64(2), mb.Size())
	err = mb.SetAbsStartOff(23)
	require.Nil(testSuite.T(), err)

	mb.Reuse()

	output, err = io.ReadAll(mb.Reader())
	assert.Nil(testSuite.T(), err)
	assert.Empty(testSuite.T(), output)
	assert.Equal(testSuite.T(), int64(0), mb.Size())
	assert.Panics(testSuite.T(), func() {
		_ = mb.AbsStartOff()
	})
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
	n, err := mb.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), 2, n)
	output, err := io.ReadAll(mb.Reader())
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), content, output)
	require.Equal(testSuite.T(), int64(2), mb.Size())

	err = mb.Deallocate()

	assert.Nil(testSuite.T(), err)
	assert.Nil(testSuite.T(), mb.(*memoryBlock).buffer)
}

func (testSuite *MemoryBlockTest) TestMemoryBlockReadAtSuccess() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	_, err = mb.Write(content)
	require.Nil(testSuite.T(), err)
	readBuffer := make([]byte, 5)

	n, err := mb.ReadAt(readBuffer, 6) // Read "world"

	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), 5, n)
	assert.Equal(testSuite.T(), []byte("world"), readBuffer)
}

func (testSuite *MemoryBlockTest) TestMemoryBlockReadAtBeyondBlockSize() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	_, err = mb.Write(content)
	require.Nil(testSuite.T(), err)
	readBuffer := make([]byte, 5)

	n, err := mb.ReadAt(readBuffer, 15) // Read beyond the block size

	assert.NotNil(testSuite.T(), err)
	assert.NotErrorIs(testSuite.T(), err, io.EOF)
	assert.Equal(testSuite.T(), 0, n)
}

func (testSuite *MemoryBlockTest) TestMemoryBlockReadAtWithNegativeOffset() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	_, err = mb.Write(content)
	require.Nil(testSuite.T(), err)
	readBuffer := make([]byte, 5)

	n, err := mb.ReadAt(readBuffer, -1) // Negative offset

	assert.NotNil(testSuite.T(), err)
	assert.NotErrorIs(testSuite.T(), err, io.EOF)
	assert.Equal(testSuite.T(), 0, n)
}

func (testSuite *MemoryBlockTest) TestMemoryBlockReadAtEOF() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	_, err = mb.Write(content)
	require.Nil(testSuite.T(), err)
	readBuffer := make([]byte, 15)

	n, err := mb.ReadAt(readBuffer, 6) // Read "world"

	assert.Equal(testSuite.T(), io.EOF, err)
	assert.Equal(testSuite.T(), 5, n)
	assert.Equal(testSuite.T(), []byte("world"), readBuffer[0:n])
}

func (testSuite *MemoryBlockTest) TestMemoryBlockAbsStartOffsetPanicsOnEmptyBlock() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)

	// The absolute start offset should be -1 initially.
	assert.Panics(testSuite.T(), func() {
		_ = mb.AbsStartOff()
	})
}

func (testSuite *MemoryBlockTest) TestMemoryBlockAbsStartOffsetValid() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)

	// Set the absolute start offset to a valid value.
	mb.(*memoryBlock).absStartOff = 100

	// The absolute start offset should return the set value.
	assert.Equal(testSuite.T(), int64(100), mb.AbsStartOff())
}

func (testSuite *MemoryBlockTest) TestMemoryBlockSetAbsStartOffsetInvalid() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)

	err = mb.SetAbsStartOff(-23)

	assert.Error(testSuite.T(), err)
}

func (testSuite *MemoryBlockTest) TestMemoryBlockSetAbsStartOffsetValid() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)

	err = mb.SetAbsStartOff(23)

	assert.NoError(testSuite.T(), err)
	assert.Equal(testSuite.T(), int64(23), mb.AbsStartOff())
}

func (testSuite *MemoryBlockTest) TestMemoryBlockSetAbsStartOffsetTwiceInvalid() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	err = mb.SetAbsStartOff(23)
	require.Nil(testSuite.T(), err)

	err = mb.SetAbsStartOff(42)

	assert.Error(testSuite.T(), err)
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

func (testSuite *MemoryBlockTest) TestAwaitReadyWaitIfNotNotify() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	ctx, cancel := context.WithTimeout(testSuite.T().Context(), 100*time.Millisecond)
	defer cancel()

	_, err = mb.AwaitReady(ctx)

	assert.NotNil(testSuite.T(), err)
	assert.Equal(testSuite.T(), context.DeadlineExceeded, err)
}

func (testSuite *MemoryBlockTest) TestAwaitReadyNotifyVariants() {
	tests := []struct {
		name         string
		notifyStatus BlockStatus
		wantStatus   BlockStatus
	}{
		{
			name:         "AfterNotifySuccess",
			notifyStatus: BlockStatusDownloaded,
			wantStatus:   BlockStatusDownloaded,
		},
		{
			name:         "AfterNotifyError",
			notifyStatus: BlockStatusDownloadFailed,
			wantStatus:   BlockStatusDownloadFailed,
		},
		{
			name:         "AfterNotifyCancelled",
			notifyStatus: BlockStatusDownloadCancelled,
			wantStatus:   BlockStatusDownloadCancelled,
		},
	}

	for _, tt := range tests {
		testSuite.T().Run(tt.name, func(t *testing.T) {
			mb, err := createBlock(12)
			require.Nil(t, err)
			go func() {
				time.Sleep(time.Millisecond)
				mb.NotifyReady(tt.notifyStatus)
			}()

			status, err := mb.AwaitReady(context.Background())

			require.Nil(t, err)
			assert.Equal(t, tt.wantStatus, status)
		})
	}
}

func (testSuite *MemoryBlockTest) TestTwoNotifyReadyWithoutAwaitReady() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	mb.NotifyReady(BlockStatusDownloaded)

	// 2nd notify will lead to panic since it is not allowed to notify a block more than once.
	assert.Panics(testSuite.T(), func() {
		mb.NotifyReady(BlockStatusDownloaded)
	})
}

func (testSuite *MemoryBlockTest) TestTwoNotifyReadyWithAwaitReady() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	ctx, cancel := context.WithTimeout(testSuite.T().Context(), 100*time.Millisecond)
	defer cancel()
	go func() {
		time.Sleep(time.Millisecond)
		mb.NotifyReady(BlockStatusDownloaded)
	}()
	status, err := mb.AwaitReady(ctx)
	require.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), BlockStatusDownloaded, status)

	// 2nd notify will lead to panic since channel is closed after first await ready.
	assert.Panics(testSuite.T(), func() {
		mb.NotifyReady(BlockStatusDownloaded)
	})
}

func (testSuite *MemoryBlockTest) TestSingleNotifyAndMultipleAwaitReady() {
	mb, err := createBlock(12)
	require.Nil(testSuite.T(), err)
	go func() {
		time.Sleep(time.Millisecond)
		mb.NotifyReady(BlockStatusDownloaded)
	}()
	ctx, cancel := context.WithTimeout(testSuite.T().Context(), 5*time.Millisecond)
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(5)

	// Multiple goroutines waiting for the same block to be ready.
	// They should all receive the same status once the block is notified.
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			time.Sleep(1 * time.Millisecond)

			status, err := mb.AwaitReady(ctx)

			require.Nil(testSuite.T(), err)
			assert.Equal(testSuite.T(), BlockStatusDownloaded, status)
		}()
	}
	wg.Wait()
}
