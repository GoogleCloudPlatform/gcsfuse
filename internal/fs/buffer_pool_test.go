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
)

func TestBufferPool_Basic(t *testing.T) {
	staticCount := 5
	bufferSize := 1024 // 1KB for testing
	// Start with all pre-allocated to test basic flow.
	bp := NewBufferPool(staticCount, staticCount, bufferSize)
	defer bp.Close()

	// 1. Get all static buffers.
	var staticBufs [][]byte
	for i := 0; i < staticCount; i++ {
		buf := bp.Get()
		if len(buf) != bufferSize {
			t.Errorf("expected buffer size %d, got %d", bufferSize, len(buf))
		}
		staticBufs = append(staticBufs, buf)
	}

	// Verify they are all recognized as static.
	for _, buf := range staticBufs {
		ptr := uintptr(unsafe.Pointer(&buf[0]))
		if _, ok := bp.staticMap.Load(ptr); !ok {
			t.Errorf("expected buffer at %x to be in staticMap", ptr)
		}
	}

	// 2. Get one more buffer (should trigger fallback).
	fallbackBuf := bp.Get()
	if len(fallbackBuf) != bufferSize {
		t.Errorf("expected fallback buffer size %d, got %d", bufferSize, len(fallbackBuf))
	}
	fbPtr := uintptr(unsafe.Pointer(&fallbackBuf[0]))
	if _, ok := bp.staticMap.Load(fbPtr); ok {
		t.Errorf("expected fallback buffer at %x to NOT be in staticMap", fbPtr)
	}

	// 3. Put them all back.
	for _, buf := range staticBufs {
		bp.Put(buf)
	}
	bp.Put(fallbackBuf)

	// 4. Get 5 buffers again. Since we put them back, they should be the static ones.
	var staticBufs2 [][]byte
	for i := 0; i < staticCount; i++ {
		buf := bp.Get()
		staticBufs2 = append(staticBufs2, buf)
	}

	for _, buf := range staticBufs2 {
		ptr := uintptr(unsafe.Pointer(&buf[0]))
		if _, ok := bp.staticMap.Load(ptr); !ok {
			t.Errorf("expected reused buffer at %x to be in staticMap", ptr)
		}
	}

	// Clean up
	for _, buf := range staticBufs2 {
		bp.Put(buf)
	}
}

func TestBufferPool_LazyAllocation(t *testing.T) {
	maxStatic := 5
	initialStatic := 2
	bufferSize := 1024
	bp := NewBufferPool(maxStatic, initialStatic, bufferSize)
	defer bp.Close()

	if bp.allocated != int32(initialStatic) {
		t.Errorf("expected allocated to be %d, got %d", initialStatic, bp.allocated)
	}

	// Get 2 buffers. They should come from the pre-allocated pool.
	buf1 := bp.Get()
	buf2 := bp.Get()

	if bp.allocated != int32(initialStatic) {
		t.Errorf("expected allocated to remain %d, got %d", initialStatic, bp.allocated)
	}

	// Get 3 more buffers. These should trigger lazy allocation of static buffers.
	buf3 := bp.Get()
	buf4 := bp.Get()
	buf5 := bp.Get()

	if bp.allocated != int32(maxStatic) {
		t.Errorf("expected allocated to reach %d, got %d", maxStatic, bp.allocated)
	}

	// Verify all 5 are static.
	bufs := [][]byte{buf1, buf2, buf3, buf4, buf5}
	for _, buf := range bufs {
		ptr := uintptr(unsafe.Pointer(&buf[0]))
		if _, ok := bp.staticMap.Load(ptr); !ok {
			t.Errorf("expected buffer at %x to be static", ptr)
		}
	}

	// Get a 6th buffer. It should be from the fallback pool.
	fallbackBuf := bp.Get()
	fbPtr := uintptr(unsafe.Pointer(&fallbackBuf[0]))
	if _, ok := bp.staticMap.Load(fbPtr); ok {
		t.Errorf("expected 6th buffer to be from fallback pool")
	}

	// Put them all back.
	for _, buf := range bufs {
		bp.Put(buf)
	}
	bp.Put(fallbackBuf)

	// Since we returned all 5 static buffers, the channel should now contain 5 buffers.
	if len(bp.staticBuffers) != maxStatic {
		t.Errorf("expected channel to have %d buffers, got %d", maxStatic, len(bp.staticBuffers))
	}
}

func TestBufferPool_PutSlicedBuffer(t *testing.T) {
	staticCount := 2
	bufferSize := 1024
	bp := NewBufferPool(staticCount, staticCount, bufferSize)
	defer bp.Close()

	// Get all static buffers to drain the channel.
	buf1 := bp.Get()
	buf2 := bp.Get()
	ptr1 := uintptr(unsafe.Pointer(&buf1[0]))

	// Slice and put buf1 back.
	sliced := buf1[:100]
	bp.Put(sliced)

	// Get it again. Since buf2 is still held, we must get buf1 back.
	reused := bp.Get()
	reusedPtr := uintptr(unsafe.Pointer(&reused[0]))

	if ptr1 != reusedPtr {
		t.Errorf("expected to reuse the same buffer, got different pointer")
	}
	if len(reused) != bufferSize {
		t.Errorf("expected reused buffer to be restored to size %d, got %d", bufferSize, len(reused))
	}

	bp.Put(reused)
	bp.Put(buf2)
}
