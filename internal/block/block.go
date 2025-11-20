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
	"fmt"
	"io"
	"syscall"
)

// Block represents the buffer which holds the data.
type Block interface {
	// ReadSeeker is the interface that groups the basic Read and Seek methods.
	// It is used to read data from the block.
	io.ReadSeeker

	// Writer provides a way to write data to the block.
	io.Writer

	// GenBlock defines reuse and deallocation of the block.
	GenBlock

	// Size provides the current data size of the block. The capacity of the block
	// can be >= data_size.
	Size() int64

	// Cap returns the capacity of the block, kind of block-size.
	Cap() int64

	// Write writes the given data to block.
	Write(bytes []byte) (n int, err error)
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

	// readSeek is used to track the position for reading data.
	readSeek int64
}

func (m *memoryBlock) Reuse() {
	m.offset.end = 0
	m.offset.start = 0
	m.readSeek = 0
}

func (m *memoryBlock) Size() int64 {
	return m.offset.end - m.offset.start
}

func (m *memoryBlock) Cap() int64 {
	return int64(cap(m.buffer))
}

// Read reads data from the block into the provided byte slice.
// Please make sure to call Seek before calling Read if you want to read from a specific position.
func (m *memoryBlock) Read(bytes []byte) (int, error) {
	if m.readSeek < m.offset.start {
		return 0, fmt.Errorf("readSeek %d is less than start offset %d", m.readSeek, m.offset.start)
	}

	if m.readSeek >= m.offset.end {
		return 0, io.EOF
	}

	n := copy(bytes, m.buffer[m.readSeek:m.offset.end])
	m.readSeek += int64(n)

	// If readSeek is beyond the end of the block, return EOF early.
	if m.readSeek >= m.offset.end {
		return n, io.EOF
	}

	return n, nil
}

// Seek sets the readSeek position in the block.
// It returns the new readSeek position and an error if any.
// The whence argument specifies how the offset should be interpreted:
//   - io.SeekStart: offset is relative to the start of the block.
//   - io.SeekCurrent: offset is relative to the current readSeek position.
//   - io.SeekEnd: offset is relative to the end of the block.
//
// It returns an error if the whence value is invalid or if the new
// readSeek position is out of bounds.
func (m *memoryBlock) Seek(offset int64, whence int) (int64, error) {
	newReadSeek := m.readSeek
	switch whence {
	case io.SeekStart:
		m.readSeek = m.offset.start + offset
	case io.SeekCurrent:
		newReadSeek += offset
	case io.SeekEnd:
		newReadSeek = m.offset.end + offset
	default:
		return 0, fmt.Errorf("invalid whence value: %d", whence)
	}

	if newReadSeek < m.offset.start || newReadSeek > m.offset.end {
		return 0, fmt.Errorf("new readSeek position %d is out of bounds", newReadSeek)
	}

	m.readSeek = newReadSeek
	return m.readSeek, nil
}

func (m *memoryBlock) Write(bytes []byte) (int, error) {
	if m.Size()+int64(len(bytes)) > int64(cap(m.buffer)) {
		return 0, fmt.Errorf("received data more than capacity of the block")
	}

	n := copy(m.buffer[m.offset.end:], bytes)
	if n != len(bytes) {
		return 0, fmt.Errorf("error in copying the data to block. Expected %d, got %d", len(bytes), n)
	}

	m.offset.end += int64(len(bytes))
	return n, nil
}

func (m *memoryBlock) Deallocate() error {
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

// createBlock creates a new block.
func createBlock(blockSize int64) (Block, error) {
	prot, flags := syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE
	addr, err := syscall.Mmap(-1, 0, int(blockSize), prot, flags)
	if err != nil {
		return nil, fmt.Errorf("mmap error: %v", err)
	}

	mb := memoryBlock{
		buffer: addr,
		offset: offset{0, 0},
	}
	return &mb, nil
}
