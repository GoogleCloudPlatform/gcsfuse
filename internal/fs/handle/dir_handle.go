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

package handle

import (
	"fmt"
	"sort"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"golang.org/x/net/context"
)

// DirHandle is the state required for reading from directories.
type DirHandle struct {
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
}

// NewDirHandle creates a directory handle that obtains listings from the supplied inode.
func NewDirHandle(
	in inode.DirInode,
	implicitDirs bool) (dh *DirHandle) {
	// Set up the basic struct.
	dh = &DirHandle{
		in:           in,
		implicitDirs: implicitDirs,
	}

	// Set up invariant checking.
	dh.Mu = locker.New("DH."+in.Name().GcsObjectName(), dh.checkInvariants)

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Directory entries, sorted by name.
type sortedDirents []fuseutil.Dirent

func (p sortedDirents) Len() int           { return len(p) }
func (p sortedDirents) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p sortedDirents) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (dh *DirHandle) checkInvariants() {
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
func fixConflictingNames(entries []fuseutil.Dirent, localEntries map[string]fuseutil.Dirent) (output []fuseutil.Dirent, err error) {
	// Sanity check.
	if !sort.IsSorted(sortedDirents(entries)) {
		err = fmt.Errorf("expected sorted input")
		return
	}

	// Examine each adjacent pair of names.
	for i := range entries {
		e := &entries[i]

		// Find the previous entry.
		if i == 0 {
			output = append(output, *e)
			continue
		}

		prev := &output[len(output)-1]

		// Does the pair have matching names?
		if e.Name != prev.Name {
			output = append(output, *e)
			continue
		}

		// We expect exactly one to be a directory.
		eIsDir := e.Type == fuseutil.DT_Directory
		prevIsDir := prev.Type == fuseutil.DT_Directory

		if eIsDir == prevIsDir {
			if _, ok := localEntries[e.Name]; ok && !eIsDir {
				// We have found same entry in GCS and local file entries, i.e, the
				// entry is uploaded to GCS but not yet deleted from local entries.
				// Do not return the duplicate entry as part of list response.
				continue
			} else {
				err = fmt.Errorf(
					"weird dirent type pair for name %q: %v, %v",
					e.Name,
					e.Type,
					prev.Type)
				return
			}
		}

		// Repair whichever is not the directory.
		if eIsDir {
			prev.Name += inode.ConflictingFileNameSuffix
		} else {
			e.Name += inode.ConflictingFileNameSuffix
		}

		output = append(output, *e)
	}

	return
}

// Read all entries for the directory, fix up conflicting names, and fill in
// offset fields.
//
// LOCKS_REQUIRED(in)
func readAllEntries(
	ctx context.Context,
	in inode.DirInode,
	localEntries map[string]fuseutil.Dirent) (entries []fuseutil.Dirent, err error) {
	// Read entries from GCS.
	// Read one batch at a time.
	var tok string
	for {
		// Read a batch.
		var batch []fuseutil.Dirent

		batch, tok, err = in.ReadEntries(ctx, tok)
		if err != nil {
			err = fmt.Errorf("ReadEntries: %w", err)
			return
		}

		// Accumulate.
		entries = append(entries, batch...)

		// Are we done?
		if tok == "" {
			break
		}
	}

	// Append local file entries (not synced to GCS).
	for _, localEntry := range localEntries {
		entries = append(entries, localEntry)
	}

	// Ensure that the entries are sorted, for use in fixConflictingNames
	// below.
	sort.Sort(sortedDirents(entries))

	// Fix name conflicts.
	// When a local file is synced to GCS but not removed from the local file map,
	// the entries list will have two duplicate entries.
	// To handle this scenario, we are removing the duplicate entry before
	// returning the response to kernel.
	entries, err = fixConflictingNames(entries, localEntries)
	if err != nil {
		err = fmt.Errorf("fixConflictingNames: %w", err)
		return
	}

	// Fix up offset fields.
	for i := 0; i < len(entries); i++ {
		entries[i].Offset = fuseops.DirOffset(i) + 1
	}

	// Return a bogus inode ID for each entry, but not the root inode ID.
	//
	// NOTE: As far as I can tell this is harmless. Minting and
	// returning a real inode ID is difficult because fuse does not count
	// readdir as an operation that increases the inode ID's lookup count, and
	// we therefore don't get a forget for it later, but we would like to not
	// have to remember every inode ID that we've ever minted for readdir.
	//
	// If it turns out this is not harmless, we'll need to switch to something
	// like inode IDs based on (object name, generation) hashes. But then what
	// about the birthday problem? And more importantly, what about our
	// semantic of not minting a new inode ID when the generation changes due
	// to a local action?
	for i := range entries {
		entries[i].Inode = fuseops.RootInodeID + 1
	}

	return
}

// LOCKS_REQUIRED(dh.Mu)
// LOCKS_EXCLUDED(dh.in)
func (dh *DirHandle) ensureEntries(ctx context.Context, localFileEntries map[string]fuseutil.Dirent) (err error) {
	dh.in.Lock()
	defer dh.in.Unlock()

	// Read entries.
	var entries []fuseutil.Dirent
	entries, err = readAllEntries(ctx, dh.in, localFileEntries)
	if err != nil {
		err = fmt.Errorf("readAllEntries: %w", err)
		return
	}

	// Update state.
	dh.entries = entries
	dh.entriesValid = true

	return
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
	op *fuseops.ReadDirOp,
	localFileEntries map[string]fuseutil.Dirent) (err error) {
	// If the request is for offset zero, we assume that either this is the first
	// call or rewinddir has been called. Reset state.
	if op.Offset == 0 {
		dh.entries = nil
		dh.entriesValid = false
	}

	// Do we need to read entries from GCS?
	if !dh.entriesValid {
		err = dh.ensureEntries(ctx, localFileEntries)
		if err != nil {
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
