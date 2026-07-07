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
	"context"
	"io"
)

// OffsetReader is an interface for reading data at an offset with a context.
type OffsetReader interface {
	Read(ctx context.Context, dst []byte, offset int64) (n int, err error)
}

type bufferWithOffset struct {
	buffer []byte
	offset int
}

// VectoredWriter implements io.Writer and writes data sequentially into a slice of bufferWithOffset.
type VectoredWriter struct {
	buffers []bufferWithOffset
	index   int
	pool    BufferPool
	maxSize int64
	written int64
}

// NewVectoredWriter creates a new VectoredWriter that allocates buffers on demand from pool.
func NewVectoredWriter(pool BufferPool, maxSize int64) *VectoredWriter {
	return &VectoredWriter{
		buffers: make([]bufferWithOffset, 0, 2),
		pool:    pool,
		maxSize: maxSize,
	}
}

// availableBuffer returns a slice of the current buffer available for writing.
// If the current buffer is full or no buffers exist, it allocates a new buffer
// from the pool. Returns nil if no more buffers can be allocated (e.g. pool is nil
// or maxSize is reached).
func (w *VectoredWriter) availableBuffer() []byte {
	for {
		if w.index < len(w.buffers) {
			sb := &w.buffers[w.index]
			if sb.offset < len(sb.buffer) {
				return sb.buffer[sb.offset:]
			}
			w.index++
			continue
		}
		if w.pool == nil || w.written >= w.maxSize {
			return nil
		}
		buf := w.pool.Get()
		if len(buf) == 0 {
			return nil
		}
		if int64(len(buf)) > w.maxSize-w.written {
			buf = buf[:w.maxSize-w.written]
		}
		w.buffers = append(w.buffers, bufferWithOffset{buffer: buf})
	}
}

// advance updates the offset of the current buffer and the total bytes written.
func (w *VectoredWriter) advance(n int) {
	w.buffers[w.index].offset += n
	w.written += int64(n)
}

func (w *VectoredWriter) Write(p []byte) (n int, err error) {
	for n < len(p) {
		buf := w.availableBuffer()
		if buf == nil {
			return n, io.ErrShortWrite
		}
		toCopy := copy(buf, p[n:])
		w.advance(toCopy)
		n += toCopy
	}
	return n, nil
}

func (w *VectoredWriter) readIntoBuffers(readFn func(buf []byte) (int, error)) (n int64, err error) {
	for {
		buf := w.availableBuffer()
		if buf == nil {
			break
		}
		var readNum int
		readNum, err = readFn(buf)
		if readNum > 0 {
			w.advance(readNum)
			n += int64(readNum)
		}
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
	}
	return n, err
}

// ReadFrom implements io.ReaderFrom. It reads data from r directly into the
// underlying buffers, avoiding intermediate allocations and double-copying.
func (w *VectoredWriter) ReadFrom(r io.Reader) (n int64, err error) {
	return w.readIntoBuffers(r.Read)
}

// ReadFromOffset reads data from r starting at offset directly into the
// underlying buffers, avoiding intermediate allocations and double-copying.
func (w *VectoredWriter) ReadFromOffset(ctx context.Context, r OffsetReader, offset int64) (n int64, err error) {
	return w.readIntoBuffers(func(buf []byte) (int, error) {
		return r.Read(ctx, buf, offset+w.written)
	})
}

// Buffers returns the slices of bytes actually written to.
func (w *VectoredWriter) Buffers() [][]byte {
	res := make([][]byte, 0, len(w.buffers))
	for _, sb := range w.buffers {
		if sb.offset > 0 {
			res = append(res, sb.buffer[:sb.offset])
		}
	}
	return res
}

// Release puts all allocated buffers back into the pool.
func (w *VectoredWriter) Release() {
	if w.pool == nil {
		return
	}
	for _, sb := range w.buffers {
		w.pool.Put(sb.buffer)
	}
	w.buffers = nil
}
