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

type InMemoryWriteBuffer struct {
	// Holds the data to be written to GCS. At any time, Buffer holds the data for
	// 2 GCS write calls.
	buffer []byte
	// [ bufferStartOffset, bufferEndOffset] is the range of file offset for which the
	// data is currently stored in buffer.
	bufferStartOffset int64
	bufferEndOffset   int64

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
	ChunkSize = int64(sizeInMB * MiB)
	BufferSize = 2 * ChunkSize
	b.bufferEndOffset = ChunkSize
	if b.buffer == nil {
		b.buffer = make([]byte, 0, BufferSize)
	}
}

func (b *InMemoryWriteBuffer) WriteAt(data []byte, dataStartOffset int64) error {
	dataSize := int64(len(data))
	dataEndOffset := dataStartOffset + dataSize

	// If dataStartOffset is not in the range of offsets stored in buffer block,
	// trigger temp file flow for writes.
	if dataStartOffset < b.bufferStartOffset || b.bufferEndOffset < dataStartOffset {
		// TODO: finalise write and trigger temp file flow.
		return fmt.Errorf(NonSequentialWriteError)
	}

	// If data can be completely written without wrapping to previously written
	// buffer block (beyond the current block of buffer), write it.
	if dataEndOffset <= b.bufferEndOffset {
		return b.copyDataToBuffer(dataStartOffset, dataEndOffset, data)
	} else {
		// Below logic is written with the assumption that the data received in a
		// single write call from the kernel will never exceed beyond buffer block size.

		// Write the chunk of data that can be written to the current buffer block.
		l := b.bufferEndOffset - dataStartOffset
		if err := b.copyDataToBuffer(dataStartOffset, b.bufferEndOffset, data[:l]); err != nil {
			return err
		}

		//TODO: get status of last write to GCS, and trigger async write for current block.

		// Update buffer offsets for next buffer block.
		b.bufferStartOffset = b.bufferEndOffset
		b.bufferEndOffset += ChunkSize
		// Write the remaining data by wrapping over to the next buffer block.
		return b.copyDataToBuffer(b.bufferStartOffset, dataEndOffset, data[l:])
	}
}

func (b *InMemoryWriteBuffer) copyDataToBuffer(dataStartOffset, dataEndOffset int64, data []byte) error {
	dataSize := int64(len(data))
	si := dataStartOffset % BufferSize
	ei := si + dataSize

	n := copy(b.buffer[si:ei], data)
	if int64(n) != dataSize {
		return fmt.Errorf(DataNOtWrittenCompletelyError, dataSize, n)
	}
	b.updateFileSize(dataEndOffset)
	return nil
}

// After successful copy of data to buffer, increments the bytes written so far.
func (b *InMemoryWriteBuffer) updateFileSize(endOffset int64) {
	if endOffset > b.fileSize {
		b.fileSize = endOffset
	}
}
