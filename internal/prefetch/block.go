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
	"context"
	"fmt"
	"syscall"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
)

// Status of the download.
const (
	BlockStatusDownloaded        int = iota + 1 // Download of this block is complete
	BlockStatusDownloadFailed                   // Download of this block has failed
	BlockStatusDownloadCancelled                // Download of this block has been cancelled
)

// Block is a memory mapped buffer with its state to hold data
type Block struct {
	offset uint64   // Start offset of the data this block holds
	id     int64    // Id of the block i.e. (offset / block size)
	status chan int // used to pass status of major download event.
	data   []byte   // Data read from blob

	endOffset  uint64 // End offset of the data this block holds
	writeSeek  uint64
	cancelFunc context.CancelFunc
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
		data:   addr,
		status: nil,
		id:     -1,
	}

	// we do not create channel here, as that will be created when buffer is retrieved
	// reinit will always be called before use and that will create the channel as well.
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
	if b.writeSeek+uint64(len(bytes)) > uint64(cap(b.data)) {
		// Add info log to print above value, just info log
		logger.Infof("Write: b.writeSeek: %d, b.offset: %d, len(bytes): %d, cap(b.data): %d", b.writeSeek, b.offset, len(bytes), cap(b.data))

		return 0, fmt.Errorf("received data more than capacity of the block")
	}

	n = copy(b.data[b.writeSeek:], bytes)
	if n != len(bytes) {
		return 0, fmt.Errorf("error in copying the data to block. Expected %d, got %d", len(bytes), n)
	}

	b.writeSeek += uint64(len(bytes))
	return n, nil
}

// ReUse reinits the Block by recreating its channel
func (b *Block) ReUse() {
	b.id = -1
	b.offset = 0
	b.endOffset = 0
	b.writeSeek = 0
	b.status = make(chan int, 1)
}

func (b *Block) Cancel() {
	if b.cancelFunc != nil {
		b.cancelFunc()
		b.cancelFunc = nil
	}
}

// Ready marks this Block is now ready for reading by its first reader (data download completed)
func (b *Block) Ready(val int) {
	select {
	case b.status <- val:
		break
	default:
		break
	}
}

// Unblock marks this Block is ready to be read in parllel now
func (b *Block) Unblock() {
	close(b.status)
}
