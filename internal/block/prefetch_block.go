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

// BlockStatus represents the status of a block.
// It contains the state of the block and an error
// that may have occurred during the block's operation.
type BlockStatus struct {
	State BlockState
	Err   error
}

// BlockState represents the state of the block.
type BlockState int

const (
	BlockStateInProgress     BlockState = iota // Download of this block is in progress
	BlockStateDownloaded                       // Download of this block is complete
	BlockStateDownloadFailed                   // Download of this block has failed
)

type PrefetchBlock interface {
	Block

	// Follows io.ReaderAt interface.
	// Here, off is relative to the start of the block.
	ReadAt(p []byte, off int64) (n int, err error)

	// ReadAtSlice returns a slice of the block's buffer if the read can be
	// satisfied. It avoids a copy. The returned slice should not be modified.
	// Here, off is relative to the start of the block.
	ReadAtSlice(size int, off int64) (p []byte, n int, err error)

	// AbsStartOff returns the absolute start offset of the block.
	// Panics if the absolute start offset is not set.
	AbsStartOff() int64

	// SetAbsStartOff sets the absolute start offset of the block.
	// This should be called only once just after getting the block from the pool.
	// It returns an error if the startOff is negative or if it is already set.
	// TODO(princer): check if a way to set it as part of constructor.
	SetAbsStartOff(startOff int64) error

	// AwaitReady waits for the block to be ready to consume.
	// It returns the status of the block and an error if any.
	AwaitReady(ctx context.Context) (BlockStatus, error)

	// NotifyReady is used by producer to mark the block as ready to consume.
	// The value indicates the status of the block:
	// - BlockStatusDownloaded: Download of this block is complete.
	// - BlockStatusDownloadFailed: Download of this block has failed.
	NotifyReady(val BlockStatus)

	// IncrementRef increments the reference count of the block.
	IncrementRef()

	// DecrementRef decrements the reference count of the block.
	DecrementRef() int32

	// RefCount returns the current reference count of the block.
	RefCount() int32
}

type prefetchMemoryBlock struct {
	memoryBlock

	// Indicates if block is in progress, downloaded, download failed or download cancelled.
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

	pmb.notification = make(chan BlockStatus, 1)
	pmb.status = BlockStatus{State: BlockStateInProgress}
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

	mb := memoryBlock{
		buffer: addr,
		offset: offset{0, 0},
	}

	pmb := prefetchMemoryBlock{
		memoryBlock:  mb,
		status:       BlockStatus{State: BlockStateInProgress},
		notification: make(chan BlockStatus, 1),
		absStartOff:  -1,
	}

	return &pmb, nil
}

// ReadAt reads data from the block at the specified offset.
// The offset is relative to the start of the block.
// It returns the number of bytes read and an error if any.
func (pmb *prefetchMemoryBlock) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= pmb.Size() {
		return 0, fmt.Errorf("prefetchMemoryBlock.ReadAt: offset %d is out of bounds for block size %d", off, pmb.Size())
	}

	n = copy(p, pmb.buffer[pmb.offset.start+off:pmb.offset.end])

	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// ReadAtSlice returns a slice of the underlying buffer.
// The offset is relative to the start of the block.
// It returns the slice and an error if any.
func (pmb *prefetchMemoryBlock) ReadAtSlice(size int, off int64) ([]byte, int, error) {
	if off < 0 || (off > 0 && off >= pmb.Size()) {
		return nil, 0, fmt.Errorf("prefetchMemoryBlock.ReadAtSlice: offset %d is out of bounds for block size %d", off, pmb.Size())
	}

	var err error
	remData := pmb.Size() - off
	if int64(size) > remData {
		size = int(remData)
		err = io.EOF
	}

	dataStart := pmb.offset.start + off
	dataEnd := dataStart + int64(size)
	return pmb.buffer[dataStart:dataEnd], size, err
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
func (pmb *prefetchMemoryBlock) AwaitReady(ctx context.Context) (BlockStatus, error) {
	select {
	case val, ok := <-pmb.notification:
		if !ok {
			return pmb.status, nil
		}

		// Close the notification channel to prevent further notifications.
		close(pmb.notification)
		// Save the last status for subsequent AwaitReady calls.
		pmb.status = val

		return pmb.status, nil
	case <-ctx.Done():
		return BlockStatus{State: BlockStateInProgress}, ctx.Err()
	}
}

// NotifyReady is used by the producer to mark the block as ready to consume.
// This should be called only once to notify the consumer.
// If called multiple times, it will panic - either because of writing to the
// closed channel or blocking due to writing over full notification channel.
func (pmb *prefetchMemoryBlock) NotifyReady(val BlockStatus) {
	select {
	case pmb.notification <- val:
	default:
		panic("Expected to notify only once, but got multiple notifications.")
	}
}

func (pmb *prefetchMemoryBlock) IncrementRef() {
	pmb.refCount.Add(1)
}

func (pmb *prefetchMemoryBlock) DecrementRef() int32 {
	return pmb.refCount.Add(-1)
}

func (pmb *prefetchMemoryBlock) RefCount() int32 {
	return pmb.refCount.Load()
}
