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

package gcsx

import (
	"errors"
	"io"
)

var ErrInvalidAdvance = errors.New("VectoredReadBuffer: Invalid Buffer Advance")

// VectoredReadBuffer implements io.Writer and writes data sequentially into a slice of byte buffers.
// Invariant: For each slice b in buffers, len(b) tracks the number of valid written bytes,
// and cap(b) tracks the total capacity allocated from the pool. We never use 3-index slicing
// to shrink cap(b) so that Release() can safely return full-capacity buffers back to the pool.
type VectoredReadBuffer struct {
	buffers [][]byte
	pool    BufferPool
	maxSize int64
	written int64
}

// NewVectoredReadBuffer creates a new VectoredReadBuffer that allocates buffers on demand from pool.
func NewVectoredReadBuffer(pool BufferPool, maxSize int64) *VectoredReadBuffer {
	return &VectoredReadBuffer{
		buffers: make([][]byte, 0, 2),
		pool:    pool,
		maxSize: maxSize,
	}
}

// availableBuffer returns a slice of the current buffer available for writing.
// If the current buffer is full or no buffers exist, it allocates a new buffer
// from the pool. Returns nil if no more buffers can be allocated (e.g. pool is nil
// or maxSize is reached).
func (v *VectoredReadBuffer) availableBuffer() []byte {
	if v.written >= v.maxSize {
		return nil
	}

	avail := v.maxSize - v.written

	// 1. Check if the active (last) buffer has remaining capacity
	if len(v.buffers) > 0 {
		last := v.buffers[len(v.buffers)-1]
		if cap(last) > len(last) {
			rem := int(min(int64(cap(last)-len(last)), avail))
			return last[len(last) : len(last)+rem]
		}
	}

	// 2. Fetch new buffer from pool
	if v.pool != nil {
		if buf := v.pool.Get(); cap(buf) > 0 {
			// Append with length 0, preserving full pool capacity for Release().
			v.buffers = append(v.buffers, buf[:0])

			// Clamp the returned slice length (NOT capacity) to remaining maxSize.
			return buf[:min(int64(cap(buf)), avail)]
		}
	}
	return nil
}

// advance increases the length of the active buffer by n bytes and updates total written bytes.
// It returns ErrInvalidAdvance if n is negative, exceeds available capacity, or no buffers exist.
func (v *VectoredReadBuffer) advance(n int) error {
	if n == 0 {
		return nil
	}
	if len(v.buffers) > 0 {
		idx := len(v.buffers) - 1
		if n > 0 && n+len(v.buffers[idx]) <= cap(v.buffers[idx]) {
			v.buffers[idx] = v.buffers[idx][:len(v.buffers[idx])+n]
			v.written += int64(n)
			return nil
		}
	}
	return ErrInvalidAdvance
}

func (v *VectoredReadBuffer) Write(p []byte) (n int, err error) {
	for n < len(p) {
		buf := v.availableBuffer()
		if buf == nil {
			return n, io.ErrShortWrite
		}
		toCopy := copy(buf, p[n:])
		if err := v.advance(toCopy); err != nil {
			return n, err
		}
		n += toCopy
	}
	return n, nil
}

func (v *VectoredReadBuffer) readIntoBuffers(readFn func(buf []byte) (int, error)) (n int64, err error) {
	for {
		buf := v.availableBuffer()
		if buf == nil {
			break
		}
		var readNum int
		readNum, err = readFn(buf)
		if readNum > 0 {
			if advErr := v.advance(readNum); advErr != nil {
				return n, advErr
			}
			n += int64(readNum)
		}
		if err != nil {
			break
		}
	}
	if err == io.EOF {
		err = nil
	}
	return n, err
}

// ReadFrom implements io.ReaderFrom. It reads data from r directly into the
// underlying buffers, avoiding intermediate allocations and double-copying.
func (v *VectoredReadBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	return v.readIntoBuffers(r.Read)
}

// ReadFromAt reads data from r starting at offset directly into the
// underlying buffers, avoiding intermediate allocations and double-copying.
func (v *VectoredReadBuffer) ReadFromAt(r io.ReaderAt, offset int64) (n int64, err error) {
	return v.readIntoBuffers(func(buf []byte) (int, error) {
		return r.ReadAt(buf, offset+v.written)
	})
}

// Buffers returns the slices of bytes actually written to.
// This executes in O(1) time with 0 allocations.
func (v *VectoredReadBuffer) Buffers() [][]byte {
	// If the last buffer was allocated by availableBuffer() but nothing was written to it,
	// exclude it from the returned slice without reallocating.
	if len(v.buffers) > 0 && len(v.buffers[len(v.buffers)-1]) == 0 {
		return v.buffers[:len(v.buffers)-1]
	}
	return v.buffers
}

// Release puts all allocated buffers back into the pool.
func (v *VectoredReadBuffer) Release() {
	if v.pool == nil {
		return
	}
	for _, b := range v.buffers {
		v.pool.Put(b[:cap(b)])
	}
	v.buffers = nil
}
