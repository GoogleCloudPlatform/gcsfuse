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

import "io"

// Create a ReadLease that never expires, unless voluntarily revoked or
// upgraded.
//
// The supplied function will be used to obtain the read lease contents, the
// first time and whenever the supplied file leaser decides to expire the
// temporary copy thus obtained. It must return the same contents every time,
// and the contents must be of the given size.
//
// This magic is not preserved after the lease is upgraded.
func NewAutoRefreshingReadLease(
	fl FileLeaser,
	size int64,
	f func() (io.ReadCloser, error)) (rl ReadLease) {
	rl = &autoRefreshingReadLease{
		leaser: fl,
		size:   size,
		f:      f,
	}

	return
}

type autoRefreshingReadLease struct {
	leaser FileLeaser
	size   int64
	f      func() (io.ReadCloser, error)
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (rl *autoRefreshingReadLease) Read(p []byte) (n int, err error) {
	panic("TODO")
}

func (rl *autoRefreshingReadLease) Seek(
	offset int64,
	whence int) (off int64, err error) {
	panic("TODO")
}

func (rl *autoRefreshingReadLease) ReadAt(p []byte, off int64) (n int, err error) {
	panic("TODO")
}

func (rl *autoRefreshingReadLease) Size() (size int64) {
	panic("TODO")
}

func (rl *autoRefreshingReadLease) Revoked() (revoked bool) {
	panic("TODO")
}

func (rl *autoRefreshingReadLease) Upgrade() (rwl ReadWriteLease, err error) {
	panic("TODO")
}

func (rl *autoRefreshingReadLease) Revoke() {
	panic("TODO")
}
