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
	"path"

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

// Create a directory inode for the directory with the given name. The name
// must end with a slash unless this is the root directory, in which case it
// must be empty.
//
// REQUIRES: name == "" || name[len(name)-1] == '/'
func NewDirInode(bucket gcs.Bucket, name string) (d *DirInode) {
	// Set up the basic struct.
	d = &DirInode{
		bucket: bucket,
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

// Read some number of entries from the directory, returning a continuation
// token that can be used to pick up the read operation where it left off.
// Supply the empty token on the first call.
//
// At the end of the directory, the returned continuation token will be empty.
// Otherwise it will be non-empty. There is no guarantee about the number of
// entries returned; it may be zero even with a non-empty continuation token.
//
// The contents of the Offset field for returned entries is undefined.
//
// SHARED_LOCKS_REQUIRED(d.Mu)
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
			Inode: GenerateFileInodeID(o.Name, o.Generation),
			Name:  path.Base(o.Name),
			Type:  fuseutil.DT_File,
		}

		entries = append(entries, e)
	}

	// Convert prefixes to entries for directories.
	for _, p := range listing.Prefixes {
		e := fuseutil.Dirent{
			Inode: GenerateDirInodeID(p),
			Name:  path.Base(p),
			Type:  fuseutil.DT_Directory,
		}

		entries = append(entries, e)
	}

	// Return an appropriate continuation token, if any.
	if listing.Next != nil {
		newTok = listing.Next.Cursor
	}

	return
}
