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
)

type InMemoryWriteBuffer struct {
	// Holds the data to be written to GCS. At any time, Buffer holds the data for
	// 2 GCS write calls.
	buffer *bytes.Buffer
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
	ChunkSize = sizeInMB * MiB
	if b.buffer == nil {
		b.buffer = bytes.NewBuffer(make([]byte, 0, 2*sizeInMB*MiB))
	}
}

func (b *InMemoryWriteBuffer) WriteAt(data []byte, offset int64) error {
	return nil
}
