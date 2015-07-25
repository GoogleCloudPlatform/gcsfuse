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
// See notes on fuseops.ReadDirOp and on AppendDirent for details.
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

// Append the supplied directory entry to the given buffer in the format
// expected in fuseops.ReadFileOp.Data, returning the resulting buffer.
func AppendDirent(input []byte, d Dirent) (output []byte) {
	// We want to append bytes with the layout of fuse_dirent
	// (http://goo.gl/BmFxob) in host order. The struct must be aligned according
	// to FUSE_DIRENT_ALIGN (http://goo.gl/UziWvH), which dictates 8-byte
	// alignment.
	type fuse_dirent struct {
		ino     uint64
		off     uint64
		namelen uint32
		type_   uint32
		name    [0]byte
	}

	const alignment = 8
	const nameOffset = 8 + 8 + 4 + 4

	// Write the header into the buffer.
	de := fuse_dirent{
		ino:     uint64(d.Inode),
		off:     uint64(d.Offset),
		namelen: uint32(len(d.Name)),
		type_:   uint32(d.Type),
	}

	output = append(input, (*[nameOffset]byte)(unsafe.Pointer(&de))[:]...)

	// Write the name afterward.
	output = append(output, d.Name...)

	// Add any necessary padding.
	if len(d.Name)%alignment != 0 {
		padLen := alignment - (len(d.Name) % alignment)

		var padding [alignment]byte
		output = append(output, padding[:padLen]...)
	}

	return
}
