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

import "io"

// A sentinel error used when a lease has been revoked. See notes on particular
// methods.
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
	// Reads for an expired lease will return an error of type *RevokedError.
	io.ReadSeeker
	io.ReaderAt

	// Return the size of the underlying file.
	Size() (size int64)

	// Attempt to upgrade the lease to a read/write lease, returning nil if the
	// lease has been revoked. After upgrading, it is as if the lease has been
	// revoked.
	Upgrade() (rwl ReadWriteLease)

	// Cause the lease to be revoked and any associated resources to be cleaned
	// up, if it has not already been revoked.
	Revoke()
}
