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

// FakeBufferPool is a generic fake implementation of Pool used for testing across packages.
type FakeBufferPool struct {
	Buffers               [][]byte
	Idx                   int
	PutBuffers            [][]byte
	ReturnNilOnExhaustion bool
	DefaultBufferSize     int
}

// Get returns the next available buffer from Buffers. If all buffers have been consumed,
// it returns nil if ReturnNilOnExhaustion is true, or a newly allocated buffer of size
// DefaultBufferSize (or 1024 if DefaultBufferSize is <= 0).
func (p *FakeBufferPool) Get() []byte {
	if p.Idx < len(p.Buffers) {
		b := p.Buffers[p.Idx]
		p.Idx++
		return b
	}
	if p.ReturnNilOnExhaustion {
		return nil
	}
	size := p.DefaultBufferSize
	if size <= 0 {
		size = 1024
	}
	return make([]byte, size)
}

// Put records the returned buffer in PutBuffers.
func (p *FakeBufferPool) Put(b []byte) {
	p.PutBuffers = append(p.PutBuffers, b)
}
