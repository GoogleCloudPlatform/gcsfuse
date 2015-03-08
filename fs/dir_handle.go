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

package fs

import (
	"fmt"
	"io"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/gcloud/syncutil"
	"github.com/jacobsa/gcsfuse/fs/inode"
	"golang.org/x/net/context"
)

// State required for reading from directories.
type dirHandle struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	in *inode.DirInode

	/////////////////////////
	// Mutable state
	/////////////////////////

	Mu syncutil.InvariantMutex

	// Entries that we have buffered from a previous call to in.ReadEntries.
	//
	// INVARIANT: For each i in range, entries[i+1].Offset == entries[i].Offset + 1
	//
	// GUARDED_BY(Mu)
	entries []fuseutil.Dirent

	// The logical offset at which entries[0] lies.
	//
	// INVARIANT: If len(entries) > 0, entriesOffset + 1 == entries[0].Offset
	//
	// GUARDED_BY(Mu)
	entriesOffset fuse.DirOffset

	// The continuation token to supply in the next call to in.ReadEntries.
	//
	// GUARDED_BY(Mu)
	tok string
}

// Create a directory handle that obtains listings from the supplied inode.
func newDirHandle(in *inode.DirInode) *dirHandle

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (dh *dirHandle) checkInvariants()

// Read some entries from the directory inode. Return an error iff zero entries
// are returned. When the end of the directory is hit, this error will be
// io.EOF.
//
// SHARED_LOCKS_REQUIRED(in.Mu)
func readEntries(
	in *inode.DirInode,
	firstEntryOffset fuse.DirOffset) (
	entries []fuseutil.Dirent, tok string, err error)

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// Handle a request to read from the directory.
//
// Because the GCS API for listing objects has no notion of a stable offset
// like the posix telldir/seekdir API does, there is no way for us to
// efficiently support seeking backwards. Therefore we return EINVAL when an
// offset for an entry that we no longer have buffered is encountered.
//
// Special case: we assume that a zero offset indicates that rewinddir has been
// called (since fuse gives us no way to intercept and know for sure), and
// start the listing process over again.
//
// EXCLUSIVE_LOCKS_REQUIRED(dh.Mu)
// LOCKS_EXCLUDED(dh.in.Mu)
func (dh *dirHandle) ReadDir(
	ctx context.Context,
	req *fuse.ReadDirRequest) (resp *fuse.ReadDirResponse, err error) {
	resp = &fuse.ReadDirResponse{}

	dh.in.Mu.RLock()
	defer dh.in.Mu.RUnlock()

	// If the request is for offset zero, we assume that either this is the first
	// call or rewinddir has been called. Reset state.
	if req.Offset == 0 {
		dh.entries = nil
		dh.entriesOffset = 0
		dh.tok = ""
	}

	// Is the offset from before what we have buffered? If not, this represents a
	// seekdir we cannot support, as discussed in the method comments above.
	if req.Offset < dh.entriesOffset {
		err = fuse.EINVAL
		return
	}

	// Is the offset past the end of what we have buffered? If so, this must be
	// an invalid seekdir according to posix.
	index := int(req.Offset - dh.entriesOffset)
	if index > len(dh.entries) {
		err = fuse.EINVAL
		return
	}

	// Do we need to read more entries?
	if index == len(dh.entries) {
		var newEntries []fuseutil.Dirent
		var newTok string

		newEntries, newTok, err = readEntries(
			dh.in,
			dh.entriesOffset+fuse.DirOffset(len(dh.entries)))

		// For EOF we simply return an empty listing.
		if err == io.EOF {
			err = nil
			return
		}

		// Propagate other errors.
		if err != nil {
			err = fmt.Errorf("readEntries: %v", err)
			return
		}

		// Update state.
		dh.entriesOffset += fuse.DirOffset(len(dh.entries))
		dh.entries = newEntries
		dh.tok = newTok

		// Now we want to read from the front of dh.entries.
		index = 0
	}

	// Now we copy out entries until we run out of entries or space.
	for i := index; i < len(dh.entries); i++ {
		resp.Data = fuseutil.AppendDirent(resp.Data, dh.entries[i])
		if len(resp.Data) > req.Size {
			resp.Data = resp.Data[:req.Size]
			break
		}
	}

	return
}
