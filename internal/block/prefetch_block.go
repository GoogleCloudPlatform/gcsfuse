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
	"errors"
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

// Status of the download.
const (
	BlockStatusDownloaded        int = iota + 1 // Download of this block is complete
	BlockStatusDownloadFailed                   // Download of this block has failed
	BlockStatusDownloadCancelled                // Download of this block has been cancelled
)

var PartialBlockReadErr = errors.New("partial block read error")

type OffsetRange struct {
	Start, End int64
}

type PrefetchBlock interface {
	Block

	// ReadAt reads len(p) bytes into p starting at offset off.
	// It returns the number of bytes read (0 <= n <= len(p)) and any
	// error encountered that caused the read to stop.  The
	// returned error is always nil if n == len(p).
	// Here, offset is relative to block startOffset.
	ReadAt(p []byte, off int64) (n int, err error)

	// AwaitReady waits for the block to be ready to consume.
	// It returns the status of the block and an error if any.
	AwaitReady(ctx context.Context) (int, error)

	// NotifyReady is used by consumer to marks the block is ready to consume.
	// The value indicates the status of the block:
	// - BlockStatusDownloaded: Download of this block is complete.
	// - BlockStatusDownloadFailed: Download of this block has failed.
	// - BlockStatusDownloadCancelled: Download of this block has been cancelled.
	NotifyReady(val int)

	// AbsStartOff returns the absolute start offset of the block.
	AbsStartOff() int64

	// SetAbsStartOff sets the absolute start offset of the block.
	// This should be called only once just after getting the block from the pool.
	SetAbsStartOff(startOff int64)
}

type prefetchBlock struct {
	memoryBlock

	// notification is a channel that notifies when the block is ready to consume.
	notification chan int

	// Represents the portion with offset [id * blockSize, (id+1) * blockSize).
	id int64

	status int // Status of the block download
}

func (p *prefetchBlock) Reuse() {
	p.memoryBlock.Reuse()
	p.notification = make(chan int, 1)
	p.id = 0
}

// createPrefetchBlock creates a new block.
func CreatePrefetchBlock(blockSize int64) (PrefetchBlock, error) {

	mb, err := createMemoryBlock(blockSize)
	if err != nil {
		return nil, fmt.Errorf("error creating memory block: %w", err)
	}

	pb := prefetchBlock{
		memoryBlock:  *mb,
		notification: make(chan int, 1),
		id:           0,
	}

	return &pb, nil
}

func (p *prefetchBlock) NotifyReady(val int) {
	if p.notification == nil {
		return
	}

	select {
	case p.notification <- val:
	default:
		logger.Warnf("Expected an empty channel while writing an block notification: %d", val)
	}
}

func (p *prefetchBlock) AwaitReady(ctx context.Context) (int, error) {
	if p.notification == nil {
		return p.status, nil
	}

	select {
	case val := <-p.notification:
		close(p.notification) // Close the channel to prevent further writes.
		p.status = val
		p.notification = nil
		return val, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func (p *prefetchBlock) AbsStartOff() int64 {
	if p.id < 0 {
		return 0
	}
	return p.id * p.Cap()
}

func (p *prefetchBlock) SetAbsStartOff(startOff int64) {
	if startOff < 0 {
		logger.Errorf("Invalid start offset %d for prefetch block", startOff)
		return
	}

	if p.Cap() == 0 {
		logger.Errorf("Cannot set start offset for an empty prefetch block")
		return
	}

	p.id = startOff / p.Cap()
}

func (p *prefetchBlock) ReadAt(pBytes []byte, off int64) (n int, err error) {
	if off < 0 || off >= p.Cap() {
		return 0, fmt.Errorf("offset %d is out of bounds for block size %d", off, p.Cap())
	}

	if len(pBytes) == 0 {
		return 0, nil // No bytes to read, return immediately.
	}

	n = copy(pBytes, p.memoryBlock.buffer[off:])

	if n < len(pBytes) {
		return n, PartialBlockReadErr
	}
	return n, nil
}
