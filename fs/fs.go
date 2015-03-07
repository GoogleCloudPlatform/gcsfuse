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

// Create a fuse file system whose root directory is the root of the supplied
// bucket. The supplied clock will be used for cache invalidation, modification
// times, etc.
func NewFuseFS(
	clock timeutil.Clock,
	bucket gcs.Bucket) (ffs fuse.FileSystem, err error) {
	// Set up the basic struct.
	fs := &fileSystem{
		clock:  clock,
		bucket: bucket,
		inodes: make([]interface{}, fuse.RootInodeID+1),
	}

	// Set up the root inode.
	fs.inodes[fuse.RootInodeID] = inode.NewDirInode("")

	// Set up invariant checking.
	fs.mu = syncutil.NewInvariantMutex(fs.checkInvariants)

	ffs = fs
	return
}

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

	// When acquiring this lock, the caller must hold no inode locks.
	mu syncutil.InvariantMutex

	// The collection of live inodes, indexed by ID. IDs of free inodes that may
	// be re-used have nil entries. No ID less than fuse.RootInodeID is ever used.
	//
	// INVARIANT: All elements are nil or of type *inode.(Dir|File)Inode
	// INVARIANT: len(inodes) > fuse.RootInodeID
	// INVARIANT: For all i < fuse.RootInodeID, inodes[i] == nil
	// INVARIANT: inodes[fuse.RootInodeID] != nil
	// INVARIANT: inodes[fuse.RootInodeID] is of type *inode.DirInode
	//
	// GUARDED_BY(mu)
	inodes []interface{}

	// A list of inode IDs within inodes available for reuse, not including the
	// reserved IDs less than fuse.RootInodeID.
	//
	// INVARIANT: This is all and only indices i of 'inodes' such that i >
	// fuse.RootInodeID and inodes[i] == nil
	//
	// GUARDED_BY(mu)
	freeInodeIDs []fuse.InodeID
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (fs *fileSystem) checkInvariants() {
	// Check reserved inodes.
	for i := 0; i < fuse.RootInodeID; i++ {
		if fs.inodes[i] != nil {
			panic(fmt.Sprintf("Non-nil inode for ID: %v", i))
		}
	}

	// Check the root inode.
	_ = fs.inodes[fuse.RootInodeID].(*inode.DirInode)

	// Check the type of each inode. While we're at it, build our own list of
	// free IDs.
	freeIDsEncountered := make(map[fuse.InodeID]struct{})
	for i := fuse.RootInodeID + 1; i < len(fs.inodes); i++ {
		in := fs.inodes[i]
		switch in.(type) {
		case *inode.DirInode:
		case *inode.FileInode:

		case nil:
			freeIDsEncountered[fuse.InodeID(i)] = struct{}{}

		default:
			panic(fmt.Sprintf("Unexpected inode type: %v", reflect.TypeOf(in)))
		}
	}

	// Check fs.freeInodeIDs.
	if len(fs.freeInodeIDs) != len(freeIDsEncountered) {
		panic(
			fmt.Sprintf(
				"Length mismatch: %v vs. %v",
				len(fs.freeInodeIDs),
				len(freeIDsEncountered)))
	}

	for _, id := range fs.freeInodeIDs {
		if _, ok := freeIDsEncountered[id]; !ok {
			panic(fmt.Sprintf("Unexected free inode ID: %v", id))
		}
	}
}

// Find the given inode and return it with its lock held for reading. Panic if
// it doesn't exist or is the wrong type.
//
// SHARED_LOCKS_REQUIRED(fs.mu)
// SHARED_LOCK_FUNCTION(inode.mu)
func (fs *fileSystem) getDirForReadingOrDie(
	id fuse.InodeID) (in *inode.DirInode) {
	in = fs.inodes[id].(*inode.DirInode)
	in.Mu.RLock()
	return
}

////////////////////////////////////////////////////////////////////////
// fuse.FileSystem methods
////////////////////////////////////////////////////////////////////////

func (fs *fileSystem) Init(
	ctx context.Context,
	req *fuse.InitRequest) (resp *fuse.InitResponse, err error) {
	// Nothing interesting to do.
	resp = &fuse.InitResponse{}
	return
}

func (fs *fileSystem) OpenDir(
	ctx context.Context,
	req *fuse.OpenDirRequest) (resp *fuse.OpenDirResponse, err error) {
	resp = &fuse.OpenDirResponse{}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Make sure the inode still exists and is a directory. If not, something has
	// screwed up because the VFS layer shouldn't have let us forget the inode
	// before opening it.
	in := fs.getDirForReadingOrDie(req.Inode)
	defer in.Mu.RUnlock()

	return
}
