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
	"errors"
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

// ListObjects call supports fetching upto 5000 results when projection is noAcl
// via maxResults param in one call. When projection is set to full, it returns
// 2000 results max. In GcsFuse flows we will be setting projection as noAcl.
// By default 1000 results are returned if maxResults is not set.
// Defining a constant to set maxResults param.
const MaxResultsForListObjectsCall = 5000

// An inode representing a directory, with facilities for listing entries,
// looking up children, and creating and deleting children. Must be locked for
// any method additional to the Inode interface.
type DirInode interface {
	Inode

	// Look up the direct child with the given relative name, returning
	// information about the object backing the child or whether it exists as an
	// implicit directory. If a file/symlink and a directory with the given name
	// both exist, the directory is preferred. Return nil result and a nil error
	// if neither is found.
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
	LookUpChild(ctx context.Context, name string) (*Core, error)

	// Read the children objects of this dir, recursively. The result count
	// is capped at the given limit. Internal caches are not refreshed from this
	// call.
	ReadDescendants(ctx context.Context, limit int) (map[Name]*Core, error)

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
	CreateChildFile(ctx context.Context, name string) (*Core, error)

	// Like CreateChildFile, except clone the supplied source object instead of
	// creating an empty object.
	// Return the full name of the child and the GCS object it backs up.
	CloneToChildFile(ctx context.Context, name string, src *gcs.Object) (*Core, error)

	// Create a symlink object with the supplied (relative) name and the supplied
	// target, failing with *gcs.PreconditionError if a backing object already
	// exists in GCS.
	// Return the full name of the child and the GCS object it backs up.
	CreateChildSymlink(ctx context.Context, name string, target string) (*Core, error)

	// Create a backing object for a child directory with the supplied (relative)
	// name, failing with *gcs.PreconditionError if a backing object already
	// exists in GCS.
	// Return the full name of the child and the GCS object it backs up.
	CreateChildDir(ctx context.Context, name string) (*Core, error)

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

	bucket     *gcsx.SyncerBucket
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
	bucket *gcsx.SyncerBucket,
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

func (d *dirInode) lookUpChildFile(ctx context.Context, name string) (*Core, error) {
	return findExplicitInode(ctx, d.Bucket(), NewFileName(d.Name(), name))
}

func (d *dirInode) lookUpChildDir(ctx context.Context, name string) (*Core, error) {
	childName := NewDirName(d.Name(), name)
	if d.implicitDirs {
		return findDirInode(ctx, d.Bucket(), childName)
	}
	return findExplicitInode(ctx, d.Bucket(), childName)
}

// Look up the file for a (file, dir) pair with conflicting names, overriding
// the default behavior. If the file doesn't exist, return a nil record with a
// nil error. If the directory doesn't exist, pretend the file doesn't exist.
//
// REQUIRES: strings.HasSuffix(name, ConflictingFileNameSuffix)
func (d *dirInode) lookUpConflicting(ctx context.Context, name string) (*Core, error) {
	strippedName := strings.TrimSuffix(name, ConflictingFileNameSuffix)

	// In order to a marked name to be accepted, we require the conflicting
	// directory to exist.
	result, err := d.lookUpChildDir(ctx, strippedName)
	if err != nil {
		return nil, fmt.Errorf("lookUpChildDir for stripped name: %w", err)
	}

	if result == nil {
		return nil, nil
	}

	// The directory name exists. Find the conflicting file.
	// Overwrite the result.
	result, err = d.lookUpChildFile(ctx, strippedName)
	if err != nil {
		return nil, fmt.Errorf("lookUpChildFile for stripped name: %w", err)
	}

	return result, nil
}

// findExplicitInode finds the file or dir inode core backed by an explicit
// object in GCS with the given name. Return nil if such object does not exist.
func findExplicitInode(ctx context.Context, bucket *gcsx.SyncerBucket, name Name) (*Core, error) {
	// Call the bucket.
	req := &gcs.StatObjectRequest{
		Name: name.GcsObjectName(),
	}

	o, err := bucket.StatObject(ctx, req)

	// Suppress "not found" errors.
	var gcsErr *gcs.NotFoundError
	if errors.As(err, &gcsErr) {
		return nil, nil
	}

	// Annotate others.
	if err != nil {
		return nil, fmt.Errorf("StatObject: %w", err)
	}

	return &Core{
		Bucket:   bucket,
		FullName: name,
		Object:   o,
	}, nil
}

// findDirInode finds the dir inode core where the directory is either explicit
// or implicit. Returns nil if no such directory exists.
func findDirInode(ctx context.Context, bucket *gcsx.SyncerBucket, name Name) (*Core, error) {
	if !name.IsDir() {
		return nil, fmt.Errorf("%q is not directory", name)
	}

	req := &gcs.ListObjectsRequest{
		Prefix:     name.GcsObjectName(),
		MaxResults: 1,
	}
	listing, err := bucket.ListObjects(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list objects: %w", err)
	}

	if len(listing.Objects) == 0 {
		return nil, nil
	}

	result := &Core{
		Bucket:   bucket,
		FullName: name,
	}
	if o := listing.Objects[0]; o.Name == name.GcsObjectName() {
		result.Object = o
	}
	return result, nil
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

func (d *dirInode) Bucket() *gcsx.SyncerBucket {
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
func (d *dirInode) LookUpChild(ctx context.Context, name string) (*Core, error) {
	// Is this a conflict marker name?
	if strings.HasSuffix(name, ConflictingFileNameSuffix) {
		return d.lookUpConflicting(ctx, name)
	}

	var fileResult *Core
	var dirResult *Core
	lookUpFile := func(ctx context.Context) (err error) {
		fileResult, err = findExplicitInode(ctx, d.Bucket(), NewFileName(d.Name(), name))
		return
	}
	lookUpExplicitDir := func(ctx context.Context) (err error) {
		dirResult, err = findExplicitInode(ctx, d.Bucket(), NewDirName(d.Name(), name))
		return
	}
	lookUpImplicitOrExplicitDir := func(ctx context.Context) (err error) {
		dirResult, err = findDirInode(ctx, d.Bucket(), NewDirName(d.Name(), name))
		return
	}

	b := syncutil.NewBundle(ctx)
	switch cachedType := d.cache.Get(d.cacheClock.Now(), name); cachedType {
	case ImplicitDirType:
		dirResult = &Core{
			Bucket:   d.Bucket(),
			FullName: NewDirName(d.Name(), name),
			Object:   nil,
		}
	case ExplicitDirType:
		b.Add(lookUpExplicitDir)
	case RegularFileType, SymlinkType:
		b.Add(lookUpFile)
	case UnknownType:
		b.Add(lookUpFile)
		if d.implicitDirs {
			b.Add(lookUpImplicitOrExplicitDir)
		} else {
			b.Add(lookUpExplicitDir)
		}
	}

	if err := b.Join(); err != nil {
		return nil, err
	}

	var result *Core
	if dirResult != nil {
		result = dirResult
	} else if fileResult != nil {
		result = fileResult
	}

	if result != nil {
		d.cache.Insert(d.cacheClock.Now(), name, result.Type())
	}
	return result, nil
}

// LOCKS_REQUIRED(d)
func (d *dirInode) ReadDescendants(ctx context.Context, limit int) (map[Name]*Core, error) {
	var tok string
	descendants := make(map[Name]*Core)
	for {
		listing, err := d.bucket.ListObjects(ctx, &gcs.ListObjectsRequest{
			Delimiter:         "", // recursively
			Prefix:            d.Name().GcsObjectName(),
			ContinuationToken: tok,
			MaxResults:        limit + 1, // to exclude itself
		})
		if err != nil {
			return nil, fmt.Errorf("list objects: %w", err)
		}

		for _, o := range listing.Objects {
			if len(descendants) >= limit {
				return descendants, nil
			}
			// skip the current directory
			if o.Name == d.Name().GcsObjectName() {
				continue
			}
			name := NewDescendantName(d.Name(), o.Name)
			descendants[name] = &Core{
				Bucket:   d.Bucket(),
				FullName: name,
				Object:   o,
			}
		}

		// Are we done listing?
		if tok = listing.ContinuationToken; tok == "" {
			return descendants, nil
		}
	}

}

// LOCKS_REQUIRED(d)
func (d *dirInode) readObjects(
	ctx context.Context,
	tok string) (cores map[Name]*Core, newTok string, err error) {
	// Ask the bucket to list some objects.
	req := &gcs.ListObjectsRequest{
		Delimiter:                "/",
		IncludeTrailingDelimiter: true,
		Prefix:                   d.Name().GcsObjectName(),
		ContinuationToken:        tok,
		MaxResults:               MaxResultsForListObjectsCall,
		// Setting Projection param to noAcl since fetching owner and acls are not
		// required.
		ProjectionVal: gcs.NoAcl,
	}

	listing, err := d.bucket.ListObjects(ctx, req)
	if err != nil {
		err = fmt.Errorf("ListObjects: %w", err)
		return
	}

	cores = make(map[Name]*Core)
	defer func() {
		now := d.cacheClock.Now()
		for fullName, c := range cores {
			d.cache.Insert(now, path.Base(fullName.LocalName()), c.Type())
		}
	}()

	for _, o := range listing.Objects {
		// Skip empty results or the directory object backing this inode.
		if o.Name == d.Name().GcsObjectName() || o.Name == "" {
			continue
		}

		nameBase := path.Base(o.Name) // ie. "bar" from "foo/bar/" or "foo/bar"

		// Given the alphabetical order of the objects, if a file "foo" and
		// directory "foo/" coexist, the directory would eventually occupy
		// the value of records["foo"].
		if strings.HasSuffix(o.Name, "/") {
			dirName := NewDirName(d.Name(), nameBase)
			explicitDir := &Core{
				Bucket:   d.Bucket(),
				FullName: dirName,
				Object:   o,
			}
			cores[dirName] = explicitDir
		} else {
			fileName := NewFileName(d.Name(), nameBase)
			file := &Core{
				Bucket:   d.Bucket(),
				FullName: fileName,
				Object:   o,
			}
			cores[fileName] = file
		}
	}

	// Return an appropriate continuation token, if any.
	newTok = listing.ContinuationToken

	if !d.implicitDirs {
		return
	}

	// Add implicit directories into the result.
	for _, p := range listing.CollapsedRuns {
		pathBase := path.Base(p)
		dirName := NewDirName(d.Name(), pathBase)
		if c, ok := cores[dirName]; ok && c.Type() == ExplicitDirType {
			continue
		}

		implicitDir := &Core{
			Bucket:   d.Bucket(),
			FullName: dirName,
			Object:   nil,
		}
		cores[dirName] = implicitDir
	}
	return
}

func (d *dirInode) ReadEntries(
	ctx context.Context,
	tok string) (entries []fuseutil.Dirent, newTok string, err error) {
	var cores map[Name]*Core
	cores, newTok, err = d.readObjects(ctx, tok)
	if err != nil {
		err = fmt.Errorf("read objects: %w", err)
		return
	}

	for fullName, core := range cores {
		entry := fuseutil.Dirent{
			Name: path.Base(fullName.LocalName()),
			Type: fuseutil.DT_Unknown,
		}
		switch core.Type() {
		case SymlinkType:
			entry.Type = fuseutil.DT_Link
		case RegularFileType:
			entry.Type = fuseutil.DT_File
		case ImplicitDirType, ExplicitDirType:
			entry.Type = fuseutil.DT_Directory
		}
		entries = append(entries, entry)
	}
	return
}

// LOCKS_REQUIRED(d)
func (d *dirInode) CreateChildFile(ctx context.Context, name string) (*Core, error) {
	metadata := map[string]string{
		FileMtimeMetadataKey: d.mtimeClock.Now().UTC().Format(time.RFC3339Nano),
	}
	fullName := NewFileName(d.Name(), name)

	o, err := d.createNewObject(ctx, fullName, metadata)
	if err != nil {
		return nil, err
	}

	d.cache.Insert(d.cacheClock.Now(), name, RegularFileType)
	return &Core{
		Bucket:   d.Bucket(),
		FullName: fullName,
		Object:   o,
	}, nil
}

// LOCKS_REQUIRED(d)
func (d *dirInode) CloneToChildFile(ctx context.Context, name string, src *gcs.Object) (*Core, error) {
	// Erase any existing type information for this name.
	d.cache.Erase(name)
	fullName := NewFileName(d.Name(), name)

	// Clone over anything that might already exist for the name.
	o, err := d.bucket.CopyObject(
		ctx,
		&gcs.CopyObjectRequest{
			SrcName:                       src.Name,
			SrcGeneration:                 src.Generation,
			SrcMetaGenerationPrecondition: &src.MetaGeneration,
			DstName:                       fullName.GcsObjectName(),
		})
	if err != nil {
		return nil, err
	}

	c := &Core{
		Bucket:   d.Bucket(),
		FullName: fullName,
		Object:   o,
	}
	d.cache.Insert(d.cacheClock.Now(), name, c.Type())
	return c, nil
}

// LOCKS_REQUIRED(d)
func (d *dirInode) CreateChildSymlink(ctx context.Context, name string, target string) (*Core, error) {
	fullName := NewFileName(d.Name(), name)
	metadata := map[string]string{
		SymlinkMetadataKey: target,
	}

	o, err := d.createNewObject(ctx, fullName, metadata)
	if err != nil {
		return nil, err
	}

	d.cache.Insert(d.cacheClock.Now(), name, SymlinkType)

	return &Core{
		Bucket:   d.Bucket(),
		FullName: fullName,
		Object:   o,
	}, nil
}

// LOCKS_REQUIRED(d)
func (d *dirInode) CreateChildDir(ctx context.Context, name string) (*Core, error) {
	fullName := NewDirName(d.Name(), name)
	o, err := d.createNewObject(ctx, fullName, nil)
	if err != nil {
		return nil, err
	}

	d.cache.Insert(d.cacheClock.Now(), name, ExplicitDirType)

	return &Core{
		Bucket:   d.Bucket(),
		FullName: fullName,
		Object:   o,
	}, nil
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
	d.cache.Erase(name)

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
	d.cache.Erase(name)

	return
}
