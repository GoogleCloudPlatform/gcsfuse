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

const (
	// MiB is the multiplication factor to convert MiB to bytes.
	MiB = 1024 * 1024
	// maxInMemoryBufferSizeMB is the upper limit on the size upto which the
	// buffer would be created in memory. Beyond this size, buffer would be on disk.
	MaxInMemoryBufferSizeMB       = 50
	NonSequentialWriteError       = "non-sequential writes are not supported with WriteBuffer"
	NotEnoughSpaceInCurrentBuffer = "not enough space in currentBuffer to write entire content"
	ZeroSizeBufferError           = "buffer of size 0 cannot be created"
)

// WriteBuffer is an interface that buffers the data to be written to GCS during
// the write flow.
// WriteBuffer is used only in create new file flow with sequential writes and
// at any point in time, only 2x of the configured buffer size is stored in the
// write buffer.
// Sample usage:
// // Create in-memory write or on-disk write buffer
// writeBuffer, err := buffer.CreateInMemoryWriteBuffer(bufferSize)
// // check err
//
//	for n times {
//		err = buffer.WriteAt([]byte("content"), offset)
//		// check err
//	}
type WriteBuffer interface {
	// WriteAt writes at an offset to the buffer.
	WriteAt(data []byte, offset int64) error
}
