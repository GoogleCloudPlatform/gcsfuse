// Copyright 2020 Google Inc. All Rights Reserved.
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
	"sync"
	"syscall"

	"github.com/jacobsa/fuse"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// An inode that
//
//	(1) represents a base directory which contains a list of
//	    subdirectories as the roots of different GCS buckets;
//	(2) implements BaseDirInode, allowing read only ops.
type baseDirInode struct {
	/////////////////////////
	// Constant data
	/////////////////////////
	id fuseops.InodeID

	// INVARIANT: name.IsDir()
	name Name

	attrs fuseops.InodeAttributes

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A mutex that must be held when calling certain methods. See documentation
	// for each method.
	mu sync.Mutex

	lc lookupCount

	// GUARDED_BY(mu)
	bucketManager gcsx.BucketManager

	// GUARDED_BY(mu)
	buckets map[string]gcsx.SyncerBucket
}

// NewBaseDirInode returns a baseDirInode that acts as the directory of
// buckets.
func NewBaseDirInode(
	id fuseops.InodeID,
	name Name,
	attrs fuseops.InodeAttributes,
	bm gcsx.BucketManager) (d DirInode) {
	typed := &baseDirInode{
		id:            id,
		name:          NewRootName(""),
		attrs:         attrs,
		bucketManager: bm,
		buckets:       make(map[string]gcsx.SyncerBucket),
	}
	typed.lc.Init(id)

	d = typed
	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (d *baseDirInode) Lock() {
	d.mu.Lock()
}

func (d *baseDirInode) Unlock() {
	d.mu.Unlock()
}

func (d *baseDirInode) ID() fuseops.InodeID {
	return d.id
}

func (d *baseDirInode) Name() Name {
	return d.name
}

// LOCKS_REQUIRED(d)
func (d *baseDirInode) IncrementLookupCount() {
	d.lc.Inc()
}

// LOCKS_REQUIRED(d)
func (d *baseDirInode) DecrementLookupCount(n uint64) (destroy bool) {
	destroy = d.lc.Dec(n)
	return
}

// LOCKS_REQUIRED(d)
func (d *baseDirInode) Destroy() (err error) {
	// Nothing interesting to do.
	return
}

// LOCKS_REQUIRED(d)
func (d *baseDirInode) Attributes(
	ctx context.Context) (attrs fuseops.InodeAttributes, err error) {
	// Set up basic attributes.
	attrs = d.attrs
	attrs.Nlink = 1

	return
}

// LOCKS_REQUIRED(d)
func (d *baseDirInode) LookUpChild(ctx context.Context, name string) (*Core, error) {
	var err error
	bucket, ok := d.buckets[name]
	if !ok {
		bucket, err = d.bucketManager.SetUpBucket(ctx, name)
		if err != nil {
			return nil, err
		}
		d.buckets[name] = bucket
	}

	return &Core{
		Bucket:   &bucket,
		FullName: NewRootName(bucket.Name()),
		Object:   nil,
	}, nil
}

// Not implemented
func (d *baseDirInode) ReadDescendants(ctx context.Context, limit int) (map[Name]*Core, error) {
	return nil, fuse.ENOSYS
}

// LOCKS_REQUIRED(d)
func (d *baseDirInode) ReadEntries(
	ctx context.Context,
	tok string) (entries []fuseutil.Dirent, newTok string, err error) {

	// The subdirectories of the base directory should be all the accessible
	// buckets. Although the user is allowed to visit each individual
	// subdirectory, listing all the subdirectories (i.e. the buckets) can be
	// very expensive and currently not supported.
	return nil, "", syscall.ENOTSUP
}

////////////////////////////////////////////////////////////////////////
// Forbidden Public interface
////////////////////////////////////////////////////////////////////////

// The base directory is a directory of buckets. Opeations for mutating
// buckets (such as creation or deletion) are not supported. When the user
// tries to mutate the base directory, they will receive a ENOSYS error
// indicating such operation is not supported.

func (d *baseDirInode) CreateChildFile(ctx context.Context, name string) (*Core, error) {
	return nil, fuse.ENOSYS
}

func (d *baseDirInode) CloneToChildFile(ctx context.Context, name string, src *gcs.Object) (*Core, error) {
	return nil, fuse.ENOSYS
}

func (d *baseDirInode) CreateChildSymlink(ctx context.Context, name string, target string) (*Core, error) {
	return nil, fuse.ENOSYS
}

func (d *baseDirInode) CreateChildDir(ctx context.Context, name string) (*Core, error) {
	return nil, fuse.ENOSYS
}

func (d *baseDirInode) DeleteChildFile(
	ctx context.Context,
	name string,
	generation int64,
	metaGeneration *int64) (err error) {
	err = fuse.ENOSYS
	return
}

func (d *baseDirInode) DeleteChildDir(
	ctx context.Context,
	name string) (err error) {
	err = fuse.ENOSYS
	return
}
