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

import "github.com/jacobsa/gcloud/syncutil"

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
	// LOCK ORDERING
	// -------------
	//
	// Define our strict partial order < as follows:
	//
	//  1. For any two leases L1 and L2 with L1.Id < L2.Id, L1.mu < L2.mu.
	//  2. For any lease L, L.mu < leaser.mu
	//
	mu syncutil.InvariantMutex

	// The unique ID to hand out for the next lease issued.
	nextID uint64

	// The current estimated total size of outstanding read/write leases. This is
	// only an estimate because each time a read/write lease is updated, the
	// updater drops the lock, acquires the FS lock, then adds the delta here.
	// This saves us from needed to serialize I/O through distinct files.
	readWriteOutstanding int64
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
func (fl *FileLeaser) New() (rwl ReadWriteLease) {
	panic("TODO")
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (fl *FileLeaser) checkInvariants() {
}
