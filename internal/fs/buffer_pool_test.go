// Copyright 2026 Google LLC
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

package fs

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBufferPool_StartupState(t *testing.T) {
	// Arrange
	maxStatic := 5
	bufferSize := 1024

	// Act
	bp := NewBufferPool(maxStatic, bufferSize)
	defer bp.Close()

	// Assert
	assert.Equal(t, int32(0), bp.allocated)
	bp.mu.Lock()
	stackLen := len(bp.staticBuffers)
	bp.mu.Unlock()
	assert.Equal(t, 0, stackLen)
}

func TestBufferPool_Get_OnDemandAllocation(t *testing.T) {
	// Arrange
	maxStatic := 5
	bufferSize := 1024
	bp := NewBufferPool(maxStatic, bufferSize)
	defer bp.Close()

	// Act
	buf := bp.Get()

	// Assert
	require.Len(t, buf, bufferSize)
	ptr := uintptr(unsafe.Pointer(&buf[0]))
	_, registered := bp.staticMap.Load(ptr)
	assert.True(t, registered)
	assert.Equal(t, int32(1), bp.allocated)
}

func TestBufferPool_Get_ReuseFromStack(t *testing.T) {
	// Arrange
	maxStatic := 5
	bufferSize := 1024
	bp := NewBufferPool(maxStatic, bufferSize)
	defer bp.Close()
	originalBuf := bp.Get()
	originalPtr := uintptr(unsafe.Pointer(&originalBuf[0]))
	bp.Put(originalBuf) // Put it back to stack

	// Act
	reusedBuf := bp.Get()

	// Assert
	reusedPtr := uintptr(unsafe.Pointer(&reusedBuf[0]))
	assert.Equal(t, originalPtr, reusedPtr)
	assert.Equal(t, int32(1), bp.allocated)
}

func TestBufferPool_Get_FallbackToSyncPool(t *testing.T) {
	// Arrange
	maxStatic := 2
	bufferSize := 1024
	bp := NewBufferPool(maxStatic, bufferSize)
	defer bp.Close()
	// Exhaust static pool
	_ = bp.Get()
	_ = bp.Get()

	// Act
	fallbackBuf := bp.Get()

	// Assert
	require.Len(t, fallbackBuf, bufferSize)
	fbPtr := uintptr(unsafe.Pointer(&fallbackBuf[0]))
	_, registered := bp.staticMap.Load(fbPtr)
	assert.False(t, registered)
	assert.Equal(t, int32(2), bp.allocated)
}

func TestBufferPool_Put_RestoresCapacityOfSlicedBuffer(t *testing.T) {
	// Arrange
	maxStatic := 2
	bufferSize := 1024
	bp := NewBufferPool(maxStatic, bufferSize)
	defer bp.Close()
	originalBuf := bp.Get()
	originalPtr := uintptr(unsafe.Pointer(&originalBuf[0]))

	// Act
	slicedBuf := originalBuf[:100]
	bp.Put(slicedBuf)
	reusedBuf := bp.Get()

	// Assert
	reusedPtr := uintptr(unsafe.Pointer(&reusedBuf[0]))
	assert.Equal(t, originalPtr, reusedPtr)
	assert.Len(t, reusedBuf, bufferSize)
}

func TestBufferPool_Put_StaticBuffer(t *testing.T) {
	// Arrange
	maxStatic := 2
	bufferSize := 1024
	bp := NewBufferPool(maxStatic, bufferSize)
	defer bp.Close()
	buf := bp.Get()
	ptr := uintptr(unsafe.Pointer(&buf[0]))

	// Act
	bp.Put(buf)

	// Assert
	bp.mu.Lock()
	defer bp.mu.Unlock()
	require.Len(t, bp.staticBuffers, 1)
	returnedPtr := uintptr(unsafe.Pointer(&bp.staticBuffers[0][0]))
	assert.Equal(t, ptr, returnedPtr)
}

func TestBufferPool_Put_FallbackToSyncPool(t *testing.T) {
	// Arrange
	maxStatic := 1
	bufferSize := 1024
	bp := NewBufferPool(maxStatic, bufferSize)
	defer bp.Close()
	_ = bp.Get()            // Exhaust static pool
	fallbackBuf := bp.Get() // Get fallback buffer from sync.Pool

	// Act
	bp.Put(fallbackBuf)

	// Assert
	bp.mu.Lock()
	stackLen := len(bp.staticBuffers)
	bp.mu.Unlock()
	assert.Equal(t, 0, stackLen)
	assert.Equal(t, int32(1), bp.allocated)
}

func TestBufferPool_Close_Success(t *testing.T) {
	// Arrange
	maxStatic := 3
	bufferSize := 1024
	bp := NewBufferPool(maxStatic, bufferSize)
	// Allocate buffers to ensure they are registered in the static map
	_ = bp.Get()
	_ = bp.Get()

	// Act
	err := bp.Close()

	// Assert
	assert.NoError(t, err)
}

func BenchmarkBufferPool_Contention_Warmed(b *testing.B) {
	staticCount := 50
	bufferSize := 1024 * 1024 // 1MB
	bp := NewBufferPool(staticCount, bufferSize)
	defer bp.Close()

	// Warm up the pool so all static buffers are allocated.
	var bufs [][]byte
	for i := 0; i < staticCount; i++ {
		bufs = append(bufs, bp.Get())
	}
	for _, buf := range bufs {
		bp.Put(buf)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bp.Get()
			if len(buf) != bufferSize {
				b.Fatalf("unexpected buffer size: %d", len(buf))
			}
			bp.Put(buf)
		}
	})
}

func BenchmarkBufferPool_Contention_Cold(b *testing.B) {
	staticCount := 50
	bufferSize := 1024 * 1024 // 1MB
	bp := NewBufferPool(staticCount, bufferSize)
	defer bp.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bp.Get()
			if len(buf) != bufferSize {
				b.Fatalf("unexpected buffer size: %d", len(buf))
			}
			bp.Put(buf)
		}
	})
}
