// Copyright 2015 Google LLC
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
	"github.com/vipnydav/gcsfuse/v3/internal/gcsx"
	"golang.org/x/net/context"
)

type Inode interface {
	// All methods below require the lock to be held unless otherwise documented.
	sync.Locker

	// Return the ID assigned to the inode.
	//
	// Does not require the lock to be held.
	ID() fuseops.InodeID

	// Return the name of the GCS object backing the inode.
	//
	// Does not require the lock to be held.
	Name() Name

	// Increment the lookup count for the inode. For use in fuse operations where
	// the kernel expects us to remember the inode.
	IncrementLookupCount()

	// Return up to date attributes for this inode.
	// The `clobberedCheck` parameter controls whether this function performs a
	// remote check to see if the backing GCS object has been modified by another
	// process.
	Attributes(ctx context.Context, clobberedCheck bool) (fuseops.InodeAttributes, error)

	// Decrement the lookup count for the inode by the given amount.
	//
	// If this method returns true, the lookup count has hit zero and the
	// Destroy() method should be called to release any local resources, perhaps
	// after releasing locks that should not be held while blocking.
	DecrementLookupCount(n uint64) (destroy bool)

	// Clean up any local resources used by the inode, putting it into an
	// indeterminate state where no method should be called except Unlock.
	//
	// This method may block. Errors are for logging purposes only.
	Destroy() (err error)

	// Unlink operation marks the inode as unlinked/deleted.
	Unlink()
}

// An inode owned by a gcs bucket.
type BucketOwnedInode interface {
	Inode

	// Return the gcs.Bucket which the dir or file belongs to.
	Bucket() *gcsx.SyncerBucket
}

// An inode that is backed by a particular generation of a GCS object.
type GenerationBackedInode interface {
	Inode

	// Requires the inode lock.
	SourceGeneration() Generation
}

// A particular generation of a GCS object, consisting of both a GCS object
// generation number and meta-generation number. Lexicographically ordered on
// the two.
//
// Cf. https://cloud.google.com/storage/docs/generations-preconditions
type Generation struct {
	Object   int64
	Metadata int64
	Size     uint64
}

// Compare returns -1, 0, or 1 according to whether src is less than, equal to,
// or greater than existing.
// Note: Ordering matters here, latest represents the object fetched from GCS
// and current represents inode cached object's generation.
func (latest Generation) Compare(current Generation) int {
	// Compare first on object generation number.
	switch {
	case latest.Object < current.Object:
		return -1

	case latest.Object > current.Object:
		return 1
	}

	// Break ties on meta-generation.
	switch {
	case latest.Metadata < current.Metadata:
		return -1

	case latest.Metadata > current.Metadata:
		return 1
	}

	// Break ties on object size.
	// Because objects in zonal buckets can be appended without altering their
	// generation or metageneration, the following case applies exclusively to
	// zonal buckets.
	if latest.Size > current.Size {
		return 1
	}
	// We ignore latest.Size < current.Size case as little staleness is expected
	// on the GCS object's size.

	return 0
}
