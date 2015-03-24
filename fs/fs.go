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
	"google.golang.org/cloud/storage"
)

type fileSystem struct {
	fuseutil.NotImplementedFileSystem

	/////////////////////////
	// Dependencies
	/////////////////////////

	clock  timeutil.Clock
	bucket gcs.Bucket

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

	// The collection of live inodes, keyed by inode ID. No ID less than
	// fuseops.RootInodeID is ever used.
	//
	// TODO(jacobsa): Implement ForgetInode support in the fuse package, then
	// implement the method here and clean up these maps.
	//
	// INVARIANT: All values are of type *inode.DirInode or *inode.FileInode
	// INVARIANT: For all keys k, k >= fuseops.RootInodeID
	// INVARIANT: For all keys k, inodes[k].ID() == k
	// INVARIANT: inodes[fuseops.RootInodeID] is of type *inode.DirInode
	//
	// GUARDED_BY(mu)
	inodes map[fuseops.InodeID]inode.Inode

	// The next inode ID to hand out. We assume that this will never overflow,
	// since even if we were handing out inode IDs at 4 GHz, it would still take
	// over a century to do so.
	//
	// INVARIANT: For all keys k in inodes, k < nextInodeID
	//
	// GUARDED_BY(mu)
	nextInodeID fuseops.InodeID

	// An index of all directory inodes by Name().
	//
	// INVARIANT: For each key k, isDirName(k)
	// INVARIANT: For each key k, dirIndex[k].Name() == k
	// INVARIANT: The values are all and only the values of the inodes map of
	// type *inode.DirInode.
	//
	// GUARDED_BY(mu)
	dirIndex map[string]*inode.DirInode

	// An index of all file inodes by (Name(), SourceGeneration()) pairs.
	//
	// INVARIANT: For each key k, !isDirName(k)
	// INVARIANT: For each key k, fileIndex[k].Name() == k.name
	// INVARIANT: For each key k, fileIndex[k].SourceGeneration() == k.gen
	// INVARIANT: The values are all and only the values of the inodes map of
	// type *inode.FileInode.
	//
	// GUARDED_BY(mu)
	fileIndex map[nameAndGen]*inode.FileInode

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

// Create a fuse file system server whose root directory is the root of the
// supplied bucket. The supplied clock will be used for cache invalidation,
// modification times, etc.
func NewServer(
	clock timeutil.Clock,
	bucket gcs.Bucket) (server fuse.Server, err error) {
	// Get ownership information.
	uid, gid, err := getUser()
	if err != nil {
		return
	}

	// Set up the basic struct.
	fs := &fileSystem{
		clock:       clock,
		bucket:      bucket,
		uid:         uid,
		gid:         gid,
		inodes:      make(map[fuseops.InodeID]inode.Inode),
		nextInodeID: fuseops.RootInodeID + 1,
		dirIndex:    make(map[string]*inode.DirInode),
		fileIndex:   make(map[nameAndGen]*inode.FileInode),
		handles:     make(map[fuseops.HandleID]interface{}),
	}

	// Set up the root inode.
	root := inode.NewDirInode(bucket, fuseops.RootInodeID, "")
	fs.inodes[fuseops.RootInodeID] = root
	fs.dirIndex[""] = root

	// Set up invariant checking.
	fs.mu = syncutil.NewInvariantMutex(fs.checkInvariants)

	server = fuseutil.NewFileSystemServer(fs)
	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func isDirName(name string) bool {
	return name == "" || name[len(name)-1] == '/'
}

func (fs *fileSystem) checkInvariants() {
	// Check inode keys.
	for id, _ := range fs.inodes {
		if id < fuseops.RootInodeID || id >= fs.nextInodeID {
			panic(fmt.Sprintf("Illegal inode ID: %v", id))
		}
	}

	// Check the root inode.
	_ = fs.inodes[fuseops.RootInodeID].(*inode.DirInode)

	// Check each inode, and the indexes over them. Keep a count of each type
	// seen.
	dirsSeen := 0
	filesSeen := 0
	for id, in := range fs.inodes {
		// Check the ID.
		if in.ID() != id {
			panic(fmt.Sprintf("ID mismatch: %v vs. %v", in.ID(), id))
		}

		// Check type-specific stuff.
		switch typed := in.(type) {
		case *inode.DirInode:
			dirsSeen++

			if !isDirName(typed.Name()) {
				panic(fmt.Sprintf("Unexpected directory name: %s", typed.Name()))
			}

			if fs.dirIndex[typed.Name()] != typed {
				panic(fmt.Sprintf("dirIndex mismatch: %s", typed.Name()))
			}

		case *inode.FileInode:
			filesSeen++

			if isDirName(typed.Name()) {
				panic(fmt.Sprintf("Unexpected file name: %s", typed.Name()))
			}

			nandg := nameAndGen{typed.Name(), typed.SourceGeneration()}
			if fs.fileIndex[nandg] != typed {
				panic(
					fmt.Sprintf(
						"fileIndex mismatch: %s, %v",
						typed.Name(),
						typed.SourceGeneration()))
			}

		default:
			panic(fmt.Sprintf("Unexpected inode type: %v", reflect.TypeOf(in)))
		}
	}

	// Make sure that the indexes are exhaustive.
	if len(fs.dirIndex) != dirsSeen {
		panic(
			fmt.Sprintf(
				"dirIndex length mismatch: %v vs. %v",
				len(fs.dirIndex),
				dirsSeen))
	}

	if len(fs.fileIndex) != filesSeen {
		panic(
			fmt.Sprintf(
				"fileIndex length mismatch: %v vs. %v",
				len(fs.fileIndex),
				dirsSeen))
	}

	// Check handles.
	for id, h := range fs.handles {
		if id >= fs.nextHandleID {
			panic(fmt.Sprintf("Illegal handle ID: %v", id))
		}

		_ = h.(*dirHandle)
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

// Find a directory inode for the given object record. Create one if there
// isn't already one available.
//
// LOCKS_REQUIRED(fs.mu)
func (fs *fileSystem) lookUpOrCreateDirInode(
	ctx context.Context,
	o *storage.Object) (in *inode.DirInode, err error) {
	// Do we already have an inode for this name?
	if in = fs.dirIndex[o.Name]; in != nil {
		return
	}

	// Mint an ID.
	id := fs.nextInodeID
	fs.nextInodeID++

	// Create and index an inode.
	in = inode.NewDirInode(fs.bucket, id, o.Name)
	fs.inodes[id] = in
	fs.dirIndex[in.Name()] = in

	return
}

// Find a file inode for the given object record. Create one if there isn't
// already one available.
//
// LOCKS_REQUIRED(fs.mu)
func (fs *fileSystem) lookUpOrCreateFileInode(
	ctx context.Context,
	o *storage.Object) (in *inode.FileInode, err error) {
	nandg := nameAndGen{
		name: o.Name,
		gen:  o.Generation,
	}

	// Do we already have an inode for this (name, generation) pair?
	if in = fs.fileIndex[nandg]; in != nil {
		return
	}

	// Mint an ID.
	id := fs.nextInodeID
	fs.nextInodeID++

	// Create and index an inode.
	if in, err = inode.NewFileInode(fs.clock, fs.bucket, id, o); err != nil {
		err = fmt.Errorf("NewFileInode: %v", err)
		return
	}

	fs.inodes[id] = in
	fs.fileIndex[nandg] = in

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

	// Is the child a directory or a file?
	var in inode.Inode
	if isDirName(o.Name) {
		in, err = fs.lookUpOrCreateDirInode(op.Context(), o)
	} else {
		in, err = fs.lookUpOrCreateFileInode(op.Context(), o)
	}

	if err != nil {
		return
	}

	in.Lock()
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
		Attrs: storage.ObjectAttrs{
			Name: path.Join(parent.Name(), op.Name) + "/",
		},
		Contents:               strings.NewReader(""),
		GenerationPrecondition: &precond,
	}

	o, err := fs.bucket.CreateObject(op.Context(), createReq)
	if err != nil {
		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	// Create a child inode.
	id := fs.nextInodeID
	fs.nextInodeID++

	child := inode.NewDirInode(fs.bucket, id, o.Name)
	child.Lock()
	defer child.Unlock()

	// Index the child inode.
	fs.inodes[child.ID()] = child
	fs.dirIndex[child.Name()] = child

	// Fill out the response.
	op.Entry.Child = child.ID()
	if op.Entry.Attributes, err = fs.getAttributes(op.Context(), child); err != nil {
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
	//
	// TODO(jacobsa): Make sure that there is a test that ensures the object
	// exists once the file is opened with O_CREAT but before it has been closed
	// (but not necessarily that its contents are there).
	var precond int64
	createReq := &gcs.CreateObjectRequest{
		Attrs: storage.ObjectAttrs{
			Name: path.Join(parent.Name(), op.Name),
		},
		Contents:               strings.NewReader(""),
		GenerationPrecondition: &precond,
	}

	o, err := fs.bucket.CreateObject(op.Context(), createReq)
	if err != nil {
		// TODO(jacobsa): Add a test that fails, then map gcs.PreconditionError to
		// EEXISTS.
		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	// Create a child inode.
	childID := fs.nextInodeID
	fs.nextInodeID++

	child, err := inode.NewFileInode(fs.clock, fs.bucket, childID, o)
	if err != nil {
		err = fmt.Errorf("NewFileInode: %v", err)
		return
	}

	child.Lock()
	defer child.Unlock()

	// Index the child inode.
	fs.inodes[childID] = child
	fs.fileIndex[nameAndGen{child.Name(), child.SourceGeneration()}] = child

	// Fill out the response.
	op.Entry.Child = childID
	if op.Entry.Attributes, err = fs.getAttributes(op.Context(), child); err != nil {
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
	err = fs.bucket.DeleteObject(op.Context(), path.Join(parent.Name(), op.Name)+"/")

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
	//
	// TODO(jacobsa): Once we figure out the object path, we don't need to
	// continue to hold this or the file system lock. Ditto with many other
	// methods.
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
	op, err = dh.ReadDir(op.Context(), op)

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
//
// TODO(jacobsa): Make sure there is a test for fsync and close behavior.
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
