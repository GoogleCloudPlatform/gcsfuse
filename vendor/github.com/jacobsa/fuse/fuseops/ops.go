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

package fuseops

import (
	"os"
	"time"

	"github.com/jacobsa/fuse/internal/fusekernel"
)

////////////////////////////////////////////////////////////////////////
// File system
////////////////////////////////////////////////////////////////////////

// OpContext contains extra context that may be needed by some file systems.
// See https://libfuse.github.io/doxygen/structfuse__context.html as a reference.
type OpContext struct {
	// FuseID is the Unique identifier for each operation from the kernel.
	FuseID uint64

	// PID of the process that is invoking the operation.
	// Not filled in case of a writepage operation.
	Pid uint32

	// UID of the process that is invoking the operation.
	// Not filled in case of a writepage operation.
	Uid uint32
}

// Return statistics about the file system's capacity and available resources.
//
// Called by statfs(2) and friends:
//
//   - (https://tinyurl.com/234ppacj) sys_statfs called user_statfs, which calls
//     vfs_statfs, which calls statfs_by_dentry.
//
//   - (https://tinyurl.com/u6keadjz) statfs_by_dentry calls the superblock
//     operation statfs, which in our case points at
//     fuse_statfs (https://tinyurl.com/mr45wd28)
//
//   - (https://tinyurl.com/3wt3dw3c) fuse_statfs sends a statfs op, then uses
//     convert_fuse_statfs to convert the response in a straightforward manner.
//
// This op is particularly important on OS X: if you don't implement it, the
// file system will not successfully mount. If you don't model a sane amount of
// free space, the Finder will refuse to copy files into the file system.
type StatFSOp struct {
	// The size of the file system's blocks. This may be used, in combination
	// with the block counts below,  by callers of statfs(2) to infer the file
	// system's capacity and space availability.
	//
	// On Linux this is surfaced as statfs::f_frsize, matching the posix standard
	// (https://tinyurl.com/2juj6ah6), which says that f_blocks and friends are
	// in units of f_frsize. On OS X this is surfaced as statfs::f_bsize, which
	// plays the same roll.
	//
	// It appears as though the original intent of statvfs::f_frsize in the posix
	// standard was to support a smaller addressable unit than statvfs::f_bsize
	// (cf. The Linux Programming Interface by Michael Kerrisk,
	// https://tinyurl.com/5n8mjtws). Therefore users should probably arrange for
	// this to be no larger than IoSize.
	//
	// On Linux this can be any value, and will be faithfully returned to the
	// caller of statfs(2) (see the code walk above). On OS X it appears that
	// only powers of 2 in the range [2^7, 2^20] are preserved, and a value of
	// zero is treated as 4096.
	//
	// This interface does not distinguish between blocks and block fragments.
	BlockSize uint32

	// The total number of blocks in the file system, the number of unused
	// blocks, and the count of the latter that are available for use by non-root
	// users.
	//
	// For each category, the corresponding number of bytes is derived by
	// multiplying by BlockSize.
	Blocks          uint64
	BlocksFree      uint64
	BlocksAvailable uint64

	// The preferred size of writes to and reads from the file system, in bytes.
	// This may affect clients that use statfs(2) to size buffers correctly. It
	// does not appear to influence the size of writes sent from the kernel to
	// the file system daemon.
	//
	// On Linux this is surfaced as statfs::f_bsize, and on OS X as
	// statfs::f_iosize. Both are documented in `man 2 statfs` as "optimal
	// transfer block size".
	//
	// On Linux this can be any value. On OS X it appears that only powers of 2
	// in the range [2^12, 2^25] are faithfully preserved, and a value of zero is
	// treated as 65536.
	IoSize uint32

	// The total number of inodes in the file system, and how many remain free.
	Inodes     uint64
	InodesFree uint64
}

////////////////////////////////////////////////////////////////////////
// Inodes
////////////////////////////////////////////////////////////////////////

// Look up a child by name within a parent directory. The kernel sends this
// when resolving user paths to dentry structs, which are then cached.
type LookUpInodeOp struct {
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

	// The resulting entry. Must be filled out by the file system.
	//
	// The lookup count for the inode is implicitly incremented. See notes on
	// ForgetInodeOp for more information.
	Entry     ChildInodeEntry
	OpContext OpContext
}

// Refresh the attributes for an inode whose ID was previously returned in a
// LookUpInodeOp. The kernel sends this when the FUSE VFS layer's cache of
// inode attributes is stale. This is controlled by the AttributesExpiration
// field of ChildInodeEntry, etc.
type GetInodeAttributesOp struct {
	// The inode of interest.
	Inode InodeID

	// Set by the file system: attributes for the inode, and the time at which
	// they should expire. See notes on ChildInodeEntry.AttributesExpiration for
	// more.
	Attributes           InodeAttributes
	AttributesExpiration time.Time
	OpContext            OpContext
}

// Change attributes for an inode.
//
// The kernel sends this for obvious cases like chmod(2), and for less obvious
// cases like ftrunctate(2).
type SetInodeAttributesOp struct {
	// The inode of interest.
	Inode InodeID

	// If set, this is ftruncate(2), otherwise it's truncate(2)
	Handle *HandleID

	// The attributes to modify, or nil for attributes that don't need a change.
	Uid   *uint32
	Gid   *uint32
	Size  *uint64
	Mode  *os.FileMode
	Atime *time.Time
	Mtime *time.Time

	// Set by the file system: the new attributes for the inode, and the time at
	// which they should expire. See notes on
	// ChildInodeEntry.AttributesExpiration for more.
	Attributes           InodeAttributes
	AttributesExpiration time.Time
	OpContext            OpContext
}

// Decrement the reference count for an inode ID previously issued by the file
// system.
//
// The comments for the ops that implicitly increment the reference count
// contain a note of this (but see also the note about the root inode below).
// For example, LookUpInodeOp and MkDirOp. The authoritative source is the
// libfuse documentation, which states that any op that returns
// fuse_reply_entry fuse_reply_create implicitly increments
// (https://tinyurl.com/2xd5zssm).
//
// If the reference count hits zero, the file system can forget about that ID
// entirely, and even re-use it in future responses. The kernel guarantees that
// it will not otherwise use it again.
//
// The reference count corresponds to fuse_inode::nlookup
// (https://tinyurl.com/ycka69ck). Some examples of where the kernel
// manipulates it:
//
//   - (https://tinyurl.com/s8dz2ays) Any caller to fuse_iget increases the
//     count.
//   - (https://tinyurl.com/mu37ceua) fuse_lookup_name calls fuse_iget.
//   - (https://tinyurl.com/2nyhhnsh) fuse_create_open calls fuse_iget.
//   - (https://tinyurl.com/mnjpu3a9) fuse_dentry_revalidate increments after
//     revalidating.
//
// In contrast to all other inodes, RootInodeID begins with an implicit
// lookup count of one, without a corresponding op to increase it. (There
// could be no such op, because the root cannot be referred to by name.) Code
// walk:
//
//   - (https://tinyurl.com/yf8m2drx) fuse_fill_super calls
//     fuse_get_root_inode.
//
//   - (https://tinyurl.com/35f86asu) fuse_get_root_inode calls fuse_iget
//     without sending any particular request.
//
//   - (https://tinyurl.com/s8dz2ays) fuse_iget increments nlookup.
//
// File systems should tolerate but not rely on receiving forget ops for
// remaining inodes when the file system unmounts, including the root inode.
// Rather they should take fuse.Connection.ReadOp returning io.EOF as
// implicitly decrementing all lookup counts to zero.
type ForgetInodeOp struct {
	// The inode whose reference count should be decremented.
	Inode InodeID

	// The amount to decrement the reference count.
	N         uint64
	OpContext OpContext
}

// BatchForgetEntry represents one Inode entry to forget in the BatchForgetOp.
//
// Everything written in the ForgetInodeOp docs applies for the BatchForgetEntry
// too.
type BatchForgetEntry struct {
	// The inode whose reference count should be decremented.
	Inode InodeID

	// The amount to decrement the reference count.
	N uint64
}

// Decrement the reference counts for a list of inode IDs previously issued by the file
// system.
//
// This operation is a batch of ForgetInodeOp operations. Every entry in
// Entries is one ForgetInodeOp operation. See the docs of ForgetInodeOp
// for further details.
type BatchForgetOp struct {
	// Entries is a list of Forget operations. One could treat every entry in the
	// list as a single ForgetInodeOp operation.
	Entries []BatchForgetEntry

	OpContext OpContext
}

////////////////////////////////////////////////////////////////////////
// Inode creation
////////////////////////////////////////////////////////////////////////

// Create a directory inode as a child of an existing directory inode. The
// kernel sends this in response to a mkdir(2) call.
//
// The Linux kernel appears to verify the name doesn't already exist (mkdir
// calls mkdirat calls user_path_create calls filename_create, which verifies:
// https://tinyurl.com/24yw46mf). Indeed, the tests in samples/memfs that call
// in parallel appear to bear this out. But osxfuse does not appear to
// guarantee this (https://tinyurl.com/22587hcf). And if names may be created
// outside of the kernel's control, it doesn't matter what the kernel does
// anyway.
//
// Therefore the file system should return EEXIST if the name already exists.
type MkDirOp struct {
	// The ID of parent directory inode within which to create the child.
	Parent InodeID

	// The name of the child to create, and the mode with which to create it.
	Name string
	Mode os.FileMode

	// Set by the file system: information about the inode that was created.
	//
	// The lookup count for the inode is implicitly incremented. See notes on
	// ForgetInodeOp for more information.
	Entry     ChildInodeEntry
	OpContext OpContext
}

// Create a file inode as a child of an existing directory inode. The kernel
// sends this in response to a mknod(2) call. It may also send it in special
// cases such as an NFS export (https://tinyurl.com/5dwxr7c9). It is more typical
// to see CreateFileOp, which is received for an open(2) that creates a file.
//
// The Linux kernel appears to verify the name doesn't already exist (mknod
// calls sys_mknodat calls user_path_create calls filename_create, which
// verifies: https://tinyurl.com/24yw46mf). But osxfuse may not guarantee this,
// as with mkdir(2). And if names may be created outside of the kernel's
// control, it doesn't matter what the kernel does anyway.
//
// Therefore the file system should return EEXIST if the name already exists.
type MkNodeOp struct {
	// The ID of parent directory inode within which to create the child.
	Parent InodeID

	// The name of the child to create, and the mode with which to create it.
	Name string
	Mode os.FileMode

	// The device number (only valid if created file is a device)
	Rdev uint32

	// Set by the file system: information about the inode that was created.
	//
	// The lookup count for the inode is implicitly incremented. See notes on
	// ForgetInodeOp for more information.
	Entry     ChildInodeEntry
	OpContext OpContext
}

// Create a file inode and open it.
//
// The kernel sends this when the user asks to open a file with the O_CREAT
// flag and the kernel has observed that the file doesn't exist. (See for
// example lookup_open, https://tinyurl.com/49899mvb). However, osxfuse doesn't
// appear to make this check atomically (https://tinyurl.com/22587hcf). And if
// names may be created outside of the kernel's control, it doesn't matter what
// the kernel does anyway.
//
// Therefore the file system should return EEXIST if the name already exists.
type CreateFileOp struct {
	// The ID of parent directory inode within which to create the child file.
	Parent InodeID

	// The name of the child to create, and the mode with which to create it.
	Name string
	Mode os.FileMode

	// Set by the file system: information about the inode that was created.
	//
	// The lookup count for the inode is implicitly incremented. See notes on
	// ForgetInodeOp for more information.
	Entry ChildInodeEntry

	// Set by the file system: an opaque ID that will be echoed in follow-up
	// calls for this file using the same struct file in the kernel. In practice
	// this usually means follow-up calls using the file descriptor returned by
	// open(2).
	//
	// The handle may be supplied in future ops like ReadFileOp that contain a
	// file handle. The file system must ensure this ID remains valid until a
	// later call to ReleaseFileHandle.
	Handle    HandleID
	OpContext OpContext
}

// Create a symlink inode. If the name already exists, the file system should
// return EEXIST (cf. the notes on CreateFileOp and MkDirOp).
type CreateSymlinkOp struct {
	// The ID of parent directory inode within which to create the child symlink.
	Parent InodeID

	// The name of the symlink to create.
	Name string

	// The target of the symlink.
	Target string

	// Set by the file system: information about the symlink inode that was
	// created.
	//
	// The lookup count for the inode is implicitly incremented. See notes on
	// ForgetInodeOp for more information.
	Entry     ChildInodeEntry
	OpContext OpContext
}

// Create a hard link to an inode. If the name already exists, the file system
// should return EEXIST (cf. the notes on CreateFileOp and MkDirOp).
type CreateLinkOp struct {
	// The ID of parent directory inode within which to create the child hard
	// link.
	Parent InodeID

	// The name of the new inode.
	Name string

	// The ID of the target inode.
	Target InodeID

	// Set by the file system: information about the inode that was created.
	//
	// The lookup count for the inode is implicitly incremented. See notes on
	// ForgetInodeOp for more information.
	Entry     ChildInodeEntry
	OpContext OpContext
}

////////////////////////////////////////////////////////////////////////
// Unlinking
////////////////////////////////////////////////////////////////////////

// Rename a file or directory, given the IDs of the original parent directory
// and the new one (which may be the same).
//
// In Linux, this is called by vfs_rename (https://tinyurl.com/2xbx9kr2), which
// is called by sys_renameat2 (https://tinyurl.com/4zyak2kt).
//
// The kernel takes care of ensuring that the source and destination are not
// identical (in which case it does nothing), that the rename is not across
// file system boundaries, and that the destination doesn't already exist with
// the wrong type. Some subtleties that the file system must care about:
//
//   - If the new name is an existing directory, the file system must ensure it
//     is empty before replacing it, returning ENOTEMPTY otherwise. (This is
//     per the posix spec: https://tinyurl.com/5n865nx9)
//
//   - The rename must be atomic from the point of view of an observer of the
//     new name. That is, if the new name already exists, there must be no
//     point at which it doesn't exist.
//
//   - It is okay for the new name to be modified before the old name is
//     removed; these need not be atomic. In fact, the Linux man page
//     explicitly says this is likely (https://tinyurl.com/mdpbpjmr).
//
//   - Linux bends over backwards (https://tinyurl.com/3hmt7puy) to ensure that
//     neither the old nor the new parent can be concurrently modified. But
//     it's not clear whether OS X does this, and in any case it doesn't matter
//     for file systems that may be modified remotely. Therefore a careful file
//     system implementor should probably ensure if possible that the unlink
//     step in the "link new name, unlink old name" process doesn't unlink a
//     different inode than the one that was linked to the new name. Still,
//     posix and the man pages are imprecise about the actual semantics of a
//     rename if it's not atomic, so it is probably not disastrous to be loose
//     about this.
type RenameOp struct {
	// The old parent directory, and the name of the entry within it to be
	// relocated.
	OldParent InodeID
	OldName   string

	// The new parent directory, and the name of the entry to be created or
	// overwritten within it.
	NewParent InodeID
	NewName   string
	OpContext OpContext
}

// Unlink a directory from its parent. Because directories cannot have a link
// count above one, this means the directory inode should be deleted as well
// once the kernel sends ForgetInodeOp.
//
// The file system is responsible for checking that the directory is empty.
//
// Sample implementation in ext2: ext2_rmdir (https://tinyurl.com/bajkpcf9)
type RmDirOp struct {
	// The ID of parent directory inode, and the name of the directory being
	// removed within it.
	Parent    InodeID
	Name      string
	OpContext OpContext
}

// Unlink a file or symlink from its parent. If this brings the inode's link
// count to zero, the inode should be deleted once the kernel sends
// ForgetInodeOp. It may still be referenced before then if a user still has
// the file open.
//
// Sample implementation in ext2: ext2_unlink (https://tinyurl.com/3wpwedcp)
type UnlinkOp struct {
	// The ID of parent directory inode, and the name of the entry being removed
	// within it.
	Parent    InodeID
	Name      string
	OpContext OpContext
}

////////////////////////////////////////////////////////////////////////
// Directory handles
////////////////////////////////////////////////////////////////////////

// Open a directory inode.
//
// On Linux the kernel sends this when setting up a struct file for a particular inode
// with type directory, usually in response to an open(2) call from a
// user-space process. On OS X it may not be sent for every open(2) (cf.
// https://github.com/osxfuse/osxfuse/issues/199).
type OpenDirOp struct {
	// The ID of the inode to be opened.
	Inode InodeID

	// Set by the file system: an opaque ID that will be echoed in follow-up
	// calls for this directory using the same struct file in the kernel. In
	// practice this usually means follow-up calls using the file descriptor
	// returned by open(2).
	//
	// The handle may be supplied in future ops like ReadDirOp that contain a
	// directory handle. The file system must ensure this ID remains valid until
	// a later call to ReleaseDirHandle.
	Handle    HandleID
	OpContext OpContext

	// CacheDir conveys to the kernel to cache the response of next
	// ReadDirOp as page cache. Once cached, listing on that directory will be
	// served from the kernel until invalidated.
	CacheDir bool

	// KeepCache instructs the kernel to not invalidate the data cache on open calls.
	KeepCache bool
}

// Read entries from a directory previously opened with OpenDir.
type ReadDirOp struct {
	// The directory inode that we are reading, and the handle previously
	// returned by OpenDir when opening that inode.
	Inode  InodeID
	Handle HandleID

	// The offset within the directory at which to read.
	//
	// Warning: this field is not necessarily a count of bytes. Its legal values
	// are defined by the results returned in ReadDirResponse. See the notes
	// below and the notes on that struct.
	//
	// In the Linux kernel this ultimately comes from file::f_pos, which starts
	// at zero and is set by llseek and by the final consumed result returned by
	// each call to ReadDir:
	//
	//  *  (https://tinyurl.com/3ueykmaj) iterate_dir, which is called by
	//     getdents(2) and readdir(2), sets dir_context::pos to file::f_pos
	//     before calling f_op->iterate, and then does the opposite assignment
	//     afterward.
	//
	//  *  (https://tinyurl.com/a8urhfy9) fuse_readdir, which implements iterate
	//     for fuse directories, passes dir_context::pos as the offset to
	//     fuse_read_fill, which passes it on to user-space. fuse_readdir later
	//     calls parse_dirfile with the same context.
	//
	//  *  (https://tinyurl.com/5cev5fn4) For each returned result (except
	//     perhaps the last, which may be truncated by the page boundary),
	//     parse_dirfile updates dir_context::pos with fuse_dirent::off.
	//
	// It is affected by the Posix directory stream interfaces in the following
	// manner:
	//
	//  *  (https://tinyurl.com/2pjv5jvz, https://tinyurl.com/2r6h4mkj) opendir
	//     initially causes filepos to be set to zero.
	//
	//  *  (https://tinyurl.com/2yvcbcpv, https://tinyurl.com/bddezwp4) readdir
	//     allows the user to iterate through the directory one entry at a time.
	//     As each entry is consumed, its d_off field is stored in
	//     __dirstream::filepos.
	//
	//  *  (https://tinyurl.com/2pfbfe9v, https://tinyurl.com/4wtat58a) telldir
	//     allows the user to obtain the d_off field from the most recently
	//     returned entry.
	//
	//  *  (https://tinyurl.com/bdynryef, https://tinyurl.com/4hysrnb8) seekdir
	//     allows the user to seek backward to an offset previously returned by
	//     telldir. It stores the new offset in filepos, and calls llseek to
	//     update the kernel's struct file.
	//
	//  *  (https://tinyurl.com/5n8dkb44, https://tinyurl.com/3jnn5nnn) rewinddir
	//     allows the user to go back to the beginning of the directory,
	//     obtaining a fresh view. It updates filepos and calls llseek to update
	//     the kernel's struct file.
	//
	// Unfortunately, FUSE offers no way to intercept seeks
	// (https://tinyurl.com/4bm2sfjd), so there is no way to cause seekdir or
	// rewinddir to fail. Additionally, there is no way to distinguish an
	// explicit rewinddir followed by readdir from the initial readdir, or a
	// rewinddir from a seekdir to the value returned by telldir just after
	// opendir.
	//
	// Luckily, Posix is vague about what the user will see if they seek
	// backwards, and requires the user not to seek to an old offset after a
	// rewind. The only requirement on freshness is that rewinddir results in
	// something that looks like a newly-opened directory. So FUSE file systems
	// may e.g. cache an entire fresh listing for each ReadDir with a zero
	// offset, and return array offsets into that cached listing.
	Offset DirOffset

	// The destination buffer, whose length gives the size of the read.
	//
	// The output data should consist of a sequence of FUSE directory entries in
	// the format generated by fuse_add_direntry (https://tinyurl.com/3r9t7d2p),
	// which is consumed by parse_dirfile (https://tinyurl.com/bevwty74). Use
	// fuseutil.WriteDirent to generate this data.
	//
	// Each entry returned exposes a directory offset to the user that may later
	// show up in ReadDirRequest.Offset. See notes on that field for more
	// information.
	Dst []byte

	// Set by the file system: the number of bytes read into Dst.
	//
	// It is okay for this to be less than len(Dst) if there are not enough
	// entries available or the final entry would not fit.
	//
	// Zero means that the end of the directory has been reached. This is
	// unambiguous because NAME_MAX (https://tinyurl.com/4r2b68jp) plus the size
	// of fuse_dirent (https://tinyurl.com/mp43bu8) plus the 8-byte alignment of
	// FUSE_DIRENT_ALIGN (https://tinyurl.com/3m3ewu7h) is less than the read
	// size of PAGE_SIZE used by fuse_readdir (https://tinyurl.com/mrwxsfxw).
	BytesRead int
	OpContext OpContext
}

// Release a previously-minted directory handle. The kernel sends this when
// there are no more references to an open directory: all file descriptors are
// closed and all memory mappings are unmapped.
//
// The kernel guarantees that the handle ID will not be used in further ops
// sent to the file system (unless it is reissued by the file system).
//
// Errors from this op are ignored by the kernel
// (https://tinyurl.com/2aaccyzk).
type ReleaseDirHandleOp struct {
	// The handle ID to be released. The kernel guarantees that this ID will not
	// be used in further calls to the file system (unless it is reissued by the
	// file system).
	Handle    HandleID
	OpContext OpContext
}

////////////////////////////////////////////////////////////////////////
// File handles
////////////////////////////////////////////////////////////////////////

// Open a file inode.
//
// On Linux the kernel sends this when setting up a struct file for a particular inode
// with type file, usually in response to an open(2) call from a user-space
// process. On OS X it may not be sent for every open(2)
// (cf.https://github.com/osxfuse/osxfuse/issues/199).
type OpenFileOp struct {
	// The ID of the inode to be opened.
	Inode InodeID

	// An opaque ID that will be echoed in follow-up calls for this file using
	// the same struct file in the kernel. In practice this usually means
	// follow-up calls using the file descriptor returned by open(2).
	//
	// The handle may be supplied in future ops like ReadFileOp that contain a
	// file handle. The file system must ensure this ID remains valid until a
	// later call to ReleaseFileHandle.
	Handle HandleID

	// By default, fuse invalidates the kernel's page cache for an inode when a
	// new file handle is opened for that inode (https://tinyurl.com/yyb497zy).
	// The intent appears to be to allow users to "see" content that has changed
	// remotely on a networked file system by re-opening the file.
	//
	// For file systems where this is not a concern because all modifications for
	// a particular inode go through the kernel, set this field to true to
	// disable this behavior.
	//
	// (More discussion: https://tinyurl.com/4znxvzwh)
	//
	// Note that on OS X it appears that the behavior is always as if this field
	// is set to true, regardless of its value, at least for files opened in the
	// same mode. (Cf. https://github.com/osxfuse/osxfuse/issues/223)
	KeepPageCache bool

	// Whether to use direct IO for this file handle. By default, the kernel
	// suppresses what it sees as redundant operations (including reads beyond
	// the precomputed EOF).
	//
	// Enabling direct IO ensures that all client operations reach the fuse
	// layer. This allows for filesystems whose file sizes are not known in
	// advance, for example, because contents are generated on the fly.
	UseDirectIO bool

	OpenFlags fusekernel.OpenFlags

	OpContext OpContext
}

// Read data from a file previously opened with CreateFile or OpenFile.
//
// Note that this op is not sent for every call to read(2) by the end user;
// some reads may be served by the page cache. See notes on WriteFileOp for
// more.
type ReadFileOp struct {
	// The file inode that we are reading, and the handle previously returned by
	// CreateFile or OpenFile when opening that inode.
	Inode  InodeID
	Handle HandleID

	// The offset within the file at which to read.
	Offset int64

	// The size of the read.
	Size int64

	// The destination buffer, whose length gives the size of the read.
	// For vectored reads, this field is always nil as the buffer is not provided.
	Dst []byte

	// Set by the file system:
	// A list of slices of data to send back to the client for vectored reads.
	Data [][]byte

	// Set by the file system: the number of bytes read.
	//
	// The FUSE documentation requires that exactly the requested number of bytes
	// be returned, except in the case of EOF or error
	// (https://tinyurl.com/2mzewn35). This appears to be because it uses file
	// mmapping machinery (https://tinyurl.com/avxy3dvm) to read a page at a
	// time. It appears to understand where EOF is by checking the inode size
	// (https://tinyurl.com/2eteerzt), returned by a previous call to
	// LookUpInode, GetInodeAttributes, etc.
	//
	// If direct IO is enabled, semantics should match those of read(2).
	BytesRead int
	OpContext OpContext

	// If set, this function will be invoked after the operation response has been
	// sent to the kernel and before the buffers containing the response data are
	// freed.
	Callback func()
}

// Write data to a file previously opened with CreateFile or OpenFile.
//
// When the user writes data using write(2), the write goes into the page
// cache and the page is marked dirty. Later the kernel may write back the
// page via the FUSE VFS layer, causing this op to be sent:
//
//   - The kernel calls address_space_operations::writepage when a dirty page
//     needs to be written to backing store (https://tinyurl.com/yck2sf5u).
//     Fuse sets this to fuse_writepage (https://tinyurl.com/5n989f8p).
//
//   - (https://tinyurl.com/mvn6zv3j) fuse_writepage calls
//     fuse_writepage_locked.
//
//   - (https://tinyurl.com/2wn8scwb) fuse_writepage_locked makes a write
//     request to the userspace server.
//
// Note that the kernel *will* ensure that writes are received and acknowledged
// by the file system before sending a FlushFileOp when closing the file
// descriptor to which they were written. Cf. the notes on
// fuse.MountConfig.DisableWritebackCaching.
//
// (See also https://tinyurl.com/5dchkdtx, fuse-devel thread "Fuse guarantees
// on concurrent requests".)
type WriteFileOp struct {
	// The file inode that we are modifying, and the handle previously returned
	// by CreateFile or OpenFile when opening that inode.
	Inode  InodeID
	Handle HandleID

	// The offset at which to write the data below.
	//
	// The man page for pwrite(2) implies that aside from changing the file
	// handle's offset, using pwrite is equivalent to using lseek(2) and then
	// write(2). The man page for lseek(2) says the following:
	//
	// "The lseek() function allows the file offset to be set beyond the end of
	// the file (but this does not change the size of the file). If data is later
	// written at this point, subsequent reads of the data in the gap (a "hole")
	// return null bytes (aq\0aq) until data is actually written into the gap."
	//
	// It is therefore reasonable to assume that the kernel is looking for
	// the following semantics:
	//
	// *   If the offset is less than or equal to the current size, extend the
	//     file as necessary to fit any data that goes past the end of the file.
	//
	// *   If the offset is greater than the current size, extend the file
	//     with null bytes until it is not, then do the above.
	//
	Offset int64

	// The data to write.
	//
	// The FUSE documentation requires that exactly the number of bytes supplied
	// be written, except on error (https://tinyurl.com/yuruk5tx). This appears
	// to be because it uses file mmapping machinery
	// (https://tinyurl.com/avxy3dvm) to write a page at a time.
	Data      []byte
	OpContext OpContext

	// If set, this function will be invoked after the operation response has been
	// sent to the kernel and before the buffers containing the response data are
	// freed.
	Callback func()
}

// Synchronize the current contents of an open file to storage.
//
// vfs.txt documents this as being called for by the fsync(2) system call
// (https://tinyurl.com/y2kdrfzw). Code walk for that case:
//
//   - (https://tinyurl.com/2s44cefz) sys_fsync calls do_fsync, calls
//     vfs_fsync, calls vfs_fsync_range.
//
//   - (https://tinyurl.com/bdhhfam5) vfs_fsync_range calls f_op->fsync.
//
// Note that this is also sent by fdatasync(2) (https://tinyurl.com/ja5wtszf),
// and may be sent for msync(2) with the MS_SYNC flag (see the notes on
// FlushFileOp).
//
// See also: FlushFileOp, which may perform a similar function when closing a
// file (but which is not used in "real" file systems).
type SyncFileOp struct {
	// The file and handle being sync'd.
	Inode     InodeID
	Handle    HandleID
	OpContext OpContext
}

// Flush the current state of an open file to storage upon closing a file
// descriptor.
//
// vfs.txt documents this as being sent for each close(2) system call
// (https://tinyurl.com/r4ujfxkc). Code walk for that case:
//
//   - (https://tinyurl.com/2kzyyjcu) sys_close calls __close_fd, calls
//     filp_close.

//   - (https://tinyurl.com/4zdxrz52) filp_close calls f_op->flush
//     (fuse_flush).
//
// But note that this is also sent in other contexts where a file descriptor is
// closed, such as dup2(2) (https://tinyurl.com/5bj3z3f5). In the case of
// close(2), a flush error is returned to the user. For dup2(2), it is not.
//
// One potentially significant case where this may not be sent is mmap'd files,
// where the behavior is complicated:
//
//   - munmap(2) does not cause flushes (https://tinyurl.com/ycy9z2jb).
//
//   - On OS X, if a user modifies a mapped file via the mapping before closing
//     the file with close(2), the WriteFileOps for the modifications may not
//     be received before the FlushFileOp for the close(2) (cf.
//     https://github.com/osxfuse/osxfuse/issues/202). It appears that this may
//     be fixed in osxfuse 3 (https://tinyurl.com/2ne2jv8u).
//
//   - However, you safely can arrange for writes via a mapping to be flushed
//     by calling msync(2) followed by close(2). On OS X msync(2) will cause a
//     WriteFileOps to go through and close(2) will cause a FlushFile as usual
//     (https://tinyurl.com/2p9b4axf). On Linux, msync(2) does nothing unless
//     you set the MS_SYNC flag, in which case it causes a SyncFileOp to be
//     sent (https://tinyurl.com/2y3d9hhj).
//
// In summary: if you make data durable in both FlushFile and SyncFile, then
// your users can get safe behavior from mapped files on both operating systems
// by calling msync(2) with MS_SYNC, followed by munmap(2), followed by
// close(2). On Linux, the msync(2) is optional (cf.
// https://tinyurl.com/unesszdp and the notes on WriteFileOp).
//
// Because of cases like dup2(2), FlushFileOps are not necessarily one to one
// with OpenFileOps. They should not be used for reference counting, and the
// handle must remain valid even after the flush op is received (use
// ReleaseFileHandleOp for disposing of it).
//
// Typical "real" file systems do not implement this, presumably relying on
// the kernel to write out the page cache to the block device eventually.
// They can get away with this because a later open(2) will see the same
// data. A file system that writes to remote storage however probably wants
// to at least schedule a real flush, and maybe do it immediately in order to
// return any errors that occur.
type FlushFileOp struct {
	// The file and handle being flushed.
	Inode     InodeID
	Handle    HandleID
	OpContext OpContext
}

// Release a previously-minted file handle. The kernel calls this when there
// are no more references to an open file: all file descriptors are closed
// and all memory mappings are unmapped.
//
// The kernel guarantees that the handle ID will not be used in further calls
// to the file system (unless it is reissued by the file system).
//
// Errors from this op are ignored by the kernel
// (https://tinyurl.com/2aaccyzk).
type ReleaseFileHandleOp struct {
	// The handle ID to be released. The kernel guarantees that this ID will not
	// be used in further calls to the file system (unless it is reissued by the
	// file system).
	Handle    HandleID
	OpContext OpContext
}

////////////////////////////////////////////////////////////////////////
// Reading symlinks
////////////////////////////////////////////////////////////////////////

// Read the target of a symlink inode.
type ReadSymlinkOp struct {
	// The symlink inode that we are reading.
	Inode InodeID

	// Set by the file system: the target of the symlink.
	Target    string
	OpContext OpContext
}

////////////////////////////////////////////////////////////////////////
// eXtended attributes
////////////////////////////////////////////////////////////////////////

// Remove an extended attribute.
//
// This is sent in response to removexattr(2). Return ENOATTR if the
// extended attribute does not exist.
type RemoveXattrOp struct {
	// The inode that we are removing an extended attribute from.
	Inode InodeID

	// The name of the extended attribute.
	Name      string
	OpContext OpContext
}

// Get an extended attribute.
//
// This is sent in response to getxattr(2). Return ENOATTR if the
// extended attribute does not exist.
type GetXattrOp struct {
	// The inode whose extended attribute we are reading.
	Inode InodeID

	// The name of the extended attribute.
	Name string

	// The destination buffer.  If the size is too small for the
	// value, the ERANGE error should be sent.
	Dst []byte

	// Set by the file system: the number of bytes read into Dst, or
	// the number of bytes that would have been read into Dst if Dst was
	// big enough (return ERANGE in this case).
	BytesRead int
	OpContext OpContext
}

// List all the extended attributes for a file.
//
// This is sent in response to listxattr(2).
type ListXattrOp struct {
	// The inode whose extended attributes we are listing.
	Inode InodeID

	// The destination buffer.  If the size is too small for the
	// value, the ERANGE error should be sent.
	//
	// The output data should consist of a sequence of NUL-terminated strings,
	// one for each xattr.
	Dst []byte

	// Set by the file system: the number of bytes read into Dst, or
	// the number of bytes that would have been read into Dst if Dst was
	// big enough (return ERANGE in this case).
	BytesRead int
	OpContext OpContext
}

// Set an extended attribute.
//
// This is sent in response to setxattr(2). Return ENOSPC if there is
// insufficient space remaining to store the extended attribute.
type SetXattrOp struct {
	// The inode whose extended attribute we are setting.
	Inode InodeID

	// The name of the extended attribute
	Name string

	// The value to for the extened attribute.
	Value []byte

	// If Flags is 0x1, and the attribute exists already, EEXIST should be returned.
	// If Flags is 0x2, and the attribute does not exist, ENOATTR should be returned.
	// If Flags is 0x0, the extended attribute will be created if need be, or will
	// simply replace the value if the attribute exists.
	Flags     uint32
	OpContext OpContext
}

type FallocateOp struct {
	// The inode and handle we are fallocating
	Inode  InodeID
	Handle HandleID

	// Start of the byte range
	Offset uint64

	// Length of the byte range
	Length uint64

	// If Mode is 0x0, allocate disk space within the range specified
	// If Mode has 0x1, allocate the space but don't increase the file size
	// If Mode has 0x2, deallocate space within the range specified
	// If Mode has 0x2, it sbould also have 0x1 (deallocate should not increase
	// file size)
	Mode      uint32
	OpContext OpContext
}

type SyncFSOp struct {
	Inode     InodeID
	OpContext OpContext
}
