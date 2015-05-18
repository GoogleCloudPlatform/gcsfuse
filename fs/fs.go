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
	"math"
	"os/user"
	"reflect"
	"strconv"
	"time"

	"github.com/googlecloudplatform/gcsfuse/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
)

type ServerConfig struct {
	// A clock used for modification times and cache expiration.
	Clock timeutil.Clock

	// The bucket that the file system is to export.
	Bucket gcs.Bucket

	// The temporary directory to use for local caching, or the empty string to
	// use the system default.
	TempDir string

	// A desired limit on temporary space usage, in bytes. May not be obeyed if
	// there is a large volume of dirtied files that have not been flushed or
	// closed.
	TempDirLimit int64

	// If set to a non-zero value N, the file system will read objects from GCS a
	// chunk at a time with a maximum read size of N, caching each chunk
	// independently. The part about separate caching does not apply to dirty
	// files, for which the entire contents will be in the temporary directory
	// regardless of this setting.
	GCSChunkSize uint64

	// By default, if a bucket contains the object "foo/bar" but no object named
	// "foo/", it's as if the directory doesn't exist. This allows us to have
	// non-flaky name resolution code.
	//
	// Setting this bool to true enables a mode where object listings are
	// consulted to allow for the directory in the situation above to exist. Note
	// that this has drawbacks in the form of name resolution flakiness and
	// surprising behavior.
	//
	// See docs/semantics.md for more info.
	ImplicitDirectories bool

	// By default, the file system will always show nlink == 1 for every inode,
	// regardless of whether its backing object has been deleted or overwritten.
	//
	// Setting SupportNlink to true causes the file system to respond to fuse
	// getattr requests with nlink == 0 for file inodes in the cases mentioned
	// above. This requires a round trip to GCS for every getattr, which can be
	// quite slow.
	SupportNlink bool

	// If non-zero, each directory will maintain a cache from child name to
	// information about whether that name exists as a file and/or directory.
	// This may speed up calls to look up and stat inodes, especially when
	// combined with a stat-caching GCS bucket, but comes at the cost of
	// consistency: if the child is removed and recreated with a different type
	// before the expiration, we may fail to find it.
	DirTypeCacheTTL time.Duration
}

// Create a fuse file system server according to the supplied configuration.
func NewServer(cfg *ServerConfig) (server fuse.Server, err error) {
	// Get ownership information.
	uid, gid, err := getUser()
	if err != nil {
		return
	}

	// Disable chunking if set to zero.
	gcsChunkSize := cfg.GCSChunkSize
	if gcsChunkSize == 0 {
		gcsChunkSize = math.MaxUint64
	}

	// Set up the basic struct.
	fs := &fileSystem{
		clock:           cfg.Clock,
		bucket:          cfg.Bucket,
		leaser:          lease.NewFileLeaser(cfg.TempDir, cfg.TempDirLimit),
		gcsChunkSize:    gcsChunkSize,
		implicitDirs:    cfg.ImplicitDirectories,
		supportNlink:    cfg.SupportNlink,
		dirTypeCacheTTL: cfg.DirTypeCacheTTL,
		uid:             uid,
		gid:             gid,
		inodes:          make(map[fuseops.InodeID]inode.Inode),
		nextInodeID:     fuseops.RootInodeID + 1,
		fileIndex:       make(map[string]*inode.FileInode),
		dirIndex:        make(map[string]*inode.DirInode),
		handles:         make(map[fuseops.HandleID]interface{}),
	}

	// Set up the root inode.
	root := inode.NewRootInode(
		fs.implicitDirs,
		fs.dirTypeCacheTTL,
		cfg.Bucket,
		fs.clock)

	root.Lock()
	root.IncrementLookupCount()
	fs.inodes[fuseops.RootInodeID] = root
	fs.dirIndex[root.Name()] = root
	root.Unlock()

	// Set up invariant checking.
	fs.mu = syncutil.NewInvariantMutex(fs.checkInvariants)

	server = fuseutil.NewFileSystemServer(fs)
	return
}

////////////////////////////////////////////////////////////////////////
// fileSystem type
////////////////////////////////////////////////////////////////////////

// LOCK ORDERING
//
// Let FS be the file system lock. Define a strict partial order < as follows:
//
//  1. For any inode lock I, I < FS.
//  2. For any directory handle lock DH and inode lock I, DH < I.
//
// We follow the rule "acquire A then B only if A < B".
//
// In other words:
//
//  *  Don't hold multiple directory handle locks at the same time.
//  *  Don't hold multiple inode locks at the same time.
//  *  Don't acquire inode locks before directory handle locks.
//  *  Don't acquire file system locks before either.
//
// The intuition is that we hold inode and directory handle locks for
// long-running operations, and we don't want to block the entire file system
// on those.
//
// See http://goo.gl/rDxxlG for more discussion, including an informal proof
// that a strict partial order is sufficient.

type fileSystem struct {
	fuseutil.NotImplementedFileSystem

	/////////////////////////
	// Dependencies
	/////////////////////////

	clock  timeutil.Clock
	bucket gcs.Bucket
	leaser lease.FileLeaser

	/////////////////////////
	// Constant data
	/////////////////////////

	gcsChunkSize    uint64
	implicitDirs    bool
	supportNlink    bool
	dirTypeCacheTTL time.Duration

	// The user and group owning everything in the file system.
	uid uint32
	gid uint32

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A lock protecting the state of the file system struct itself (distinct
	// from per-inode locks). Make sure to see the notes on lock ordering above.
	mu syncutil.InvariantMutex

	// The next inode ID to hand out. We assume that this will never overflow,
	// since even if we were handing out inode IDs at 4 GHz, it would still take
	// over a century to do so.
	//
	// GUARDED_BY(mu)
	nextInodeID fuseops.InodeID

	// The collection of live inodes, keyed by inode ID. No ID less than
	// fuseops.RootInodeID is ever used.
	//
	// INVARIANT: For all keys k, fuseops.RootInodeID <= k < nextInodeID
	// INVARIANT: For all keys k, inodes[k].ID() == k
	// INVARIANT: inodes[fuseops.RootInodeID] is missing or of type *inode.DirInode
	// INVARIANT: For all v, if isDirName(v.Name()) then v is *inode.DirInode
	// INVARIANT: For all v, if !isDirName(v.Name()) then v is *inode.FileInode
	//
	// GUARDED_BY(mu)
	inodes map[fuseops.InodeID]inode.Inode

	// A map from object name to a file inode that represents that name.
	// Populated during the name -> inode lookup process, cleared during the
	// forget inode process.
	//
	// Entries may be stale for two reasons:
	//
	//  1. There is a newer generation in GCS, not caused by the inode. The next
	//     name lookup will detect this by statting the object, acquiring the
	//     inode's lock (to get an up to date look at what the latest generation
	//     the inode caused was), and replacing the entry if the inode's
	//     generation is less than the stat generation.
	//
	//  2. The object no longer exists. This is harmless; the name lookup process
	//     will return ENOENT before it ever consults this map. Eventually the
	//     kernel will send ForgetInodeOp and we will clear the entry.
	//
	// Crucially, we never replace an up to date entry with a stale one. If the
	// name lookup process sees that the stat result is older than the inode, it
	// starts over, statting again.
	//
	// Note that there is no invariant that says *all* of the file inodes are
	// represented here because we may have multiple distinct inodes for a given
	// name existing concurrently if we observe an object generation that was not
	// caused by our existing inode (e.g. if the file is clobbered remotely). We
	// must retain the old inode until the kernel tells us to forget it.
	//
	// INVARIANT: For each k/v, v.Name() == k
	// INVARIANT: For each value v, inodes[v.ID()] == v
	//
	// GUARDED_BY(mu)
	fileIndex map[string]*inode.FileInode

	// A map from object name to the directory inode that represents that name,
	// if any. There can be at most one inode for a given name accessible to us
	// at any given time.
	//
	// INVARIANT: For each k/v, v.Name() == k
	// INVARIANT: For each value v, inodes[v.ID()] == v
	// INVARIANT: For each *inode.DirInode d in inodes, dirIndex[d.Name()] == d
	//
	// GUARDED_BY(mu)
	dirIndex map[string]*inode.DirInode

	// The collection of live handles, keyed by handle ID.
	//
	// INVARIANT: All values are of type *dirHandle
	//
	// GUARDED_BY(mu)
	handles map[fuseops.HandleID]interface{}

	// The next handle ID to hand out. We assume that this will never overflow.
	//
	// INVARIANT: For all keys k in handles, k < nextHandleID
	//
	// GUARDED_BY(mu)
	nextHandleID fuseops.HandleID
}

func getUser() (uid uint32, gid uint32, err error) {
	// Ask for the current user.
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	// Parse UID.
	uid64, err := strconv.ParseUint(user.Uid, 10, 32)
	if err != nil {
		err = fmt.Errorf("Parsing UID (%s): %v", user.Uid, err)
		return
	}

	// Parse GID.
	gid64, err := strconv.ParseUint(user.Gid, 10, 32)
	if err != nil {
		err = fmt.Errorf("Parsing GID (%s): %v", user.Gid, err)
		return
	}

	uid = uint32(uid64)
	gid = uint32(gid64)

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func isDirName(name string) bool {
	return name == "" || name[len(name)-1] == '/'
}

func (fs *fileSystem) checkInvariants() {
	//////////////////////////////////
	// inodes
	//////////////////////////////////

	// INVARIANT: For all keys k, fuseops.RootInodeID <= k < nextInodeID
	for id, _ := range fs.inodes {
		if id < fuseops.RootInodeID || id >= fs.nextInodeID {
			panic(fmt.Sprintf("Illegal inode ID: %v", id))
		}
	}

	// INVARIANT: For all keys k, inodes[k].ID() == k
	for id, in := range fs.inodes {
		if in.ID() != id {
			panic(fmt.Sprintf("ID mismatch: %v vs. %v", in.ID(), id))
		}
	}

	// INVARIANT: inodes[fuseops.RootInodeID] is missing or of type *inode.DirInode
	//
	// The missing case is when we've received a forget request for the root
	// inode, while unmounting.
	switch in := fs.inodes[fuseops.RootInodeID].(type) {
	case nil:
	case *inode.DirInode:
	default:
		panic(fmt.Sprintf("Unexpected type for root: %v", reflect.TypeOf(in)))
	}

	// INVARIANT: For all v, if isDirName(v.Name()) then v is *inode.DirInode
	for _, in := range fs.inodes {
		if isDirName(in.Name()) {
			_, ok := in.(*inode.DirInode)
			if !ok {
				panic(fmt.Sprintf(
					"Unexpected inode type for name \"%s\": %v",
					in.Name(),
					reflect.TypeOf(in)))
			}
		}
	}

	// INVARIANT: For all v, if !isDirName(v.Name()) then v is *inode.FileInode
	for _, in := range fs.inodes {
		if !isDirName(in.Name()) {
			_, ok := in.(*inode.FileInode)
			if !ok {
				panic(fmt.Sprintf(
					"Unexpected inode type for name \"%s\": %v",
					in.Name(),
					reflect.TypeOf(in)))
			}
		}
	}

	//////////////////////////////////
	// fileIndex
	//////////////////////////////////

	// INVARIANT: For each k/v, v.Name() == k
	for k, v := range fs.fileIndex {
		if !(v.Name() == k) {
			panic(fmt.Sprintf(
				"Unexpected name: \"%s\" vs. \"%s\"",
				v.Name(),
				k))
		}
	}

	// INVARIANT: For each value v, inodes[v.ID()] == v
	for _, v := range fs.fileIndex {
		if fs.inodes[v.ID()] != v {
			panic(fmt.Sprintf(
				"Mismatch for ID %v: %p %p",
				v.ID(),
				fs.inodes[v.ID()],
				v))
		}
	}

	//////////////////////////////////
	// dirIndex
	//////////////////////////////////

	// INVARIANT: For each k/v, v.Name() == k
	for k, v := range fs.dirIndex {
		if !(v.Name() == k) {
			panic(fmt.Sprintf(
				"Unexpected name: \"%s\" vs. \"%s\"",
				v.Name(),
				k))
		}
	}

	// INVARIANT: For each value v, inodes[v.ID()] == v
	for _, v := range fs.dirIndex {
		if fs.inodes[v.ID()] != v {
			panic(fmt.Sprintf(
				"Mismatch for ID %v: %p %p",
				v.ID(),
				fs.inodes[v.ID()],
				v))
		}
	}

	// INVARIANT: For each *inode.DirInode d in inodes, dirIndex[d.Name()] == d
	for _, in := range fs.inodes {
		if d, ok := in.(*inode.DirInode); ok {
			if !(fs.dirIndex[d.Name()] == d) {
				panic(fmt.Sprintf(
					"dirIndex mismatch: %q %p %p",
					d.Name(),
					fs.dirIndex[d.Name()],
					d))
			}
		}
	}

	//////////////////////////////////
	// handles
	//////////////////////////////////

	// INVARIANT: All values are of type *dirHandle
	for _, h := range fs.handles {
		_ = h.(*dirHandle)
	}

	//////////////////////////////////
	// nextHandleID
	//////////////////////////////////

	// INVARIANT: For all keys k in handles, k < nextHandleID
	for k, _ := range fs.handles {
		if k >= fs.nextHandleID {
			panic(fmt.Sprintf("Illegal handle ID: %v", k))
		}
	}
}

// Get attributes for the inode, fixing up ownership information.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCKS_REQUIRED(in)
func (fs *fileSystem) getAttributes(
	ctx context.Context,
	in inode.Inode) (attrs fuseops.InodeAttributes, err error) {
	attrs, err = in.Attributes(ctx)
	if err != nil {
		return
	}

	attrs.Uid = fs.uid
	attrs.Gid = fs.gid

	return
}

// Implementation detail of lookUpOrCreateInodeIfNotStale; do not use outside
// of that function.
//
// LOCKS_REQUIRED(fs.mu)
func (fs *fileSystem) mintInode(o *gcs.Object) (in inode.Inode) {
	// Choose an ID.
	id := fs.nextInodeID
	fs.nextInodeID++

	// Create the inode.
	if isDirName(o.Name) {
		d := inode.NewDirInode(
			id,
			o.Name,
			fs.implicitDirs,
			fs.dirTypeCacheTTL,
			fs.bucket,
			fs.clock)

		fs.dirIndex[d.Name()] = d
		in = d
	} else {
		in = inode.NewFileInode(
			id,
			o,
			fs.gcsChunkSize,
			fs.supportNlink,
			fs.bucket,
			fs.leaser,
			fs.clock)
	}

	// Place it in our map of IDs to inodes.
	fs.inodes[in.ID()] = in

	return
}

// Attempt to find an inode for the given object record, or create one if one
// has never yet existed and the record is newer than any inode we've yet
// recorded.
//
// If the record is stale (i.e. some newer inode exists), return nil. In this
// case, the caller may obtain a fresh record and try again.
//
// Special case: We don't care about generation numbers for directories.
//
// UNLOCK_FUNCTION(fs.mu)
// LOCK_FUNCTION(in)
func (fs *fileSystem) lookUpOrCreateInodeIfNotStale(
	o *gcs.Object) (in inode.Inode) {
	// Ensure that no matter which inode we return, we increase its lookup count
	// on the way out.
	defer func() {
		if in != nil {
			in.IncrementLookupCount()
		}
	}()

	// Handle directories.
	if isDirName(o.Name) {
		var ok bool

		// If we don't have an entry, create one.
		in, ok = fs.dirIndex[o.Name]
		if !ok {
			in = fs.mintInode(o)
		}

		fs.mu.Unlock()
		in.Lock()

		return
	}

	// Retry loop for the stale index entry case below. On entry, we hold fs.mu
	// but no inode lock.
	for {
		// Look at the current index entry.
		existingInode, ok := fs.fileIndex[o.Name]

		// If we have no existing record for this name, mint an inode and return it.
		if !ok {
			in = fs.mintInode(o)
			fs.fileIndex[in.Name()] = in.(*inode.FileInode)

			fs.mu.Unlock()
			in.Lock()

			return
		}

		// Otherwise we need to grab the inode lock to find out if this is our
		// inode, our record is stale, or the inode is stale. are stale compared to
		// it. We must exclude concurrent actions on the inode to get a definitive
		// answer.
		//
		// Drop the file system lock and acquire the inode lock.
		fs.mu.Unlock()
		existingInode.Lock()

		// Have we found the correct inode?
		if o.Generation == existingInode.SourceGeneration() {
			in = existingInode
			return
		}

		// Are we stale?
		if o.Generation < existingInode.SourceGeneration() {
			existingInode.Unlock()
			return
		}

		// We've observed that the record is newer than the existing inode, while
		// holding the inode lock, excluding concurrent actions by the inode (in
		// particular concurrent calls to Sync, which changes generation numbers).
		// This means we've proven that the record cannot have been caused by the
		// inode's actions, and therefore it is not the inode we want.
		//
		// Re-acquire the file system lock. If the index entry still points at
		// existingInode, we have proven we can replace it with an entry for a a
		// newly-minted inode.
		fs.mu.Lock()
		if fs.fileIndex[o.Name] == existingInode {
			in = fs.mintInode(o)
			fs.fileIndex[in.Name()] = in.(*inode.FileInode)

			fs.mu.Unlock()
			existingInode.Unlock()
			in.Lock()

			return
		}

		// The index entry has been changed in the meantime, so there may be a new
		// inode that we have to contend with. Go around and try again.
		existingInode.Unlock()
	}
}

// Given a function that returns a "fresh" object record, implement the calling
// loop documented for lookUpOrCreateInodeIfNotStale. Call the function once to
// begin with, and again each time it returns a stale record.
//
// For each call to f, neither the file system mutex nor any inode locks will
// be held. The caller of this function must also not hold any inode locks.
//
// Return ENOENT if the function ever returns a nil record. Never return a nil
// inode with a nil error.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCK_FUNCTION(in)
func (fs *fileSystem) lookUpOrCreateInode(
	f func() (*gcs.Object, error)) (in inode.Inode, err error) {
	const maxTries = 3
	for n := 0; n < maxTries; n++ {
		// Create a record.
		var o *gcs.Object
		o, err = f()

		if err != nil {
			return
		}

		if o == nil {
			err = fuse.ENOENT
			return
		}

		// Attempt to create the inode. Return if successful.
		fs.mu.Lock()
		in = fs.lookUpOrCreateInodeIfNotStale(o)
		if in != nil {
			return
		}
	}

	err = fmt.Errorf("Did not converge after %v tries", maxTries)
	return
}

// Synchronize the supplied file inode to GCS, updating the index as
// appropriate.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCKS_REQUIRED(f)
func (fs *fileSystem) syncFile(
	ctx context.Context,
	f *inode.FileInode) (err error) {
	// Sync the inode.
	err = f.Sync(ctx)
	if err != nil {
		err = fmt.Errorf("FileInode.Sync: %v", err)
		return
	}

	// We need not update fileIndex:
	//
	// We've held the inode lock the whole time, so there's no way that this
	// inode could have been booted from the index. Therefore if it's not in the
	// index at the moment, it must not have been in there when we started. That
	// is, it must have been clobbered remotely, which we treat as unlinking.
	//
	// In other words, either this inode is still in the index or it has been
	// unlinked and *should* be anonymous.

	return
}

////////////////////////////////////////////////////////////////////////
// fuse.FileSystem methods
////////////////////////////////////////////////////////////////////////

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) Init(
	op *fuseops.InitOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) LookUpInode(
	op *fuseops.LookUpInodeOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	// Find the parent directory in question.
	fs.mu.Lock()
	parent := fs.inodes[op.Parent].(*inode.DirInode)
	fs.mu.Unlock()

	// Set up a function that will find a record for the child with the given
	// name, or nil if none.
	f := func() (o *gcs.Object, err error) {
		parent.Lock()
		defer parent.Unlock()

		// We need not hold a lock for LookUpChild.
		o, err = parent.LookUpChild(op.Context(), op.Name)
		if err != nil {
			err = fmt.Errorf("LookUpChild: %v", err)
			return
		}

		return
	}

	// Use that function to find or mint an inode.
	in, err := fs.lookUpOrCreateInode(f)
	if err != nil {
		return
	}

	defer in.Unlock()

	// Fill out the response.
	op.Entry.Child = in.ID()
	if op.Entry.Attributes, err = fs.getAttributes(op.Context(), in); err != nil {
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) GetInodeAttributes(
	op *fuseops.GetInodeAttributesOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode]
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Grab its attributes.
	op.Attributes, err = fs.getAttributes(op.Context(), in)
	if err != nil {
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) SetInodeAttributes(
	op *fuseops.SetInodeAttributesOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode]
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// The only thing we support changing is size, and then only for directories.
	if op.Mode != nil || op.Atime != nil || op.Mtime != nil {
		err = fuse.ENOSYS
		return
	}

	file, ok := in.(*inode.FileInode)
	if !ok {
		err = fuse.ENOSYS
		return
	}

	// Set the size, if specified.
	if op.Size != nil {
		if err = file.Truncate(op.Context(), int64(*op.Size)); err != nil {
			err = fmt.Errorf("Truncate: %v", err)
			return
		}
	}

	// Fill in the response.
	op.Attributes, err = fs.getAttributes(op.Context(), in)
	if err != nil {
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ForgetInode(
	op *fuseops.ForgetInodeOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode]
	fs.mu.Unlock()

	// Acquire the inode lock without holding the file system lock.
	in.Lock()
	defer in.Unlock()

	// Re-acquire the file system lock to exclude concurrent lookups (which may
	// otherwise find an inode whose lookup count has gone to zero), then
	// decrement the lookup count.
	//
	// If we're told to destroy the inode, remove it from the file system
	// immediately to make it inaccessible, then destroy it without holding the
	// file system lock (since doing so may block).
	fs.mu.Lock()

	name := in.Name()
	shouldDestroy := in.DecrementLookupCount(op.N)
	if shouldDestroy {
		delete(fs.inodes, op.Inode)

		// Update indexes if necessary.
		if fs.fileIndex[name] == in {
			delete(fs.fileIndex, name)
		}

		if fs.dirIndex[name] == in {
			delete(fs.dirIndex, name)
		}
	}

	fs.mu.Unlock()

	if shouldDestroy {
		err = in.Destroy()
		if err != nil {
			err = fmt.Errorf("Destroy: %v", err)
			return
		}
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) MkDir(
	op *fuseops.MkDirOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	// Find the parent.
	fs.mu.Lock()
	parent := fs.inodes[op.Parent].(*inode.DirInode)
	fs.mu.Unlock()

	// Create an empty backing object for the child, failing if it already
	// exists.
	parent.Lock()
	o, err := parent.CreateChildDir(op.Context(), op.Name)
	parent.Unlock()
	if err != nil {
		err = fmt.Errorf("CreateChildDir: %v", err)
		return
	}

	// Attempt to create a child inode using the object we created. If we fail to
	// do so, it means someone beat us to the punch with a newer generation
	// (unlikely, so we're probably okay with failing here).
	fs.mu.Lock()
	child := fs.lookUpOrCreateInodeIfNotStale(o)
	if child == nil {
		err = fmt.Errorf("Newly-created record is already stale")
		return
	}

	defer child.Unlock()

	// Fill out the response.
	op.Entry.Child = child.ID()
	op.Entry.Attributes, err = fs.getAttributes(op.Context(), child)

	if err != nil {
		err = fmt.Errorf("getAttributes: %v", err)
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) CreateFile(
	op *fuseops.CreateFileOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	// Find the parent.
	fs.mu.Lock()
	parent := fs.inodes[op.Parent].(*inode.DirInode)
	fs.mu.Unlock()

	// Create an empty backing object for the child, failing if it already
	// exists.
	parent.Lock()
	o, err := parent.CreateChildFile(op.Context(), op.Name)
	parent.Unlock()
	if err != nil {
		err = fmt.Errorf("CreateChildFile: %v", err)
		return
	}

	// Attempt to create a child inode using the object we created. If we fail to
	// do so, it means someone beat us to the punch with a newer generation
	// (unlikely, so we're probably okay with failing here).
	fs.mu.Lock()
	child := fs.lookUpOrCreateInodeIfNotStale(o)
	if child == nil {
		err = fmt.Errorf("Newly-created record is already stale")
		return
	}

	defer child.Unlock()

	// Fill out the response.
	op.Entry.Child = child.ID()
	op.Entry.Attributes, err = fs.getAttributes(op.Context(), child)

	if err != nil {
		err = fmt.Errorf("getAttributes: %v", err)
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) RmDir(
	op *fuseops.RmDirOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	// Find the parent. We assume that it exists because otherwise the kernel has
	// done something mildly concerning.
	fs.mu.Lock()
	parent := fs.inodes[op.Parent].(*inode.DirInode)
	fs.mu.Unlock()

	// Delete the backing object.
	//
	// No lock is required.
	parent.Lock()
	err = parent.DeleteChildDir(op.Context(), op.Name)
	parent.Unlock()
	if err != nil {
		err = fmt.Errorf("DeleteChildDir: %v", err)
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) Unlink(
	op *fuseops.UnlinkOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	// Find the parent.
	fs.mu.Lock()
	parent := fs.inodes[op.Parent].(*inode.DirInode)
	fs.mu.Unlock()

	// Delete the backing object.
	//
	// No lock is required here.
	parent.Lock()
	err = parent.DeleteChildFile(op.Context(), op.Name)
	parent.Unlock()
	if err != nil {
		err = fmt.Errorf("DeleteChildFile: %v", err)
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) OpenDir(
	op *fuseops.OpenDirOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Make sure the inode still exists and is a directory. If not, something has
	// screwed up because the VFS layer shouldn't have let us forget the inode
	// before opening it.
	in := fs.inodes[op.Inode].(*inode.DirInode)

	// Allocate a handle.
	handleID := fs.nextHandleID
	fs.nextHandleID++

	fs.handles[handleID] = newDirHandle(in, fs.implicitDirs)
	op.Handle = handleID

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReadDir(
	op *fuseops.ReadDirOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	// Find the handle.
	fs.mu.Lock()
	dh := fs.handles[op.Handle].(*dirHandle)
	fs.mu.Unlock()

	dh.Mu.Lock()
	defer dh.Mu.Unlock()

	// Serve the request.
	err = dh.ReadDir(op)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReleaseDirHandle(
	op *fuseops.ReleaseDirHandleOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Sanity check that this handle exists and is of the correct type.
	_ = fs.handles[op.Handle].(*dirHandle)

	// Clear the entry from the map.
	delete(fs.handles, op.Handle)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) OpenFile(
	op *fuseops.OpenFileOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Sanity check that this inode exists and is of the correct type.
	_ = fs.inodes[op.Inode].(*inode.FileInode)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReadFile(
	op *fuseops.ReadFileOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode].(*inode.FileInode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Serve the request.
	op.Data, err = in.Read(op.Context(), op.Offset, op.Size)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) WriteFile(
	op *fuseops.WriteFileOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode].(*inode.FileInode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Serve the request.
	err = in.Write(op.Context(), op.Data, op.Offset)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) SyncFile(
	op *fuseops.SyncFileOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode].(*inode.FileInode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Sync it.
	err = fs.syncFile(op.Context(), in)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) FlushFile(
	op *fuseops.FlushFileOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode].(*inode.FileInode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Sync it.
	err = fs.syncFile(op.Context(), in)

	return
}
