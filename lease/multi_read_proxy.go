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

package lease

import (
	"fmt"
	"io"
	"sort"

	"golang.org/x/net/context"
)

// Create a read proxy consisting of the contents defined by the supplied
// refreshers concatenated. See NewReadProxy for more.
//
// If rl is non-nil, it will be used as the first temporary copy of the
// contents, and must match the concatenation of the content returned by the
// refreshers.
func NewMultiReadProxy(
	fl FileLeaser,
	refreshers []Refresher,
	rl ReadLease) (rp ReadProxy) {
	// Create one wrapped read proxy per refresher.
	var wrappedProxies []readProxyAndOffset
	var size int64
	for _, r := range refreshers {
		wrapped := NewReadProxy(fl, r, nil)
		wrappedProxies = append(wrappedProxies, readProxyAndOffset{size, wrapped})
		size += wrapped.Size()
	}

	rp = &multiReadProxy{
		size:  size,
		rps:   wrappedProxies,
		lease: rl,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type multiReadProxy struct {
	// The size of the proxied content.
	size int64

	// The wrapped read proxies, indexed by their logical starting offset.
	//
	// INVARIANT: If len(rps) != 0, rps[0].off == 0
	// INVARIANT: For each x, x.rp.Size() >= 0
	// INVARIANT: For each i>0, rps[i].off == rps[i-i].off + rps[i-i].rp.Size()
	// INVARIANT: size is the sum over the wrapped proxy sizes.
	rps []readProxyAndOffset

	// A read lease for the entire contents. May be nil.
	//
	// INVARIANT: If lease != nil, size == lease.Size()
	lease ReadLease

	destroyed bool
}

func (mrp *multiReadProxy) Size() (size int64) {
	size = mrp.size
	return
}

func (mrp *multiReadProxy) ReadAt(
	ctx context.Context,
	p []byte,
	off int64) (n int, err error) {
	// Special case: we don't support negative offsets, silly user.
	if off < 0 {
		err = fmt.Errorf("Invalid offset: %v", off)
		return
	}

	// Special case: offsets at or beyond the end of our content can never yield
	// any content, and the io.ReaderAt spec allows us to return EOF. Knock them
	// out here so we know off is in range when we start below.
	if off >= mrp.Size() {
		err = io.EOF
		return
	}

	// The read proxy that contains off is the *last* read proxy whose start
	// offset is less than or equal to off. Find the first that is greater and
	// move back one.
	//
	// Because we handled the special cases above, this must be in range.
	wrappedIndex := mrp.upperBound(off) - 1

	if wrappedIndex < 0 || wrappedIndex >= len(mrp.rps) {
		panic(fmt.Sprintf("Unexpected index: %v", wrappedIndex))
	}

	// Keep going until we've got nothing left to do.
	for len(p) > 0 {
		// Have we run out of wrapped read proxies?
		if wrappedIndex == len(mrp.rps) {
			err = io.EOF
			return
		}

		// Read from the wrapped proxy, accumulating into our total before checking
		// for a read error.
		wrappedN, wrappedErr := mrp.readFromOne(ctx, index, p, off)
		n += wrappedN
		if wrappedErr != nil {
			err = wrappedErr
			return
		}

		// readFromOne guarantees to either fill our buffer or exhaust the wrapped
		// proxy. So advance the buffer, the offset, and the wrapped proxy index
		// and go again.
		p = p[wrappedN:]
		off += int64(wrappedN)
		wrappedIndex++
	}

	return
}

func (mrp *multiReadProxy) Upgrade(
	ctx context.Context) (rwl ReadWriteLease, err error) {
	panic("TODO")
}

func (mrp *multiReadProxy) Destroy() {
	// Destroy all of the wrapped proxies.
	for _, entry := range mrp.rps {
		entry.rp.Destroy()
	}

	// Destroy the lease for the entire contents, if any.
	if mrp.lease != nil {
		mrp.lease.Revoke()
	}

	// Crash early if called again.
	mrp.rps = nil
	mrp.lease = nil
}

func (mrp *multiReadProxy) CheckInvariants() {
	// INVARIANT: If len(rps) != 0, rps[0].off == 0
	if len(mrp.rps) != 0 && mrp.rps[0].off != 0 {
		panic(fmt.Sprintf("Unexpected starting point: %v", mrp.rps[0].off))
	}

	// INVARIANT: For each x, x.rp.Size() >= 0
	for _, x := range mrp.rps {
		if x.rp.Size() < 0 {
			panic(fmt.Sprintf("Negative size: %v", x.rp.Size()))
		}
	}

	// INVARIANT: For each i>0, rps[i].off == rps[i-i].off + rps[i-i].rp.Size()
	for i := range mrp.rps {
		if i > 0 && !(mrp.rps[i].off == mrp.rps[i-1].off+mrp.rps[i-1].rp.Size()) {
			panic("Offsets are not indexed correctly.")
		}
	}

	// INVARIANT: size is the sum over the wrapped proxy sizes.
	var sum int64
	for _, wrapped := range mrp.rps {
		sum += wrapped.rp.Size()
	}

	if sum != mrp.size {
		panic(fmt.Sprintf("Size mismatch: %v vs. %v", sum, mrp.size))
	}

	// INVARIANT: If lease != nil, size == lease.Size()
	if mrp.lease != nil && mrp.size != mrp.lease.Size() {
		panic(fmt.Sprintf("Size mismatch: %v vs. %v", mrp.size, mrp.lease.Size()))
	}
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type readProxyAndOffset struct {
	off int64
	rp  ReadProxy
}

// Return the index within mrp.rps of the first read proxy whose logical offset
// is greater than off. If there is none, return len(mrp.rps).
func (mrp *multiReadProxy) upperBound(off int64) (index int) {
	pred := func(i int) bool {
		return mrp.rps[i].off > off
	}

	return sort.Search(len(mrp.rps), pred)
}

// Serve a read from the wrapped proxy at the given index within our array of
// wrapped proxies. The offset is relative to the start of the multiReadProxy,
// not the wrapped proxy.
//
// Guarantees:
//
//  *  If err == nil, either n == len(p) or n == wrapped.Size().
//  *  Never returns err == io.EOF.
//
// REQUIRES: index < len(mrp.rps)
// REQUIRES: mrp.rps[i].off <= off < mrp.rps[i].off + wrapped.Size()
func (mrp *multiReadProxy) readFromOne(
	ctx context.Context,
	index int,
	p []byte,
	off int64) (n int, err error)
