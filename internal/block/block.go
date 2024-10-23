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
	"bytes"
	"fmt"
	"io"
	"syscall"
)

// Block represents the buffer which holds the data.
type Block interface {
	// Reuse resets the blocks for reuse.
	Reuse()

	// Size provides the current data size of the block. The capacity of the block
	// can be >= data_size.
	Size() int64

	// Write writes the given data to block.
	Write(bytes []byte) error

	// Reader interface helps in copying the data directly to storage.writer
	// while uploading to GCS.
	Reader() io.Reader

	DeAllocate() error
}

// TODO: check if we need offset or just storing end is sufficient. We might need
// for handling ordered writes. It will be decided after ordered writes design.
type offset struct {
	start, end int64
}

type memoryBlock struct {
	Block
	buffer []byte
	offset offset
}

func (m *memoryBlock) Reuse() {
	clear(m.buffer)

	m.offset.end = 0
	m.offset.start = 0
}

func (m *memoryBlock) Size() int64 {
	return m.offset.end - m.offset.start
}
func (m *memoryBlock) Write(bytes []byte) error {
	if m.Size()+int64(len(bytes)) > int64(cap(m.buffer)) {
		return fmt.Errorf("received data more than capacity of the block")
	}

	n := copy(m.buffer[m.offset.end:], bytes)
	if n != len(bytes) {
		return fmt.Errorf("error in copying the data to block. Expected %d, got %d", len(bytes), n)
	}

	m.offset.end += int64(len(bytes))
	return nil
}

func (m *memoryBlock) Reader() io.Reader {
	return bytes.NewReader(m.buffer[0:m.offset.end])
}

func (m *memoryBlock) DeAllocate() error {
	if m.buffer == nil {
		return fmt.Errorf("invalid buffer")
	}

	err := syscall.Munmap(m.buffer)
	m.buffer = nil
	if err != nil {
		// if we get here, there is likely memory corruption.
		return fmt.Errorf("munmap error: %v", err)
	}

	return nil
}
