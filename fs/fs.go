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
	"log"
	"math"
	"os"
	"reflect"
	"time"

	"github.com/googlecloudplatform/gcsfuse/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/gcsproxy"
	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
)

type ServerConfig struct {
	// A clock used for modification times and cache expiration.
	Clock timeutil.Clock

	// The bucket that the file system is to export.
	Bucket gcs.Bucket

	// The temporary directory to use for local caching, or the empty string to
	// use the system default.
	TempDir string

	// A desired limit on the number of open files used for storing temporary
	// object contents. May not be obeyed if there is a large number of dirtied
	// files that have not been flushed or closed.
	//
	// Most users will want to use ChooseTempDirLimitNumFiles to choose this.
	TempDirLimitNumFiles int

	// A desired limit on temporary space usage, in bytes. May not be obeyed if
	// there is a large volume of dirtied files that have not been flushed or
	// closed.
	TempDirLimitBytes int64

	// If set to a non-zero value N, the file system will read objects from GCS a
	// chunk at a time with a maximum read size of N, caching each chunk
	// independently. The part about separate caching does not apply to dirty
	// files, for which the entire contents will be in the temporary directory
	// regardless of this setting.
	GCSChunkSize uint64

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

	// Files backed by on object of length at least AppendThreshold that have
	// only been appended to (i.e. none of the object's contents have been
	// dirtied) will be written out by "appending" to the object in GCS with this
	// process:
	//
	// 1. Write out a temporary object containing the appended contents whose
	//    name begins with TmpObjectPrefix.
	//
	// 2. Compose the original object and the temporary object on top of the
	//    original object.
	//
	// 3. Delete the temporary object.
	//
	// Note that if the process fails or is interrupted the temporary object will
	// not be cleaned up, so the user must ensure that TmpObjectPrefix is
	// periodically garbage collected.
	AppendThreshold int64
	TmpObjectPrefix string
}

// Create a fuse file system server according to the supplied configuration.
func NewServer(cfg *ServerConfig) (server fuse.Server, err error) {
	// Check permissions bits.
	if cfg.FilePerms&^os.ModePerm != 0 {
		err = fmt.Errorf("Illegal file perms: %v", cfg.FilePerms)
		return
	}

	if cfg.DirPerms&^os.ModePerm != 0 {
		err = fmt.Errorf("Illegal dir perms: %v", cfg.FilePerms)
		return
	}

	// Disable chunking if set to zero.
	gcsChunkSize := cfg.GCSChunkSize
	if gcsChunkSize == 0 {
		gcsChunkSize = math.MaxUint64
	}

	// Create the file leaser.
	leaser := lease.NewFileLeaser(
		cfg.TempDir,
		cfg.TempDirLimitNumFiles,
		cfg.TempDirLimitBytes)

	// Create the object syncer.
	// Check TmpObjectPrefix.
	if cfg.TmpObjectPrefix == "" {
		err = errors.New("You must set TmpObjectPrefix.")
		return
	}

	objectSyncer := gcsproxy.NewObjectSyncer(
		cfg.AppendThreshold,
		cfg.TmpObjectPrefix,
		cfg.Bucket)

	// Set up the basic struct.
	fs := &fileSystem{
		clock:                  cfg.Clock,
		bucket:                 cfg.Bucket,
		leaser:                 leaser,
		objectSyncer:           objectSyncer,
		gcsChunkSize:           gcsChunkSize,
		implicitDirs:           cfg.ImplicitDirectories,
		dirTypeCacheTTL:        cfg.DirTypeCacheTTL,
		uid:                    cfg.Uid,
		gid:                    cfg.Gid,
		fileMode:               cfg.FilePerms,
		dirMode:                cfg.DirPerms | os.ModeDir,
		inodes:                 make(map[fuseops.InodeID]inode.Inode),
		nextInodeID:            fuseops.RootInodeID + 1,
		generationBackedInodes: make(map[string]GenerationBackedInode),
		implicitDirInodes:      make(map[string]inode.DirInode),
		handles:                make(map[fuseops.HandleID]interface{}),
	}

	// Set up the root inode.
	root := inode.NewDirInode(
		fuseops.RootInodeID,
		"", // name
		fuseops.InodeAttributes{
			Uid:  fs.uid,
			Gid:  fs.gid,
			Mode: fs.dirMode,
		},
		fs.implicitDirs,
		fs.dirTypeCacheTTL,
		cfg.Bucket,
		fs.clock)

	root.Lock()
	root.IncrementLookupCount()
	fs.inodes[fuseops.RootInodeID] = root
	fs.implicitDirInodes[root.Name()] = root
	root.Unlock()

	// Set up invariant checking.
	fs.mu = syncutil.NewInvariantMutex(fs.checkInvariants)

	// Periodically garbage collect temporary objects.
	var gcCtx context.Context
	gcCtx, fs.stopGarbageCollecting = context.WithCancel(context.Background())
	go garbageCollect(gcCtx, cfg.TmpObjectPrefix, fs.bucket)

	server = fuseutil.NewFileSystemServer(fs)
	return
}

// Choose a reasonable value for ServerConfig.TempDirLimitNumFiles based on
// process limits.
func ChooseTempDirLimitNumFiles() (limit int) {
	// Ask what the process's limit on open files is. Use a default on error.
	var rlimit unix.Rlimit
	err := unix.Getrlimit(unix.RLIMIT_NOFILE, &rlimit)
	if err != nil {
		const defaultLimit = 512
		log.Println(
			"Warning: failed to query RLIMIT_NOFILE. Using default "+
				"file count limit of %d",
			defaultLimit)

		limit = defaultLimit
		return
	}

	// Heuristic: Use about 75% of the limit.
	limit64 := rlimit.Cur/2 + rlimit.Cur/4

	// But not too large.
	const reasonableLimit = 1 << 15
	if limit64 > reasonableLimit {
		limit64 = reasonableLimit
	}

	limit = int(limit64)

	return
}

////////////////////////////////////////////////////////////////////////
// fileSystem type
////////////////////////////////////////////////////////////////////////

// LOCK ORDERING
//
// Let FS be the file system lock. Define a strict partial order < as follows:
//
//  1. For any inode lock I, I < FS.
//  2. For any directory handle lock DH and inode lock I, DH < I.
//
// We follow the rule "acquire A then B only if A < B".
//
// In other words:
//
//  *  Don't hold multiple directory handle locks at the same time.
//  *  Don't hold multiple inode locks at the same time.
//  *  Don't acquire inode locks before directory handle locks.
//  *  Don't acquire file system locks before either.
//
// The intuition is that we hold inode and directory handle locks for
// long-running operations, and we don't want to block the entire file system
// on those.
//
// See http://goo.gl/rDxxlG for more discussion, including an informal proof
// that a strict partial order is sufficient.

type fileSystem struct {
	fuseutil.NotImplementedFileSystem

	/////////////////////////
	// Dependencies
	/////////////////////////

	clock        timeutil.Clock
	bucket       gcs.Bucket
	objectSyncer gcsproxy.ObjectSyncer
	leaser       lease.FileLeaser

	/////////////////////////
	// Constant data
	/////////////////////////

	gcsChunkSize    uint64
	implicitDirs    bool
	dirTypeCacheTTL time.Duration

	// The user and group owning everything in the file system.
	uid uint32
	gid uint32

	// Mode bits for all inodes.
	fileMode os.FileMode
	dirMode  os.FileMode

	// A function that shuts down the garbage collector.
	stopGarbageCollecting func()

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A lock protecting the state of the file system struct itself (distinct
	// from per-inode locks). Make sure to see the notes on lock ordering above.
	mu syncutil.InvariantMutex

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
	// INVARIANT: For all v, if IsDirName(v.Name()) then v is inode.DirInode
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
	generationBackedInodes map[string]GenerationBackedInode

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
	implicitDirInodes map[string]inode.DirInode

	// The collection of live handles, keyed by handle ID.
	//
	// INVARIANT: All values are of type *dirHandle
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

// A common interface for inodes backed by particular object generations.
// Implemented by FileInode and SymlinkInode.
type GenerationBackedInode interface {
	inode.Inode
	SourceGeneration() int64
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

	// INVARIANT: For all v, if IsDirName(v.Name()) then v is inode.DirInode
	for _, in := range fs.inodes {
		if inode.IsDirName(in.Name()) {
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
				"Mismatch for ID %v: %p %p",
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
				"Mismatch for ID %v: %p %p",
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
					"implicitDirInodes mismatch: %q %p %p",
					in.Name(),
					fs.implicitDirInodes[in.Name()],
					in))
			}
		}
	}

	//////////////////////////////////
	// handles
	//////////////////////////////////

	// INVARIANT: All values are of type *dirHandle
	for _, h := range fs.handles {
		_ = h.(*dirHandle)
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
func (fs *fileSystem) mintInode(name string, o *gcs.Object) (in inode.Inode) {
	// Choose an ID.
	id := fs.nextInodeID
	fs.nextInodeID++

	// Create the inode.
	switch {
	// Explicit directories
	case o != nil && inode.IsDirName(o.Name):
		in = inode.NewExplicitDirInode(
			id,
			o,
			fuseops.InodeAttributes{
				Uid:  fs.uid,
				Gid:  fs.gid,
				Mode: fs.dirMode,
			},
			fs.implicitDirs,
			fs.dirTypeCacheTTL,
			fs.bucket,
			fs.clock)

	// Implicit directories
	case inode.IsDirName(name):
		in = inode.NewDirInode(
			id,
			name,
			fuseops.InodeAttributes{
				Uid:  fs.uid,
				Gid:  fs.gid,
				Mode: fs.dirMode,
			},
			fs.implicitDirs,
			fs.dirTypeCacheTTL,
			fs.bucket,
			fs.clock)

	case inode.IsSymlink(o):
		in = inode.NewSymlinkInode(
			id,
			o,
			fuseops.InodeAttributes{
				Uid:  fs.uid,
				Gid:  fs.gid,
				Mode: fs.fileMode | os.ModeSymlink,
			})

	default:
		in = inode.NewFileInode(
			id,
			o,
			fuseops.InodeAttributes{
				Uid:  fs.uid,
				Gid:  fs.gid,
				Mode: fs.fileMode,
			},
			fs.gcsChunkSize,
			fs.bucket,
			fs.leaser,
			fs.objectSyncer,
			fs.clock)
	}

	// Place it in our map of IDs to inodes.
	fs.inodes[in.ID()] = in

	return
}

// Attempt to find an inode for the given name, backed by the supplied object
// record (or nil for implicit directories). Create one if one has never yet
// existed and, if the record is non-nil, the record is newer than any inode
// we've yet recorded.
//
// If the record is stale (i.e. some newer inode exists), return nil. In this
// case, the caller may obtain a fresh record and try again. Otherwise,
// increment the inode's lookup count and return it locked.
//
// UNLOCK_FUNCTION(fs.mu)
// LOCK_FUNCTION(in)
func (fs *fileSystem) lookUpOrCreateInodeIfNotStale(
	name string,
	o *gcs.Object) (in inode.Inode) {
	// Sanity check.
	if o != nil && name != o.Name {
		panic(fmt.Sprintf("Name mismatch: %q vs. %q", name, o.Name))
	}

	// Ensure that no matter which inode we return, we increase its lookup count
	// on the way out.
	defer func() {
		if in != nil {
			in.IncrementLookupCount()
		}
	}()

	// Handle implicit directories.
	if o == nil {
		if !inode.IsDirName(name) {
			panic(fmt.Sprintf("Unexpected name for an implicit directory: %q", name))
		}

		// If we don't have an entry, create one.
		var ok bool
		in, ok = fs.implicitDirInodes[name]
		if !ok {
			in = fs.mintInode(name, nil)
			fs.implicitDirInodes[in.Name()] = in.(inode.DirInode)
		}

		fs.mu.Unlock()
		in.Lock()

		return
	}

	// Retry loop for the stale index entry case below. On entry, we hold fs.mu
	// but no inode lock.
	for {
		// Look at the current index entry.
		existingInode, ok := fs.generationBackedInodes[o.Name]

		// If we have no existing record for this name, mint an inode and return it.
		if !ok {
			in = fs.mintInode(o.Name, o)
			fs.generationBackedInodes[in.Name()] = in.(GenerationBackedInode)

			fs.mu.Unlock()
			in.Lock()

			return
		}

		// Otherwise we need to grab the inode lock to find out if this is our
		// inode, our record is stale, or the inode is stale. are stale compared to
		// it. We must exclude concurrent actions on the inode to get a definitive
		// answer.
		//
		// Drop the file system lock and acquire the inode lock.
		fs.mu.Unlock()
		existingInode.Lock()

		// Have we found the correct inode?
		if o.Generation == existingInode.SourceGeneration() {
			in = existingInode
			return
		}

		// Are we stale?
		if o.Generation < existingInode.SourceGeneration() {
			existingInode.Unlock()
			return
		}

		// We've observed that the record is newer than the existing inode, while
		// holding the inode lock, excluding concurrent actions by the inode (in
		// particular concurrent calls to Sync, which changes generation numbers).
		// This means we've proven that the record cannot have been caused by the
		// inode's actions, and therefore it is not the inode we want.
		//
		// Re-acquire the file system lock. If the index entry still points at
		// existingInode, we have proven we can replace it with an entry for a a
		// newly-minted inode.
		fs.mu.Lock()
		if fs.generationBackedInodes[o.Name] == existingInode {
			in = fs.mintInode(o.Name, o)
			fs.generationBackedInodes[in.Name()] = in.(GenerationBackedInode)

			fs.mu.Unlock()
			existingInode.Unlock()
			in.Lock()

			return
		}

		// The index entry has been changed in the meantime, so there may be a new
		// inode that we have to contend with. Go around and try again.
		existingInode.Unlock()
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
	getLookupResult := func() (r inode.LookUpResult, err error) {
		parent.Lock()
		defer parent.Unlock()

		r, err = parent.LookUpChild(ctx, childName)
		if err != nil {
			err = fmt.Errorf("LookUpChild: %v", err)
			return
		}

		return
	}

	// Run a retry loop around lookUpOrCreateInodeIfNotStale.
	const maxTries = 3
	for n := 0; n < maxTries; n++ {
		// Create a record.
		var result inode.LookUpResult
		result, err = getLookupResult()

		if err != nil {
			return
		}

		if !result.Exists() {
			err = fuse.ENOENT
			return
		}

		// Attempt to create the inode. Return if successful.
		fs.mu.Lock()
		child = fs.lookUpOrCreateInodeIfNotStale(result.FullName, result.Object)
		if child != nil {
			return
		}
	}

	err = fmt.Errorf("Did not converge after %v tries", maxTries)
	return
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
		err = fmt.Errorf("FileInode.Sync: %v", err)
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
// UNLOCK_FUNCTION(fs.mu)
// UNLOCK_FUNCTION(in)
func (fs *fileSystem) unlockAndDecrementLookupCount(
	in inode.Inode,
	N uint64) {
	name := in.Name()

	// Decrement the lookup count.
	shouldDestroy := in.DecrementLookupCount(N)

	// Update file system state, orphaning the inode if we're going to destroy it
	// below.
	if shouldDestroy {
		delete(fs.inodes, in.ID())

		// Update indexes if necessary.
		if fs.generationBackedInodes[name] == in {
			delete(fs.generationBackedInodes, name)
		}

		if fs.implicitDirInodes[name] == in {
			delete(fs.implicitDirInodes, name)
		}
	}

	// We are done with the file system.
	fs.mu.Unlock()

	// Now we can destroy the inode if necessary.
	if shouldDestroy {
		destroyErr := in.Destroy()
		if destroyErr != nil {
			log.Printf("Error destroying inode %q: %v", name, destroyErr)
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
//     func (fs *fileSystem) doFoo() (err error) {
//       in, err := fs.lookUpOrCreateInodeIfNotStale(...)
//       if err != nil {
//         return
//       }
//
//       defer fs.unlockAndMaybeDisposeOfInode(in, &err)
//
//       ...
//     }
//
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

	// Otherwise, go through the decrement helper, which requires the file system
	// lock.
	fs.mu.Lock()
	fs.unlockAndDecrementLookupCount(in, 1)
}

////////////////////////////////////////////////////////////////////////
// fuse.FileSystem methods
////////////////////////////////////////////////////////////////////////

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) Init(
	op *fuseops.InitOp) (err error) {
	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) LookUpInode(
	op *fuseops.LookUpInodeOp) (err error) {
	// Find the parent directory in question.
	fs.mu.Lock()
	parent := fs.inodes[op.Parent].(inode.DirInode)
	fs.mu.Unlock()

	// Find or create the child inode.
	child, err := fs.lookUpOrCreateChildInode(op.Context(), parent, op.Name)
	if err != nil {
		return
	}

	defer fs.unlockAndMaybeDisposeOfInode(child, &err)

	// Fill out the response.
	op.Entry.Child = child.ID()
	if op.Entry.Attributes, err = child.Attributes(op.Context()); err != nil {
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) GetInodeAttributes(
	op *fuseops.GetInodeAttributesOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode]
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Grab its attributes.
	op.Attributes, err = in.Attributes(op.Context())
	if err != nil {
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) SetInodeAttributes(
	op *fuseops.SetInodeAttributesOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode]
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// The only thing we support changing is size, and then only for directories.
	if op.Mode != nil || op.Atime != nil || op.Mtime != nil {
		err = fuse.ENOSYS
		return
	}

	file, ok := in.(*inode.FileInode)
	if !ok {
		err = fuse.ENOSYS
		return
	}

	// Set the size, if specified.
	if op.Size != nil {
		if err = file.Truncate(op.Context(), int64(*op.Size)); err != nil {
			err = fmt.Errorf("Truncate: %v", err)
			return
		}
	}

	// Fill in the response.
	op.Attributes, err = in.Attributes(op.Context())
	if err != nil {
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ForgetInode(
	op *fuseops.ForgetInodeOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode]
	fs.mu.Unlock()

	// Acquire both locks in the correct order.
	in.Lock()
	fs.mu.Lock()

	// Decrement and unlock.
	fs.unlockAndDecrementLookupCount(in, op.N)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) MkDir(
	op *fuseops.MkDirOp) (err error) {
	// Find the parent.
	fs.mu.Lock()
	parent := fs.inodes[op.Parent].(inode.DirInode)
	fs.mu.Unlock()

	// Create an empty backing object for the child, failing if it already
	// exists.
	parent.Lock()
	o, err := parent.CreateChildDir(op.Context(), op.Name)
	parent.Unlock()

	// Special case: *gcs.PreconditionError means the name already exists.
	if _, ok := err.(*gcs.PreconditionError); ok {
		err = fuse.EEXIST
		return
	}

	// Propagate other errors.
	if err != nil {
		err = fmt.Errorf("CreateChildDir: %v", err)
		return
	}

	// Attempt to create a child inode using the object we created. If we fail to
	// do so, it means someone beat us to the punch with a newer generation
	// (unlikely, so we're probably okay with failing here).
	fs.mu.Lock()
	child := fs.lookUpOrCreateInodeIfNotStale(o.Name, o)
	if child == nil {
		err = fmt.Errorf("Newly-created record is already stale")
		return
	}

	defer fs.unlockAndMaybeDisposeOfInode(child, &err)

	// Fill out the response.
	op.Entry.Child = child.ID()
	op.Entry.Attributes, err = child.Attributes(op.Context())

	if err != nil {
		err = fmt.Errorf("Attributes: %v", err)
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) CreateFile(
	op *fuseops.CreateFileOp) (err error) {
	// Find the parent.
	fs.mu.Lock()
	parent := fs.inodes[op.Parent].(inode.DirInode)
	fs.mu.Unlock()

	// Create an empty backing object for the child, failing if it already
	// exists.
	parent.Lock()
	o, err := parent.CreateChildFile(op.Context(), op.Name)
	parent.Unlock()

	// Special case: *gcs.PreconditionError means the name already exists.
	if _, ok := err.(*gcs.PreconditionError); ok {
		err = fuse.EEXIST
		return
	}

	// Propagate other errors.
	if err != nil {
		err = fmt.Errorf("CreateChildFile: %v", err)
		return
	}

	// Attempt to create a child inode using the object we created. If we fail to
	// do so, it means someone beat us to the punch with a newer generation
	// (unlikely, so we're probably okay with failing here).
	fs.mu.Lock()
	child := fs.lookUpOrCreateInodeIfNotStale(o.Name, o)
	if child == nil {
		err = fmt.Errorf("Newly-created record is already stale")
		return
	}

	defer fs.unlockAndMaybeDisposeOfInode(child, &err)

	// Fill out the response.
	op.Entry.Child = child.ID()
	op.Entry.Attributes, err = child.Attributes(op.Context())

	if err != nil {
		err = fmt.Errorf("Attributes: %v", err)
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) CreateSymlink(
	op *fuseops.CreateSymlinkOp) (err error) {
	// Find the parent.
	fs.mu.Lock()
	parent := fs.inodes[op.Parent].(inode.DirInode)
	fs.mu.Unlock()

	// Create the object in GCS, failing if it already exists.
	parent.Lock()
	o, err := parent.CreateChildSymlink(op.Context(), op.Name, op.Target)
	parent.Unlock()

	// Special case: *gcs.PreconditionError means the name already exists.
	if _, ok := err.(*gcs.PreconditionError); ok {
		err = fuse.EEXIST
		return
	}

	// Propagate other errors.
	if err != nil {
		err = fmt.Errorf("CreateChildSymlink: %v", err)
		return
	}

	// Attempt to create a child inode using the object we created. If we fail to
	// do so, it means someone beat us to the punch with a newer generation
	// (unlikely, so we're probably okay with failing here).
	fs.mu.Lock()
	child := fs.lookUpOrCreateInodeIfNotStale(o.Name, o)
	if child == nil {
		err = fmt.Errorf("Newly-created record is already stale")
		return
	}

	defer fs.unlockAndMaybeDisposeOfInode(child, &err)

	// Fill out the response.
	op.Entry.Child = child.ID()
	op.Entry.Attributes, err = child.Attributes(op.Context())

	if err != nil {
		err = fmt.Errorf("Attributes: %v", err)
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) RmDir(
	op *fuseops.RmDirOp) (err error) {
	// Find the parent. We assume that it exists because otherwise the kernel has
	// done something mildly concerning.
	fs.mu.Lock()
	parent := fs.inodes[op.Parent].(inode.DirInode)
	fs.mu.Unlock()

	// Find or create the child inode.
	child, err := fs.lookUpOrCreateChildInode(op.Context(), parent, op.Name)
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
			fs.mu.Lock()
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
		entries, tok, err = childDir.ReadEntries(op.Context(), tok)
		if err != nil {
			err = fmt.Errorf("ReadEntries: %v", err)
			return
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
	err = parent.DeleteChildDir(op.Context(), op.Name)
	parent.Unlock()

	if err != nil {
		err = fmt.Errorf("DeleteChildDir: %v", err)
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) Unlink(
	op *fuseops.UnlinkOp) (err error) {
	// Find the parent.
	fs.mu.Lock()
	parent := fs.inodes[op.Parent].(inode.DirInode)
	fs.mu.Unlock()

	parent.Lock()
	defer parent.Unlock()

	// Delete the backing object.
	err = parent.DeleteChildFile(op.Context(), op.Name)
	if err != nil {
		err = fmt.Errorf("DeleteChildFile: %v", err)
		return
	}

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) OpenDir(
	op *fuseops.OpenDirOp) (err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Make sure the inode still exists and is a directory. If not, something has
	// screwed up because the VFS layer shouldn't have let us forget the inode
	// before opening it.
	in := fs.inodes[op.Inode].(inode.DirInode)

	// Allocate a handle.
	handleID := fs.nextHandleID
	fs.nextHandleID++

	fs.handles[handleID] = newDirHandle(in, fs.implicitDirs)
	op.Handle = handleID

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReadDir(
	op *fuseops.ReadDirOp) (err error) {
	// Find the handle.
	fs.mu.Lock()
	dh := fs.handles[op.Handle].(*dirHandle)
	fs.mu.Unlock()

	dh.Mu.Lock()
	defer dh.Mu.Unlock()

	// Serve the request.
	err = dh.ReadDir(op)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReleaseDirHandle(
	op *fuseops.ReleaseDirHandleOp) (err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Sanity check that this handle exists and is of the correct type.
	_ = fs.handles[op.Handle].(*dirHandle)

	// Clear the entry from the map.
	delete(fs.handles, op.Handle)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) OpenFile(
	op *fuseops.OpenFileOp) (err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Sanity check that this inode exists and is of the correct type.
	_ = fs.inodes[op.Inode].(*inode.FileInode)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReadFile(
	op *fuseops.ReadFileOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode].(*inode.FileInode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Serve the request.
	op.Data, err = in.Read(op.Context(), op.Offset, op.Size)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) ReadSymlink(
	op *fuseops.ReadSymlinkOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode].(*inode.SymlinkInode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Serve the request.
	op.Target = in.Target()

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) WriteFile(
	op *fuseops.WriteFileOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode].(*inode.FileInode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Serve the request.
	err = in.Write(op.Context(), op.Data, op.Offset)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) SyncFile(
	op *fuseops.SyncFileOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode].(*inode.FileInode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Sync it.
	err = fs.syncFile(op.Context(), in)

	return
}

// LOCKS_EXCLUDED(fs.mu)
func (fs *fileSystem) FlushFile(
	op *fuseops.FlushFileOp) (err error) {
	// Find the inode.
	fs.mu.Lock()
	in := fs.inodes[op.Inode].(*inode.FileInode)
	fs.mu.Unlock()

	in.Lock()
	defer in.Unlock()

	// Sync it.
	err = fs.syncFile(op.Context(), in)

	return
}
