// Copyright 2015 Google Inc. All Rights Reserved.
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
	"io"
	"syscall"
	"unsafe"

	"github.com/jacobsa/fuse/internal/fusekernel"
)

// All requests read from the kernel, without data, are shorter than
// this.
var pageSize int

// We size the buffer to have enough room for a fuse request plus data
// associated with a write request.
var bufSize int

func init() {
	pageSize = syscall.Getpagesize()
	bufSize = pageSize + MaxWriteSize
}

// An incoming message from the kernel, including leading fusekernel.InHeader
// struct. Provides storage for messages and convenient access to their
// contents.
type InMessage struct {
	remaining []byte
	storage   []byte
	size      int
}

// NewInMessage creates a new InMessage with its storage initialized.
func NewInMessage() *InMessage {
	return &InMessage{
		storage: make([]byte, bufSize),
	}
}

// Initialize with the data read by a single call to r.Read. The first call to
// Consume will consume the bytes directly after the fusekernel.InHeader
// struct.
func (m *InMessage) Init(r io.Reader) error {
	n, err := r.Read(m.storage[:])
	if err != nil {
		return err
	}

	// Make sure the message is long enough.
	const headerSize = unsafe.Sizeof(fusekernel.InHeader{})
	if uintptr(n) < headerSize {
		return fmt.Errorf("Unexpectedly read only %d bytes.", n)
	}

	m.size = n
	m.remaining = m.storage[headerSize:n]

	// Check the header's length.
	if int(m.Header().Len) != n {
		return fmt.Errorf(
			"Header says %d bytes, but we read %d",
			m.Header().Len,
			n)
	}

	return nil
}

// Return a reference to the header read in the most recent call to Init.
func (m *InMessage) Header() *fusekernel.InHeader {
	return (*fusekernel.InHeader)(unsafe.Pointer(&m.storage[0]))
}

// Return the number of bytes left to consume.
func (m *InMessage) Len() uintptr {
	return uintptr(len(m.remaining))
}

// Consume the next n bytes from the message, returning a nil pointer if there
// are fewer than n bytes available.
func (m *InMessage) Consume(n uintptr) unsafe.Pointer {
	if m.Len() == 0 || n > m.Len() {
		return nil
	}

	p := unsafe.Pointer(&m.remaining[0])
	m.remaining = m.remaining[n:]

	return p
}

// Equivalent to Consume, except returns a slice of bytes. The result will be
// nil if Consume would fail.
func (m *InMessage) ConsumeBytes(n uintptr) []byte {
	if n > m.Len() {
		return nil
	}

	b := m.remaining[:n]
	m.remaining = m.remaining[n:]

	return b
}

// Get the next n bytes after the message to use them as a temporary buffer
func (m *InMessage) GetFree(n int) []byte {
	if n <= 0 || n > len(m.storage)-m.size {
		return nil
	}
	return m.storage[m.size : m.size+n]
}
