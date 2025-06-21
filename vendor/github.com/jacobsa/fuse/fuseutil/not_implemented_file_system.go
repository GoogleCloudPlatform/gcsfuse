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
	"context"

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

func (fs *NotImplementedFileSystem) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) LookUpInode(
	ctx context.Context,
	op *fuseops.LookUpInodeOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) SetInodeAttributes(
	ctx context.Context,
	op *fuseops.SetInodeAttributesOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) BatchForget(
	ctx context.Context,
	op *fuseops.BatchForgetOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) MkDir(
	ctx context.Context,
	op *fuseops.MkDirOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) MkNode(
	ctx context.Context,
	op *fuseops.MkNodeOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) CreateSymlink(
	ctx context.Context,
	op *fuseops.CreateSymlinkOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) CreateLink(
	ctx context.Context,
	op *fuseops.CreateLinkOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) Rename(
	ctx context.Context,
	op *fuseops.RenameOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) RmDir(
	ctx context.Context,
	op *fuseops.RmDirOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) Unlink(
	ctx context.Context,
	op *fuseops.UnlinkOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) WriteFile(
	ctx context.Context,
	op *fuseops.WriteFileOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) FlushFile(
	ctx context.Context,
	op *fuseops.FlushFileOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) RemoveXattr(
	ctx context.Context,
	op *fuseops.RemoveXattrOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) SetXattr(
	ctx context.Context,
	op *fuseops.SetXattrOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) Fallocate(
	ctx context.Context,
	op *fuseops.FallocateOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) SyncFS(
	ctx context.Context,
	op *fuseops.SyncFSOp) error {
	return fuse.ENOSYS
}

func (fs *NotImplementedFileSystem) Destroy() {
}
