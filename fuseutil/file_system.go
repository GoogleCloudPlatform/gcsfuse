// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fuseutil

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
		ctx context.Contexst,
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
	// XXX: Fields
}

type NothingImplementedFileSystem struct {
}

var _ FileSystem = NothingImplementedFileSystem{}
