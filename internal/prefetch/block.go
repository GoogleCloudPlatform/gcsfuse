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

package prefetch

import (
	"container/list"
	"fmt"
	"syscall"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
)

// State of the block.
const (
	BlockFlagFresh       uint16 = iota
	BlockFlagDownloading        // Block is being downloaded
	BlockFlagFailed             // Block upload/download has failed
)

// State of the download.
const (
	BlockStatusDownloaded     int = iota + 1 // Download of this block is complete
	BlockStatusDownloadFailed                // Download of this block has failed
)

type BitMap16 uint16

// IsSet : Check whether the given bit is set or not
func (bm BitMap16) IsSet(bit uint16) bool { return (bm & (1 << bit)) != 0 }

// Set : Set the given bit in bitmap
func (bm *BitMap16) Set(bit uint16) { *bm |= (1 << bit) }

// Clear : Clear the given bit from bitmap
func (bm *BitMap16) Clear(bit uint16) { *bm &= ^(1 << bit) }

// Reset : Reset the whole bitmap by setting it to 0
func (bm *BitMap16) Reset() { *bm = 0 }

// Block is a memory mapped buffer with its state to hold data
type Block struct {
	offset uint64        // Start offset of the data this block holds
	id     int64         // Id of the block i.e. (offset / block size)
	state  chan int      // Channel depicting data has been read for this block or not
	flags  BitMap16      // Various states of the block
	data   []byte        // Data read from blob
	node   *list.Element // node representation of this block in the list inside prefetcher

	endOffset uint64 // End offset of the data this block holds
}

// AllocateBlock creates a new memory mapped buffer for the given size
func AllocateBlock(size uint64) (*Block, error) {
	if size == 0 {
		return nil, fmt.Errorf("invalid size")
	}

	prot, flags := syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE
	addr, err := syscall.Mmap(-1, 0, int(size), prot, flags)

	if err != nil {
		return nil, fmt.Errorf("mmap error: %v", err)
	}

	block := &Block{
		data:  addr,
		state: nil,
		id:    -1,
		node:  nil,
	}

	// we do not create channel here, as that will be created when buffer is retrieved
	// reinit will always be called before use and that will create the channel as well.
	block.flags.Reset()
	block.flags.Set(BlockFlagFresh)
	return block, nil
}

// Delete cleans up the memory mapped buffer
func (b *Block) Delete() error {
	if b.data == nil {
		return fmt.Errorf("invalid buffer")
	}

	err := syscall.Munmap(b.data)
	b.data = nil
	if err != nil {
		// if we get here, there is likely memory corruption.
		return fmt.Errorf("munmap error: %v", err)
	}

	return nil
}

func (b *Block) Write(bytes []byte) (n int, err error) {
	if b.endOffset+uint64(len(bytes)) > uint64(cap(b.data)) {
		// Add info log to print above value, just info log
		logger.Infof("Write: b.endOffset: %d, b.offset: %d, len(bytes): %d, cap(b.data): %d", b.endOffset, b.offset, len(bytes), cap(b.data))

		return 0, fmt.Errorf("received data more than capacity of the block")
	}

	n = copy(b.data[b.endOffset:], bytes)
	if n != len(bytes) {
		return 0, fmt.Errorf("error in copying the data to block. Expected %d, got %d", len(bytes), n)
	}

	b.endOffset += uint64(len(bytes))
	return n, nil
}

// ReUse reinits the Block by recreating its channel
func (b *Block) ReUse() {
	b.id = -1
	b.offset = 0
	b.flags.Reset()
	b.flags.Set(BlockFlagFresh)
	b.state = make(chan int, 1)
}

// Ready marks this Block is now ready for reading by its first reader (data download completed)
func (b *Block) Ready(val int) {
	select {
	case b.state <- val:
		break
	default:
		break
	}
}

// Unblock marks this Block is ready to be read in parllel now
func (b *Block) Unblock() {
	close(b.state)
}

// Mark this block as failed
func (b *Block) Failed() {
	b.flags.Set(BlockFlagFailed)
}

// Check this block as failed
func (b *Block) IsFailed() bool {
	return b.flags.IsSet(BlockFlagFailed)
}
