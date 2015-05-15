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
	"strings"
	"sync"
	"time"

	"github.com/googlecloudplatform/gcsfuse/timeutil"
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
	clock  timeutil.Clock

	/////////////////////////
	// Constant data
	/////////////////////////

	id           fuseops.InodeID
	implicitDirs bool

	// INVARIANT: name == "" || name[len(name)-1] == '/'
	name string

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A mutex that must be held when calling certain methods. See documentation
	// for each method.
	mu syncutil.InvariantMutex

	// GUARDED_BY(mu)
	lc lookupCount

	// cache.CheckInvariants() does not panic.
	//
	// GUARDED_BY(mu)
	cache typeCache
}

var _ Inode = &DirInode{}

// Create a directory inode for the root of the file system. The initial lookup
// count is zero.
func NewRootInode(
	implicitDirs bool,
	typeCacheTTL time.Duration,
	bucket gcs.Bucket,
	clock timeutil.Clock) (d *DirInode) {
	d = NewDirInode(
		fuseops.RootInodeID,
		"",
		implicitDirs,
		typeCacheTTL,
		bucket,
		clock)

	return
}

// Create a directory inode for the supplied object name. The name must end
// with a slash unless this is the root directory, in which case it must be
// empty.
//
// If implicitDirs is set, LookUpChild will use ListObjects to find child
// directories that are "implicitly" defined by the existence of their own
// descendents. For example, if there is an object named "foo/bar/baz" and this
// is the directory "foo", a child directory named "bar" will be implied.
//
// If typeCacheTTL is non-zero, a cache from child name to information about
// whether that name exists as a file and/or directory will be maintained. This
// may speed up calls to LookUpChild, especially when combined with a
// stat-caching GCS bucket, but comes at the cost of consistency: if the child
// is removed and recreated with a different type before the expiration, we may
// fail to find it.
//
// The initial lookup count is zero.
//
// REQUIRES: name == "" || name[len(name)-1] == '/'
func NewDirInode(
	id fuseops.InodeID,
	name string,
	implicitDirs bool,
	typeCacheTTL time.Duration,
	bucket gcs.Bucket,
	clock timeutil.Clock) (d *DirInode) {
	if name != "" && name[len(name)-1] != '/' {
		panic(fmt.Sprintf("Unexpected name: %s", name))
	}

	// Set up the struct.
	const typeCacheCapacity = 1 << 16
	d = &DirInode{
		bucket:       bucket,
		clock:        clock,
		id:           id,
		implicitDirs: implicitDirs,
		name:         name,
		cache:        newTypeCache(typeCacheCapacity/2, typeCacheTTL),
	}

	d.lc.Init(id)

	// Set up invariant checking.
	d.mu = syncutil.NewInvariantMutex(d.checkInvariants)

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (d *DirInode) checkInvariants() {
	// INVARIANT: name == "" || name[len(name)-1] == '/'
	if !(d.name == "" || d.name[len(d.name)-1] == '/') {
		panic(fmt.Sprintf("Unexpected name: %s", d.name))
	}

	// cache.CheckInvariants() does not panic.
	d.cache.CheckInvariants()
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
			Name: d.Name() + name + "/",
		}
	}

	return
}

// Look up the file for a (file, dir) pair with conflicting names, overriding
// the default behavior. If the file doesn't exist, return a nil record with a
// nil error. If the directory doesn't exist, pretend the file doesn't exist.
//
// REQUIRES: strings.HasSuffix(name, ConflictingFileNameSuffix)
func (d *DirInode) lookUpConflicting(
	ctx context.Context,
	name string) (o *gcs.Object, err error) {
	strippedName := strings.TrimSuffix(name, ConflictingFileNameSuffix)

	// In order to a marked name to be accepted, we require the conflicting
	// directory to exist.
	var dir *gcs.Object
	dir, err = d.lookUpChildDir(ctx, strippedName)
	if err != nil {
		err = fmt.Errorf("lookUpChildDir for stripped name: %v", err)
		return
	}

	if dir == nil {
		return
	}

	// The directory name exists. Find the conflicting file.
	o, err = d.lookUpChildFile(ctx, strippedName)
	if err != nil {
		err = fmt.Errorf("lookUpChildFile for stripped name: %v", err)
		return
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

// Fail if the name already exists.
func (d *DirInode) createNewObject(
	ctx context.Context,
	name string) (o *gcs.Object, err error) {
	// Create an empty backing object for the child, failing if it already
	// exists.
	var precond int64
	createReq := &gcs.CreateObjectRequest{
		Name:                   name,
		Contents:               strings.NewReader(""),
		GenerationPrecondition: &precond,
	}

	o, err = d.bucket.CreateObject(ctx, createReq)
	if err != nil {
		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	return
}

// An implementation detail fo filterMissingChildDirs.
func filterMissingChildDirNames(
	ctx context.Context,
	bucket gcs.Bucket,
	dirName string,
	unfiltered <-chan string,
	filtered chan<- string) (err error) {
	for name := range unfiltered {
		var o *gcs.Object

		// Stat the placeholder.
		o, err = statObjectMayNotExist(ctx, bucket, dirName+name+"/")
		if err != nil {
			err = fmt.Errorf("statObjectMayNotExist: %v", err)
			return
		}

		// Should we pass on this name?
		if o == nil {
			continue
		}

		select {
		case <-ctx.Done():
			err = ctx.Err()
			return

		case filtered <- name:
		}
	}

	return
}

// Given a list of child names that appear to be directories according to
// d.bucket.ListObjects (which always behaves as if implicit directories are
// enabled), filter out the ones for which a placeholder object does not
// actually exist. If implicit directories are enabled, simply return them all.
//
// LOCKS_REQUIRED(d)
func (d *DirInode) filterMissingChildDirs(
	ctx context.Context,
	in []string) (out []string, err error) {
	// Do we need to do anything?
	if d.implicitDirs {
		out = in
		return
	}

	b := syncutil.NewBundle(ctx)

	// First add any names that we already know are directories according to our
	// cache, removing them from the input.
	now := d.clock.Now()
	var tmp []string
	for _, name := range in {
		if d.cache.IsDir(now, name) {
			out = append(out, name)
		} else {
			tmp = append(tmp, name)
		}
	}

	in = tmp

	// Feed names into a channel.
	unfiltered := make(chan string, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(unfiltered)

		for _, name := range in {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				return

			case unfiltered <- name:
			}
		}

		return
	})

	// Stat the placeholder object for each, filtering out placeholders that are
	// not found. Use some parallelism.
	const statWorkers = 32
	filtered := make(chan string, 100)
	var wg sync.WaitGroup
	for i := 0; i < statWorkers; i++ {
		wg.Add(1)
		b.Add(func(ctx context.Context) (err error) {
			defer wg.Done()
			err = filterMissingChildDirNames(
				ctx,
				d.bucket,
				d.Name(),
				unfiltered,
				filtered)

			return
		})
	}

	go func() {
		wg.Wait()
		close(filtered)
	}()

	// Accumulate into a slice.
	var filteredSlice []string
	b.Add(func(ctx context.Context) (err error) {
		for name := range filtered {
			filteredSlice = append(filteredSlice, name)
		}

		return
	})

	// Wait for everything to complete.
	err = b.Join()

	// Update the cache with everything we learned.
	now = d.clock.Now()
	for _, name := range filteredSlice {
		d.cache.NoteDir(now, name)
	}

	// Return everything we learned.
	out = append(out, filteredSlice...)

	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (d *DirInode) Lock() {
	d.mu.Lock()
}

func (d *DirInode) Unlock() {
	d.mu.Unlock()
}

func (d *DirInode) ID() fuseops.InodeID {
	return d.id
}

// Return the full name of the directory object in GCS, including the trailing
// slash (e.g. "foo/bar/").
func (d *DirInode) Name() string {
	return d.name
}

// LOCKS_REQUIRED(d)
func (d *DirInode) IncrementLookupCount() {
	d.lc.Inc()
}

// LOCKS_REQUIRED(d)
func (d *DirInode) DecrementLookupCount(n uint64) (destroy bool) {
	destroy = d.lc.Dec(n)
	return
}

// LOCKS_REQUIRED(d)
func (d *DirInode) Destroy() (err error) {
	// Nothing interesting to do.
	return
}

// LOCKS_REQUIRED(d)
func (d *DirInode) Attributes(
	ctx context.Context) (attrs fuseops.InodeAttributes, err error) {
	// Set up basic attributes.
	attrs = fuseops.InodeAttributes{
		Mode:  0700 | os.ModeDir,
		Nlink: 1,
	}

	return
}

// A suffix that can be used to unambiguously tag a file system name.
// (Unambiguous because U+000A is not allowed in GCS object names.) This is
// used to refer to the file in a (file, directory) pair with conflicting
// object names.
//
// See also the notes on DirInode.LookUpChild.
const ConflictingFileNameSuffix = "\n"

// Look up the direct child with the given relative name, returning a record
// for the current object of that name in the GCS bucket. If both a file and a
// directory with the given name exist, the directory is preferred. Return a
// nil record with a nil error if neither is found.
//
// Special case: if the name ends in ConflictingFileNameSuffix, we strip the
// suffix, confirm that a conflicting directory exists, then return a record
// for the file.
//
// If this inode was created with implicitDirs is set, this method will use
// ListObjects to find child directories that are "implicitly" defined by the
// existence of their own descendents. For example, if there is an object named
// "foo/bar/baz" and this is the directory "foo", a child directory named "bar"
// will be implied.
//
// LOCKS_REQUIRED(d)
func (d *DirInode) LookUpChild(
	ctx context.Context,
	name string) (o *gcs.Object, err error) {
	// Consult the cache about the type of the child. This may save us work
	// below.
	now := d.clock.Now()
	cacheSaysFile := d.cache.IsFile(now, name)
	cacheSaysDir := d.cache.IsDir(now, name)

	// Is this a conflict marker name?
	if strings.HasSuffix(name, ConflictingFileNameSuffix) {
		o, err = d.lookUpConflicting(ctx, name)
		return
	}

	// Stat the child as a file, unless the cache has told us it's a directory
	// but not a file.
	b := syncutil.NewBundle(ctx)

	var fileRecord *gcs.Object
	if !(cacheSaysDir && !cacheSaysFile) {
		b.Add(func(ctx context.Context) (err error) {
			fileRecord, err = d.lookUpChildFile(ctx, name)
			return
		})
	}

	// Stat the child as a directory, unless the cache has told us it's a file
	// but not a directory.
	var dirRecord *gcs.Object
	if !(cacheSaysFile && !cacheSaysDir) {
		b.Add(func(ctx context.Context) (err error) {
			dirRecord, err = d.lookUpChildDir(ctx, name)
			return
		})
	}

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
	}

	// Update the cache.
	now = d.clock.Now()
	if fileRecord != nil {
		d.cache.NoteFile(now, name)
	}

	if dirRecord != nil {
		d.cache.NoteDir(now, name)
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
//
// LOCKS_REQUIRED(d)
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
		// Skip the entry for the backing object itself, which of course has its
		// own name as a prefix but which we don't wan tto appear to contain
		// itself.
		if o.Name == d.Name() {
			continue
		}

		e := fuseutil.Dirent{
			Name: path.Base(o.Name),
			Type: fuseutil.DT_File,
		}

		entries = append(entries, e)
	}

	// Extract directory names from the collapsed runs.
	var dirNames []string
	for _, p := range listing.CollapsedRuns {
		dirNames = append(dirNames, path.Base(p))
	}

	// Filter the directory names according to our implicit directory settings.
	dirNames, err = d.filterMissingChildDirs(ctx, dirNames)
	if err != nil {
		err = fmt.Errorf("filterMissingChildDirs: %v", err)
		return
	}

	// Return entries for directories.
	for _, name := range dirNames {
		e := fuseutil.Dirent{
			Name: name,
			Type: fuseutil.DT_Directory,
		}

		entries = append(entries, e)
	}

	// Return an appropriate continuation token, if any.
	newTok = listing.ContinuationToken

	// Update the type cache with everything we learned.
	now := d.clock.Now()
	for _, e := range entries {
		switch e.Type {
		case fuseutil.DT_File:
			d.cache.NoteFile(now, e.Name)

		case fuseutil.DT_Directory:
			d.cache.NoteDir(now, e.Name)
		}
	}

	return
}

// Create an empty child file with the supplied (relative) name, failing if a
// backing object already exists in GCS.
//
// LOCKS_REQUIRED(d)
func (d *DirInode) CreateChildFile(
	ctx context.Context,
	name string) (o *gcs.Object, err error) {
	o, err = d.createNewObject(ctx, path.Join(d.Name(), name))
	if err != nil {
		return
	}

	d.cache.NoteFile(d.clock.Now(), name)

	return
}

// Create a backing object for a child directory with the supplied (relative)
// name, failing if a backing object already exists in GCS.
//
// LOCKS_REQUIRED(d)
func (d *DirInode) CreateChildDir(
	ctx context.Context,
	name string) (o *gcs.Object, err error) {
	o, err = d.createNewObject(ctx, path.Join(d.Name(), name)+"/")
	if err != nil {
		return
	}

	d.cache.NoteDir(d.clock.Now(), name)

	return
}

// Delete the backing object for the child file with the given (relative) name.
//
// LOCKS_REQUIRED(d)
func (d *DirInode) DeleteChildFile(
	ctx context.Context,
	name string) (err error) {
	d.cache.Erase(name)

	err = d.bucket.DeleteObject(ctx, path.Join(d.Name(), name))
	if err != nil {
		err = fmt.Errorf("DeleteObject: %v", err)
		return
	}

	return
}

// Delete the backing object for the child directory with the given (relative)
// name.
//
// LOCKS_REQUIRED(d)
func (d *DirInode) DeleteChildDir(
	ctx context.Context,
	name string) (err error) {
	d.cache.Erase(name)

	// Delete the backing object. Unfortunately we have no way to precondition
	// this on the directory being empty.
	err = d.bucket.DeleteObject(ctx, path.Join(d.Name(), name)+"/")
	if err != nil {
		err = fmt.Errorf("DeleteObject: %v", err)
		return
	}

	return
}
