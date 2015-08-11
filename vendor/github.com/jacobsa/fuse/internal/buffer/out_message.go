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
	"log"
	"reflect"
	"unsafe"

	"github.com/jacobsa/fuse/internal/fusekernel"
)

const outHeaderSize = unsafe.Sizeof(fusekernel.OutHeader{})

// OutMessage structs begin life with Len() == OutMessageInitialSize.
const OutMessageInitialSize = outHeaderSize

// We size out messages to be large enough to hold a header for the response
// plus the largest read that may come in.
const outMessageSize = outHeaderSize + MaxReadSize

// OutMessage provides a mechanism for constructing a single contiguous fuse
// message from multiple segments, where the first segment is always a
// fusekernel.OutHeader message.
//
// Must be initialized with Reset.
type OutMessage struct {
	offset  uintptr
	storage [outMessageSize]byte
}

// Make sure alignment works out correctly, at least for the header.
func init() {
	a := unsafe.Alignof(OutMessage{})
	o := unsafe.Offsetof(OutMessage{}.storage)
	e := unsafe.Alignof(fusekernel.OutHeader{})

	if a%e != 0 || o%e != 0 {
		log.Panicf("Bad alignment or offset: %d, %d, need %d", a, o, e)
	}
}

// Reset the message so that it is ready to be used again. Afterward, the
// contents are solely a zeroed header.
func (m *OutMessage) Reset() {
	m.offset = OutMessageInitialSize
	memclr(unsafe.Pointer(&m.storage), OutMessageInitialSize)
}

// Return a pointer to the header at the start of the message.
func (b *OutMessage) OutHeader() (h *fusekernel.OutHeader) {
	h = (*fusekernel.OutHeader)(unsafe.Pointer(&b.storage))
	return
}

// Grow the buffer by the supplied number of bytes, returning a pointer to the
// start of the new segment, which is zeroed. If there is no space left, return
// the nil pointer.
func (b *OutMessage) Grow(size uintptr) (p unsafe.Pointer) {
	p = b.GrowNoZero(size)
	if p != nil {
		memclr(p, size)
	}

	return
}

// Equivalent to Grow, except the new segment is not zeroed. Use with caution!
func (b *OutMessage) GrowNoZero(size uintptr) (p unsafe.Pointer) {
	if outMessageSize-b.offset < size {
		return
	}

	p = unsafe.Pointer(uintptr(unsafe.Pointer(&b.storage)) + b.offset)
	b.offset += size

	return
}

// Shrink to the supplied size. Panic if the size is greater than Len() or less
// than OutMessageInitialSize.
func (b *OutMessage) ShrinkTo(n uintptr) {
	if n < OutMessageInitialSize || n > b.offset {
		panic(fmt.Sprintf("ShrinkTo(%d) out of range for offset %d", n, b.offset))
	}

	b.offset = n
}

// Equivalent to growing by the length of p, then copying p over the new
// segment. Panics if there is not enough room available.
func (b *OutMessage) Append(src []byte) {
	p := b.GrowNoZero(uintptr(len(src)))
	if p == nil {
		panic(fmt.Sprintf("Can't grow %d bytes", len(src)))
	}

	sh := (*reflect.SliceHeader)(unsafe.Pointer(&src))
	memmove(p, unsafe.Pointer(sh.Data), uintptr(sh.Len))

	return
}

// Equivalent to growing by the length of s, then copying s over the new
// segment. Panics if there is not enough room available.
func (b *OutMessage) AppendString(src string) {
	p := b.GrowNoZero(uintptr(len(src)))
	if p == nil {
		panic(fmt.Sprintf("Can't grow %d bytes", len(src)))
	}

	sh := (*reflect.StringHeader)(unsafe.Pointer(&src))
	memmove(p, unsafe.Pointer(sh.Data), uintptr(sh.Len))

	return
}

// Return the current size of the buffer.
func (b *OutMessage) Len() int {
	return int(b.offset)
}

// Return a reference to the current contents of the buffer.
func (b *OutMessage) Bytes() []byte {
	return b.storage[:int(b.offset)]
}
