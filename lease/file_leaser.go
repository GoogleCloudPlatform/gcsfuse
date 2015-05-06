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

	// A lock that guards the mutable state in this struct, which must not be
	// held for any blocking operation.
	//
	// Lock ordering
	// -------------
	//
	// Define our strict partial order < as follows:
	//
	//  1. For any read/write lease W, W < leaser.
	//  2. For any read lease R, R < leaser.
	//  3. For any read/write lease W and read lease R, W < R.
	//
	// In other words: read/write before read before leaser, and never hold two
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
	// INVARIANT: For each k, v: v.Value.(*readLease) == k
	// INVARIANT: Contains all and only the lements of readLeases
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
		dir:   dir,
		limit: limitBytes,
	}

	fl.mu = syncutil.NewInvariantMutex(fl.checkInvariants)

	return
}

// Create a new anonymous file, and return a read/write lease for it. The
// read/write lease will pin resources until rwl.Downgrade is called. It need
// not be called if the process is exiting.
func (fl *FileLeaser) NewFile() (rwl ReadWriteLease) {
	panic("TODO")
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (fl *FileLeaser) checkInvariants() {
	panic("TODO")
}

// Add the supplied delta to the leaser's view of outstanding read/write lease
// bytes, then revoke read leases until we're under limit or we run out of
// leases to revoke.
//
// Called by readWriteLease while holding its lock.
//
// LOCKS_EXCLUDED(fl.mu)
func (fl *FileLeaser) addReadWriteByteDelta(delta int64) {
	// TODO(jacobsa): When evicting, repeatedly:
	// 1. Find least recently used read lease.
	// 2. Drop leaser lock.
	// 3. Acquire read lease lock.
	// 4. Reacquire leaser lock.
	// 5. If under limit now, drop both locks and return.
	// 6. If lease already evicted, drop its lock and go to #1.
	// 7. Evict lease, drop both locks. If still above limit, start over.
	panic("TODO")
}

// Downgrade the supplied read/write lease, given its current size.
//
// Called by readWriteLease.
//
// LOCKS_REQUIRED(rwl)
// LOCKS_EXCLUDED(fl.mu)
func (fl *FileLeaser) downgrade(
	rwl *readWriteLease,
	size int64) (rl *readLease) {
	panic("TODO")
}
