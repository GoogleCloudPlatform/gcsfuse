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
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

// An inode representing a directory, with facilities for listing entries,
// looking up children, and creating and deleting children. Must be locked for
// any method additional to the Inode interface.
type DirInode interface {
	Inode

	// Look up the direct child with the given relative name, returning
	// information about the object backing the child or whether it exists as an
	// implicit directory. If a file/symlink and a directory with the given name
	// both exist, the directory is preferred. Return a result with
	// !result.Exists() and a nil error if neither is found.
	//
	// Special case: if the name ends in ConflictingFileNameSuffix, we strip the
	// suffix, confirm that a conflicting directory exists, then return a result
	// for the file/symlink.
	//
	// If this inode was created with implicitDirs is set, this method will use
	// ListObjects to find child directories that are "implicitly" defined by the
	// existence of their own descendents. For example, if there is an object
	// named "foo/bar/baz" and this is the directory "foo", a child directory
	// named "bar" will be implied. In this case, result.ImplicitDir will be
	// true.
	LookUpChild(
		ctx context.Context,
		name string) (result Core, err error)

	// Read the children objects of this dir, recursively. The result count
	// is capped at the given limit. Internal caches are not refreshed from this
	// call.
	ReadDescendants(
		ctx context.Context,
		limit int) (descendants map[Name]Core, err error)

	// Read some number of objects from the directory, returning a continuation
	// token that can be used to pick up the read operation where it left off.
	// Supply the empty token on the first call.
	//
	// At the end of the directory, the returned continuation token will be
	// empty. Otherwise it will be non-empty. There is no guarantee about the
	// number of objects returned; it may be zero even with a non-empty
	// continuation token.
	ReadObjects(
		ctx context.Context,
		tok string) (files []Core, dirs []Core, newTok string, err error)

	// Read some number of entries from the directory, returning a continuation
	// token that can be used to pick up the read operation where it left off.
	// Supply the empty token on the first call.
	//
	// At the end of the directory, the returned continuation token will be
	// empty. Otherwise it will be non-empty. There is no guarantee about the
	// number of entries returned; it may be zero even with a non-empty
	// continuation token.
	//
	// The contents of the Offset and Inode fields for returned entries is
	// undefined.
	ReadEntries(
		ctx context.Context,
		tok string) (entries []fuseutil.Dirent, newTok string, err error)

	// Create an empty child file with the supplied (relative) name, failing with
	// *gcs.PreconditionError if a backing object already exists in GCS.
	// Return the full name of the child and the GCS object it backs up.
	CreateChildFile(
		ctx context.Context,
		name string) (result Core, err error)

	// Like CreateChildFile, except clone the supplied source object instead of
	// creating an empty object.
	// Return the full name of the child and the GCS object it backs up.
	CloneToChildFile(
		ctx context.Context,
		name string,
		src *gcs.Object) (result Core, err error)

	// Create a symlink object with the supplied (relative) name and the supplied
	// target, failing with *gcs.PreconditionError if a backing object already
	// exists in GCS.
	// Return the full name of the child and the GCS object it backs up.
	CreateChildSymlink(
		ctx context.Context,
		name string,
		target string) (result Core, err error)

	// Create a backing object for a child directory with the supplied (relative)
	// name, failing with *gcs.PreconditionError if a backing object already
	// exists in GCS.
	// Return the full name of the child and the GCS object it backs up.
	CreateChildDir(
		ctx context.Context,
		name string) (result Core, err error)

	// Delete the backing object for the child file or symlink with the given
	// (relative) name and generation number, where zero means the latest
	// generation. If the object/generation doesn't exist, no error is returned.
	//
	// metaGeneration may be set to a non-nil pointer giving a meta-generation
	// precondition, but need not be.
	DeleteChildFile(
		ctx context.Context,
		name string,
		generation int64,
		metaGeneration *int64) (err error)

	// Delete the backing object for the child directory with the given
	// (relative) name.
	DeleteChildDir(
		ctx context.Context,
		name string) (err error)
}

// An inode that represents a directory from a GCS bucket.
type BucketOwnedDirInode interface {
	DirInode
	BucketOwnedInode
}

type dirInode struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	bucket     gcsx.SyncerBucket
	mtimeClock timeutil.Clock
	cacheClock timeutil.Clock

	/////////////////////////
	// Constant data
	/////////////////////////

	id           fuseops.InodeID
	implicitDirs bool

	// INVARIANT: name.IsDir()
	name Name

	attrs fuseops.InodeAttributes

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A mutex that must be held when calling certain methods. See documentation
	// for each method.
	mu locker.Locker

	// GUARDED_BY(mu)
	lc lookupCount

	// cache.CheckInvariants() does not panic.
	//
	// GUARDED_BY(mu)
	cache typeCache
}

var _ DirInode = &dirInode{}

// Create a directory inode for the name, representing the directory containing
// the objects for which it is an immediate prefix. For the root directory,
// this is the empty string.
//
// If implicitDirs is set, LookUpChild will use ListObjects to find child
// directories that are "implicitly" defined by the existence of their own
// descendents. For example, if there is an object named "foo/bar/baz" and this
// is the directory "foo", a child directory named "bar" will be implied.
//
// If typeCacheTTL is non-zero, a cache from child name to information about
// whether that name exists as a file/symlink and/or directory will be
// maintained. This may speed up calls to LookUpChild, especially when combined
// with a stat-caching GCS bucket, but comes at the cost of consistency: if the
// child is removed and recreated with a different type before the expiration,
// we may fail to find it.
//
// The initial lookup count is zero.
//
// REQUIRES: name.IsDir()
func NewDirInode(
	id fuseops.InodeID,
	name Name,
	attrs fuseops.InodeAttributes,
	implicitDirs bool,
	typeCacheTTL time.Duration,
	bucket gcsx.SyncerBucket,
	mtimeClock timeutil.Clock,
	cacheClock timeutil.Clock) (d DirInode) {

	if !name.IsDir() {
		panic(fmt.Sprintf("Unexpected name: %s", name))
	}

	// Set up the struct.
	const typeCacheCapacity = 1 << 16
	typed := &dirInode{
		bucket:       bucket,
		mtimeClock:   mtimeClock,
		cacheClock:   cacheClock,
		id:           id,
		implicitDirs: implicitDirs,
		name:         name,
		attrs:        attrs,
		cache:        newTypeCache(typeCacheCapacity/2, typeCacheTTL),
	}

	typed.lc.Init(id)

	// Set up invariant checking.
	typed.mu = locker.New(name.GcsObjectName(), typed.checkInvariants)

	d = typed
	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (d *dirInode) checkInvariants() {
	// INVARIANT: d.name.IsDir()
	if !d.name.IsDir() {
		panic(fmt.Sprintf("Unexpected name: %s", d.name))
	}

	// cache.CheckInvariants() does not panic.
	d.cache.CheckInvariants()
}

func (d *dirInode) lookUpChildFile(
	ctx context.Context,
	name string) (result Core, err error) {
	result.Bucket = d.Bucket()
	result.FullName = NewFileName(d.Name(), name)
	result.Object, err = statObjectMayNotExist(ctx, d.bucket, result.FullName)
	if err != nil {
		err = fmt.Errorf("statObjectMayNotExist: %w", err)
		return
	}

	return
}

func (d *dirInode) lookUpChildDir(
	ctx context.Context,
	dirName string) (result Core, err error) {
	childName := NewDirName(d.Name(), dirName)

	// Stat the placeholder object.
	result.Bucket = d.Bucket()
	result.FullName = childName
	result.Object, err = statObjectMayNotExist(ctx, d.bucket, result.FullName)
	if err != nil {
		err = fmt.Errorf("statObjectMayNotExist: %w", err)
		return
	}
	if result.Object != nil {
		// Core found. This is an explicit directory.
		return
	}

	// If implicit directories are enabled, find out whether the child name is
	// implicitly defined.
	if d.implicitDirs {
		result.ImplicitDir, err = objectNamePrefixNonEmpty(
			ctx,
			d.bucket,
			childName.GcsObjectName())

		if err != nil {
			err = fmt.Errorf("objectNamePrefixNonEmpty: %w", err)
			return
		}
	}
	return
}

// Look up the file for a (file, dir) pair with conflicting names, overriding
// the default behavior. If the file doesn't exist, return a nil record with a
// nil error. If the directory doesn't exist, pretend the file doesn't exist.
//
// REQUIRES: strings.HasSuffix(name, ConflictingFileNameSuffix)
func (d *dirInode) lookUpConflicting(
	ctx context.Context,
	name string) (result Core, err error) {
	strippedName := strings.TrimSuffix(name, ConflictingFileNameSuffix)

	// In order to a marked name to be accepted, we require the conflicting
	// directory to exist.
	var dirResult Core
	dirResult, err = d.lookUpChildDir(ctx, strippedName)
	if err != nil {
		err = fmt.Errorf("lookUpChildDir for stripped name: %w", err)
		return
	}

	if !dirResult.Exists() {
		return
	}

	// The directory name exists. Find the conflicting file.
	result, err = d.lookUpChildFile(ctx, strippedName)
	if err != nil {
		err = fmt.Errorf("lookUpChildFile for stripped name: %w", err)
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
		err = fmt.Errorf("ListObjects: %w", err)
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
	name Name) (o *gcs.Object, err error) {
	// Call the bucket.
	req := &gcs.StatObjectRequest{
		Name: name.GcsObjectName(),
	}

	o, err = bucket.StatObject(ctx, req)

	// Suppress "not found" errors.
	if _, ok := err.(*gcs.NotFoundError); ok {
		err = nil
	}

	// Annotate others.
	if err != nil {
		err = fmt.Errorf("StatObject: %w", err)
		return
	}

	return
}

// Fail if the name already exists. Pass on errors directly.
func (d *dirInode) createNewObject(
	ctx context.Context,
	name Name,
	metadata map[string]string) (o *gcs.Object, err error) {
	// Create an empty backing object for the child, failing if it already
	// exists.
	var precond int64
	createReq := &gcs.CreateObjectRequest{
		Name:                   name.GcsObjectName(),
		Contents:               strings.NewReader(""),
		GenerationPrecondition: &precond,
		Metadata:               metadata,
	}

	o, err = d.bucket.CreateObject(ctx, createReq)
	if err != nil {
		return
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (d *dirInode) Lock() {
	d.mu.Lock()
}

func (d *dirInode) Unlock() {
	d.mu.Unlock()
}

func (d *dirInode) ID() fuseops.InodeID {
	return d.id
}

func (d *dirInode) Name() Name {
	return d.name
}

// LOCKS_REQUIRED(d)
func (d *dirInode) IncrementLookupCount() {
	d.lc.Inc()
}

// LOCKS_REQUIRED(d)
func (d *dirInode) DecrementLookupCount(n uint64) (destroy bool) {
	destroy = d.lc.Dec(n)
	return
}

// LOCKS_REQUIRED(d)
func (d *dirInode) Destroy() (err error) {
	// Nothing interesting to do.
	return
}

// LOCKS_REQUIRED(d)
func (d *dirInode) Attributes(
	ctx context.Context) (attrs fuseops.InodeAttributes, err error) {
	// Set up basic attributes.
	attrs = d.attrs
	attrs.Nlink = 1

	return
}

func (d *dirInode) Bucket() gcsx.SyncerBucket {
	return d.bucket
}

// A suffix that can be used to unambiguously tag a file system name.
// (Unambiguous because U+000A is not allowed in GCS object names.) This is
// used to refer to the file/symlink in a (file/symlink, directory) pair with
// conflicting object names.
//
// See also the notes on DirInode.LookUpChild.
const ConflictingFileNameSuffix = "\n"

// LOCKS_REQUIRED(d)
func (d *dirInode) LookUpChild(
	ctx context.Context,
	name string) (result Core, err error) {
	// Consult the cache about the type of the child. This may save us work
	// below.
	now := d.cacheClock.Now()
	cacheSaysFile := d.cache.IsFile(now, name)
	cacheSaysDir := d.cache.IsDir(now, name)
	cacheSaysImplicitDir := d.cache.IsImplicitDir(now, name)

	// Is this a conflict marker name?
	if strings.HasSuffix(name, ConflictingFileNameSuffix) {
		result, err = d.lookUpConflicting(ctx, name)
		return
	}

	// Stat the child as a file, unless the cache has told us it's a directory
	// but not a file.
	b := syncutil.NewBundle(ctx)

	var fileResult Core
	if !(cacheSaysDir && !cacheSaysFile) {
		b.Add(func(ctx context.Context) (err error) {
			fileResult, err = d.lookUpChildFile(ctx, name)
			return
		})
	}

	// Stat the child as a directory, unless the cache has told us it's a file
	// but not a directory.
	var dirResult Core
	if !(cacheSaysFile && !cacheSaysDir) {
		if cacheSaysImplicitDir {
			dirResult = Core{
				Bucket:      d.Bucket(),
				FullName:    NewDirName(d.Name(), name),
				Object:      nil,
				ImplicitDir: true,
			}
		} else {
			b.Add(func(ctx context.Context) (err error) {
				dirResult, err = d.lookUpChildDir(ctx, name)
				return
			})
		}
	}

	// Wait for both.
	err = b.Join()
	if err != nil {
		return
	}

	// Prefer directories over files.
	switch {
	case dirResult.Exists():
		result = dirResult
	case fileResult.Exists():
		result = fileResult
	}

	// Update the cache.
	now = d.cacheClock.Now()
	if fileResult.Exists() {
		d.cache.NoteFile(now, name)
	}

	if dirResult.Exists() {
		d.cache.NoteDir(now, name)
		if dirResult.ImplicitDir {
			d.cache.NoteImplicitDir(now, name)
		}
	}

	return
}

// LOCKS_REQUIRED(d)
func (d *dirInode) ReadDescendants(
	ctx context.Context,
	limit int) (descendants map[Name]Core, err error) {
	var tok string
	var listing *gcs.Listing
	descendants = make(map[Name]Core)
	for {
		listing, err = d.bucket.ListObjects(ctx, &gcs.ListObjectsRequest{
			Delimiter:         "", // recursively
			Prefix:            d.Name().GcsObjectName(),
			ContinuationToken: tok,
			MaxResults:        limit + 1, // to exclude itself
		})
		if err != nil {
			err = fmt.Errorf("list objects: %w", err)
			return
		}

		for _, o := range listing.Objects {
			if len(descendants) >= limit {
				return
			}
			// skip the current directory
			if o.Name == d.Name().GcsObjectName() {
				continue
			}
			name := NewDescendantName(d.Name(), o.Name)
			descendants[name] = Core{
				Bucket:      d.Bucket(),
				FullName:    name,
				Object:      o,
				ImplicitDir: false,
			}
		}

		// Are we done listing?
		if tok = listing.ContinuationToken; tok == "" {
			return
		}
	}

}

// LOCKS_REQUIRED(d)
func (d *dirInode) ReadObjects(
	ctx context.Context,
	tok string) (files []Core, dirs []Core, newTok string, err error) {
	// Ask the bucket to list some objects.
	req := &gcs.ListObjectsRequest{
		Delimiter:                "/",
		IncludeTrailingDelimiter: true,
		Prefix:                   d.Name().GcsObjectName(),
		ContinuationToken:        tok,
	}

	listing, err := d.bucket.ListObjects(ctx, req)
	if err != nil {
		err = fmt.Errorf("ListObjects: %w", err)
		return
	}
	now := d.cacheClock.Now()

	explicitDirs := make(map[string]bool)
	for _, o := range listing.Objects {
		// Skip empty results or the directory object backing this inode.
		if o.Name == d.Name().GcsObjectName() || o.Name == "" {
			continue
		}

		nameBase := path.Base(o.Name) // ie. "bar" from "foo/bar/" or "foo/bar"
		if strings.HasSuffix(o.Name, "/") {
			explicitDir := Core{
				Bucket:      d.Bucket(),
				FullName:    NewDirName(d.Name(), nameBase),
				Object:      o,
				ImplicitDir: false,
			}
			dirs = append(dirs, explicitDir)
			explicitDirs[nameBase] = true
			d.cache.NoteDir(now, nameBase)
		} else {
			file := Core{
				Bucket:      d.Bucket(),
				FullName:    NewFileName(d.Name(), nameBase),
				Object:      o,
				ImplicitDir: false,
			}
			files = append(files, file)
			d.cache.NoteFile(now, nameBase)
		}
	}

	// Return an appropriate continuation token, if any.
	newTok = listing.ContinuationToken

	if !d.implicitDirs {
		return
	}

	// Add implicit directories into the result.
	for _, p := range listing.CollapsedRuns {
		pathBase := path.Base(p) // ie. "bar" from "foo/bar/"

		if _, ok := explicitDirs[pathBase]; ok {
			continue
		}

		implicitDir := Core{
			Bucket:      d.Bucket(),
			FullName:    NewDirName(d.Name(), pathBase),
			Object:      nil,
			ImplicitDir: true,
		}
		dirs = append(dirs, implicitDir)
		d.cache.NoteDir(now, pathBase)
		d.cache.NoteImplicitDir(now, pathBase)
	}

	return
}

// LOCKS_REQUIRED(d)
func (d *dirInode) ReadEntries(
	ctx context.Context,
	tok string) (entries []fuseutil.Dirent, newTok string, err error) {
	var files, dirs []Core
	files, dirs, newTok, err = d.ReadObjects(ctx, tok)
	if err != nil {
		err = fmt.Errorf("read objects: %w", err)
		return
	}

	for _, file := range files {
		entryType := fuseutil.DT_File
		if IsSymlink(file.Object) {
			entryType = fuseutil.DT_Link
		}
		entries = append(entries, fuseutil.Dirent{
			Name: path.Base(file.FullName.LocalName()),
			Type: entryType,
		})
	}
	for _, dir := range dirs {
		entries = append(entries, fuseutil.Dirent{
			Name: path.Base(dir.FullName.LocalName()),
			Type: fuseutil.DT_Directory,
		})
	}
	return
}

// LOCKS_REQUIRED(d)
func (d *dirInode) CreateChildFile(
	ctx context.Context,
	name string) (result Core, err error) {
	result.Bucket = d.Bucket()
	metadata := map[string]string{
		FileMtimeMetadataKey: d.mtimeClock.Now().UTC().Format(time.RFC3339Nano),
	}
	result.FullName = NewFileName(d.Name(), name)

	result.Object, err = d.createNewObject(ctx, result.FullName, metadata)
	if err != nil {
		return
	}

	d.cache.NoteFile(d.cacheClock.Now(), name)

	return
}

// LOCKS_REQUIRED(d)
func (d *dirInode) CloneToChildFile(
	ctx context.Context,
	name string,
	src *gcs.Object) (result Core, err error) {
	result.Bucket = d.Bucket()
	// Erase any existing type information for this name.
	d.cache.Erase(name)
	result.FullName = NewFileName(d.Name(), name)

	// Clone over anything that might already exist for the name.
	result.Object, err = d.bucket.CopyObject(
		ctx,
		&gcs.CopyObjectRequest{
			SrcName:                       src.Name,
			SrcGeneration:                 src.Generation,
			SrcMetaGenerationPrecondition: &src.MetaGeneration,
			DstName:                       result.FullName.GcsObjectName(),
		})

	if err != nil {
		return
	}

	// Update the type cache.
	d.cache.NoteFile(d.cacheClock.Now(), name)

	return
}

// LOCKS_REQUIRED(d)
func (d *dirInode) CreateChildSymlink(
	ctx context.Context,
	name string,
	target string) (result Core, err error) {
	result.Bucket = d.Bucket()
	result.FullName = NewFileName(d.Name(), name)
	metadata := map[string]string{
		SymlinkMetadataKey: target,
	}

	result.Object, err = d.createNewObject(ctx, result.FullName, metadata)
	if err != nil {
		return
	}

	d.cache.NoteFile(d.cacheClock.Now(), name)

	return
}

// LOCKS_REQUIRED(d)
func (d *dirInode) CreateChildDir(
	ctx context.Context,
	name string) (result Core, err error) {
	result.Bucket = d.Bucket()
	result.FullName = NewDirName(d.Name(), name)

	result.Object, err = d.createNewObject(ctx, result.FullName, nil)
	if err != nil {
		return
	}

	d.cache.NoteDir(d.cacheClock.Now(), name)

	return
}

// LOCKS_REQUIRED(d)
func (d *dirInode) DeleteChildFile(
	ctx context.Context,
	name string,
	generation int64,
	metaGeneration *int64) (err error) {
	d.cache.Erase(name)
	childName := NewFileName(d.Name(), name)

	err = d.bucket.DeleteObject(
		ctx,
		&gcs.DeleteObjectRequest{
			Name:                       childName.GcsObjectName(),
			Generation:                 generation,
			MetaGenerationPrecondition: metaGeneration,
		})

	if err != nil {
		err = fmt.Errorf("DeleteObject: %w", err)
		return
	}

	return
}

// LOCKS_REQUIRED(d)
func (d *dirInode) DeleteChildDir(
	ctx context.Context,
	name string) (err error) {
	d.cache.Erase(name)
	childName := NewDirName(d.Name(), name)

	// Delete the backing object. Unfortunately we have no way to precondition
	// this on the directory being empty.
	err = d.bucket.DeleteObject(
		ctx,
		&gcs.DeleteObjectRequest{
			Name: childName.GcsObjectName(),
		})

	if err != nil {
		err = fmt.Errorf("DeleteObject: %w", err)
		return
	}

	return
}
