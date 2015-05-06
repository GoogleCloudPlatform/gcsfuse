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
	"io"
	"os"
	"sync"
)

// A sentinel error used when a lease has been revoked.
type RevokedError struct {
}

func (re *RevokedError) Error() string {
	return "Lease revoked"
}

// A read-only wrapper around a file that may be revoked, when e.g. there is
// temporary disk space pressure. A read lease may also be upgraded to a write
// lease, if it is still valid.
//
// All methods are safe for concurrent access.
type ReadLease interface {
	io.ReadSeeker
	io.ReaderAt

	// Return the size of the underlying file.
	Size() (size int64, err error)

	// Attempt to upgrade the lease to a read/write lease, returning nil if the
	// lease has been revoked. After upgrading, it is as if the lease has been
	// revoked.
	Upgrade() (rwl ReadWriteLease)

	// Cause the lease to be revoked and any associated resources to be cleaned
	// up, if it has not already been revoked.
	Revoke()
}

type readLease struct {
	mu sync.Mutex

	/////////////////////////
	// Constant data
	/////////////////////////

	// The size, to be used only when file != nil. This is redundant with file
	// because file.Stat() may fail, and we want FileLeaser to be able to depend
	// on Size() never failing for non-revoked leases.
	size int64

	/////////////////////////
	// Dependencies
	/////////////////////////

	// The leaser that issued this lease.
	leaser *FileLeaser

	// The underlying file, set to nil once revoked.
	//
	// GUARDED_BY(mu)
	file *os.File
}

var _ ReadLease = &readLease{}

func newReadLease(
	size int64,
	leaser *FileLeaser,
	file *os.File) (rl *readLease) {
	rl = &readLease{
		size:   size,
		leaser: leaser,
		file:   file,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// LOCKS_EXCLUDED(rl.mu)
func (rl *readLease) Read(p []byte) (n int, err error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Have we been revoked?
	if rl.file == nil {
		err = &RevokedError{}
		return
	}

	n, err = rl.file.Read(p)
	return
}

// LOCKS_EXCLUDED(rl.mu)
func (rl *readLease) Seek(
	offset int64,
	whence int) (off int64, err error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Have we been revoked?
	if rl.file == nil {
		err = &RevokedError{}
		return
	}

	off, err = rl.file.Seek(offset, whence)
	return
}

// LOCKS_EXCLUDED(rl.mu)
func (rl *readLease) ReadAt(p []byte, off int64) (n int, err error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Have we been revoked?
	if rl.file == nil {
		err = &RevokedError{}
		return
	}

	n, err = rl.file.ReadAt(p, off)
	return
}

// Guaranteed not to fail if the lease is still valid.
//
// LOCKS_EXCLUDED(rl.mu)
func (rl *readLease) Size() (size int64, err error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Have we been revoked?
	if rl.file == nil {
		err = &RevokedError{}
		return
	}

	size = rl.size
	return
}

// LOCKS_EXCLUDED(rl.mu)
func (rl *readLease) Upgrade() (rwl ReadWriteLease) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Have we been revoked?
	if rl.file == nil {
		return
	}

	// Call the leaser.
	rwl = rl.leaser.upgrade(rl, rl.size, rl.file)

	// Note that we've been revoked.
	rl.file = nil

	return
}

// LOCKS_EXCLUDED(rl.mu)
func (rl *readLease) Revoke() {
	panic("TODO")
}
