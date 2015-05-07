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

package inode

import (
	"sync"

	"github.com/jacobsa/fuse/fuseops"
	"golang.org/x/net/context"
)

type Inode interface {
	// All methods below require the lock to be held unless otherwise documented.
	sync.Locker

	// Return the ID assigned to the inode.
	//
	// Does not require the lock to be held.
	ID() fuseops.InodeID

	// Return the name of the GCS object backing the inode. This may be "foo/bar"
	// for a file, or "foo/bar/" for a directory.
	//
	// Does not require the lock to be held.
	Name() string

	// Increment the lookup count for the inode. For use in fuse operations where
	// the kernel expects us to remember the inode.
	IncrementLookupCount()

	// Return up to date attributes for this inode.
	Attributes(ctx context.Context) (fuseops.InodeAttributes, error)

	// Decrement the lookup count for the inode by the given amount.
	//
	// If this method returns true, the lookup count has hit zero and the
	// Destroy() method should be called to release any local resources, perhaps
	// after releasing locks that should not be held while blocking.
	DecrementLookupCount(n uint64) (destroy bool)

	// Clean up any local resources used by the inode, putting it into an
	// indeterminate state. This method may block. Errors are for logging
	// purposes only.
	Destroy() (err error)
}
