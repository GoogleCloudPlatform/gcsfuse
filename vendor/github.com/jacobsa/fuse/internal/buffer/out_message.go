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
	"reflect"
	"unsafe"

	"github.com/jacobsa/fuse/internal/fusekernel"
)

// OutMessage provides a mechanism for constructing a single contiguous fuse
// message from multiple segments, where the first segment is always a
// fusekernel.OutHeader message.
//
// Must be created with NewOutMessage. Exception: the zero value has
// Bytes() == nil.
type OutMessage struct {
	slice []byte
}

// Create a new buffer whose initial contents are a zeroed fusekernel.OutHeader
// message, and with room enough to grow by extra bytes.
func NewOutMessage(extra uintptr) (b OutMessage) {
	const headerSize = unsafe.Sizeof(fusekernel.OutHeader{})
	b.slice = make([]byte, headerSize, headerSize+extra)
	return
}

// Return a pointer to the header at the start of the buffer.
func (b *OutMessage) OutHeader() (h *fusekernel.OutHeader) {
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&b.slice))
	h = (*fusekernel.OutHeader)(unsafe.Pointer(sh.Data))
	return
}

// Grow the buffer by the supplied number of bytes, returning a pointer to the
// start of the new segment. The sum of the arguments given to Grow must not
// exceed the argument given to New when creating the buffer.
func (b *OutMessage) Grow(size uintptr) (p unsafe.Pointer) {
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&b.slice))
	p = unsafe.Pointer(sh.Data + uintptr(sh.Len))
	b.slice = b.slice[:len(b.slice)+int(size)]
	return
}

// Equivalent to growing by the length of p, then copying p into the new segment.
func (b *OutMessage) Append(p []byte) {
	sh := reflect.SliceHeader{
		Data: uintptr(b.Grow(uintptr(len(p)))),
		Len:  len(p),
		Cap:  len(p),
	}

	copy(*(*[]byte)(unsafe.Pointer(&sh)), p)
}

// Equivalent to growing by the length of s, then copying s into the new segment.
func (b *OutMessage) AppendString(s string) {
	sh := reflect.SliceHeader{
		Data: uintptr(b.Grow(uintptr(len(s)))),
		Len:  len(s),
		Cap:  len(s),
	}

	copy(*(*[]byte)(unsafe.Pointer(&sh)), s)
}

// Return a reference to the current contents of the buffer.
func (b *OutMessage) Bytes() []byte {
	return b.slice
}
