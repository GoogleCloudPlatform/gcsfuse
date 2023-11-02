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
	dataSize := int64(len(content))
	receivedContentEndOffset := offset + dataSize
	if dataSize == 0 {
		return nil
	}

	// Ensure b.currentBuffer != nil
	if err := b.ensureCurrentBuffer(); err != nil {
		return err
	}

	// If receivedContentStartOffset is not in the range of offsets stored in current
	// buffer, trigger temp file flow for writes.
	if offset < b.current.offset.start ||
		offset > b.current.offset.end {
		// TODO: finalise write and trigger temp file flow.
		logger.Debugf("TODO: finalise write and trigger temp file flow.")
		return fmt.Errorf(NonSequentialWriteError)
	}

	// If data can be completely written to the current buffer, write it.
	if receivedContentEndOffset <= b.current.offset.end {
		return b.copyDataToBuffer(offset, receivedContentEndOffset, content)
	}
	// Else, write the chunk of data that can be written to the current buffer block.
	l := b.current.offset.end - offset
	err := b.copyDataToBuffer(offset, b.current.offset.end, content[:l])
	if err != nil {
		return err
	}
	//TODO: get status of last write to GCS. If unsuccessful, trigger temp file flow.
	logger.Debugf("TODO: get status of last write to GCS and if " +
		"unsuccessful, trigger temp file flow.")
	if err = b.advanceToNextChunk(); err != nil {
		return err
	}
	// TODO: trigger async upload to GCS for flushedBuffer.
	logger.Debugf("TODO: trigger async upload to GCS for flushedBuffer.")

	// Write the remaining data to currentBuffer.
	return b.copyDataToBuffer(b.current.offset.start, receivedContentEndOffset, content[l:])
}

// Copies received content to currentChunk if it has capacity.
func (b *InMemoryWriteBuffer) copyDataToBuffer(contentStartOffset, contentEndOffset int64, content []byte) error {
	dataSize := int64(len(content))
	si := contentStartOffset % b.bufferSize
	ei := si + dataSize
	if ei > b.bufferSize {
		return fmt.Errorf(NotEnoughSpaceInCurrentBuffer)
	}

	copy(b.current.buffer[si:ei], content)
	b.updateFileSize(contentEndOffset)
	return nil
}

func (b *InMemoryWriteBuffer) advanceToNextChunk() error {
	// ensure b.flushed.buffer != nil
	if err := b.ensureFlushedBuffer(); err != nil {
		return err
	}

	// Move current buffer to flushed buffer.
	b.flushed.buffer, b.current.buffer = b.current.buffer, b.flushed.buffer
	b.flushed.offset.start = b.current.offset.start
	b.flushed.offset.end = b.current.offset.end

	// Make current buffer ready for new writes coming from kernel.
	clear(b.current.buffer)
	b.current.offset.start = b.current.offset.end
	b.current.offset.end += b.bufferSize
	return nil
}

// After successful copy of data to buffer, increments the bytes written so far.
func (b *InMemoryWriteBuffer) updateFileSize(endOffset int64) {
	if endOffset > b.fileSize {
		b.fileSize = endOffset
	}
}
