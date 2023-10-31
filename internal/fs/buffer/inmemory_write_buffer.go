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
)

// InMemoryWriteBuffer implements WriteBuffer and is stored in memory.
type InMemoryWriteBuffer struct {
	// Holds the incoming data from kernel write calls.
	currentBuffer []byte
	// [ currentBufferStartOffset, currentBufferEndOffset) is the range of file offset for which the
	// data is currently stored in currentBuffer.
	currentBufferStartOffset int64
	currentBufferEndOffset   int64

	// Holds the data that is sent to go storage client for upload. Data in this buffer
	// will persist until the entire data is successfully uploaded to GCS.
	flushedBuffer []byte
	// [ flushedBufferStartOffset, flushedBufferEndOffset) is the range of file offset for which the
	// data is currently stored in flushedBuffer.
	flushedBufferStartOffset int64
	flushedBufferEndOffset   int64

	// Holds the last offset to which data is written.
	fileSize int64
}

// CreateInMemoryWriteBuffer creates a buffer with InMemoryWriteBuffer.buffer
// set to nil. Memory is allocated to the buffer when the first write call comes.
// This avoids unnecessarily bloating GCSFuse memory consumption.
//
// To allocate memory to the buffer, use WriteBuffer.InitializeBuffer.
func CreateInMemoryWriteBuffer() *InMemoryWriteBuffer {
	b := &InMemoryWriteBuffer{}
	// TODO: set mtime attribute.
	return b
}

func (b *InMemoryWriteBuffer) InitializeBuffer(sizeInMB int) {
	if ChunkSize == 0 {
		ChunkSize = int64(sizeInMB * MiB)
	}
	if b.currentBuffer == nil {
		b.currentBuffer = make([]byte, 0, ChunkSize)
		b.currentBufferEndOffset = ChunkSize
	}
	if b.flushedBuffer == nil {
		b.flushedBuffer = make([]byte, 0, ChunkSize)
	}
}

func (b *InMemoryWriteBuffer) WriteAt(content []byte, receivedContentStartOffset int64) error {
	dataSize := int64(len(content))
	receivedContentEndOffset := receivedContentStartOffset + dataSize

	// If receivedContentStartOffset is not in the range of offsets stored in current
	// buffer, trigger temp file flow for writes.
	if receivedContentStartOffset < b.currentBufferStartOffset ||
		b.currentBufferEndOffset < receivedContentStartOffset {
		// TODO: finalise write and trigger temp file flow.
		return fmt.Errorf(NonSequentialWriteError)
	}

	// If data can be completely written to the current buffer block, write it.
	if receivedContentEndOffset <= b.currentBufferEndOffset {
		return b.copyDataToBuffer(receivedContentStartOffset, receivedContentEndOffset, content)
	} else {
		// Below logic is written with the assumption that the data received in a
		// single write call from the kernel will never exceed beyond buffer block size.
		// This should always hold true because kernel writes never exceed 1 MiB size.

		// Write the chunk of data that can be written to the current buffer block.
		l := b.currentBufferEndOffset - receivedContentStartOffset
		if err := b.copyDataToBuffer(receivedContentStartOffset, b.currentBufferEndOffset, content[:l]); err != nil {
			return err
		}

		//TODO: get status of last write to GCS.
		// If unsuccessful, trigger temp file flow.
		// Else:
		b.swapBuffers()
		// TODO: trigger async upload to GCS for flushedBuffer.

		// Update buffer offsets for next buffer block.
		b.currentBufferStartOffset = b.currentBufferEndOffset
		b.currentBufferEndOffset += ChunkSize
		// Write the remaining data to currentBuffer.
		return b.copyDataToBuffer(b.currentBufferStartOffset, receivedContentEndOffset, content[l:])
	}
}

func (b *InMemoryWriteBuffer) copyDataToBuffer(contentStartOffset, contentEndOffset int64, content []byte) error {
	dataSize := int64(len(content))
	si := contentStartOffset % ChunkSize
	ei := si + dataSize

	n := copy(b.currentBuffer[si:ei], content)
	if int64(n) != dataSize {
		return fmt.Errorf(DataNOtWrittenCompletelyError, dataSize, n)
	}
	b.updateFileSize(contentEndOffset)
	return nil
}

func (b *InMemoryWriteBuffer) swapBuffers() {
	b.flushedBufferStartOffset = b.currentBufferStartOffset
	b.flushedBufferEndOffset = b.currentBufferEndOffset
	b.flushedBuffer, b.currentBuffer = b.currentBuffer, b.flushedBuffer
	clear(b.currentBuffer)
}

// After successful copy of data to buffer, increments the bytes written so far.
func (b *InMemoryWriteBuffer) updateFileSize(endOffset int64) {
	if endOffset > b.fileSize {
		b.fileSize = endOffset
	}
}
