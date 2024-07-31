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
	"reflect"
	"unsafe"

	"github.com/jacobsa/fuse/internal/fusekernel"
)

// OutMessageHeaderSize is the size of the leading header in every
// properly-constructed OutMessage. Reset brings the message back to this size.
const OutMessageHeaderSize = int(unsafe.Sizeof(fusekernel.OutHeader{}))

// OutMessage provides a mechanism for constructing a single contiguous fuse
// message from multiple segments, where the first segment is always a
// fusekernel.OutHeader message.
//
// Must be initialized with Reset.
type OutMessage struct {
	header fusekernel.OutHeader
	Sglist [][]byte
}

// Reset resets m so that it's ready to be used again. Afterward, the contents
// are solely a zeroed fusekernel.OutHeader struct.
func (m *OutMessage) Reset() {
	m.header = fusekernel.OutHeader{}
	m.Sglist = nil
}

// OutHeader returns a pointer to the header at the start of the message.
func (m *OutMessage) OutHeader() *fusekernel.OutHeader {
	return &m.header
}

// Grow adds a new buffer of <n> bytes to the message, returning a pointer to
// the start of the new segment, which is guaranteed to be zeroed.
func (m *OutMessage) Grow(n int) unsafe.Pointer {
	b := make([]byte, n)
	m.Append(b)
	p := unsafe.Pointer(&b[0])
	return p
}

// ShrinkTo shrinks m to the given size. It panics if the size is greater than
// Len() or less than OutMessageHeaderSize.
func (m *OutMessage) ShrinkTo(n int) {
	if n < OutMessageHeaderSize || n > m.Len() {
		panic(fmt.Sprintf(
			"ShrinkTo(%d) out of range (current Len: %d)",
			n,
			m.Len()))
	}
	if n == OutMessageHeaderSize {
		m.Sglist = nil
	} else {
		i := 1
		n -= OutMessageHeaderSize
		for len(m.Sglist) > i && n >= len(m.Sglist[i]) {
			n -= len(m.Sglist[i])
			i++
		}
		if n > 0 {
			m.Sglist[i] = m.Sglist[i][0:n]
			i++
		}
		m.Sglist = m.Sglist[0:i]
	}
}

// Append is equivalent to growing by len(src), then copying src over the new
// segment. Int panics if there is not enough room available.
func (m *OutMessage) Append(src ...[]byte) {
	if m.Sglist == nil {
		// First element of Sglist is pre-filled with a pointer to the header
		// to allow sending it with a single writev() call without copying the
		// slice again
		m.Sglist = append(m.Sglist, m.OutHeaderBytes())
	}
	m.Sglist = append(m.Sglist, src...)
	return
}

// AppendString is like Append, but accepts string input.
func (m *OutMessage) AppendString(src string) {
	m.Append([]byte(src))
	return
}

// Len returns the current size of the message, including the leading header.
func (m *OutMessage) Len() int {
	if m.Sglist == nil {
		return OutMessageHeaderSize
	}
	// First element of Sglist is the header, so we don't need to count it here
	r := 0
	for _, b := range m.Sglist {
		r += len(b)
	}
	return r
}

// OutHeaderBytes returns a byte slice containing the current header.
func (m *OutMessage) OutHeaderBytes() []byte {
	l := OutMessageHeaderSize
	sh := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&m.header)),
		Len:  l,
		Cap:  l,
	}
	return *(*[]byte)(unsafe.Pointer(&sh))
}
