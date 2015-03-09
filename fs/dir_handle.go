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

	// The continuation token to supply in the next call to in.ReadEntries. If
	// empty, we have finished reading the directory and must be rewound to
	// obtain more entries.
	//
	// GUARDED_BY(Mu)
	tok string
}

// Create a directory handle that obtains listings from the supplied inode.
func newDirHandle(in *inode.DirInode) (dh *dirHandle) {
	// Set up the basic struct.
	dh = &dirHandle{
		in: in,
	}

	// Set up invariant checking.
	dh.Mu = syncutil.NewInvariantMutex(dh.checkInvariants)

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (dh *dirHandle) checkInvariants() {
	// Check that the offsets increment by one each time.
	for i := 0; i < len(dh.entries)-1; i++ {
		if dh.entries[i].Offset+1 != dh.entries[i+1].Offset {
			panic(
				fmt.Sprintf(
					"Unexpected offset sequence: %v, %v",
					dh.entries[i].Offset,
					dh.entries[i+1].Offset))
		}
	}

	// Check the first offset.
	if len(dh.entries) > 0 && dh.entries[0].Offset != dh.entriesOffset+1 {
		panic(fmt.Sprintf("Unexpected entriesOffset value."))
	}
}

// Read some entries from the directory inode. Return newTok == "" (possibly
// with a non-empty list of entries) when the end of the directory has been
// hit.
//
// SHARED_LOCKS_REQUIRED(in.Mu)
func readEntries(
	ctx context.Context,
	in *inode.DirInode,
	tok string,
	firstEntryOffset fuse.DirOffset) (
	entries []fuseutil.Dirent, newTok string, err error) {
	// Fix up the offset of any entries returned.
	defer func() {
		for i := 0; i < len(entries); i++ {
			entries[i].Offset = firstEntryOffset + 1 + fuse.DirOffset(i)
		}
	}()

	// Loop until we get a non-empty result, we finish, or an error occurs.
	for {
		entries, newTok, err = in.ReadEntries(ctx, tok)

		// Propagate errors.
		if err != nil {
			return
		}

		// If some entries were returned, we are done.
		if len(entries) > 0 {
			return
		}

		// If we're at the end of the directory, we're done.
		if newTok == "" {
			return
		}

		// Otherwise, go around and ask for more entries.
		tok = newTok
	}
}

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

	// Do we need to read more entries, and can we?
	if index == len(dh.entries) && dh.tok != "" {
		var newEntries []fuseutil.Dirent
		var newTok string

		// Read some entries.
		newEntries, newTok, err = readEntries(
			ctx,
			dh.in,
			dh.tok,
			dh.entriesOffset+fuse.DirOffset(len(dh.entries)))

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
