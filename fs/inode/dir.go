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
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
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

	id fuse.InodeID

	// The name of the GCS object backing the inode, used as a prefix when
	// listing. Special case: the empty string means this is the root inode.
	//
	// INVARIANT: name == "" || name[len(name)-1] == '/'
	name string

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A mutex that must be held when calling certain methods. See documentation
	// for each method.
	Mu syncutil.InvariantMutex
}

var _ Inode = &DirInode{}

// Create a directory inode for the directory with the given name. The name
// must end with a slash unless this is the root directory, in which case it
// must be empty.
//
// REQUIRES: name == "" || name[len(name)-1] == '/'
func NewDirInode(
	bucket gcs.Bucket,
	id fuse.InodeID,
	name string) (d *DirInode) {
	// Set up the basic struct.
	d = &DirInode{
		bucket: bucket,
		id:     id,
		name:   name,
	}

	// Set up invariant checking.
	d.Mu = syncutil.NewInvariantMutex(d.checkInvariants)

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (d *DirInode) checkInvariants() {
	// Check the name.
	if !(d.name == "" || d.name[len(d.name)-1] == '/') {
		panic(fmt.Sprintf("Unexpected name: %s", d.name))
	}
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// Return the ID of this inode.
func (d *DirInode) ID() fuse.InodeID {
	return d.id
}

// Return the full name of the directory object in GCS, including the trailing
// slash (e.g. "foo/bar/").
func (d *DirInode) Name() string {
	return d.name
}

// Return up to date attributes for the directory.
func (d *DirInode) Attributes(
	ctx context.Context) (attrs fuse.InodeAttributes, err error) {
	attrs = fuse.InodeAttributes{
		Mode: 0700 | os.ModeDir,
		// TODO(jacobsa): Track mtime and maybe atime.
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
	// See if the child is a directory first.
	statReq := &gcs.StatObjectRequest{
		Name: d.name + name + "/",
	}

	o, err = d.bucket.StatObject(ctx, statReq)
	if err != nil && err != gcs.ErrNotFound {
		err = fmt.Errorf("StatObject: %v", err)
		return
	}

	// Try again as a file.
	statReq.Name = d.name + name
	o, err = d.bucket.StatObject(ctx, statReq)

	if err == gcs.ErrNotFound {
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
		Prefix:    d.name,
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
