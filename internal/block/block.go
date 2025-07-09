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
	Write(bytes []byte) (n int, err error)

	// Reader interface helps in copying the data directly to storage.writer
	// while uploading to GCS.
	Reader() io.Reader

	Deallocate() error

	// Follows io.ReaderAt interface.
	// Here, off is relative to the start of the block.
	ReadAt(p []byte, off int64) (n int, err error)

	// AbsStartOff returns the absolute start offset of the block.
	// Panics if the absolute start offset is not set.
	AbsStartOff() int64

	// SetAbsStartOff sets the absolute start offset of the block.
	// This should be called only once just after getting the block from the pool.
	// It returns an error if the startOff is negative or if it is already set.
	// TODO(princer): check if a way to set it as part of constructor.
	SetAbsStartOff(startOff int64) error
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

	// Stores the absolute start offset of the block-segment in the file.
	absStartOff int64
}

func (m *memoryBlock) Reuse() {
	clear(m.buffer)

	m.offset.end = 0
	m.offset.start = 0
	m.absStartOff = -1
}

func (m *memoryBlock) Size() int64 {
	return m.offset.end - m.offset.start
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

func (m *memoryBlock) Reader() io.Reader {
	return bytes.NewReader(m.buffer[0:m.offset.end])
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
		buffer:      addr,
		offset:      offset{0, 0},
		absStartOff: -1,
	}
	return &mb, nil
}

// ReadAt reads data from the block at the specified offset.
// The offset is relative to the start of the block.
// It returns the number of bytes read and an error if any.
func (m *memoryBlock) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= m.Size() {
		return 0, fmt.Errorf("offset %d is out of bounds for block size %d", off, m.Size())
	}

	n = copy(p, m.buffer[m.offset.start+off:m.offset.end])

	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (m *memoryBlock) AbsStartOff() int64 {
	if m.absStartOff < 0 {
		panic("AbsStartOff is not set, it should be set before calling this method.")
	}
	return m.absStartOff
}

func (m *memoryBlock) SetAbsStartOff(startOff int64) error {
	if startOff < 0 {
		return fmt.Errorf("startOff cannot be negative, got %d", startOff)
	}

	// If absStartOff is already set, then return an error.
	if m.absStartOff >= 0 {
		return fmt.Errorf("AbsStartOff is already set, it should be set only once.")
	}

	m.absStartOff = startOff
	return nil
}
