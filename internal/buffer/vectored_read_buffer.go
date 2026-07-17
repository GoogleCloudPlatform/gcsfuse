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

package buffer

import (
	"io"
)

// VectoredReadBuffer accumulates data across a sequence of pooled byte buffers up to maxSize.
// It implements io.Writer and io.ReaderFrom, and provides ReadFromAt to support zero-allocation,
// zero-copy reads from random-access sources (io.ReaderAt). Buffers are allocated lazily from pool
// on demand as data is written or read.
//
// Invariant: For each slice b in buffers, len(b) tracks the number of valid populated (written or read)
// bytes, and cap(b) tracks the total capacity allocated from the pool. We never use 3-index slicing
// to shrink cap(b) so that Release() can safely return full-capacity buffers back to the pool.
type VectoredReadBuffer struct {
	// buffers holds the slices of bytes allocated from pool.
	// For each buffer b in buffers, len(b) represents the bytes actually populated (written or read),
	// while cap(b) represents the full capacity allocated from the pool.
	buffers [][]byte

	// pool is the underlying pool from which new byte buffers are allocated on demand.
	// If nil, no new buffers will be allocated once existing buffer capacity is exhausted.
	pool Pool

	// maxSize is the maximum total number of bytes allowed to be populated across all buffers.
	// Once written reaches maxSize, subsequent writes will return io.ErrShortWrite or reads will stop.
	maxSize int64

	// written tracks the cumulative number of bytes populated (written or read) into buffers so far.
	written int64
}

// NewVectoredReadBuffer creates a new VectoredReadBuffer that allocates buffers on demand from pool.
func NewVectoredReadBuffer(pool Pool, maxSize int64) *VectoredReadBuffer {
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

	// 1. Check if the active (last) buffer has remaining capacity.
	if len(v.buffers) > 0 {
		lastBuf := v.buffers[len(v.buffers)-1]
		if remCap := cap(lastBuf) - len(lastBuf); remCap > 0 {
			// Return the unpopulated tail of lastBuf, clamped by remaining maxSize.
			rem := int(min(int64(remCap), avail))
			return lastBuf[len(lastBuf) : len(lastBuf)+rem]
		}
	}

	// 2. Fetch new buffer from pool.
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

func (v *VectoredReadBuffer) Write(p []byte) (int, error) {
	var totalWritten int
	for totalWritten < len(p) {
		buf := v.availableBuffer()
		if buf == nil {
			return totalWritten, io.ErrShortWrite
		}
		// Copy up to len(buf) (the available space in this chunk) bytes from p into the buffer.
		bytesCopied := copy(buf, p[totalWritten:])

		// Expand the active buffer's length to include the newly written bytes.
		idx := len(v.buffers) - 1
		lastBuf := v.buffers[idx]
		v.buffers[idx] = lastBuf[:len(lastBuf)+bytesCopied]

		// Track cumulative bytes written across all buffers and total bytes written in this call.
		v.written += int64(bytesCopied)
		totalWritten += bytesCopied
	}
	return totalWritten, nil
}

func (v *VectoredReadBuffer) readIntoBuffers(readFn func(buf []byte) (int, error)) (int64, error) {
	var totalRead int64
	for {
		buf := v.availableBuffer()
		if buf == nil {
			return totalRead, nil
		}
		// Read up to len(buf) (the available space in this chunk) bytes directly into the buffer.
		bytesRead, err := readFn(buf)
		if bytesRead > 0 {
			// Expand the active buffer's length to include the newly read bytes.
			idx := len(v.buffers) - 1
			lastBuf := v.buffers[idx]
			v.buffers[idx] = lastBuf[:len(lastBuf)+bytesRead]

			// Track cumulative bytes read across all buffers and total bytes read in this call.
			v.written += int64(bytesRead)
			totalRead += int64(bytesRead)
		}
		if err != nil {
			if err == io.EOF {
				return totalRead, nil
			}
			return totalRead, err
		}
	}
}

// ReadFrom implements io.ReaderFrom. It reads data from r directly into the
// underlying buffers, avoiding intermediate allocations and double-copying.
func (v *VectoredReadBuffer) ReadFrom(r io.Reader) (int64, error) {
	return v.readIntoBuffers(r.Read)
}

// ReadFromAt reads data from r starting at offset directly into the
// underlying buffers, avoiding intermediate allocations and double-copying.
func (v *VectoredReadBuffer) ReadFromAt(r io.ReaderAt, offset int64) (int64, error) {
	return v.readIntoBuffers(func(buf []byte) (int, error) {
		return r.ReadAt(buf, offset+v.written)
	})
}

// Buffers returns the slices of bytes actually written to.
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
