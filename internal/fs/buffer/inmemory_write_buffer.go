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
	"bytes"
	"fmt"
)

type InMemoryWriteBuffer struct {
	// Holds the data to be written to GCS. At any time, Buffer holds the data for
	// 2 GCS write calls.
	buffer *bytes.Buffer
	// [ currStartOffset, currEndOffset] is the range of file offset for which the
	// data is currently stored in buffer.
	currStartOffset int64
	currEndOffset   int64
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
	b.currEndOffset = ChunkSize
	if b.buffer == nil {
		b.buffer = bytes.NewBuffer(make([]byte, 0, BufferSize))
	}
}

func (b *InMemoryWriteBuffer) WriteAt(data []byte, startOffset int64) error {
	unreadBuffer := b.buffer.Bytes()
	size := int64(len(data))
	endOffset := startOffset + size

	// If startOffset is not in the range of current offset in buffer, trigger
	// temp file flow for writes.
	if startOffset < b.currStartOffset {
		// todo: finalise the write and trigger temp file flow.
		return nil
	}

	// get slice of unread buffer that we are currently writing to.
	bufferStartOffset := b.currStartOffset % BufferSize
	bufferEndOffset := b.currEndOffset % BufferSize
	unreadBuffer = unreadBuffer[bufferStartOffset:bufferEndOffset]

	// if data can be completely written without wrapping to previously written
	// buffer offsets (beyond the current block of buffer), write it.
	if endOffset <= b.currEndOffset {
		fmt.Println("startOffset = ", startOffset)
		fmt.Println("startOffset%BufferSize = ", startOffset%BufferSize)
		bufferEndOffset = 20
		fmt.Println("bufferEndOffset = ", bufferEndOffset)
		n := copy(unreadBuffer[startOffset%BufferSize:bufferEndOffset], data)
		if int64(n) != size {
			return fmt.Errorf("could not write all the data to buffer. "+
				"Expected bytes to be written: %d, actually written: %d", size, n)
		}
		fmt.Println("unreadBuffer = ", unreadBuffer[startOffset%BufferSize:bufferEndOffset])
		fmt.Println("data = ", data)
	} else {
		// Else write the chunk of data that can be written, check status of last
		// write to GCS, trigger write call for current buffer block, and write
		// remaining data by wrapping.
		l := b.currEndOffset - startOffset
		n := copy(unreadBuffer[startOffset%BufferSize:bufferEndOffset], data[:l])
		if int64(n) != l {
			return fmt.Errorf("could not write all the data to buffer. "+
				"Expected bytes to be written: %d, actually written: %d", l, n)
		}
		//TODO: logic to get status of last write and trigger async write.

		// Update buffer offsets.
		b.currStartOffset = b.currEndOffset + 1
		b.currEndOffset = b.currEndOffset + ChunkSize
		bufferStartOffset = b.currStartOffset % BufferSize
		bufferEndOffset = b.currEndOffset % BufferSize

		// Write the remaining data by wrapping over to the next buffer block.
		l = size - l
		n = copy(unreadBuffer[(startOffset+l)%BufferSize:bufferEndOffset], data[:l])
		if int64(n) != l {
			return fmt.Errorf("could not write all the data to buffer. "+
				"Expected bytes to be written: %d, actually written: %d", l, n)
		}
	}
	return nil
}
