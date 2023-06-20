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
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"golang.org/x/net/context"
)



type Record struct {
	sync.Mutex
	length int

	cond *sync.Cond
}

func NewRecord() *Record {
	r := Record{}
	r.cond = sync.NewCond(&r)
	return &r
}

var ContinuationToken string
var length int
var rec = NewRecord()


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

// Read all entries for the directory, fix up conflicting names, and fill in
// offset fields.
//
// LOCKS_REQUIRED(in)
func readAllEntries(
	ctx context.Context,
	in inode.DirInode) (entries []fuseutil.Dirent, err error) {
	// Read one batch
	var batch []fuseutil.Dirent
	batch, ContinuationToken, err = in.ReadEntries(ctx, ContinuationToken)
	if err != nil {
		err = fmt.Errorf("ReadEntries: %w", err)
		return
	}
	// Accumulate.
	entries = append(entries, batch...)

	// Ensure that the entries are sorted, for use in fixConflictingNames
	// below.
	// TODO: Fix this after  asynchronous fetch is added.
	sort.Sort(sortedDirents(entries))

	// Fix name conflicts.
	err = fixConflictingNames(entries)
	if err != nil {
		err = fmt.Errorf("fixConflictingNames: %w", err)
		return
	}

	// Return a bogus inode ID for each entry, but not the root inode ID.
	//
	// NOTE(jacobsa): As far as I can tell this is harmless. Minting and
	// returning a real inode ID is difficult because fuse does not count
	// readdir as an operation that increases the inode ID's lookup count and
	// we therefore don't get a forget for it later, but we would like to not
	// have to remember every inode ID that we've ever minted for readdir.
	//
	// If it turns out this is not harmless, we'll need to switch to something
	// like inode IDs based on (object name, generation) hashes. But then what
	// about the birthday problem? And more importantly, what about our
	// semantic of not minting a new inode ID when the generation changes due
	// to a local action?
	for i, _ := range entries {
		entries[i].Inode = fuseops.RootInodeID + 1
	}

	return
}

// LOCKS_REQUIRED(dh.Mu)
// LOCKS_EXCLUDED(dh.in)
func (dh *dirHandle) ensureEntries(ctx context.Context) (err error) {
	dh.in.Lock()
	defer dh.in.Unlock()

	// Read entries.
	var entries []fuseutil.Dirent
	entries, err = readAllEntries(ctx, dh.in)
	if err != nil {
		err = fmt.Errorf("readAllEntries: %w", err)
		return
	}

	// Update state.
	// Fix up offset fields.
	for i := 0; i < len(entries); i++ {
		entries[i].Offset = fuseops.DirOffset(uint64(len(dh.entries) + i + 1))
	}
	dh.entries = append(dh.entries, entries...)
	dh.entriesValid = true

	return
}

func (dh *dirHandle) FetchEntriesAsync(
    ctx context.Context,
    rootInodeId int,
    firstCall int ) (err error) {

   logger.Infof("Started fetching entries \n")

   for ContinuationToken != "" || firstCall==1 {

        var entries []fuseutil.Dirent


        dh.in.Lock()
        logger.Info("Inode lock acquired\n")
        ctx = context.Background()
        entries,ContinuationToken,err = dh.in.ReadEntries(ctx, ContinuationToken)
        logger.Infof("Entries fetched from dh.in.ReadEntries : %v",len(entries))
        dh.in.Unlock()
        logger.Info("Inode lock released\n")




        if err != nil{
            err = fmt.Errorf("ReadEntries: %w",err)
            logger.Infof("Issue with the dh.in.read Entries: %v \n",err)
            return
        }
        sort.Sort(sortedDirents(entries))
        logger.Info("Sorting done!\n")
        err = fixConflictingNames(entries)
        if err != nil{
            err = fmt.Errorf("fixConflictingNames: %w",err)
            logger.Infof("Issue with the fixing conflicts: %w \n",err)
            return
        }

        logger.Info("Fixed naming conflicts!\n")
        rec.Lock()
        logger.Info("rec Lock acquired\n")

        for i,_ := range entries {
            entries[i].Inode = fuseops.InodeID(rootInodeId + 1)
            entries[i].Offset =fuseops.DirOffset(uint64(rec.length + i + 1))
        }

        logger.Info("Entries fileds updated!\n")

        rec.length += len(entries)
        logger.Infof("Rec.length is now : %v \n",rec.length)

        rec.Unlock()

        logger.Info("rec Lock released\n")
        dh.Mu.Lock()
        logger.Info("Mutex Lock acquired \n")
        dh.entries = append(dh.entries,entries...)
        dh.entriesValid = true
        dh.Mu.Unlock()
        logger.Info("Mutex Lock released \n")

        rec.cond.Broadcast()
        logger.Info("Broadcasted !\n")


        firstCall = 0

    }
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
func (dh *dirHandle) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) (err error) {
	// If the request is for offset zero, we assume that either this is the first
	// call or rewinddir has been called. Reset state.
    logger.Infof("Called readdir at offset : %v",op.Offset)
    dh.Mu.Lock()
    logger.Infof("Entries present : %v",len(dh.entries))
    dh.Mu.Unlock()
	if op.Offset == 0 {

		dh.entries = nil
		dh.entriesValid = false

		logger.Infof("Done with init \n")
		go dh.FetchEntriesAsync(ctx,fuseops.RootInodeID,1)

	}



    logger.Info("readdir served first\n")
    rec.Lock()
    if rec.length <= int(op.Offset) && ( op.Offset == 0 || ContinuationToken != ""){

            logger.Info("Before getting blocked on the length \n")

            rec.cond.Wait()
            logger.Info("Got unblocked")


    }


        // Is the offset past the end of what we have buffered? If so, this must be
        	// an invalid seekdir according to posix.
        	index := int(op.Offset)
        	if index > rec.length {
        	    logger.Infof("Error faced in fuse.EINVAL")
        		err = fuse.EINVAL
        		return
        	}
rec.Unlock()
        	// We copy out entries until we run out of entries or space.

        	dh.Mu.Lock()
        	for i := index; i < len(dh.entries); i++ {
        		n := fuseutil.WriteDirent(op.Dst[op.BytesRead:], dh.entries[i])

        		if n == 0 {
        		logger.Infof("Write Dirent stopped at %v",i)
        			break
        		}

        		op.BytesRead += n
        	}
        	dh.Mu.Unlock()


    logger.Infof("Main go at offset %v got out first \n",op.Offset)


	return
}
