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

package inode

import (
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

// TODO(jacobsa): Add a Destroy method here that calls ObjectProxy.Destroy, and
// make sure it's called when the inode is forgotten. Also, make sure package
// fuse has support for actually calling Forget.
type FileInode struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	bucket gcs.Bucket

	/////////////////////////
	// Constant data
	/////////////////////////

	id   fuse.InodeID
	name string

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A mutex that must be held when calling certain methods. See documentation
	// for each method.
	mu syncutil.InvariantMutex

	// A record for the object from which this inode was branched. The object's
	// generation is used as a precondition in object write requests.
	//
	// GUARDED_BY(mu)
	srcObject storage.Object
}

var _ Inode = &FileInode{}

// Create a file inode for the given object in GCS.
//
// REQUIRES: o != nil
// REQUIRES: len(o.Name) > 0
// REQUIRES: o.Name[len(o.Name)-1] != '/'
func NewFileInode(
	bucket gcs.Bucket,
	id fuse.InodeID,
	o *storage.Object) (f *FileInode) {
	// Set up the basic struct.
	f = &FileInode{
		bucket:    bucket,
		id:        id,
		name:      o.Name,
		srcObject: *o,
	}

	// Set up invariant checking.
	f.mu = syncutil.NewInvariantMutex(f.checkInvariants)

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (f *FileInode) checkInvariants() {
	if len(f.name) == 0 || f.name[len(f.name)-1] == '/' {
		panic("Illegal file name: " + f.name)
	}
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (f *FileInode) Lock() {
	f.mu.Lock()
}

func (f *FileInode) Unlock() {
	f.mu.Unlock()
}

func (f *FileInode) ID() fuse.InodeID {
	return f.id
}

func (f *FileInode) Name() string {
	return f.name
}

func (f *FileInode) Attributes(
	ctx context.Context) (attrs fuse.InodeAttributes, err error) {
	attrs = fuse.InodeAttributes{
		Size:  uint64(f.srcObject.Size),
		Mode:  0700,
		Mtime: f.srcObject.Updated,
	}

	return
}

// Return the generation number from which this inode was branched. This is
// used as a precondition in object write requests.
//
// TODO(jacobsa): Make sure to add a test for opening a file with O_CREAT then
// opening it again for reading, and sharing data across the two descriptors.
// This should fail if we have screwed up the fuse lookup process with regards
// to the zero generation.
//
// SHARED_LOCKS_REQUIRED(f.mu)
func (f *FileInode) SourceGeneration() int64 {
	return f.srcObject.Generation
}
