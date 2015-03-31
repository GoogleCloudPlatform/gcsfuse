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
		inodeIndex:   make(map[nameAndGen]inode.Inode),
		handles:      make(map[fuseops.HandleID]interface{}),
	}

	// Set up the root inode.
	root := inode.NewRootInode(cfg.Bucket, fs.implicitDirs)
	fs.inodes[fuseops.RootInodeID] = root
	fs.inodeIndex[nameAndGen{root.Name(), root.SourceGeneration()}] = root

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

	// An index of all inodes by (Name(), SourceGeneration()) pairs.
	//
	// INVARIANT: For each key k, inodeIndex[k].Name() == k.name
	// INVARIANT: For each key k, inodeIndex[k].SourceGeneration() == k.gen
	// INVARIANT: For each key k, k.gen == ImplicitDirGen only if implicitDirs
	// INVARIANT: The values are all and only the values of the inodes map
	//
	// GUARDED_BY(mu)
	inodeIndex map[nameAndGen]inode.Inode

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

	// INVARIANT: For each key k, inodeIndex[k].Name() == k.name
	for k, in := range fs.inodeIndex {
		if in.Name() != k.name {
			panic(fmt.Sprintf("Name mismatch: %v vs. %v", in.Name(), k.name))
		}
	}

	// INVARIANT: For each key k, inodeIndex[k].SourceGeneration() == k.gen
	for k, in := range fs.inodeIndex {
		if in.SourceGeneration() != k.gen {
			panic(fmt.Sprintf(
				"Generation mismatch: %v vs. %v",
				in.SourceGeneration(),
				k.gen))
		}
	}

	// INVARIANT: For each key k, k.gen == ImplicitDirGen only if implicitDirs
	for k, in := range fs.inodeIndex {
		if k.gen == inode.ImplicitDirGen && !fs.implicitDirs {
			panic(fmt.Sprintf("Unexpected implicit generation: %s", in.Name()))
		}
	}

	// INVARIANT: The values are all and only the values of the inodes map
	for id, in := range fs.inodes {
		if in != fs.inodes[id] {
			panic("inodeIndex is not a subset of inodes")
		}
	}

	if len(fs.inodeIndex) != len(fs.inodes) {
		panic("inodeIndex values are not the same set as inodes")
	}

	// INVARIANT: All values are of type *dirHandle
	for _, h := range fs.handles {
		_ = h.(*dirHandle)
	}

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

// Find an inode for the given object record. Create one if there isn't already
// one available. Return the inode locked.
//
// LOCKS_REQUIRED(fs.mu)
// LOCK_FUNCTION(in)
func (fs *fileSystem) lookUpOrCreateInode(
	ctx context.Context,
	o *gcs.Object) (in inode.Inode, err error) {
	// Make sure to return the inode locked.
	defer func() {
		if in != nil {
			in.Lock()
		}
	}()

	// Build the index key.
	nandg := nameAndGen{
		name: o.Name,
		gen:  o.Generation,
	}

	// Do we already have an inode for this (name, generation) pair? If so,
	// increase its lookup count and return it.
	if in = fs.inodeIndex[nandg]; in != nil {
		in.IncrementLookupCount()
		return
	}

	// Mint an ID.
	id := fs.nextInodeID
	fs.nextInodeID++

	// Create an inode.
	if isDirName(o.Name) {
		in = inode.NewDirInode(fs.bucket, id, o, fs.implicitDirs)
	} else {
		in, err = inode.NewFileInode(fs.clock, fs.bucket, id, o)
		if err != nil {
			err = fmt.Errorf("NewFileInode: %v", err)
			return
		}
	}

	// Index the inode.
	fs.inodes[id] = in
	fs.inodeIndex[nandg] = in

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
		return
	}

	// Find or mint an inode.
	var in inode.Inode
	if in, err = fs.lookUpOrCreateInode(op.Context(), o); err != nil {
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
	child, err := fs.lookUpOrCreateInode(op.Context(), o)
	if err != nil {
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

	// Create and index a child inode.
	child, err := fs.lookUpOrCreateInode(op.Context(), o)
	if err != nil {
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

	fs.handles[handleID] = newDirHandle(in)
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
