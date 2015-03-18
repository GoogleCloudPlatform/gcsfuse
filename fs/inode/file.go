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
	"fmt"
	"io"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"github.com/jacobsa/gcsfuse/gcsproxy"
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

	id fuse.InodeID

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A mutex that must be held when calling certain methods. See documentation
	// for each method.
	mu syncutil.InvariantMutex

	// A proxy for the backing object in GCS.
	//
	// INVARIANT: proxy.CheckInvariants() does not panic
	//
	// GUARDED_BY(mu)
	proxy *gcsproxy.ObjectProxy
}

var _ Inode = &FileInode{}

// Create a file inode for the given object in GCS.
//
// REQUIRES: o != nil
// REQUIRES: o.Generation > 0
// REQUIRES: len(o.Name) > 0
// REQUIRES: o.Name[len(o.Name)-1] != '/'
func NewFileInode(
	ctx context.Context,
	bucket gcs.Bucket,
	id fuse.InodeID,
	o *storage.Object) (f *FileInode, err error) {
	// Set up the basic struct.
	f = &FileInode{
		bucket: bucket,
		id:     id,
	}

	// Set up the proxy.
	f.proxy, err = gcsproxy.NewObjectProxy(ctx, bucket, o)
	if err != nil {
		err = fmt.Errorf("NewObjectProxy: %v", err)
		return
	}

	// Set up invariant checking.
	f.mu = syncutil.NewInvariantMutex(f.checkInvariants)

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (f *FileInode) checkInvariants() {
	// Make sure the name is legal.
	name := f.proxy.Name()
	if len(name) == 0 || name[len(name)-1] == '/' {
		panic("Illegal file name: " + name)
	}

	// INVARIANT: proxy.CheckInvariants() does not panic
	f.proxy.CheckInvariants()
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

// EXCLUSIVE_LOCKS_REQUIRED(f.mu)
func (f *FileInode) Name() string {
	return f.proxy.Name()
}

// Return the generation number from which this inode was branched. This is
// used as a precondition in object write requests.
//
// TODO(jacobsa): Make sure to add a test for opening a file with O_CREAT then
// opening it again for reading, and sharing data across the two descriptors.
// This should fail if we have screwed up the fuse lookup process with regards
// to the zero generation.
//
// EXCLUSIVE_LOCKS_REQUIRED(f.mu)
func (f *FileInode) SourceGeneration() int64 {
	return f.proxy.SourceGeneration()
}

// EXCLUSIVE_LOCKS_REQUIRED(f.mu)
func (f *FileInode) Attributes(
	ctx context.Context) (attrs fuse.InodeAttributes, err error) {
	// Stat the object.
	sr, err := f.proxy.Stat(ctx)
	if err != nil {
		err = fmt.Errorf("Stat: %v", err)
		return
	}

	// Fill out the struct.
	//
	// TODO(jacobsa): Make ObjectProxy.Stat return a struct containing mtime as
	// well as size and clobbered. (Get mtime from the local file when around,
	// otherwise the source object.) Then include Mtime here. But first make sure
	// there is a failing test.
	attrs = fuse.InodeAttributes{
		Nlink: 1,
		Size:  uint64(sr.Size),
		Mode:  0700,
	}

	// If the object has been clobbered, we reflect that as the inode being
	// unlinked.
	if sr.Clobbered {
		attrs.Nlink = 0
	}

	return
}

// Serve a read request for this file.
//
// EXCLUSIVE_LOCKS_REQUIRED(f.mu)
func (f *FileInode) ReadFile(
	ctx context.Context,
	req *fuse.ReadFileRequest) (resp *fuse.ReadFileResponse, err error) {
	resp = &fuse.ReadFileResponse{}

	// Read from the proxy.
	buf := make([]byte, req.Size)
	n, err := f.proxy.ReadAt(ctx, buf, req.Offset)

	// We don't return errors for EOF. Otherwise, propagate errors.
	if err == io.EOF {
		err = nil
	} else if err != nil {
		err = fmt.Errorf("ReadAt: %v", err)
		return
	}

	// Fill in the response.
	resp.Data = buf[:n]

	return
}
