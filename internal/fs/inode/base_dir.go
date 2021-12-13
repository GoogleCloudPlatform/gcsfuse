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

	"github.com/jacobsa/fuse"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// An inode that
//  (1) represents a base directory which contains a list of
//      subdirectories as the roots of different GCS buckets;
//  (2) implements BaseDirInode, allowing read only ops.
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
// Helpers
////////////////////////////////////////////////////////////////////////

// LOCKS_REQUIRED(d)
func (d *baseDirInode) lookUpOrSetUpBucket(
	ctx context.Context,
	name string) (b gcsx.SyncerBucket, err error) {
	var ok bool
	b, ok = d.buckets[name]
	if ok {
		return
	}

	b, err = d.bucketManager.SetUpBucket(ctx, name)
	if err != nil {
		return
	}
	d.buckets[name] = b
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
func (d *baseDirInode) LookUpChild(
	ctx context.Context,
	name string) (result BackObject, err error) {
	bucket, ok := d.buckets[name]
	if !ok {
		bucket, err = d.bucketManager.SetUpBucket(ctx, name)
		if err != nil {
			return
		}
		d.buckets[name] = bucket
	}

	result.Bucket = bucket
	result.FullName = NewRootName(bucket.Name())
	result.ImplicitDir = false
	result.Object = nil
	return
}

// Not implemented
func (d *baseDirInode) ReadDescendants(
	ctx context.Context,
	limit int) (descendants map[Name]BackObject, err error) {
	err = fuse.ENOSYS
	return
}

// Not implemented
func (d *baseDirInode) ReadObjects(
	ctx context.Context,
	tok string) (files []BackObject, dirs []BackObject, newTok string, err error) {
	err = fuse.ENOSYS
	return
}

// LOCKS_REQUIRED(d)
func (d *baseDirInode) ReadEntries(
	ctx context.Context,
	tok string) (entries []fuseutil.Dirent, newTok string, err error) {
	var bucketNames []string
	bucketNames, err = d.bucketManager.ListBuckets(ctx)
	if err != nil {
		return
	}
	for _, name := range bucketNames {
		entry := fuseutil.Dirent{
			Name: name,
			Type: fuseutil.DT_Directory,
		}
		entries = append(entries, entry)
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Forbidden Public interface
////////////////////////////////////////////////////////////////////////

// The base directory is a directory of buckets. Opeations for mutating
// buckets (such as creation or deletion) are not supported. When the user
// tries to mutate the base directory, they will receive a ENOSYS error
// indicating such operation is not supported.

func (d *baseDirInode) CreateChildFile(
	ctx context.Context,
	name string) (result BackObject, err error) {
	err = fuse.ENOSYS
	return
}

func (d *baseDirInode) CloneToChildFile(
	ctx context.Context,
	name string,
	src *gcs.Object) (result BackObject, err error) {
	err = fuse.ENOSYS
	return
}

func (d *baseDirInode) CreateChildSymlink(
	ctx context.Context,
	name string,
	target string) (result BackObject, err error) {
	err = fuse.ENOSYS
	return
}

func (d *baseDirInode) CreateChildDir(
	ctx context.Context,
	name string) (result BackObject, err error) {
	err = fuse.ENOSYS
	return
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
