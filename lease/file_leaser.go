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
	"log"
	"os"

	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/gcloud/syncutil"
)

// A type that manages read and read/write leases for anonymous temporary files.
//
// Safe for concurrent access.
type FileLeaser interface {
	// Create a new anonymous file, and return a read/write lease for it. The
	// read/write lease will pin resources until rwl.Downgrade is called. It need
	// not be called if the process is exiting.
	NewFile() (rwl ReadWriteLease, err error)

	// Revoke all read leases that have been issued. For testing use only.
	RevokeReadLeases()
}

// Create a new file leaser that uses the supplied directory for temporary
// files (before unlinking them) and attempts to keep usage in number of files
// and bytes below the given limits. If dir is empty, the system default will be
// used.
//
// Usage may exceed the given limits if there are read/write leases whose total
// size exceeds the limits, since such leases cannot be revoked.
func NewFileLeaser(
	dir string,
	limitNumFiles int,
	limitBytes int64) (fl FileLeaser) {
	typed := &fileLeaser{
		dir:             dir,
		limitNumFiles:   limitNumFiles,
		limitBytes:      limitBytes,
		readLeasesIndex: make(map[*readLease]*list.Element),
	}

	typed.mu = syncutil.NewInvariantMutex(typed.checkInvariants)

	fl = typed
	return
}

type fileLeaser struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	dir           string
	limitNumFiles int
	limitBytes    int64

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

	// The number of outstanding read/write leases.
	//
	// INVARIANT: readWriteCount >= 0
	readWriteCount int

	// The current estimated total size of outstanding read/write leases. This is
	// only an estimate because we can't synchronize its update with a call to
	// the wrapped file to e.g. write or truncate.
	readWriteBytes int64

	// All outstanding read leases, ordered by recency of use.
	//
	// INVARIANT: Each element is of type *readLease
	// INVARIANT: No element has been revoked.
	// INVARIANT: 0 <= readLeases.Len() <= max(0, limitNumFiles - readWriteCount)
	readLeases list.List

	// The sum of all outstanding read lease sizes.
	//
	// INVARIANT: Equal to the sum over readLeases sizes.
	// INVARIANT: 0 <= readOutstanding
	// INVARIANT: readOutstanding <= max(0, limitBytes - readWriteBytes)
	readOutstanding int64

	// Index of read leases by pointer.
	//
	// INVARIANT: Is an index of exactly the elements of readLeases
	readLeasesIndex map[*readLease]*list.Element
}

func (fl *fileLeaser) NewFile() (rwl ReadWriteLease, err error) {
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

// LOCKS_EXCLUDED(fl.mu)
func (fl *fileLeaser) RevokeReadLeases() {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	fl.evict(0, 0)
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func maxInt(a int, b int) int {
	if a > b {
		return a
	}

	return b
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}

	return b
}

// LOCKS_REQUIRED(fl.mu)
func (fl *fileLeaser) checkInvariants() {
	// INVARIANT: readWriteCount >= 0
	if fl.readWriteCount < 0 {
		panic(fmt.Sprintf("Unexpected read/write count: %d", fl.readWriteCount))
	}

	// INVARIANT: Each element is of type *readLease
	// INVARIANT: No element has been revoked.
	for e := fl.readLeases.Front(); e != nil; e = e.Next() {
		rl := e.Value.(*readLease)
		func() {
			rl.Mu.Lock()
			defer rl.Mu.Unlock()

			if rl.revoked() {
				panic("Found revoked read lease")
			}
		}()
	}

	// INVARIANT: 0 <= readLeases.Len() <= max(0, limitNumFiles - readWriteCount)
	if !(0 <= fl.readLeases.Len() &&
		fl.readLeases.Len() <= maxInt(0, fl.limitNumFiles-fl.readWriteCount)) {
		panic(fmt.Sprintf(
			"Out of range read lease count: %d, limitNumFiles: %d, readWriteCount: %d",
			fl.readLeases.Len(),
			fl.limitNumFiles,
			fl.readWriteCount))
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

	// INVARIANT: readOutstanding <= max(0, limitBytes - readWriteBytes)
	if !(fl.readOutstanding <= maxInt64(0, fl.limitBytes-fl.readWriteBytes)) {
		panic(fmt.Sprintf(
			"Unexpected readOutstanding: %v. limitBytes: %v, readWriteBytes: %v",
			fl.readOutstanding,
			fl.limitBytes,
			fl.readWriteBytes))
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
// bytes, then revoke read leases until we're under limitBytes or we run out of
// leases to revoke.
//
// Called by readWriteLease while holding its lock.
//
// LOCKS_EXCLUDED(fl.mu)
func (fl *fileLeaser) addReadWriteByteDelta(delta int64) {
	fl.readWriteCount++
	fl.readWriteBytes += delta
	fl.evict(fl.limitNumFiles, fl.limitBytes)
}

// LOCKS_REQUIRED(fl.mu)
func (fl *fileLeaser) overLimit(limitNumFiles int, limitBytes int64) bool {
	return fl.readLeases.Len()+fl.readWriteCount > limitNumFiles ||
		fl.readOutstanding+fl.readWriteBytes > limitBytes
}

// Revoke read leases until we're within the given limitBytes or we run out of
// things to revoke.
//
// LOCKS_REQUIRED(fl.mu)
func (fl *fileLeaser) evict(limitNumFiles int, limitBytes int64) {
	for fl.overLimit(limitNumFiles, limitBytes) {
		// Do we have anything to revoke?
		lru := fl.readLeases.Back()
		if lru == nil {
			return
		}

		// Revoke it.
		rl := lru.Value.(*readLease)
		func() {
			rl.Mu.Lock()
			defer rl.Mu.Unlock()

			fl.revoke(rl)
		}()
	}
}

// Note that a read/write lease of the given size is destroying itself, and
// turn it into a read lease of the supplied size wrapped around the given
// file.
//
// Called by readWriteLease with its lock held.
//
// LOCKS_EXCLUDED(fl.mu)
func (fl *fileLeaser) downgrade(
	size int64,
	file *os.File) (rl ReadLease) {
	// Create the read lease.
	rlTyped := newReadLease(size, fl, file)
	rl = rlTyped

	// Update the leaser's state, noting the new read lease and that the
	// read/write lease has gone away.
	fl.mu.Lock()
	defer fl.mu.Unlock()

	fl.readWriteCount--
	fl.readWriteBytes -= size
	fl.readOutstanding += size

	e := fl.readLeases.PushFront(rl)
	fl.readLeasesIndex[rlTyped] = e

	// Ensure that we're not now over capacity.
	fl.evict(fl.limitNumFiles, fl.limitBytes)

	return
}

// Upgrade the supplied read lease.
//
// Called by readLease with no lock held.
//
// LOCKS_EXCLUDED(fl.mu, rl.Mu)
func (fl *fileLeaser) upgrade(rl *readLease) (rwl ReadWriteLease, err error) {
	// Grab each lock in turn.
	fl.mu.Lock()
	defer fl.mu.Unlock()

	rl.Mu.Lock()
	defer rl.Mu.Unlock()

	// Has the lease already been revoked?
	if rl.revoked() {
		err = &RevokedError{}
		return
	}

	size := rl.Size()

	// Update leaser state.
	fl.readWriteBytes += size
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

// Promote the given read lease to most recently used, if we still know it.
// Because our lock order forbids us from acquiring the leaser lock while
// holding a read lease lock, this of course races with other promotions.
//
// Called by readLease without holding a lock.
//
// LOCKS_EXCLUDED(fl.mu)
func (fl *fileLeaser) promoteToMostRecent(rl *readLease) {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	e := fl.readLeasesIndex[rl]
	if e != nil {
		fl.readLeases.MoveToFront(e)
	}
}

// Forcibly revoke the supplied read lease.
//
// REQUIRES: !rl.revoked()
//
// LOCKS_REQUIRED(fl.mu)
// LOCKS_REQUIRED(rl.Mu)
func (fl *fileLeaser) revoke(rl *readLease) {
	if rl.revoked() {
		panic("Already revoked")
	}

	size := rl.Size()

	// Update leaser state.
	fl.readOutstanding -= size

	e := fl.readLeasesIndex[rl]
	delete(fl.readLeasesIndex, rl)
	fl.readLeases.Remove(e)

	// Kill the lease and close its file.
	file := rl.release()
	if err := file.Close(); err != nil {
		log.Println("Error closing file for revoked lease:", err)
	}
}

// Called by the read lease when the user wants to manually revoke it.
//
// LOCKS_EXCLUDED(fl.mu)
// LOCKS_EXCLUDED(rl.Mu)
func (fl *fileLeaser) revokeVoluntarily(rl *readLease) {
	// Grab each lock in turn.
	fl.mu.Lock()
	defer fl.mu.Unlock()

	rl.Mu.Lock()
	defer rl.Mu.Unlock()

	// Has the lease already been revoked?
	if rl.revoked() {
		return
	}

	// Revoke it.
	fl.revoke(rl)
}
