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
	// INVARIANT: src.Name == "" || src.Name[len(name)-1] == '/'
	src *storage.Object
}

var _ Inode = &DirInode{}

// Create a directory inode for the supplied source object. The object's name
// must end with a slash unless this is the root directory, in which case it
// must be empty.
//
// REQUIRES: o != nil
// REQUIRES: o.Name == "" || o.Name[len(name)-1] == '/'
func NewDirInode(
	bucket gcs.Bucket,
	id fuseops.InodeID,
	o *storage.Object) (d *DirInode) {
	if !(o.Name == "" || o.Name[len(o.Name)-1] == '/') {
		panic(fmt.Sprintf("Unexpected name: %s", o.Name))
	}

	// Set up the basic struct.
	d = &DirInode{
		bucket: bucket,
		id:     id,
		src:    o,
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
	attrs = fuseops.InodeAttributes{
		Nlink: 1,
		Mode:  0700 | os.ModeDir,
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
	// Stat the child as a directory first.
	statReq := &gcs.StatObjectRequest{
		Name: d.Name() + name + "/",
	}

	o, err = d.bucket.StatObject(ctx, statReq)

	// Did we find it successfully?
	if err == nil {
		return
	}

	// Propagate all errors except "not found".
	if err != nil {
		if _, ok := err.(*gcs.NotFoundError); !ok {
			err = fmt.Errorf("StatObject: %v", err)
			return
		}
	}

	// Try again as a file.
	statReq.Name = d.Name() + name
	o, err = d.bucket.StatObject(ctx, statReq)

	if _, ok := err.(*gcs.NotFoundError); ok {
		err = fuse.ENOENT
		return
	}

	if err != nil {
		err = fmt.Errorf("StatObject: %v", err)
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
