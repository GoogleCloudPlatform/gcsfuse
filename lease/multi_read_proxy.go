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
	var off int64
	for _, r := range refreshers {
		wrapped := NewReadProxy(fl, r, nil)
		wrappedProxies = append(wrappedProxies, readProxyAndOffset{off, wrapped})
		off += wrapped.Size()
	}

	rp = &multiReadProxy{
		rps:   wrappedProxies,
		lease: rl,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type multiReadProxy struct {
	// The wrapped read proxies, indexed by their logical starting offset.
	//
	// INVARIANT: For each i>0, rps[i].off == rps[i-i].off + rps[i-i].rp.Size()
	rps []readProxyAndOffset

	// A read lease for the entire contents. May be nil.
	//
	// INVARIANT: If lease != nil, lease.Size() is the sum over wrapped proxy
	// sizes.
	lease ReadLease
}

func (mrp *multiReadProxy) Size() (size int64) {
	panic("TODO")
}

func (mrp *multiReadProxy) ReadAt(
	ctx context.Context,
	p []byte,
	off int64) (n int, err error) {
	panic("TODO")
}

func (mrp *multiReadProxy) Upgrade(
	ctx context.Context) (rwl ReadWriteLease, err error) {
	panic("TODO")
}

func (mrp *multiReadProxy) Destroy() {
	panic("TODO")
}

func (mrp *multiReadProxy) CheckInvariants() {
	// INVARIANT: For each i>0, rps[i].off == rps[i-i].off + rps[i-i].rp.Size()
	for i := range mrp.rps {
		if i > 0 && !(mrp.rps[i].off == mrp.rps[i-1].off+mrp.rps[i-1].rp.Size()) {
			panic("Offsets are not indexed correctly.")
		}
	}

	// INVARIANT: If lease != nil, lease.Size() is the sum over wrapped proxy
	// sizes.
	if mrp.lease != nil {
		var sum int64
		for _, wrapped := range mrp.rps {
			sum += wrapped.rp.Size()
		}

		if sum != mrp.lease.Size() {
			panic(fmt.Sprintf("Size mismatch: %v vs. %v", sum, mrp.lease.Size()))
		}
	}
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type readProxyAndOffset struct {
	off int64
	rp  ReadProxy
}
