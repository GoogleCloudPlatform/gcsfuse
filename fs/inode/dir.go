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
	"os"
	"path"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

type DirInode struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	bucket gcs.Bucket

	/////////////////////////
	// Constant data
	/////////////////////////

	id fuseops.InodeID

	// The the GCS object backing the inode. The object's name is used as a
	// prefix when listing. Special case: the empty string means this is the root
	// inode.
	//
	// INVARIANT: src != nil
	// INVARIANT: src.Name == "" || src.Name[len(name)-1] == '/'
	src storage.Object
}

var _ Inode = &DirInode{}

// Create a directory inode for the root of the file system. For this inode,
// the result of SourceGeneration() is unspecified but stable.
func NewRootInode(bucket gcs.Bucket) (d *DirInode) {
	d = &DirInode{
		bucket: bucket,
		id:     fuseops.RootInodeID,

		// A dummy object whose name is the empty string.
		src: storage.Object{},
	}

	return
}

// Create a directory inode for the supplied source object. The object's name
// must end with a slash unless this is the root directory, in which case it
// must be empty.
//
// REQUIRES: o != nil
// REQUIRES: o.Name != ""
// REQUIRES: o.Name[len(o.Name)-1] == '/'
func NewDirInode(
	bucket gcs.Bucket,
	id fuseops.InodeID,
	o *storage.Object) (d *DirInode) {
	if o.Name[len(o.Name)-1] != '/' {
		panic(fmt.Sprintf("Unexpected name: %s", o.Name))
	}

	// Set up the struct.
	d = &DirInode{
		bucket: bucket,
		id:     id,
		src:    *o,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (d *DirInode) checkInvariants() {
	// INVARIANT: src.Name == "" || src.Name[len(name)-1] == '/'
	if !(d.src.Name == "" || d.src.Name[len(d.src.Name)-1] == '/') {
		panic(fmt.Sprintf("Unexpected name: %s", d.src.Name))
	}
}

func (d *DirInode) clobbered(ctx context.Context) (clobbered bool, err error) {
	// Special case: the root is never clobbered.
	if d.ID() == fuseops.RootInodeID {
		return
	}

	// Stat the backing object.
	req := &gcs.StatObjectRequest{
		Name: d.Name(),
	}

	o, err := d.bucket.StatObject(ctx, req)

	// "Not found" means clobbered.
	if _, ok := err.(*gcs.NotFoundError); ok {
		clobbered = true
		err = nil
		return
	}

	// Propagate other errors.
	if err != nil {
		err = fmt.Errorf("StatObject: %v", err)
		return
	}

	// We are clobbered if the generation number has changed.
	clobbered = o.Generation != d.SourceGeneration()

	return
}

// Stat the object with the given name, returning (nil, nil) if the object
// doesn't exist rather than failing.
func statObjectMayNotExist(
	ctx context.Context,
	bucket gcs.Bucket,
	name string) (o *storage.Object, err error) {
	// Call the bucket.
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	o, err = bucket.StatObject(ctx, req)

	// Suppress "not found" errors.
	if _, ok := err.(*gcs.NotFoundError); ok {
		err = nil
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (d *DirInode) Lock() {
	// We don't require locks.
}

func (d *DirInode) Unlock() {
	// We don't require locks.
}

// Return the ID of this inode.
func (d *DirInode) ID() fuseops.InodeID {
	return d.id
}

// Return the full name of the directory object in GCS, including the trailing
// slash (e.g. "foo/bar/").
func (d *DirInode) Name() string {
	return d.src.Name
}

// Return the generation number from which this inode was branched.
func (d *DirInode) SourceGeneration() int64 {
	return d.src.Generation
}

func (d *DirInode) Attributes(
	ctx context.Context) (attrs fuseops.InodeAttributes, err error) {
	// Find out whether the backing object has been clobbered in GCS.
	clobbered, err := d.clobbered(ctx)
	if err != nil {
		err = fmt.Errorf("clobbered: %v", err)
		return
	}

	// Set up basic attributes.
	attrs = fuseops.InodeAttributes{
		Mode: 0700 | os.ModeDir,
	}

	// Modify Nlink as appropriate.
	if !clobbered {
		attrs.Nlink = 1
	}

	return
}

// Look up the direct child with the given relative name, returning a record
// for the current object of that name in the GCS bucket. If both a file and a
// directory with the given name exist, be consistent from call to call about
// which is preferred. Return fuse.ENOENT if neither is found.
func (d *DirInode) LookUpChild(
	ctx context.Context,
	name string) (o *storage.Object, err error) {
	// Stat the child as a file first.
	o, err = statObjectMayNotExist(ctx, d.bucket, d.Name()+name)
	if err != nil {
		err = fmt.Errorf("statObjectMayNotExist: %v", err)
		return
	}

	// Did we find it successfully?
	if o != nil {
		return
	}

	// Try again as a directory.
	o, err = statObjectMayNotExist(ctx, d.bucket, d.Name()+name+"/")
	if err != nil {
		err = fmt.Errorf("statObjectMayNotExist: %v", err)
		return
	}

	// This time "not found" is an error.
	if o == nil {
		err = fuse.ENOENT
		return
	}

	return
}

// Read some number of entries from the directory, returning a continuation
// token that can be used to pick up the read operation where it left off.
// Supply the empty token on the first call.
//
// At the end of the directory, the returned continuation token will be empty.
// Otherwise it will be non-empty. There is no guarantee about the number of
// entries returned; it may be zero even with a non-empty continuation token.
//
// The contents of the Offset and Inode fields for returned entries is
// undefined.
func (d *DirInode) ReadEntries(
	ctx context.Context,
	tok string) (entries []fuseutil.Dirent, newTok string, err error) {
	// Ask the bucket to list some objects.
	query := &storage.Query{
		Delimiter: "/",
		Prefix:    d.Name(),
		Cursor:    tok,
	}

	listing, err := d.bucket.ListObjects(ctx, query)
	if err != nil {
		err = fmt.Errorf("ListObjects: %v", err)
		return
	}

	// Convert objects to entries for files.
	for _, o := range listing.Results {
		e := fuseutil.Dirent{
			Name: path.Base(o.Name),
			Type: fuseutil.DT_File,
		}

		entries = append(entries, e)
	}

	// Convert prefixes to entries for directories.
	for _, p := range listing.Prefixes {
		e := fuseutil.Dirent{
			Name: path.Base(p),
			Type: fuseutil.DT_Directory,
		}

		entries = append(entries, e)
	}

	// Return an appropriate continuation token, if any.
	if listing.Next != nil {
		newTok = listing.Next.Cursor
	}

	return
}
