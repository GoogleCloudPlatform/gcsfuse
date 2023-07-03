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
	"sort"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"golang.org/x/net/context"
)

// State required for reading from directories.
type dirHandle struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	in           inode.DirInode
	implicitDirs bool

	/////////////////////////
	// Mutable state
	/////////////////////////

	Mu locker.Locker

	// All entries in the directory. Populated the first time we need one.
	//
	// INVARIANT: For each i, entries[i+1].Offset == entries[i].Offset + 1
	//
	// GUARDED_BY(Mu)
	entries []fuseutil.Dirent

	// Has entries yet been populated?
	//
	// INVARIANT: If !entriesValid, then len(entries) == 0
	//
	// GUARDED_BY(Mu)
	entriesValid bool

	// Condition variable is for signalling whether a fresh set of entries has been fetched.
	cond *sync.Cond

	// Error during the fetching goroutine must be communicated using this to the main go routine
	// serving kernel requests.
	err error

	// Using this as a identification flag to indicate all entries present in the directory
	// have been fetched already so that the main goroutine does not wait indefinitely for more entries.
	fetchOver bool

	// To stop the fetching of entries in case of interrupts to main goroutine like ctrl +c
	// from the user.
	cancel context.CancelFunc
}

// Create a directory handle that obtains listings from the supplied inode.
func newDirHandle(
	in inode.DirInode,
	implicitDirs bool) (dh *dirHandle) {
	// Set up the basic struct.
	dh = &dirHandle{
		in:           in,
		implicitDirs: implicitDirs,
	}

	// Set up invariant checking.
	dh.Mu = locker.New("DH."+in.Name().GcsObjectName(), dh.checkInvariants)
	// Creating a condition variable to indicate events for locking and unlocking dh.Mu mutex.
	dh.cond = sync.NewCond(dh.Mu)
	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Dirents, sorted by name.
type sortedDirents []fuseutil.Dirent

func (p sortedDirents) Len() int           { return len(p) }
func (p sortedDirents) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p sortedDirents) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (dh *dirHandle) checkInvariants() {
	// INVARIANT: For each i, entries[i+1].Offset == entries[i].Offset + 1
	for i := 0; i < len(dh.entries)-1; i++ {
		if !(dh.entries[i+1].Offset == dh.entries[i].Offset+1) {
			panic(
				fmt.Sprintf(
					"Unexpected offset sequence: %v, %v",
					dh.entries[i].Offset,
					dh.entries[i+1].Offset))
		}
	}

	// INVARIANT: If !entriesValid, then len(entries) == 0
	if !dh.entriesValid && len(dh.entries) != 0 {
		panic("Unexpected non-empty entries slice")
	}
}

// Resolve name conflicts between file objects and directory objects (e.g. the
// objects "foo/bar" and "foo/bar/") by appending U+000A, which is illegal in
// GCS object names, to conflicting file names.
//
// Input must be sorted by name.
func fixConflictingNames(entries []fuseutil.Dirent) (err error) {
	// Sanity check.
	if !sort.IsSorted(sortedDirents(entries)) {
		err = fmt.Errorf("Expected sorted input")
		return
	}

	// Examine each adjacent pair of names.
	for i, _ := range entries {
		e := &entries[i]

		// Find the previous entry.
		if i == 0 {
			continue
		}

		prev := &entries[i-1]

		// Does the pair have matching names?
		if e.Name != prev.Name {
			continue
		}

		// We expect exactly one to be a directory.
		eIsDir := e.Type == fuseutil.DT_Directory
		prevIsDir := prev.Type == fuseutil.DT_Directory

		if eIsDir == prevIsDir {
			err = fmt.Errorf(
				"Weird dirent type pair for name %q: %v, %v",
				e.Name,
				e.Type,
				prev.Type)

			return
		}

		// Repair whichever is not the directory.
		if eIsDir {
			prev.Name += inode.ConflictingFileNameSuffix
		} else {
			e.Name += inode.ConflictingFileNameSuffix
		}
	}

	return
}

func (dh *dirHandle) setErrorAndBroadcast(err error) {
	err = fmt.Errorf("ReadEntries: %w", err)
	dh.Mu.Lock()
	dh.err = err
	dh.Mu.Unlock()
	// Signal the suspended go routine that an error has occurred.
	dh.cond.Broadcast()
}

// Fetch Dirent entries from GCSfuse.Will be used as a goroutine which is run asynchronously
// to fetch data in the background while kernel requests are also served simultaneously.
func (dh *dirHandle) FetchEntriesAsync(
	rootInodeId int) {
	// New context is needed as the parent goroutine exiting earlier than the child will cause the
	// context to be cancelled prematurely.
	var ctx context.Context
	ctx, dh.cancel = context.WithCancel(context.Background())
	var err error
	var entryForSorting fuseutil.Dirent
	// ContinuationToken is also empty in case of firstCall and after all entries have been fetched.
	// Keeping continuation token local so as to lessen the time for which the mutex is held.
	// Keep fetching entries in batches of MaxResultsForListObjectsCall.
	var ContinuationToken string
	for {
		var entries []fuseutil.Dirent
		dh.in.Lock()
		entries, ContinuationToken, err = dh.in.ReadEntries(ctx, ContinuationToken)
		dh.in.Unlock()
		if err != nil {
			dh.setErrorAndBroadcast(err)
			break
		}
		// Use the last entry from the last fetch for fixing naming conflicts.
		dh.Mu.Lock()
		if entryForSorting != (fuseutil.Dirent{}) {
			entries = append(entries, entryForSorting)
		}
		dh.Mu.Unlock()
		sort.Sort(sortedDirents(entries))
		err = fixConflictingNames(entries)

		if err != nil {
			dh.setErrorAndBroadcast(err)
			break
		}

		// Save the last entry from current fetch to use it for
		// fixing naming conflicts for next fetch.
		if ContinuationToken != "" {
			entryForSorting = entries[len(entries)-1]
			entries = entries[:len(entries)-1]
		}
		dh.Mu.Lock()
		// Update InodeID and Offset for the entries.
		for i := range entries {
			entries[i].Inode = fuseops.InodeID(rootInodeId + 1)
			entries[i].Offset = fuseops.DirOffset(uint64(len(dh.entries) + i + 1))
		}

		dh.entries = append(dh.entries, entries...)
		dh.entriesValid = true
		dh.Mu.Unlock()
		// Signal the suspended go routine that the next set of entries has
		// been fetched.
		dh.cond.Broadcast()
		if ContinuationToken == "" {
			break
		}
	}
	dh.Mu.Lock()
	dh.fetchOver = true
	dh.Mu.Unlock()

}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// ReadDir handles a request to read from the directory, without responding.
//
// Special case: we assume that a zero offset indicates that rewinddir has been
// called (since fuse gives us no way to intercept and know for sure), and
// start the listing process over again.
//
// LOCKS_REQUIRED(dh.Mu)
// LOCKS_EXCLUDED(du.in)
func (dh *dirHandle) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) (err error) {
	// If the request is for offset zero, we assume that either this is the first
	// call or rewinddir has been called. Reset state.
	dh.Mu.Lock()
	defer dh.Mu.Unlock()
	if op.Offset == 0 {
		dh.entries = nil
		dh.entriesValid = false
		dh.err = nil
		dh.fetchOver = false
	}
	if !dh.entriesValid {
		go dh.FetchEntriesAsync(fuseops.RootInodeID)
	}

	// If the fetched entries is not sufficient to serve the request, then wait only
	// if there are more entries to be fetched (fetchOver is false).
	if len(dh.entries) <= int(op.Offset) && !dh.fetchOver {
		// Internally, cond.Wait() unlocks the mutex and locks it again when woken up
		// by other go routines through a signal or a broadcast.
		dh.cond.Wait()
		if dh.err != nil {
			err = dh.err
			return
		}
	}

	// Is the offset past the end of what we have buffered? If so, this must be
	// an invalid seekdir according to posix.
	index := int(op.Offset)
	if index > len(dh.entries) {
		err = fuse.EINVAL
		return
	}

	// We copy out entries until we run out of entries or space.
	for i := index; i < len(dh.entries); i++ {
		n := fuseutil.WriteDirent(op.Dst[op.BytesRead:], dh.entries[i])
		if n == 0 {
			break
		}
		op.BytesRead += n
	}
	return
}
