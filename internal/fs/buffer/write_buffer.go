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
	// InMemoryBufferThresholdMB is the upper limit on the size upto which the buffer should
	// be created in memory. Beyond this size, buffer should be on disk.
	InMemoryBufferThresholdMB     = 50
	NonSequentialWriteError       = "non-sequential writes are not supported with buffer"
	DataNotWrittenCompletelyError = "could not write all the data to buffer. Expected bytes to be written: %d, actually written: %d"
)

// ChunkSize (bytes) is the size of data to be written in 1 write call to GCS.
// Ensure ChunkSize <= 100MB to avoid memory bloat.
var ChunkSize int64

// WriteBuffer is an interface that buffers the data to be written to GCS during
// the write flow.
// WriteBuffer is used only in create new file flow with sequential writes and
// at any point in time, only 2x of the configured buffer size is stored in the
// write buffer.
type WriteBuffer interface {
	// Initialize assigns a buffer of 2*configured buffer size to the WriteBuffer.
	// Initialize should be called before any write calls to the buffer.
	Initialize(sizeInMB int)

	// WriteAt writes at an offset to the buffer.
	WriteAt(data []byte, offset int64) error
}
