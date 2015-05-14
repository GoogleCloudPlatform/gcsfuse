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

import "golang.org/x/net/context"

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
	rp = &multiReadProxy{}
	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type multiReadProxy struct {
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
}
