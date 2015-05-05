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
	"errors"
	"os"
	"sync"
)

// A sentinel error used when a lease has been revoked. See notes on particular
// methods.
type RevokedError struct {
}

func (re *RevokedError) Error() string {
	return "Lease revoked"
}

// A read-only wrapper around a file that may be revoked, when e.g. there is
// temporary disk space pressure.
//
// All methods are safe for concurrent access. Must be created with
// NewReadLease.
type ReadLease struct {
	// Functions injected by the user.
	revoked func()
	upgrade func(*os.File) *WriteLease

	mu sync.Mutex

	// The wrapped file. If the lease has been revoked, this will be nil.
	//
	// GUARDED_BY(mu)
	f *os.File
}

// Create a read lease wrapping the supplied file. The lease "owns" this file
// until it is revoked or upgraded, if ever.
//
// If the lease is revoked, the file will be closed and the supplied function
// will be notified (at most once).
//
// If the lease is upgraded, the supplied function will be used to create an
// appropriate write lease and then this lease will forget the file. The
// function must not return nil.
func NewReadLease(
	f *os.File,
	revoked func(),
	upgrade func(f *os.File) *WriteLease) (rl *ReadLease) {
	rl = &ReadLease{
		f:       f,
		revoked: revoked,
		upgrade: upgrade,
	}

	return
}

// Attempt to read within the wrapped file, returning an error of type
// *RevokedError if the lease has been revoked.
func (rl *ReadLease) ReadAt(p []byte, off int64) (n int, err error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Do we still have a file?
	if rl.f == nil {
		err = &RevokedError{}
		return
	}

	// Read via the file.
	n, err = rl.f.ReadAt(p, off)

	return
}

// Attempt to revoke the lease, freeing any resources associated with it. It is
// an error to revoke more than once.
func (rl *ReadLease) Revoke() (err error) {
	err = errors.New("TODO")
	return
}

// Attempt to upgrade the lease to a write lease, returning an error of type
// *RevokedError if the lease has been revoked. After upgrading, it is as if
// the lease has been revoked.
func (rl *ReadLease) Upgrade() (wl *WriteLease, err error) {
	err = errors.New("TODO")
	return
}
