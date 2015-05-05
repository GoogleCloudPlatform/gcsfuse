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

	// A lock that guards the mutable state in this struct and in all of the
	// leases it manages.
	//
	// NOTE(jacobsa): This implies that there can be no I/O parallelism when
	// accessing multiple files concurrently. That stinks, but it's not clear
	// what the alternative is if we want to keep our limit up to date.
	mu syncutil.InvariantMutex
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
