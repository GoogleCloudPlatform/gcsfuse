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
	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
)

type DirInode struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	bucket gcs.Bucket

	/////////////////////////
	// Constant data
	/////////////////////////

	id           fuseops.InodeID
	implicitDirs bool

	// The the GCS object backing the inode. The object's name is used as a
	// prefix when listing. Special case: the empty string means this is the root
	// inode.
	//
	// INVARIANT: src != nil
	// INVARIANT: src.Name == "" || src.Name[len(name)-1] == '/'
	src gcs.Object

	/////////////////////////
	// Mutable state
	/////////////////////////

	lc lookupCount
}

var _ Inode = &DirInode{}

// Set notes on NewRootInode.
const RootGen int64 = 0

// Create a directory inode for the root of the file system. For this inode,
// the result of SourceGeneration() is guaranteed to be RootGen.
//
// The initial lookup count is one.
func NewRootInode(
	bucket gcs.Bucket,
	implicitDirs bool) (d *DirInode) {
	dummy := &gcs.Object{
		Name:       "",
		Generation: RootGen,
	}

	d = NewDirInode(bucket, fuseops.RootInodeID, dummy, implicitDirs)
	return
}

// Create a directory inode for the supplied source object. The object's name
// must end with a slash unless this is the root directory, in which case it
// must be empty.
//
// If implicitDirs is set, LookUpChild will use ListObjects to find child
// directories that are "implicitly" defined by the existence of their own
// descendents. For example, if there is an object named "foo/bar/baz" and this
// is the directory "foo", a child directory named "bar" will be implied.
//
// The initial lookup count is one.
//
// REQUIRES: o != nil
// REQUIRES: o.Name == "" || o.Name[len(o.Name)-1] == '/'
func NewDirInode(
	bucket gcs.Bucket,
	id fuseops.InodeID,
	o *gcs.Object,
	implicitDirs bool) (d *DirInode) {
	if o.Name != "" && o.Name[len(o.Name)-1] != '/' {
		panic(fmt.Sprintf("Unexpected name: %s", o.Name))
	}

	// Set up the struct.
	d = &DirInode{
		bucket:       bucket,
		id:           id,
		implicitDirs: implicitDirs,
		src:          *o,
	}

	d.lc = lookupCount{
		count:   1,
		destroy: func() error { return nil },
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

func (d *DirInode) lookUpChildFile(
	ctx context.Context,
	name string) (o *gcs.Object, err error) {
	o, err = statObjectMayNotExist(ctx, d.bucket, d.Name()+name)
	if err != nil {
		err = fmt.Errorf("statObjectMayNotExist: %v", err)
		return
	}

	return
}

func (d *DirInode) lookUpChildDir(
	ctx context.Context,
	name string) (o *gcs.Object, err error) {
	b := syncutil.NewBundle(ctx)

	// Stat the placeholder object.
	b.Add(func(ctx context.Context) (err error) {
		o, err = statObjectMayNotExist(ctx, d.bucket, d.Name()+name+"/")
		if err != nil {
			err = fmt.Errorf("statObjectMayNotExist: %v", err)
			return
		}

		return
	})

	// If implicit directories are enabled, find out whether the child name is
	// implicitly defined.
	var implicitlyDefined bool
	if d.implicitDirs {
		b.Add(func(ctx context.Context) (err error) {
			implicitlyDefined, err = objectNamePrefixNonEmpty(
				ctx,
				d.bucket,
				d.Name()+name+"/")

			if err != nil {
				err = fmt.Errorf("objectNamePrefixNonEmpty: %v", err)
				return
			}

			return
		})
	}

	// Wait for both.
	err = b.Join()
	if err != nil {
		return
	}

	// If statting failed by the directory is implicitly defined, fake a source
	// object.
	if o == nil && implicitlyDefined {
		o = &gcs.Object{
			Name:       d.Name() + name + "/",
			Generation: ImplicitDirGen,
		}
	}

	return
}

// List the supplied object name prefix to find out whether it is non-empty.
func objectNamePrefixNonEmpty(
	ctx context.Context,
	bucket gcs.Bucket,
	prefix string) (nonEmpty bool, err error) {
	req := &gcs.ListObjectsRequest{
		Prefix:     prefix,
		MaxResults: 1,
	}

	listing, err := bucket.ListObjects(ctx, req)
	if err != nil {
		err = fmt.Errorf("ListObjects: %v", err)
		return
	}

	nonEmpty = len(listing.Objects) != 0
	return
}

// Stat the object with the given name, returning (nil, nil) if the object
// doesn't exist rather than failing.
func statObjectMayNotExist(
	ctx context.Context,
	bucket gcs.Bucket,
	name string) (o *gcs.Object, err error) {
	// Call the bucket.
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	o, err = bucket.StatObject(ctx, req)

	// Suppress "not found" errors.
	if _, ok := err.(*gcs.NotFoundError); ok {
		err = nil
	}

	// Annotate others.
	if err != nil {
		err = fmt.Errorf("StatObject: %v", err)
		return
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

// LOCKS_REQUIRED(d.mu)
func (d *DirInode) IncrementLookupCount() {
	d.lc.Inc()
}

// LOCKS_REQUIRED(d.mu)
func (d *DirInode) DecrementLookupCount(n uint64) (destroyed bool) {
	destroyed = d.lc.Dec(n)
	return
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

// See notes on DirInode.LookUpChild.
const ImplicitDirGen int64 = -1

// Look up the direct child with the given relative name, returning a record
// for the current object of that name in the GCS bucket. If both a file and a
// directory with the given name exist, be consistent from call to call about
// which is preferred. Return fuse.ENOENT if neither is found.
//
// If this inode was created with implicitDirs is set, this method will use
// ListObjects to find child directories that are "implicitly" defined by the
// existence of their own descendents. For example, if there is an object named
// "foo/bar/baz" and this is the directory "foo", a child directory named "bar"
// will be implied. In this case, the child's generation will be the sentinel
// value ImplicitDirGen.
func (d *DirInode) LookUpChild(
	ctx context.Context,
	name string) (o *gcs.Object, err error) {
	b := syncutil.NewBundle(ctx)

	// Stat the child as a file.
	var fileRecord *gcs.Object
	b.Add(func(ctx context.Context) (err error) {
		fileRecord, err = d.lookUpChildFile(ctx, name)
		return
	})

	// Stat the child as a directory.
	var dirRecord *gcs.Object
	b.Add(func(ctx context.Context) (err error) {
		dirRecord, err = d.lookUpChildDir(ctx, name)
		return
	})

	// Wait for both.
	err = b.Join()
	if err != nil {
		return
	}

	// Prefer directories over files.
	switch {
	case dirRecord != nil:
		o = dirRecord
	case fileRecord != nil:
		o = fileRecord
	default:
		err = fuse.ENOENT
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
	req := &gcs.ListObjectsRequest{
		Delimiter:         "/",
		Prefix:            d.Name(),
		ContinuationToken: tok,
	}

	listing, err := d.bucket.ListObjects(ctx, req)
	if err != nil {
		err = fmt.Errorf("ListObjects: %v", err)
		return
	}

	// Convert objects to entries for files.
	for _, o := range listing.Objects {
		e := fuseutil.Dirent{
			Name: path.Base(o.Name),
			Type: fuseutil.DT_File,
		}

		entries = append(entries, e)
	}

	// Convert runs to entries for directories.
	for _, p := range listing.CollapsedRuns {
		e := fuseutil.Dirent{
			Name: path.Base(p),
			Type: fuseutil.DT_Directory,
		}

		entries = append(entries, e)
	}

	// Return an appropriate continuation token, if any.
	newTok = listing.ContinuationToken

	return
}
