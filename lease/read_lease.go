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

	// Return the size of the underlying file, or what the size used to be if the
	// lease has been revoked.
	Size() (size int64)

	// Attempt to upgrade the lease to a read/write lease, returning nil if the
	// lease has been revoked. After upgrading, it is as if the lease has been
	// revoked.
	Upgrade() (rwl ReadWriteLease)

	// Cause the lease to be revoked and any associated resources to be cleaned
	// up, if it has not already been revoked.
	Revoke()
}

type readLease struct {
	// Used internally and by FileLeaser eviction logic.
	Mu sync.Mutex

	/////////////////////////
	// Constant data
	/////////////////////////

	size int64

	/////////////////////////
	// Dependencies
	/////////////////////////

	// The leaser that issued this lease.
	leaser *FileLeaser

	// The underlying file, set to nil once revoked.
	//
	// Writing requires holding both Mu and leaser.mu. Therefore reading is
	// allowed while holding either.
	//
	// GUARDED_BY([see above])
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

// LOCKS_EXCLUDED(rl.Mu)
func (rl *readLease) Read(p []byte) (n int, err error) {
	rl.Mu.Lock()
	defer rl.Mu.Unlock()

	// Have we been revoked?
	if rl.revoked() {
		err = &RevokedError{}
		return
	}

	n, err = rl.file.Read(p)
	return
}

// LOCKS_EXCLUDED(rl.Mu)
func (rl *readLease) Seek(
	offset int64,
	whence int) (off int64, err error) {
	rl.Mu.Lock()
	defer rl.Mu.Unlock()

	// Have we been revoked?
	if rl.revoked() {
		err = &RevokedError{}
		return
	}

	off, err = rl.file.Seek(offset, whence)
	return
}

// LOCKS_EXCLUDED(rl.Mu)
func (rl *readLease) ReadAt(p []byte, off int64) (n int, err error) {
	rl.Mu.Lock()
	defer rl.Mu.Unlock()

	// Have we been revoked?
	if rl.revoked() {
		err = &RevokedError{}
		return
	}

	n, err = rl.file.ReadAt(p, off)
	return
}

// No lock necessary.
func (rl *readLease) Size() (size int64) {
	size = rl.size
	return
}

// LOCKS_EXCLUDED(rl.Mu)
func (rl *readLease) Upgrade() (rwl ReadWriteLease) {
	rl.Mu.Lock()
	defer rl.Mu.Unlock()

	// Have we been revoked?
	if rl.revoked() {
		return
	}

	// Call the leaser.
	rwl = rl.leaser.upgrade(rl, rl.size, rl.file)

	// Note that we've been revoked.
	rl.file = nil

	return
}

// LOCKS_EXCLUDED(rl.Mu)
func (rl *readLease) Revoke() {
	rl.Mu.Lock()
	defer rl.Mu.Unlock()

	// Have we already been revoked?
	if rl.revoked() {
		return
	}

	rl.leaser.revokeVoluntarily(rl)
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Has the lease been revoked?
//
// LOCKS_REQUIRED(rl.Mu || rl.leaser.mu)
func (rl *readLease) revoked() bool {
	return rl.file == nil
}

// Close the file and note that the lease has been revoked. Called by the file
// leaser.
//
// REQUIRES: Not yet revoked.
//
// LOCKS_REQUIRED(rl.Mu)
// LOCKS_REQUIRED(rl.leaser.mu)
func (rl *readLease) destroy() {
	panic("TODO")
}
