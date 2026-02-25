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

package fuseutil

import (
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/internal/fusekernel"
	"syscall"
	"unsafe"

	"github.com/jacobsa/fuse/fuseops"
)

type DirentType uint32

const (
	DT_Unknown   DirentType = 0
	DT_Socket    DirentType = syscall.DT_SOCK
	DT_Link      DirentType = syscall.DT_LNK
	DT_File      DirentType = syscall.DT_REG
	DT_Block     DirentType = syscall.DT_BLK
	DT_Directory DirentType = syscall.DT_DIR
	DT_Char      DirentType = syscall.DT_CHR
	DT_FIFO      DirentType = syscall.DT_FIFO
)

// A struct representing an entry within a directory file, describing a child.
// See notes on fuseops.ReadDirOp and on WriteDirent for details.
type Dirent struct {
	// The (opaque) offset within the directory file of the entry following this
	// one. See notes on fuseops.ReadDirOp.Offset for details.
	Offset fuseops.DirOffset

	// The inode of the child file or directory, and its name within the parent.
	Inode fuseops.InodeID
	Name  string

	// The type of the child. The zero value (DT_Unknown) is legal, but means
	// that the kernel will need to call GetAttr when the type is needed.
	Type DirentType
}

// A struct representing an entry along with its attributes within a directory file,
// describing a child.
// See notes on fuseops.ReadDirPlusOp and on WriteDirentPlus for details.
type DirentPlus struct {
	// The basic directory entry information (offset, inode, name, type).
	Dirent Dirent

	// Detailed information about the child inode, including its attributes
	// (like size, mode, timestamps) and cache expiration times for both the
	// attributes and the name lookup.
	Entry fuseops.ChildInodeEntry
}

type fuse_dirent struct {
	ino     uint64
	off     uint64
	namelen uint32
	type_   uint32
	name    [0]byte
}

// Write the supplied directory entry into the given buffer in the format
// expected in fuseops.ReadFileOp.Data, returning the number of bytes written.
// Return zero if the entry would not fit.
func WriteDirent(buf []byte, d Dirent) (n int) {
	// We want to write bytes with the layout of fuse_dirent
	// (https://tinyurl.com/4k7y2h9r) in host order. The struct must be aligned
	// according to FUSE_DIRENT_ALIGN (https://tinyurl.com/3m3ewu7h), which
	// dictates 8-byte alignment.

	const direntAlignment = 8
	const direntSize = 8 + 8 + 4 + 4

	// Compute the number of bytes of padding we'll need to maintain alignment
	// for the next entry.
	var padLen int
	if len(d.Name)%direntAlignment != 0 {
		padLen = direntAlignment - (len(d.Name) % direntAlignment)
	}

	// Do we have enough room?
	totalLen := direntSize + len(d.Name) + padLen
	if totalLen > len(buf) {
		return n
	}

	// Write the header.
	de := fuse_dirent{
		ino:     uint64(d.Inode),
		off:     uint64(d.Offset),
		namelen: uint32(len(d.Name)),
		type_:   uint32(d.Type),
	}

	n += copy(buf[n:], (*[direntSize]byte)(unsafe.Pointer(&de))[:])

	// Write the name afterward.
	n += copy(buf[n:], d.Name)

	// Add any necessary padding.
	if padLen != 0 {
		var padding [direntAlignment]byte
		n += copy(buf[n:], padding[:padLen])
	}

	return n
}

// Write the supplied directory entry with attributes into the given buffer in the format
// expected in fuseops.ReadDirPlusOp.Dst returning the number of bytes written.
// Return zero if the entry would not fit.
func WriteDirentPlus(buf []byte, d DirentPlus) (n int) {
	type fuse_entry_out struct {
		nodeid           uint64
		generation       uint64
		entry_valid      uint64
		attr_valid       uint64
		entry_valid_nsec uint32
		attr_valid_nsec  uint32
		attr             fusekernel.Attr
	}

	// We want to write bytes with the layout of fuse_direntplus
	// (http://shortn/_LNqd8uXg2p) in host order. The struct must be aligned
	// according to FUSE_DIRENT_ALIGN (https://tinyurl.com/3m3ewu7h), which
	// dictates 8-byte alignment.
	type fuse_direntplus struct {
		entry_out fuse_entry_out
		dirent    fuse_dirent
	}

	const direntPlusAlignment = 8

	// size of fuse_attr
	const fuseAttrSize = 8 + 8 + 8 + 8 + 8 + 8 + 4 + 4 + 4 + 4 + 4 + 4 + 4 + 4 + 4 + 4

	// size of fuse_entry_out without fuse_attr
	const fuseEntryOutSize = 8 + 8 + 8 + 8 + 4 + 4

	// size of fuse_dirent
	const fuseDirentSize = 8 + 8 + 4 + 4

	const direntPlusHeaderSize = fuseAttrSize + fuseEntryOutSize + fuseDirentSize

	// Compute the number of bytes of padding we'll need to maintain alignment
	// for the next entry.
	var padLen int
	if pad := len(d.Dirent.Name) % direntPlusAlignment; pad != 0 {
		padLen = direntPlusAlignment - pad
	}

	// Do we have enough room?
	totalLen := int(direntPlusHeaderSize) + len(d.Dirent.Name) + padLen
	if totalLen > len(buf) {
		return 0
	}

	entryValid, entryValidNsec := fuse.ConvertExpirationTime(d.Entry.EntryExpiration)
	attrValid, attrValidNsec := fuse.ConvertExpirationTime(d.Entry.AttributesExpiration)
	var rdev uint32
	mode := fuse.ConvertGoMode(d.Entry.Attributes.Mode)
	if mode&(syscall.S_IFCHR|syscall.S_IFBLK) != 0 {
		rdev = d.Entry.Attributes.Rdev
	}

	// Write the header.
	dp := fuse_direntplus{
		entry_out: fuse_entry_out{
			nodeid:           uint64(d.Entry.Child),
			generation:       uint64(d.Entry.Generation),
			entry_valid:      entryValid,
			attr_valid:       attrValid,
			entry_valid_nsec: entryValidNsec,
			attr_valid_nsec:  attrValidNsec,
			attr: fusekernel.Attr{
				Ino:  uint64(d.Entry.Child),
				Size: d.Entry.Attributes.Size,
				// round up to the nearest 512 boundary (In POSIX a "block" is a unit of 512 bytes)
				Blocks:    (d.Entry.Attributes.Size + 512 - 1) / 512,
				Atime:     uint64(d.Entry.Attributes.Atime.UnixNano() / 1e9),
				Mtime:     uint64(d.Entry.Attributes.Mtime.UnixNano() / 1e9),
				Ctime:     uint64(d.Entry.Attributes.Ctime.UnixNano() / 1e9),
				AtimeNsec: uint32(d.Entry.Attributes.Atime.UnixNano() % 1e9),
				MtimeNsec: uint32(d.Entry.Attributes.Mtime.UnixNano() % 1e9),
				CtimeNsec: uint32(d.Entry.Attributes.Ctime.UnixNano() % 1e9),
				Mode:      mode,
				Nlink:     d.Entry.Attributes.Nlink,
				Rdev:      rdev,
				Uid:       d.Entry.Attributes.Uid,
				Gid:       d.Entry.Attributes.Gid,
			},
		},

		dirent: fuse_dirent{
			ino:     uint64(d.Dirent.Inode),
			off:     uint64(d.Dirent.Offset),
			namelen: uint32(len(d.Dirent.Name)),
			type_:   uint32(d.Dirent.Type),
		},
	}

	n += copy(buf[n:], (*[direntPlusHeaderSize]byte)(unsafe.Pointer(&dp))[:])

	// Write the name afterward.
	n += copy(buf[n:], d.Dirent.Name)

	// Add any necessary padding.
	if padLen != 0 {
		var padding [direntPlusAlignment]byte
		n += copy(buf[n:], padding[:padLen])
	}

	return n
}
