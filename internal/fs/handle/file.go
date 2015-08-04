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

package handle

import (
	"errors"

	"github.com/googlecloudplatform/gcsfuse/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"golang.org/x/net/context"
)

type FileHandle struct {
	inode *inode.FileInode

	// A random reader configured to some (potentially previous) generation of
	// the object backing the inode, or nil.
	//
	// INVARIANT: If reader != nil, reader.CheckInvariants() doesn't panic.
	//
	// GUARDED_BY(inode)
	reader gcsx.RandomReader
}

func NewFileHandle(inode *inode.FileInode) (fh *FileHandle, err error) {
	err = errors.New("TODO")
	return
}

// Panic if any internal invariants are violated.
//
// LOCKS_REQUIRED(fh.inode)
func (fh *FileHandle) CheckInvariants() {
	// INVARIANT: If reader != nil, reader.CheckInvariants() doesn't panic.
	if fh.reader != nil {
		fh.reader.CheckInvariants()
	}
}

// Destroy any resources associated with the handle, which must not be used
// again.
func (fh *FileHandle) Destroy() {
	if fh.reader != nil {
		fh.reader.Destroy()
	}
}

// Return the inode backing this handle.
func (fh *FileHandle) Inode() *inode.FileInode {
	return fh.inode
}

// Equivalent to locking fh.Inode() and calling fh.Inode().Read, but may be
// more efficient.
//
// LOCKS_EXCLUDED(fh.inode)
func (fh *FileHandle) Read(
	ctx context.Context,
	dst []byte,
	offset int64) (n int, err error) {
	err = errors.New("TODO")
	return
}
