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

package fs

import (
	"context"
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"math"
	"os"
	"path"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"

	"golang.org/x/sync/semaphore"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	cacheutil "github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/handle"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
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

	// NewConfig has all the config specified by the user using config-file or CLI flags.
	NewConfig *cfg.Config

	MetricHandle metrics.MetricHandle

	// Notifier allows the file system to send invalidation messages to the FUSE
	// kernel module. This enables proactive cache invalidation (e.g., for dentries)
	// when underlying content changes, improving consistency while still leveraging
	// kernel caching.
	Notifier *fuse.Notifier
}

// Create a fuse file system server according to the supplied configuration.
func NewFileSystem(ctx context.Context, serverCfg *ServerConfig) (fuseutil.FileSystem, error) {
	// Check permissions bits.
	if serverCfg.FilePerms&^os.ModePerm != 0 {
		return nil, fmt.Errorf("illegal file perms: %v", serverCfg.FilePerms)
	}

	if serverCfg.DirPerms&^os.ModePerm != 0 {
		return nil, fmt.Errorf("illegal dir perms: %v", serverCfg.FilePerms)
	}

	mtimeClock := timeutil.RealClock()

	contentCache := contentcache.New(serverCfg.TempDir, mtimeClock)

	if serverCfg.LocalFileCache {
		err := contentCache.RecoverCache()
		if err != nil {
			fmt.Printf("Encountered error retrieving files from cache directory, disabling local file cache: %v", err)
			serverCfg.LocalFileCache = false
		}
	}

	// Create file cache handler if cache is enabled by user. Cache is considered
	// enabled only if cache-dir is not empty and file-cache:max-size-mb is non 0.
	var fileCacheHandler *file.CacheHandler
	if cfg.IsFileCacheEnabled(serverCfg.NewConfig) {
		var err error
		fileCacheHandler, err = createFileCacheHandler(serverCfg)
		if err != nil {
			return nil, err
		}
	}

	// Set up the basic struct.
	fs := &fileSystem{
		mtimeClock:                 mtimeClock,
		cacheClock:                 serverCfg.CacheClock,
		bucketManager:              serverCfg.BucketManager,
		localFileCache:             serverCfg.LocalFileCache,
		contentCache:               contentCache,
		implicitDirs:               serverCfg.ImplicitDirectories,
		enableNonexistentTypeCache: serverCfg.EnableNonexistentTypeCache,
		inodeAttributeCacheTTL:     serverCfg.InodeAttributeCacheTTL,
		dirTypeCacheTTL:            serverCfg.DirTypeCacheTTL,
		kernelListCacheTTL:         cfg.ListCacheTTLSecsToDuration(serverCfg.NewConfig.FileSystem.KernelListCacheTtlSecs),
		renameDirLimit:             serverCfg.RenameDirLimit,
		sequentialReadSizeMb:       serverCfg.SequentialReadSizeMb,
		uid:                        serverCfg.Uid,
		gid:                        serverCfg.Gid,
		fileMode:                   serverCfg.FilePerms,
		dirMode:                    serverCfg.DirPerms | os.ModeDir,
		inodes:                     make(map[fuseops.InodeID]inode.Inode),
		nextInodeID:                fuseops.RootInodeID + 1,
		generationBackedInodes:     make(map[inode.Name]inode.GenerationBackedInode),
		implicitDirInodes:          make(map[inode.Name]inode.DirInode),
		folderInodes:               make(map[inode.Name]inode.DirInode),
		localFileInodes:            make(map[inode.Name]inode.Inode),
		handles:                    make(map[fuseops.HandleID]any),
		newConfig:                  serverCfg.NewConfig,
		fileCacheHandler:           fileCacheHandler,
		cacheFileForRangeRead:      serverCfg.NewConfig.FileCache.CacheFileForRangeRead,
		metricHandle:               serverCfg.MetricHandle,
		enableAtomicRenameObject:   serverCfg.NewConfig.EnableAtomicRenameObject,
		globalMaxWriteBlocksSem:    semaphore.NewWeighted(serverCfg.NewConfig.Write.GlobalMaxBlocks),
		globalMaxReadBlocksSem:     semaphore.NewWeighted(serverCfg.NewConfig.Read.GlobalMaxBlocks),
	}
	if serverCfg.Notifier != nil {
		fs.notifier = serverCfg.Notifier
	}

	if serverCfg.NewConfig.Read.EnableBufferedRead {
		var err error
		fs.bufferedReadWorkerPool, err = workerpool.NewStaticWorkerPoolForCurrentCPU(serverCfg.NewConfig.Read.GlobalMaxBlocks)
		if err != nil {
			return nil, fmt.Errorf("failed to create worker pool for buffered read: %w", err)
		}
	}

	// Set up root bucket
	var root inode.DirInode
	if serverCfg.BucketName == "" || serverCfg.BucketName == "_" {
		logger.Info("Set up root directory for all accessible buckets")
		root = makeRootForAllBuckets(fs)
	} else {
		logger.Info("Set up root directory for bucket " + serverCfg.BucketName)
		syncerBucket, err := fs.bucketManager.SetUpBucket(ctx, serverCfg.BucketName, false, fs.metricHandle)
		if err != nil {
			return nil, fmt.Errorf("SetUpBucket: %w", err)
		}
		root = makeRootForBucket(fs, syncerBucket)
	}
	root.Lock()
	root.IncrementLookupCount()
	fs.inodes[fuseops.RootInodeID] = root
	fs.implicitDirInodes[root.Name()] = root
	fs.folderInodes[root.Name()] = root
	root.Unlock()

	// Set up invariant checking.
	fs.mu = locker.New("FS", fs.checkInvariants)
	return fs, nil
}

func createFileCacheHandler(serverCfg *ServerConfig) (fileCacheHandler *file.CacheHandler, err error) {
	var sizeInBytes uint64
	// -1 means unlimited size for cache, the underlying LRU cache doesn't handle
	// -1 explicitly, hence we pass MaxUint64 as capacity in that case.
	if serverCfg.NewConfig.FileCache.MaxSizeMb == -1 {
		sizeInBytes = math.MaxUint64
	} else {
		sizeInBytes = uint64(serverCfg.NewConfig.FileCache.MaxSizeMb) * cacheutil.MiB
	}
	fileInfoCache := lru.NewCache(sizeInBytes)

	cacheDir := string(serverCfg.NewConfig.CacheDir)
	// Adding a new directory inside cacheDir to keep file-cache separate from
	// metadata cache if and when we support storing metadata cache on disk in
	// the future.
	cacheDir = path.Join(cacheDir, cacheutil.FileCache)

	filePerm := cacheutil.DefaultFilePerm
	dirPerm := cacheutil.DefaultDirPerm

	cacheDirErr := cacheutil.CreateCacheDirectoryIfNotPresentAt(cacheDir, dirPerm)
	if cacheDirErr != nil {
		return nil, fmt.Errorf("createFileCacheHandler: while creating file cache directory: %w", cacheDirErr)
	}

	jobManager := downloader.NewJobManager(fileInfoCache, filePerm, dirPerm, cacheDir, serverCfg.SequentialReadSizeMb, &serverCfg.NewConfig.FileCache, serverCfg.MetricHandle)
	fileCacheHandler = file.NewCacheHandler(fileInfoCache, jobManager, cacheDir, filePerm, dirPerm, serverCfg.NewConfig.FileCache.ExcludeRegex, serverCfg.NewConfig.FileCache.IncludeRegex)
	return
}

func makeRootForBucket(
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
		fs.newConfig.List.EnableEmptyManagedFolders,
		fs.enableNonexistentTypeCache,
		fs.dirTypeCacheTTL,
		&syncerBucket,
		fs.mtimeClock,
		fs.cacheClock,
		fs.newConfig.MetadataCache.TypeCacheMaxSizeMb,
		fs.newConfig.EnableHns,
		fs.newConfig.EnableUnsupportedDirSupport,
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
		fs.metricHandle,
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
// See https://tinyurl.com/4nh4w7u9 for more discussion, including an informal
// proof that a strict partial order is sufficient.

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

	// kernelListCacheTTL specifies the duration to keep the readdir response cached
	// in kernel. After ttl, gcsfuse, (filesystem) on next opendir call (just before as part
	// of next list call) from user, asks the kernel to evict the old cache entries.
	kernelListCacheTTL time.Duration

	renameDirLimit       int64
	sequentialReadSizeMb int32

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

	// A map from folder name to the folder inode that represents
	// that name, if any. There can be at most one folder inode for a
	// given name accessible to us at any given time.
	//
	// INVARIANT: For each k/v, v.Name() == k
	// INVARIANT: For each value v, inodes[v.ID()] == v
	//
	// GUARDED_BY(mu)
	folderInodes map[inode.Name]inode.DirInode

	// A map from object name to the local fileInode that represents
	// that name. There can be at most one local file inode for a
	// given name accessible to us at any given time.
	//
	// INVARIANT: For each k/v, v.Name() == k
	// INVARIANT: For each value v, inodes[v.ID()] == v
	// INVARIANT: For each value v, v is not fileInode
	// INVARIANT: For each f in inodes that is local fileInode,
	//            localFileInodes[f.Name()] == f
	//
	// GUARDED_BY(mu)
	localFileInodes map[inode.Name]inode.Inode

	// The collection of live handles, keyed by handle ID.
	//
	// INVARIANT: All values are of type *dirHandle or *handle.FileHandle
	//
	// GUARDED_BY(mu)
	handles map[fuseops.HandleID]any

	// The next handle ID to hand out. We assume that this will never overflow.
	//
	// INVARIANT: For all keys k in handles, k < nextHandleID
	//
	// GUARDED_BY(mu)
	nextHandleID fuseops.HandleID

	// newConfig specified by the user using config-file flag and CLI flags.
	newConfig *cfg.Config

	// fileCacheHandler manages read only file cache. It is non-nil only when
	// file cache is enabled at the time of mounting.
	fileCacheHandler *file.CacheHandler

	// cacheFileForRangeRead when true downloads file into cache even for
	// random file access.
	cacheFileForRangeRead bool

	metricHandle metrics.MetricHandle

	enableAtomicRenameObject bool

	// Limits the max number of blocks that can be created across file system when
	// streaming writes are enabled.
	globalMaxWriteBlocksSem *semaphore.Weighted

	// notifier allows sending invalidation messages to the FUSE kernel module.
	// It is used to invalidate the kernel's dentry cache,
	// providing feedback to the kernel about dynamic content changes.
	notifier *fuse.Notifier

	// bufferedReadWorkerPool is used for asynchronous prefetching of data for buffered reads.
	// It executes download tasks associated with prefetch blocks.
	bufferedReadWorkerPool workerpool.WorkerPool

	// globalMaxReadBlocksSem is a semaphore that limits the total number of blocks
	// that can be allocated for buffered read across all file-handles in the file system.
	// This helps control the overall memory usage for buffered reads.
	globalMaxReadBlocksSem *semaphore.Weighted
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (fs *fileSystem) checkInvariantsForLocalFileInodes() {
	// INVARIANT: For each k/v, v.Name() == k
	for k, v := range fs.localFileInodes {
		if !(v.Name() == k) {
			panic(fmt.Sprintf(
				"Unexpected name: \"%s\" vs. \"%s\"",
				v.Name(),
				k))
		}
	}

	// INVARIANT: For each value v, inodes[v.ID()] == v
	for _, v := range fs.localFileInodes {
		if fs.inodes[v.ID()] != v {
			panic(fmt.Sprintf(
				"Mismatch for ID %v: %v %v",
				v.ID(),
				fs.inodes[v.ID()],
				v))
		}
	}

	// INVARIANT: For each value v, v is not fileInode
	for _, v := range fs.localFileInodes {
		if _, ok := v.(*inode.FileInode); !ok {
			panic(fmt.Sprintf(
				"Unexpected file inode %d, type %T",
				v.ID(),
				v))
		}
	}

	// INVARIANT: For each f in inodes that is local fileInode
	//            localFileInodes[d.Name()] == f
	for _, in := range fs.inodes {
		fileInode, ok := in.(*inode.FileInode)

		if ok && fileInode.IsLocal() && !fileInode.IsUnlinked() {
			if !(fs.localFileInodes[in.Name()] == in) {
				panic(fmt.Sprintf(
					"localFileInodes mismatch: %q %v %v",
					in.Name(),
					fs.localFileInodes[in.Name()],
					in))
			}
		}
	}
}

func (fs *fileSystem) checkInvariantsForFolderInodes() {
	// INVARIANT: For each k/v, v.Name() == k
	for k, v := range fs.folderInodes {
		if !(v.Name() == k) {
			panic(fmt.Sprintf(
				"Unexpected name: \"%s\" vs. \"%s\"",
				v.Name(),
				k))
		}
	}

	// INVARIANT: For each value v, inodes[v.ID()] == v
	for _, v := range fs.folderInodes {
		if fs.inodes[v.ID()] != v {
			panic(fmt.Sprintf(
				"Mismatch for ID %v: %v %v",
				v.ID(),
				fs.inodes[v.ID()],
				v))
		}
	}
}

func (fs *fileSystem) checkInvariantsForImplicitDirs() {
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
}

func (fs *fileSystem) checkInvariantsForGenerationBackedInodes() {
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
}

func (fs *fileSystem) checkInvariantsForInodes() {
	// INVARIANT: For all keys k, fuseops.RootInodeID <= k < nextInodeID
	for id := range fs.inodes {
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
}

func (fs *fileSystem) checkInvariants() {
	// Check invariants for different type of inodes
	fs.checkInvariantsForInodes()
	fs.checkInvariantsForGenerationBackedInodes()
	fs.checkInvariantsForImplicitDirs()
	fs.checkInvariantsForFolderInodes()
	fs.checkInvariantsForLocalFileInodes()

	//////////////////////////////////
	// handles
	//////////////////////////////////

	// INVARIANT: All values are of type *dirHandle or *handle.FileHandle
	for _, h := range fs.handles {
		switch h.(type) {
		case *handle.DirHandle:
		case *handle.FileHandle:
		default:
			panic(fmt.Sprintf("Unexpected handle type: %T", h))
		}
	}

	//////////////////////////////////
	// nextHandleID
	//////////////////////////////////

	// INVARIANT: For all keys k in handles, k < nextHandleID
	for k := range fs.handles {
		if k >= fs.nextHandleID {
			panic(fmt.Sprintf("Illegal handle ID: %v", k))
		}
	}
}

func (fs *fileSystem) createExplicitDirInode(inodeID fuseops.InodeID, ic inode.Core) inode.Inode {
	in := inode.NewExplicitDirInode(
		inodeID,
		ic.FullName,
		ic.MinObject,
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
		fs.newConfig.List.EnableEmptyManagedFolders,
		fs.enableNonexistentTypeCache,
		fs.dirTypeCacheTTL,
		ic.Bucket,
		fs.mtimeClock,
		fs.cacheClock,
		fs.newConfig.MetadataCache.TypeCacheMaxSizeMb,
		fs.newConfig.EnableHns,
		fs.newConfig.EnableUnsupportedDirSupport)

	return in
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
	// Explicit directories or folders in hierarchical bucket.
	case (ic.MinObject != nil && ic.FullName.IsDir()), ic.Folder != nil:
		in = fs.createExplicitDirInode(id, ic)

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
			fs.newConfig.List.EnableEmptyManagedFolders,
			fs.enableNonexistentTypeCache,
			fs.dirTypeCacheTTL,
			ic.Bucket,
			fs.mtimeClock,
			fs.cacheClock,
			fs.newConfig.MetadataCache.TypeCacheMaxSizeMb,
			fs.newConfig.EnableHns,
			fs.newConfig.EnableUnsupportedDirSupport,
		)

	case inode.IsSymlink(ic.MinObject):
		in = inode.NewSymlinkInode(
			id,
			ic.FullName,
			ic.Bucket,
			ic.MinObject,
			fuseops.InodeAttributes{
				Uid:  fs.uid,
				Gid:  fs.gid,
				Mode: fs.fileMode | os.ModeSymlink,
			})

	default:
		in = inode.NewFileInode(
			id,
			ic.FullName,
			ic.MinObject,
			fuseops.InodeAttributes{
				Uid:  fs.uid,
				Gid:  fs.gid,
				Mode: fs.fileMode,
			},
			ic.Bucket,
			fs.localFileCache,
			fs.contentCache,
			fs.mtimeClock,
			ic.Local,
			fs.newConfig,
			fs.globalMaxWriteBlocksSem)
	}

	// Place it in our map of IDs to inodes.
	fs.inodes[in.ID()] = in

	return
}

// Return the dir Inode.
//
// LOCKS_EXCLUDED(fs.mu)
// UNLOCK_FUNCTION(fs.mu)
// LOCK_FUNCTION(in)
func (fs *fileSystem) createDirInode(ic inode.Core, inodes map[inode.Name]inode.DirInode) inode.Inode {
	if !ic.FullName.IsDir() {
		panic(fmt.Sprintf("Unexpected name for a directory: %q", ic.FullName))
	}

	var maxTriesToCreateInode = 3

	for range maxTriesToCreateInode {
		in, ok := (inodes)[ic.FullName]
		// Create a new inode when a folder is created first time, or when a folder is deleted and then recreated with the same name.
		if !ok || in.IsUnlinked() {
			in := fs.mintInode(ic)
			(inodes)[in.Name()] = in.(inode.DirInode)
			in.Lock()
			return in
		}

		fs.mu.Unlock()
		in.Lock()
		fs.mu.Lock()

		if (inodes)[ic.FullName] != in {
			in.Unlock()
			continue
		}

		return in
	}

	return nil
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

	// Handle Folders in hierarchical bucket.
	if ic.Folder != nil {
		return fs.createDirInode(ic, fs.folderInodes)
	}

	// Handle implicit directories.
	if ic.MinObject == nil {
		return fs.createDirInode(ic, fs.implicitDirInodes)
	}

	oGen := inode.Generation{
		Object:   ic.MinObject.Generation,
		Metadata: ic.MinObject.MetaGeneration,
		Size:     ic.MinObject.Size,
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
	// First check if the requested child is a localFileInode.
	child, err = fs.lookUpLocalFileInode(parent, childName)
	if err != nil {
		return nil, err
	}
	if child != nil {
		return
	}

	// If the requested child is not a localFileInode, continue with the existing
	// flow of checking GCS for file/directory.

	// Set up a function that will find a lookup result for the child with the
	// given name. Expects no locks to be held.
	getLookupResult := func() (*inode.Core, error) {
		if fs.newConfig.FileSystem.DisableParallelDirops {
			parent.Lock()
			defer parent.Unlock()
		} else {
			// LockForChildLookup takes read-only or exclusive lock based on the
			// inode when its child is looked up.
			parent.LockForChildLookup()
			defer parent.UnlockForChildLookup()
		}
		return parent.LookUpChild(ctx, childName)
	}

	// Run a retry loop around lookUpOrCreateInodeIfNotStale.
	const maxTries = 3
	for range maxTries {
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

// Look up the localFileInodes to check if a file with given name exists.
// Return inode if it exists, else return nil.
// LOCKS_EXCLUDED(fs.mu)
// LOCKS_EXCLUDED(parent)
// UNLOCK_FUNCTION(fs.mu)
// LOCK_FUNCTION(child)
func (fs *fileSystem) lookUpLocalFileInode(parent inode.DirInode, childName string) (child inode.Inode, err error) {
	// If the path specified is "a/\n", the child would come as \n which is not a valid childname.
	// In such cases, simply return a file-not-found.
	if childName == inode.ConflictingFileNameSuffix {
		return nil, syscall.ENOENT
	}
	// Trim the suffix assigned to fix conflicting names.
	childName = strings.TrimSuffix(childName, inode.ConflictingFileNameSuffix)
	fileName := inode.NewFileName(parent.Name(), childName)

	fs.mu.Lock()
	defer func() {
		if child != nil {
			child.IncrementLookupCount()
		}
		fs.mu.Unlock()
	}()

	var maxTriesToLookupInode = 3
	for range maxTriesToLookupInode {
		child = fs.localFileInodes[fileName]

		if child == nil {
			return
		}

		// If the inode already exists, we need to follow the lock ordering rules
		// to get the lock. First get inode lock and then fs lock.
		fs.mu.Unlock()
		child.Lock()
		// Acquiring fs lock early to use common defer function even though it is
		// not required to check if local file inode has been unlinked.
		// Filesystem lock will be held till we increment lookUpCount to avoid
		// deletion of inode from fs.inodes/fs.localFileInodes map by other flows.
		fs.mu.Lock()
		// Check if local file inode has been unlinked?
		fileInode, ok := child.(*inode.FileInode)
		if ok && fileInode.IsUnlinked() {
			child.Unlock()
			child = nil
			return
		}
		// Once we get fs lock, validate if the inode is still valid. If not
		// try to fetch it again. Eg: If the inode is deleted by other thread after
		// we fetched it from fs.localFileInodes map, then any call to perform
		// inode operation will crash GCSFuse since the inode is not valid. Hence
		// it is important to acquire lock and increment lookUpCount before letting
		// other threads modify it.
		if fs.localFileInodes[fileName] != child {
			child.Unlock()
			continue
		}

		return
	}

	// In case we exhausted the retries, return nil object.
	child = nil
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

// promoteToGenerationBacked updates the file system maps for the given file inode
// after it has been synced to GCS.
// The inode is removed from the localFileInodes map and added to the
// generationBackedInodes map.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCKS_REQUIRED(f)
func (fs *fileSystem) promoteToGenerationBacked(f *inode.FileInode) {
	fs.mu.Lock()
	delete(fs.localFileInodes, f.Name())
	if _, ok := fs.generationBackedInodes[f.Name()]; !ok {
		fs.generationBackedInodes[f.Name()] = f
	}
	fs.mu.Unlock()

	// We need not update fileIndex:
	//
	// We've held the inode lock the whole time, so there's no way that this
	// inode could have been booted from the index. Therefore, if it's not in the
	// index at the moment, it must not have been in there when we started. That
	// is, it must have been clobbered remotely.
	//
	// In other words, either this inode is still in the index or it has been
	// clobbered and *should* be anonymous.
}

// Flushes the supplied file inode to GCS, updating the index as
// appropriate.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCKS_REQUIRED(f)
func (fs *fileSystem) flushFile(
	ctx context.Context,
	f *inode.FileInode) error {
	// FlushFile mirrors the behavior of native filesystems by not returning an error
	// when file to be synced has been unlinked from the same mount.
	if f.IsUnlinked() {
		return nil
	}

	// Flush the inode.
	err := f.Flush(ctx)
	if err != nil {
		err = fmt.Errorf("FileInode.Sync: %w", err)
		// If the inode was local file inode, treat it as unlinked.
		fs.mu.Lock()
		delete(fs.localFileInodes, f.Name())
		fs.mu.Unlock()
		return err
	}

	// Promote the inode to generationBackedInodes in fs maps.
	fs.promoteToGenerationBacked(f)
	return nil
}

// Synchronizes the supplied file inode to GCS, updating the index as
// appropriate.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCKS_REQUIRED(f)
func (fs *fileSystem) syncFile(
	ctx context.Context,
	f *inode.FileInode) error {
	// SyncFile mirrors the behavior of native filesystems by not returning an error
	// when file to be synced has been unlinked from the same mount.
	if f.IsUnlinked() {
		return nil
	}

	// Sync the inode.
	gcsSynced, err := f.Sync(ctx)
	if err != nil {
		err = fmt.Errorf("FileInode.Sync: %w", err)
		// If the inode was local file inode, treat it as unlinked.
		fs.mu.Lock()
		delete(fs.localFileInodes, f.Name())
		fs.mu.Unlock()
		return err
	}

	// If gcsSynced is true, it means the inode was fully synced to GCS In this
	// case, we need to promote the inode to generationBackedInodes in fs maps.
	if gcsSynced {
		fs.promoteToGenerationBacked(f)
	}
	return nil
}

// Initializes Buffered Write Handler if Eligible and synchronizes the file inode to GCS if initialization succeeds.
// Otherwise creates an empty temp writer if temp file nil.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCKS_REQUIRED(f.mu)
func (fs *fileSystem) createBufferedWriteHandlerAndSyncOrTempWriter(ctx context.Context, f *inode.FileInode, openMode util.OpenMode) error {
	err := fs.initBufferedWriteHandlerAndSyncFileIfEligible(ctx, f, openMode)
	if err != nil {
		return err
	}
	err = f.CreateEmptyTempFile(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Initializes Buffered Write Handler if Eligible and synchronizes the file inode to GCS if initialization succeeds.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCKS_REQUIRED(f.mu)
func (fs *fileSystem) initBufferedWriteHandlerAndSyncFileIfEligible(ctx context.Context, f *inode.FileInode, openMode util.OpenMode) error {
	initialized, err := f.InitBufferedWriteHandlerIfEligible(ctx, openMode)
	if err != nil {
		return err
	}
	if initialized {
		// Calling syncFile is safe here as we have file inode lock and BWH is initialized.
		// Thus sync method of BWH will be invoked.
		// 1. In case of zonal bucket it creates unfinalized new generation object.
		// 2. In case of non zonal bucket it's no-op as we don't have pending buffers to upload.
		err = fs.syncFile(ctx, f)
		if err != nil {
			return err
		}
	}
	return nil
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
		if fs.localFileInodes[name] == in {
			delete(fs.localFileInodes, name)
		}
		if fs.folderInodes[name] == in {
			delete(fs.folderInodes, name)
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
	attr, err = in.Attributes(ctx, true)
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

// invalidateChildFileCacheIfExist invalidates the file in read cache. This is used to
// invalidate the file in read cache after deletion of original file.
func (fs *fileSystem) invalidateChildFileCacheIfExist(parentInode inode.DirInode, objectGCSName string) (err error) {
	if fs.fileCacheHandler != nil {
		if bucketOwnedDirInode, ok := parentInode.(inode.BucketOwnedDirInode); ok {
			bucketName := bucketOwnedDirInode.Bucket().Name()
			// Invalidate the file cache entry if it exists.
			err := fs.fileCacheHandler.InvalidateCache(objectGCSName, bucketName)
			if err != nil {
				return fmt.Errorf("invalidateChildFileCacheIfExist: while invalidating the file cache: %w", err)
			}
		} else {
			// The parentInode is not owned by any bucket, which means it's the base
			// directory that holds all the buckets' root directories. So, this op
			// is to delete a bucket, which is not supported.
			return fmt.Errorf("invalidateChildFileCacheIfExist: not an BucketOwnedDirInode: %w", syscall.ENOTSUP)
		}
	}

	return nil
}

// coreToDirentPlus creates a fuseutil.DirentPlus entry from an inode core.
// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) coreToDirentPlus(ctx context.Context, fullName inode.Name, core inode.Core) (entryPlus *fuseutil.DirentPlus, err error) {
	// Look up or create the inode for the core.
	child := fs.lookUpOrCreateInodeIfNotStale(core)
	if child == nil {
		return nil, fmt.Errorf("coreToDirentPlus: stale record for %s", path.Base(fullName.LocalName()))
	}
	defer child.Unlock()

	// Extract the child's attributes.
	attributes, err := child.Attributes(ctx, false)
	if err != nil {
		// The inode is valid, but we couldn't get attributes.
		return nil, fmt.Errorf("coreToDirentPlus: unable to fetch attributes for %s: %w", path.Base(fullName.LocalName()), err)
	}

	expiration := time.Now().Add(fs.inodeAttributeCacheTTL)
	entryPlus = &fuseutil.DirentPlus{
		Dirent: fuseutil.Dirent{
			Name:  path.Base(fullName.LocalName()),
			Type:  fuseutil.DT_Unknown,
			Inode: child.ID(),
		},
		Entry: fuseops.ChildInodeEntry{
			Child:                child.ID(),
			Attributes:           attributes,
			AttributesExpiration: expiration,
		},
	}
	if fs.newConfig.FileSystem.ExperimentalEnableDentryCache {
		entryPlus.Entry.EntryExpiration = expiration
	}

	// Set the directory entry type based on the core type.
	switch core.Type() {
	case metadata.SymlinkType:
		entryPlus.Dirent.Type = fuseutil.DT_Link
	case metadata.RegularFileType:
		entryPlus.Dirent.Type = fuseutil.DT_File
	case metadata.ImplicitDirType, metadata.ExplicitDirType:
		entryPlus.Dirent.Type = fuseutil.DT_Directory
	}

	return entryPlus, nil
}

// LocalFileEntries lists the local files (file that is not yet present on GCS) present in the directory.
// For each entry, only the Dirent field is populated; the ChildInodeEntry field is not set.
// LOCKS_REQUIRED(fs.mu)
func (fs *fileSystem) localFileEntriesPlus(parent inode.Name) (localEntriesPlus map[string]fuseutil.DirentPlus) {
	localEntriesPlus = make(map[string]fuseutil.DirentPlus)

	for localInodeName, in := range fs.localFileInodes {
		// It is possible that the local file inode has been unlinked, but
		// still present in localFileInodes map because of open file handle.
		// So, if the inode has been unlinked, skip the entry.
		file, ok := in.(*inode.FileInode)
		if ok && file.IsUnlinked() {
			continue
		}
		if localInodeName.IsDirectChildOf(parent) {
			entry := fuseutil.DirentPlus{
				Dirent: fuseutil.Dirent{
					Name:  path.Base(localInodeName.LocalName()),
					Type:  fuseutil.DT_File,
					Inode: in.ID(),
				},
			}
			localEntriesPlus[entry.Dirent.Name] = entry
		}
	}
	return
}

// lookupAndFetchAttributesForLocalFileEntriesPlus performs a lookup for each local file entry,
// fetches its attributes, and updates the corresponding DirentPlus.Entry field.
// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) lookupAndFetchAttributesForLocalFileEntriesPlus(parentName inode.DirInode, localFileEntriesPlus map[string]fuseutil.DirentPlus) (err error) {
	for localEntryName, localEntryPlus := range localFileEntriesPlus {
		// Lookup the child inode under the parent directory.
		child, err := fs.lookUpLocalFileInode(parentName, localEntryName)
		if err != nil {
			return fmt.Errorf("lookupAndFetchAttributesForLocalFileEntriesPlus: while looking up local file %q: %w", localEntryName, err)
		}
		if child == nil {
			// This indicates a potential race condition where the file was removed after being listed.
			return fmt.Errorf("lookupAndFetchAttributesForLocalFileEntriesPlus: local file %q disappeared", localEntryName)
		}
		// Fetch attributes from the child inode.
		attrs, err := child.Attributes(context.Background(), false)
		if err != nil {
			child.Unlock()
			return fmt.Errorf("lookupAndFetchAttributesForLocalFileEntriesPlus: unable to fetch attributes for %s: %w", localEntryName, err)
		}
		// Unlock the inode after retrieving its attributes.
		child.Unlock()

		expiration := time.Now().Add(fs.inodeAttributeCacheTTL)
		childInodeEntry := fuseops.ChildInodeEntry{
			Child:                child.ID(),
			Attributes:           attrs,
			AttributesExpiration: expiration,
		}
		if fs.newConfig.FileSystem.ExperimentalEnableDentryCache {
			childInodeEntry.EntryExpiration = expiration
		}
		localEntryPlus.Entry = childInodeEntry
		localFileEntriesPlus[localEntryName] = localEntryPlus
	}
	return
}

// invalidateCachedEntry sends a notification to the kernel to invalidate a stale
// directory entry, ensuring consistency when file content changes dynamically.
// It identifies the parent of the given child inode and sends a notification
// to the kernel to remove the entry corresponding to the child's
// name within that parent directory.
//
// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) invalidateCachedEntry(childID fuseops.InodeID) error {
	fs.mu.Lock()
	childInode, ok := fs.inodes[childID]
	fs.mu.Unlock()
	if !ok {
		return fmt.Errorf("invalidateCachedEntry: inode with ID %d not found", childID)
	}

	childName := childInode.Name()
	parentPath := path.Dir(childName.LocalName())
	// If the parent path resolves to the current directory ".", it means the parent
	// is the root of the file system.
	if parentPath == "." {
		return fs.notifier.InvalidateEntry(fuseops.RootInodeID, path.Base(childInode.Name().LocalName()))
	}

	parentName, err := childName.ParentName()
	if err != nil {
		return fmt.Errorf("invalidateCachedEntry: cannot find Parent name: %w", err)
	}
	childBase := path.Base(childName.LocalName())

	fs.mu.Lock()
	defer fs.mu.Unlock()

	var parentInodeID fuseops.InodeID
	// Check in all maps: implicit dirs → folders → generation-backed
	if parentInode, ok := fs.implicitDirInodes[parentName]; ok {
		parentInodeID = parentInode.ID()
	} else if parentInode, ok := fs.folderInodes[parentName]; ok {
		parentInodeID = parentInode.ID()
	} else if parentInode, ok := fs.generationBackedInodes[parentName]; ok {
		parentInodeID = parentInode.ID()
	} else {
		return fmt.Errorf("invalidateCachedEntry: failed to invalidate the entry, parent inode not found for child ID %d (parent: %s)", childID, parentName.String())
	}

	return fs.notifier.InvalidateEntry(parentInodeID, childBase)
}

////////////////////////////////////////////////////////////////////////
// fuse.FileSystem methods
////////////////////////////////////////////////////////////////////////

func (fs *fileSystem) Destroy() {
	fs.bucketManager.ShutDown()
	if fs.fileCacheHandler != nil {
		_ = fs.fileCacheHandler.Destroy()
	}
	if fs.bufferedReadWorkerPool != nil {
		fs.bufferedReadWorkerPool.Stop()
	}
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
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}
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
	if fs.newConfig.FileSystem.ExperimentalEnableDentryCache {
		e.EntryExpiration = e.AttributesExpiration
	}

	if err != nil {
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) (err error) {
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}
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
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}
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
		// Initialize BWH if eligible and Sync file inode.
		err = fs.initBufferedWriteHandlerAndSyncFileIfEligible(ctx, file, util.Write)
		if err != nil {
			return
		}
		gcsSynced, err := file.Truncate(ctx, int64(*op.Size))
		// Sync the inode if finalize during truncate is successful
		// even if the truncate operation later resulted error.
		if gcsSynced {
			fs.promoteToGenerationBacked(file)
		}
		if err != nil {
			err = fmt.Errorf("truncate: %w", err)
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
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}
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
		err = fmt.Errorf("newly-created record is already stale")
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
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}
	if (op.Mode & (iofs.ModeNamedPipe | iofs.ModeSocket)) != 0 {
		return syscall.ENOTSUP
	}

	// Create the child.
	child, err := fs.createFile(ctx, op.Parent, op.Name)
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
	name string) (child inode.Inode, err error) {
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
		err = fmt.Errorf("newly-created record is already stale")
		return
	}

	return
}

// Creates localFileInode with the given name under the parent inode.
// LOCKS_EXCLUDED(fs.mu)
// UNLOCK_FUNCTION(fs.mu)
// LOCK_FUNCTION(child)
func (fs *fileSystem) createLocalFile(ctx context.Context, parentID fuseops.InodeID, name string) (child inode.Inode, err error) {
	// Find the parent.
	fs.mu.Lock()
	parent := fs.dirInodeOrDie(parentID)

	defer func() {
		if err != nil {
			if child == nil {
				return
			}
			// fs.mu lock is already taken
			delete(fs.localFileInodes, child.Name())
		}
		// We need to release the filesystem lock before acquiring the inode lock.
		fs.mu.Unlock()

		if child != nil {
			child.Lock()
			child.IncrementLookupCount()
			// Unlock is done by the calling method.
		}
	}()

	fullName := inode.NewFileName(parent.Name(), name)
	child, ok := fs.localFileInodes[fullName]

	if ok && !child.(*inode.FileInode).IsUnlinked() {
		return
	}

	// Create a new inode when a file is created first time, or when a local file is unlinked and then recreated with the same name.
	core, err := parent.CreateLocalChildFileCore(name)
	if err != nil {
		return
	}
	child = fs.mintInode(core)
	fs.localFileInodes[child.Name()] = child
	fileInode := child.(*inode.FileInode)
	// Use deferred locking on filesystem so that it is locked before the defer call that unlocks the mutex and it doesn't fail.
	// We need to release the filesystem lock before acquiring the inode lock.
	fs.mu.Unlock()
	defer fs.mu.Lock()
	fileInode.Lock()
	err = fs.createBufferedWriteHandlerAndSyncOrTempWriter(ctx, fileInode, util.Write)
	fileInode.Unlock()
	if err != nil {
		return
	}

	parent.Lock()
	defer parent.Unlock()
	parent.InsertFileIntoTypeCache(name)
	return child, nil
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) (err error) {
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}
	// Create the child.
	var child inode.Inode
	if fs.newConfig.Write.CreateEmptyFile {
		child, err = fs.createFile(ctx, op.Parent, op.Name)
	} else {
		child, err = fs.createLocalFile(ctx, op.Parent, op.Name)
	}

	if err != nil {
		return err
	}

	defer fs.unlockAndMaybeDisposeOfInode(child, &err)

	// Allocate a handle.
	fs.mu.Lock()

	handleID := fs.nextHandleID
	fs.nextHandleID++

	// CreateFile() invoked to create new files, can be safely considered as filehandle
	// opened in append mode.
	fs.handles[handleID] = handle.NewFileHandle(child.(*inode.FileInode), fs.fileCacheHandler, fs.cacheFileForRangeRead, fs.metricHandle, util.Append, fs.newConfig, fs.bufferedReadWorkerPool, fs.globalMaxReadBlocksSem)
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
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}
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
		err = fmt.Errorf("newly-created record is already stale")
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
	// When rm -r or os.RemoveAll call is made, the following calls are made in order
	//	 1. RmDir (only in the case of os.RemoveAll)
	//	 2. Unlink all nested files,
	//	 3. lookupInode call on implicit directory
	//	 4. Rmdir on the directory.
	//
	// When type cache ttl is set, we construct an implicitDir even though one doesn't
	// exist on GCS (https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/internal/fs/inode/dir.go#L452),
	// and thus, we get rmDir call to GCSFuse.
	// Whereas when ttl is zero, lookupInode call itself fails and RmDir is not called
	// because object is not present in GCS.

	ctx context.Context,
	op *fuseops.RmDirOp) (err error) {
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}
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

	// Check for local file entries.
	fs.mu.Lock()
	localFileEntries := childDir.LocalFileEntries(fs.localFileInodes)
	fs.mu.Unlock()
	// Are there any local entries?
	if len(localFileEntries) != 0 {
		err = fuse.ENOTEMPTY
		return
	}

	// Check for entries on GCS.
	var tok string
	for {
		var entries []fuseutil.Dirent
		var unsupportedDirs []string
		entries, unsupportedDirs, tok, err = childDir.ReadEntries(ctx, tok)
		if err != nil {
			err = fmt.Errorf("ReadEntries: %w", err)
			return err
		}

		// If there are unsupported objects, delete them recursively.
		if len(unsupportedDirs) > 0 {
			err = childDir.DeleteObjects(ctx, unsupportedDirs)
			if err != nil {
				return fmt.Errorf("RmDir: failed to delete unsupported objects: %w", err)
			}
			// After deleting, we need to re-check for emptiness.
			continue
		}

		if fs.kernelListCacheTTL > 0 {
			// Clear kernel list cache after removing a directory. This ensures remote
			// GCS files are included in future directory listings for unlinking.
			childDir.InvalidateKernelListCache()
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
	fs.mu.Lock()
	_, isImplicitDir := fs.implicitDirInodes[child.Name()]
	fs.mu.Unlock()
	parent.Lock()
	err = parent.DeleteChildDir(ctx, op.Name, isImplicitDir, childDir)
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
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}
	// Find the old and new parents.
	fs.mu.Lock()
	oldParent := fs.dirInodeOrDie(op.OldParent)
	newParent := fs.dirInodeOrDie(op.NewParent)
	fs.mu.Unlock()

	if oldParentInode, ok := oldParent.(inode.BucketOwnedInode); !ok {
		// The old parent is not owned by any bucket, which means it's the base
		// directory that holds all the buckets' root directories. So, this op
		// is to rename a bucket, which is not supported.
		return fmt.Errorf("rename a bucket: %w", syscall.ENOTSUP)
	} else {
		// The target path must exist in the same bucket.
		oldBucket := oldParentInode.Bucket().Name()
		if newParentInode, ok := newParent.(inode.BucketOwnedInode); !ok || oldBucket != newParentInode.Bucket().Name() {
			return fmt.Errorf("move out of bucket %q: %w", oldBucket, syscall.ENOTSUP)
		}
	}

	child, err := fs.lookUpOrCreateChildInode(ctx, oldParent, op.OldName)
	if err != nil {
		return err
	}
	if child == nil {
		return fuse.ENOENT
	}
	child.DecrementLookupCount(1)
	child.Unlock()

	childBktOwned, ok := child.(inode.BucketOwnedInode)
	if !ok { // Won't happen in ideal case.
		return fmt.Errorf("child inode (id %v) is not owned by any bucket", child.ID())
	}

	if child.Name().IsDir() {
		// If 'enable-hns' flag is false, the bucket type is set to 'NonHierarchical' even for HNS buckets because the control client is nil.
		// Therefore, an additional 'enable hns' check is not required here.
		if childBktOwned.Bucket().BucketType().Hierarchical {
			return fs.renameHierarchicalDir(ctx, oldParent, op.OldName, newParent, op.NewName)
		}
		return fs.renameNonHierarchicalDir(ctx, oldParent, op.OldName, newParent, op.NewName)
	}

	return fs.renameFile(ctx, op, childBktOwned, oldParent, newParent)
}

// LOCKS_EXCLUDED(oldParent)
// LOCKS_EXCLUDED(newParent)
func (fs *fileSystem) renameFile(ctx context.Context, op *fuseops.RenameOp, child inode.BucketOwnedInode, oldParent, newParent inode.DirInode) error {
	var updatedMinObject *gcs.MinObject
	var err error

	switch c := child.(type) {
	case *inode.FileInode:
		updatedMinObject, err = fs.flushPendingWrites(ctx, c)
		if err != nil {
			return fmt.Errorf("flushPendingWrites: %w", err)
		}
	case *inode.SymlinkInode:
		updatedMinObject = c.Source()
	default:
		return fmt.Errorf("child inode (id %v) is not a file or symlink inode", child.ID())
	}
	if fs.enableAtomicRenameObject || child.Bucket().BucketType().Zonal {
		return fs.atomicRename(ctx, oldParent, op.OldName, updatedMinObject, newParent, op.NewName)
	}
	return fs.nonAtomicRename(ctx, oldParent, op.OldName, updatedMinObject, newParent, op.NewName)
}

// LOCKS_EXCLUDED(fileInode)
func (fs *fileSystem) flushPendingWrites(ctx context.Context, fileInode *inode.FileInode) (minObject *gcs.MinObject, err error) {
	// We will return modified minObject if flush is done, otherwise the original
	// minObject is returned. Original minObject is the one passed in the request.
	fileInode.Lock()
	defer fileInode.Unlock()
	minObject = fileInode.Source()
	// Try to flush if there are any pending writes.
	err = fs.flushFile(ctx, fileInode)
	minObject = fileInode.Source()
	return
}

// LOCKS_EXCLUDED(oldParent)
// LOCKS_EXCLUDED(newParent)
func (fs *fileSystem) atomicRename(ctx context.Context, oldParent inode.DirInode, oldName string, oldObject *gcs.MinObject, newParent inode.DirInode, newName string) error {
	oldParent.Lock()
	defer oldParent.Unlock()

	if newParent != oldParent {
		newParent.Lock()
		defer newParent.Unlock()
	}

	newFileName := inode.NewFileName(newParent.Name(), newName)

	if _, err := oldParent.RenameFile(ctx, oldObject, newFileName.GcsObjectName()); err != nil {
		return fmt.Errorf("renameFile: while renaming file: %w", err)
	}

	if err := fs.invalidateChildFileCacheIfExist(oldParent, oldName); err != nil {
		return fmt.Errorf("atomicRename: while invalidating cache for renamed file: %w", err)
	}

	// Insert new file in type cache.
	newParent.InsertFileIntoTypeCache(newName)

	return nil
}

// LOCKS_EXCLUDED(oldParent)
// LOCKS_EXCLUDED(newParent)
func (fs *fileSystem) nonAtomicRename(
	ctx context.Context,
	oldParent inode.DirInode,
	oldName string,
	oldObject *gcs.MinObject,
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

	if err := fs.invalidateChildFileCacheIfExist(oldParent, oldObject.Name); err != nil {
		return fmt.Errorf("nonAtomicRename: while invalidating cache for delete file: %w", err)
	}

	oldParent.Unlock()

	if err != nil {
		err = fmt.Errorf("DeleteChildFile: %w", err)
		return err
	}

	return nil
}

func (fs *fileSystem) releaseInodes(inodes *[]inode.DirInode) {
	for _, in := range *inodes {
		fs.unlockAndDecrementLookupCount(in, 1)
	}
	*inodes = []inode.DirInode{}
}

func (fs *fileSystem) getBucketDirInode(ctx context.Context, parent inode.DirInode, name string) (inode.BucketOwnedDirInode, error) {
	dir, err := fs.lookUpOrCreateChildDirInode(ctx, parent, name)
	if err != nil {
		return nil, fmt.Errorf("lookup directory: %w", err)
	}
	return dir, nil
}

func (fs *fileSystem) ensureNoLocalFilesInDirectory(dir inode.BucketOwnedDirInode, name string) error {
	fs.mu.Lock()
	entries := dir.LocalFileEntries(fs.localFileInodes)
	fs.mu.Unlock()

	if len(entries) != 0 {
		return fmt.Errorf("can't rename directory %s with open files: %w", name, syscall.ENOTSUP)
	}
	return nil
}

func (fs *fileSystem) checkDirNotEmpty(dir inode.BucketOwnedDirInode, name string) error {
	unexpected, err := dir.ReadDescendants(context.Background(), 1)
	if err != nil {
		return fmt.Errorf("read descendants of the new directory %q: %w", name, err)
	}

	if len(unexpected) > 0 {
		return fuse.ENOTEMPTY
	}
	return nil
}

// Rename an old folder to a new folder in a hierarchical bucket. If the new folder already
// exists and is non-empty, return ENOTEMPTY. If old folder have open files then return
// ENOTSUP.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCKS_EXCLUDED(oldParent)
// LOCKS_EXCLUDED(newParent)
func (fs *fileSystem) renameHierarchicalDir(ctx context.Context, oldParent inode.DirInode, oldName string, newParent inode.DirInode, newName string) (err error) {
	// Set up a function that throws away the lookup count increment from
	// lookUpOrCreateChildInode (since the pending inodes are not sent back to
	// the kernel) and unlocks the pending inodes, but only once.
	var pendingInodes []inode.DirInode
	defer fs.releaseInodes(&pendingInodes)

	oldDirInode, err := fs.getBucketDirInode(ctx, oldParent, oldName)
	if err != nil {
		return err
	}
	pendingInodes = append(pendingInodes, oldDirInode)

	if err = fs.ensureNoLocalFilesInDirectory(oldDirInode, oldName); err != nil {
		return err
	}

	oldDirName := inode.NewDirName(oldParent.Name(), oldName)
	newDirName := inode.NewDirName(newParent.Name(), newName)

	// If the call for getBucketDirInode fails it means directory does not exist.
	newDirInode, err := fs.getBucketDirInode(ctx, newParent, newName)
	if err == nil {
		// If the directory exists, then check if it is empty or not.
		if err = fs.checkDirNotEmpty(newDirInode, newName); err != nil {
			return err
		}

		// This refers to an empty destination directory.
		// The RenameFolder API does not allow renaming to an existing empty directory.
		// To make this work, we delete the empty directory first from gcsfuse and then perform rename.
		newParent.Lock()
		_ = newParent.DeleteChildDir(ctx, newName, false, newDirInode)
		newParent.Unlock()
		pendingInodes = append(pendingInodes, newDirInode)
	}

	// Note:The renameDirLimit is not utilized in the folder rename operation because there is no user-defined limit on new renames.
	oldParent.Lock()
	defer oldParent.Unlock()

	if newParent != oldParent {
		newParent.Lock()
		defer newParent.Unlock()
	}

	// Rename old directory to the new directory, keeping both parent directories locked.
	_, err = oldParent.RenameFolder(ctx, oldDirName.GcsObjectName(), newDirName.GcsObjectName())
	if err != nil {
		return fmt.Errorf("failed to rename folder: %w", err)
	}

	return
}

// Rename an old directory to a new directory in a non-hierarchical bucket. If the new directory already
// exists and is non-empty, return ENOTEMPTY.
//
// LOCKS_EXCLUDED(fs.mu)
// LOCKS_EXCLUDED(oldParent)
// LOCKS_EXCLUDED(newParent)
func (fs *fileSystem) renameNonHierarchicalDir(
	ctx context.Context,
	oldParent inode.DirInode,
	oldName string,
	newParent inode.DirInode,
	newName string) error {

	// Set up a function that throws away the lookup count increment from
	// lookUpOrCreateChildInode (since the pending inodes are not sent back to
	// the kernel) and unlocks the pending inodes, but only once
	var pendingInodes []inode.DirInode
	defer fs.releaseInodes(&pendingInodes)

	oldDir, err := fs.getBucketDirInode(ctx, oldParent, oldName)
	if err != nil {
		return err
	}
	pendingInodes = append(pendingInodes, oldDir)

	if err = fs.ensureNoLocalFilesInDirectory(oldDir, oldName); err != nil {
		return err
	}

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

	newDir, err := fs.getBucketDirInode(ctx, newParent, newName)
	if err != nil {
		return err
	}
	pendingInodes = append(pendingInodes, newDir)

	if err = fs.checkDirNotEmpty(newDir, newName); err != nil {
		return err
	}

	// Move all the files from the old directory to the new directory, keeping both directories locked.
	for _, descendant := range descendants {
		nameDiff := strings.TrimPrefix(descendant.FullName.GcsObjectName(), oldDir.Name().GcsObjectName())
		if nameDiff == descendant.FullName.GcsObjectName() {
			return fmt.Errorf("unwanted descendant %q not from dir %q", descendant.FullName, oldDir.Name())
		}

		o := descendant.MinObject
		// Use copy-delete if atomic rename is disabled, or if the object is a directory or of unknown type.
		// Otherwise, for files with atomic rename enabled, use move.
		isDirOrUnknown := descendant.Type() == metadata.ExplicitDirType || descendant.Type() == metadata.UnknownType
		if !fs.enableAtomicRenameObject || isDirOrUnknown {
			if _, err = newDir.CloneToChildFile(ctx, nameDiff, o); err != nil {
				return fmt.Errorf("copy file %q: %w", o.Name, err)
			}
			if err = oldDir.DeleteChildFile(ctx, nameDiff, o.Generation, &o.MetaGeneration); err != nil {
				return fmt.Errorf("delete file %q: %w", o.Name, err)
			}
		} else {
			// For regular files, perform an in-place rename by constructing the new GCS object name.
			// Standard path.Join is avoided here because object names in GCS are distinct from
			// directory prefixes; the "/" character is *always* treated as a separate directory
			// element, not part of the object's base name. This manual approach correctly
			// handles those GCS naming edge cases (like objects with unsupported characters).
			newObject := newDir.Name().GcsObjectName() + nameDiff
			if _, err = oldDir.RenameFile(ctx, o, newObject); err != nil {
				return fmt.Errorf("renameFile: while renaming file: %w", err)
			}
		}

		if err = fs.invalidateChildFileCacheIfExist(oldDir, o.Name); err != nil {
			return fmt.Errorf("unlink: while invalidating cache for delete file: %w", err)
		}
	}

	fs.releaseInodes(&pendingInodes)

	// Delete the backing object of the old directory.
	fs.mu.Lock()
	_, isImplicitDir := fs.implicitDirInodes[oldDir.Name()]
	fs.mu.Unlock()
	oldParent.Lock()
	err = oldParent.DeleteChildDir(ctx, oldName, isImplicitDir, oldDir)
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
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}

	fs.mu.Lock()

	// Find the parent and file name.
	parent := fs.dirInodeOrDie(op.Parent)
	fileName := inode.NewFileName(parent.Name(), op.Name)

	// Get the inode for the given file.
	// Files must have an associated inode, which can be found in either:
	//  - localFileInodes: For files created locally.
	//  - generationBackedInodes: For files backed by an object.
	// We are not checking implicitDirInodes or folderInodes because
	// the unlink operation is only applicable to files.
	in, isLocalFile := fs.localFileInodes[fileName]
	if !isLocalFile {
		in = fs.generationBackedInodes[fileName]
	}

	fs.mu.Unlock()

	if in != nil {
		// Perform the unlink operation on the inode.
		in.Lock()
		in.Unlink()
		in.Unlock()
	}

	// If the inode represents a local file, we don't need to delete
	// the backing object on GCS, so return early.
	if isLocalFile {
		return
	}

	// Delete the backing object present on GCS.
	parent.Lock()
	defer parent.Unlock()

	err = parent.DeleteChildFile(
		ctx,
		op.Name,
		0,   // Latest generation
		nil) // No meta-generation precondition

	if err != nil {
		err = fmt.Errorf("DeleteChildFile: %w", err)
		return err
	}

	if err := fs.invalidateChildFileCacheIfExist(parent, fileName.GcsObjectName()); err != nil {
		return fmt.Errorf("unlink: while invalidating cache for delete file: %w", err)
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) (err error) {
	fs.mu.Lock()

	// Make sure the inode still exists and is a directory. If not, something has
	// screwed up because the VFS layer shouldn't have let us forget the inode
	// before opening it.
	in := fs.dirInodeOrDie(op.Inode)

	// Allocate a handle.
	handleID := fs.nextHandleID
	fs.nextHandleID++

	fs.handles[handleID] = handle.NewDirHandle(in, fs.implicitDirs)
	op.Handle = handleID

	fs.mu.Unlock()

	// Enables kernel list-cache in case of non-zero kernelListCacheTTL.
	if fs.kernelListCacheTTL > 0 {
		// Invalidates the kernel list-cache once the last cached response is out of
		// kernelListCacheTTL.
		op.KeepCache = !in.ShouldInvalidateKernelListCache(fs.kernelListCacheTTL)

		op.CacheDir = true
	}
	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) (err error) {
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}
	// Find the handle.
	fs.mu.Lock()
	dh := fs.handles[op.Handle].(*handle.DirHandle)
	in := fs.dirInodeOrDie(op.Inode)
	// Fetch local file entries beforehand and pass it to directory handle as
	// we need fs lock to fetch local file entries.
	localFileEntries := in.LocalFileEntries(fs.localFileInodes)
	fs.mu.Unlock()

	dh.Mu.Lock()
	defer dh.Mu.Unlock()
	// Serve the request.
	if err := dh.ReadDir(ctx, op, localFileEntries); err != nil {
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReadDirPlus(ctx context.Context, op *fuseops.ReadDirPlusOp) (err error) {
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}
	// Find the handle.
	fs.mu.Lock()
	dh := fs.handles[op.Handle].(*handle.DirHandle)
	in := fs.dirInodeOrDie(op.Inode)
	// Fetch local file entries beforehand for passing it to directory handle as
	// we need fs lock to fetch local file entries.
	localFileEntriesPlus := fs.localFileEntriesPlus(in.Name())
	// Unlock fs lock and fetch attributes for local file entries as it requires inode lock.
	fs.mu.Unlock()

	err = fs.lookupAndFetchAttributesForLocalFileEntriesPlus(in, localFileEntriesPlus)
	if err != nil {
		return err
	}

	dh.Mu.Lock()
	defer dh.Mu.Unlock()
	// Serve the request.
	var cores map[inode.Name]*inode.Core
	cores, err = dh.FetchEntryCores(ctx, op)
	if err != nil {
		return fmt.Errorf("FetchDirCores: %w", err)
	}
	// dh.mu lock is not required during iteration over cores, but was acquired earlier
	// for code readability and to use a common defer unlock pattern. Holding the
	// lock here has no performance overhead.
	var entriesPlus []fuseutil.DirentPlus
	for fullName, core := range cores {
		entry, err := fs.coreToDirentPlus(ctx, fullName, *core)
		if err != nil {
			return err
		}
		entriesPlus = append(entriesPlus, *entry)
	}

	if err := dh.ReadDirPlus(op, entriesPlus, localFileEntriesPlus); err != nil {
		return err
	}

	return nil
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) (err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Sanity check that this handle exists and is of the correct type.
	_ = fs.handles[op.Handle].(*handle.DirHandle)

	// Clear the entry from the map.
	delete(fs.handles, op.Handle)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) (err error) {
	// Bypass the kernel's page cache for file reads and writes
	if fs.newConfig.FileSystem.ODirect {
		op.UseDirectIO = true
	}

	fs.mu.Lock()

	// Find the inode.
	in := fs.fileInodeOrDie(op.Inode)
	// Follow lock ordering rules to get inode lock.
	// Inode lock is required to register fileHandle with the inode.
	fs.mu.Unlock()
	in.Lock()
	defer in.Unlock()

	// Get the fs lock again.
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Allocate a handle.
	handleID := fs.nextHandleID
	fs.nextHandleID++

	// Figure out the mode in which the file is being opened.
	openMode := util.FileOpenMode(op)
	fs.handles[handleID] = handle.NewFileHandle(in, fs.fileCacheHandler, fs.cacheFileForRangeRead, fs.metricHandle, openMode, fs.newConfig, fs.bufferedReadWorkerPool, fs.globalMaxReadBlocksSem)
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
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}
	// Save readOp in context for access in logs.
	ctx = context.WithValue(ctx, gcsx.ReadOp, op)

	// Find the handle and lock it.
	fs.mu.Lock()
	fh := fs.handles[op.Handle].(*handle.FileHandle)
	fs.mu.Unlock()

	fh.Inode().Lock()
	if fh.Inode().IsUsingBWH() {
		// Flush/Sync Pending streaming writes and issue read within same inode lock.
		if fh.Inode().Bucket().BucketType().Zonal {
			// With zonal buckets, we can read from unfinalized objects as well.
			// Hence, there is no need to finalize the object from here for zonal buckets.
			// Hence, if FinalizeFileForRapid is set, then we will call syncFile otherwise
			// we can call flushFile (as it will not finalize when FinalizeFileForRapid is false) itself.
			if fs.newConfig.Write.FinalizeFileForRapid {
				err = fs.syncFile(ctx, fh.Inode())
			} else {
				err = fs.flushFile(ctx, fh.Inode())
			}
		} else {
			err = fs.flushFile(ctx, fh.Inode())
		}

		if err != nil {
			fh.Inode().Unlock()
			return err
		}
	}
	// Serve the read.

	if fs.newConfig.EnableNewReader {
		op.Dst, op.BytesRead, err = fh.ReadWithReadManager(ctx, op.Dst, op.Offset, fs.sequentialReadSizeMb)
	} else {
		op.Dst, op.BytesRead, err = fh.Read(ctx, op.Dst, op.Offset, fs.sequentialReadSizeMb)
	}

	// A FileClobberedError indicates the underlying GCS object has changed,
	// making the kernel's dentry for this file stale. We use the notifier to
	// invalidate this entry, providing feedback to the kernel about the dynamic
	// content change and ensuring subsequent lookups fetch the correct metadata.
	if fs.newConfig.FileSystem.ExperimentalEnableDentryCache {
		var clobberedErr *gcsfuse_errors.FileClobberedError
		if err != nil && errors.As(err, &clobberedErr) {
			if invalidateErr := fs.invalidateCachedEntry(op.Inode); invalidateErr != nil {
				err = fmt.Errorf("%w; additionally failed to invalidate entry: %w", err, invalidateErr)
			}
		}
	}
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
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}

	// Find the inode( and file handle in case of appends).
	fs.mu.Lock()
	fh := fs.handles[op.Handle].(*handle.FileHandle)
	in := fs.fileInodeOrDie(op.Inode)
	fs.mu.Unlock()

	var gcsSynced bool
	in.Lock()
	defer in.Unlock()
	if err = fs.initBufferedWriteHandlerAndSyncFileIfEligible(ctx, in, fh.OpenMode()); err != nil {
		// A FileClobberedError on write indicates the file was modified in GCS,
		// making the kernel's dentry stale. By invalidating the cache
		// entry, we ensure the filesystem corrects the inconsistency caused by this
		// dynamic content change.
		if fs.newConfig.FileSystem.ExperimentalEnableDentryCache {
			var clobberedErr *gcsfuse_errors.FileClobberedError
			if errors.As(err, &clobberedErr) {
				if invalidateErr := fs.invalidateCachedEntry(op.Inode); invalidateErr != nil {
					err = fmt.Errorf("%w; additionally failed to invalidate entry: %w", err, invalidateErr)
				}
			}
		}
		return err
	}
	if fs.newConfig.Write.EnableRapidAppends {
		// Serve the request via the file handle.
		gcsSynced, err = fh.Write(ctx, op.Data, op.Offset)
	} else {
		// Serve the request.
		gcsSynced, err = in.Write(ctx, op.Data, op.Offset, util.Write)
	}
	if err != nil {
		return
	}
	// Sync the inode if finalize during write is successful
	// even if the write operation later resulted in error.
	if gcsSynced {
		fs.promoteToGenerationBacked(in)
	}
	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) (err error) {
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}
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
	if fs.newConfig.FileSystem.IgnoreInterrupts {
		// When ignore interrupts config is set, we are creating a new context not
		// cancellable by parent context.
		ctx = context.Background()
	}
	// Find the inode.
	fs.mu.Lock()
	in := fs.fileInodeOrDie(op.Inode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Sync it.
	if err := fs.flushFile(ctx, in); err != nil {
		return err
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) (err error) {
	fs.mu.Lock()

	fileHandle := fs.handles[op.Handle].(*handle.FileHandle)
	// Update the map. We are okay updating the map before destroy is called
	// since destroy is doing only internal cleanup.
	delete(fs.handles, op.Handle)
	fs.mu.Unlock()

	// Destroy the handle.
	fileHandle.Lock()
	defer fileHandle.Unlock()
	fileHandle.Destroy()

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

func (fs *fileSystem) SyncFS(
	ctx context.Context,
	op *fuseops.SyncFSOp) error {
	return syscall.ENOSYS
}
