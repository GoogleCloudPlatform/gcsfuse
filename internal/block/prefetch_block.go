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
	"io"
	"sync/atomic"
	"syscall"
)

const (
	MiB = 1024 * 1024 // 1 MiB
)

// BlockStatus represents the status of a block.
// It contains the downloaded size of the block and an error
// that may have occurred during the block's operation.
type BlockStatus struct {
	Size int
	Err  error
	Complete bool
}

type PrefetchBlock interface {
	Block

	// Follows io.ReaderAt interface.
	// Here, off is relative to the start of the block.
	ReadAt(p []byte, off int64) (n int, err error)

	// ReadAtSlice provides a way to read data from the block by returning a
	// slice of the underlying buffer. The returned slice must not be modified.
	// The offset is relative to the start of the block.
	ReadAtSlice(off int64, size int) (p []byte, err error)

	// AbsStartOff returns the absolute start offset of the block.
	// Panics if the absolute start offset is not set.
	AbsStartOff() int64

	// SetAbsStartOff sets the absolute start offset of the block.
	// This should be called only once just after getting the block from the pool.
	// It returns an error if the startOff is negative or if it is already set.
	// TODO(princer): check if a way to set it as part of constructor.
	SetAbsStartOff(startOff int64) error

	// AwaitReady waits for the block to be ready to consume if the downloaded block size is more than requested block size.
	// It returns the size of the block and an error if any.
	AwaitReady(ctx context.Context, requestedBlockSize int) (BlockStatus, error)

	// NotifyReady is used by producer to update the block download progress.
	NotifyReady(val BlockStatus)

	// IncRef increments the reference count of the block.
	IncRef()

	// DecRef decrements the reference count of the block. It returns true if
	// the reference count reaches 0, otherwise false. Panics if the reference
	// count becomes negative.
	DecRef() bool

	// RefCount returns the current reference count of the block.
	RefCount() int32
}

type prefetchMemoryBlock struct {
	memoryBlock

	// Indicates the current block download size and any error if occurred.
	status BlockStatus

	// notification is a channel that notifies when the block is ready to consume.
	notification chan BlockStatus

	// Stores the absolute start offset of the block-segment in the file.
	absStartOff int64

	// refCount tracks the number of active references to the block.
	refCount atomic.Int32
}

func (pmb *prefetchMemoryBlock) Reuse() {
	pmb.memoryBlock.Reuse()

	pmb.notification = make(chan BlockStatus, (pmb.Cap()+MiB-1)/MiB)
	pmb.status = BlockStatus{Size: 0}
	pmb.absStartOff = -1
	pmb.refCount.Store(0)
}

// createPrefetchBlock creates a new PrefetchBlock.
func createPrefetchBlock(blockSize int64) (PrefetchBlock, error) {
	prot, flags := syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE
	addr, err := syscall.Mmap(-1, 0, int(blockSize), prot, flags)
	if err != nil {
		return nil, fmt.Errorf("createPrefetchBlock: Mmap: %w", err)
	}

	pmb := prefetchMemoryBlock{
		memoryBlock: memoryBlock{
			buffer: addr,
		},
		status:       BlockStatus{Size: 0},
		notification: make(chan BlockStatus, (blockSize+MiB-1)/MiB),
		absStartOff:  -1,
	}

	return &pmb, nil
}

// ReadAt reads data from the block at the specified offset.
// The offset is relative to the start of the block.
// It returns the number of bytes read and an error if any.
func (pmb *prefetchMemoryBlock) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= int64(pmb.Size()) {
		return 0, fmt.Errorf("prefetchMemoryBlock.ReadAt: offset %d is out of bounds for block size %d", off, pmb.Size())
	}

	// We should only read up to the current size of the block.
	readableBytes := pmb.buffer[off:pmb.Size()]
	n = copy(p, readableBytes)

	if n < len(p) {
		return n, io.EOF // We didn't fill the buffer, so we must have hit the end.
	}
	return n, nil
}

// ReadAtSlice returns a slice of the underlying buffer starting at the given
// offset, which is relative to the start of the block. This allows for reading
// data without an additional copy. The returned slice must not be modified by
// the caller.
//
// If the requested size exceeds the available data from the offset, it returns
// a slice of the available data and an io.EOF error. If the offset is out of
// bounds, it returns an error.
func (pmb *prefetchMemoryBlock) ReadAtSlice(off int64, size int) ([]byte, error) {
	if off < 0 || off >= int64(pmb.Size()) {
		return nil, fmt.Errorf("prefetchMemoryBlock.ReadAtSlice: offset %d is out of bounds for block size %d", off, pmb.Size())
	}

	dataEnd := off + int64(size)
	if dataEnd > int64(pmb.Size()) {
		dataEnd = int64(pmb.Size())
		return pmb.buffer[off:dataEnd], io.EOF
	}

	return pmb.buffer[off:dataEnd], nil
}

func (pmb *prefetchMemoryBlock) AbsStartOff() int64 {
	if pmb.absStartOff < 0 {
		panic("AbsStartOff is not set, it should be set before calling this method.")
	}
	return pmb.absStartOff
}

func (pmb *prefetchMemoryBlock) SetAbsStartOff(startOff int64) error {
	if startOff < 0 {
		return fmt.Errorf("SetAbsStartOff: negative startOff %d is not allowed", startOff)
	}

	// If absStartOff is already set, then return an error.
	if pmb.absStartOff >= 0 {
		return fmt.Errorf("SetAbsStartOff: absStartOff is already set, re-setting is not allowed")
	}

	pmb.absStartOff = startOff
	return nil
}

// AwaitReady waits for the block to be ready to consume.
// It returns the status of the block and an error if any.
func (pmb *prefetchMemoryBlock) AwaitReady(ctx context.Context, size int) (BlockStatus, error) {
	if (size > 0 && pmb.status.Size >= size) || pmb.status.Complete || pmb.status.Err != nil {
		return pmb.status, nil
	}

	for {
		select {
		case status, ok := <-pmb.notification:
			if !ok {
				// Channel closed, which means download is complete (or failed). Return last saved status.
				return pmb.status, nil
			}
			pmb.status = status
			if (size > 0 && pmb.status.Size >= size) || pmb.status.Complete || pmb.status.Err != nil {
				return pmb.status, nil
			}
		case <-ctx.Done():
			return pmb.status, ctx.Err()
		}
	}
}

// NotifyReady is used by the producer to mark the block as ready to consume.
// This can be called multiple times to provide progress updates.
func (pmb *prefetchMemoryBlock) NotifyReady(val BlockStatus) {
	select {
	case pmb.notification <- val:
	default:
		panic("The channel can't be full producer must not send more than channels capacity.")
	}
}

func (pmb *prefetchMemoryBlock) IncRef() {
	pmb.refCount.Add(1)
}

func (pmb *prefetchMemoryBlock) DecRef() bool {
	newRefCount := pmb.refCount.Add(-1)
	if newRefCount < 0 {
		panic("DecRef called more times than IncRef, resulting in a negative refCount.")
	}
	return newRefCount == 0
}

func (pmb *prefetchMemoryBlock) RefCount() int32 {
	return pmb.refCount.Load()
}
