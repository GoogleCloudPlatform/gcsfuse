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
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PrefetchMemoryBlockTest struct {
	MemoryBlockTest
}

func TestPrefetchMemoryBlockTestSuite(t *testing.T) {
	suite.Run(t, new(PrefetchMemoryBlockTest))
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockReuse() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	content := []byte("hi")
	n, err := pmb.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), 2, n)
	output, err := io.ReadAll(pmb)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), content, output)
	require.Equal(testSuite.T(), 2, pmb.Size())
	err = pmb.SetAbsStartOff(23)
	require.Nil(testSuite.T(), err)
	pmb.IncRef()
	assert.Equal(testSuite.T(), int32(1), pmb.RefCount())

	pmb.Reuse()

	assert.Equal(testSuite.T(), int64(0), pmb.(*prefetchMemoryBlock).readSeek)
	output, err = io.ReadAll(pmb)
	assert.Nil(testSuite.T(), err)
	assert.Empty(testSuite.T(), output)
	assert.Equal(testSuite.T(), 0, pmb.Size())
	assert.Panics(testSuite.T(), func() {
		_ = pmb.AbsStartOff()
	})
	assert.Equal(testSuite.T(), int32(0), pmb.RefCount())
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockReadAtSuccess() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	_, err = pmb.Write(content)
	require.Nil(testSuite.T(), err)
	readBuffer := make([]byte, 5)

	n, err := pmb.ReadAt(readBuffer, 6) // Read "world"

	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), 5, n)
	assert.Equal(testSuite.T(), []byte("world"), readBuffer)
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockReadAtBeyondBlockSize() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	_, err = pmb.Write(content)
	require.Nil(testSuite.T(), err)
	readBuffer := make([]byte, 5)

	n, err := pmb.ReadAt(readBuffer, 15) // Read beyond the block size

	assert.NotNil(testSuite.T(), err)
	assert.NotErrorIs(testSuite.T(), err, io.EOF)
	assert.Equal(testSuite.T(), 0, n)
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockReadAtWithNegativeOffset() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	_, err = pmb.Write(content)
	require.Nil(testSuite.T(), err)
	readBuffer := make([]byte, 5)

	n, err := pmb.ReadAt(readBuffer, -1) // Negative offset

	assert.NotNil(testSuite.T(), err)
	assert.NotErrorIs(testSuite.T(), err, io.EOF)
	assert.Equal(testSuite.T(), 0, n)
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockReadAtEOF() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	_, err = pmb.Write(content)
	require.Nil(testSuite.T(), err)
	readBuffer := make([]byte, 15)

	n, err := pmb.ReadAt(readBuffer, 6) // Read "world"

	assert.Equal(testSuite.T(), io.EOF, err)
	assert.Equal(testSuite.T(), 5, n)
	assert.Equal(testSuite.T(), []byte("world"), readBuffer[0:n])
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockReadAtSliceSuccess() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	_, err = pmb.Write(content)
	require.Nil(testSuite.T(), err)

	slice, err := pmb.ReadAtSlice(6, 5) // Read "world"

	assert.NoError(testSuite.T(), err)
	assert.Equal(testSuite.T(), []byte("world"), slice)
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockReadAtSliceEOF() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	content := []byte("hello world")
	_, err = pmb.Write(content)
	require.Nil(testSuite.T(), err)

	slice, err := pmb.ReadAtSlice(6, 15) // Read "world" and beyond

	assert.Equal(testSuite.T(), io.EOF, err)
	assert.Equal(testSuite.T(), []byte("world"), slice)
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockReadAtSliceWithNegativeOffset() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	_, err = pmb.Write([]byte("hello world"))
	require.Nil(testSuite.T(), err)

	_, err = pmb.ReadAtSlice(-1, 5)

	assert.Error(testSuite.T(), err)
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockReadAtSliceWithOffsetOutOfBounds() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	_, err = pmb.Write([]byte("hello"))
	require.Nil(testSuite.T(), err)

	_, err = pmb.ReadAtSlice(5, 1) // Offset is equal to size, which is out of bounds

	assert.Error(testSuite.T(), err)
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockAbsStartOffsetPanicsOnEmptyBlock() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)

	// The absolute start offset should be -1 initially.
	assert.Panics(testSuite.T(), func() {
		_ = pmb.AbsStartOff()
	})
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockAbsStartOffsetValid() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)

	// Set the absolute start offset to a valid value.
	pmb.(*prefetchMemoryBlock).absStartOff = 100

	// The absolute start offset should return the set value.
	assert.Equal(testSuite.T(), int64(100), pmb.AbsStartOff())
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockSetAbsStartOffsetInvalid() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)

	err = pmb.SetAbsStartOff(-23)

	assert.Error(testSuite.T(), err)
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockSetAbsStartOffsetSuccess() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)

	err = pmb.SetAbsStartOff(23)

	assert.NoError(testSuite.T(), err)
	assert.Equal(testSuite.T(), int64(23), pmb.AbsStartOff())
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockSetAbsStartOffsetTwiceInvalid() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	err = pmb.SetAbsStartOff(23)
	require.Nil(testSuite.T(), err)

	err = pmb.SetAbsStartOff(42)

	assert.Error(testSuite.T(), err)
}

func (testSuite *PrefetchMemoryBlockTest) TestAwaitReadyWaitIfNotNotify() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	ctx, cancel := context.WithTimeout(testSuite.T().Context(), 100*time.Millisecond)
	defer cancel()

	_, err = pmb.AwaitReady(ctx, 1)

	assert.NotNil(testSuite.T(), err)
	assert.EqualError(testSuite.T(), context.DeadlineExceeded, err.Error())
}

func (testSuite *PrefetchMemoryBlockTest) TestAwaitReadyReturnsErrorOnContextCancellation() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	ctx, cancel := context.WithCancel(testSuite.T().Context())
	cancel() // Cancel the context immediately

	_, err = pmb.AwaitReady(ctx, 1)

	require.NotNil(testSuite.T(), err)
	assert.EqualError(testSuite.T(), context.Canceled, err.Error())
}

func (testSuite *PrefetchMemoryBlockTest) TestAwaitReadyNotifyVariants() {
	errFailed := errors.New("failed")
	tests := []struct {
		name         string
		notifyStatus BlockStatus
		wantStatus   BlockStatus
		requestSize  int
	}{
		{
			name:         "AfterNotifySuccess",
			notifyStatus: BlockStatus{Size: 100, Err: nil},
			wantStatus:   BlockStatus{Size: 100, Err: nil},
			requestSize:  1,
		},
		{
			name:         "AfterNotifyError",
			notifyStatus: BlockStatus{Size: 0, Err: errFailed},
			wantStatus:   BlockStatus{Size: 0, Err: errFailed},
			requestSize:  0,
		},
	}

	for _, tt := range tests {
		testSuite.T().Run(tt.name, func(t *testing.T) {
			pmb, err := createPrefetchBlock(MiB)
			require.Nil(t, err)
			go func() {
				time.Sleep(time.Millisecond)
				pmb.NotifyReady(tt.notifyStatus)
			}()

			status, err := pmb.AwaitReady(t.Context(), tt.requestSize)

			require.Nil(t, err)
			assert.Equal(t, tt.wantStatus, status)
		})
	}
}

func (testSuite *PrefetchMemoryBlockTest) TestMultipleNotifyReadyWithoutAwaitReady() {
	pmb, err := createPrefetchBlock(2 * MiB)
	require.Nil(testSuite.T(), err)

	pmb.NotifyReady(BlockStatus{Size: 100})
	pmb.NotifyReady(BlockStatus{Size: 200})
	// 3rd notify will lead to panic since it is not allowed to notify a block more than channel capacity (which is 2 for 2 MiB block).
	assert.Panics(testSuite.T(), func() {
		pmb.NotifyReady(BlockStatus{Size: 300})
	})
}

func (testSuite *PrefetchMemoryBlockTest) TestNotifyReadyAfterAwaitReady() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	go func() {
		pmb.NotifyReady(BlockStatus{Size: 100})
	}()
	status, err := pmb.AwaitReady(testSuite.T().Context(), 1)
	require.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), BlockStatus{Size: 100}, status)

	// 2nd notify should NOT panic because AwaitReady consumed the first notification, freeing up the channel.
	assert.NotPanics(testSuite.T(), func() {
		pmb.NotifyReady(BlockStatus{Size: 200})
	})
}

func (testSuite *PrefetchMemoryBlockTest) TestSingleNotifyAndMultipleAwaitReady() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	go func() {
		time.Sleep(10 * time.Millisecond)
		pmb.NotifyReady(BlockStatus{Size: 100})
	}()
	var wg sync.WaitGroup
	wg.Add(5)

	// Multiple goroutines waiting for the same block to be ready.
	// Only one will receive the notification.
	// We use a timeout for the others to avoid hanging the test.
	for range 5 {
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(testSuite.T().Context(), 100*time.Millisecond)
			defer cancel()
			status, err := pmb.AwaitReady(ctx, 1)

			if err == nil {
				assert.Equal(testSuite.T(), BlockStatus{Size: 100}, status)
			} else {
				assert.ErrorIs(testSuite.T(), err, context.DeadlineExceeded)
			}
		}()
	}
	wg.Wait()
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockIncRef() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)

	pmb.IncRef()

	assert.Equal(testSuite.T(), int32(1), pmb.RefCount())
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockDecRef() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)
	pmb.IncRef()
	pmb.IncRef()

	isZero := pmb.DecRef()

	assert.False(testSuite.T(), isZero)
	assert.Equal(testSuite.T(), int32(1), pmb.RefCount())

	isZero = pmb.DecRef()

	assert.True(testSuite.T(), isZero)
	assert.Equal(testSuite.T(), int32(0), pmb.RefCount())
}

func (testSuite *PrefetchMemoryBlockTest) TestPrefetchMemoryBlockDecRefPanics() {
	pmb, err := createPrefetchBlock(MiB)
	require.Nil(testSuite.T(), err)

	assert.PanicsWithValue(testSuite.T(), "DecRef called more times than IncRef, resulting in a negative refCount.", func() {
		pmb.DecRef()
	})
}
