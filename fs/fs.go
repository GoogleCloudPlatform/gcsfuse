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
	fs.inodes[fuseops.RootInodeID] = root
	fs.inodeIndex[root.Name()] = cachedGen{root, root.SourceGeneration()}

	// Set up invariant checking.
	fs.mu = syncutil.NewInvariantMutex(fs.checkInvariants)

	server = fuseutil.NewFileSystemServer(fs)
	return
}

////////////////////////////////////////////////////////////////////////
// fileSystem type
////////////////////////////////////////////////////////////////////////

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

	// When acquiring this lock, the caller must hold no inode or dir handle
	// locks.
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

type nameAndGen struct {
	name string
	gen  int64
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
			panic(fmt.Sprintf("Mismatch for ID %v", v.in.ID()))
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

// Implementation detail of lookUpOrCreateInode; do not use outside of that
// function.
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

	return
}

// Attempt to find an inode for the given object record.
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
//  *  We have an existing inode for the object's name, and its current
//     generation number matches the record. Return it, locked.
//
//  *  We have an existing inode for the object's name, and its current
//     generation number exceeds that of the record. Return nil, because the
//     record is stale.
//
// In other words, if this method returns nil, the caller should obtain a fresh
// record and try again. If the method returns non-nil, the inode is locked.
//
// TODO(jacobsa): This logic should result in an infinite loop when we unlink a
// directory with a placeholder object, transitioning to an implicit directory.
// Make sure there is a test that shows this, then switch to math.MaxInt64 for
// placeholder directory generation numbers. I believe this should resolve the
// issue: placeholder inodes will be preferred over explicit ones, but so way.
// Name resolution will still check for the "existence" of the placeholder
// directories and return ENOENT if they no longer exist, and the kernel will
// eventually forget them.
//
// Oh shit, but this will just cause the issue in the other direction: when a
// directory transitions from implicit to explicit, we will loop forever trying
// to get an "up to date" record that beats the sticky implicit one. I think we
// will need to special case implicit directories, sigh. Perhaps never require
// beating an implicit directory, instead just returning its inode when
// present.
//
// TODO(jacobsa): We will need to replace this primitive if we want to
// parallelize with long-running operations holding the inode lock (issue #23).
// Instead we'll want to return the cachedGen record when present, without
// locking the inode, and let the caller deal with waiting on the lock and then
// going around if they've grabbed the wrong inode, replacing if they observe
// they've beaten out the inode once they have its lock.
//
// LOCKS_REQUIRED(fs.mu)
// LOCK_FUNCTION(in)
func (fs *fileSystem) lookUpOrCreateInode(
	o *gcs.Object) (in inode.Inode) {
	cg, ok := fs.inodeIndex[o.Name]
	existingInode := cg.in

	// If we have no existing record for this name, mint an inode and return it.
	if !ok {
		in = fs.mintInode(o)
		in.Lock()

		fs.inodeIndex[in.Name()] = cachedGen{in, in.SourceGeneration()}
		return
	}

	// Otherwise, we need the lock for the existing inode below.
	existingInode.Lock()

	shouldUnlock := true
	defer func() {
		if shouldUnlock {
			existingInode.Unlock()
		}
	}()

	// Have we beaten out the existing inode? If so, mint a new inode and replace
	// the index entry.
	if existingInode.SourceGeneration() < o.Generation {
		in = fs.mintInode(o)
		in.Lock()

		fs.inodeIndex[in.Name()] = cachedGen{in, in.SourceGeneration()}
		return
	}

	// If the generations match, we've found our inode.
	if existingInode.SourceGeneration() == o.Generation {
		in = existingInode
		shouldUnlock = false
		return
	}

	// Otherwise, the object record is stale. Return nil.
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
	oldGen := f.SourceGeneration()

	// Sync the inode.
	err = f.Sync(ctx)
	if err != nil {
		err = fmt.Errorf("FileInode.Sync: %v", err)
		return
	}

	// Update the index if necessary.
	newGen := f.SourceGeneration()
	if oldGen != newGen {
		delete(fs.inodeIndex, nameAndGen{f.Name(), oldGen})
		fs.inodeIndex[nameAndGen{f.Name(), newGen}] = f
	}

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

	// Find a record for the child with the given name.
	o, err := parent.LookUpChild(op.Context(), op.Name)
	if err != nil {
		err = fmt.Errorf("LookUpChild: %v", err)
		return
	}

	if o == nil {
		err = fuse.ENOENT
		return
	}

	// Find or mint an inode.
	in := fs.lookUpOrCreateInode(o)
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

	// Decrement the lookup count. If destroyed, we should remove it from the
	// index.
	nandg := nameAndGen{
		name: in.Name(),
		gen:  in.SourceGeneration(),
	}

	if in.DecrementLookupCount(op.N) {
		delete(fs.inodes, op.Inode)
		delete(fs.inodeIndex, nandg)
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

	// Create and index a child inode.
	child := fs.lookUpOrCreateInode(o)
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

	// Create and index a child inode.
	child := fs.lookUpOrCreateInode(o)
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
