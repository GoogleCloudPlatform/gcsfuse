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
	"reflect"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"github.com/jacobsa/gcsfuse/fs/inode"
	"github.com/jacobsa/gcsfuse/timeutil"
	"golang.org/x/net/context"
)

type fileSystem struct {
	fuseutil.NotImplementedFileSystem

	/////////////////////////
	// Dependencies
	/////////////////////////

	clock  timeutil.Clock
	bucket gcs.Bucket

	/////////////////////////
	// Mutable state
	/////////////////////////

	// When acquiring this lock, the caller must hold no inode or dir handle
	// locks.
	mu syncutil.InvariantMutex

	// The user and group owning everything in the file system.
	//
	// GUARDED_BY(Mu)
	uid uint32
	gid uint32

	// The collection of live inodes, keyed by inode ID. No ID less than
	// fuse.RootInodeID is ever used.
	//
	// INVARIANT: All values are of type *inode.DirInode or *inode.FileInode
	// INVARIANT: For all keys k, k >= fuse.RootInodeID
	// INVARIANT: inodes[fuse.RootInodeID] is of type *inode.DirInode
	//
	// GUARDED_BY(mu)
	inodes map[fuse.InodeID]interface{}

	// The next inode ID to hand out. We assume that this will never overflow,
	// since even if we were handing out inode IDs at 4 GHz, it would still take
	// over a century to do so.
	//
	// INVARIANT: For all keys k in inodes, k < nextInodeID
	//
	// GUARDED_BY(mu)
	nextInodeID fuse.InodeID

	// An index of all directory inodes by Name().
	//
	// INVARIANT: For each key k, dirNameIndex[k].Name() == k
	//
	// INVARIANT: The values are all and only the values of the inodes map of
	// type *inode.DirHandle.
	//
	// GUARDED_BY(mu)
	dirNameIndex map[string]*inode.DirInode

	// The collection of live handles, keyed by handle ID.
	//
	// INVARIANT: All values are of type *dirHandle
	//
	// GUARDED_BY(mu)
	handles map[fuse.HandleID]interface{}

	// The next handle ID to hand out. We assume that this will never overflow.
	//
	// INVARIANT: For all keys k in handles, k < nextHandleID
	//
	// GUARDED_BY(mu)
	nextHandleID fuse.HandleID
}

// Create a fuse file system whose root directory is the root of the supplied
// bucket. The supplied clock will be used for cache invalidation, modification
// times, etc.
func NewFileSystem(
	clock timeutil.Clock,
	bucket gcs.Bucket) (ffs fuse.FileSystem, err error) {
	// Set up the basic struct.
	fs := &fileSystem{
		clock:        clock,
		bucket:       bucket,
		inodes:       make(map[fuse.InodeID]interface{}),
		nextInodeID:  fuse.RootInodeID + 1,
		dirNameIndex: make(map[string]*inode.DirInode),
		handles:      make(map[fuse.HandleID]interface{}),
	}

	// Set up the root inode.
	root := inode.NewDirInode(bucket, "")
	fs.inodes[fuse.RootInodeID] = root
	fs.dirNameIndex[""] = root

	// Set up invariant checking.
	fs.mu = syncutil.NewInvariantMutex(fs.checkInvariants)

	ffs = fs
	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (fs *fileSystem) checkInvariants() {
	// Check inode keys.
	for id, _ := range fs.inodes {
		if id < fuse.RootInodeID || id >= fs.nextInodeID {
			panic(fmt.Sprintf("Illegal inode ID: %v", id))
		}
	}

	// Check the root inode.
	_ = fs.inodes[fuse.RootInodeID].(*inode.DirInode)

	// Check each inode, and the indexes over them. Keep a count of each type
	// seen.
	dirsSeen := 0
	filesSeen := 0
	for _, in := range fs.inodes {
		switch typed := in.(type) {
		case *inode.DirInode:
			dirsSeen++
			if fs.dirNameIndex[typed.Name()] != typed {
				panic(fmt.Sprintf("dirNameIndex mismatch: %s", typed.Name()))
			}

		case *inode.FileInode:
			filesSeen++

		default:
			panic(fmt.Sprintf("Unexpected inode type: %v", reflect.TypeOf(in)))
		}
	}

	// Make sure that the indexes are exhaustive.
	if len(fs.dirNameIndex) != dirsSeen {
		panic(
			fmt.Sprintf(
				"dirNameIndex length mismatch: %v vs. %v",
				len(fs.dirNameIndex),
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

// Find the given inode and return it with its lock held for reading. Panic if
// it doesn't exist or is the wrong type.
//
// SHARED_LOCKS_REQUIRED(fs.mu)
// SHARED_LOCK_FUNCTION(in.mu)
func (fs *fileSystem) getDirForReadingOrDie(
	id fuse.InodeID) (in *inode.DirInode) {
	in = fs.inodes[id].(*inode.DirInode)
	in.Mu.RLock()
	return
}

// Get attributes for the given directory inode, fixing up ownership information.
//
// SHARED_LOCKS_REQUIRED(fs.mu)
// SHARED_LOCK_FUNCTION(d.mu)
func (fs *fileSystem) getAttributes(
	ctx context.Context,
	d *inode.DirInode) (attrs fuse.InodeAttributes, err error) {
	attrs, err = d.Attributes(ctx)
	if err != nil {
		return
	}

	attrs.Uid = fs.uid
	attrs.Gid = fs.gid

	return
}

////////////////////////////////////////////////////////////////////////
// fuse.FileSystem methods
////////////////////////////////////////////////////////////////////////

func (fs *fileSystem) Init(
	ctx context.Context,
	req *fuse.InitRequest) (resp *fuse.InitResponse, err error) {
	resp = &fuse.InitResponse{}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Store the mounting user's info for later.
	fs.uid = req.Header.Uid
	fs.gid = req.Header.Gid

	return
}

func (fs *fileSystem) GetInodeAttributes(
	ctx context.Context,
	req *fuse.GetInodeAttributesRequest) (
	resp *fuse.GetInodeAttributesResponse, err error) {
	resp = &fuse.GetInodeAttributesResponse{}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Find the inode.
	in := fs.inodes[req.Inode]

	// Grab its attributes.
	switch typed := in.(type) {
	case *inode.DirInode:
		resp.Attributes, err = fs.getAttributes(ctx, typed)
		if err != nil {
			err = fmt.Errorf("DirInode.Attributes: %v", err)
			return
		}

	default:
		panic(
			fmt.Sprintf(
				"Unknown inode type for ID %v: %v",
				req.Inode,
				reflect.TypeOf(in)))
	}

	return
}

func (fs *fileSystem) OpenDir(
	ctx context.Context,
	req *fuse.OpenDirRequest) (resp *fuse.OpenDirResponse, err error) {
	resp = &fuse.OpenDirResponse{}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Make sure the inode still exists and is a directory. If not, something has
	// screwed up because the VFS layer shouldn't have let us forget the inode
	// before opening it.
	in := fs.getDirForReadingOrDie(req.Inode)
	defer in.Mu.RUnlock()

	// Allocate a handle.
	handleID := fs.nextHandleID
	fs.nextHandleID++

	fs.handles[handleID] = newDirHandle(in)
	resp.Handle = handleID

	return
}

func (fs *fileSystem) ReadDir(
	ctx context.Context,
	req *fuse.ReadDirRequest) (resp *fuse.ReadDirResponse, err error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Find the handle.
	dh := fs.handles[req.Handle].(*dirHandle)
	dh.Mu.Lock()
	defer dh.Mu.Unlock()

	// Serve the request.
	resp, err = dh.ReadDir(ctx, req)

	return
}

func (fs *fileSystem) ReleaseDirHandle(
	ctx context.Context,
	req *fuse.ReleaseDirHandleRequest) (
	resp *fuse.ReleaseDirHandleResponse, err error) {
	resp = &fuse.ReleaseDirHandleResponse{}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Sanity check that this handle exists and is of the correct type.
	_ = fs.handles[req.Handle].(*dirHandle)

	// Clear the entry from the map.
	delete(fs.handles, req.Handle)

	return
}
