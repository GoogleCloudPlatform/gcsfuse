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
	"runtime"
	"sort"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"golang.org/x/net/context"
)

// Record stores metadata of dirent entries fetched from GCSfuse
type Record struct {
	sync.Mutex
	length          int
	err             error
	entryForSorting fuseutil.Dirent
	cond            *sync.Cond
}

func NewRecord() *Record {
	r := Record{}
	r.cond = sync.NewCond(&r)
	return &r
}

var ContinuationToken string
var rec = NewRecord()

// State required for reading from directories.
type DirHandle struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	In           inode.DirInode
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
	Entries []fuseutil.Dirent

	// Has entries yet been populated?
	//
	// INVARIANT: If !entriesValid, then len(entries) == 0
	//
	// GUARDED_BY(Mu)
	EntriesValid bool
}

// Create a directory handle that obtains listings from the supplied inode.
func NewDirHandle(
	in inode.DirInode,
	implicitDirs bool) (dh *DirHandle) {
	// Set up the basic struct.
	dh = &DirHandle{
		In:           in,
		implicitDirs: implicitDirs,
	}

	// Set up invariant checking.
	dh.Mu = locker.New("DH."+in.Name().GcsObjectName(), dh.checkInvariants)

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

func (dh *DirHandle) checkInvariants() {
	// INVARIANT: For each i, entries[i+1].Offset == entries[i].Offset + 1
	for i := 0; i < len(dh.Entries)-1; i++ {
		if !(dh.Entries[i+1].Offset == dh.Entries[i].Offset+1) {
			panic(
				fmt.Sprintf(
					"Unexpected offset sequence: %v, %v",
					dh.Entries[i].Offset,
					dh.Entries[i+1].Offset))
		}
	}

	// INVARIANT: If !entriesValid, then len(entries) == 0
	if !dh.EntriesValid && len(dh.Entries) != 0 {
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

// Fetch Dirent entries from GCSfuse.Will be used as a goroutine which is run asynchronously
// to fetch data in the background while kernel requests are also served simultaneously.
func (dh *DirHandle) FetchEntriesAsync(
	rootInodeId int,
	firstCall bool) {
	ctx, cancel := context.WithCancel(context.Background())
	var err error
	//ContinuationToken is also empty in case of firstCall and after all entries have been fetched.
	//Keep fetching entries in batches of MaxResultsForListObjectsCall
	for ContinuationToken != "" || firstCall {
		var entries []fuseutil.Dirent
		dh.In.Lock()
		entries, ContinuationToken, err = dh.In.ReadEntries(ctx, ContinuationToken)
		dh.In.Unlock()
		if err != nil {
			err = fmt.Errorf("ReadEntries: %w", err)
			rec.Lock()
			rec.err = err
			rec.Unlock()
			//Signal the suspended go routine that an error has occurred.
			rec.cond.Broadcast()
			//cancel the context to release the associated resources
			cancel()
			//cancelling the context does not kill the go routine.Killing
			//to prevent go routine leak.
			runtime.Goexit()
		}
		//Use the last entry from the last fetch for fixing naming conflicts
		if !firstCall {
			rec.Lock()
			entries = append(entries, rec.entryForSorting)
			rec.Unlock()
		}
		sort.Sort(sortedDirents(entries))
		err = fixConflictingNames(entries)

		if err != nil {
			err = fmt.Errorf("fixConflictingNames: %w", err)
			rec.Lock()
			rec.err = err
			rec.Unlock()
			//Signal the suspended go routine that an error has occurred.
			rec.cond.Broadcast()
			cancel()
			runtime.Goexit()
		}

		//Save the last entry from current fetch to use it for
		//fixing naming conflicts for next fetch
		if ContinuationToken != "" {
			rec.Lock()
			rec.entryForSorting = entries[len(entries)-1]
			rec.Unlock()
			entries = entries[:len(entries)-1]
		}
		rec.Lock()
		//Update InodeID and Offset for the entries
		for i := range entries {
			entries[i].Inode = fuseops.InodeID(rootInodeId + 1)
			entries[i].Offset = fuseops.DirOffset(uint64(rec.length + i + 1))
		}
		rec.length += len(entries)
		rec.Unlock()

		dh.Mu.Lock()
		dh.Entries = append(dh.Entries, entries...)
		dh.EntriesValid = true
		dh.Mu.Unlock()
		//Signal the suspended go routine that the next set of entries has
		//been fetched.
		rec.cond.Broadcast()
		firstCall = false
	}
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
func (dh *DirHandle) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) (err error) {
	// If the request is for offset zero, we assume that either this is the first
	// call or rewinddir has been called. Reset state.

	if op.Offset == 0 {
		dh.Entries = nil
		dh.EntriesValid = false
		rec.Lock()
		rec.length = 0
		rec.err = nil
		rec.Unlock()
		//Offset zero implies first GCS call so start the process of fetching entries
		go dh.FetchEntriesAsync(fuseops.RootInodeID, true)

	}
	rec.Lock()
	//if there are not enough entries fetched till now to serve the current request,
	//suspend this goroutine until sufficient entries .Resume after waking up.
	if rec.length <= int(op.Offset) && (op.Offset == 0 || ContinuationToken != "") {
		rec.cond.Wait()
		//Return if error faced during latest fetch
		if rec.err != nil {
			err = rec.err
			rec.Unlock()
			return
		}
	}

	// Is the offset past the end of what we have buffered? If so, this must be
	// an invalid seekdir according to posix.
	index := int(op.Offset)
	if index > rec.length {
		err = fuse.EINVAL
		rec.Unlock()
		return
	}
	rec.Unlock()
	// We copy out entries until we run out of entries or space.
	dh.Mu.Lock()
	for i := index; i < len(dh.Entries); i++ {
		n := fuseutil.WriteDirent(op.Dst[op.BytesRead:], dh.Entries[i])
		if n == 0 {
			break
		}
		op.BytesRead += n
	}
	dh.Mu.Unlock()
	return
}
