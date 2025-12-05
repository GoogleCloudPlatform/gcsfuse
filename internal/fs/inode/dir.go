// Copyright 2015 Google LLC
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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
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

	// Rename the file.
	RenameFile(ctx context.Context, fileToRename *gcs.MinObject, destinationFileName string) (*gcs.Object, error)

	// Rename the directiory/folder.
	RenameFolder(ctx context.Context, folderName string, destinationFolderId string) (*gcs.Folder, error)

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
		tok string) (entries []fuseutil.Dirent, unsupportedPaths []string, newTok string, err error)

	// ReadEntryCores reads a batch of directory entries and returns them as a
	// map of `inode.Core` objects along with a continuation token that can be
	// used to pick up the read operation where it left off.
	// Supply the empty token on the first call.
	//
	// At the end of the directory, the returned continuation token will be
	// empty. Otherwise it will be non-empty. There is no guarantee about the
	// number of entries returned; it may be zero even with a non-empty
	// continuation token.
	ReadEntryCores(ctx context.Context, tok string) (cores map[Name]*Core, unsupportedPaths []string, newTok string, err error)

	// Create an empty child file with the supplied (relative) name, failing with
	// *gcs.PreconditionError if a backing object already exists in GCS.
	// Return the full name of the child and the GCS object it backs up.
	CreateChildFile(ctx context.Context, name string) (*Core, error)

	// CreateLocalChildFileCore returns an empty local child file core.
	CreateLocalChildFileCore(name string) (Core, error)

	// Like CreateChildFile, except clone the supplied source object instead of
	// creating an empty object.
	// Return the full name of the child and the GCS object it backs up.
	CloneToChildFile(ctx context.Context, name string, src *gcs.MinObject) (*Core, error)

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
	// (relative) name if it is not an Implicit Directory.
	DeleteChildDir(
		ctx context.Context,
		name string,
		isImplicitDir bool,
		dirInode DirInode) (err error)

	// DeleteObjects recursively deletes the given objects and prefixes.
	DeleteObjects(ctx context.Context, objectNames []string) error

	// LocalFileEntries lists the local files present in the directory.
	// Local means that the file is not yet present on GCS.
	LocalFileEntries(localFileInodes map[Name]Inode) (localEntries map[string]fuseutil.Dirent)

	// LockForChildLookup takes appropriate kind of lock when an inode's child is
	// looked up.
	LockForChildLookup()

	// UnlockForChildLookup unlocks the lock taken with LockForChildLookup.
	UnlockForChildLookup()

	// ShouldInvalidateKernelListCache tells the filesystem whether kernel list-cache
	// should be invalidated or not.
	ShouldInvalidateKernelListCache(ttl time.Duration) bool

	// InvalidateKernelListCache guarantees that the subsequent list call will be
	// served from GCSFuse.
	InvalidateKernelListCache()

	// RLock readonly lock.
	RLock()

	// RUnlock readonly unlock.
	RUnlock()

	IsUnlinked() bool

	Unlink()
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

	id                       fuseops.InodeID
	implicitDirs             bool
	includeFoldersAsPrefixes bool

	enableNonexistentTypeCache bool

	// INVARIANT: name.IsDir()
	name Name

	attrs fuseops.InodeAttributes

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A RW mutex that must be held when calling certain methods. See
	// documentation for each method.
	mu locker.RWLocker

	// GUARDED_BY(mu)
	lc lookupCount

	// prevDirListingTimeStamp is the time stamp of previous listing when user asked
	// (via kernel) the directory listing from the filesystem.
	// Specially used when kernelListCacheTTL > 0 that means kernel list-cache is
	// enabled.
	prevDirListingTimeStamp time.Time
	isHNSEnabled            bool

	isUnsupportedPathSupportEnabled bool

	// Represents if folder has been unlinked in hierarchical bucket. This is not getting used in
	// non-hierarchical bucket.
	unlinked bool
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
	includeFoldersAsPrefixes bool,
	enableNonexistentTypeCache bool,
	bucket *gcsx.SyncerBucket,
	mtimeClock timeutil.Clock,
	cacheClock timeutil.Clock,
	isHNSEnabled bool,
	isUnsupportedPathSupportEnabled bool,
) (d DirInode) {

	if !name.IsDir() {
		panic(fmt.Sprintf("Unexpected name: %s", name))
	}

	typed := &dirInode{
		bucket:                          bucket,
		mtimeClock:                      mtimeClock,
		cacheClock:                      cacheClock,
		id:                              id,
		implicitDirs:                    implicitDirs,
		includeFoldersAsPrefixes:        includeFoldersAsPrefixes,
		enableNonexistentTypeCache:      enableNonexistentTypeCache,
		name:                            name,
		attrs:                           attrs,
		isHNSEnabled:                    isHNSEnabled,
		isUnsupportedPathSupportEnabled: isUnsupportedPathSupportEnabled,
		unlinked:                        false,
	}

	typed.lc.Init(id)

	// Set up invariant checking.
	typed.mu = locker.NewRW(name.GcsObjectName(), typed.checkInvariants)

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
}

func (d *dirInode) lookUpChildFile(ctx context.Context, name string) (*Core, error) {
	return findExplicitInode(ctx, d.Bucket(), NewFileName(d.Name(), name), true)
}

func (d *dirInode) lookUpChildDir(ctx context.Context, name string) (*Core, error) {
	childName := NewDirName(d.Name(), name)
	if d.isBucketHierarchical() {
		return findExplicitFolder(ctx, d.Bucket(), childName, true)
	}

	if d.implicitDirs {
		return findDirInode(ctx, d.Bucket(), childName, true)
	}
	return findExplicitInode(ctx, d.Bucket(), childName, true)
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
func findExplicitInode(ctx context.Context, bucket *gcsx.SyncerBucket, name Name, forceFetchFromCache bool) (*Core, error) {
	// Call the bucket.
	req := &gcs.StatObjectRequest{
		Name:                name.GcsObjectName(),
		ForceFetchFromCache: forceFetchFromCache,
	}

	m, _, err := bucket.StatObject(ctx, req)
	var notFoundErr *gcs.NotFoundError
	if errors.As(err, &notFoundErr) {
		return nil, nil
	}

	// Annotate others.
	if err != nil {
		return nil, fmt.Errorf("StatObject: %w", err)
	}

	return &Core{
		Bucket:    bucket,
		FullName:  name,
		MinObject: m,
	}, nil
}

func findExplicitFolder(ctx context.Context, bucket *gcsx.SyncerBucket, name Name, forceFetchFromCache bool) (*Core, error) {
	req := &gcs.GetFolderRequest{
		Name:                name.GcsObjectName(),
		ForceFetchFromCache: forceFetchFromCache,
	}
	folder, err := bucket.GetFolder(ctx, req)
	var notFoundErr *gcs.NotFoundError
	if errors.As(err, &notFoundErr) {
		return nil, nil
	}

	// Annotate others.
	if folder == nil {
		return nil, fmt.Errorf("error in get folder for lookup : %w", err)
	}

	return &Core{
		Bucket:   bucket,
		FullName: name,
		Folder:   folder,
	}, nil
}

// findDirInode finds the dir inode core where the directory is either explicit
// or implicit. Returns nil if no such directory exists.
func findDirInode(ctx context.Context, bucket *gcsx.SyncerBucket, name Name, forceFetchFromCache bool) (*Core, error) {
	if !name.IsDir() {
		return nil, fmt.Errorf("%q is not directory", name)
	}

	req := &gcs.ListObjectsRequest{
		Prefix:              name.GcsObjectName(),
		MaxResults:          1,
		ForceFetchFromCache: forceFetchFromCache,
	}
	listing, err := bucket.ListObjects(ctx, req)
	var notFoundErr *gcs.NotFoundError
	if errors.As(err, &notFoundErr) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if len(listing.MinObjects) == 0 && len(listing.CollapsedRuns) == 0 {
		return nil, nil
	}

	result := &Core{
		Bucket:   bucket,
		FullName: name,
	}

	if len(listing.MinObjects) != 0 {
		if o := listing.MinObjects[0]; o.Name == name.GcsObjectName() {
			result.MinObject = o
		}
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

func (d *dirInode) RLock() {
	d.mu.RLock()
}

func (d *dirInode) RUnlock() {
	d.mu.RUnlock()
}

// LockForChildLookup takes read-only lock on inode when the inode's child is
// looked up. It is safe to take read-only lock to allow parallel lookups of
// children because (a) during lookup, GCS is only read (list/stat), so as long
// as GCS is not changed remotely, lookup will be consistent (b) all the other
// directory level operations (read or write type) take exclusive locks.
func (d *dirInode) LockForChildLookup() {
	d.mu.RLock()
}

func (d *dirInode) UnlockForChildLookup() {
	d.mu.RUnlock()
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
	ctx context.Context, clobberedCheck bool) (attrs fuseops.InodeAttributes, err error) {
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

func (d *dirInode) LookUpChild(ctx context.Context, name string) (*Core, error) {
	// 1. Check for conflict marker name
	if strings.HasSuffix(name, ConflictingFileNameSuffix) {
		return d.lookUpConflicting(ctx, name)
	}

	// Define result variables outside to be updated by concurrent lookups
	var fileResult *Core
	var dirResult *Core
	var err error

	// Define the inner lookup functions as closures (necessary for context)
	lookUpFile := func(forceFetchFromCache bool) (err error) {
		fileResult, err = findExplicitInode(ctx, d.Bucket(), NewFileName(d.Name(), name), forceFetchFromCache)
		return
	}
	lookUpExplicitDir := func(forceFetchFromCache bool) (err error) {
		dirResult, err = findExplicitInode(ctx, d.Bucket(), NewDirName(d.Name(), name), forceFetchFromCache)
		return
	}
	lookUpImplicitOrExplicitDir := func(forceFetchFromCache bool) (err error) {
		dirResult, err = findDirInode(ctx, d.Bucket(), NewDirName(d.Name(), name), forceFetchFromCache)
		return
	}
	lookUpHNSDir := func(forceFetchFromCache bool) (err error) {
		dirResult, err = findExplicitFolder(ctx, d.Bucket(), NewDirName(d.Name(), name), forceFetchFromCache)
		return
	}

	fileResult, err = findExplicitInode(ctx, d.Bucket(), NewFileName(d.Name(), name), true)
	// Suppress "not found" errors.
	var gcsErr *gcs.NotFoundCacheError
	if err != nil && !errors.As(err, &gcsErr) {
		return nil, fmt.Errorf("lookUpFile: %w", err)
	}

	if d.isBucketHierarchical() {
		dirResult, err = findExplicitFolder(ctx, d.Bucket(), NewDirName(d.Name(), name), true)
	} else {
		if d.implicitDirs {
			dirResult, err = findDirInode(ctx, d.Bucket(), NewDirName(d.Name(), name), true)
		} else {
			dirResult, err = findExplicitInode(ctx, d.Bucket(), NewDirName(d.Name(), name), true)
		}
	}
	if err != nil && !errors.As(err, &gcsErr) {
		return nil, fmt.Errorf("lookUpdir: %w", err)
	}
	var result *Core
	if dirResult != nil {
		result = dirResult
	} else if fileResult != nil {
		result = fileResult
	}

	if result != nil {
		return result, nil
	}

	// Always create a fresh errgroup for each phase (cache-check or force-fetch)
	group, _ := errgroup.WithContext(ctx)

	// Always look up the file path concurrently
	group.Go(func() error {
		return lookUpFile(false)
	})

	// Determine which directory lookup to run based on bucket configuration
	if d.isBucketHierarchical() {
		group.Go(func() error {
			return lookUpHNSDir(false)
		})
	} else {
		if d.implicitDirs {
			group.Go(func() error {
				return lookUpImplicitOrExplicitDir(false)
			})
		} else {
			group.Go(func() error {
				return lookUpExplicitDir(false)
			})
		}
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}

	// 4. Consolidate and return the result
	if dirResult != nil {
		result = dirResult
	} else if fileResult != nil {
		result = fileResult
	}

	// If both are nil, the function returns nil, nil (not found).
	return result, nil
}

func (d *dirInode) IsUnlinked() bool {
	return d.unlinked
}

func (d *dirInode) Unlink() {
	d.unlinked = true
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

		for _, o := range listing.MinObjects {
			if len(descendants) >= limit {
				return descendants, nil
			}
			// skip the current directory
			if o.Name == d.Name().GcsObjectName() {
				continue
			}
			name := NewDescendantName(d.Name(), o.Name)
			descendants[name] = &Core{
				Bucket:    d.Bucket(),
				FullName:  name,
				MinObject: o,
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
	tok string) (cores map[Name]*Core, unsupportedPaths []string, newTok string, err error) {
	if d.isBucketHierarchical() {
		d.includeFoldersAsPrefixes = true
	}
	// Ask the bucket to list some objects.
	req := &gcs.ListObjectsRequest{
		Delimiter:                "/",
		IncludeTrailingDelimiter: true,
		Prefix:                   d.Name().GcsObjectName(),
		ContinuationToken:        tok,
		MaxResults:               MaxResultsForListObjectsCall,
		// Setting Projection param to noAcl since fetching owner and acls are not
		// required.
		ProjectionVal:            gcs.NoAcl,
		IncludeFoldersAsPrefixes: d.includeFoldersAsPrefixes,
		ForceFetchFromCache:      false,
	}

	listing, err := d.bucket.ListObjects(ctx, req)
	if err != nil {
		err = fmt.Errorf("ListObjects: %w", err)
		return
	}

	cores = make(map[Name]*Core)

	for _, o := range listing.MinObjects {
		if storageutil.IsUnsupportedPath(o.Name) {
			unsupportedPaths = append(unsupportedPaths, o.Name)
			// Skip unsupported objects in the listing, as the kernel cannot process these file system elements.
			// TODO: Remove this check once we gain confidence that it is not causing any issues.
			if d.isUnsupportedPathSupportEnabled {
				continue
			}
		}

		// Skip empty results or the directory object backing this inode.
		if o.Name == d.Name().GcsObjectName() || o.Name == "" {
			continue
		}

		nameBase := path.Base(o.Name) // ie. "bar" from "foo/bar/" or "foo/bar"

		// Given the alphabetical order of the objects, if a file "foo" and
		// directory "foo/" coexist, the directory would eventually occupy
		// the value of records["foo"].
		if strings.HasSuffix(o.Name, "/") {
			// In a hierarchical bucket, create a folder entry instead of a minObject for each prefix.
			// This is because in a hierarchical bucket, every directory is considered a folder.
			// Adding folder entries while looping to through CollapsedRuns instead of here to avoid duplicate entries.
			if !d.isBucketHierarchical() {
				dirName := NewDirName(d.Name(), nameBase)
				explicitDir := &Core{
					Bucket:    d.Bucket(),
					FullName:  dirName,
					MinObject: o,
				}
				cores[dirName] = explicitDir
			}
		} else {
			fileName := NewFileName(d.Name(), nameBase)
			file := &Core{
				Bucket:    d.Bucket(),
				FullName:  fileName,
				MinObject: o,
			}
			cores[fileName] = file
		}
	}

	// Return an appropriate continuation token, if any.
	newTok = listing.ContinuationToken

	if !d.implicitDirs && !d.isBucketHierarchical() {
		return
	}

	// Add implicit directories into the result.
	for _, p := range listing.CollapsedRuns {
		pathBase := path.Base(p)
		if storageutil.IsUnsupportedPath(p) {
			unsupportedPaths = append(unsupportedPaths, p)
			// Skip unsupported objects in the listing, as the kernel cannot process these file system elements.
			// TODO: Remove this check once we gain confidence that it is not causing any issues.
			if d.isUnsupportedPathSupportEnabled {
				continue
			}
		}
		dirName := NewDirName(d.Name(), pathBase)
		if d.isBucketHierarchical() {
			folder := gcs.Folder{Name: dirName.objectName}

			folderCore := &Core{
				Bucket:   d.Bucket(),
				FullName: dirName,
				Folder:   &folder,
			}
			cores[dirName] = folderCore
		} else {
			if c, ok := cores[dirName]; ok && c.Type() == metadata.ExplicitDirType {
				continue
			}

			implicitDir := &Core{
				Bucket:    d.Bucket(),
				FullName:  dirName,
				MinObject: nil,
			}
			cores[dirName] = implicitDir
		}
	}
	if len(unsupportedPaths) > 0 {
		logger.Warnf("Encountered unsupported prefixes during listing: %v", unsupportedPaths)
	}
	return
}

// LOCKS_REQUIRED(d)
func (d *dirInode) ReadEntries(
	ctx context.Context,
	tok string) (entries []fuseutil.Dirent, unsupportedPaths []string, newTok string, err error) {
	var cores map[Name]*Core
	cores, unsupportedPaths, newTok, err = d.ReadEntryCores(ctx, tok)
	if err != nil {
		return
	}

	for fullName, core := range cores {
		entry := fuseutil.Dirent{
			Name: path.Base(fullName.LocalName()),
			Type: fuseutil.DT_Unknown,
		}
		switch core.Type() {
		case metadata.SymlinkType:
			entry.Type = fuseutil.DT_Link
		case metadata.RegularFileType:
			entry.Type = fuseutil.DT_File
		case metadata.ImplicitDirType, metadata.ExplicitDirType:
			entry.Type = fuseutil.DT_Directory
		}
		entries = append(entries, entry)
	}

	return
}

// LOCKS_REQUIRED(d)
func (d *dirInode) ReadEntryCores(ctx context.Context, tok string) (cores map[Name]*Core, unsupportedPaths []string, newTok string, err error) {
	cores, unsupportedPaths, newTok, err = d.readObjects(ctx, tok)
	if err != nil {
		err = fmt.Errorf("read objects: %w", err)
		return
	}

	d.prevDirListingTimeStamp = d.cacheClock.Now()
	return
}

// LOCKS_REQUIRED(d)
func (d *dirInode) CreateChildFile(ctx context.Context, name string) (*Core, error) {
	childMetadata := map[string]string{
		FileMtimeMetadataKey: d.mtimeClock.Now().UTC().Format(time.RFC3339Nano),
	}
	fullName := NewFileName(d.Name(), name)

	o, err := d.createNewObject(ctx, fullName, childMetadata)
	if err != nil {
		return nil, err
	}
	m := storageutil.ConvertObjToMinObject(o)

	return &Core{
		Bucket:    d.Bucket(),
		FullName:  fullName,
		MinObject: m,
	}, nil
}

func (d *dirInode) CreateLocalChildFileCore(name string) (Core, error) {
	return Core{
		Bucket:    d.Bucket(),
		FullName:  NewFileName(d.Name(), name),
		MinObject: nil,
		Local:     true,
	}, nil
}

// LOCKS_REQUIRED(d)
func (d *dirInode) CloneToChildFile(ctx context.Context, name string, src *gcs.MinObject) (*Core, error) {
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
	m := storageutil.ConvertObjToMinObject(o)

	c := &Core{
		Bucket:    d.Bucket(),
		FullName:  fullName,
		MinObject: m,
	}
	return c, nil
}

// LOCKS_REQUIRED(d)
func (d *dirInode) CreateChildSymlink(ctx context.Context, name string, target string) (*Core, error) {
	fullName := NewFileName(d.Name(), name)
	childMetadata := map[string]string{
		SymlinkMetadataKey: target,
	}

	o, err := d.createNewObject(ctx, fullName, childMetadata)
	if err != nil {
		return nil, err
	}
	m := storageutil.ConvertObjToMinObject(o)

	return &Core{
		Bucket:    d.Bucket(),
		FullName:  fullName,
		MinObject: m,
	}, nil
}

// LOCKS_REQUIRED(d)
func (d *dirInode) CreateChildDir(ctx context.Context, name string) (*Core, error) {
	// Generate the full name for the new directory.
	fullName := NewDirName(d.Name(), name)
	var m *gcs.MinObject
	var f *gcs.Folder
	var err error

	// Check the bucket type.
	if d.isBucketHierarchical() {
		// For hierarchical buckets, create a folder.
		f, err = d.bucket.CreateFolder(ctx, fullName.objectName)
		if err != nil {
			return nil, err
		}
	} else {
		var o *gcs.Object
		// For non-hierarchical buckets, create a new object.
		o, err = d.createNewObject(ctx, fullName, nil)
		if err != nil {
			return nil, err
		}
		// Convert the object to a minimal object.
		m = storageutil.ConvertObjToMinObject(o)
	}

	return &Core{
		Bucket:    d.Bucket(),
		FullName:  fullName,
		MinObject: m,
		Folder:    f,
	}, nil
}

// LOCKS_REQUIRED(d)
func (d *dirInode) DeleteChildFile(
	ctx context.Context,
	name string,
	generation int64,
	metaGeneration *int64) (err error) {
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
	name string,
	isImplicitDir bool,
	dirInode DirInode) error {

	// If the directory is an implicit directory, then no backing object
	// exists in the gcs bucket, so returning from here.
	// Hierarchical buckets don't have implicit dirs so this will be always false in hierarchical bucket case.
	if isImplicitDir {
		return nil
	}

	childName := NewDirName(d.Name(), name)

	// Delete the backing object. Unfortunately we have no way to precondition
	// this on the directory being empty.
	err := d.bucket.DeleteObject(
		ctx,
		&gcs.DeleteObjectRequest{
			Name:       childName.GcsObjectName(),
			Generation: 0, // Delete the latest version of object named after dir.
		})

	if !d.isBucketHierarchical() {
		if err != nil {
			return fmt.Errorf("DeleteObject: %w", err)
		}
		return nil
	}

	// Ignoring delete object error here, as in case of hns there is no way of knowing
	// if underlying placeholder object exists or not in Hierarchical bucket.
	// The DeleteFolder operation handles removing empty folders.
	if err = d.bucket.DeleteFolder(ctx, childName.GcsObjectName()); err != nil {
		return fmt.Errorf("DeleteFolder: %w", err)
	}

	if d.isBucketHierarchical() {
		dirInode.Unlink()
	}

	return nil
}

// LOCKS_REQUIRED(d)
func (d *dirInode) DeleteObjects(ctx context.Context, objectNames []string) error {
	for _, objectName := range objectNames {
		if strings.HasSuffix(objectName, "/") {
			// Initiate deletion for a prefix (directory). This logic handles pagination internally.
			if err := d.deletePrefixRecursively(ctx, objectName); err != nil {
				return fmt.Errorf("recursively deleting prefix %q: %w", objectName, err)
			}
		} else {
			// Handle single file-like object deletion.
			if err := d.deleteObject(ctx, objectName); err != nil {
				return fmt.Errorf("deleting unsupported object %q: %w", objectName, err)
			}
		}
	}
	return nil
}

// Helper to delete a single object, handling 'Not Found' errors gracefully.
// This is important for idempotency. For example, in a recursive delete, we
// list objects and then delete them. If an object is deleted by another process
// between our List and Delete calls, we'd get a 'Not Found' error. By ignoring
// it, we ensure the delete operation succeeds if the object is already gone.
func (d *dirInode) deleteObject(ctx context.Context, objectName string) error {
	// For HNS buckets, the directory entry might be backed by:
	// 1. **Only a folder:** Deleting the 'object' will fail (not found), and the DeleteFolder call is needed.
	// 2. **A 0-byte placeholder object + an empty folder:** We first attempt to delete the 0-byte object. The subsequent DeleteFolder handles the folder removal.

	// 1. Attempt to delete the underlying GCS object (This handles files and 0-byte directory placeholders).
	err := d.bucket.DeleteObject(ctx, &gcs.DeleteObjectRequest{Name: objectName})

	// If it's a non-HNS bucket, return any error other than Not Found.
	if !d.isBucketHierarchical() && err != nil {
		var notFoundErr *gcs.NotFoundError
		if !errors.As(err, &notFoundErr) {
			return err
		}
	}

	// Ignoring delete object error here, as in case of hns there is no way of knowing
	// if underlying placeholder object exists or not in Hierarchical bucket.
	// The DeleteFolder operation handles removing empty folders.
	if d.isBucketHierarchical() && strings.HasSuffix(objectName, "/") {
		if err := d.bucket.DeleteFolder(ctx, objectName); err != nil {
			var notFoundErr *gcs.NotFoundError
			if !errors.As(err, &notFoundErr) {
				return err
			}
		}
		return nil
	}

	return nil
}

// Core recursive function to list, delete, and handle pagination for a prefix.
func (d *dirInode) deletePrefixRecursively(ctx context.Context, prefix string) error {
	var tok string
	for {
		objects, err := d.bucket.ListObjects(ctx, &gcs.ListObjectsRequest{
			Prefix:                   prefix,
			MaxResults:               MaxResultsForListObjectsCall,
			Delimiter:                "/", // Use Delimiter to separate nested folders (CollapsedRuns)
			ContinuationToken:        tok,
			IncludeFoldersAsPrefixes: d.includeFoldersAsPrefixes,
		})
		if err != nil {
			return fmt.Errorf("listing objects under prefix %q: %w", prefix, err)
		}

		// 1. Delete all file-like objects and recurse into subdirectories in parallel.
		g, gCtx := errgroup.WithContext(ctx)
		for _, obj := range objects.MinObjects {
			// obj.Name is guaranteed to start with 'prefix'.
			if !strings.HasSuffix(obj.Name, "/") {
				// It's a file, delete it.
				g.Go(func() error {
					return d.deleteObject(gCtx, obj.Name)
				})
			}
		}

		for _, nestedPrefix := range objects.CollapsedRuns {
			g.Go(func() error {
				return d.deletePrefixRecursively(gCtx, nestedPrefix)
			})
		}

		if err = g.Wait(); err != nil {
			return err // Propagate the first error encountered.
		}

		// If there are no more pages, we are done with this prefix's contents.
		tok = objects.ContinuationToken
		if tok == "" {
			break
		}
	}

	return d.deleteObject(ctx, prefix)
}

// LOCKS_REQUIRED(fs)
func (d *dirInode) LocalFileEntries(localFileInodes map[Name]Inode) (localEntries map[string]fuseutil.Dirent) {
	localEntries = make(map[string]fuseutil.Dirent)

	for localInodeName, in := range localFileInodes {
		// It is possible that the local file inode has been unlinked, but
		// still present in localFileInodes map because of open file handle.
		// So, if the inode has been unlinked, skip the entry.
		file, ok := in.(*FileInode)
		if ok && file.IsUnlinked() {
			continue
		}

		if localInodeName.IsDirectChildOf(d.Name()) {
			entry := fuseutil.Dirent{
				Name: path.Base(localInodeName.LocalName()),
				Type: fuseutil.DT_File,
			}
			localEntries[entry.Name] = entry
		}
	}
	return
}

// ShouldInvalidateKernelListCache doesn't require any lock as d.prevDirListingTimeStamp
// is concurrency safe, and we are okay with the in-consistent value.
func (d *dirInode) ShouldInvalidateKernelListCache(ttl time.Duration) bool {
	// prevDirListingTimeStamp.IsZero() true means listing has not happened yet, and we should
	// invalidate for clean start.
	if d.prevDirListingTimeStamp.IsZero() {
		return true
	}

	cachedDuration := d.cacheClock.Now().Sub(d.prevDirListingTimeStamp)
	return cachedDuration >= ttl
}

// LOCKS_REQUIRED(d)
// LOCKS_REQUIRED(parent of destinationFileName)
func (d *dirInode) RenameFile(ctx context.Context, fileToRename *gcs.MinObject, destinationFileName string) (*gcs.Object, error) {
	req := &gcs.MoveObjectRequest{
		SrcName:                       fileToRename.Name,
		DstName:                       destinationFileName,
		SrcGeneration:                 fileToRename.Generation,
		SrcMetaGenerationPrecondition: &fileToRename.MetaGeneration,
	}

	o, err := d.bucket.MoveObject(ctx, req)

	return o, err
}

func (d *dirInode) RenameFolder(ctx context.Context, folderName string, destinationFolderName string) (*gcs.Folder, error) {
	folder, err := d.bucket.RenameFolder(ctx, folderName, destinationFolderName)
	if err != nil {
		return nil, err
	}

	return folder, nil
}

func (d *dirInode) InvalidateKernelListCache() {
	// Set prevDirListingTimeStamp to Zero time so that cache is invalidated.
	d.prevDirListingTimeStamp = time.Time{}
}

func (d *dirInode) isBucketHierarchical() bool {
	if d.isHNSEnabled && d.bucket.BucketType().Hierarchical {
		return true
	}
	return false
}
