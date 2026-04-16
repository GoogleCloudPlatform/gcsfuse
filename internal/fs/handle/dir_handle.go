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
	"context"
	"fmt"
	"path"
	"slices"
	"time"
	// Removed unused "strconv" and "unicode"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
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
func (d *dirent) EntryName() string          { return d.Name }
func (d *dirent) SetName(name string)        { d.Name = name }
func (d *dirent) EntryType() fuseutil.DirentType { return d.Type }
func (d *dirent) SetOffset(offset fuseops.DirOffset) { d.Offset = offset }

// For DirentPlus
func (dp *direntPlus) EntryName() string          { return dp.Dirent.Name }
func (dp *direntPlus) SetName(name string)        { dp.Dirent.Name = name }
func (dp *direntPlus) EntryType() fuseutil.DirentType { return dp.Dirent.Type }
func (dp *direntPlus) SetOffset(offset fuseops.DirOffset) { dp.Dirent.Offset = offset }

const (
	// Default timeouts for directory entry attributes.
	DefaultEntryTimeout = time.Second
	DefaultAttrTimeout  = time.Second
)

// FileSystem defines the interface required by DirHandle to interact with the
// underlying file system, breaking the direct dependency.
type FileSystem interface {
	LookUpOrCreateInodeIfNotStale(ctx context.Context, ic inode.Core) (inode.Inode, error)
	GetAttributes(ctx context.Context, in inode.Inode) (fuseops.InodeAttributes, time.Time, error)
}

// DirHandle is the state required for reading from directories.
type DirHandle struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	in           inode.DirInode
	implicitDirs bool
	bucket       *gcsx.SyncerBucket
	fs           FileSystem // Interface to filesystem

	/////////////////////////
	// Mutable state
	/////////////////////////

	Mu locker.Locker

	// State for ReadDir (original buffering implementation)
	// GUARDED_BY(Mu)
	entries []fuseutil.Dirent
	// GUARDED_BY(Mu)
	entriesValid bool

	// State for streaming directory entries for ReadDirPlus.
	// GUARDED_BY(Mu)
	gcsMarkerPlus string
	// GUARDED_BY(Mu)
	gcsListDonePlus bool
	// GUARDED_BY(Mu)
	bufferedEntriesPlus []fuseutil.DirentPlus
	// GUARDED_BY(Mu)
	bufferIndexPlus int
	// GUARDED_BY(Mu)
	currentOffsetPlus fuseops.DirOffset
}

// NewDirHandle creates a directory handle that obtains listings from the supplied inode.
func NewDirHandle(
	in inode.DirInode,
	implicitDirs bool,
	bucket *gcsx.SyncerBucket,
	fs FileSystem) (dh *DirHandle) {
	// Set up the basic struct.
	dh = &DirHandle{
		in:                in,
		implicitDirs:      implicitDirs,
		bucket:            bucket,
		fs:                fs,
		currentOffsetPlus: 1, // Initialize streaming offset for ReadDirPlus
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

	// Add checks for ReadDirPlus state if needed in the future.
}

// coreToDirentPlus converts an inode.Name and inode.Core to a fuseutil.DirentPlus entry.
// It looks up the inode to get the ID and full attributes.
func (dh *DirHandle) coreToDirentPlus(ctx context.Context, name inode.Name, core *inode.Core) (*fuseutil.DirentPlus, error) {
	child, err := dh.fs.LookUpOrCreateInodeIfNotStale(dh.in.Context(), *core)
	if err != nil {
		return nil, fmt.Errorf("coreToDirentPlus: LookUpOrCreateInodeIfNotStale: %w", err)
	}
	if child == nil {
		// This can happen if the object was deleted between listing and lookup.
		return nil, nil
	}
	defer child.Unlock()

	attributes, expiration, err := dh.fs.GetAttributes(ctx, child)
	if err != nil {
		return nil, fmt.Errorf("coreToDirentPlus: GetAttributes: %w", err)
	}

	entry := &fuseutil.DirentPlus{
		Dirent: fuseutil.Dirent{
			Name:  path.Base(name.LocalName()),
			Type:  fuseutil.DT_Unknown, // Set below
			Inode: child.ID(),
		},
		Entry: fuseops.ChildInodeEntry{
			Child:                child.ID(),
			Attributes:           attributes,
			AttributesExpiration: expiration,
			EntryExpiration:      expiration,
		},
	}

	coreType := core.Type()
	switch coreType {
	case metadata.ImplicitDirType, metadata.ExplicitDirType:
		entry.Dirent.Type = fuseutil.DT_Directory
	case metadata.RegularFileType:
		entry.Dirent.Type = fuseutil.DT_File
	case metadata.SymlinkType:
		entry.Dirent.Type = fuseutil.DT_Link
	default:
		entry.Dirent.Type = fuseutil.DT_Unknown
	}

	return entry, nil
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

// ReadDir handles a request to read from the directory
// LOCKS_EXCLUDED(dh.Mu)
func (dh *DirHandle) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp,
	localFileEntries map[string]fuseutil.Dirent) (err error) {
	dh.Mu.Lock()
	defer dh.Mu.Unlock()

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

// ReadDirPlus handles a request to read from the directory with attributes.
// This implementation streams results from GCS to the kernel.
// LOCKS_EXCLUDED(dh.Mu)
func (dh *DirHandle) ReadDirPlus(ctx context.Context, op *fuseops.ReadDirPlusOp, localEntries map[string]fuseutil.DirentPlus) (err error) {
	logger.Infof("ReadDirPlus called with offset %d", op.Offset)
	dh.Mu.Lock()
	defer dh.Mu.Unlock()

	if op.Offset == 0 {
		// First call for this directory open, reset state.
		dh.gcsMarkerPlus = ""
		dh.gcsListDonePlus = false
		dh.bufferedEntriesPlus = nil
		dh.bufferIndexPlus = 0
		dh.currentOffsetPlus = 1
		logger.Tracef("ReadDirPlus: Resetting state for offset 0")
	} else if op.Offset != dh.currentOffsetPlus-1 {
		// Kernel is asking for an offset we don't expect.
		// This indicates an issue, potentially an invalid seek or a bug.
		logger.Warnf("ReadDirPlus: Offset mismatch, kernel wants %d, we are at %d. Returning EINVAL.", op.Offset, dh.currentOffsetPlus)
		return fuse.EINVAL
	}

	for { // Loop until the destination buffer (op.Dst) is full or entries are exhausted.

		// Step 1: Ensure there are entries in the current page buffer (dh.bufferedEntriesPlus)
		if dh.bufferIndexPlus >= len(dh.bufferedEntriesPlus) {
			if dh.gcsListDonePlus {
				logger.Tracef("ReadDirPlus: GCS list is done, no more entries.")
				break // No more entries to fetch from GCS.
			}

			logger.Tracef("ReadDirPlus: Buffer empty, fetching next page from GCS with marker '%s'", dh.gcsMarkerPlus)
			// Fetch the NEXT page of entries from GCS.
			var cores map[inode.Name]*inode.Core
			var newMarker string
			var unsupported []string
			func() {
				dh.in.Lock()
				defer dh.in.Unlock()
				cores, unsupported, newMarker, err = dh.in.ReadEntryCores(ctx, dh.gcsMarkerPlus)
			}()
			if err != nil {
				logger.Errorf("ReadDirPlus: inode.ReadEntryCores: %v", err)
				return fmt.Errorf("inode.ReadEntryCores: %w", err)
			}
			_ = unsupported // TODO(b/233580853): Handle unsupported paths if necessary.

			dh.gcsMarkerPlus = newMarker
			dh.gcsListDonePlus = (dh.gcsMarkerPlus == "")
			logger.Tracef("ReadDirPlus: GCS returned %d cores, new marker '%s', done: %t", len(cores), dh.gcsMarkerPlus, dh.gcsListDonePlus)

			// Extract inode.Name keys from the map
			names := make([]inode.Name, 0, len(cores))
			for name := range cores {
				names = append(names, name)
			}

			// Sort the keys lexicographically based on the full GCS object name.
			slices.SortFunc(names, func(a, b inode.Name) int {
				return cmp.Compare(a.GcsObjectName(), b.GcsObjectName())
			})

			// Build the batch in the correct lexicographical order
			batch := make([]fuseutil.DirentPlus, 0, len(names))
			for _, name := range names {
				core := cores[name]
				direntPlus, err := dh.coreToDirentPlus(ctx, name, core)
				if err != nil {
					logger.Warnf("ReadDirPlus: Failed to convert core to DirentPlus for %s: %v", name, err)
					continue
				}
				if direntPlus != nil {
					batch = append(batch, *direntPlus)
				}
			}

			dh.bufferedEntriesPlus = batch
			dh.bufferIndexPlus = 0

			// If the page from GCS was empty, and GCS is not done, loop again to fetch the next page.
			if len(dh.bufferedEntriesPlus) == 0 && !dh.gcsListDonePlus {
				logger.Tracef("ReadDirPlus: Empty page received, but GCS not done, continuing fetch.")
				continue
			}
			// If the page is empty and GCS is done, break out.
			if len(dh.bufferedEntriesPlus) == 0 && dh.gcsListDonePlus {
				logger.Tracef("ReadDirPlus: Empty page received and GCS is done.")
				break
			}
		}

		// Step 2: Write entries from the current buffer to op.Dst.
		entry := dh.bufferedEntriesPlus[dh.bufferIndexPlus]
		entry.Dirent.Offset = dh.currentOffsetPlus

		// Log entry being written for debugging order
		logger.Tracef("ReadDirPlus: Writing entry to kernel: %s, Offset: %d", entry.Dirent.Name, entry.Dirent.Offset)

		n := fuseutil.WriteDirentPlus(op.Dst[op.BytesRead:], entry)
		if n == 0 {
			// Destination buffer is full. Return, and the kernel will call again.
			logger.Tracef("ReadDirPlus: Kernel buffer full, returning. Next offset to expect: %d", dh.currentOffsetPlus)
			return nil
		}
		op.BytesRead += n
		dh.bufferIndexPlus++
		dh.currentOffsetPlus++
	}

	// TODO(b/233580853): Add any remaining localEntries if applicable after exhausting GCS entries.
	// This requires careful sorting and merging.
	logger.Tracef("ReadDirPlus: Finished serving entries for this call.")
	return nil
}
