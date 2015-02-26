// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fuseutil

import (
	"time"

	"golang.org/x/net/context"
)

// An interface that must be implemented by file systems to be mounted with
// FUSE. Comments reflect requirements on the file system imposed by the
// kernel. See also the comments on request and response structs.
//
// Not all methods need to have interesting implementations. Embed a field of
// type NothingImplementedFileSystem to inherit defaults that return ENOSYS to
// the kernel.
type FileSystem interface {
	// Look up a child by name within a parent directory. The kernel calls this
	// when resolving user paths to dentry structs, which are then cached.
	//
	// The returned inode ID must be valid until a later call to Forget.
	Lookup(
		ctx context.Context,
		req *LookupRequest) (*LookupResponse, error)

	// Forget an inode ID previously issued (e.g. by Lookup). The kernel calls
	// this when removing an inode from its internal caches.
	//
	// The node ID will not be used in further calls to the file system (unless
	// it is reissued by the file system).
	Forget(
		ctx context.Context,
		req *ForgetRequest) (*ForgetResponse, error)
}

// A 64-bit number used to uniquely identify a file or directory in the file
// system.
//
// This corresponds to struct inode::i_no in the VFS layer.
// (Cf. http://goo.gl/tvYyQt)
type InodeID uint64

// A generation number for an inode. Irrelevant for file systems that won't be
// exported over NFS. For those that will and that reuse inode IDs when they
// become free, the generation number must change when an ID is reused.
//
// This corresponds to struct inode::i_generation in the VFS layer.
// (Cf. http://goo.gl/tvYyQt)
//
// Some related reading:
//
//     http://fuse.sourceforge.net/doxygen/structfuse__entry__param.html
//     http://stackoverflow.com/q/11071996/1505451
//     http://goo.gl/CqvwyX
//     http://julipedia.meroh.net/2005/09/nfs-file-handles.html
//     http://goo.gl/wvo3MB
//
type GenerationNumber uint64

// Attributes for a file or directory inode. Corresponds to struct inode (cf.
// http://goo.gl/tvYyQt).
type InodeAttributes struct {
	// The size of the file in bytes.
	Size uint64
}

// A request to look up a child by name within a parent directory. This is sent
// by the kernel when resolving user paths to dentry structs, which are then
// cached.
type LookupRequest struct {
	// The ID of the directory inode to which the child belongs.
	Parent InodeID

	// The name of the child of interest, relative to the parent. For example, in
	// this directory structure:
	//
	//     foo/
	//         bar/
	//             baz
	//
	// the file system may receive a request to look up the child named "bar" for
	// the parent foo/.
	Name string
}

// XXX: Comments
type LookupResponse struct {
	// The ID of the child inode. This must remain valid until a later call to
	// Forget.
	InodeID InodeID

	// A generation number for this incarnation of the inode with the given ID.
	// See comments on type GenerationNumber for more.
	GenerationNumber GenerationNumber

	// Attributes for the child inode.
	InodeAttributes InodeAttributes

	// The time until which the kernel may maintain an entry for this name to
	// inode mapping in its dentry cache. After this time, it will revalidate the
	// dentry.
	EntryExpiration time.Time

	// The FUSE VFS layer in the kernel maintains a cache of file attributes,
	// used whenever up to date information about size, mode, etc. is needed.
	//
	// For example, this is the abridged call chain for fstat(2):
	//
	//  *  (https://github.com/torvalds/linux/blob/b7a6ec52dd4eced4a9bcda9ca85b3c8af84d3c90/fs/stat.c#L203) fstat calls vfs_fstat.
	//  *  (https://github.com/torvalds/linux/blob/b7a6ec52dd4eced4a9bcda9ca85b3c8af84d3c90/fs/stat.c#L77) vfs_fstat eventuall calls vfs_getattr_nosec.
	//  *  (https://github.com/torvalds/linux/blob/b7a6ec52dd4eced4a9bcda9ca85b3c8af84d3c90/fs/stat.c#L40) vfs_getattr_nosec calls i_op->getattr (fuse_getattr).
	//  *  (https://github.com/torvalds/linux/blob/e36cb0b89ce20b4f8786a57e8a6bc8476f577650/fs/fuse/dir.c#L1726) fuse_getattr calls fuse_update_attributes.
	//  *  (https://github.com/torvalds/linux/blob/e36cb0b89ce20b4f8786a57e8a6bc8476f577650/fs/fuse/dir.c#L909) fuse_update_attributes uses the values in the struct inode if allowed, otherwise calling out to the user-space code.
	//
	// In addition to obvious cases like fstat, this is also used in more subtle cases like updating size information before seeking (https://github.com/torvalds/linux/blob/e36cb0b89ce20b4f8786a57e8a6bc8476f577650/fs/fuse/file.c#L2258) or reading (https://github.com/torvalds/linux/blob/e36cb0b89ce20b4f8786a57e8a6bc8476f577650/fs/fuse/file.c#L891).
	//
	// This field controls when the attributes returned in this response and
	// stashed in the struct inode should be re-queried.
	//
	// More reading:
	//     http://stackoverflow.com/q/21540315/1505451
	AttrExpiration time.Time
}

type NothingImplementedFileSystem struct {
}

var _ FileSystem = NothingImplementedFileSystem{}
