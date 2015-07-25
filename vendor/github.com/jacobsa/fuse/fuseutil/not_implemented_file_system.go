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
	"github.com/jacobsa/fuse/fuseops"
)

// A FileSystem that responds to all ops with fuse.ENOSYS. Embed this in your
// struct to inherit default implementations for the methods you don't care
// about, ensuring your struct will continue to implement FileSystem even as
// new methods are added.
type NotImplementedFileSystem struct {
}

var _ FileSystem = &NotImplementedFileSystem{}

func (fs *NotImplementedFileSystem) LookUpInode(
	op *fuseops.LookUpInodeOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) GetInodeAttributes(
	op *fuseops.GetInodeAttributesOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) SetInodeAttributes(
	op *fuseops.SetInodeAttributesOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) ForgetInode(
	op *fuseops.ForgetInodeOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) MkDir(
	op *fuseops.MkDirOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) CreateFile(
	op *fuseops.CreateFileOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) CreateSymlink(
	op *fuseops.CreateSymlinkOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) Rename(
	op *fuseops.RenameOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) RmDir(
	op *fuseops.RmDirOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) Unlink(
	op *fuseops.UnlinkOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) OpenDir(
	op *fuseops.OpenDirOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) ReadDir(
	op *fuseops.ReadDirOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) ReleaseDirHandle(
	op *fuseops.ReleaseDirHandleOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) OpenFile(
	op *fuseops.OpenFileOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) ReadFile(
	op *fuseops.ReadFileOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) WriteFile(
	op *fuseops.WriteFileOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) SyncFile(
	op *fuseops.SyncFileOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) FlushFile(
	op *fuseops.FlushFileOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) ReleaseFileHandle(
	op *fuseops.ReleaseFileHandleOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) ReadSymlink(
	op *fuseops.ReadSymlinkOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *NotImplementedFileSystem) Destroy() {
}
