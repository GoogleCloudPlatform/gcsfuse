// Copyright 2023 Google Inc. All Rights Reserved.
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

package buffer

import (
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/internal/logger"
)

type offset struct {
	// [ start, end) is the range of file offset for which the
	// data is currently stored in the buffer.
	start, end int64
}

func (o *offset) advanceBy(value int64) {
	o.start += value
	o.end += value
}

type inMemoryBuffer struct {
	buffer []byte
	offset offset
}

// InMemoryWriteBuffer implements WriteBuffer and is stored in memory.
// Refer to buffer.WriteBuffer for sample code usage.
type InMemoryWriteBuffer struct {
	// Holds the incoming data from kernel write calls, currently being buffered
	// for upload in the future.
	current inMemoryBuffer

	// Holds the data that is currently being uploaded asynchronously to GCS. Data
	// in this buffer will persist until the entire data is successfully uploaded.
	flushed inMemoryBuffer

	// Holds the last offset to which data is written.
	fileSize int64

	// Holds the buffer's size (in bytes) to be allocated to currentBuffer and flushedBuffer.
	bufferSize int64
}

// CreateInMemoryWriteBuffer returns a new InMemoryWriteBuffer.
// Memory is lazily allocated to current and flushed buffer on their first usage.
func CreateInMemoryWriteBuffer(bufferSizeMB uint) (*InMemoryWriteBuffer, error) {
	if bufferSizeMB == 0 {
		return nil, fmt.Errorf(ZeroSizeBufferError)
	}
	b := &InMemoryWriteBuffer{
		bufferSize: int64(bufferSizeMB * MiB),
	}
	// TODO: set mtime attribute.
	logger.Debugf("TODO: set mtime attribute.")
	return b, nil
}

// Allocates memory to currentBuffer if it hasn't been allocated already.
func (b *InMemoryWriteBuffer) ensureCurrentBuffer() error {
	if b.bufferSize == 0 {
		return fmt.Errorf(ZeroSizeBufferError)
	}
	if b.current.buffer == nil {
		b.current.buffer = make([]byte, 0, b.bufferSize)
		b.current.offset.end = b.bufferSize
	}
	return nil
}

// Allocates memory to flushedBuffer if it hasn't been allocated already.
func (b *InMemoryWriteBuffer) ensureFlushedBuffer() error {
	if b.bufferSize == 0 {
		return fmt.Errorf(ZeroSizeBufferError)
	}
	if b.flushed.buffer == nil {
		b.flushed.buffer = make([]byte, 0, b.bufferSize)
	}
	return nil
}

func (b *InMemoryWriteBuffer) WriteAt(content []byte, offset int64) error {
	contentSize := int64(len(content))
	if contentSize == 0 {
		return nil
	}

	// Ensure b.currentBuffer != nil
	if err := b.ensureCurrentBuffer(); err != nil {
		return fmt.Errorf("ensureCurrentBuffer: %w", err)
	}

	if offset < b.current.offset.start ||
		offset >= b.current.offset.end+b.bufferSize {
		// TODO: finalise write and trigger temp file flow.
		logger.Debugf("Non-sequential write encountered. TODO: Switch to " +
			"temp-file flow insteading of erroring out.")
		return fmt.Errorf(NonSequentialWriteError)
	}

	var contentWrittenSoFar int64
	for {
		n := b.writePartialContentToBuffer(content[contentWrittenSoFar:], offset+contentWrittenSoFar)
		contentWrittenSoFar += n
		if contentWrittenSoFar == int64(len(content)) {
			// all content written successfully to buffer.
			return nil
		}
		//TODO: Wait/timeout on last write to GCS. If unsuccessful, trigger temp file flow.
		logger.Debugf("TODO: Wait/timeout on last write to GCS and if " +
			"unsuccessful, trigger temp file flow.")
		if err := b.advanceToNextChunk(); err != nil {
			return fmt.Errorf("advanceToNextChunk: %w", err)
		}
		// TODO: trigger async upload to GCS for flushedBuffer.
		logger.Debugf("TODO: trigger async upload to GCS for flushedBuffer.")
	}
}

// Helper method to copy received content to currentBuffer.
func (b *InMemoryWriteBuffer) copyDataToBuffer(contentStartOffset int64, content []byte) {
	contentSize := int64(len(content))
	contentEndOffset := contentStartOffset + contentSize
	si := contentStartOffset % b.bufferSize
	ei := si + contentSize

	copy(b.current.buffer[si:ei], content)
	b.updateFileSize(contentEndOffset)
}

func (b *InMemoryWriteBuffer) advanceToNextChunk() error {
	// ensure b.flushed.buffer != nil
	if err := b.ensureFlushedBuffer(); err != nil {
		return fmt.Errorf("ensureFlushedBuffer: %w", err)
	}

	// Move current buffer to flushed buffer.
	b.flushed.buffer, b.current.buffer = b.current.buffer, b.flushed.buffer
	b.flushed.offset = b.current.offset

	// Make current buffer ready for new writes coming from kernel.
	// TODO: revisit this when implementing on-disk-write-buffer.
	clear(b.current.buffer[0:b.bufferSize])
	b.current.offset.advanceBy(b.bufferSize)
	return nil
}

// After successful copy of data to buffer, increments the bytes written so far.
func (b *InMemoryWriteBuffer) updateFileSize(endOffset int64) {
	if endOffset > b.fileSize {
		b.fileSize = endOffset
	}
}

func (b *InMemoryWriteBuffer) writePartialContentToBuffer(content []byte, offset int64) int64 {
	if offset < b.current.offset.start ||
		offset >= b.current.offset.end {
		// Nothing to copy to current buffer.
		return 0
	}

	// copy content to current buffer
	capacityOfCurrentBuffer := b.current.offset.end - offset
	l := min(capacityOfCurrentBuffer, int64(len(content)))
	b.copyDataToBuffer(offset, content[:l])
	return l
}
