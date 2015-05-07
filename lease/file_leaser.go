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
	"container/list"
	"fmt"
	"os"

	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/gcloud/syncutil"
)

// A type that manages read and read/write leases for anonymous temporary files.
//
// Safe for concurrent access. Must be created with NewFileLeaser.
type FileLeaser struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	dir   string
	limit int64

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A lock that guards the mutable state in this struct. Usually this is used
	// only for light weight operations, but while evicting it may require
	// waiting on a goroutine that is holding a read lease lock while reading
	// from a file.
	//
	// Lock ordering
	// -------------
	//
	// Define < to be the minimum strict partial order satisfying:
	//
	//  1. For any read/write lease W, W < leaser.
	//  2. For any read lease R, leaser < R.
	//
	// In other words: read/write before leaser before read, and never hold two
	// locks from the same category together.
	mu syncutil.InvariantMutex

	// The current estimated total size of outstanding read/write leases. This is
	// only an estimate because we can't synchronize its update with a call to
	// the wrapped file to e.g. write or truncate.
	readWriteOutstanding int64

	// All outstanding read leases, ordered by recency of use.
	//
	// INVARIANT: Each element is of type *readLease
	// INVARIANT: No element has been revoked.
	readLeases list.List

	// The sum of all outstanding read lease sizes.
	//
	// INVARIANT: Equal to the sum over readLeases sizes.
	// INVARIANT: 0 <= readOutstanding
	// INVARIANT: readOutstanding <= max(0, limit - readWriteOutstanding)
	readOutstanding int64

	// Index of read leases by pointer.
	//
	// INVARIANT: Is an index of exactly the elements of readLeases
	readLeasesIndex map[*readLease]*list.Element
}

// Create a new file leaser that uses the supplied directory for temporary
// files (before unlinking them) and attempts to keep usage in bytes below the
// given limit. If dir is empty, the system default will be used.
//
// Usage may exceed the given limit if there are read/write leases whose total
// size exceeds the limit, since such leases cannot be revoked.
func NewFileLeaser(
	dir string,
	limitBytes int64) (fl *FileLeaser) {
	fl = &FileLeaser{
		dir:             dir,
		limit:           limitBytes,
		readLeasesIndex: make(map[*readLease]*list.Element),
	}

	fl.mu = syncutil.NewInvariantMutex(fl.checkInvariants)

	return
}

// Create a new anonymous file, and return a read/write lease for it. The
// read/write lease will pin resources until rwl.Downgrade is called. It need
// not be called if the process is exiting.
func (fl *FileLeaser) NewFile() (rwl ReadWriteLease, err error) {
	// Create an anonymous file.
	f, err := fsutil.AnonymousFile(fl.dir)
	if err != nil {
		err = fmt.Errorf("AnonymousFile: %v", err)
		return
	}

	// Wrap a lease around it.
	rwl = newReadWriteLease(fl, 0, f)

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}

	return b
}

// LOCKS_REQUIRED(fl.mu)
func (fl *FileLeaser) checkInvariants() {
	// INVARIANT: Each element is of type *readLease
	// INVARIANT: No element has been revoked.
	for e := fl.readLeases.Front(); e != nil; e = e.Next() {
		rl := e.Value.(*readLease)
		rl.Mu.Lock()

		if rl.revoked() {
			panic("Found revoked read lease")
		}

		rl.Mu.Unlock()
	}

	// INVARIANT: Equal to the sum over readLeases sizes.
	var sum int64
	for e := fl.readLeases.Front(); e != nil; e = e.Next() {
		rl := e.Value.(*readLease)
		sum += rl.Size()
	}

	if fl.readOutstanding != sum {
		panic(fmt.Sprintf(
			"readOutstanding mismatch: %v vs. %v",
			fl.readOutstanding,
			sum))
	}

	// INVARIANT: 0 <= readOutstanding
	if !(0 <= fl.readOutstanding) {
		panic(fmt.Sprintf("Unexpected readOutstanding: %v", fl.readOutstanding))
	}

	// INVARIANT: readOutstanding <= max(0, limit - readWriteOutstanding)
	if !(fl.readOutstanding <= maxInt64(0, fl.limit-fl.readWriteOutstanding)) {
		panic(fmt.Sprintf(
			"Unexpected readOutstanding: %v. limit: %v, readWriteOutstanding: %v",
			fl.readOutstanding,
			fl.limit,
			fl.readWriteOutstanding))
	}

	// INVARIANT: Is an index of exactly the elements of readLeases
	if len(fl.readLeasesIndex) != fl.readLeases.Len() {
		panic(fmt.Sprintf(
			"readLeasesIndex length mismatch: %v vs. %v",
			len(fl.readLeasesIndex),
			fl.readLeases.Len()))
	}

	for e := fl.readLeases.Front(); e != nil; e = e.Next() {
		if fl.readLeasesIndex[e.Value.(*readLease)] != e {
			panic("Mismatch in readLeasesIndex")
		}
	}
}

// Add the supplied delta to the leaser's view of outstanding read/write lease
// bytes, then revoke read leases until we're under limit or we run out of
// leases to revoke.
//
// Called by readWriteLease while holding its lock.
//
// LOCKS_EXCLUDED(fl.mu)
func (fl *FileLeaser) addReadWriteByteDelta(delta int64) {
	fl.readWriteOutstanding += delta
	fl.evict()
}

// LOCKS_REQUIRED(fl.mu)
func (fl *FileLeaser) overLimit() bool {
	return fl.readOutstanding+fl.readWriteOutstanding > fl.limit
}

// Revoke read leases until we're under limit or we run out of things to revoke.
//
// LOCKS_REQUIRED(fl.mu)
func (fl *FileLeaser) evict() {
	for fl.overLimit() {
		// Do we have anything to revoke?
		lru := fl.readLeases.Back()
		if lru == nil {
			return
		}

		_ = lru.Value.(*readLease)
		panic("TODO")
	}
}

// Downgrade the supplied read/write lease, given its current size and the
// underlying file.
//
// Called by readWriteLease with its lock held.
//
// LOCKS_EXCLUDED(fl.mu)
func (fl *FileLeaser) downgrade(
	rwl *readWriteLease,
	size int64,
	file *os.File) (rl ReadLease) {
	// Create the read lease.
	rlTyped := newReadLease(size, fl, file)
	rl = rlTyped

	// Update the leaser's state, noting the new read lease and that the
	// read/write lease has gone away.
	fl.mu.Lock()
	defer fl.mu.Unlock()

	fl.readWriteOutstanding -= size
	fl.readOutstanding += size

	e := fl.readLeases.PushFront(rl)
	fl.readLeasesIndex[rlTyped] = e

	// Ensure that we're not now over capacity.
	fl.evict()

	return
}

// Upgrade the supplied read lease.
//
// Called by readLease with no lock held.
//
// LOCKS_EXCLUDED(fl.mu, rl.Mu)
func (fl *FileLeaser) upgrade(rl *readLease) (rwl ReadWriteLease) {
	// Grab each lock in turn.
	fl.mu.Lock()
	defer fl.mu.Unlock()

	rl.Mu.Lock()
	defer rl.Mu.Unlock()

	// Has the lease already been revoked?
	if rl.revoked() {
		return
	}

	size := rl.Size()

	// Update leaser state.
	fl.readWriteOutstanding += size
	fl.readOutstanding -= size

	e := fl.readLeasesIndex[rl]
	delete(fl.readLeasesIndex, rl)
	fl.readLeases.Remove(e)

	// Extract the interesting information from the read lease, leaving it an
	// empty husk.
	file := rl.release()

	// Create the read/write lease, telling it that we already know its initial
	// size.
	rwl = newReadWriteLease(fl, size, file)

	return
}

// Forcibly revoke the supplied read lease.
//
// LOCKS_REQUIRED(rl, fl.mu)
func (fl *FileLeaser) revoke(rl *readLease) {
	panic("TODO")
}

// Called by the read lease when the user wants to manually revoke it.
//
// LOCKS_EXCLUDED(fl.mu)
// LOCKS_EXCLUDED(rl.Mu)
func (fl *FileLeaser) revokeVoluntarily(rl *readLease) {
	// Grab each lock in turn.
	fl.mu.Lock()
	defer fl.mu.Unlock()

	rl.Mu.Lock()
	defer rl.Mu.Unlock()

	// Has the lease already been revoked?
	if rl.revoked() {
		return
	}

	panic("TODO")
}
