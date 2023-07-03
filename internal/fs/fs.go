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

package fs

import (
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/internal/fs/handle"
	"github.com/googlecloudplatform/gcsfuse/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

type ServerConfig struct {
	// A clock used for cache expiration. It is *not* used for inode times, for
	// which we use the wall clock.
	CacheClock timeutil.Clock

	// The bucket manager is responsible for setting up buckets.
	BucketManager gcsx.BucketManager

	// The name of the specific GCS bucket to be mounted. If it's empty or "_",
	// all accessible GCS buckets are mounted as subdirectories of the FS root.
	BucketName string

	// LocalFileCache
	LocalFileCache bool

	// Enable debug messages
	DebugFS bool

	// The temporary directory to use for local caching, or the empty string to
	// use the system default.
	TempDir string

	// By default, if a bucket contains the object "foo/bar" but no object named
	// "foo/", it's as if the directory doesn't exist. This allows us to have
	// non-flaky name resolution code.
	//
	// Setting this bool to true enables a mode where object listings are
	// consulted to allow for the directory in the situation above to exist. Note
	// that this has drawbacks in the form of name resolution flakiness and
	// surprising behavior.
	//
	// See docs/semantics.md for more info.
	ImplicitDirectories bool

	// By default, if a file/directory does not exist in GCS, this nonexistent state is
	// not cached in type cache. So the inode lookup request will hit GCS every
	// time.
	//
	// Setting this bool to true enables the nonexistent type cache so if the
	// inode state is NonexistentType in type cache, the lookup request will
	// return nil immediately.
	EnableNonexistentTypeCache bool

	// How long to allow the kernel to cache inode attributes.
	//
	// Any given object generation in GCS is immutable, and a new generation
	// results in a new inode number. So every update from a remote system results
	// in a new inode number, and it's therefore safe to allow the kernel to cache
	// inode attributes.
	//
	// The one exception to the above logic is that objects can be _deleted_, in
	// which case stat::st_nlink changes. So choosing this value comes down to
	// whether you care about that field being up to date.
	InodeAttributeCacheTTL time.Duration

	// If non-zero, each directory will maintain a cache from child name to
	// information about whether that name exists as a file and/or directory.
	// This may speed up calls to look up and stat inodes, especially when
	// combined with a stat-caching GCS bucket, but comes at the cost of
	// consistency: if the child is removed and recreated with a different type
	// before the expiration, we may fail to find it.
	DirTypeCacheTTL time.Duration

	// The UID and GID that owns all inodes in the file system.
	Uid uint32
	Gid uint32

	// Permissions bits to use for files and directories. No bits outside of
	// os.ModePerm may be set.
	FilePerms os.FileMode
	DirPerms  os.FileMode

	// Allow renaming a directory containing fewer descendants than this limit.
	RenameDirLimit int64

	// File chunk size to read from GCS in one call. Specified in MB.
	SequentialReadSizeMb int32
}

// Create a fuse file system server according to the supplied configuration.
func NewFileSystem(
	ctx context.Context,
	cfg *ServerConfig) (fuseutil.FileSystem, error) {
	// Check permissions bits.
	if cfg.FilePerms&^os.ModePerm != 0 {
		return nil, fmt.Errorf("Illegal file perms: %v", cfg.FilePerms)
	}

	if cfg.DirPerms&^os.ModePerm != 0 {
		return nil, fmt.Errorf("Illegal dir perms: %v", cfg.FilePerms)
	}

	mtimeClock := timeutil.RealClock()

	contentCache := contentcache.New(cfg.TempDir, mtimeClock)

	if cfg.LocalFileCache {
		err := contentCache.RecoverCache()
		if err != nil {
			fmt.Printf("Encountered error retrieving files from cache directory, disabling local file cache: %v", err)
			cfg.LocalFileCache = false
		}
	}

	// Set up the basic struct.
	fs := &fileSystem{
		mtimeClock:                 mtimeClock,
		cacheClock:                 cfg.CacheClock,
		bucketManager:              cfg.BucketManager,
		localFileCache:             cfg.LocalFileCache,
		contentCache:               contentCache,
		implicitDirs:               cfg.ImplicitDirectories,
		enableNonexistentTypeCache: cfg.EnableNonexistentTypeCache,
		inodeAttributeCacheTTL:     cfg.InodeAttributeCacheTTL,
		dirTypeCacheTTL:            cfg.DirTypeCacheTTL,
		renameDirLimit:             cfg.RenameDirLimit,
		sequentialReadSizeMb:       cfg.SequentialReadSizeMb,
		uid:                        cfg.Uid,
		gid:                        cfg.Gid,
		fileMode:                   cfg.FilePerms,
		dirMode:                    cfg.DirPerms | os.ModeDir,
		inodes:                     make(map[fuseops.InodeID]inode.Inode),
		nextInodeID:                fuseops.RootInodeID + 1,
		generationBackedInodes:     make(map[inode.Name]inode.GenerationBackedInode),
		implicitDirInodes:          make(map[inode.Name]inode.DirInode),
		handles:                    make(map[fuseops.HandleID]interface{}),
	}

	// Set up root bucket
	var root inode.DirInode
	if cfg.BucketName == "" || cfg.BucketName == "_" {
		logger.Info("Set up root directory for all accessible buckets")
		root = makeRootForAllBuckets(fs)
	} else {
		logger.Info("Set up root directory for bucket " + cfg.BucketName)
		syncerBucket, err := fs.bucketManager.SetUpBucket(ctx, cfg.BucketName)
		if err != nil {
			return nil, fmt.Errorf("SetUpBucket: %w", err)
		}
		root = makeRootForBucket(ctx, fs, syncerBucket)
	}
	root.Lock()
	root.IncrementLookupCount()
	fs.inodes[fuseops.RootInodeID] = root
	fs.implicitDirInodes[root.Name()] = root
	root.Unlock()

	// Set up invariant checking.
	fs.mu = locker.New("FS", fs.checkInvariants)
	return fs, nil
}

func makeRootForBucket(
	ctx context.Context,
	fs *fileSystem,
	syncerBucket gcsx.SyncerBucket) inode.DirInode {
	return inode.NewDirInode(
		fuseops.RootInodeID,
		inode.NewRootName(""),
		fuseops.InodeAttributes{
			Uid:  fs.uid,
			Gid:  fs.gid,
			Mode: fs.dirMode,

			// We guarantee only that directory times be "reasonable".
			Atime: fs.mtimeClock.Now(),
			Ctime: fs.mtimeClock.Now(),
			Mtime: fs.mtimeClock.Now(),
		},
		fs.implicitDirs,
		fs.enableNonexistentTypeCache,
		fs.dirTypeCacheTTL,
		&syncerBucket,
		fs.mtimeClock,
		fs.cacheClock,
	)
}

func makeRootForAllBuckets(fs *fileSystem) inode.DirInode {
	return inode.NewBaseDirInode(
		fuseops.RootInodeID,
		inode.NewRootName(""),
		fuseops.InodeAttributes{
			Uid:  fs.uid,
			Gid:  fs.gid,
			Mode: fs.dirMode,

			// We guarantee only that directory times be "reasonable".
			Atime: fs.mtimeClock.Now(),
			Ctime: fs.mtimeClock.Now(),
			Mtime: fs.mtimeClock.Now(),
		},
		fs.bucketManager,
	)
}

////////////////////////////////////////////////////////////////////////
// fileSystem type
////////////////////////////////////////////////////////////////////////

// LOCK ORDERING
//
// Let FS be the file system lock. Define a strict partial order < as follows:
//
//  1. For any inode lock I, I < FS.
//  2. For any handle lock H and inode lock I, H < I.
//
// We follow the rule "acquire A then B only if A < B".
//
// In other words:
//
//  *  Don't hold multiple handle locks at the same time.
//  *  Don't hold multiple inode locks at the same time.
//  *  Don't acquire inode locks before handle locks.
//  *  Don't acquire file system locks before either.
//
// The intuition is that we hold inode and handle locks for long-running
// operations, and we don't want to block the entire file system on those.
//
// See http://goo.gl/rDxxlG for more discussion, including an informal proof
// that a strict partial order is sufficient.

type fileSystem struct {
	fuseutil.NotImplementedFileSystem

	/////////////////////////
	// Dependencies
	/////////////////////////

	mtimeClock    timeutil.Clock
	cacheClock    timeutil.Clock
	bucketManager gcsx.BucketManager

	/////////////////////////
	// Constant data
	/////////////////////////

	localFileCache             bool
	contentCache               *contentcache.ContentCache
	implicitDirs               bool
	enableNonexistentTypeCache bool
	inodeAttributeCacheTTL     time.Duration
	dirTypeCacheTTL            time.Duration
	renameDirLimit             int64
	sequentialReadSizeMb       int32

	// The user and group owning everything in the file system.
	uid uint32
	gid uint32

	// Mode bits for all inodes.
	fileMode os.FileMode
	dirMode  os.FileMode

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A lock protecting the state of the file system struct itself (distinct
	// from per-inode locks). Make sure to see the notes on lock ordering above.
	mu locker.Locker

	// The next inode ID to hand out. We assume that this will never overflow,
	// since even if we were handing out inode IDs at 4 GHz, it would still take
	// over a century to do so.
	//
	// GUARDED_BY(mu)
	nextInodeID fuseops.InodeID

	// The collection of live inodes, keyed by inode ID. No ID less than
	// fuseops.RootInodeID is ever used.
	//
	// INVARIANT: For all keys k, fuseops.RootInodeID <= k < nextInodeID
	// INVARIANT: For all keys k, inodes[k].ID() == k
	// INVARIANT: inodes[fuseops.RootInodeID] is missing or of type inode.DirInode
	// INVARIANT: For all v, if v.Name().IsDir() then v is inode.DirInode
	//
	// GUARDED_BY(mu)
	inodes map[fuseops.InodeID]inode.Inode

	// A map from object name to an inode for that name backed by a GCS object.
	// Populated during the name -> inode lookup process, cleared during the
	// forget inode process.
	//
	// Entries may be stale for two reasons:
	//
	//  1. There is a newer generation in GCS, not caused by the inode. The next
	//     name lookup will detect this by statting the object, acquiring the
	//     inode's lock (to get an up to date look at what the latest generation
	//     the inode caused was), and replacing the entry if the inode's
	//     generation is less than the stat generation.
	//
	//  2. The object no longer exists. This is harmless; the name lookup process
	//     will return ENOENT before it ever consults this map. Eventually the
	//     kernel will send ForgetInodeOp and we will clear the entry.
	//
	// Crucially, we never replace an up to date entry with a stale one. If the
	// name lookup process sees that the stat result is older than the inode, it
	// starts over, statting again.
	//
	// Note that there is no invariant that says *all* of the object-backed
	// inodes are represented here because we may have multiple distinct inodes
	// for a given name existing concurrently if we observe an object generation
	// that was not caused by our existing inode (e.g. if the file is clobbered
	// remotely). We must retain the old inode until the kernel tells us to
	// forget it.
	//
	// INVARIANT: For each k/v, v.Name() == k
	// INVARIANT: For each value v, inodes[v.ID()] == v
	//
	// GUARDED_BY(mu)
	generationBackedInodes map[inode.Name]inode.GenerationBackedInode

	// A map from object name to the implicit directory inode that represents
	// that name, if any. There can be at most one implicit directory inode for a
	// given name accessible to us at any given time.
	//
	// INVARIANT: For each k/v, v.Name() == k
	// INVARIANT: For each value v, inodes[v.ID()] == v
	// INVARIANT: For each value v, v is not ExplicitDirInode
	// INVARIANT: For each in in inodes such that in is DirInode but not
	//            ExplicitDirInode, implicitDirInodes[d.Name()] == d
	//
	// GUARDED_BY(mu)
	implicitDirInodes map[inode.Name]inode.DirInode

	// The collection of live handles, keyed by handle ID.
	//
	// INVARIANT: All values are of type *dirHandle or *handle.FileHandle
	//
	// GUARDED_BY(mu)
	handles map[fuseops.HandleID]interface{}

	// The next handle ID to hand out. We assume that this will never overflow.
	//
	// INVARIANT: For all keys k in handles, k < nextHandleID
	//
	// GUARDED_BY(mu)
	nextHandleID fuseops.HandleID
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (fs *fileSystem) checkInvariants() {
	//////////////////////////////////
	// inodes
	//////////////////////////////////

	// INVARIANT: For all keys k, fuseops.RootInodeID <= k < nextInodeID
	for id, _ := range fs.inodes {
		if id < fuseops.RootInodeID || id >= fs.nextInodeID {
			panic(fmt.Sprintf("Illegal inode ID: %v", id))
		}
	}

	// INVARIANT: For all keys k, inodes[k].ID() == k
	for id, in := range fs.inodes {
		if in.ID() != id {
			panic(fmt.Sprintf("ID mismatch: %v vs. %v", in.ID(), id))
		}
	}

	// INVARIANT: inodes[fuseops.RootInodeID] is missing or of type inode.DirInode
	//
	// The missing case is when we've received a forget request for the root
	// inode, while unmounting.
	switch in := fs.inodes[fuseops.RootInodeID].(type) {
	case nil:
	case inode.DirInode:
	default:
		panic(fmt.Sprintf("Unexpected type for root: %v", reflect.TypeOf(in)))
	}

	// INVARIANT: For all v, if v.Name().IsDir() then v is inode.DirInode
	for _, in := range fs.inodes {
		if in.Name().IsDir() {
			_, ok := in.(inode.DirInode)
			if !ok {
				panic(fmt.Sprintf(
					"Unexpected inode type for name \"%s\": %v",
					in.Name(),
					reflect.TypeOf(in)))
			}
		}
	}

	//////////////////////////////////
	// generationBackedInodes
	//////////////////////////////////

	// INVARIANT: For each k/v, v.Name() == k
	for k, v := range fs.generationBackedInodes {
		if !(v.Name() == k) {
			panic(fmt.Sprintf(
				"Unexpected name: \"%s\" vs. \"%s\"",
				v.Name(),
				k))
		}
	}

	// INVARIANT: For each value v, inodes[v.ID()] == v
	for _, v := range fs.generationBackedInodes {
		if fs.inodes[v.ID()] != v {
			panic(fmt.Sprintf(
				"Mismatch for ID %v: %v %v",
				v.ID(),
				fs.inodes[v.ID()],
				v))
		}
	}

	//////////////////////////////////
	// implicitDirInodes
	//////////////////////////////////

	// INVARIANT: For each k/v, v.Name() == k
	for k, v := range fs.implicitDirInodes {
		if !(v.Name() == k) {
			panic(fmt.Sprintf(
				"Unexpected name: \"%s\" vs. \"%s\"",
				v.Name(),
				k))
		}
	}

	// INVARIANT: For each value v, inodes[v.ID()] == v
	for _, v := range fs.implicitDirInodes {
		if fs.inodes[v.ID()] != v {
			panic(fmt.Sprintf(
				"Mismatch for ID %v: %v %v",
				v.ID(),
				fs.inodes[v.ID()],
				v))
		}
	}

	// INVARIANT: For each value v, v is not ExplicitDirInode
	for _, v := range fs.implicitDirInodes {
		if _, ok := v.(inode.ExplicitDirInode); ok {
			panic(fmt.Sprintf(
				"Unexpected implicit dir inode %d, type %T",
				v.ID(),
				v))
		}
	}

	// INVARIANT: For each in in inodes such that in is DirInode but not
	//            ExplicitDirInode, implicitDirInodes[d.Name()] == d
	for _, in := range fs.inodes {
		_, dir := in.(inode.DirInode)
		_, edir := in.(inode.ExplicitDirInode)

		if dir && !edir {
			if !(fs.implicitDirInodes[in.Name()] == in) {
				panic(fmt.Sprintf(
					"implicitDirInodes mismatch: %q %v %v",
					in.Name(),
					fs.implicitDirInodes[in.Name()],
					in))
			}
		}
	}

	//////////////////////////////////
	// handles
	//////////////////////////////////

	// INVARIANT: All values are of type *dirHandle or *handle.FileHandle
	for _, h := range fs.handles {
		switch h.(type) {
		case *dirHandle:
		case *handle.FileHandle:
		default:
			panic(fmt.Sprintf("Unexpected handle type: %T", h))
		}
	}

	//////////////////////////////////
	// nextHandleID
	//////////////////////////////////

	// INVARIANT: For all keys k in handles, k < nextHandleID
	for k, _ := range fs.handles {
		if k >= fs.nextHandleID {
			panic(fmt.Sprintf("Illegal handle ID: %v", k))
		}
	}
}

// Implementation detail of lookUpOrCreateInodeIfNotStale; do not use outside
// of that function.
//
// LOCKS_REQUIRED(fs.mu)
func (fs *fileSystem) mintInode(ic inode.Core) (in inode.Inode) {
	// Choose an ID.
	id := fs.nextInodeID
	fs.nextInodeID++

	// Create the inode.
	switch {
	// Explicit directories
	case ic.Object != nil && ic.FullName.IsDir():
		in = inode.NewExplicitDirInode(
			id,
			ic.FullName,
			ic.Object,
			fuseops.InodeAttributes{
				Uid:  fs.uid,
				Gid:  fs.gid,
				Mode: fs.dirMode,

				// We guarantee only that directory times be "reasonable".
				Atime: fs.mtimeClock.Now(),
				Ctime: fs.mtimeClock.Now(),
				Mtime: fs.mtimeClock.Now(),
			},
			fs.implicitDirs,
			fs.enableNonexistentTypeCache,
			fs.dirTypeCacheTTL,
			ic.Bucket,
			fs.mtimeClock,
			fs.cacheClock)

		// Implicit directories
	case ic.FullName.IsDir():
		in = inode.NewDirInode(
			id,
			ic.FullName,
			fuseops.InodeAttributes{
				Uid:  fs.uid,
				Gid:  fs.gid,
				Mode: fs.dirMode,

				// We guarantee only that directory times be "reasonable".
				Atime: fs.mtimeClock.Now(),
				Ctime: fs.mtimeClock.Now(),
				Mtime: fs.mtimeClock.Now(),
			},
			fs.implicitDirs,
			fs.enableNonexistentTypeCache,
			fs.dirTypeCacheTTL,
			ic.Bucket,
			fs.mtimeClock,
			fs.cacheClock)

	case inode.IsSymlink(ic.Object):
		in = inode.NewSymlinkInode(
			id,
			ic.FullName,
			ic.Object,
			fuseops.InodeAttributes{
				Uid:  fs.uid,
				Gid:  fs.gid,
				Mode: fs.fileMode | os.ModeSymlink,
			})

	default:
		in = inode.NewFileInode(
			id,
			ic.FullName,
			ic.Object,
			fuseops.InodeAttributes{
				Uid:  fs.uid,
				Gid:  fs.gid,
				Mode: fs.fileMode,
			},
			ic.Bucket,
			fs.localFileCache,
			fs.contentCache,
			fs.mtimeClock)
	}

	// Place it in our map of IDs to inodes.
	fs.inodes[in.ID()] = in

	return
}

// Attempt to find an inode for a backing object or an implicit directory.
// Create an inode if (1) it has never yet existed, or (2) the object is newer
// than the existing one.
//
// If the backing object is older than the existing inode, return nil. In this
// case, the caller may obtain a fresh record and try again. Otherwise,
// increment the inode's lookup count and return it locked.
//
// LOCKS_EXCLUDED(fs.mu)
// UNLOCK_FUNCTION(fs.mu)
// LOCK_FUNCTION(in)
func (fs *fileSystem) lookUpOrCreateInodeIfNotStale(ic inode.Core) (in inode.Inode) {

	// Sanity check.
	if err := ic.SanityCheck(); err != nil {
		panic(err.Error())
	}

	// Ensure that no matter which inode we return, we increase its lookup count
	// on the way out and then release the file system lock.
	defer func() {
		if in != nil {
			in.IncrementLookupCount()
		}

		fs.mu.Unlock()
	}()

	fs.mu.Lock()

	// Handle implicit directories.
	if ic.Object == nil {
		if !ic.FullName.IsDir() {
			panic(fmt.Sprintf("Unexpected name for an implicit directory: %q", ic.FullName))
		}

		var ok bool
		var maxTriesToCreateInode = 3
		for n := 0; n < maxTriesToCreateInode; n++ {
			in, ok = fs.implicitDirInodes[ic.FullName]
			// If we don't have an entry, create one.
			if !ok {
				in = fs.mintInode(ic)
				fs.implicitDirInodes[in.Name()] = in.(inode.DirInode)
				// Since we are creating inode here, there is no chance that something else
				// is holding the lock for inode. Hence its safe to take lock on inode
				// without releasing fs.mu.lock.
				in.Lock()
				return
			}

			// If the inode already exists, we need to follow the lock ordering rules
			// to get the lock. First get inode lock and then fs lock.
			fs.mu.Unlock()
			in.Lock()
			fs.mu.Lock()

			// Check if inode is still valid by the time we got the lock. If not,
			// its means inode is in the process of getting destroyed. Try creating it
			// again.
			if fs.implicitDirInodes[ic.FullName] != in {
				in.Unlock()
				continue
			}

			return
		}

		// Incase we exhausted the number of tries to createInode, we will return
		// nil object. Returning nil is handled by callers to throw appropriate
		// errors back to kernel.
		in = nil
		return
	}

	oGen := inode.Generation{
		Object:   ic.Object.Generation,
		Metadata: ic.Object.MetaGeneration,
	}

	// Retry loop for the stale index entry case below. On entry, we hold fs.mu
	// but no inode lock.
	for {
		// Look at the current index entry.
		existingInode, ok := fs.generationBackedInodes[ic.FullName]

		// If we have no existing record, mint an inode and return it.
		if !ok {
			in = fs.mintInode(ic)
			fs.generationBackedInodes[in.Name()] = in.(inode.GenerationBackedInode)

			in.Lock()
			return
		}

		// Otherwise we need to read the inode's source generation below, which
		// requires the inode's lock. We must not hold the inode lock while
		// acquiring the file system lock, so drop it while acquiring the inode's
		// lock, then reacquire.
		fs.mu.Unlock()
		existingInode.Lock()
		fs.mu.Lock()

		// Check that the index still points at this inode. If not, it's possible
		// that the inode is in the process of being destroyed and is unsafe to
		// use. Go around and try again.
		if fs.generationBackedInodes[ic.FullName] != existingInode {
			existingInode.Unlock()
			continue
		}

		// Have we found the correct inode?
		cmp := oGen.Compare(existingInode.SourceGeneration())
		if cmp == 0 {
			in = existingInode
			return
		}

		// The existing inode is newer than the backing object. The caller
		// should call again with a newer backing object.
		if cmp == -1 {
			existingInode.Unlock()
			return
		}

		// The backing object is newer than the existing inode, while
		// holding the inode lock, excluding concurrent actions by the inode (in
		// particular concurrent calls to Sync, which changes generation numbers).
		// This means we've proven that the record cannot have been caused by the
		// inode's actions, and therefore this is not the inode we want.
		//
		// Replace it with a newly-mintend inode and then go around, acquiring its
		// lock in accordance with our lock ordering rules.
		existingInode.Unlock()

		in = fs.mintInode(ic)
		fs.generationBackedInodes[in.Name()] = in.(inode.GenerationBackedInode)

		continue
	}
}

// Look up the child with the given name within the parent, then return an
// existing inode for that child or create a new one if necessary. Return
// ENOENT if the child doesn't exist.
//
// Return the child locked, incrementing its lookup count.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCKS_EXCLUDED(parent)
// LOCK_FUNCTION(child)
func (fs *fileSystem) lookUpOrCreateChildInode(
	ctx context.Context,
	parent inode.DirInode,
	childName string) (child inode.Inode, err error) {
	// Set up a function that will find a lookup result for the child with the
	// given name. Expects no locks to be held.
	getLookupResult := func() (*inode.Core, error) {
		parent.Lock()
		defer parent.Unlock()
		return parent.LookUpChild(ctx, childName)
	}

	// Run a retry loop around lookUpOrCreateInodeIfNotStale.
	const maxTries = 3
	for n := 0; n < maxTries; n++ {
		// Create a record.
		var core *inode.Core
		core, err = getLookupResult()

		if err != nil {
			return
		}

		if core == nil {
			err = fuse.ENOENT
			return
		}

		// Attempt to create the inode. Return if successful.
		child = fs.lookUpOrCreateInodeIfNotStale(*core)
		if child != nil {
			return
		}
	}

	err = fmt.Errorf("cannot find %q in %q with %v tries", childName, parent.Name(), maxTries)
	return
}

// Look up the child directory with the given name within the parent, then
// return an existing dir inode for that child or create a new one if necessary.
// Return ENOENT if the child doesn't exist.
//
// Return the child locked, incrementing its lookup count.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCKS_EXCLUDED(parent)
// LOCK_FUNCTION(child)
func (fs *fileSystem) lookUpOrCreateChildDirInode(
	ctx context.Context,
	parent inode.DirInode,
	childName string) (child inode.BucketOwnedDirInode, err error) {
	in, err := fs.lookUpOrCreateChildInode(ctx, parent, childName)
	if err != nil {
		return nil, fmt.Errorf("lookup or create %q: %w", childName, err)
	}
	var ok bool
	if child, ok = in.(inode.BucketOwnedDirInode); !ok {
		fs.unlockAndDecrementLookupCount(in, 1)
		return nil, fmt.Errorf("not a bucket owned directory: %q", childName)
	}
	return child, nil
}

// Synchronize the supplied file inode to GCS, updating the index as
// appropriate.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCKS_REQUIRED(f)
func (fs *fileSystem) syncFile(
	ctx context.Context,
	f *inode.FileInode) (err error) {
	// Sync the inode.
	err = f.Sync(ctx)
	if err != nil {
		err = fmt.Errorf("FileInode.Sync: %w", err)
		return
	}

	// We need not update fileIndex:
	//
	// We've held the inode lock the whole time, so there's no way that this
	// inode could have been booted from the index. Therefore if it's not in the
	// index at the moment, it must not have been in there when we started. That
	// is, it must have been clobbered remotely, which we treat as unlinking.
	//
	// In other words, either this inode is still in the index or it has been
	// unlinked and *should* be anonymous.

	return
}

// Decrement the supplied inode's lookup count, destroying it if the inode says
// that it has hit zero.
//
// We require the file system lock to exclude concurrent lookups, which might
// otherwise find an inode whose lookup count has gone to zero.
//
// LOCKS_REQUIRED(in)
// LOCKS_EXCLUDED(fs.mu)
// UNLOCK_FUNCTION(fs.mu)
// UNLOCK_FUNCTION(in)
func (fs *fileSystem) unlockAndDecrementLookupCount(in inode.Inode, N uint64) {
	name := in.Name()

	// Decrement the lookup count.
	shouldDestroy := in.DecrementLookupCount(N)

	// Update file system state, orphaning the inode if we're going to destroy it
	// below.
	if shouldDestroy {
		fs.mu.Lock()
		delete(fs.inodes, in.ID())

		// Update indexes if necessary.
		if fs.generationBackedInodes[name] == in {
			delete(fs.generationBackedInodes, name)
		}
		if fs.implicitDirInodes[name] == in {
			delete(fs.implicitDirInodes, name)
		}
		fs.mu.Unlock()
	}

	// Now we can destroy the inode if necessary.
	if shouldDestroy {
		destroyErr := in.Destroy()
		if destroyErr != nil {
			logger.Infof("Error destroying inode %q: %v", name, destroyErr)
		}
	}

	in.Unlock()
}

// A helper function for use after incrementing an inode's lookup count.
// Ensures that the lookup count is decremented again if the caller is going to
// return in error (in which case the kernel and gcsfuse would otherwise
// disagree about the lookup count for the inode's ID), so that the inode isn't
// leaked.
//
// Typical usage:
//
//	func (fs *fileSystem) doFoo() (err error) {
//	  in, err := fs.lookUpOrCreateInodeIfNotStale(...)
//	  if err != nil {
//	    return
//	  }
//
//	  defer fs.unlockAndMaybeDisposeOfInode(in, &err)
//
//	  ...
//	}
//
// LOCKS_REQUIRED(in)
// LOCKS_EXCLUDED(fs.mu)
// UNLOCK_FUNCTION(in)
func (fs *fileSystem) unlockAndMaybeDisposeOfInode(
	in inode.Inode,
	err *error) {
	// If there is no error, just unlock.
	if *err == nil {
		in.Unlock()
		return
	}

	// Otherwise, go through the decrement helper
	fs.unlockAndDecrementLookupCount(in, 1)
}

// Fetch attributes for the supplied inode and fill in an appropriate
// expiration time for them.
//
// LOCKS_REQUIRED(in)
func (fs *fileSystem) getAttributes(
	ctx context.Context,
	in inode.Inode) (
	attr fuseops.InodeAttributes,
	expiration time.Time,
	err error) {
	// Call through.
	attr, err = in.Attributes(ctx)
	if err != nil {
		return
	}

	// Set up the expiration time.
	if fs.inodeAttributeCacheTTL > 0 {
		expiration = time.Now().Add(fs.inodeAttributeCacheTTL)
	}

	return
}

// inodeOrDie returns the inode with the given ID, panicking with a helpful
// error message if it doesn't exist.
//
// LOCKS_REQUIRED(fs.mu)
func (fs *fileSystem) inodeOrDie(id fuseops.InodeID) (in inode.Inode) {
	in = fs.inodes[id]
	if in == nil {
		panic(fmt.Sprintf("inode %d doesn't exist", id))
	}

	return
}

// dirInodeOrDie returns the directory inode with the given ID, panicking with
// a helpful error message if it doesn't exist or is the wrong type.
//
// LOCKS_REQUIRED(fs.mu)
func (fs *fileSystem) dirInodeOrDie(id fuseops.InodeID) (in inode.DirInode) {
	tmp := fs.inodes[id]
	in, ok := tmp.(inode.DirInode)
	if !ok {
		panic(fmt.Sprintf("inode %d is %T, wanted inode.DirInode", id, tmp))
	}

	return
}

// fileInodeOrDie returns the file inode with the given ID, panicking with a
// helpful error message if it doesn't exist or is the wrong type.
//
// LOCKS_REQUIRED(fs.mu)
func (fs *fileSystem) fileInodeOrDie(id fuseops.InodeID) (in *inode.FileInode) {
	tmp := fs.inodes[id]
	in, ok := tmp.(*inode.FileInode)
	if !ok {
		panic(fmt.Sprintf("inode %d is %T, wanted *inode.FileInode", id, tmp))
	}

	return
}

// symlinkInodeOrDie returns the symlink inode with the given ID, panicking
// with a helpful error message if it doesn't exist or is the wrong type.
//
// LOCKS_REQUIRED(fs.mu)
func (fs *fileSystem) symlinkInodeOrDie(
	id fuseops.InodeID) (in *inode.SymlinkInode) {
	tmp := fs.inodes[id]
	in, ok := tmp.(*inode.SymlinkInode)
	if !ok {
		panic(fmt.Sprintf("inode %d is %T, wanted *inode.SymlinkInode", id, tmp))
	}

	return
}

////////////////////////////////////////////////////////////////////////
// fuse.FileSystem methods
////////////////////////////////////////////////////////////////////////

func (fs *fileSystem) Destroy() {
	fs.bucketManager.ShutDown()
}

func (fs *fileSystem) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) (err error) {
	// Simulate a large amount of free space so that the Finder doesn't refuse to
	// copy in files. (See issue #125.) Use 2^17 as the block size because that
	// is the largest that OS X will pass on.
	op.BlockSize = 1 << 17
	op.Blocks = 1 << 33
	op.BlocksFree = op.Blocks
	op.BlocksAvailable = op.Blocks

	// Similarly with inodes.
	op.Inodes = 1 << 50
	op.InodesFree = op.Inodes

	// Prefer large transfers. This is the largest value that OS X will
	// faithfully pass on, according to fuseops/ops.go.
	op.IoSize = 1 << 20

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) LookUpInode(
	ctx context.Context,
	op *fuseops.LookUpInodeOp) (err error) {
	// Find the parent directory in question.
	fs.mu.Lock()
	parent := fs.dirInodeOrDie(op.Parent)
	fs.mu.Unlock()

	// Find or create the child inode.
	child, err := fs.lookUpOrCreateChildInode(ctx, parent, op.Name)
	if err != nil {
		return err
	}

	defer fs.unlockAndMaybeDisposeOfInode(child, &err)

	// Fill out the response.
	e := &op.Entry
	e.Child = child.ID()
	e.Attributes, e.AttributesExpiration, err = fs.getAttributes(ctx, child)

	if err != nil {
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.inodeOrDie(op.Inode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Grab its attributes.
	op.Attributes, op.AttributesExpiration, err = fs.getAttributes(ctx, in)
	if err != nil {
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) SetInodeAttributes(
	ctx context.Context,
	op *fuseops.SetInodeAttributesOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.inodeOrDie(op.Inode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()
	file, isFile := in.(*inode.FileInode)

	// Set file mtimes.
	if isFile && op.Mtime != nil {
		err = file.SetMtime(ctx, *op.Mtime)
		if err != nil {
			err = fmt.Errorf("SetMtime: %w", err)
			return err
		}
	}

	// Truncate files.
	if isFile && op.Size != nil {
		err = file.Truncate(ctx, int64(*op.Size))
		if err != nil {
			err = fmt.Errorf("Truncate: %w", err)
			return err
		}
	}

	// We silently ignore updates to mode and atime.

	// Fill in the response.
	op.Attributes, op.AttributesExpiration, err = fs.getAttributes(ctx, in)
	if err != nil {
		err = fmt.Errorf("getAttributes: %w", err)
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.inodeOrDie(op.Inode)
	fs.mu.Unlock()

	// Decrement and unlock.
	in.Lock()
	fs.unlockAndDecrementLookupCount(in, op.N)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) MkDir(
	ctx context.Context,
	op *fuseops.MkDirOp) (err error) {
	// Find the parent.
	fs.mu.Lock()
	parent := fs.dirInodeOrDie(op.Parent)
	fs.mu.Unlock()

	// Create an empty backing object for the child, failing if it already
	// exists.
	parent.Lock()
	result, err := parent.CreateChildDir(ctx, op.Name)
	parent.Unlock()

	// Special case: *gcs.PreconditionError means the name already exists.
	var preconditionErr *gcs.PreconditionError
	if errors.As(err, &preconditionErr) {
		err = fuse.EEXIST
		return
	}

	// Propagate other errors.
	if err != nil {
		err = fmt.Errorf("CreateChildDir: %w", err)
		return err
	}

	// Attempt to create a child inode using the object we created. If we fail to
	// do so, it means someone beat us to the punch with a newer generation
	// (unlikely, so we're probably okay with failing here).
	child := fs.lookUpOrCreateInodeIfNotStale(*result)
	if child == nil {
		err = fmt.Errorf("Newly-created record is already stale")
		return err
	}

	defer fs.unlockAndMaybeDisposeOfInode(child, &err)

	// Fill out the response.
	e := &op.Entry
	e.Child = child.ID()
	e.Attributes, e.AttributesExpiration, err = fs.getAttributes(ctx, child)

	if err != nil {
		err = fmt.Errorf("getAttributes: %w", err)
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) MkNode(
	ctx context.Context,
	op *fuseops.MkNodeOp) (err error) {
	if (op.Mode & (iofs.ModeNamedPipe | iofs.ModeSocket)) != 0 {
		return syscall.ENOTSUP
	}

	// Create the child.
	child, err := fs.createFile(ctx, op.Parent, op.Name, op.Mode)
	if err != nil {
		return err
	}

	defer fs.unlockAndMaybeDisposeOfInode(child, &err)

	// Fill out the response.
	e := &op.Entry
	e.Child = child.ID()
	e.Attributes, e.AttributesExpiration, err = fs.getAttributes(ctx, child)

	if err != nil {
		err = fmt.Errorf("getAttributes: %w", err)
		return err
	}

	return
}

// Create a child of the parent with the given ID, returning the child locked
// and with its lookup count incremented.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCK_FUNCTION(child)
func (fs *fileSystem) createFile(
	ctx context.Context,
	parentID fuseops.InodeID,
	name string,
	mode os.FileMode) (child inode.Inode, err error) {
	// Find the parent.
	fs.mu.Lock()
	parent := fs.dirInodeOrDie(parentID)
	fs.mu.Unlock()

	// Create an empty backing object for the child, failing if it already
	// exists.
	parent.Lock()
	result, err := parent.CreateChildFile(ctx, name)
	parent.Unlock()

	// Special case: *gcs.PreconditionError means the name already exists.
	var preconditionErr *gcs.PreconditionError
	if errors.As(err, &preconditionErr) {
		err = fuse.EEXIST
		return
	}

	// Propagate other errors.
	if err != nil {
		err = fmt.Errorf("CreateChildFile: %w", err)
		return
	}

	// Attempt to create a child inode using the object we created. If we fail to
	// do so, it means someone beat us to the punch with a newer generation
	// (unlikely, so we're probably okay with failing here).
	child = fs.lookUpOrCreateInodeIfNotStale(*result)
	if child == nil {
		err = fmt.Errorf("Newly-created record is already stale")
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) (err error) {
	// Create the child.
	child, err := fs.createFile(ctx, op.Parent, op.Name, op.Mode)
	if err != nil {
		return err
	}

	defer fs.unlockAndMaybeDisposeOfInode(child, &err)

	// Allocate a handle.
	fs.mu.Lock()

	handleID := fs.nextHandleID
	fs.nextHandleID++

	fs.handles[handleID] = handle.NewFileHandle(child.(*inode.FileInode))
	op.Handle = handleID

	fs.mu.Unlock()

	// Fill out the response.
	e := &op.Entry
	e.Child = child.ID()
	e.Attributes, e.AttributesExpiration, err = fs.getAttributes(ctx, child)

	if err != nil {
		err = fmt.Errorf("getAttributes: %w", err)
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) CreateSymlink(
	ctx context.Context,
	op *fuseops.CreateSymlinkOp) (err error) {
	// Find the parent.
	fs.mu.Lock()
	parent := fs.dirInodeOrDie(op.Parent)
	fs.mu.Unlock()

	// Create the object in GCS, failing if it already exists.
	parent.Lock()
	result, err := parent.CreateChildSymlink(ctx, op.Name, op.Target)
	parent.Unlock()

	// Special case: *gcs.PreconditionError means the name already exists.
	var preconditionErr *gcs.PreconditionError
	if errors.As(err, &preconditionErr) {
		err = fuse.EEXIST
		return
	}

	// Propagate other errors.
	if err != nil {
		err = fmt.Errorf("CreateChildSymlink: %w", err)
		return err
	}

	// Attempt to create a child inode using the object we created. If we fail to
	// do so, it means someone beat us to the punch with a newer generation
	// (unlikely, so we're probably okay with failing here).
	child := fs.lookUpOrCreateInodeIfNotStale(*result)
	if child == nil {
		err = fmt.Errorf("Newly-created record is already stale")
		return err
	}

	defer fs.unlockAndMaybeDisposeOfInode(child, &err)

	// Fill out the response.
	e := &op.Entry
	e.Child = child.ID()
	e.Attributes, e.AttributesExpiration, err = fs.getAttributes(ctx, child)

	if err != nil {
		err = fmt.Errorf("getAttributes: %w", err)
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) RmDir(
	ctx context.Context,
	op *fuseops.RmDirOp) (err error) {
	// Find the parent.
	fs.mu.Lock()
	parent := fs.dirInodeOrDie(op.Parent)
	fs.mu.Unlock()

	// Find or create the child inode, locked.
	child, err := fs.lookUpOrCreateChildInode(ctx, parent, op.Name)
	if err != nil {
		return
	}

	// Set up a function that throws away the lookup count increment that we
	// implicitly did above (since we're not handing the child back to the
	// kernel) and unlocks the child, but only once. Ensure it is called at least
	// once in case we exit early.
	childCleanedUp := false
	cleanUpAndUnlockChild := func() {
		if !childCleanedUp {
			childCleanedUp = true
			fs.unlockAndDecrementLookupCount(child, 1)
		}
	}

	defer cleanUpAndUnlockChild()

	// Is the child a directory?
	childDir, ok := child.(inode.DirInode)
	if !ok {
		err = fuse.ENOTDIR
		return
	}

	// Ensure that the child directory is empty.
	//
	// Yes, this is not atomic with the delete below. See here for discussion:
	//
	//     https://github.com/GoogleCloudPlatform/gcsfuse/issues/9
	//
	//
	var tok string
	for {
		var entries []fuseutil.Dirent
		entries, tok, err = childDir.ReadEntries(ctx, tok)
		if err != nil {
			err = fmt.Errorf("ReadEntries: %w", err)
			return err
		}

		// Are there any entries?
		if len(entries) != 0 {
			err = fuse.ENOTEMPTY
			return
		}

		// Are we done listing?
		if tok == "" {
			break
		}
	}

	// We are done with the child.
	cleanUpAndUnlockChild()

	// Delete the backing object.
	parent.Lock()
	err = parent.DeleteChildDir(ctx, op.Name)
	parent.Unlock()

	if err != nil {
		err = fmt.Errorf("DeleteChildDir: %w", err)
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) Rename(
	ctx context.Context,
	op *fuseops.RenameOp) (err error) {
	// Find the old and new parents.
	fs.mu.Lock()
	oldParent := fs.dirInodeOrDie(op.OldParent)
	newParent := fs.dirInodeOrDie(op.NewParent)
	fs.mu.Unlock()

	if oldInode, ok := oldParent.(inode.BucketOwnedInode); !ok {
		// The old parent is not owned by any bucket, which means it's the base
		// directory that holds all the buckets' root directories. So, this op
		// is to rename a bucket, which is not supported.
		return fmt.Errorf("rename a bucket: %w", syscall.ENOTSUP)
	} else {
		// The target path must exist in the same bucket.
		oldBucket := oldInode.Bucket().Name()
		if newInode, ok := newParent.(inode.BucketOwnedInode); !ok || oldBucket != newInode.Bucket().Name() {
			return fmt.Errorf("move out of bucket %q: %w", oldBucket, syscall.ENOTSUP)
		}
	}

	// Find the object in the old location.
	oldParent.Lock()
	child, err := oldParent.LookUpChild(ctx, op.OldName)
	oldParent.Unlock()

	if err != nil {
		err = fmt.Errorf("LookUpChild: %w", err)
		return err
	}

	if child == nil {
		err = fuse.ENOENT
		return err
	}

	if child.FullName.IsDir() {
		return fs.renameDir(ctx, oldParent, op.OldName, newParent, op.NewName)
	}
	return fs.renameFile(ctx, oldParent, op.OldName, child.Object, newParent, op.NewName)
}

// LOCKS_EXCLUDED(fs.mu)
// LOCKS_EXCLUDED(oldParent)
// LOCKS_EXCLUDED(newParent)
func (fs *fileSystem) renameFile(
	ctx context.Context,
	oldParent inode.DirInode,
	oldName string,
	oldObject *gcs.Object,
	newParent inode.DirInode,
	newFileName string) error {
	// Clone into the new location.
	newParent.Lock()
	_, err := newParent.CloneToChildFile(ctx, newFileName, oldObject)
	newParent.Unlock()

	if err != nil {
		err = fmt.Errorf("CloneToChildFile: %w", err)
		return err
	}

	// Delete behind. Make sure to delete exactly the generation we cloned, in
	// case the referent of the name has changed in the meantime.
	oldParent.Lock()
	err = oldParent.DeleteChildFile(
		ctx,
		oldName,
		oldObject.Generation,
		&oldObject.MetaGeneration)
	oldParent.Unlock()

	if err != nil {
		err = fmt.Errorf("DeleteChildFile: %w", err)
		return err
	}

	return nil
}

// Rename an old directory to a new directory. If the new directory already
// exists and is non-empty, return ENOTEMPTY.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCKS_EXCLUDED(oldParent)
// LOCKS_EXCLUDED(newParent)
func (fs *fileSystem) renameDir(
	ctx context.Context,
	oldParent inode.DirInode,
	oldName string,
	newParent inode.DirInode,
	newName string) error {

	// Set up a function that throws away the lookup count increment from
	// lookUpOrCreateChildInode (since the pending inodes are not sent back to
	// the kernel) and unlocks the pending inodes, but only once
	var pendingInodes []inode.DirInode
	releaseInodes := func() {
		for _, in := range pendingInodes {
			fs.unlockAndDecrementLookupCount(in, 1)
		}
		pendingInodes = []inode.DirInode{}
	}
	defer releaseInodes()

	// Get the inode of the old directory
	oldDir, err := fs.lookUpOrCreateChildDirInode(ctx, oldParent, oldName)
	if err != nil {
		return fmt.Errorf("lookup old directory: %w", err)
	}
	pendingInodes = append(pendingInodes, oldDir)

	// Fetch all the descendants of the old directory recursively
	descendants, err := oldDir.ReadDescendants(ctx, int(fs.renameDirLimit+1))
	if err != nil {
		return fmt.Errorf("read descendants of the old directory %q: %w", oldName, err)
	}
	if len(descendants) > int(fs.renameDirLimit) {
		return fmt.Errorf("too many objects to be renamed: %w", syscall.EMFILE)
	}

	// Create the backing object of the new directory.
	newParent.Lock()
	_, err = newParent.CreateChildDir(ctx, newName)
	newParent.Unlock()
	if err != nil {
		var preconditionErr *gcs.PreconditionError
		if errors.As(err, &preconditionErr) {
			// This means the new directory already exists, which is OK if
			// it is empty (checked below).
		} else {
			return fmt.Errorf("CreateChildDir: %w", err)
		}
	}

	// Get the inode of the new directory
	newDir, err := fs.lookUpOrCreateChildDirInode(ctx, newParent, newName)
	if err != nil {
		return fmt.Errorf("lookup new directory: %w", err)
	}
	pendingInodes = append(pendingInodes, newDir)

	// Fail the operation if the new directory is non-empty.
	unexpected, err := newDir.ReadDescendants(ctx, 1)
	if err != nil {
		return fmt.Errorf("read descendants of the new directory %q: %w", newName, err)
	}
	if len(unexpected) > 0 {
		return fuse.ENOTEMPTY
	}

	// Move all the files from the old directory to the new directory, keeping
	// both directories locked.
	for _, descendant := range descendants {
		nameDiff := strings.TrimPrefix(
			descendant.FullName.GcsObjectName(), oldDir.Name().GcsObjectName())
		if nameDiff == descendant.FullName.GcsObjectName() {
			return fmt.Errorf("unwanted descendant %q not from dir %q", descendant.FullName, oldDir.Name())
		}

		o := descendant.Object
		if _, err := newDir.CloneToChildFile(ctx, nameDiff, o); err != nil {
			return fmt.Errorf("copy file %q: %w", o.Name, err)
		}
		if err := oldDir.DeleteChildFile(ctx, nameDiff, o.Generation, &o.MetaGeneration); err != nil {
			return fmt.Errorf("delete file %q: %w", o.Name, err)
		}
	}

	// We are done with both directories.
	releaseInodes()

	// Delete the backing object of the old directory.
	oldParent.Lock()
	err = oldParent.DeleteChildDir(ctx, oldName)
	oldParent.Unlock()
	if err != nil {
		return fmt.Errorf("DeleteChildDir: %w", err)
	}

	return nil
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) Unlink(
	ctx context.Context,
	op *fuseops.UnlinkOp) (err error) {
	// Find the parent.
	fs.mu.Lock()
	parent := fs.dirInodeOrDie(op.Parent)
	fs.mu.Unlock()

	parent.Lock()
	defer parent.Unlock()

	// Delete the backing object.
	err = parent.DeleteChildFile(
		ctx,
		op.Name,
		0,   // Latest generation
		nil) // No meta-generation precondition

	if err != nil {
		err = fmt.Errorf("DeleteChildFile: %w", err)
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) (err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Make sure the inode still exists and is a directory. If not, something has
	// screwed up because the VFS layer shouldn't have let us forget the inode
	// before opening it.
	in := fs.dirInodeOrDie(op.Inode)

	// Allocate a handle.
	handleID := fs.nextHandleID
	fs.nextHandleID++

	fs.handles[handleID] = newDirHandle(in, fs.implicitDirs)
	op.Handle = handleID

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) (err error) {
	// Find the handle.
	fs.mu.Lock()
	dh := fs.handles[op.Handle].(*dirHandle)
	fs.mu.Unlock()

	// Serve the request.
	if err := dh.ReadDir(ctx, op); err != nil {
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) (err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Sanity check that this handle exists and is of the correct type.
	dh := fs.handles[op.Handle].(*dirHandle)
	// Cancel context of goroutine which is fetching data.
	if dh.cancel != nil {
		dh.cancel()
	}

	// Clear the entry from the map.
	delete(fs.handles, op.Handle)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) (err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Find the inode.
	in := fs.fileInodeOrDie(op.Inode)

	// Allocate a handle.
	handleID := fs.nextHandleID
	fs.nextHandleID++

	fs.handles[handleID] = handle.NewFileHandle(in)
	op.Handle = handleID

	// When we observe object generations that we didn't create, we assign them
	// new inode IDs. So for a given inode, all modifications go through the
	// kernel. Therefore it's safe to tell the kernel to keep the page cache from
	// open to open for a given inode.
	op.KeepPageCache = true

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) (err error) {
	// Find the handle and lock it.
	fs.mu.Lock()
	fh := fs.handles[op.Handle].(*handle.FileHandle)
	fs.mu.Unlock()

	fh.Lock()
	defer fh.Unlock()

	// Serve the read.
	op.BytesRead, err = fh.Read(ctx, op.Dst, op.Offset, fs.sequentialReadSizeMb)

	// As required by fuse, we don't treat EOF as an error.
	if err == io.EOF {
		err = nil
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.symlinkInodeOrDie(op.Inode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Serve the request.
	op.Target = in.Target()

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) WriteFile(
	ctx context.Context,
	op *fuseops.WriteFileOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.fileInodeOrDie(op.Inode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Serve the request.
	if err := in.Write(ctx, op.Data, op.Offset); err != nil {
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.inodeOrDie(op.Inode)
	fs.mu.Unlock()

	file, ok := in.(*inode.FileInode)
	if !ok {
		// No-op if the target is not a file
		return
	}

	file.Lock()
	defer file.Unlock()

	// Sync it.
	if err := fs.syncFile(ctx, file); err != nil {
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) FlushFile(
	ctx context.Context,
	op *fuseops.FlushFileOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.fileInodeOrDie(op.Inode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Sync it.
	if err := fs.syncFile(ctx, in); err != nil {
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) (err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Destroy the handle.
	fs.handles[op.Handle].(*handle.FileHandle).Destroy()

	// Update the map.
	delete(fs.handles, op.Handle)

	return
}

func (fs *fileSystem) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) (err error) {
	return syscall.ENOSYS
}

func (fs *fileSystem) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) error {
	return syscall.ENOSYS
}
