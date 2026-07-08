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

type bufferWithOffset struct {
	buffer []byte
	offset int
}

// VectoredWriter implements io.Writer and writes data sequentially into a slice of bufferWithOffset.
type VectoredWriter struct {
	buffers []bufferWithOffset
	index   int
}

// NewVectoredWriter creates a new VectoredWriter that writes into the provided buffers.
func NewVectoredWriter(buffers [][]byte) *VectoredWriter {
	w := &VectoredWriter{buffers: make([]bufferWithOffset, len(buffers))}
	for i, b := range buffers {
		w.buffers[i].buffer = b
	}
	return w
}

func (w *VectoredWriter) Write(p []byte) (n int, err error) {
	for n < len(p) {
		if w.index >= len(w.buffers) {
			return n, io.ErrShortWrite
		}
		sb := &w.buffers[w.index]
		avail := len(sb.buffer) - sb.offset
		if avail <= 0 {
			w.index++
			continue
		}
		toCopy := min(len(p)-n, avail)
		copy(sb.buffer[sb.offset:sb.offset+toCopy], p[n:n+toCopy])
		sb.offset += toCopy
		n += toCopy
	}
	return n, nil
}

// ReadFrom implements io.ReaderFrom. It reads data from r directly into the
// underlying buffers, avoiding intermediate allocations and double-copying.
func (w *VectoredWriter) ReadFrom(r io.Reader) (n int64, err error) {
	for w.index < len(w.buffers) {
		sb := &w.buffers[w.index]
		if sb.offset >= len(sb.buffer) {
			w.index++
			continue
		}

		var readNum int
		readNum, err = r.Read(sb.buffer[sb.offset:])
		sb.offset += readNum
		n += int64(readNum)

		if err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
	}
	return n, err
}
