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

import "io"

// VectoredWriter implements io.Writer and writes data sequentially into a slice of byte slices.
type VectoredWriter struct {
	buffers [][]byte
	offset  int
	index   int
}

// NewVectoredWriter creates a new VectoredWriter that writes into the provided buffers.
func NewVectoredWriter(buffers [][]byte) *VectoredWriter {
	return &VectoredWriter{
		buffers: buffers,
	}
}

func (w *VectoredWriter) Write(p []byte) (n int, err error) {
	for n < len(p) {
		if w.index >= len(w.buffers) {
			return n, io.ErrShortWrite
		}
		buf := w.buffers[w.index]
		avail := len(buf) - w.offset
		if avail <= 0 {
			w.index++
			w.offset = 0
			continue
		}
		toCopy := len(p) - n
		if toCopy > avail {
			toCopy = avail
		}
		copy(buf[w.offset:w.offset+toCopy], p[n:n+toCopy])
		w.offset += toCopy
		n += toCopy
	}
	return n, nil
}

// ReadFrom implements io.ReaderFrom. It reads data from r directly into the
// underlying buffers, avoiding intermediate allocations and double-copying.
func (w *VectoredWriter) ReadFrom(r io.Reader) (n int64, err error) {
	for w.index < len(w.buffers) {
		buf := w.buffers[w.index]
		avail := len(buf) - w.offset
		if avail <= 0 {
			w.index++
			w.offset = 0
			continue
		}

		var readNum int
		readNum, err = r.Read(buf[w.offset:])
		w.offset += readNum
		n += int64(readNum)

		if err != nil {
			if err == io.EOF {
				return n, nil
			}
			return n, err
		}
	}
	return n, nil
}

// GetVectoredBuffers returns the buffers to be used for the read operation and the actual size to read.
// The returned size will not exceed the total capacity of the buffers, or the provided limit (if limit > 0).
func GetVectoredBuffers(req *ReadRequest, limit int64) ([][]byte, int64) {
	sizeToRead := req.GetReadSize(limit)

	if len(req.Buffers) > 0 {
		return req.Buffers, sizeToRead
	}
	return [][]byte{req.Buffer[:sizeToRead]}, sizeToRead
}
