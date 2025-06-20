// Copyright 2025 Google LLC
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
	"context"
	"fmt"
	"syscall"
)

type PrefetchBlock interface {
	Block

	// Returns complete buffer slice to read any specific data.
	Data() []byte

	// NotificationChannel returns a channel that notifies when the block is
	// ready to consume.
	NotificationChannel() <-chan int

	// GetId returns the id of the block. The id is used to identify the portion of
	// the block that is being prefetched. The id is used to calculate the offset
	// of the block in the buffer, where the data is stored.
	GetId() int64

	// SetId sets the id of the block. The id is used to identify the portion of
	SetId(id int64)
}

type prefetchBlock struct {
	memoryBlock

	// notification is a channel that notifies when the block is ready to consume.
	notification chan int

	// cancelFunc is used to cancel the prefetch operation if needed.
	cancelFunc context.CancelCauseFunc

	// Represents the portion with offset [id * blockSize, (id+1) * blockSize).
	id int64
}

func (p *prefetchBlock) Reuse() {
	p.memoryBlock.Reuse()
	p.notification = make(chan int, 1)
	p.cancelFunc = nil
	p.id = 0
}

func (p *prefetchBlock) Data() []byte {
	return p.buffer
}

func (p *prefetchBlock) NotificationChannel() <-chan int {
	return p.notification
}

// createPrefetchBlock creates a new block.
func createPrefetchBlock(blockSize int64) (PrefetchBlock, error) {
	prot, flags := syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE
	addr, err := syscall.Mmap(-1, 0, int(blockSize), prot, flags)
	if err != nil {
		return nil, fmt.Errorf("mmap error: %v", err)
	}

	pb := prefetchBlock{
		memoryBlock: memoryBlock{
			buffer: addr,
			offset: offset{0, 0},
		},
		notification: make(chan int, 1),
		cancelFunc:   nil,
		id:           0,
	}

	return &pb, nil
}

func (p *prefetchBlock) GetId() int64 {
	return p.id
}

func (p *prefetchBlock) SetId(id int64) {
	p.id = id
}
