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
	"sync"
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
	Size() int

	// Cap returns the capacity of the block, kind of block-size.
	Cap() int

	LimitedReadFrom(r io.Reader, n int) (int, error)
}

type memoryBlock struct {
	buffer []byte

	// readSeek is used to track the position for reading data.
	readSeek int64

	// currentSize is used to track the current size of the buffer (Protected by mu).
	currentSize int
	mu          sync.RWMutex
}

func (m *memoryBlock) Reuse() {
	m.currentSize = 0
	m.readSeek = 0
}

func (m *memoryBlock) Cap() int {
	return cap(m.buffer)
}

func (m *memoryBlock) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentSize
}

// Read reads data from the block into the provided byte slice.
// Please make sure to call Seek before calling Read if you want to read from a specific position.
func (m *memoryBlock) Read(bytes []byte) (int, error) {
	if m.readSeek < 0 {
		return 0, fmt.Errorf("readSeek %d is less than start offset 0", m.readSeek)
	}

	if m.readSeek >= int64(m.Size()) {
		return 0, io.EOF
	}

	// We should only read up to the current size of the block.
	readableBytes := m.buffer[m.readSeek:m.Size()]

	n := copy(bytes, readableBytes)
	m.readSeek += int64(n)

	// If we have read up to the end of the written data, return EOF.
	if m.readSeek >= int64(m.Size()) || n < len(bytes) {
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
		newReadSeek = offset
	case io.SeekCurrent:
		newReadSeek += offset
	case io.SeekEnd:
		newReadSeek = int64(m.Size()) + offset
	default:
		return 0, fmt.Errorf("invalid whence value: %d", whence)
	}

	if newReadSeek < 0 || newReadSeek > int64(m.Size()) {
		return 0, fmt.Errorf("new readSeek position %d is out of bounds", newReadSeek)
	}

	m.readSeek = newReadSeek
	return m.readSeek, nil
}

func (m *memoryBlock) Write(bytes []byte) (int, error) {
	if len(bytes) > m.Cap()-m.Size() {
		return 0, fmt.Errorf("received data more than capacity of the block")
	}
	n := copy(m.buffer[m.Size():], bytes)
	m.mu.Lock()
	m.currentSize += n
	m.mu.Unlock()
	return n, nil
}

func (m *memoryBlock) LimitedReadFrom(r io.Reader, limit int) (n int, err error) {
	currentSize := m.Size()
	if currentSize+limit > m.Cap() {
		return 0, fmt.Errorf("limit is more than remaining capacity of block")
	}
	limitedReadFrom, err := io.ReadFull(r, m.buffer[currentSize:currentSize+limit])
	m.mu.Lock()
	m.currentSize += limitedReadFrom
	m.mu.Unlock()
	return limitedReadFrom, err
}

func (m *memoryBlock) Deallocate() error {
	if m.buffer == nil {
		return fmt.Errorf("invalid buffer")
	}

	err := syscall.Munmap(m.buffer[:cap(m.buffer)])
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
	}
	return &mb, nil
}
