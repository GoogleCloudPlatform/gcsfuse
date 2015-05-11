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
	"log"
	"os"

	"github.com/jacobsa/gcloud/syncutil"
)

// A read-write wrapper around a file. Unlike a read lease, this cannot be
// revoked.
//
// All methods are safe for concurrent access.
type ReadWriteLease interface {
	// Methods with semantics matching *os.File.
	io.ReadWriteSeeker
	io.ReaderAt
	io.WriterAt
	Truncate(size int64) (err error)

	// Return the current size of the underlying file.
	Size() (size int64, err error)

	// Downgrade to a read lease, releasing any resources pinned by this lease to
	// the pool that may be revoked, as with any read lease. After downgrading,
	// this lease must not be used again.
	Downgrade() (rl ReadLease)
}

type readWriteLease struct {
	mu syncutil.InvariantMutex

	/////////////////////////
	// Dependencies
	/////////////////////////

	// The leaser that issued this lease.
	leaser *fileLeaser

	// The underlying file, set to nil once downgraded.
	//
	// GUARDED_BY(mu)
	file *os.File

	/////////////////////////
	// Mutable state
	/////////////////////////

	// The cumulative number of bytes we have reported to the leaser using
	// fileLeaser.addReadWriteByteDelta. When the size changes, we report the
	// difference between the new size and this value.
	//
	// GUARDED_BY(mu)
	reportedSize int64

	// Our current view of the file's size, or a negative value if we dirtied the
	// file but then failed to find its size.
	//
	// INVARIANT: If fileSize >= 0, fileSize agrees with file.Stat()
	// INVARIANT: fileSize < 0 || fileSize == reportedSize
	//
	// GUARDED_BY(mu)
	fileSize int64
}

var _ ReadWriteLease = &readWriteLease{}

// size is the size that the leaser has already recorded for us. It must match
// the file's size.
func newReadWriteLease(
	leaser *fileLeaser,
	size int64,
	file *os.File) (rwl *readWriteLease) {
	rwl = &readWriteLease{
		leaser:       leaser,
		file:         file,
		reportedSize: size,
		fileSize:     size,
	}

	rwl.mu = syncutil.NewInvariantMutex(rwl.checkInvariants)

	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// LOCKS_EXCLUDED(rwl.mu)
func (rwl *readWriteLease) Read(p []byte) (n int, err error) {
	rwl.mu.Lock()
	defer rwl.mu.Unlock()

	n, err = rwl.file.Read(p)
	return
}

// LOCKS_EXCLUDED(rwl.mu)
func (rwl *readWriteLease) Write(p []byte) (n int, err error) {
	rwl.mu.Lock()
	defer rwl.mu.Unlock()

	// Ensure that we reconcile our size when we're done.
	defer rwl.reconcileSize()

	// Call through.
	n, err = rwl.file.Write(p)

	return
}

// LOCKS_EXCLUDED(rwl.mu)
func (rwl *readWriteLease) Seek(
	offset int64,
	whence int) (off int64, err error) {
	rwl.mu.Lock()
	defer rwl.mu.Unlock()

	off, err = rwl.file.Seek(offset, whence)
	return
}

// LOCKS_EXCLUDED(rwl.mu)
func (rwl *readWriteLease) ReadAt(p []byte, off int64) (n int, err error) {
	rwl.mu.Lock()
	defer rwl.mu.Unlock()

	n, err = rwl.file.ReadAt(p, off)
	return
}

// LOCKS_EXCLUDED(rwl.mu)
func (rwl *readWriteLease) WriteAt(p []byte, off int64) (n int, err error) {
	rwl.mu.Lock()
	defer rwl.mu.Unlock()

	// Ensure that we reconcile our size when we're done.
	defer rwl.reconcileSize()

	// Call through.
	n, err = rwl.file.WriteAt(p, off)

	return
}

// LOCKS_EXCLUDED(rwl.mu)
func (rwl *readWriteLease) Truncate(size int64) (err error) {
	rwl.mu.Lock()
	defer rwl.mu.Unlock()

	// Ensure that we reconcile our size when we're done.
	defer rwl.reconcileSize()

	// Call through.
	err = rwl.file.Truncate(size)

	return
}

// LOCKS_EXCLUDED(rwl.mu)
func (rwl *readWriteLease) Size() (size int64, err error) {
	rwl.mu.Lock()
	defer rwl.mu.Unlock()

	size, err = rwl.sizeLocked()
	return
}

// LOCKS_EXCLUDED(rwl.mu)
func (rwl *readWriteLease) Downgrade() (rl ReadLease) {
	rwl.mu.Lock()
	defer rwl.mu.Unlock()

	// Ensure that we will crash if used again.
	defer func() {
		rwl.file = nil
	}()

	// Special case: if we don't know the file's current size, we can't reliably
	// create a read lease wrapping the file, since we might be lying about its
	// size.
	//
	// In this case, return a lease whose ostensible  size matches our state last
	// time we succeeded to modify the file, but whose contents cannot be read.
	// Throw away our file, and report to the file leaser that we've done so.
	if rwl.fileSize < 0 {
		rl = &alwaysRevokedReadLease{size: rwl.reportedSize}
		rwl.file.Close()
		rwl.leaser.addReadWriteByteDelta(-rwl.reportedSize)
		return
	}

	// Otherwise, just call through to the leaser.
	rl = rwl.leaser.downgrade(rwl.fileSize, rwl.file)

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// LOCKS_REQUIRED(rwl.mu)
func (rwl *readWriteLease) checkInvariants() {
	// Have we been dowgraded?
	if rwl.file == nil {
		return
	}

	// INVARIANT: If fileSize >= 0, fileSize agrees with file.Stat()
	if rwl.fileSize >= 0 {
		fi, err := rwl.file.Stat()
		if err != nil {
			panic(fmt.Sprintf("Failed to stat file: %v", err))
		}

		if rwl.fileSize != fi.Size() {
			panic(fmt.Sprintf("Size mismatch: %v vs. %v", rwl.fileSize, fi.Size()))
		}
	}

	// INVARIANT: fileSize < 0 || fileSize == reportedSize
	if !(rwl.fileSize < 0 || rwl.fileSize == rwl.reportedSize) {
		panic(fmt.Sprintf("Size mismatch: %v vs. %v", rwl.fileSize, rwl.reportedSize))
	}
}

// LOCKS_REQUIRED(rwl.mu)
func (rwl *readWriteLease) sizeLocked() (size int64, err error) {
	// Stat the file to get its size.
	fi, err := rwl.file.Stat()
	if err != nil {
		err = fmt.Errorf("Stat: %v", err)
		return
	}

	size = fi.Size()
	return
}

// Notify the leaser if our size has changed. Log errors when we fail to find
// our size.
//
// LOCKS_REQUIRED(rwl.mu)
// LOCKS_EXCLUDED(rwl.leaser.mu)
func (rwl *readWriteLease) reconcileSize() {
	var err error

	// If we fail to find the size, we must note that this happened.
	defer func() {
		if err != nil {
			rwl.fileSize = -1
		}
	}()

	// Find our size.
	size, err := rwl.sizeLocked()
	if err != nil {
		log.Println("Error getting size for reconciliation:", err)
		return
	}

	// Let the leaser know about any change.
	delta := size - rwl.reportedSize
	if delta != 0 {
		rwl.leaser.addReadWriteByteDelta(delta)
		rwl.reportedSize = size
	}

	// Update our view of the file's size.
	rwl.fileSize = size
}

////////////////////////////////////////////////////////////////////////
// alwaysRevokedReadLease
////////////////////////////////////////////////////////////////////////

type alwaysRevokedReadLease struct {
	size int64
}

func (rl *alwaysRevokedReadLease) Read(p []byte) (n int, err error) {
	err = &RevokedError{}
	return
}

func (rl *alwaysRevokedReadLease) Seek(
	offset int64,
	whence int) (off int64, err error) {
	err = &RevokedError{}
	return
}

func (rl *alwaysRevokedReadLease) ReadAt(
	p []byte, off int64) (n int, err error) {
	err = &RevokedError{}
	return
}

func (rl *alwaysRevokedReadLease) Size() (size int64) {
	size = rl.size
	return
}

func (rl *alwaysRevokedReadLease) Revoked() (revoked bool) {
	revoked = true
	return
}

func (rl *alwaysRevokedReadLease) Upgrade() (rwl ReadWriteLease, err error) {
	err = &RevokedError{}
	return
}

func (rl *alwaysRevokedReadLease) Revoke() {
}
