package block

import (
	"fmt"
	"io"
	"os"
)

// Block - Represents the buffer which holds the data.
type Block interface {
	// Reader interface. This is required to copy the data to
	//  storage.writer.
	io.Reader

	// Reuse resets the blocks for reuse. In case of memory block, the same block
	// will be reused. For diskBlock, we will close and create a new file.
	Reuse(chan Block)

	// Size provides the current size of the block
	Size() int

	// Write writes the given data to block.
	Write(bytes []byte) error
}

type memoryBlock struct {
	Block
	buffer         []byte
	offset         offset
	readerPosition int
}

func (m *memoryBlock) Reuse(blocksCh chan Block) {
	for i := range m.buffer {
		m.buffer[i] = 0 // 0 is for null character
	}
	//m.buffer = make([]byte, 0, len(m.buffer))
	m.offset = offset{
		start: 0,
		end:   0,
	}
	m.readerPosition = 0
	blocksCh <- m
}

func (m *memoryBlock) Size() int {
	return m.offset.end

	//if m.offset.end != 0 && int64(m.offset.end%cap(m.buffer)) == 0 {
	//	return int64(cap(m.buffer))
	//}
	//return int64(m.offset.end % cap(m.buffer))
}
func (m *memoryBlock) Write(bytes []byte) error {
	if m.offset.end+len(bytes) > cap(m.buffer) {
		return fmt.Errorf("received buffer more than capacity")
	}

	n := copy(m.buffer[m.offset.end:m.offset.end+len(bytes)], bytes)
	if n != len(bytes) {
		return fmt.Errorf("could not copy entire thing. Expected %d, got %d", len(bytes), n)
	}
	m.offset.end += len(bytes)
	return nil
}

func (m *memoryBlock) Read(p []byte) (n int, err error) {
	if m.readerPosition >= m.Size() {
		return 0, io.EOF // End of data
	}

	n = copy(p, m.buffer[m.readerPosition:m.Size()]) // Copy data from the byte array to p
	m.readerPosition += n                            // Update the reading position

	return n, nil
}

type diskBlock struct {
	Block
	file   *os.File
	offset offset
}

type offset struct {
	start, end int
}
