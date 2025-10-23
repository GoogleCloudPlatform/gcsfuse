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
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"golang.org/x/net/context"
)

// DirEntry is a generic interface for directory entries that expose
// name, type, and offset. It is implemented by fuseutil.Dirent and fuseutil.DirentPlus.
type DirEntry interface {
	EntryName() string
	SetName(name string)
	EntryType() fuseutil.DirentType
	SetOffset(offset fuseops.DirOffset)
}

type dirent fuseutil.Dirent
type direntPlus fuseutil.DirentPlus

// For Dirent
func (d *dirent) EntryName() string                  { return d.Name }
func (d *dirent) SetName(name string)                { d.Name = name }
func (d *dirent) EntryType() fuseutil.DirentType     { return d.Type }
func (d *dirent) SetOffset(offset fuseops.DirOffset) { d.Offset = offset }

// For DirentPlus
func (dp *direntPlus) EntryName() string                  { return dp.Dirent.Name }
func (dp *direntPlus) SetName(name string)                { dp.Dirent.Name = name }
func (dp *direntPlus) EntryType() fuseutil.DirentType     { return dp.Dirent.Type }
func (dp *direntPlus) SetOffset(offset fuseops.DirOffset) { dp.Dirent.Offset = offset }

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

	// All entries in the directory along with their attributes. Populated the first time we need one.
	//
	// INVARIANT: For each i, entriesPlus[i+1].Offset == entriesPlus[i].Offset + 1
	//
	// GUARDED_BY(Mu)
	entriesPlus []fuseutil.DirentPlus

	// Has entries yet been populated?
	//
	// INVARIANT: If !entriesValid, then len(entries) == 0
	//
	// GUARDED_BY(Mu)
	entriesValid bool

	// Has entriesPlus yet been populated?
	//
	// INVARIANT: If !entriesPlusValid, then len(entriesPlus) == 0
	//
	// GUARDED_BY(Mu)
	entriesPlusValid bool
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

	// INVARIANT: For each i, entriesPlus[i+1].Dirent.Offset == entriesPlus[i].Dirent.Offset + 1
	for i := 0; i < len(dh.entriesPlus)-1; i++ {
		if !(dh.entriesPlus[i+1].Dirent.Offset == dh.entriesPlus[i].Dirent.Offset+1) {
			panic(
				fmt.Sprintf(
					"Unexpected offset sequence: %v, %v",
					dh.entriesPlus[i].Dirent.Offset,
					dh.entriesPlus[i+1].Dirent.Offset))
		}
	}

	// INVARIANT: If !entriesPlusValid, then len(entriesPlus) == 0
	if !dh.entriesPlusValid && len(dh.entriesPlus) != 0 {
		panic("Unexpected non-empty entries slice")
	}
}

// compareEntriesByName provides a comparison function for sorting directory entries
// by name.
func compareEntriesByName[T DirEntry](a, b T) int {
	return cmp.Compare(a.EntryName(), b.EntryName())
}

// Resolve name conflicts between file objects and directory objects (e.g. the
// objects "foo/bar" and "foo/bar/") by appending U+000A, which is illegal in
// GCS object names, to conflicting file names.
//
// Input must be sorted by name.
func fixConflictingNames[T DirEntry](entries []T, localEntries map[string]T) (output []T, err error) {
	// Sanity check.
	if !slices.IsSortedFunc(entries, compareEntriesByName) {
		err = fmt.Errorf("expected sorted input")
		return
	}

	// Examine each adjacent pair of names.
	for i := range entries {
		e := entries[i]

		// Find the previous entry.
		if i == 0 {
			output = append(output, e)
			continue
		}

		prev := output[len(output)-1]

		// Does the pair have matching names?
		if e.EntryName() != prev.EntryName() {
			output = append(output, e)
			continue
		}

		// We expect exactly one to be a directory.
		eIsDir := e.EntryType() == fuseutil.DT_Directory
		prevIsDir := prev.EntryType() == fuseutil.DT_Directory

		if eIsDir == prevIsDir {
			if _, ok := localEntries[e.EntryName()]; ok && !eIsDir {
				// We have found same entry in GCS and local file entries, i.e, the
				// entry is uploaded to GCS but not yet deleted from local entries.
				// Do not return the duplicate entry as part of list response.
				continue
			} else {
				err = fmt.Errorf(
					"weird dirent type pair for name %q: %v, %v",
					e.EntryName(),
					e.EntryType(),
					prev.EntryType())
				return
			}
		}

		// Repair whichever is not the directory.
		if eIsDir {
			prev.SetName(prev.EntryName() + inode.ConflictingFileNameSuffix)
		} else {
			e.SetName(e.EntryName() + inode.ConflictingFileNameSuffix)
		}

		output = append(output, e)
	}

	return
}

// sortAndResolveEntries is a generic helper function that takes a list of
// directory entries, sorts them, resolves name conflicts, and sets their
// offsets.
//
// This generic function supports both fuseutil.Dirent and fuseutil.DirentPlus
// by wrapping/ unwrapping them into DirEntry interface-compatible types.
func sortAndResolveEntries[Entry any, WrappedEntry DirEntry](entries []Entry, localEntries map[string]Entry, wrap func(Entry) WrappedEntry, unwrap func(WrappedEntry) Entry) ([]Entry, error) {
	// Wrap and append local file entry (not synced to GCS).
	wrappedLocalEntries := make(map[string]WrappedEntry)
	for name, localEntry := range localEntries {
		wrappedLocalEntries[name] = wrap(localEntry)
		entries = append(entries, localEntry)
	}
	wrappedEntries := make([]WrappedEntry, 0, len(entries))
	for _, entry := range entries {
		wrappedEntries = append(wrappedEntries, wrap(entry))
	}

	// Ensure that the entries are sorted, for use in fixConflictingNames
	// below.
	slices.SortFunc(wrappedEntries, compareEntriesByName)

	// Fix name conflicts.
	// When a local file is synced to GCS but not removed from the local file map,
	// the entries list will have two duplicate entries.
	// To handle this scenario, we are removing the duplicate entry before
	// returning the response to kernel.
	fixedEntries, err := fixConflictingNames(wrappedEntries, wrappedLocalEntries)
	if err != nil {
		err = fmt.Errorf("fixConflictingNames: %w", err)
		return nil, err
	}

	// Fix up offset fields and unwrap.
	finalEntries := make([]Entry, 0, len(fixedEntries))
	for i, fe := range fixedEntries {
		fe.SetOffset(fuseops.DirOffset(i) + 1)
		finalEntries = append(finalEntries, unwrap(fe))
	}

	return finalEntries, nil
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

		batch, _, tok, err = in.ReadEntries(ctx, tok)
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

	// Sort, resolve conflicts, and set offsets.
	entries, err = sortAndResolveEntries(entries, localEntries, func(e fuseutil.Dirent) *dirent { d := dirent(e); return &d }, func(w *dirent) fuseutil.Dirent { return fuseutil.Dirent(*w) })
	if err != nil {
		return nil, err
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

// readAllEntryCores retrieves all directory entry cores for the given inode,
// handling pagination and accumulating the results.
// LOCKS_REQUIRED(in)
func readAllEntryCores(ctx context.Context, in inode.DirInode) (cores map[inode.Name]*inode.Core, err error) {
	// Read entries from GCS.
	// Read one batch at a time.
	var tok string
	cores = make(map[inode.Name]*inode.Core)
	for {
		// Read a batch from GCS
		var batch map[inode.Name]*inode.Core
		batch, _, tok, err = in.ReadEntryCores(ctx, tok)
		if err != nil {
			return
		}
		// Accumulate.
		maps.Copy(cores, batch)

		// Are we done?
		if tok == "" {
			break
		}
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

// FetchEntryCores retrieves the core inode data for all entries within the directory from GCS.
//
// Special case: If the request offset is zero, it assumes the directory is being read from the
// beginning and resets the cached list of entries.
//
// LOCKS_REQUIRED(dh.Mu)
// LOCKS_EXCLUDED(dh.in)
func (dh *DirHandle) FetchEntryCores(ctx context.Context, op *fuseops.ReadDirPlusOp) (cores map[inode.Name]*inode.Core, err error) {
	// If the request is for offset zero, we assume that either this is the first
	// call or rewinddir has been called. Reset state.
	if op.Offset == 0 {
		dh.entriesPlus = nil
		dh.entriesPlusValid = false
	}

	// Do we need to read entries from GCS?
	if !dh.entriesPlusValid {
		dh.in.Lock()
		cores, err = readAllEntryCores(ctx, dh.in)
		if err != nil {
			dh.in.Unlock()
			return
		}
		dh.in.Unlock()
	}

	return
}

// ReadDirPlus populates the FUSE response buffer using a pre-processed list
// of directory entries.
// LOCKS_REQUIRED(dh.Mu)
func (dh *DirHandle) ReadDirPlus(op *fuseops.ReadDirPlusOp, entries []fuseutil.DirentPlus, localEntries map[string]fuseutil.DirentPlus) (err error) {
	// If entriesPlus has not been populated yet, populate it.
	if !dh.entriesPlusValid {
		// Sort, resolve conflicts, and set offsets.
		entries, err = sortAndResolveEntries(entries, localEntries, func(e fuseutil.DirentPlus) *direntPlus { dp := direntPlus(e); return &dp }, func(w *direntPlus) fuseutil.DirentPlus { return fuseutil.DirentPlus(*w) })
		if err != nil {
			return
		}
		// Update state.
		dh.entriesPlus = entries
		dh.entriesPlusValid = true
	}

	// Is the offset past the end of what we have buffered? If so, this must be
	// an invalid seekdir according to posix.
	index := int(op.Offset)
	if index > len(dh.entriesPlus) {
		err = fuse.EINVAL
		return
	}

	//We copy out entries until we run out of entries or space.
	for i := index; i < len(dh.entriesPlus); i++ {
		n := fuseutil.WriteDirentPlus(op.Dst[op.BytesRead:], dh.entriesPlus[i])
		if n == 0 {
			break
		}

		op.BytesRead += n
	}

	return
}
