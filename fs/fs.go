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
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"github.com/jacobsa/gcsfuse/timeutil"
	"golang.org/x/net/context"
)

// Create a fuse file system whose root directory is the root of the supplied
// bucket. The supplied clock will be used for cache invalidation, modification
// times, etc.
func NewFuseFS(
	clock timeutil.Clock,
	bucket gcs.Bucket) (fs fuse.FileSystem, err error) {
	fs = &fileSystem{
		clock:  clock,
		bucket: bucket,
	}

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
	// INVARIANT: All elements are of type *inode.FileInode or *inode.DirInode
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
// fuse.FileSystem methods
////////////////////////////////////////////////////////////////////////

func (fs *fileSystem) Init(
	ctx context.Context,
	req *fuse.InitRequest) (resp *fuse.InitResponse, err error) {
	// Nothing interesting to do.
	resp = &fuse.InitResponse{}
	return
}
