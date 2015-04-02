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
	"os/user"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
)

type ServerConfig struct {
	// A clock used for cache validation and modification times.
	Clock timeutil.Clock

	// The bucket that the file system is to export.
	Bucket gcs.Bucket

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
}

// Create a fuse file system server according to the supplied configuration.
func NewServer(cfg *ServerConfig) (server fuse.Server, err error) {
	// Get ownership information.
	uid, gid, err := getUser()
	if err != nil {
		return
	}

	// Set up the basic struct.
	fs := &fileSystem{
		clock:        cfg.Clock,
		bucket:       cfg.Bucket,
		implicitDirs: cfg.ImplicitDirectories,
		uid:          uid,
		gid:          gid,
		inodes:       make(map[fuseops.InodeID]inode.Inode),
		nextInodeID:  fuseops.RootInodeID + 1,
		inodeIndex:   make(map[string]cachedGen),
		handles:      make(map[fuseops.HandleID]interface{}),
	}

	// Set up the root inode.
	root := inode.NewRootInode(cfg.Bucket, fs.implicitDirs)

	root.Lock()
	root.IncrementLookupCount()
	fs.inodes[fuseops.RootInodeID] = root
	fs.inodeIndex[root.Name()] = cachedGen{root, root.SourceGeneration()}
	root.Unlock()

	// Set up invariant checking.
	fs.mu = syncutil.NewInvariantMutex(fs.checkInvariants)

	server = fuseutil.NewFileSystemServer(fs)
	return
}

////////////////////////////////////////////////////////////////////////
// fileSystem type
////////////////////////////////////////////////////////////////////////

// Two simple rules about lock ordering:
//
//  1. No two inode locks may be held at the same time.
//  2. No inode lock may be acquired while holding the file system lock.
//
// In other words, the strict partial order is defined by all pairs (I, FS)
// where I is any inode lock and FS is the file system lock.
//
// See http://goo.gl/rDxxlG for more discussion, including an informal proof
// that a strict partial order is sufficient.

type fileSystem struct {
	fuseutil.NotImplementedFileSystem

	/////////////////////////
	// Dependencies
	/////////////////////////

	clock        timeutil.Clock
	bucket       gcs.Bucket
	implicitDirs bool

	/////////////////////////
	// Constant data
	/////////////////////////

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

	// A map from object name to an inode I backed by that object, where I has
	// the largest generation number we've yet observed for that name (possibly
	// since forgetting an inode for the name), and the generation number that we
	// most recently observed for I.
	//
	// In order to replace an entry in this map, you must hold in hand a
	// gcs.Object record whose generation number is larger than I's current
	// generation number (not the cached one, which may be out of date if I has
	// recently been sync'd or flushed). Note that in order to find I's current
	// generation number you must lock I, excluding concurrent syncs or flushes.
	//
	// INVARIANT: For each k/v, v.in.Name() == k
	// INVARIANT: For each value v, v.gen == inode.ImplicitDirGen => implicitDirs
	// INVARIANT: For each value v, inodes[v.in.ID()] == v.in
	// INVARIANT: For each value v, v.gen <= v.in.SourceGeneration()
	//
	// GUARDED_BY(mu)
	inodeIndex map[string]cachedGen

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

type cachedGen struct {
	in  inode.Inode
	gen int64
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
	// inodeIndex
	//////////////////////////////////

	// INVARIANT: For each k/v, v.in.Name() == k
	for k, v := range fs.inodeIndex {
		if !(v.in.Name() == k) {
			panic(fmt.Sprintf(
				"Unexpected name: \"%s\" vs. \"%s\"",
				v.in.Name(),
				k))
		}
	}

	// INVARIANT: For each value v, v.gen == inode.ImplicitDirGen => implicitDirs
	for _, v := range fs.inodeIndex {
		if v.gen == inode.ImplicitDirGen && !fs.implicitDirs {
			panic("Unexpected implicit directory")
		}
	}

	// INVARIANT: For each value v, inodes[v.in.ID()] == v.in
	for _, v := range fs.inodeIndex {
		if fs.inodes[v.in.ID()] != v.in {
			panic(fmt.Sprintf(
				"Mismatch for ID %v: %p %p",
				v.in.ID(),
				fs.inodes[v.in.ID()],
				v.in))
		}
	}

	// INVARIANT: For each value v, v.gen <= v.in.SourceGeneration()
	for _, v := range fs.inodeIndex {
		if !(v.gen <= v.in.SourceGeneration()) {
			panic(fmt.Sprintf(
				"Generation weirdness: %v vs. %v",
				v.gen,
				v.in.SourceGeneration()))
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
// LOCKS_REQUIRED(fs.mu)
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
		in = inode.NewDirInode(fs.bucket, id, o, fs.implicitDirs)
	} else {
		in = inode.NewFileInode(fs.clock, fs.bucket, id, o)
	}

	// Place it in our map of IDs to inodes.
	fs.inodes[in.ID()] = in

	return
}

// Attempt to find an inode for the given object record. Return nil, or the
// inode with the lock held. Release the file system lock.
//
// There are four possibilities:
//
//  *  We have no existing inode for the object's name. In this case, create an
//     inode, place it in the index, and return it.
//
//  *  We have an existing inode for the object's name, but its generation
//     number is less than that of the record. The inode is stale. Create a new
//     inode, place it in the index, and return it.
//
//     Special case: we always treat implicit directories as authoritative,
//     even though they would otherwise appear to be stale when an explicit
//     placeholder object has once been seen (since the implicit generation is
//     negative). This saves from an infinite loop when a placeholder object is
//     deleted but the directory still implicitly exists.
//
//  *  We have an existing inode for the object's name, and its current
//     generation number matches the record. Return it.
//
//  *  We have an existing inode for the object's name, and its current
//     generation number exceeds that of the record. Return nil, because the
//     record is stale.
//
// In other words, if this method returns nil, the caller should obtain a fresh
// record and try again. If the method returns non-nil, the inode is locked.
// Either way, the file system is unlocked.
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

	// Retry loop for the stale index entry case below.
	for {
		// Look for the current index entry.
		cg, ok := fs.inodeIndex[o.Name]
		existingInode := cg.in

		// If we have no existing record for this name, mint an inode and return it.
		if !ok {
			in = fs.mintInode(o)
			fs.inodeIndex[in.Name()] = cachedGen{in, in.SourceGeneration()}

			fs.mu.Unlock()
			in.Lock()

			return
		}

		// Otherwise, we will probably need to acquire the inode lock below. Drop the
		// file system lock now.
		fs.mu.Unlock()

		// Since we never re-use generations, if the cached generation is equal to
		// the record's generation, we know we've found our inode.
		if o.Generation == cg.gen {
			in = existingInode
			in.Lock()
			return
		}

		// If the cached generation is newer than our source generation, we know we
		// are stale.
		if o.Generation < cg.gen {
			return
		}

		// Otherwise it appears we are newer than the inode. Lock the inode so we can
		// attempt to prove it.
		existingInode.Lock()

		// Again, are we exactly right or stale?
		if o.Generation == existingInode.SourceGeneration() {
			in = existingInode
			return
		}

		if o.Generation < existingInode.SourceGeneration() {
			return
		}

		// We've observed that the record is newer than the existing inode, while
		// holding the inode lock, excluding concurrent actions by the inode (in
		// particular concurrent calls to Sync). This means we've proven that the
		// record cannot have been caused by the inode's actions, and therefore it is
		// not the inode we want.
		//
		// Re-acquire the file system lock. If nobody has updated the index entry in
		// the meantime, replace the entry, mint an inode, and return it. (There is
		// no ABA problem here because the entry's generation is strictly
		// increasing.) If it has been changed in the meantime, it's possible that
		// there's a new inode we have to contend with. Start over.
		//
		// TODO(jacobsa): There probably lurk implicit directory problems here!
		panic("TODO")
	}
}

// Given a function that returns a "fresh" object record, implement the calling
// loop documented for lookUpOrCreateInodeIfNotStale. Call the function once to
// begin with, and again each time it returns a stale record.
//
// Return ENOENT if the function ever returns a nil record. Never return a nil
// inode with a nil error.
//
// LOCKS_REQUIRED(fs.mu)
// LOCK_FUNCTION(in)
func (fs *fileSystem) lookUpOrCreateInode(
	f func() (*gcs.Object, error)) (in inode.Inode, err error) {
	const maxTries = 3
	for n := 0; n < maxTries; n++ {
		var o *gcs.Object

		// Create a record.
		o, err = f()
		if err != nil {
			return
		}

		if o == nil {
			err = fuse.ENOENT
			return
		}

		// Attempt to create the inode. Return if successful.
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
// LOCKS_REQUIRED(fs.mu)
// LOCKS_REQUIRED(f.mu)
func (fs *fileSystem) syncFile(
	ctx context.Context,
	f *inode.FileInode) (err error) {

	// Sync the inode.
	err = f.Sync(ctx)
	if err != nil {
		err = fmt.Errorf("FileInode.Sync: %v", err)
		return
	}

	// Update the index.
	fs.inodeIndex[f.Name()] = cachedGen{f, f.SourceGeneration()}

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

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Find the parent directory in question.
	parent := fs.inodes[op.Parent].(*inode.DirInode)

	// Set up a function taht will find a record for the child with the given
	// name, or nil if none.
	f := func() (o *gcs.Object, err error) {
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

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Find the inode.
	in := fs.inodes[op.Inode]

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

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Find the inode.
	in := fs.inodes[op.Inode]

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

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Find the inode.
	in := fs.inodes[op.Inode]

	in.Lock()
	defer in.Unlock()

	// Decrement the lookup count. If destroyed, we should remove it from our
	// maps.
	name := in.Name()
	if in.DecrementLookupCount(op.N) {
		delete(fs.inodes, op.Inode)

		// Is this the latest entry for the name?
		if cg := fs.inodeIndex[name]; cg.in == in {
			delete(fs.inodeIndex, name)
		}
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) MkDir(
	op *fuseops.MkDirOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Find the parent.
	parent := fs.inodes[op.Parent]

	parent.Lock()
	defer parent.Unlock()

	// Create an empty backing object for the child, failing if it already
	// exists.
	var precond int64
	createReq := &gcs.CreateObjectRequest{
		Name:                   path.Join(parent.Name(), op.Name) + "/",
		Contents:               strings.NewReader(""),
		GenerationPrecondition: &precond,
	}

	o, err := fs.bucket.CreateObject(op.Context(), createReq)
	if err != nil {
		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	// Attempt to create a child inode using the object we created. If we fail to
	// do so, it means someone beat us to the punch with a newer generation
	// (unlikely, so we're probably okay with failing here).
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

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Find the parent.
	parent := fs.inodes[op.Parent]

	parent.Lock()
	defer parent.Unlock()

	// Create an empty backing object for the child, failing if it already
	// exists.
	var precond int64
	createReq := &gcs.CreateObjectRequest{
		Name:                   path.Join(parent.Name(), op.Name),
		Contents:               strings.NewReader(""),
		GenerationPrecondition: &precond,
	}

	o, err := fs.bucket.CreateObject(op.Context(), createReq)
	if err != nil {
		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	// Attempt to create a child inode using the object we created. If we fail to
	// do so, it means someone beat us to the punch with a newer generation
	// (unlikely, so we're probably okay with failing here).
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

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Find the parent. We assume that it exists because otherwise the kernel has
	// done something mildly concerning.
	parent := fs.inodes[op.Parent]
	parent.Lock()
	defer parent.Unlock()

	// Delete the backing object. Unfortunately we have no way to precondition
	// this on the directory being empty.
	err = fs.bucket.DeleteObject(
		op.Context(),
		path.Join(parent.Name(), op.Name)+"/")

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) Unlink(
	op *fuseops.UnlinkOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Find the parent.
	parent := fs.inodes[op.Parent]

	parent.Lock()
	defer parent.Unlock()

	// Delete the backing object.
	err = fs.bucket.DeleteObject(op.Context(), path.Join(parent.Name(), op.Name))

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
	in.Lock()
	defer in.Unlock()

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
	dh := fs.handles[op.Handle].(*dirHandle)
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

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Find the inode.
	in := fs.inodes[op.Inode].(*inode.FileInode)
	in.Lock()
	defer in.Unlock()

	// Serve the request.
	err = in.Read(op)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) WriteFile(
	op *fuseops.WriteFileOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Find the inode.
	in := fs.inodes[op.Inode].(*inode.FileInode)
	in.Lock()
	defer in.Unlock()

	// Serve the request.
	err = in.Write(op)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) SyncFile(
	op *fuseops.SyncFileOp) {
	var err error
	defer fuseutil.RespondToOp(op, &err)

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Find the inode.
	in := fs.inodes[op.Inode].(*inode.FileInode)
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

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Find the inode.
	in := fs.inodes[op.Inode].(*inode.FileInode)
	in.Lock()
	defer in.Unlock()

	// Sync it.
	err = fs.syncFile(op.Context(), in)

	return
}
